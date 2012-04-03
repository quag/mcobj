package main

import (
	"fmt"
	"github.com/quag/mcobj/mcworld"
	"github.com/quag/mcobj/nbt"
	"image"
	"image/png"
	"io"
	"os"
)

func main() {
	//dir := "/Users/jonathan/Library/Application Support/minecraft/saves/World1"
	dir := "/Users/jonathan/Library/Application Support/minecraft/saves/New World"
	//dir := "/Users/jonathan/Library/Application Support/minecraft/saves/1.8.1"
	//dir := "../../../world"
	//mask := &mcworld.AllChunksMask{}
	mask := &mcworld.RectangleChunkMask{-100, -100, 100, 100}

	world := mcworld.OpenWorld(dir)
	chunks, box, err := ZigZagChunks(world, mask)
	if err != nil {
		fmt.Println("ZigZagChunks:", err)
		return
	}

	width, height := 16*(box.X1-box.X0), 16*(box.Z1-box.Z0)
	xoffset, zoffset := -16*box.X0, -16*box.Z0

	fmt.Println(box, width, height)

	img := image.NewNRGBA(image.Rect(0, 0, width, height))

	for chunk := range chunks {
		//fmt.Println(chunk.X, chunk.Z, xoffset, zoffset)
		err := useChunk(chunk, img, xoffset+16*chunk.X, zoffset+16*chunk.Z)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf(".")
	}

	pngFile, err := os.Create("map.png")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer pngFile.Close()
	png.Encode(pngFile, img)
}

func useChunk(chunk *Chunk, img *image.NRGBA, xoffset, zoffset int) error {
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
			//fmt.Printf("%7x", color[v&0xff])
			img.Set(xoffset+x, zoffset+z, rgb(color[v&0xff]))
			//fmt.Printf("%7x", img.At(x, z))
		}
		//fmt.Println()
	}

	//fmt.Println()
	return nil
}

type Chunk struct {
	opener mcworld.ChunkOpener
	X, Z   int
}

func (c *Chunk) Open() (io.ReadCloser, error) {
	return c.opener.OpenChunk(c.X, c.Z)
}

func zigzag(n int) int {
	return (n << 1) ^ (n >> 31)
}

func unzigzag(n int) int {
	return (n >> 1) ^ (-(n & 1))
}

func ZigZagChunks(world mcworld.World, mask mcworld.ChunkMask) (chan *Chunk, *mcworld.BoundingBox, error) {
	pool, err := world.ChunkPool(mask)
	if err != nil {
		return nil, &mcworld.BoundingBox{}, err
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

	return c, pool.BoundingBox(), nil
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

type rgb uint32

func (c rgb) RGBA() (r, g, b, a uint32) {
	r = (uint32(c) >> 16) << 8
	g = (uint32(c) >> 8 & 0xff) << 8
	b = (uint32(c) >> 0 & 0xff) << 8
	a = 0xff << 8
	return
}

var (
	color []uint32
)

func init() {
	color = make([]uint32, 256)
	color[0] = 0xfefeff
	color[1] = 0x7d7d7d
	color[2] = 0x52732c
	color[3] = 0x866043
	color[4] = 0x757575
	color[5] = 0x9d804f
	color[6] = 0x5d7e1e
	color[7] = 0x545454
	color[8] = 0x009aff
	color[9] = 0x009aff
	color[10] = 0xf54200
	color[11] = 0xf54200
	color[12] = 0xdad29e
	color[13] = 0x887f7e
	color[14] = 0x908c7d
	color[15] = 0x88837f
	color[16] = 0x737373
	color[17] = 0x665132
	color[18] = 0x1c4705
	color[19] = 0xb7b739
	color[20] = 0xffffff
	color[21] = 0x667087
	color[22] = 0x1d47a6
	color[23] = 0x6c6c6c
	color[24] = 0xd5cd94
	color[25] = 0x654433
	color[26] = 0x8f1717
	color[26] = 0xaf7475
	color[27] = 0x87714e
	color[28] = 0x766251
	color[30] = 0xdadada
	color[31] = 0x7c4f19
	color[32] = 0x7c4f19
	color[35] = 0xdedede
	color[37] = 0xc1c702
	color[38] = 0xcb060a
	color[39] = 0x967158
	color[40] = 0xc53c3f
	color[41] = 0xfaec4e
	color[42] = 0xe6e6e6
	color[43] = 0xa7a7a7
	color[44] = 0xa7a7a7
	color[45] = 0x9c6e62
	color[46] = 0xa6553f
	color[47] = 0x6c583a
	color[48] = 0x5b6c5b
	color[49] = 0x14121e
	color[50] = 0xffda66
	color[51] = 0xff7700
	color[52] = 0x1d4f72
	color[53] = 0x9d804f
	color[54] = 0x835e25
	color[55] = 0xcb0000
	color[56] = 0x828c8f
	color[57] = 0x64dcd6
	color[58] = 0x6b472b
	color[59] = 0x83c144
	color[60] = 0x4b290e
	color[61] = 0x4e4e4e
	color[62] = 0x7d6655
	color[63] = 0x9d804f
	color[64] = 0x9d804f
	color[65] = 0x9d804f
	color[66] = 0x75664c
	color[67] = 0x757575
	color[68] = 0x9d804f
	color[69] = 0x9d804f
	color[70] = 0x7d7d7d
	color[71] = 0xb2b2b2
	color[72] = 0x9d804f
	color[73] = 0x856b6b
	color[74] = 0xbd6b6b
	color[75] = 0x440000
	color[76] = 0xfe0000
	color[77] = 0x7d7d7d
	color[78] = 0xf0fbfb
	color[79] = 0x7daeff
	color[80] = 0xf0fbfb
	color[81] = 0x0d6418
	color[82] = 0x9fa5b1
	color[83] = 0x83c447
	color[84] = 0x6b4937
	color[85] = 0x9d804f
	color[86] = 0xc57918
	color[87] = 0x6e3533
	color[88] = 0x554134
	color[89] = 0x897141
	color[90] = 0x381d55
	color[91] = 0xb9861d
	color[92] = 0xe5cecf
	color[93] = 0x989494
	color[94] = 0xa19494
	color[95] = 0x835e25
	color[96] = 0x81602f
}
