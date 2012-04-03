package main

import (
	"github.com/quag/mcobj/nbt"
)

var (
	emptySide   = &FixedChunkSide{0}
	solidSide   = &FixedChunkSide{1}
	defaultSide = solidSide
)

type ChunkSide interface {
	BlockId(x, y int) nbt.Block
}

type FixedChunkSide struct {
	blockId nbt.Block
}

func (s *FixedChunkSide) BlockId(x, y int) nbt.Block {
	return s.blockId
}

func NewChunkSide(height int) *ChunkSideData {
	return &ChunkSideData{make([]nbt.Block, height*16)}
}

type ChunkSideData struct {
	data []nbt.Block
}

type ChunkSidesData [4]*ChunkSideData

func (s *ChunkSideData) index(x, y int) int {
	return y + (x * s.height())
}

func (s *ChunkSideData) BlockId(x, y int) nbt.Block {
	return s.data[s.index(x, y)]
}

func (s *ChunkSideData) Column(x int) BlockColumn {
	var i = s.height() * x
	return BlockColumn(s.data[i : i+s.height()])
}

func (s *ChunkSideData) height() int {
	return len(s.data) / 16
}
