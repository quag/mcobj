package main

type ChunkMask interface {
	IsMasked(x, z int) bool
}

type RectangeChunkMask struct {
	x0, z0, x1, z1 int
}

func (m *RectangeChunkMask) IsMasked(x, z int) bool {
	return x <= m.x0 || x > m.x1 || z <= m.z0 || z > m.z1
}

type AllChunksMask struct{}

func (m *AllChunksMask) IsMasked(x, z int) bool {
	return false
}
