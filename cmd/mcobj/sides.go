package main

var (
	emptySide   = &FixedChunkSide{0}
	solidSide   = &FixedChunkSide{1}
	defaultSide = solidSide
)

type ChunkSide interface {
	BlockId(x, y int) uint16
}

type FixedChunkSide struct {
	blockId uint16
}

func (s *FixedChunkSide) BlockId(x, y int) uint16 {
	return s.blockId
}

func NewChunkSide(height int) *ChunkSideData {
	return &ChunkSideData{make([]uint16, height*16)}
}

type ChunkSideData struct {
	data []uint16
}

type ChunkSidesData [4]*ChunkSideData

func (s *ChunkSideData) index(x, y int) int {
	return y + (x * s.height())
}

func (s *ChunkSideData) BlockId(x, y int) uint16 {
	return s.data[s.index(x, y)]
}

func (s *ChunkSideData) Column(x int) BlockColumn {
	var i = s.height() * x
	return BlockColumn(s.data[i : i+s.height()])
}

func (s *ChunkSideData) height() int {
	return len(s.data) / 16
}
