package main

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

	s.chunks[s.key(xPos, zPos)] = calculateSides(blocks)
}

func (s *SideCache) HasSide(x, z int) bool {
	if s.chunks == nil {
		return false
	}
	var _, present = s.chunks[s.key(x, z)]
	return present
}

func (s *SideCache) EncloseChunk(x, z int, blocks Blocks) *EnclosedChunk {
	return &EnclosedChunk{
		blocks,
		EnclosingSides{
			s.getSide(x-1, z, 1),
			s.getSide(x+1, z, 0),
			s.getSide(x, z-1, 3),
			s.getSide(x, z+1, 2),
		},
	}
}

func calculateSides(blocks Blocks) *ChunkSides {
	var sides = &ChunkSides{}
	for i := 0; i < 16; i++ {
		copy(sides[0].Column(i), blocks.Column(0, i))
		copy(sides[1].Column(i), blocks.Column(15, i))
		copy(sides[2].Column(i), blocks.Column(i, 0))
		copy(sides[3].Column(i), blocks.Column(i, 15))
	}

	return sides
}

func (s *SideCache) getSide(x, z int, side int) *ChunkSide {
	var defaultSide = &solidSide

	if s.chunks == nil {
		return defaultSide
	}
	var chunk, present = s.chunks[s.key(x, z)]
	if !present {
		return defaultSide
	}

	return &chunk[side]
}

func (s *SideCache) key(x, z int) uint64 {
	return (uint64(x) << 32) + uint64(z)
}
