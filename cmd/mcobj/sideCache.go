package main

import (
	"github.com/quag/mcobj/nbt"
)

type SideCache struct {
	chunks map[uint64]*ChunkSidesData
}

func (s *SideCache) Clear() {
	s.chunks = nil
}

func (s *SideCache) AddChunk(chunk *nbt.Chunk) {
	if s.HasSide(chunk.XPos, chunk.ZPos) {
		return
	}

	if s.chunks == nil {
		s.chunks = make(map[uint64]*ChunkSidesData)
	}

	s.chunks[s.key(chunk.XPos, chunk.ZPos)] = calculateSides(wrapBlockData(chunk.Blocks))
}

func wrapBlockData(data []uint16) Blocks {
	return Blocks{data, len(data) / (16 * 16)}
}

func (s *SideCache) HasSide(x, z int) bool {
	if s.chunks == nil {
		return false
	}
	var _, present = s.chunks[s.key(x, z)]
	return present
}

func (s *SideCache) EncloseChunk(chunk *nbt.Chunk) *EnclosedChunk {
	return &EnclosedChunk{
		chunk.XPos,
		chunk.ZPos,
		wrapBlockData(chunk.Blocks),
		EnclosingSides{
			s.getSide(chunk.XPos-1, chunk.ZPos, 1),
			s.getSide(chunk.XPos+1, chunk.ZPos, 0),
			s.getSide(chunk.XPos, chunk.ZPos-1, 3),
			s.getSide(chunk.XPos, chunk.ZPos+1, 2),
		},
	}
}

func calculateSides(blocks Blocks) *ChunkSidesData {
	var sides = &ChunkSidesData{NewChunkSide(blocks.height), NewChunkSide(blocks.height), NewChunkSide(blocks.height), NewChunkSide(blocks.height)}
	for i := 0; i < 16; i++ {
		copy(sides[0].Column(i), blocks.Column(0, i))
		copy(sides[1].Column(i), blocks.Column(15, i))
		copy(sides[2].Column(i), blocks.Column(i, 0))
		copy(sides[3].Column(i), blocks.Column(i, 15))
	}

	return sides
}

func (s *SideCache) getSide(x, z int, side int) ChunkSide {
	if s.chunks == nil {
		return defaultSide
	}
	var chunk, present = s.chunks[s.key(x, z)]
	if !present {
		return defaultSide
	}

	var chunkSide = chunk[side]

	chunk[side] = nil

	if chunk[0] == nil && chunk[1] == nil && chunk[2] == nil && chunk[3] == nil {
		delete(s.chunks, s.key(x, z))
	}

	return chunkSide
}

func (s *SideCache) key(x, z int) uint64 {
	return (uint64(x) << 32) + uint64(z)
}
