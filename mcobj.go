package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"json"
	"math"
	"./nbt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

var (
	out        *bufio.Writer
	yMin       int
	blockFaces bool
	hideBottom bool
	noColor    bool

	faceCount int
	faceLimit int

	chunkCount int
	chunkLimit int

	chunkMask ChunkMask
)

func main() {
	var bx, bz float64
	var cx, cz int
	var square int
	var rectx, rectz int
	var maxProcs = runtime.GOMAXPROCS(0)
	var prt bool
	var solidSides bool
	var mtlNumber bool

	var defaultObjOutFilename = "a.obj"
	var defaultPrtOutFilename = "a.prt"

	var outFilename string
	flag.IntVar(&maxProcs, "cpu", maxProcs, "Number of cores to use")
	flag.StringVar(&outFilename, "o", defaultObjOutFilename, "Name for output file")
	flag.IntVar(&yMin, "y", 0, "Omit all blocks below this height. 63 is sea level")
	flag.BoolVar(&solidSides, "sides", false, "Solid sides, rather than showing underground")
	flag.BoolVar(&blockFaces, "bf", false, "Don't combine adjacent faces of the same block within a column")
	flag.BoolVar(&hideBottom, "hb", false, "Hide bottom of world")
	flag.BoolVar(&noColor, "g", false, "Omit materials")
	flag.Float64Var(&bx, "x", 0, "Center x coordinate in blocks")
	flag.Float64Var(&bz, "z", 0, "Center z coordinate in blocks")
	flag.IntVar(&cx, "cx", 0, "Center x coordinate in chunks")
	flag.IntVar(&cz, "cz", 0, "Center z coordinate in chunks")
	flag.IntVar(&square, "s", math.MaxInt32, "Chunk square size")
	flag.IntVar(&rectx, "rx", math.MaxInt32, "Width(x) of rectangle size")
	flag.IntVar(&rectz, "rz", math.MaxInt32, "Height(z) of rectangle size")
	flag.IntVar(&faceLimit, "fk", math.MaxInt32, "Face limit (thousands of faces)")
	flag.BoolVar(&prt, "prt", false, "Write out PRT file instead of Obj file")
	flag.BoolVar(&mtlNumber, "mtlnum", false, "Number materials instead of using names")
	var showHelp = flag.Bool("h", false, "Show Help")
	flag.Parse()

	runtime.GOMAXPROCS(maxProcs)
	fmt.Printf("mcobj %v (cpu: %d) Copyright (c) 2011 Jonathan Wright\n", version, runtime.GOMAXPROCS(0))

	if *showHelp || flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Usage: mcobj -cpu 4 -s 20 -o world1.obj", exampleWorldPath)
		fmt.Fprintln(os.Stderr)
		flag.PrintDefaults()
		return
	}

	if faceLimit != math.MaxInt32 {
		faceLimit *= 1000
	}

	if mtlNumber {
		MaterialNamer = new(NumberBlockIdNamer)
	} else {
		MaterialNamer = new(NameBlockIdNamer)
	}

	switch {
	case bx == 0 && cx == 0:
		cx = 0
	case cx == 0:
		cx = int(math.Floor(bx / 16))
	}

	switch {
	case bz == 0 && cz == 0:
		cz = 0
	case cz == 0:
		cz = int(math.Floor(bz / 16))
	}

	if square != math.MaxInt32 {
		chunkLimit = square * square
		var h = square / 2
		chunkMask = &RectangeChunkMask{cx - h, cz - h, cx - h + square, cz - h + square}
	} else if rectx != math.MaxInt32 || rectz != math.MaxInt32 {
		switch {
		case rectx != math.MaxInt32 && rectz != math.MaxInt32:
			chunkLimit = rectx * rectz
			var (
				hx = rectx / 2
				hz = rectz / 2
			)
			chunkMask = &RectangeChunkMask{cx - hx, cz - hz, cx - hx + rectx, cz - hz + rectz}
		case rectx != math.MaxInt32:
			chunkLimit = math.MaxInt32
			var hx = rectx / 2
			chunkMask = &RectangeChunkMask{cx - hx, math.MinInt32, cx - hx + rectx, math.MaxInt32}
		case rectz != math.MaxInt32:
			chunkLimit = math.MaxInt32
			var hz = rectz / 2
			chunkMask = &RectangeChunkMask{math.MinInt32, cz - hz, math.MaxInt32, cz - hz + rectz}
		}
	} else {
		chunkLimit = math.MaxInt32
		chunkMask = &AllChunksMask{}
	}

	if prt && outFilename == defaultObjOutFilename {
		outFilename = defaultPrtOutFilename
	}

	if solidSides {
		defaultSide = &emptySide
	}

	{
		var dir, _ = filepath.Split(strings.Replace(os.Args[0], "\\", "/", -1))
		var jsonError = loadBlockTypesJson(filepath.Join(dir, "blocks.json"))
		if jsonError != nil {
			fmt.Fprintln(os.Stderr, jsonError)
			return
		}
	}

	for i := 0; i < flag.NArg(); i++ {
		var dirpath = flag.Arg(i)
		var fi, err = os.Stat(dirpath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		} else if !fi.IsDirectory() {
			fmt.Fprintln(os.Stderr, dirpath, "is not a directory")
		}

		var world = OpenWorld(dirpath, chunkMask)
		var pool, poolErr = world.ChunkPool()
		if poolErr != nil {
			fmt.Println(poolErr)
			continue
		}

		var generator OutputGenerator
		if prt {
			generator = new(PrtGenerator)
		} else {
			generator = new(ObjGenerator)
		}
		var boundary = new(BoundaryLocator)
		boundary.Init()
		generator.Start(outFilename, pool.Remaining(), maxProcs, boundary)

		if walkEnclosedChunks(pool, world, cx, cz, generator.GetEnclosedJobsChan()) {
			<-generator.GetCompleteChan()
		}

		generator.Close()
	}
}

