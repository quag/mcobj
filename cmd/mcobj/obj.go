package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"github.com/quag/mcobj/nbt"
)

type ObjGenerator struct {
	enclosedsChan  chan *EnclosedChunkJob
	writeFacesChan chan *WriteFacesJob
	completeChan   chan bool

	freelist         chan *MemoryWriter
	memoryWriterPool *MemoryWriterPool

	total int

	outFile, voutFile, foutFile *os.File
	out, vout, fout             *bufio.Writer

	outFilename, vFilename, fFilename string
}

func (o *ObjGenerator) Start(outFilename string, total int, maxProcs int, boundary *BoundaryLocator) error {
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
				var vb = o.memoryWriterPool.GetWriter()

				var faceCount, vertexCount, mtls = faces.ProcessChunk(job.enclosed, b, vb)

				o.writeFacesChan <- &WriteFacesJob{job.enclosed.xPos, job.enclosed.zPos, faceCount, vertexCount, mtls, b, vb, job.last}
			}
		}()
	}

	go func() {
		var chunkCount = 0
		var size = 0
		var vertexBase = 0
		for {
			var job = <-o.writeFacesChan
			chunkCount++
			o.out.Write(job.b.buf)
			o.out.Flush()

			if obj3dsmax {
				o.vout.Write(job.vb.buf)
				o.vout.Flush()

				for _, mtl := range job.mtls {
					printMtl(o.fout, mtl.blockId)
					for _, face := range mtl.faces {
						printFaceLine(o.fout, face, vertexBase)
					}
				}
				o.fout.Flush()
				vertexBase += job.vertexCount
			}

			size += len(job.b.buf)
			fmt.Printf("%4v/%-4v (%3v,%3v) Faces: %4d Size: %4.1fMB\n", chunkCount, o.total, job.xPos, job.zPos, job.faceCount, float64(size)/1024/1024)

			o.memoryWriterPool.ReuseWriter(job.b)
			o.memoryWriterPool.ReuseWriter(job.vb)

			if job.last {
				o.completeChan <- true
			}
		}
	}()

	var mtlFilename = fmt.Sprintf("%s.mtl", outFilename[:len(outFilename)-len(filepath.Ext(outFilename))])
	var mtlErr = writeMtlFile(mtlFilename)
	if mtlErr != nil {
		return mtlErr
	}

	o.outFilename = outFilename
	o.vFilename = outFilename + ".v"
	o.fFilename = outFilename + ".f"

	var outFile, voutFile, foutFile *os.File
	var outErr error
	outFile, outErr = os.Create(o.outFilename)
	if outErr != nil {
		return outErr
	}
	defer func() {
		if outFile != nil {
			outFile.Close()
		}
	}()

	o.out = bufio.NewWriterSize(outFile, 1024*1024)

	if obj3dsmax {
		voutFile, outErr = os.Create(o.vFilename)
		if outErr != nil {
			return outErr
		}
		defer func() {
			if voutFile != nil {
				voutFile.Close()
			}
		}()

		foutFile, outErr = os.Create(o.fFilename)
		if outErr != nil {
			return outErr
		}
		defer func() {
			if foutFile != nil {
				foutFile.Close()
			}
		}()

		o.vout = bufio.NewWriterSize(voutFile, 1024*1024)
		o.fout = bufio.NewWriterSize(foutFile, 1024*1024)
	}

	var mw io.Writer
	if obj3dsmax {
		mw = io.MultiWriter(o.out, o.vout)
	} else {
		mw = o.out
	}
	fmt.Fprintln(mw, "mtllib", filepath.Base(mtlFilename))

	o.outFile, outFile = outFile, nil
	o.voutFile, voutFile = voutFile, nil
	o.foutFile, foutFile = foutFile, nil

	return nil
}

func (o *ObjGenerator) GetEnclosedJobsChan() chan *EnclosedChunkJob {
	return o.enclosedsChan
}

func (o *ObjGenerator) GetCompleteChan() chan bool {
	return o.completeChan
}

func (o *ObjGenerator) Close() error {
	o.out.Flush()
	o.outFile.Close()

	if obj3dsmax {
		o.vout.Flush()
		o.fout.Flush()

		o.voutFile.Close()
		o.foutFile.Close()

		var tFilename = o.outFilename + ".tmp"

		toutFile, outErr := os.Create(tFilename)
		if outErr != nil {
			return outErr
		}
		defer toutFile.Close()

		err := copyFile(toutFile, o.vFilename)
		if err != nil {
			return err
		}
		err = copyFile(toutFile, o.fFilename)
		if err != nil {
			return err
		}
		err = toutFile.Close()
		if err != nil {
			return err
		}
		err = os.Rename(tFilename, o.outFilename)
		if err != nil {
			return err
		}
		err = os.Remove(o.vFilename)
		if err != nil {
			return err
		}
		err = os.Remove(o.fFilename)
		if err != nil {
			return err
		}
	}

	return nil
}

func copyFile(w io.Writer, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}

	_, err = io.Copy(w, file)
	return err
}

