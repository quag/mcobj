package main

import (
	"fmt"
	"github.com/quag/mcobj/nbt"
	"github.com/quag/mcobj/mcworld"
	"os"
)

func main() {
	world := mcworld.OpenWorld("/home/jonathan/Library/Minecraft/saves/mca")
	pool, poolErr := world.ChunkPool(&mcworld.AllChunksMask{})
	if poolErr != nil {
		fmt.Fprintln(os.Stderr, "Chunk pool error:", poolErr)
		return
	}

	box := pool.BoundingBox()

	fmt.Println(box)
	for x := box.X0; x <= box.X1; x++ {
		for z := box.Z0; z <= box.Z1; z++ {
			if pool.Pop(x, z) {
				fmt.Println(x, z, pool.Remaining())

				chunk, chunkErr := world.OpenChunk(x, z)
				if chunkErr != nil {
					fmt.Fprintln(os.Stderr, chunkErr)
					return
				}
				nbt.Explain(chunk, os.Stdout)
			}
		}
	}
}
