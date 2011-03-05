package main

import (
	"io"
	"os"
	"path"
)

type World interface {
	OpenChunk(x, z int) (io.ReadCloser, os.Error)
	ChunkPool() (ChunkPool, os.Error)
}

type ChunkPool interface {
	Pop(x, z int) bool
	Remaining() int
}

func OpenWorld(worldDir string) World {
	var _, err = os.Stat(path.Join(worldDir, "region"))
	if err != nil {
		return &AlphaWorld{worldDir}
	}
	return &BetaWorld{worldDir}
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
