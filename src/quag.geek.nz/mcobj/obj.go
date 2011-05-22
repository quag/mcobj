package main

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
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

func (o *ObjGenerator) Start(outFilename string, total int, maxProcs int, boundary *BoundaryLocator) os.Error {
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

				var faceCount, vCount, vtCount, mtls = faces.ProcessChunk(job.enclosed, b, vb)

				o.writeFacesChan <- &WriteFacesJob{job.enclosed.xPos, job.enclosed.zPos, faceCount, vCount, vtCount, mtls, b, vb, job.last}
			}
		}()
	}

	go func() {
		var chunkCount = 0
		var size = 0
		var vBase = 0
		var vtBase = 0
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
						printFaceLine(o.fout, face, vBase, vtBase)
					}
				}
				o.fout.Flush()
				vBase += job.vCount
				vtBase += job.vtCount
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
	var outErr os.Error
	outFile, outErr = os.Create(o.outFilename)
	if outErr != nil {
		return outErr
	}
	defer func() {
		if outFile != nil {
			outFile.Close()
		}
	}()

	var bufErr os.Error
	o.out, bufErr = bufio.NewWriterSize(outFile, 1024*1024)
	if bufErr != nil {
		return bufErr
	}

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

		o.vout, bufErr = bufio.NewWriterSize(voutFile, 1024*1024)
		if bufErr != nil {
			return bufErr
		}
		o.fout, bufErr = bufio.NewWriterSize(foutFile, 1024*1024)
		if bufErr != nil {
			return bufErr
		}
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

func (o *ObjGenerator) Close() os.Error {
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

func copyFile(w io.Writer, filename string) os.Error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}

	_, err = io.Copy(w, file)
	return err
}

type WriteFacesJob struct {
	xPos, zPos, faceCount, vCount, vtCount int
	mtls                                   []*MtlFaces
	b, vb                                  *MemoryWriter
	last                                   bool
}

type Faces struct {
	xPos, zPos int
	count      int

	vertexes  Vertexes
	texcoords TexCoords
	faces     []IndexFace
	boundary  *BoundaryLocator
}

func (fs *Faces) ProcessChunk(enclosed *EnclosedChunk, w io.Writer, vw io.Writer) (faceCount, vCount, vtCount int, mtls []*MtlFaces) {
	fs.clean(enclosed.xPos, enclosed.zPos)
	fs.processBlocks(enclosed)
	vCount, vtCount, mtls = fs.Write(w, vw)
	return len(fs.faces), vCount, vtCount, mtls
}

func (fs *Faces) clean(xPos, zPos int) {
	fs.xPos = xPos
	fs.zPos = zPos

	if fs.vertexes == nil {
		fs.vertexes = make([]int16, (128+1)*(16+1)*(16+1))
	} else {
		fs.vertexes.Clear()
	}
	fs.texcoords.Clear()
	if fs.faces == nil {
		fs.faces = make([]IndexFace, 0, 8192)
	} else {
		fs.faces = fs.faces[:0]
	}
}

type IndexFace struct {
	blockId    uint16
	indexes    [4]int
	texIndexes [4]int
}

type VertexNumFace struct {
	v  [4]int
	vt [4]int
}

type MtlFaces struct {
	blockId uint16
	faces   []*VertexNumFace
}

func crossProductTop(v1, v2, v3 Vertex) bool {
	a := v2.sub(v1)
	b := v3.sub(v1)
	return (a.z*b.x - a.x*b.z) < 0
}

func (fs *Faces) AddFace(blockId uint16, v1, v2, v3, v4 Vertex) {

	mtl := blockTypeMap[uint8(blockId&255)].findMaterial(blockId)
	var tc TexCoord
	numRepetitions := 0
	if v1.y == v2.y && v2.y == v3.y && v3.y == v4.y {
		if crossProductTop(v1, v2, v3) {
			tc = mtl.topTex
		} else {
			tc = mtl.botTex
		}
	} else {
		if v3.y-v2.y > 1 {
			numRepetitions = v3.y - v2.y
			if v1.x == v2.x && v2.x == v3.x && v3.x == v4.x {
				tc = mtl.repeatingSideOffset
			} else {
				tc = mtl.repeatingFrontOffset
			}
		} else {
			if v1.x == v2.x && v2.x == v3.x && v3.x == v4.x {
				tc = mtl.sideTex
			} else {
				tc = mtl.frontTex
			}
		}
	}
	var face = IndexFace{blockId, [4]int{fs.vertexes.Use(v1), fs.vertexes.Use(v2), fs.vertexes.Use(v3), fs.vertexes.Use(v4)}, [4]int{fs.texcoords.Use(tc, true, true, numRepetitions), fs.texcoords.Use(tc, false, true, numRepetitions), fs.texcoords.Use(tc, false, false, numRepetitions), fs.texcoords.Use(tc, true, false, numRepetitions)}}
	fs.faces = append(fs.faces, face)
}

