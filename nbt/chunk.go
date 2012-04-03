package nbt

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
)

type Chunk struct {
	XPos, ZPos int
	Blocks     []Block
}

var (
	ErrListUnknown = errors.New("Lists of unknown type aren't supported")
)

func ReadChunkDat(reader io.Reader) (*Chunk, error) {
	r, err := gzip.NewReader(reader)
	defer r.Close()
	if err != nil {
		return nil, err
	}

	return ReadChunkNbt(r)
}

func ReadChunkNbt(reader io.Reader) (*Chunk, error) {
	chunkData := new(chunkData)
	chunkData.sections = make([]*sectionData, 0)
	if err := chunkData.parse(NewReader(reader), false); err != nil {
		return nil, err
	}

	chunk := &Chunk{chunkData.xPos, chunkData.zPos, nil}

	if len(chunkData.sections) != 0 {
		chunk.Blocks = make([]Block, 256*16*16) // Hard coded height for now. TODO: Make variable height chunks.
		for _, section := range chunkData.sections {
			for i, blockId := range section.blocks {
				var metadata byte
				if i&1 == 1 {
					metadata = section.data[i/2] >> 4
				} else {
					metadata = section.data[i/2] & 0xf
				}
				// Note that the old format is XZY and the new format is YZX
				x, z, y := indexToCoords(i, 16, 16)
				chunk.Blocks[coordsToIndex(x, z, y+16*section.y, 16, 256)] = Block(blockId) + (Block(metadata) << 8)
			}
		}
	} else {
		if chunkData.blocks != nil && chunkData.data != nil {
			chunk.Blocks = make([]Block, len(chunkData.blocks))
			for i, blockId := range chunkData.blocks {
				var metadata byte
				if i&1 == 1 {
					metadata = chunkData.data[i/2] >> 4
				} else {
					metadata = chunkData.data[i/2] & 0xf
				}
				chunk.Blocks[i] = Block(blockId) + (Block(metadata) << 8)
			}
		}
	}

	return chunk, nil
}

func indexToCoords(i, aMax, bMax int) (a, b, c int) {
	a = i % aMax
	b = (i / aMax) % bMax
	c = i / (aMax * bMax)
	return
}

func coordsToIndex(a, b, c, bMax, cMax int) int {
	return c + cMax*(b+bMax*a)
}

func yzxToXzy(yzx, xMax, zMax, yMax int) int {
	x := yzx % xMax
	z := (yzx / xMax) % zMax
	y := (yzx / (xMax * zMax)) % yMax

	// yzx := x + xMax*(z + zMax*y)
	xzy := y + yMax*(z+zMax*x)
	return xzy
}

type chunkData struct {
	xPos, zPos int
	blocks     []byte
	data       []byte
	section    *sectionData
	sections   []*sectionData
}

type sectionData struct {
	y      int
	blocks []byte
	data   []byte
}

func (chunk *chunkData) parse(r *Reader, listStruct bool) error {
	structDepth := 0
	if listStruct {
		structDepth++
	}

	for {
		typeId, name, err := r.ReadTag()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		switch typeId {
		case TagStruct:
			structDepth++
		case TagStructEnd:
			structDepth--
			if structDepth == 0 {
				return nil
			}
		case TagByteArray:
			bytes, err := r.ReadBytes()
			if err != nil {
				return err
			}
			if name == "Blocks" {
				if chunk.section != nil {
					chunk.section.blocks = bytes
				} else {
					chunk.blocks = bytes
				}
			} else if name == "Data" {
				if chunk.section != nil {
					chunk.section.data = bytes
				} else {
					chunk.data = bytes
				}
			}
		case TagIntArray:
			_, err := r.ReadInts()
			if err != nil {
				return err
			}
		case TagInt8:
			number, err := r.ReadInt8()
			if err != nil {
				return err
			}
			if name == "Y" {
				chunk.section.y = int(number)
			}
		case TagInt16:
			_, err := r.ReadInt16()
			if err != nil {
				return err
			}
		case TagInt32:
			number, err := r.ReadInt32()
			if err != nil {
				return err
			}

			if name == "xPos" {
				chunk.xPos = number
			}
			if name == "zPos" {
				chunk.zPos = number
			}
		case TagInt64:
			_, err := r.ReadInt64()
			if err != nil {
				return err
			}
		case TagFloat32:
			_, err := r.ReadFloat32()
			if err != nil {
				return err
			}
		case TagFloat64:
			_, err := r.ReadFloat64()
			if err != nil {
				return err
			}
		case TagString:
			_, err := r.ReadString()
			if err != nil {
				return err
			}
		case TagList:
			itemTypeId, length, err := r.ReadListHeader()
			if err != nil {
				return err
			}
			switch itemTypeId {
			case TagInt8:
				for i := 0; i < length; i++ {
					_, err := r.ReadInt8()
					if err != nil {
						return err
					}
				}
			case TagFloat32:
				for i := 0; i < length; i++ {
					_, err := r.ReadFloat32()
					if err != nil {
						return err
					}
				}
			case TagFloat64:
				for i := 0; i < length; i++ {
					_, err := r.ReadFloat64()
					if err != nil {
						return err
					}
				}
			case TagStruct:
				for i := 0; i < length; i++ {
					if name == "Sections" {
						chunk.section = new(sectionData)
						chunk.sections = append(chunk.sections, chunk.section)
					}
					err := chunk.parse(r, true)
					if err != nil {
						return err
					}
				}
			default:
				fmt.Printf("# %s list todo(%v) %v\n", name, itemTypeId, length)
				return ErrListUnknown
			}
		default:
			fmt.Printf("# %s todo(%d)\n", name, typeId)
		}
	}

	return nil
}
