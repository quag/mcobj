package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"json"
	"math"
	"nbt"
	"os"
	"path"
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
)

func main() {
	var cx, cz int
	var square int
	var maxProcs = runtime.GOMAXPROCS(0)
	var prt bool

	var defaultObjOutFilename = "a.obj"
	var defaultPrtOutFilename = "a.prt"

	var outFilename string
	flag.IntVar(&maxProcs, "cpu", maxProcs, "Number of cores to use")
	flag.IntVar(&square, "s", math.MaxInt32, "Chunk square size")
	flag.StringVar(&outFilename, "o", defaultObjOutFilename, "Name for output file")
	flag.IntVar(&yMin, "y", 0, "Omit all blocks below this height. 63 is sea level")
	flag.BoolVar(&blockFaces, "bf", false, "Don't combine adjacent faces of the same block within a column")
	flag.BoolVar(&hideBottom, "hb", false, "Hide bottom of world")
	flag.BoolVar(&noColor, "g", false, "Omit materials")
	flag.IntVar(&cx, "cx", 0, "Center x coordinate")
	flag.IntVar(&cz, "cz", 0, "Center z coordinate")
	flag.IntVar(&faceLimit, "fk", math.MaxInt32, "Face limit (thousands of faces)")
	flag.BoolVar(&prt, "prt", false, "Write out PRT file instead of Obj file")
	var showHelp = flag.Bool("h", false, "Show Help")
	flag.Parse()

	runtime.GOMAXPROCS(maxProcs)
	fmt.Printf("mcobj %v (cpu: %d) Copyright (c) 2011 Jonathan Wright\n", version, runtime.GOMAXPROCS(0))

	if *showHelp || flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Usage: mcobj -cpu 4 -s 20 -o world1.obj %AppData%\\.minecraft\\saves\\World1")
		fmt.Fprintln(os.Stderr)
		flag.PrintDefaults()
		return
	}

	if faceLimit != math.MaxInt32 {
		faceLimit *= 1000
	}

	if square != math.MaxInt32 {
		chunkLimit = square * square
	} else {
		chunkLimit = math.MaxInt32
	}

	if prt && outFilename == defaultObjOutFilename {
		outFilename = defaultPrtOutFilename
	}

	{
		var dir, _ = path.Split(strings.Replace(os.Args[0], "\\", "/", -1))
		var jsonError = loadBlockTypesJson(path.Join(dir, "blocks.json"))
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

		var world = OpenWorld(dirpath)
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

func base36(i int) string {
	return strconv.Itob(i, 36)
}

func encodeFolder(i int) string {
	return base36(((i % 64) + 64) % 64)
}

func chunkPath(world string, x, z int) string {
	return path.Join(world, encodeFolder(x), encodeFolder(z), "c."+base36(x)+"."+base36(z)+".dat")
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
	var file, fileErr = os.Open(filename, os.O_RDONLY, 0666)
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
	if !sideCache.HasSide(x, z) {
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
						data = byte(v.(float64))
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
				if data != 255 {
					extraData[blockId] = true
					colors = append(colors, MTL{blockId, data, color, name})
				} else {
					colors[blockId] = MTL{blockId, data, color, name}
				}
			}
		}
	}

	return nil
}
