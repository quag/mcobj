package main

type EnclosingSides [4]*ChunkSide
type EnclosedChunk struct {
	xPos, zPos int
	blocks     Blocks
	enclosing  EnclosingSides
}

func (s *EnclosingSides) side(i int) *ChunkSide {
	return (*s)[i]
}

func (e *EnclosedChunk) Get(x, y, z int) (blockId uint16) {
	switch {
	case y < 0 && hideBottom:
		blockId = 7 // Bedrock
	case y < 0 && !hideBottom:
	case y > 127:
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

func (b *EnclosedChunk) IsBoundary(x, y, z int, blockId uint16) bool {
	var (
		empty, air, water       = IsEmptyBlock(blockId)
		otherEmpty, otherAir, _ = IsEmptyBlock(b.Get(x, y, z))
	)

	return (empty && !air && otherAir) || (!empty && otherEmpty) || (water && otherAir)
}
