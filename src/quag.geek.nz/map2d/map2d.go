package main

import (
	"fmt"
	"io"
	"os"
	"quag.geek.nz/mcworld"
	"quag.geek.nz/nbt"
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

		err := useChunk(chunk)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func useChunk(chunk *Chunk) os.Error {
	var r, openErr = chunk.Open()
	if openErr != nil {
		return openErr
	}
	defer r.Close()

	var c, nbtErr = nbt.ReadChunkNbt(r)
	if nbtErr != nil {
		return nbtErr
	}

	blocks := Blocks(c.Blocks)

	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			column := blocks.Column(x, z)
			v := uint16(0)
			for y := 127; y > 0; y-- {
				if column[y] != 0 {
					v = column[y]
					break
				}
			}
			fmt.Printf("%4x", v)
		}
		fmt.Println()
	}

	fmt.Println()
	return nil
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

type Blocks []uint16

type BlockColumn []uint16

func (b *Blocks) Get(x, y, z int) uint16 {
	return (*b)[y+(z*128+(x*128*16))]
}

func (b *Blocks) Column(x, z int) BlockColumn {
	var i = 128 * (z + x*16)
	return BlockColumn((*b)[i : i+128])
}
