package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type ObjGenerator struct {
	enclosedsChan  chan *EnclosedChunkJob
	writeFacesChan chan *WriteFacesJob
	completeChan   chan bool

	freelist chan *MemoryWriter
	memoryWriterPool *MemoryWriterPool

	total int

	outFile *os.File
	out     *bufio.Writer
}

func (o *ObjGenerator) Start(outFilename string, total int, maxProcs int, boundary *BoundaryLocator) {
	o.enclosedsChan = make(chan *EnclosedChunkJob, maxProcs*2)
	o.writeFacesChan = make(chan *WriteFacesJob, maxProcs*2)
	o.completeChan = make(chan bool)

	o.memoryWriterPool = NewMemoryWriterPool(maxProcs*2, 128*1024)
	o.total = total

	for i := 0; i < maxProcs; i++ {
		go func() {
			var faces Faces
			faces.boundary = boundary
			for {
				var job = <-o.enclosedsChan

				var b = o.memoryWriterPool.GetWriter()
				var faceCount = faces.ProcessChunk(job.enclosed, b)

				o.writeFacesChan <- &WriteFacesJob{job.enclosed.xPos, job.enclosed.zPos, faceCount, b, job.last}
			}
		}()
	}

	go func() {
		var chunkCount = 0
		var size = 0
		for {
			var job = <-o.writeFacesChan
			chunkCount++
			o.out.Write(job.b.buf)
			o.out.Flush()

			size += len(job.b.buf)
			fmt.Printf("%4v/%-4v (%3v,%3v) Faces: %4d Size: %4.1fMB\n", chunkCount, o.total, job.xPos, job.zPos, job.faceCount, float64(size)/1024/1024)

			o.memoryWriterPool.ReuseWriter(job.b)

			if job.last {
				o.completeChan <- true
			}
		}
	}()

	var mtlFilename = fmt.Sprintf("%s.mtl", outFilename[:len(outFilename)-len(filepath.Ext(outFilename))])
	var mtlErr = writeMtlFile(mtlFilename)
	if mtlErr != nil {
		fmt.Fprintln(os.Stderr, mtlErr)
		return
	}

	var outFile, outErr = os.Create(outFilename)
	if outErr != nil {
		fmt.Fprintln(os.Stderr, outErr)
		return
	}
	defer func() {
		if outFile != nil {
			outFile.Close()
		}
	}()
	var bufErr os.Error
	o.out, bufErr = bufio.NewWriterSize(outFile, 1024*1024)
	if bufErr != nil {
		fmt.Fprintln(os.Stderr, bufErr)
		return
	}

	fmt.Fprintln(o.out, "mtllib", filepath.Base(mtlFilename))

	o.outFile, outFile = outFile, nil
}

func (o *ObjGenerator) GetEnclosedJobsChan() chan *EnclosedChunkJob {
	return o.enclosedsChan
}

func (o *ObjGenerator) GetCompleteChan() chan bool {
	return o.completeChan
}

func (o *ObjGenerator) Close() {
	o.out.Flush()
	o.outFile.Close()
}

type WriteFacesJob struct {
	xPos, zPos, faceCount int
	b                     *MemoryWriter
	last                  bool
}

type Faces struct {
	xPos, zPos int
	count      int

	vertexes Vertexes
	faces    []Face
	boundary *BoundaryLocator
}

func (fs *Faces) ProcessChunk(enclosed *EnclosedChunk, w io.Writer) (count int) {
	fs.clean(enclosed.xPos, enclosed.zPos)
	fs.processBlocks(enclosed)
	fs.Write(w)
	return len(fs.faces)
}

func (fs *Faces) clean(xPos, zPos int) {
	fs.xPos = xPos
	fs.zPos = zPos

	if fs.vertexes == nil {
		fs.vertexes = make([]int16, (128+1)*(16+1)*(16+1))
	} else {
		fs.vertexes.Clear()
	}

	if fs.faces == nil {
		fs.faces = make([]Face, 0, 8192)
	} else {
		fs.faces = fs.faces[:0]
	}
}

