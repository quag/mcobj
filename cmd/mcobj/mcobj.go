package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/quag/mcobj/commandline"
	"github.com/quag/mcobj/mcworld"
	"github.com/quag/mcobj/nbt"
	"io"
	"io/ioutil"
	"math"
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

	obj3dsmax bool
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

	commandLine := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	var outFilename string
	commandLine.IntVar(&maxProcs, "cpu", maxProcs, "Number of cores to use")
	commandLine.StringVar(&outFilename, "o", defaultObjOutFilename, "Name for output file")
	commandLine.IntVar(&yMin, "y", 0, "Omit all blocks below this height. 63 is sea level")
	commandLine.BoolVar(&solidSides, "sides", false, "Solid sides, rather than showing underground")
	commandLine.BoolVar(&blockFaces, "bf", false, "Don't combine adjacent faces of the same block within a column")
	commandLine.BoolVar(&hideBottom, "hb", false, "Hide bottom of world")
	commandLine.BoolVar(&noColor, "g", false, "Omit materials")
	commandLine.Float64Var(&bx, "x", 0, "Center x coordinate in blocks")
	commandLine.Float64Var(&bz, "z", 0, "Center z coordinate in blocks")
	commandLine.IntVar(&cx, "cx", 0, "Center x coordinate in chunks")
	commandLine.IntVar(&cz, "cz", 0, "Center z coordinate in chunks")
	commandLine.IntVar(&square, "s", math.MaxInt32, "Chunk square size")
	commandLine.IntVar(&rectx, "rx", math.MaxInt32, "Width(x) of rectangle size")
	commandLine.IntVar(&rectz, "rz", math.MaxInt32, "Height(z) of rectangle size")
	commandLine.IntVar(&faceLimit, "fk", math.MaxInt32, "Face limit (thousands of faces)")
	commandLine.BoolVar(&prt, "prt", false, "Write out PRT file instead of Obj file")
	commandLine.BoolVar(&obj3dsmax, "3dsmax", false, "Create .obj file compatible with 3dsMax")
	commandLine.BoolVar(&mtlNumber, "mtlnum", false, "Number materials instead of using names")
	var showHelp = commandLine.Bool("h", false, "Show Help")
	commandLine.Parse(os.Args[1:])

	runtime.GOMAXPROCS(maxProcs)
	fmt.Printf("mcobj %v (cpu: %d) Copyright (c) 2011-2012 Jonathan Wright\n", version, runtime.GOMAXPROCS(0))

	exeDir, _ := filepath.Split(strings.Replace(os.Args[0], "\\", "/", -1))

	if *showHelp || commandLine.NArg() == 0 {
		settingsPath := filepath.Join(exeDir, "settings.txt")
		fi, err := os.Stat(settingsPath)
		if err == nil && (!fi.IsDir() || fi.Mode()&os.ModeSymlink != 0) {
			line, err := ioutil.ReadFile(settingsPath)
			if err != nil {
				fmt.Fprintln(os.Stderr, "ioutil.ReadFile:", err)
			} else {
				parseFakeCommandLine(commandLine, line)
			}
		}

		if commandLine.NArg() == 0 {
			fmt.Fprintln(os.Stderr)
			fmt.Fprintln(os.Stderr, "Usage: mcobj -cpu 4 -s 20 -o world1.obj", ExampleWorldPath)
			fmt.Fprintln(os.Stderr)
			commandLine.PrintDefaults()

			fmt.Println()
			stdin := bufio.NewReader(os.Stdin)

			for commandLine.NArg() == 0 {
				fmt.Printf("command line: ")
				line, _, err := stdin.ReadLine()
				if err == io.EOF {
					fmt.Println()
					return
				} else if err != nil {
					fmt.Fprintln(os.Stderr, "stdin.ReadLine:", err)
					return
				}

				parseFakeCommandLine(commandLine, line)
				fmt.Println()
			}
		}
	}

	manualCenter := false
	commandLine.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "x":
			fallthrough
		case "z":
			fallthrough
		case "cx":
			fallthrough
		case "cz":
			manualCenter = true
		}
	})

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

	if prt && outFilename == defaultObjOutFilename {
		outFilename = defaultPrtOutFilename
	}

	if solidSides {
		defaultSide = emptySide
	}

	{
		var jsonError = loadBlockTypesJson(filepath.Join(exeDir, "blocks.json"))
		if jsonError != nil {
			fmt.Fprintln(os.Stderr, "blocks.json error:", jsonError)
			return
		}
	}

	settings := &ProcessingSettings{
		Prt:          prt,
		OutFilename:  outFilename,
		MaxProcs:     maxProcs,
		ManualCenter: manualCenter,
		Cx:           cx,
		Cz:           cz,
		Square:       square,
		Rectx:        rectx,
		Rectz:        rectz,
	}

	validPath := false
	for _, dirpath := range commandLine.Args() {
		var fi, err = os.Stat(dirpath)
		validPath = validPath || (err == nil && !fi.IsDir())
	}

	if validPath {
		for _, dirpath := range commandLine.Args() {
			processWorldDir(dirpath, settings)
		}
	} else {
		processWorldDir(strings.Join(commandLine.Args(), " "), settings)
	}
}