type OutputGenerator interface {
	Start(outFilename string, total int, maxProcs int, boundary *BoundaryLocator)
	GetEnclosedJobsChan() chan *EnclosedChunkJob
	GetCompleteChan() chan bool
	Close()
}

type EnclosedChunkJob struct {
	last     bool
	enclosed *EnclosedChunk
}

func walkEnclosedChunks(pool ChunkPool, opener ChunkOpener, cx, cz int, enclosedsChan chan *EnclosedChunkJob) bool {
	var (
		sideCache = new(SideCache)
		started   = false
	)

	for i := 0; moreChunks(pool.Remaining()); i++ {
		for x := 0; x < i && moreChunks(pool.Remaining()); x++ {
			for z := 0; z < i && moreChunks(pool.Remaining()); z++ {
				var (
					ax = cx + unzigzag(x)
					az = cz + unzigzag(z)
				)

				if pool.Pop(ax, az) {
					loadSide(sideCache, opener, ax-1, az)
					loadSide(sideCache, opener, ax+1, az)
					loadSide(sideCache, opener, ax, az-1)
					loadSide(sideCache, opener, ax, az+1)

					var chunk, loadErr = loadChunk2(opener, ax, az)
					if loadErr != nil {
						fmt.Println(loadErr)
					} else {
						var enclosed = sideCache.EncloseChunk(chunk)
						sideCache.AddChunk(chunk)
						chunkCount++
						enclosedsChan <- &EnclosedChunkJob{!moreChunks(pool.Remaining()), enclosed}
						started = true
					}
				}
			}
		}
	}

	return started
}

type Blocks []uint16

type BlockColumn []uint16

func (b *Blocks) Get(x, y, z int) uint16 {
	return (*b)[y+(z*128+(x*128*16))]
}

func (b *Blocks) Column(x, z int) BlockColumn {
	var i = 128 * (z + x*16)
	return BlockColumn((*b)[i : i+128])
}