type Face struct {
	blockId uint16
	indexes [4]int
}

func (fs *Faces) AddFace(blockId uint16, v1, v2, v3, v4 Vertex) {
	var face = Face{blockId, [4]int{fs.vertexes.Use(v1), fs.vertexes.Use(v2), fs.vertexes.Use(v3), fs.vertexes.Use(v4)}}
	fs.faces = append(fs.faces, face)
}

func (fs *Faces) Write(w io.Writer) {
	fs.vertexes.Number()
	var vc = int16(fs.vertexes.Print(w, fs.xPos, fs.zPos))

	var blockIds = make([]uint16, 0, 16)
	for _, face := range fs.faces {
		var found = false
		for _, id := range blockIds {
			if id == face.blockId {
				found = true
				break
			}
		}

		if !found {
			blockIds = append(blockIds, face.blockId)
		}
	}

	for _, blockId := range blockIds {
		printMtl(w, blockId)
		for _, face := range fs.faces {
			if face.blockId == blockId {
				fmt.Fprintln(w, "f", fs.vertexes.Get(face.indexes[0])-vc-1, fs.vertexes.Get(face.indexes[1])-vc-1, fs.vertexes.Get(face.indexes[2])-vc-1, fs.vertexes.Get(face.indexes[3])-vc-1)
				faceCount++
			}
		}
	}
}

type Vertex struct {
	x, y, z int
}

type Vertexes []int16

func (vs *Vertexes) Index(x, y, z int) int {
	return y + (z*129 + (x * 129 * 17))
}

func (vs *Vertexes) Use(v Vertex) int {
	var i = vs.Index(v.x, v.y, v.z)
	(*vs)[i]++
	return i
}

func (vs *Vertexes) Release(v Vertex) int {
	var i = vs.Index(v.x, v.y, v.z)
	(*vs)[i]--
	return i
}

func (vs *Vertexes) Get(i int) int16 {
	return (*vs)[i]
}

func (vs *Vertexes) Clear() {
	for i, _ := range *vs {
		(*vs)[i] = 0
	}
}

func (vs *Vertexes) Number() {
	var count int16 = 0
	for i, references := range *vs {
		if references != 0 {
			count++
			(*vs)[i] = count
		} else {
			(*vs)[i] = -1
		}
	}
}

func (vs *Vertexes) Print(w io.Writer, xPos, zPos int) (count int) {
	var buf = make([]byte, 64)
	copy(buf[0:2], "v ")

	count = 0
	for i := 0; i < len(*vs); i += 129 {
		var x, z = (i / 129) / 17, (i / 129) % 17

		var column = (*vs)[i : i+129]
		for y, offset := range column {
			if offset != -1 {

				count++

				var (
					xa = x + xPos*16
					ya = y - 64
					za = z + zPos*16
				)

				buf = buf[:2]
				buf = appendCoord(buf, xa)
				buf = append(buf, ' ')
				buf = appendCoord(buf, ya)
				buf = append(buf, ' ')
				buf = appendCoord(buf, za)
				buf = append(buf, '\n')

				w.Write(buf)
			}
		}
	}
	return
}

func appendCoord(buf []byte, x int) []byte {
	var b [64]byte
	var j = len(b)

	var neg = x < 0
	if neg {
		x = -x
	}

	var (
		high    = x / 20
		low     = (x % 20) * 5
		numbers = "0123456789"
	)

	for i := 0; i < 2; i++ {
		j--
		b[j] = numbers[low%10]
		low /= 10
	}

	j--
	b[j] = '.'

	if high == 0 {
		j--
		b[j] = '0'
	} else {
		for high > 0 {
			j--
			b[j] = numbers[high%10]
			high /= 10
		}
	}

	if neg {
		j--
		b[j] = '-'
	}

	var end = len(buf) + len(b) - j
	var d = buf[len(buf):end]
	copy(d, b[j:])
	return buf[:end]
}