func (fs *Faces) Write(w io.Writer, vw io.Writer) (vCount, vtCount int, mtls []*MtlFaces) {
	fs.vertexes.Number()
	mw := io.MultiWriter(w, vw)
	var vc = int16(fs.vertexes.Print(mw, fs.xPos, fs.zPos))
	fs.texcoords.Number()
	var tc = int16(fs.texcoords.Print(mw, 1024, 1024))
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

	var mfs = make([]*MtlFaces, 0, len(blockIds))

	for _, blockId := range blockIds {
		printMtl(w, blockId)
		var mf = &MtlFaces{blockId, make([]*VertexNumFace, 0, len(fs.faces))}
		mfs = append(mfs, mf)
		for _, face := range fs.faces {
			if face.blockId == blockId {
				var vf = face.VertexNumFace(fs.vertexes)
				printFaceLine(w, vf, -int(vc+1), -int(tc+1))
				mf.faces = append(mf.faces, vf)
				faceCount++
			}
		}
	}

	return int(vc), int(tc), mfs
}

func printFaceLine(w io.Writer, f *VertexNumFace, vbase int, vtbase int) {
	fmt.Fprintf(w, "f %d/%d %d/%d %d/%d %d/%d\n", f.v[0]+vbase, f.vt[0]+vtbase, f.v[1]+vbase, f.vt[1]+vtbase, f.v[2]+vbase, f.vt[2]+vtbase, f.v[3]+vbase, f.vt[3]+vtbase)
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

func (face *IndexFace) VertexNumFace(vs Vertexes) *VertexNumFace {
	return &VertexNumFace{[4]int{
		int(vs.Get(face.indexes[0])),
		int(vs.Get(face.indexes[1])),
		int(vs.Get(face.indexes[2])),
		int(vs.Get(face.indexes[3]))},
		[4]int{
			int(vs.Get(face.texIndexes[0])),
			int(vs.Get(face.texIndexes[1])),
			int(vs.Get(face.texIndexes[2])),
			int(vs.Get(face.texIndexes[3]))}}
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


const numBlockPatternsAcross = 16
const numBlockPatterns = numBlockPatternsAcross * numBlockPatternsAcross
const numNonrepeatingTexcoordsAcross = numBlockPatternsAcross * 2
const numNonrepeatingTexcoords = numBlockPatternsAcross * numBlockPatternsAcross

const numRepeatingPatternsAcross = 64
const numRepeatingTexcoordsAcross = numRepeatingPatternsAcross * 2
const maxDepth = 129 //fencepost (need both top and bottom);

const totalNumTexcoords = numNonrepeatingTexcoords + maxDepth*numRepeatingTexcoordsAcross

type TexCoords [totalNumTexcoords]int16

func (vs *TexCoords) Index(x int, y int, xright bool, ybot bool, reps bool) int {
	var xoffset = 0
	if xright {
		xoffset = -1 //get the coordinate just on the closer edge of the pixel
	}
	if x == 0 && xoffset < 0 {
		return 0
	}

	if reps {
		return x*2 + xoffset + numRepeatingTexcoordsAcross*y + numNonrepeatingTexcoords
	}
	var yoffset = 0
	if ybot {
		yoffset = -1 //get the coordinate just on the closer edge of the pixel
	}
	if y == 0 && yoffset < 0 {
		return 0
	}
	return x*2 + xoffset + numNonrepeatingTexcoordsAcross*(y*2+yoffset)
}

func (tcs *TexCoords) Use(tc TexCoord, xright bool, ybot bool, numReps int) int {
	isRepeating := (numReps != 0)
	var i = 0
	if xright {
		if ybot {
			i = tcs.Index(int(tc.bottomRight.x), int(tc.bottomRight.y)+numReps, xright, ybot, isRepeating) //fixme reps
		} else {
			i = tcs.Index(int(tc.bottomRight.x), int(tc.topLeft.y), xright, ybot, isRepeating) //fixme reps
		}
	} else {
		if ybot {
			i = tcs.Index(int(tc.topLeft.x), int(tc.bottomRight.y)+numReps, xright, ybot, isRepeating) //fixme reps
		} else {
			i = tcs.Index(int(tc.topLeft.x), int(tc.topLeft.y), xright, ybot, isRepeating) //fixme reps
		}
	}
	(*tcs)[i]++
	return i
}

func (tcs *TexCoords) Release(tc TexCoord, xright bool, ybot bool, numReps int) int {
	var i = tcs.Use(tc, xright, ybot, numReps)
	(*tcs)[i]--
	(*tcs)[i]--
	return i
}

func (tcs *TexCoords) Get(i int) int16 {
	return (*tcs)[i]
}

func (tcs *TexCoords) Clear() {
	for i, _ := range *tcs {
		(*tcs)[i] = 0
	}
}

func (tcs *TexCoords) Number() {
	var count int16 = 0
	for i, references := range *tcs {
		if references != 0 {
			count++
			(*tcs)[i] = count
		} else {
			(*tcs)[i] = -1
		}
	}
}

func (tcs *TexCoords) Print(w io.Writer, imageWidth int, imageHeight int) (count int) {
	var buf = make([]byte, 64)
	copy(buf[0:3], "vt ")
	patternWidth := (imageWidth / numBlockPatternsAcross)
	count = 0
	for j := 0; j < numBlockPatternsAcross; j++ {
		for jsub := 0; jsub < 2; jsub++ {
			for i := 0; i < numBlockPatternsAcross; i++ {
				for isub := 0; isub < 2; isub++ {
					xPixel := i*patternWidth + isub*(patternWidth-1)
					yPixel := j*patternWidth + jsub*(patternWidth-1)
					index := i*2 + isub + (j*2+jsub)*numBlockPatternsAcross*2
					if (*tcs)[index] != -1 {
						count++
						xCoord := float64(xPixel) / float64(imageWidth-1)
						yCoord := 1 - float64(yPixel)/float64(imageHeight-1)
						buf = buf[:3]
						if xCoord == xCoord {
							buf = appendFloat(buf, xCoord)
						}
						buf = append(buf, ' ')
						if yCoord == yCoord {
							buf = appendFloat(buf, yCoord)
						}
						buf = append(buf, '\n')
						w.Write(buf)
					}
				}
			}
		}
	}
	repeatingImageWidth := imageWidth / numBlockPatternsAcross * numRepeatingPatternsAcross
	repeatingImageHeight := imageWidth / numBlockPatternsAcross
	for i := 0; i < numRepeatingPatternsAcross; i++ {
		for j := 0; j < maxDepth; j++ {
			for isub := 0; isub < 2; isub++ {
				xPixel := i*patternWidth + isub*(patternWidth-1)
				yPixel := j*patternWidth + (patternWidth - 1)
				if j == 0 {
					yPixel = 0
				}
				index := i*2 + isub + j*numRepeatingPatternsAcross*2 + numNonrepeatingTexcoords
				if (*tcs)[index] != -1 {
					count++
					xCoord := float64(xPixel) / float64(repeatingImageWidth-1)
					yCoord := 1 - float64(yPixel)/float64(repeatingImageHeight-1)
					buf = buf[:3]
					buf = appendFloat(buf, xCoord)
					buf = append(buf, ' ')
					buf = appendFloat(buf, yCoord)
					buf = append(buf, '\n')
					w.Write(buf)
				}
			}
		}
	}

	return
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

func appendFloat(buf []byte, x float64) []byte {
	var highbar float64 = 1.0
	const precision = 8 //match below
	var b [64]byte
	var j = 0
	if x < 0 {
		x = -x
		b[j] = '-'
		j += 1
	}
	for highbar = 1.0; highbar <= x; highbar *= 10 {

	}
	numbers := "0123456789"
	for k := j; k < precision; k++ {
		if highbar < 5 && highbar > .5 {
			b[k] = '.'
			k++
		}

		highbar /= 10
		digit := math.Floor(x / highbar)
		b[k] = numbers[int(digit)%10]
		x -= digit * highbar
	}
	var end = len(buf) + precision
	var d = buf[len(buf):end]
	copy(d, b[0:precision])
	return buf[:end]
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

func (v Vertex) add(v2 Vertex) Vertex {
	return Vertex{v.x + v2.x, v.y + v2.y, v.z + v2.z}
}
func (v Vertex) sub(v2 Vertex) Vertex {
	return Vertex{v.x - v2.x, v.y - v2.y, v.z - v2.z}
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

	var updateBlockRun func(rp **blockRun, nr *blockRun)
	if !blockFaces {
		updateBlockRun = func(rp **blockRun, nr *blockRun) {
			var r = *rp
			if r.dirty {
				if nr.blockId == r.blockId {
					r.v3 = nr.v3
					r.v4 = nr.v4
				} else {
					finishRun(r)
					*rp = nr
				}
			} else {
				*rp = nr
			}
		}
	} else {
		updateBlockRun = func(rp **blockRun, nr *blockRun) {
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
				updateBlockRun(&r1, &blockRun{blockId, Vertex{x, y, z}, Vertex{x, y, z + 1}, Vertex{x, y + 1, z + 1}, Vertex{x, y + 1, z}, true})
			} else {
				finishRun(r1)
			}

			if fs.boundary.IsBoundary(blockId, enclosedChunk.Get(x+1, y, z)) {
				updateBlockRun(&r2, &blockRun{blockId, Vertex{x + 1, y, z}, Vertex{x + 1, y + 1, z}, Vertex{x + 1, y + 1, z + 1}, Vertex{x + 1, y, z + 1}, true})
			} else {
				finishRun(r2)
			}

			if fs.boundary.IsBoundary(blockId, enclosedChunk.Get(x, y, z-1)) {
				updateBlockRun(&r3, &blockRun{blockId, Vertex{x, y, z}, Vertex{x, y + 1, z}, Vertex{x + 1, y + 1, z}, Vertex{x + 1, y, z}, true})
			} else {
				finishRun(r3)
			}

			if fs.boundary.IsBoundary(blockId, enclosedChunk.Get(x, y, z+1)) {
				updateBlockRun(&r4, &blockRun{blockId, Vertex{x, y, z + 1}, Vertex{x + 1, y, z + 1}, Vertex{x + 1, y + 1, z + 1}, Vertex{x, y + 1, z + 1}, true})
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
