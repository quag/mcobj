package main

import (
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"strconv"
)

var (
	ChunkNotFoundError = os.NewError("Chunk Missing")
)

type BetaWorld struct {
	worldDir string
	mask     ChunkMask
}

type McrFile struct {
	*os.File
}

func (w *BetaWorld) OpenChunk(x, z int) (io.ReadCloser, os.Error) {
	var mcrName = fmt.Sprintf("r.%v.%v.mcr", x>>5, z>>5)
	var mcrPath = path.Join(w.worldDir, "region", mcrName)

	var file, mcrOpenErr = os.Open(mcrPath)
	if mcrOpenErr != nil {
		return nil, mcrOpenErr
	}
	defer func() {
		if file != nil {
			file.Close()
		}
	}()

	var mcr = &McrFile{file}
	var loc, readLocErr = mcr.ReadLocation(x, z)
	if readLocErr != nil {
		return nil, readLocErr
	}

	if loc == 0 {
		return nil, os.NewError(fmt.Sprintf("Chunk missing: %v,%v in %v. %v", x, z, mcrName, (x&31)+(z&31)*32))
	}

	var (
		length          uint32
		compressionType byte
	)

	var _, seekErr = mcr.Seek(int64(loc.Offset()), 0)
	if seekErr != nil {
		return nil, seekErr
	}

	var lengthReadErr = binary.Read(mcr, binary.BigEndian, &length)
	if lengthReadErr != nil {
		return nil, lengthReadErr
	}

	var compressionTypeErr = binary.Read(mcr, binary.BigEndian, &compressionType)
	if compressionTypeErr != nil {
		return nil, compressionTypeErr
	}

	var r, zlibNewErr = zlib.NewReader(mcr)
	if zlibNewErr != nil {
		return nil, zlibNewErr
	}

	var pair = &ReadCloserPair{r, file}
	file = nil
	return pair, nil
}

func (r McrFile) ReadLocation(x, z int) (ChunkLocation, os.Error) {
	var _, seekErr = r.Seek(int64(4*((x&31)+(z&31)*32)), 0)
	if seekErr != nil {
		return ChunkLocation(0), seekErr
	}
	var location uint32
	var readErr = binary.Read(r, binary.BigEndian, &location)
	if readErr != nil {
		return ChunkLocation(0), readErr
	}
	return ChunkLocation(location), nil
}

type ChunkLocation uint32

func (cl ChunkLocation) Offset() int {
	return 4096 * (int(cl) >> 8)
}

func (cl ChunkLocation) Sectors() int {
	return (int(cl) & 0xff)
}

func (w *BetaWorld) ChunkPool() (ChunkPool, os.Error) {
	var regionDirname = path.Join(w.worldDir, "region")
	var dir, dirOpenErr = os.Open(regionDirname)
	if dirOpenErr != nil {
		return nil, dirOpenErr
	}
	defer dir.Close()

	var pool = &BetaChunkPool{make(map[uint64]bool)}

	for {
		var filenames, readErr = dir.Readdirnames(1)
		if readErr != nil {
			return nil, readErr
		}

		if len(filenames) == 0 {
			break
		}

		var fields = strings.FieldsFunc(filenames[0], func(c int) bool { return c == '.' })

		if len(fields) == 4 {
			var (
				rx, rxErr = strconv.Atoi(fields[1])
				rz, ryErr = strconv.Atoi(fields[2])
			)

			if rxErr == nil && ryErr == nil {
				var regionFilename = path.Join(regionDirname, filenames[0])
				var region, regionOpenErr = os.Open(regionFilename)
				if regionOpenErr != nil {
					return nil, regionOpenErr
				}
				defer region.Close()

				for cz := 0; cz < 32; cz++ {
					for cx := 0; cx < 32; cx++ {
						var location uint32
						var readErr = binary.Read(region, binary.BigEndian, &location)
						if readErr != nil {
							return nil, readErr
						}
						if location != 0 {
							var (
								x = rx*32 + cx
								z = rz*32 + cz
							)

							if !w.mask.IsMasked(x, z) {
								pool.chunkMap[betaChunkPoolKey(x, z)] = true
							}
						}
					}
				}
			}
		}
	}

	return pool, nil
}

type BetaChunkPool struct {
	chunkMap map[uint64]bool
}

func (p *BetaChunkPool) Pop(x, z int) bool {
	var key = betaChunkPoolKey(x, z)
	var _, exists = p.chunkMap[key]
	p.chunkMap[key] = false, false
	return exists
}

func (p *BetaChunkPool) Remaining() int {
	return len(p.chunkMap)
}

func betaChunkPoolKey(x, z int) uint64 {
	return uint64(x)<<32 + uint64(z)
}
