package main

import (
	"bufio"
	"flag"
	"fmt"
	"math"
	"nbt"
	"os"
	"path"
	"runtime"
	"strconv"
)

var (
	out        *bufio.Writer
	sideCache  SideCache
	yMin       int
	blockFaces bool
	hideBottom bool
	hideStone  bool
	noColor    bool

	faceCount int
	faceLimit int

	chunkCount int
	chunkLimit int
)

type MemoryWriter struct {
	buf []byte
}

func (m *MemoryWriter) Clean() {
	if m.buf != nil {
		m.buf = m.buf[:0]
	}
}

func (m *MemoryWriter) Write(p []byte) (n int, err os.Error) {
	m.buf = append(m.buf, p...)
	return len(p), nil
}

func main() {
	var cx, cz int
	var square int
	var maxProcs = runtime.GOMAXPROCS(0)

	var filename string
	flag.IntVar(&maxProcs, "cpu", maxProcs, "Number of cores to use")
	flag.IntVar(&square, "s", math.MaxInt32, "Chunk square size")
	flag.StringVar(&filename, "o", "a.obj", "Name for output file")
	flag.IntVar(&yMin, "y", 0, "Omit all blocks below this height. 63 is sea level")
	flag.BoolVar(&blockFaces, "bf", false, "Don't combine adjacent faces of the same block within a column")
	flag.BoolVar(&hideBottom, "hb", false, "Hide bottom of world")
	flag.BoolVar(&hideStone, "hs", false, "Hide stone")
	flag.BoolVar(&noColor, "g", false, "Omit materials")
	flag.IntVar(&cx, "cx", 0, "Center x coordinate")
	flag.IntVar(&cz, "cz", 0, "Center z coordinate")
	flag.IntVar(&faceLimit, "fk", math.MaxInt32, "Face limit (thousands of faces)")
	var showHelp = flag.Bool("h", false, "Show Help")
	flag.Parse()

	runtime.GOMAXPROCS(maxProcs)
	fmt.Printf("mcobj %v (cpu: %d)\n", version, runtime.GOMAXPROCS(0))

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

	type facesJob struct {
		last     bool
		enclosed *EnclosedChunk
	}

	type facesDoneJob struct {
		xPos, zPos int
		last       bool
		job        *facesJob
		faces      *Faces
	}

	var total = 0

	var (
		facesChan        = make(chan *facesJob, maxProcs*2)
		facesJobDoneChan = make(chan *facesDoneJob, maxProcs*2)
		faceDoneChan     = make(chan bool)
		voffsetChan      = make(chan int, 1)

		bufferFreelist = make(chan *MemoryWriter, maxProcs*2)
		facesFreelist  = make(chan *Faces, maxProcs)
		started        = false
	)

	voffsetChan <- 0

	for i := 0; i < maxProcs; i++ {
		facesFreelist <- new(Faces)
		go func() {
			for {
				var faces = <-facesFreelist
				var job = <-facesChan
				faces.ProcessChunk(job.enclosed)
				facesJobDoneChan <- &facesDoneJob{job.enclosed.xPos, job.enclosed.zPos, job.last, job, faces}
			}
		}()
	}

	go func() {
		var chunkCount = 0
		var size = 0
		for {
			var job = <-facesJobDoneChan

			var (
				faces = job.faces
			)
			chunkCount++

			var b *MemoryWriter
			select {
			case b = <-bufferFreelist:
				// Got a buffer
			default:
				b = &MemoryWriter{make([]byte, 0, 128*1024)}
			}

			faces.Write(b, voffsetChan)
			var faceCount = faces.FaceCount()
			fmt.Fprintln(b)
			facesFreelist <- faces

			out.Write(b.buf)
			out.Flush()

			size += len(b.buf)
			fmt.Printf("%4v/%-4v (%3v,%3v) Faces: %4d Size: %4.1fMB\n", chunkCount, total, job.xPos, job.zPos, faceCount, float64(size)/1024/1024)

			b.Clean()
			select {
			case bufferFreelist <- b:
				// buffer added to free list
			default:
				// free list is full, discard the buffer
			}

			if job.last {
				faceDoneChan <- true
			}
		}
	}()

	if flag.NArg() != 0 {
		var mtlFilename = fmt.Sprintf("%s.mtl", filename[:len(filename)-len(path.Ext(filename))])
		var mtlErr = writeMtlFile(mtlFilename)
		if mtlErr != nil {
			fmt.Fprintln(os.Stderr, mtlErr)
			return
		}

		var outFile, outErr = os.Open(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
		if outErr != nil {
			fmt.Fprintln(os.Stderr, outErr)
			return
		}
		defer outFile.Close()
		var bufErr os.Error
		out, bufErr = bufio.NewWriterSize(outFile, 1024*1024)
		if bufErr != nil {
			fmt.Fprintln(os.Stderr, bufErr)
			return
		}
		defer out.Flush()

		fmt.Fprintln(out, "mtllib", path.Base(mtlFilename))

		for i := 0; i < flag.NArg(); i++ {
			var filepath = flag.Arg(i)
			var fi, err = os.Stat(filepath)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				continue
			}

			switch {
			case fi.IsRegular():
				var chunk, loadErr = loadChunk(filepath)
				if loadErr != nil {
					fmt.Println(loadErr)
				} else {
					var enclosed = sideCache.EncloseChunk(chunk)
					sideCache.AddChunk(chunk)
					facesChan <- &facesJob{true, enclosed}
					<-faceDoneChan
				}
			case fi.IsDirectory():
				var worldDir = filepath
				var world = OpenWorld(worldDir)
				var pool, poolErr = world.ChunkPool()
				if poolErr != nil {
					fmt.Println(poolErr)
					continue
				}

				total = pool.Remaining()

				for i := 0; moreChunks(pool.Remaining()); i++ {
					for x := 0; x < i && moreChunks(pool.Remaining()); x++ {
						for z := 0; z < i && moreChunks(pool.Remaining()); z++ {
							var (
								ax = cx + unzigzag(x)
								az = cz + unzigzag(z)
							)

							if pool.Pop(ax, az) {
								loadSide(&sideCache, world, ax-1, az)
								loadSide(&sideCache, world, ax+1, az)
								loadSide(&sideCache, world, ax, az-1)
								loadSide(&sideCache, world, ax, az+1)

								var chunk, loadErr = loadChunk2(world, ax, az)
								if loadErr != nil {
									fmt.Println(loadErr)
								} else {
									var enclosed = sideCache.EncloseChunk(chunk)
									sideCache.AddChunk(chunk)
									chunkCount++
									facesChan <- &facesJob{!moreChunks(pool.Remaining()), enclosed}
									started = true
								}
							}
						}
					}
				}
			}
		}
	}

	if started {
		<-faceDoneChan
	}
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

func loadChunk2(world World, x, z int) (*nbt.Chunk, os.Error) {
	var r, openErr = world.OpenChunk(x, z)
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

func loadSide(sideCache *SideCache, world World, x, z int) {
	if !sideCache.HasSide(x, z) {
		var chunk, loadErr = loadChunk2(world, x, z)
		if loadErr != nil {
			fmt.Println(loadErr)
		} else {
			sideCache.AddChunk(chunk)
		}
	}
}
