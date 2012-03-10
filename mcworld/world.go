package mcworld

import (
	"io"
	"math"
	"os"
	"path/filepath"
)

type ChunkOpener interface {
	OpenChunk(x, z int) (io.ReadCloser, error)
}

type ChunkPooler interface {
	ChunkPool(mask ChunkMask) (ChunkPool, error)
}

type World interface {
	ChunkOpener
	ChunkPooler
}

type ChunkPool interface {
	Pop(x, z int) bool
	Remaining() int
	BoundingBox() *BoundingBox
}

type BoundingBox struct {
	X0, Z0, X1, Z1 int
}

func OpenWorld(worldDir string) World {
	var _, err = os.Stat(filepath.Join(worldDir, "region"))
	if err != nil {
		return &AlphaWorld{worldDir}
	}
	return &BetaWorld{worldDir}
}

type ReadCloserPair struct {
	reader io.ReadCloser
	closer io.Closer
}

func (r *ReadCloserPair) Read(p []byte) (int, error) {
	return r.reader.Read(p)
}

func (r *ReadCloserPair) Close() error {
	var (
		readerErr = r.reader.Close()
		closerErr = r.closer.Close()
	)

	if closerErr != nil {
		return closerErr
	}
	return readerErr
}

func EmptyBoundingBox() *BoundingBox {
	return &BoundingBox{math.MaxInt32, math.MaxInt32, math.MinInt32, math.MinInt32}
}

func (b *BoundingBox) Union(x, z int) {
	if x < b.X0 {
		b.X0 = x
	} else if x > b.X1 {
		b.X1 = x
	}

	if z < b.Z0 {
		b.Z0 = z
	} else if z > b.Z1 {
		b.Z1 = z
	}
}
