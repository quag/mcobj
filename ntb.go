package main

import (
	"bufio"
	"flag"
	"fmt"
	"math"
	"os"
	"path"
	"strconv"
	"nbt"
)

var (
	out        *bufio.Writer
	faces      *Faces
	sideCache  *SideCache
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

func main() {
	var cx, cz int
	var square int

	var filename string
	flag.StringVar(&filename, "o", "a.obj", "Name for output file")
	flag.IntVar(&yMin, "y", 0, "Omit all blocks below this height. 63 is sea level")
	flag.BoolVar(&blockFaces, "bf", false, "Don't combine adjacent faces of the same block within a column")
	flag.BoolVar(&hideBottom, "hb", false, "Hide bottom of world")
	flag.BoolVar(&hideStone, "hs", false, "Hide stone")
	flag.BoolVar(&noColor, "g", false, "Omit materials")
	flag.IntVar(&cx, "cx", 0, "Center x coordinate")
	flag.IntVar(&cz, "cz", 0, "Center z coordinate")
	flag.IntVar(&faceLimit, "fk", math.MaxInt32, "Face limit (thousands of faces)")
	flag.IntVar(&square, "s", math.MaxInt32, "Chunk square size")
	flag.Parse()

	if faceLimit != math.MaxInt32 {
		faceLimit *= 1000
	}

	if square != math.MaxInt32 {
		chunkLimit = square * square
	} else {
		chunkLimit = math.MaxInt32
	}

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

		sideCache = new(SideCache)
		faces = NewFaces(sideCache)

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
				processChunk(filepath, faces)
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

				var total = len(v.chunks)

				for i := 0; moreChunks(v.chunks); i++ {
					for x := 0; x < i && moreChunks(v.chunks); x++ {
						for z := 0; z < i && moreChunks(v.chunks); z++ {
							var (
								ax = cx + unzigzag(x)
								az = cz + unzigzag(z)
							)

							var chunk = chunkPath(filepath, ax, az)
							var _, exists = v.chunks[chunk]
							if exists {
								loadSide(sideCache, filepath, v.chunks, ax-1, az)
								loadSide(sideCache, filepath, v.chunks, ax+1, az)
								loadSide(sideCache, filepath, v.chunks, ax, az-1)
								loadSide(sideCache, filepath, v.chunks, ax, az+1)

								v.chunks[chunk] = false, false
								fmt.Printf("%v/%v ", total-len(v.chunks), total)
								processChunk(chunk, faces)
								chunkCount++
							}
						}
					}
				}
			}
		}
	}
}

type Blocks []byte

type BlockColumn []byte

func (b *Blocks) Get(x, y, z int) byte {
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
			processChunk(fileName, sideCache)
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

type ProcessBlocker interface {
	ProcessBlock(xPos, zPos int, blocks []byte)
}

func processChunk(filename string, processor ProcessBlocker) {
	fmt.Fprintln(out, "#", filename)
	var file, fileErr = os.Open(filename, os.O_RDONLY, 0666)
	if fileErr != nil {
		fmt.Println(fileErr)
	}
	var err, chunk = nbt.ReadChunk(file)
	if err != nil && err != os.EOF {
		fmt.Println(err)
	}
	processor.ProcessBlock(chunk.XPos, chunk.ZPos, chunk.Blocks)
	fmt.Fprintln(out)
	out.Flush()
}
