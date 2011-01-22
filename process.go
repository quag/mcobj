package main

import (
	"fmt"
	"io"
	"os"
)

func (fs *Faces) ProcessBlock(xPos, zPos int, bytes []byte) {
	fs.Clean(xPos, zPos)
	processBlocks(bytes, fs)
	fs.Process()
}

type Blocks []byte

func (b *Blocks) Get(x, y, z int) byte {
	return (*b)[y+(z*128+(x*128*16))]
}

func (b *Blocks) IsEmpty(x, y, z int) bool {
	if x < 0 || y < 0 || z < 0 || x >= 16 || y >= 128 || z >= 16 {
		return true
	}
	return isEmptyBlockId(b.Get(x, y, z))
}

func (b *Blocks) IsBoundary(x, y, z int, blockId byte) bool {
	if x < 0 || y < 0 || z < 0 || x >= 16 || y >= 128 || z >= 16 {
		if showTunnels {
			return blockId == 18 /* leaves */ || blockId == 17 /* wood */
		} else {
			return blockId != 0 && blockId != 9 && (!hideStone || blockId != 1)
		}
	}

	var otherBlockId = b.Get(x, y, z)

	if (blockId == 9 || (hideStone && blockId == 1)) && otherBlockId == 0 {
		return true
	}

	return (blockId != 0 && blockId != 9 && (!hideStone || blockId != 1)) && (otherBlockId == 0 || otherBlockId == 9 || (hideStone && otherBlockId == 1))
}

func isEmptyBlockId(blockId byte) bool {
	return blockId == 0 || blockId == 1
}

type AddFacer interface {
	AddFace(blockId byte, v1, v2, v3, v4 Vertex)
}

type Faces struct {
	xPos, zPos int
	count      int

	vertexes Vertexes
	faces    []Face
}

type Face struct {
	blockId byte
	indexes [4]int
}

func NewFaces() *Faces {
	return &Faces{0, 0, 0, make([]int16, (128+1)*(16+1)*(16+1)), make([]Face, 0, 8192)}
}

func (fs *Faces) Clean(xPos, zPos int) {
	fs.xPos = xPos
	fs.zPos = zPos
	fs.vertexes.Clear()
	fs.faces = fs.faces[0:0]
}

func (fs *Faces) AddFace(blockId byte, v1, v2, v3, v4 Vertex) {
	var face = Face{blockId, [4]int{fs.vertexes.Use(v1), fs.vertexes.Use(v2), fs.vertexes.Use(v3), fs.vertexes.Use(v4)}}
	fs.faces = append(fs.faces, face)
}

func (fs *Faces) Process() {
	fs.vertexes.Number()
	var vc = int16(fs.vertexes.Print(out, fs.xPos, fs.zPos))

	var blockIds = make([]byte, 0, 16)
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
		printMtl(out, blockId)
		for _, face := range fs.faces {
			if face.blockId == blockId {
				fmt.Fprintln(out, "f", fs.vertexes.Get(face.indexes[0])-vc-1, fs.vertexes.Get(face.indexes[1])-vc-1, fs.vertexes.Get(face.indexes[2])-vc-1, fs.vertexes.Get(face.indexes[3])-vc-1)
				faceCount++
			}
		}
	}

	fmt.Fprintln(os.Stderr, "Faces:", len(fs.faces))
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

func (vs *Vertexes) Print(writer io.Writer, xPos, zPos int) (count int) {
	count = 0
	for i := 0; i < len(*vs); i += 129 {
		var x, z = (i / 129) / 17, (i / 129) % 17

		var column = (*vs)[i : i+129]
		for y, offset := range column {
			if offset != -1 {
				count++
				fmt.Fprintf(writer, "v %.2f %.2f %.2f\n", float64(x+xPos*16)*0.05, float64(y)*0.05, float64(z+zPos*16)*0.05)
			}
		}
	}
	return
}

type Vertex struct {
	x, y, z int
}

func printFace(xPos, zPos int, vertexes ...Vertex) {
	for _, vertex := range vertexes {
		fmt.Fprintf(out, "v %.2f %.2f %.2f\n", float64(vertex.x+xPos*16)*0.05, float64(vertex.y)*0.05, float64(vertex.z+zPos*16)*0.05)
	}
	fmt.Fprintf(out, "f")
	for i, _ := range vertexes {
		fmt.Fprintf(out, " -%d", len(vertexes)-i)
	}
	fmt.Fprintln(out)
}

type blockRun struct {
	blockId        byte
	v1, v2, v3, v4 Vertex
	dirty          bool
}

func (r *blockRun) AddFace(faces AddFacer) {
	if r.dirty {
		faces.AddFace(r.blockId, r.v1, r.v2, r.v3, r.v4)
		r.dirty = false
	}
}

func (r *blockRun) Update(faces AddFacer, nr *blockRun, flag bool) {
	if !blockFaces {
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
				r.AddFace(faces)
				*r = *nr
			}
		} else {
			*r = *nr
		}
	} else {
		nr.AddFace(faces)
		nr.dirty = false
	}
}

func processBlocks(bytes []byte, faces AddFacer) {
	var blocks = Blocks(bytes)

	for i := 0; i < len(blocks); i += 128 {
		var x, z = (i / 128) / 16, (i / 128) % 16

		var r1, r2, r3, r4 blockRun

		var column = blocks[i : i+128]
		for y, blockId := range column {
			if y < yMin {
				continue
			}

			if blocks.IsBoundary(x, y-1, z, blockId) {
				faces.AddFace(blockId, Vertex{x, y, z}, Vertex{x + 1, y, z}, Vertex{x + 1, y, z + 1}, Vertex{x, y, z + 1})
			}

			if blocks.IsBoundary(x, y+1, z, blockId) {
				faces.AddFace(blockId, Vertex{x, y + 1, z}, Vertex{x, y + 1, z + 1}, Vertex{x + 1, y + 1, z + 1}, Vertex{x + 1, y + 1, z})
			}

			if blocks.IsBoundary(x-1, y, z, blockId) {
				r1.Update(faces, &blockRun{blockId, Vertex{x, y, z}, Vertex{x, y, z + 1}, Vertex{x, y + 1, z + 1}, Vertex{x, y + 1, z}, true}, true)
			} else {
				r1.AddFace(faces)
			}

			if blocks.IsBoundary(x+1, y, z, blockId) {
				r2.Update(faces, &blockRun{blockId, Vertex{x + 1, y, z}, Vertex{x + 1, y + 1, z}, Vertex{x + 1, y + 1, z + 1}, Vertex{x + 1, y, z + 1}, true}, false)
			} else {
				r2.AddFace(faces)
			}

			if blocks.IsBoundary(x, y, z-1, blockId) {
				r3.Update(faces, &blockRun{blockId, Vertex{x, y, z}, Vertex{x, y + 1, z}, Vertex{x + 1, y + 1, z}, Vertex{x + 1, y, z}, true}, false)
			} else {
				r3.AddFace(faces)
			}

			if blocks.IsBoundary(x, y, z+1, blockId) {
				r4.Update(faces, &blockRun{blockId, Vertex{x, y, z + 1}, Vertex{x + 1, y, z + 1}, Vertex{x + 1, y + 1, z + 1}, Vertex{x, y + 1, z + 1}, true}, true)
			} else {
				r4.AddFace(faces)
			}
		}

		r1.AddFace(faces)
		r2.AddFace(faces)
		r3.AddFace(faces)
		r4.AddFace(faces)
	}
}
