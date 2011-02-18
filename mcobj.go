package main

import (
	"bufio"
	"flag"
	"fmt"
	"math"
	"nbt"
	"os"
	"path"
	"strconv"
	"runtime"
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
		filepath string
		enclosed *EnclosedChunk
	}

	type facesDoneJob struct {
		xPos, zPos, faceCount int
		b                     *MemoryWriter
		last                  bool
	}

	var total = 0

	var (
		facesChan        = make(chan *facesJob, maxProcs*2)
		facesJobDoneChan = make(chan *facesDoneJob, maxProcs*2)
		faceDoneChan     = make(chan bool)

		freelist = make(chan *MemoryWriter, maxProcs*2)
		started  = false
	)

	for i := 0; i < maxProcs; i++ {
		go func() {
			var faces Faces
			for {
				var job = <-facesChan

				var b *MemoryWriter
				select {
				case b = <-freelist:
					// Got a buffer
				default:
					b = &MemoryWriter{make([]byte, 0, 128*1024)}
				}

				fmt.Fprintln(b, "#", job.filepath)
				var faceCount = faces.ProcessChunk(job.enclosed, b)
				fmt.Fprintln(b)

				facesJobDoneChan <- &facesDoneJob{job.enclosed.xPos, job.enclosed.zPos, faceCount, b, job.last}
			}
		}()
	}

	go func() {
		var chunkCount = 0
		var size = 0
		for {
			var job = <-facesJobDoneChan
			chunkCount++
			out.Write(job.b.buf)
			out.Flush()

			size += len(job.b.buf)
			fmt.Printf("%4v/%-4v (%3v,%3v) Faces: %4d Size: %4.1fMB\n", chunkCount, total, job.xPos, job.zPos, job.faceCount, float64(size)/1024/1024)

			job.b.Clean()
			select {
			case freelist <- job.b:
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
				var loadErr, chunk = loadChunk(filepath)
				if loadErr != nil {
					fmt.Println(loadErr)
				} else {
					var enclosed = sideCache.EncloseChunk(chunk)
					sideCache.AddChunk(chunk)
					facesChan <- &facesJob{true, filepath, enclosed}
					<-faceDoneChan
				}
			case fi.IsDirectory():
				var errors = make(chan os.Error, 5)
				var done = make(chan bool)
				go func() {
					for error := range errors {
						fmt.Fprintln(os.Stderr, error)
					}
					done <- true
				}()
				var v = &visitor{make(map[string]bool)}
				path.Walk(filepath, v, errors)
				close(errors)
				<-done

				total = len(v.chunks)

				for i := 0; moreChunks(v.chunks); i++ {
					for x := 0; x < i && moreChunks(v.chunks); x++ {
						for z := 0; z < i && moreChunks(v.chunks); z++ {
							var (
								ax = cx + unzigzag(x)
								az = cz + unzigzag(z)
							)

							var chunkFilename = chunkPath(filepath, ax, az)
							var _, exists = v.chunks[chunkFilename]
							if exists {
								loadSide(&sideCache, filepath, v.chunks, ax-1, az)
								loadSide(&sideCache, filepath, v.chunks, ax+1, az)
								loadSide(&sideCache, filepath, v.chunks, ax, az-1)
								loadSide(&sideCache, filepath, v.chunks, ax, az+1)

								v.chunks[chunkFilename] = false, false
								var loadErr, chunk = loadChunk(chunkFilename)
								if loadErr != nil {
									fmt.Println(loadErr)
								} else {
									var enclosed = sideCache.EncloseChunk(chunk)
									sideCache.AddChunk(chunk)
									chunkCount++
									facesChan <- &facesJob{!moreChunks(v.chunks), chunkFilename, enclosed}
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

func moreChunks(chunks map[string]bool) bool {
	return len(chunks) > 0 && faceCount < faceLimit && chunkCount < chunkLimit
}

func loadSide(sideCache *SideCache, world string, chunks map[string]bool, x, z int) {
	if !sideCache.HasSide(x, z) {
		var fileName = chunkPath(world, x, z)
		var _, err = os.Stat(fileName)
		if err == nil {
			var loadErr, chunk = loadChunk(fileName)
			if loadErr != nil {
				fmt.Println(loadErr)
			} else {
				sideCache.AddChunk(chunk)
			}
		}
	}
}

type visitor struct {
	chunks map[string]bool
}

func (v *visitor) VisitDir(dir string, f *os.FileInfo) bool {
	return true
}

func (v *visitor) VisitFile(file string, f *os.FileInfo) {
	var match, err = path.Match("c.*.dat", path.Base(file))
	if match && err == nil {
		v.chunks[file] = true
	}
}

func loadChunk(filename string) (os.Error, *nbt.Chunk) {
	var file, fileErr = os.Open(filename, os.O_RDONLY, 0666)
	defer file.Close()
	if fileErr != nil {
		return fileErr, nil
	}
	var err, chunk = nbt.ReadChunk(file)
	if err == os.EOF {
		err = nil
	}
	return err, chunk
}
