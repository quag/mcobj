package main

import (
	"fmt"
	"io"
	"os"
	"quag.geek.nz/mcworld"
)

func main() {
	world := mcworld.OpenWorld("/Users/jonathan/Library/Application Support/minecraft/saves/1.8.1")
	chunks, err := ZigZagChunks(world, &mcworld.AllChunksMask{})
	if err != nil {
		fmt.Println("ZigZagChunks:", err)
		return
	}

	for chunk := range chunks {
		fmt.Println(chunk.X, chunk.Z)
	}
}

type Chunk struct {
	opener mcworld.ChunkOpener
	X, Z   int
}

func (c *Chunk) Open() (io.ReadCloser, os.Error) {
	return c.opener.OpenChunk(c.X, c.Z)
}

func zigzag(n int) int {
	return (n << 1) ^ (n >> 31)
}

func unzigzag(n int) int {
	return (n >> 1) ^ (-(n & 1))
}

func ZigZagChunks(world mcworld.World, mask mcworld.ChunkMask) (chan *Chunk, os.Error) {
	pool, err := world.ChunkPool(mask)
	if err != nil {
		return nil, err
	}

	c := make(chan *Chunk)

	go func() {
		defer close(c)
		for i := 0; ; i++ {
			for x := 0; x < i; x++ {
				for z := 0; z < i; z++ {
					if pool.Remaining() == 0 {
						return
					}
					ax, az := unzigzag(x), unzigzag(z)
					if pool.Pop(ax, az) {
						c <- &Chunk{world, ax, az}
					}
				}
			}
		}
	}()

	return c, nil
}
