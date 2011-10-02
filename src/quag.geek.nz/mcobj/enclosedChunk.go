package main

type EnclosingSides [4]IChunkSide
type EnclosedChunk struct {
	xPos, zPos int
	blocks     Blocks
	enclosing  EnclosingSides
}

func (s *EnclosingSides) side(i int) IChunkSide {
	return (*s)[i]
}

func (e *EnclosedChunk) Get(x, y, z int) (blockId uint16) {
	switch {
	case y < 0 && hideBottom:
		blockId = 7 // Bedrock
	case y < 0 && !hideBottom:
	case y >= e.blocks.height:
		blockId = 0
	case x == -1:
		blockId = e.enclosing.side(0).BlockId(z, y)
	case x == 16:
		blockId = e.enclosing.side(1).BlockId(z, y)
	case z == -1:
		blockId = e.enclosing.side(2).BlockId(x, y)
	case z == 16:
		blockId = e.enclosing.side(3).BlockId(x, y)
	default:
		blockId = e.blocks.Get(x, y, z)
	}

	return
}

func (e *EnclosedChunk) height() int {
	return e.blocks.height
}
