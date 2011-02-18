package main

var (
	emptySide ChunkSide
	solidSide ChunkSide
)

func init() {
	for i, _ := range solidSide {
		solidSide[i] = 1
	}
}

type ChunkSide [128 * 16]uint16
type ChunkSides [4]*ChunkSide

func (s *ChunkSides) Side(i int) *ChunkSide {
	return (*s)[i]
}

func (s *ChunkSide) Index(x, y int) int {
	return y + (x * 128)
}

func (s *ChunkSide) BlockId(x, y int) uint16 {
	return (*s)[s.Index(x, y)]
}

func (s *ChunkSide) Column(x int) BlockColumn {
	var i = 128 * x
	return BlockColumn((*s)[i : i+128])
}

func (s *ChunkSide) SetBlockId(x, y int, blockId uint16) {
	(*s)[s.Index(x, y)] = blockId
}
