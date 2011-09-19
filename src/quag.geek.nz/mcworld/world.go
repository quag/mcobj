package mcworld

import (
	"io"
	"os"
	"path/filepath"
)

type ChunkOpener interface {
	OpenChunk(x, z int) (io.ReadCloser, os.Error)
}

type ChunkPooler interface {
	ChunkPool() (ChunkPool, os.Error)
}

type World interface {
	ChunkOpener
	ChunkPooler
}

type ChunkPool interface {
	Pop(x, z int) bool
	Remaining() int
}

func OpenWorld(worldDir string, mask ChunkMask) World {
	var _, err = os.Stat(filepath.Join(worldDir, "region"))
	if err != nil {
		return &AlphaWorld{worldDir, mask}
	}
	return &BetaWorld{worldDir, mask}
}

type ReadCloserPair struct {
	reader io.ReadCloser
	closer io.Closer
}

func (r *ReadCloserPair) Read(p []byte) (int, os.Error) {
	return r.reader.Read(p)
}

func (r *ReadCloserPair) Close() os.Error {
	var (
		readerErr = r.reader.Close()
		closerErr = r.closer.Close()
	)

	if closerErr != nil {
		return closerErr
	}
	return readerErr
}