func zigzag(n int) int {
	return (n << 1) ^ (n >> 31)
}

func unzigzag(n int) int {
	return (n >> 1) ^ (-(n & 1))
}

func moreChunks(unprocessedCount int) bool {
	return unprocessedCount > 0 && faceCount < faceLimit && chunkCount < chunkLimit
}

func loadChunk(filename string) (*nbt.Chunk, os.Error) {
	var file, fileErr = os.Open(filename)
	defer file.Close()
	if fileErr != nil {
		return nil, fileErr
	}
	var chunk, err = nbt.ReadDat(file)
	if err == os.EOF {
		err = nil
	}
	return chunk, err
}

func loadChunk2(opener ChunkOpener, x, z int) (*nbt.Chunk, os.Error) {
	var r, openErr = opener.OpenChunk(x, z)
	if openErr != nil {
		return nil, openErr
	}
	defer r.Close()

	var chunk, nbtErr = nbt.ReadNbt(r)
	if nbtErr != nil {
		return nil, nbtErr
	}
	return chunk, nil
}

func loadSide(sideCache *SideCache, opener ChunkOpener, x, z int) {
	if !sideCache.HasSide(x, z) && !chunkMask.IsMasked(x, z) {
		var chunk, loadErr = loadChunk2(opener, x, z)
		if loadErr != nil {
			fmt.Println(loadErr)
		} else {
			sideCache.AddChunk(chunk)
		}
	}
}

func loadBlockTypesJson(filename string) os.Error {
	var jsonBytes, jsonIoError = ioutil.ReadFile(filename)

	if jsonIoError != nil {
		return jsonIoError
	}

	var f interface{}
	var unmarshalError = json.Unmarshal(jsonBytes, &f)
	if unmarshalError != nil {
		return unmarshalError
	}

	var lines, linesOk = f.([]interface{})
	if linesOk {
		for _, line := range lines {
			var fields, fieldsOk = line.(map[string]interface{})
			if fieldsOk {
				var (
					blockId      byte
					data         byte = 255
					dataArray    []byte
					name         string
					mass         SingularOrAggregate = Mass
					transparency Transparency        = Opaque
					empty        bool                = false
					color        uint32
				)
				for k, v := range fields {
					switch k {
					case "name":
						name = v.(string)
					case "color":
						switch len(v.(string)) {
						case 7:
							var n, numErr = strconv.Btoui64(v.(string)[1:], 16)
							if numErr == nil {
								color = uint32(n*0x100 + 0xff)
							}
						case 9:
							var n, numErr = strconv.Btoui64(v.(string)[1:], 16)
							if numErr == nil {
								color = uint32(n)
							}
						}
					case "blockId":
						blockId = byte(v.(float64))
					case "data":
						switch d := v.(type) {
						case float64:
							data = byte(d)
						case []interface{}:
							dataArray = make([]byte, len(d))
							for i, value := range d {
								dataArray[i] = byte(value.(float64))
							}
						}
					case "item":
						if v.(bool) {
							mass = Item
							transparency = Transparent
						} else {
							mass = Mass
							transparency = Opaque
						}
					case "transparent":
						if v.(bool) {
							transparency = Transparent
						} else {
							transparency = Opaque
						}
					case "empty":
						if v.(bool) {
							empty = true
							transparency = Transparent
							mass = Mass
						} else {
							empty = false
						}
					}
				}

				blockTypeMap[blockId] = &BlockType{blockId, mass, transparency, empty}
				if dataArray == nil {
					if data != 255 {
						extraData[blockId] = true
						colors = append(colors, MTL{blockId, data, color, name})
					} else {
						colors[blockId] = MTL{blockId, data, color, name}
					}
				} else {
					extraData[blockId] = true
					for _, data = range dataArray {
						colors = append(colors, MTL{blockId, data, color, fmt.Sprintf("%s_%d", name, data)})
					}
				}
			}
		}
	}

	return nil
}
