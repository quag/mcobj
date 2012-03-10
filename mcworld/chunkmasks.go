package mcworld

type ChunkMask interface {
	IsMasked(x, z int) bool
}

type RectangleChunkMask struct {
	X0, Z0, X1, Z1 int
}

func (m *RectangleChunkMask) IsMasked(x, z int) bool {
	return x < m.X0 || x >= m.X1 || z < m.Z0 || z >= m.Z1
}

type AllChunksMask struct{}

func (m *AllChunksMask) IsMasked(x, z int) bool {
	return false
}
