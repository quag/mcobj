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

