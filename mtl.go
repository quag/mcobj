package main

import (
	"fmt"
	"io"
	"os"
)

func printMtl(w io.Writer, blockId byte) {
	if !noColor {
		fmt.Fprintln(w, "usemtl", blockId)
	}
}

func writeMtlFile(filename string) os.Error {
	if noColor {
		return nil
	}

	var outFile, outErr = os.Open(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if outErr != nil {
		return outErr
	}
	defer outFile.Close()

	var p = 0

	for i := 0; i < 256; i++ {
		var mtl *MTL
		if p < len(colors) && colors[p].blockId == byte(i) {
			mtl = &colors[p]
			p++
		} else {
			mtl = &MTL{byte(i), 0x7f7f7f, "Unknown", Voxel}
		}

		mtl.Print(outFile)
	}

	return nil
}

type MTL struct {
	blockId byte
	color   uint32
	name    string
	model   BlockModel
}

type BlockModel int

const (
	Voxel BlockModel = iota
	Mesh
)

func (mtl *MTL) Print(w io.Writer) {
	var (
		r = mtl.color >> 24
		g = mtl.color >> 16 & 0xff
		b = mtl.color >> 8 & 0xff
		a = mtl.color & 0xff
	)

	fmt.Fprintf(w, "# %s\nnewmtl %d\nKd %.4f %.4f %.4f\nd %.4f\nillum 1\n\n", mtl.name, mtl.blockId, float64(r)/255, float64(g)/255, float64(b)/255, float64(a)/255)
}

func IsMeshBlockId(blockId byte) bool {
	var value, present = nonVoxelBlocks[blockId]
	return value && present
}

func init() {
	nonVoxelBlocks = make(map[byte]bool)
	for _, mtl := range colors {
		if mtl.model != Voxel {
			nonVoxelBlocks[mtl.blockId] = true
		}
	}
}

var (
	nonVoxelBlocks map[byte]bool

	colors = []MTL{
		MTL{0, 0xff0000ff, "Air", Voxel},
		MTL{1, 0x757575ff, "Stone", Voxel},
		MTL{2, 0x2e6d05ff, "Grass", Voxel},
		MTL{3, 0x593d29ff, "Dirt", Voxel},
		MTL{4, 0x3d3d3dff, "Cobblestone", Voxel},
		MTL{5, 0x9f844dff, "Wooden Plank", Voxel},
		MTL{6, 0x7f7f7fff, "Sapling", Voxel},
		MTL{7, 0x070707ff, "Bedrock", Voxel},
		MTL{8, 0x3d6dff3f, "Water", Voxel},
		MTL{9, 0x3d6dff3f, "Stationary water", Voxel},
		MTL{10, 0x7f7f7fff, "Lava", Voxel},
		MTL{11, 0x7f7f7fff, "Stationary lava", Voxel},
		MTL{12, 0xbfb882ff, "Sand", Voxel},
		MTL{13, 0x7f7f7fff, "Gravel", Voxel},
		MTL{14, 0x7f7f7fff, "Gold ore", Voxel},
		MTL{15, 0x7f7f7fff, "Iron ore", Voxel},
		MTL{16, 0x7f7f7fff, "Coal ore", Voxel},
		MTL{17, 0x675231ff, "Wood", Voxel},
		MTL{18, 0x509026ff, "Leaves", Voxel},
		MTL{19, 0x7f7f7fff, "Sponge", Voxel},
		MTL{20, 0x7f7f7fff, "Glass", Voxel},
		MTL{21, 0x7f7f7fff, "Lapis Lazuli Ore", Voxel},
		MTL{22, 0x7f7f7fff, "Lapis Lazuli Block", Voxel},
		MTL{23, 0x7f7f7fff, "Dispenser", Voxel},
		MTL{24, 0xbfb882ff, "Sandstone", Voxel},
		MTL{25, 0x7f7f7fff, "Note Block", Voxel},
		MTL{35, 0x7f7f7fff, "Wool", Voxel},
		MTL{37, 0x7f7f7fff, "Yellow flower", Mesh},
		MTL{38, 0x7f7f7fff, "Red rose", Mesh},
		MTL{39, 0x7f7f7fff, "Brown Mushroom", Mesh},
		MTL{40, 0x7f7f7fff, "Red Mushroom", Mesh},
		MTL{41, 0x7f7f7fff, "Gold Block", Voxel},
		MTL{42, 0x7f7f7fff, "Iron Block", Voxel},
		MTL{43, 0x7f7f7fff, "Double Stone Slab", Voxel},
		MTL{44, 0x7f7f7fff, "Stone Slab", Voxel},
		MTL{45, 0x7f7f7fff, "Brick", Voxel},
		MTL{46, 0x7f7f7fff, "TNT", Voxel},
		MTL{47, 0x7f7f7fff, "Bookshelf", Voxel},
		MTL{48, 0x7f7f7fff, "Moss Stone", Voxel},
		MTL{49, 0x7f7f7fff, "Obsidian", Voxel},
		MTL{50, 0x7f7f7fff, "Torch", Mesh},
		MTL{51, 0x7f7f7fff, "Fire", Voxel},
		MTL{52, 0x7f7f7fff, "Monster Spawner", Voxel},
		MTL{53, 0x7f7f7fff, "Wooden Stairs", Voxel},
		MTL{54, 0x7f7f7fff, "Chest", Voxel},
		MTL{55, 0x7f7f7fff, "Redstone Wire", Voxel},
		MTL{56, 0x7f7f7fff, "Diamond Ore", Voxel},
		MTL{57, 0x7f7f7fff, "Diamond Block", Voxel},
		MTL{58, 0x7f7f7fff, "Workbench", Voxel},
		MTL{59, 0x7f7f7fff, "Crops", Voxel},
		MTL{60, 0x7f7f7fff, "Soil", Voxel},
		MTL{61, 0x7f7f7fff, "Furnace", Voxel},
		MTL{62, 0x7f7f7fff, "Burning Furnace", Voxel},
		MTL{63, 0x7f7f7fff, "Sign Post", Mesh},
		MTL{64, 0x7f7f7fff, "Wooden Door", Mesh},
		MTL{65, 0x7f7f7fff, "Ladder", Mesh},
		MTL{66, 0x7f7f7fff, "Minecart Tracks", Mesh},
		MTL{67, 0x7f7f7fff, "Cobblestone Stairs", Voxel}, // Should be Mesh
		MTL{68, 0x7f7f7fff, "Wall Sign", Mesh},
		MTL{69, 0x7f7f7fff, "Lever", Mesh},
		MTL{70, 0x7f7f7fff, "Stone Pressure Plate", Mesh},
		MTL{71, 0x7f7f7fff, "Iron Door", Mesh},
		MTL{72, 0x7f7f7fff, "Wooden Pressure Plate", Mesh},
		MTL{73, 0x7f7f7fff, "Redstone Ore", Voxel},
		MTL{74, 0x7f7f7fff, "Glowing Redstone Ore", Voxel},
		MTL{75, 0x7f7f7fff, "Redstone torch (\"off\" state)", Mesh},
		MTL{76, 0x7f7f7fff, "Redstone torch (\"on\" state)", Mesh},
		MTL{77, 0x7f7f7fff, "Stone Button", Mesh},
		MTL{78, 0xffffffff, "Snow", Voxel},
		MTL{79, 0x70a0ffff, "Ice", Voxel},
		MTL{80, 0xffffffff, "Snow Block", Voxel},
		MTL{81, 0x7f7f7fff, "Cactus", Voxel},
		MTL{82, 0x7f7f7fff, "Clay", Voxel},
		MTL{83, 0x7f7f7fff, "Sugar Cane", Voxel},
		MTL{84, 0x7f7f7fff, "Jukebox", Voxel},
		MTL{85, 0x7f7f7fff, "Fence", Voxel},
		MTL{86, 0x7f7f7fff, "Pumpkin", Voxel},
		MTL{87, 0x7f7f7fff, "Netherrack", Voxel},
		MTL{88, 0x7f7f7fff, "Soul Sand", Voxel},
		MTL{89, 0x7f7f7fff, "Glowstone", Voxel},
		MTL{90, 0x7f7f7fff, "Portal", Voxel},
		MTL{91, 0x7f7f7fff, "Jack-O-Lantern", Voxel},
		MTL{92, 0xffffffff, "Cake Block", Voxel},
	}
)
