package mcworld

import (
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type AlphaWorld struct {
	worldDir string
}

func (w *AlphaWorld) OpenChunk(x, z int) (io.ReadCloser, error) {
	var file, fileErr = os.Open(chunkPath(w.worldDir, x, z))
	if fileErr != nil {
		return nil, fileErr
	}
	var decompressor, gzipErr = gzip.NewReader(file)
	if gzipErr != nil {
		file.Close()
		return nil, gzipErr
	}
	return &ReadCloserPair{decompressor, file}, nil
}

type AlphaChunkPool struct {
	chunkMap map[string]bool
	box      *BoundingBox
	worldDir string
}

func (p *AlphaChunkPool) Pop(x, z int) bool {
	var chunkFilename = chunkPath(p.worldDir, x, z)
	var _, exists = p.chunkMap[chunkFilename]
	delete(p.chunkMap, chunkFilename)
	return exists
}

func (p *AlphaChunkPool) BoundingBox() *BoundingBox {
	return p.box
}

func (p *AlphaChunkPool) Remaining() int {
	return len(p.chunkMap)
}

func (w *AlphaWorld) ChunkPool(mask ChunkMask) (ChunkPool, error) {
	chunks := make(map[string]bool)
	box := EmptyBoundingBox()

	err := filepath.Walk(w.worldDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			var match, err = filepath.Match("c.*.*.dat", filepath.Base(path))
			if match && err == nil {
				var (
					s       = strings.SplitN(filepath.Base(path), ".", 4)
					x, xErr = strconv.ParseInt(s[1], 36, 64)
					z, zErr = strconv.ParseInt(s[2], 36, 64)
				)
				if xErr == nil && zErr == nil && !mask.IsMasked(int(x), int(z)) {
					chunks[path] = true
					box.Union(int(x), int(z))
				}
			}
		}

		return nil
	})
	return &AlphaChunkPool{chunks, box, w.worldDir}, err
}

func chunkPath(world string, x, z int) string {
	return filepath.Join(world, encodeFolder(x), encodeFolder(z), "c."+base36(x)+"."+base36(z)+".dat")
}

func base36(i int) string {
	return strconv.FormatInt(int64(i), 36)
}

func encodeFolder(i int) string {
	return base36(((i % 64) + 64) % 64)
}