func parseFakeCommandLine(commandLine *flag.FlagSet, line []byte) {
	args := commandline.SplitCommandLine(strings.Trim(string(line), " \r\n"))
	if len(args) >= 1 && args[0] == "mcobj" {
		args = args[1:]
	}
	for i, arg := range args {
		if strings.HasPrefix(arg, "~/") {
			args[i] = filepath.Join(os.Getenv("HOME"), arg[2:])
		} else if strings.HasPrefix(strings.ToUpper(arg), "%APPDATA%\\") || strings.HasPrefix(strings.ToUpper(arg), "%APPDATA%/") {
			args[i] = filepath.Join(os.Getenv("APPDATA"), arg[len("%APPDATA%/"):])
		}
	}
	commandLine.Parse(args)
}

type ProcessingSettings struct {
	Prt          bool
	OutFilename  string
	MaxProcs     int
	ManualCenter bool
	Cx, Cz       int
	Square       int
	Rectx, Rectz int
}

func processWorldDir(dirpath string, settings *ProcessingSettings) {
	var fi, err = os.Stat(dirpath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "World error:", err)
		return
	} else if !fi.IsDir() {
		fmt.Fprintln(os.Stderr, dirpath, "is not a directory")
	}

	// Pick cx, cz
	var cx, cz int
	if settings.ManualCenter {
		cx, cz = settings.Cx, settings.Cz
	} else {
		var file, fileErr = os.Open(filepath.Join(dirpath, "level.dat"))
		defer file.Close()
		if fileErr != nil {
			fmt.Fprintln(os.Stderr, "os.Open", fileErr)
			return
		}
		level, err := nbt.ReadLevelDat(file)
		if err != nil {
			fmt.Fprintln(os.Stderr, "nbt.ReadLevelDat", err)
			return
		}
		file.Close()
		cx, cz = level.SpawnX/16, level.SpawnZ/16
	}

	// Create ChunkMask
	var (
		chunkMask  mcworld.ChunkMask
		chunkLimit int
	)
	if settings.Square != math.MaxInt32 {
		chunkLimit = settings.Square * settings.Square
		var h = settings.Square / 2
		chunkMask = &mcworld.RectangleChunkMask{cx - h, cz - h, cx - h + settings.Square, cz - h + settings.Square}
	} else if settings.Rectx != math.MaxInt32 || settings.Rectz != math.MaxInt32 {
		switch {
		case settings.Rectx != math.MaxInt32 && settings.Rectz != math.MaxInt32:
			chunkLimit = settings.Rectx * settings.Rectz
			var (
				hx = settings.Rectx / 2
				hz = settings.Rectz / 2
			)
			chunkMask = &mcworld.RectangleChunkMask{cx - hx, cz - hz, cx - hx + settings.Rectx, cz - hz + settings.Rectz}
		case settings.Rectx != math.MaxInt32:
			chunkLimit = math.MaxInt32
			var hx = settings.Rectx / 2
			chunkMask = &mcworld.RectangleChunkMask{cx - hx, math.MinInt32, cx - hx + settings.Rectx, math.MaxInt32}
		case settings.Rectz != math.MaxInt32:
			chunkLimit = math.MaxInt32
			var hz = settings.Rectz / 2
			chunkMask = &mcworld.RectangleChunkMask{math.MinInt32, cz - hz, math.MaxInt32, cz - hz + settings.Rectz}
		}
	} else {
		chunkLimit = math.MaxInt32
		chunkMask = &mcworld.AllChunksMask{}
	}

	var world = mcworld.OpenWorld(dirpath)
	var pool, poolErr = world.ChunkPool(chunkMask)
	if poolErr != nil {
		fmt.Fprintln(os.Stderr, "Chunk pool error:", poolErr)
		return
	}

	var generator OutputGenerator
	if settings.Prt {
		generator = new(PrtGenerator)
	} else {
		generator = new(ObjGenerator)
	}
	var boundary = new(BoundaryLocator)
	boundary.Init()
	var startErr = generator.Start(settings.OutFilename, pool.Remaining(), settings.MaxProcs, boundary)
	if startErr != nil {
		fmt.Fprintln(os.Stderr, "Generator start error:", startErr)
		return
	}

	if walkEnclosedChunks(pool, world, chunkMask, chunkLimit, cx, cz, generator.GetEnclosedJobsChan()) {
		<-generator.GetCompleteChan()
	}

	var closeErr = generator.Close()
	if closeErr != nil {
		fmt.Fprintln(os.Stderr, "Generator close error:", closeErr)
		return
	}
}