func (fs *Faces) processBlocks(enclosedChunk *EnclosedChunk) {
	type blockRun struct {
		blockId        uint16
		v1, v2, v3, v4 Vertex
		dirty          bool
	}

	var finishRun = func(r *blockRun) {
		if r.dirty {
			fs.AddFace(r.blockId, r.v1, r.v2, r.v3, r.v4)
			r.dirty = false
		}
	}

	var updateBlockRun func(rp **blockRun, nr *blockRun, flag bool)
	if !blockFaces {
		updateBlockRun = func(rp **blockRun, nr *blockRun, flag bool) {
			var r = *rp
			if r.dirty {
				if nr.blockId == r.blockId {
					if flag {
						r.v3 = nr.v3
						r.v4 = nr.v4
					} else {
						r.v2 = nr.v2
						r.v3 = nr.v3
					}
				} else {
					finishRun(r)
					*rp = nr
				}
			} else {
				*rp = nr
			}
		}
	} else {
		updateBlockRun = func(rp **blockRun, nr *blockRun, flag bool) {
			finishRun(nr)
		}
	}

	for i := 0; i < len(enclosedChunk.blocks); i += 128 {
		var x, z = (i / 128) / 16, (i / 128) % 16

		var (
			r1 = new(blockRun)
			r2 = new(blockRun)
			r3 = new(blockRun)
			r4 = new(blockRun)
		)

		var column = BlockColumn(enclosedChunk.blocks[i : i+128])
		for y, blockId := range column {
			if y < yMin {
				continue
			}

			if fs.boundary.IsBoundary(blockId, enclosedChunk.Get(x, y-1, z)) {
				fs.AddFace(blockId, Vertex{x, y, z}, Vertex{x + 1, y, z}, Vertex{x + 1, y, z + 1}, Vertex{x, y, z + 1})
			}

			if fs.boundary.IsBoundary(blockId, enclosedChunk.Get(x, y+1, z)) {
				fs.AddFace(blockId, Vertex{x, y + 1, z}, Vertex{x, y + 1, z + 1}, Vertex{x + 1, y + 1, z + 1}, Vertex{x + 1, y + 1, z})
			}

			if fs.boundary.IsBoundary(blockId, enclosedChunk.Get(x-1, y, z)) {
				updateBlockRun(&r1, &blockRun{blockId, Vertex{x, y, z}, Vertex{x, y, z + 1}, Vertex{x, y + 1, z + 1}, Vertex{x, y + 1, z}, true}, true)
			} else {
				finishRun(r1)
			}

			if fs.boundary.IsBoundary(blockId, enclosedChunk.Get(x+1, y, z)) {
				updateBlockRun(&r2, &blockRun{blockId, Vertex{x + 1, y, z}, Vertex{x + 1, y + 1, z}, Vertex{x + 1, y + 1, z + 1}, Vertex{x + 1, y, z + 1}, true}, false)
			} else {
				finishRun(r2)
			}

			if fs.boundary.IsBoundary(blockId, enclosedChunk.Get(x, y, z-1)) {
				updateBlockRun(&r3, &blockRun{blockId, Vertex{x, y, z}, Vertex{x, y + 1, z}, Vertex{x + 1, y + 1, z}, Vertex{x + 1, y, z}, true}, false)
			} else {
				finishRun(r3)
			}

			if fs.boundary.IsBoundary(blockId, enclosedChunk.Get(x, y, z+1)) {
				updateBlockRun(&r4, &blockRun{blockId, Vertex{x, y, z + 1}, Vertex{x + 1, y, z + 1}, Vertex{x + 1, y + 1, z + 1}, Vertex{x, y + 1, z + 1}, true}, true)
			} else {
				finishRun(r4)
			}
		}

		finishRun(r1)
		finishRun(r2)
		finishRun(r3)
		finishRun(r4)
	}
}
