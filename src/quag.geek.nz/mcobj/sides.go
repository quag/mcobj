package main

var (
	emptySide   = &FixedChunkSide{0}
	solidSide   = &FixedChunkSide{1}
	defaultSide = solidSide
)

type IChunkSide interface {
	BlockId(x, y int) uint16
}

type FixedChunkSide struct {
	blockId uint16
}

func (s *FixedChunkSide) BlockId(x, y int) uint16 {
	return s.blockId
}

func NewChunkSide(height int) *ChunkSide {
	return &ChunkSide{make([]uint16, height*16)}
}

type ChunkSide struct {
	data []uint16
}

type ChunkSides [4]*ChunkSide

func (s *ChunkSide) index(x, y int) int {
	return y + (x * s.height())
}

func (s *ChunkSide) BlockId(x, y int) uint16 {
	return s.data[s.index(x, y)]
}

func (s *ChunkSide) Column(x int) BlockColumn {
	var i = s.height() * x
	return BlockColumn(s.data[i : i+s.height()])
}

func (s *ChunkSide) height() int {
	return len(s.data) / 16
}
