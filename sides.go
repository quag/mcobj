package main

import (
	"fmt"
	"os"
)

var (
	emptySide ChunkSide
	solidSide ChunkSide
)

func init() {
	for i, _ := range solidSide {
		solidSide[i] = 1
	}
}

type SideCache struct {
	chunks map[uint64]*ChunkSides
}

func (s *SideCache) Clear() {
	s.chunks = nil
}

func (s *SideCache) ProcessBlock(xPos, zPos int, blocks []byte) {
	if s.HasSide(xPos, zPos) {
		return
	}

	if s.chunks == nil {
		s.chunks = make(map[uint64]*ChunkSides)
	}

	s.chunks[s.key(xPos, zPos)] = CalculateSides(blocks)
}

func (s *SideCache) HasSide(x, z int) bool {
	if s.chunks == nil {
		return false
	}
	var _, present = s.chunks[s.key(x, z)]
	return present
}

func (s *SideCache) GetSide(x, z int, side int) *ChunkSide {
	var defaultSide = &solidSide

	if s.chunks == nil {
		return defaultSide
	}
	var chunk, present = s.chunks[s.key(x, z)]
	if !present {
		fmt.Fprintf(os.Stderr, "(%3v,%3v) Missing Side\n", x, z)
		return defaultSide
	}

	return &chunk[side]
}

func (s *SideCache) key(x, z int) uint64 {
	return (uint64(x) << 32) + uint64(z)
}

type ChunkSide [128 * 16]byte
type ChunkSides [4]ChunkSide
type EnclosingSides [4]*ChunkSide
type EnclosedChunk struct {
	blocks    Blocks
	enclosing EnclosingSides
}

func (s *ChunkSides) Side(i int) *ChunkSide {
	return &((*s)[i])
}

func (s *EnclosingSides) Side(i int) *ChunkSide {
	return (*s)[i]
}

func (s *ChunkSide) Index(x, y int) int {
	return y + (x * 128)
}

func (s *ChunkSide) BlockId(x, y int) byte {
	return (*s)[s.Index(x, y)]
}

func (s *ChunkSide) Column(x int) BlockColumn {
	var i = 128 * x
	return BlockColumn((*s)[i : i+128])
}

func (s *ChunkSide) SetBlockId(x, y int, blockId byte) {
	(*s)[s.Index(x, y)] = blockId
}

func (e *EnclosedChunk) Get(x, y, z int) (blockId byte) {
	switch {
	case y < 0 && hideBottom:
		blockId = 7 // Bedrock
	case y < 0 && !hideBottom:
	case y > 127:
		blockId = 0
	case x == -1:
		blockId = e.enclosing.Side(0).BlockId(z, y)
	case x == 16:
		blockId = e.enclosing.Side(1).BlockId(z, y)
	case z == -1:
		blockId = e.enclosing.Side(2).BlockId(x, y)
	case z == 16:
		blockId = e.enclosing.Side(3).BlockId(x, y)
	default:
		blockId = e.blocks.Get(x, y, z)
	}

	return
}

func (s *SideCache) EncloseChunk(x, z int, blocks Blocks) *EnclosedChunk {
	return &EnclosedChunk{
		blocks,
		EnclosingSides{
			s.GetSide(x-1, z, 1),
			s.GetSide(x+1, z, 0),
			s.GetSide(x, z-1, 3),
			s.GetSide(x, z+1, 2),
		},
	}
}


func CalculateSides(blocks Blocks) *ChunkSides {
	var sides = &ChunkSides{}
	for i := 0; i < 16; i++ {
		copy(sides[0].Column(i), blocks.Column(0, i))
		copy(sides[1].Column(i), blocks.Column(15, i))
		copy(sides[2].Column(i), blocks.Column(i, 0))
		copy(sides[3].Column(i), blocks.Column(i, 15))
	}

	return sides
}