type OutputGenerator interface {
	Start(outFilename string, total int, maxProcs int, boundary *BoundaryLocator) error
	GetEnclosedJobsChan() chan *EnclosedChunkJob
	GetCompleteChan() chan bool
	Close() error
}

type EnclosedChunkJob struct {
	last     bool
	enclosed *EnclosedChunk
}

func walkEnclosedChunks(pool mcworld.ChunkPool, opener mcworld.ChunkOpener, chunkMask mcworld.ChunkMask, chunkLimit int, cx, cz int, enclosedsChan chan *EnclosedChunkJob) bool {
	var (
		sideCache = new(SideCache)
		started   = false
	)

	for i := 0; moreChunks(pool.Remaining(), chunkLimit); i++ {
		for x := 0; x < i && moreChunks(pool.Remaining(), chunkLimit); x++ {
			for z := 0; z < i && moreChunks(pool.Remaining(), chunkLimit); z++ {
				var (
					ax = cx + unzigzag(x)
					az = cz + unzigzag(z)
				)

				if pool.Pop(ax, az) {
					loadSide(sideCache, opener, chunkMask, ax-1, az)
					loadSide(sideCache, opener, chunkMask, ax+1, az)
					loadSide(sideCache, opener, chunkMask, ax, az-1)
					loadSide(sideCache, opener, chunkMask, ax, az+1)

					var chunk, loadErr = loadChunk2(opener, ax, az)
					if loadErr != nil {
						fmt.Println(loadErr)
					} else {
						var enclosed = sideCache.EncloseChunk(chunk)
						sideCache.AddChunk(chunk)
						chunkCount++
						enclosedsChan <- &EnclosedChunkJob{!moreChunks(pool.Remaining(), chunkLimit), enclosed}
						started = true
					}
				}
			}
		}
	}

	return started
}

type Blocks struct {
	data   []nbt.Block
	height int
}

type BlockColumn []nbt.Block

func (b *Blocks) Get(x, y, z int) nbt.Block {
	return b.data[y+(z*b.height+(x*b.height*16))]
}

func (b *Blocks) Column(x, z int) BlockColumn {
	var i = b.height * (z + x*16)
	return BlockColumn(b.data[i : i+b.height])
}

func zigzag(n int) int {
	return (n << 1) ^ (n >> 31)
}

func unzigzag(n int) int {
	return (n >> 1) ^ (-(n & 1))
}

func moreChunks(unprocessedCount, chunkLimit int) bool {
	return unprocessedCount > 0 && faceCount < faceLimit && chunkCount < chunkLimit
}

func loadChunk(filename string) (*nbt.Chunk, error) {
	var file, fileErr = os.Open(filename)
	defer file.Close()
	if fileErr != nil {
		return nil, fileErr
	}
	var chunk, err = nbt.ReadChunkDat(file)
	if err == io.EOF {
		err = nil
	}
	return chunk, err
}

func loadChunk2(opener mcworld.ChunkOpener, x, z int) (*nbt.Chunk, error) {
	var r, openErr = opener.OpenChunk(x, z)
	if openErr != nil {
		return nil, openErr
	}
	defer r.Close()

	var chunk, nbtErr = nbt.ReadChunkNbt(r)
	if nbtErr != nil {
		return nil, nbtErr
	}
	return chunk, nil
}

func loadSide(sideCache *SideCache, opener mcworld.ChunkOpener, chunkMask mcworld.ChunkMask, x, z int) {
	if !sideCache.HasSide(x, z) && !chunkMask.IsMasked(x, z) {
		var chunk, loadErr = loadChunk2(opener, x, z)
		if loadErr != nil {
			fmt.Println(loadErr)
		} else {
			sideCache.AddChunk(chunk)
		}
	}
}

func loadBlockTypesJson(filename string) error {
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
							var n, numErr = strconv.ParseUint(v.(string)[1:], 16, 64)
							if numErr == nil {
								color = uint32(n*0x100 + 0xff)
							}
						case 9:
							var n, numErr = strconv.ParseUint(v.(string)[1:], 16, 64)
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