type WriteFacesJob struct {
	xPos, zPos, faceCount, vertexCount int
	mtls                               []*MtlFaces
	b, vb                              *MemoryWriter
	last                               bool
}

type Faces struct {
	xPos, zPos int
	count      int

	vertexes Vertexes
	faces    []IndexFace
	boundary *BoundaryLocator
}

func (fs *Faces) ProcessChunk(enclosed *EnclosedChunk, w io.Writer, vw io.Writer) (faceCount, vertexCount int, mtls []*MtlFaces) {
	fs.clean(enclosed.xPos, enclosed.zPos, enclosed.height())
	fs.processBlocks(enclosed)
	vertexCount, mtls = fs.Write(w, vw)
	return len(fs.faces), vertexCount, mtls
}

func (fs *Faces) clean(xPos, zPos int, height int) {
	fs.xPos = xPos
	fs.zPos = zPos

	if fs.vertexes.data == nil {
		fs.vertexes.data = make([]int16, (height+1)*(16+1)*(16+1))
	} else {
		fs.vertexes.Clear()
	}

	if fs.faces == nil {
		fs.faces = make([]IndexFace, 0, 8192)
	} else {
		fs.faces = fs.faces[:0]
	}
}

type IndexFace struct {
	blockId nbt.Block
	indexes [4]int
}

type VertexNumFace [4]int

type MtlFaces struct {
	blockId nbt.Block
	faces   []*VertexNumFace
}

func (fs *Faces) AddFace(blockId nbt.Block, v1, v2, v3, v4 Vertex) {
	var face = IndexFace{blockId, [4]int{fs.vertexes.Use(v1), fs.vertexes.Use(v2), fs.vertexes.Use(v3), fs.vertexes.Use(v4)}}
	fs.faces = append(fs.faces, face)
}

func (fs *Faces) Write(w io.Writer, vw io.Writer) (vertexCount int, mtls []*MtlFaces) {
	fs.vertexes.Number()
	var vc = int16(fs.vertexes.Print(io.MultiWriter(w, vw), fs.xPos, fs.zPos))

	var blockIds = make([]nbt.Block, 0, 16)
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

	var mfs = make([]*MtlFaces, 0, len(blockIds))

	for _, blockId := range blockIds {
		printMtl(w, blockId)
		var mf = &MtlFaces{blockId, make([]*VertexNumFace, 0, len(fs.faces))}
		mfs = append(mfs, mf)
		for _, face := range fs.faces {
			if face.blockId == blockId {
				var vf = face.VertexNumFace(fs.vertexes)
				printFaceLine(w, vf, -int(vc+1))
				mf.faces = append(mf.faces, vf)
				faceCount++
			}
		}
	}

	return int(vc), mfs
}

func printFaceLine(w io.Writer, f *VertexNumFace, offset int) {
	fmt.Fprintln(w, "f", f[0]+offset, f[1]+offset, f[2]+offset, f[3]+offset)
}

type Vertex struct {
	x, y, z int
}

type Vertexes struct {
	data   []int16
	height int
}

func (vs *Vertexes) Index(x, y, z int) int {
	return y + (z*(vs.height+1) + (x * (vs.height + 1) * 17))
}

func (vs *Vertexes) Use(v Vertex) int {
	var i = vs.Index(v.x, v.y, v.z)
	vs.data[i]++
	return i
}

func (vs *Vertexes) Release(v Vertex) int {
	var i = vs.Index(v.x, v.y, v.z)
	vs.data[i]--
	return i
}

func (vs *Vertexes) Get(i int) int16 {
	return vs.data[i]
}

func (face *IndexFace) VertexNumFace(vs Vertexes) *VertexNumFace {
	return &VertexNumFace{
		int(vs.Get(face.indexes[0])),
		int(vs.Get(face.indexes[1])),
		int(vs.Get(face.indexes[2])),
		int(vs.Get(face.indexes[3]))}
}

func (vs *Vertexes) Clear() {
	for i, _ := range vs.data {
		vs.data[i] = 0
	}
}

func (vs *Vertexes) Number() {
	var count int16 = 0
	for i, references := range vs.data {
		if references != 0 {
			count++
			vs.data[i] = count
		} else {
			vs.data[i] = -1
		}
	}
}

func (vs *Vertexes) Print(w io.Writer, xPos, zPos int) (count int) {
	var buf = make([]byte, 64)
	copy(buf[0:2], "v ")

	count = 0
	for i := 0; i < len(vs.data); i += (vs.height + 1) {
		var x, z = (i / (vs.height + 1)) / 17, (i / (vs.height + 1)) % 17

		var column = vs.data[i : i+(vs.height+1)]
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
		blockId        nbt.Block
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

	height := enclosedChunk.blocks.height

	for i := 0; i < len(enclosedChunk.blocks.data); i += height {
		var x, z = (i / height) / 16, (i / height) % 16

		var (
			r1 = new(blockRun)
			r2 = new(blockRun)
			r3 = new(blockRun)
			r4 = new(blockRun)
		)

		var column = BlockColumn(enclosedChunk.blocks.data[i : i+height])
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
