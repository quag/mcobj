package nbt

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
)

type Chunk struct {
	XPos, ZPos int
	Blocks     []uint16
}

var (
	ErrListUnknown = os.NewError("Lists of unknown type aren't supported")
)

func ReadChunkDat(reader io.Reader) (*Chunk, os.Error) {
	r, err := gzip.NewReader(reader)
	defer r.Close()
	if err != nil {
		return nil, err
	}

	return ReadChunkNbt(r)
}

func ReadChunkNbt(reader io.Reader) (*Chunk, os.Error) {
	chunkData := new(chunkData)
	if err := chunkData.parse(NewReader(reader), false); err != nil {
		return nil, err
	}

	chunk := &Chunk{chunkData.xPos, chunkData.zPos, nil}

	if chunkData.blocks != nil && chunkData.data != nil {
		chunk.Blocks = make([]uint16, len(chunkData.blocks))
		for i, blockId := range chunkData.blocks {
			var metadata byte
			if i&1 == 1 {
				metadata = chunkData.data[i/2] >> 4
			} else {
				metadata = chunkData.data[i/2] & 0xf
			}
			chunk.Blocks[i] = uint16(blockId) + (uint16(metadata) << 8)
		}
	}

	return chunk, nil
}

type chunkData struct {
	xPos, zPos int
	blocks     []byte
	data       []byte
}

func (chunk *chunkData) parse(r *Reader, listStruct bool) os.Error {
	structDepth := 0
	if listStruct {
		structDepth++
	}

	for {
		typeId, name, err := r.ReadTag()
		if err != nil {
			if err == os.EOF {
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
				chunk.blocks = bytes
			} else if name == "Data" {
				chunk.data = bytes
			}
		case TagInt8:
			_, err := r.ReadInt8()
			if err != nil {
				return err
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
