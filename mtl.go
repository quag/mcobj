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
		MTL{1, 0x7f7f7fff, "Stone", Voxel},
		MTL{2, 0x2e6d05ff, "Grass", Voxel},
		MTL{3, 0x79553aff, "Dirt", Voxel},
		MTL{4, 0xa8a8a8ff, "Cobblestone", Voxel},
		MTL{5, 0xbc9862ff, "Wooden Plank", Voxel},
		MTL{6, 0x006400ff, "Sapling", Voxel},
		MTL{7, 0x575757ff, "Bedrock", Voxel},
		MTL{8, 0x009affff, "Water", Voxel},
		MTL{9, 0x002affff, "Stationary water", Voxel},
		MTL{10, 0x002affff, "Lava", Voxel},
		MTL{11, 0xfc4a00ff, "Stationary lava", Voxel},
		MTL{12, 0xf2e7bbff, "Sand", Voxel},
		MTL{13, 0x9a9b9fff, "Gravel", Voxel},
		MTL{14, 0xfff144ff, "Gold ore", Voxel},
		MTL{15, 0xe2c0aaff, "Iron ore", Voxel},
		MTL{16, 0x454545ff, "Coal ore", Voxel},
		MTL{17, 0x695433ff, "Wood", Voxel},
		MTL{18, 0x509026ff, "Leaves", Voxel},
		MTL{19, 0xe5e54eff, "Sponge", Voxel},
		MTL{20, 0xffffff10, "Glass", Voxel},
		MTL{21, 0x224da1ff, "Lapis Lazuli Ore", Voxel},
		MTL{22, 0x224dffff, "Lapis Lazuli Block", Voxel},
		MTL{23, 0x7f7f7fff, "Dispenser", Voxel},
		MTL{24, 0xdad2abff, "Sandstone", Voxel},
		MTL{25, 0x9b664bff, "Note Block", Voxel},
		MTL{35, 0xd1d1d1ff, "Wool", Voxel},
		MTL{37, 0xf1f902ff, "Yellow flower", Mesh},
		MTL{38, 0xf11102ff, "Red rose", Mesh},
		MTL{39, 0xba7038ff, "Brown Mushroom", Mesh},
		MTL{40, 0xcf3626ff, "Red Mushroom", Mesh},
		MTL{41, 0xfff199ff, "Gold Block", Voxel},
		MTL{42, 0xdededeff, "Iron Block", Voxel},
		MTL{43, 0xc8c8c8ff, "Double Stone Slab", Voxel},
		MTL{44, 0xa8a8a8ff, "Stone Slab", Voxel},
		MTL{45, 0xa14f38ff, "Brick", Voxel},
		MTL{46, 0xdb441aff, "TNT", Voxel},
		MTL{47, 0x9f844dff, "Bookshelf", Voxel},
		MTL{48, 0xa7a8a7ff, "Moss Stone", Voxel},
		MTL{49, 0x101019ff, "Obsidian", Voxel},
		MTL{50, 0xffae0c99, "Torch", Mesh},
		MTL{51, 0xff000099, "Fire", Voxel},
		MTL{52, 0x929292ff, "Monster Spawner", Voxel},
		MTL{53, 0x9c743aff, "Wooden Stairs", Voxel},
		MTL{54, 0xab792dff, "Chest", Voxel},
		MTL{55, 0xcc0000ff, "Redstone Wire", Voxel},
		MTL{56, 0x50aba6ff, "Diamond Ore", Voxel},
		MTL{57, 0x69dfdaff, "Diamond Block", Voxel},
		MTL{58, 0xab9472ff, "Workbench", Voxel},
		MTL{59, 0x1a7508ff, "Crops", Voxel},
		MTL{60, 0x573d2aff, "Soil", Voxel},
		MTL{61, 0x919191ff, "Furnace", Voxel},
		MTL{62, 0x919191ff, "Burning Furnace", Voxel},
		MTL{63, 0xd6b88bff, "Sign Post", Mesh},
		MTL{64, 0x82592cff, "Wooden Door", Mesh},
		MTL{65, 0xab8944ff, "Ladder", Mesh},
		MTL{66, 0xc7c7c7ff, "Minecart Tracks", Mesh},
		MTL{67, 0x919191ff, "Cobblestone Stairs", Voxel}, // Should be Mesh
		MTL{68, 0xd6b88bff, "Wall Sign", Mesh},
		MTL{69, 0xd6b88bff, "Lever", Mesh},
		MTL{70, 0x919191ff, "Stone Pressure Plate", Mesh},
		MTL{71, 0xbababaff, "Iron Door", Mesh},
		MTL{72, 0x7a6340ff, "Wooden Pressure Plate", Mesh},
		MTL{73, 0xbd0000ff, "Redstone Ore", Voxel},
		MTL{74, 0xff0000ff, "Glowing Redstone Ore", Voxel},
		MTL{75, 0xbd0000ff, "Redstone torch (\"off\" state)", Mesh},
		MTL{76, 0xff0000ff, "Redstone torch (\"on\" state)", Mesh},
		MTL{77, 0x919191ff, "Stone Button", Mesh},
		MTL{78, 0xfefefeff, "Snow", Voxel},
		MTL{79, 0x77a9ffff, "Ice", Voxel},
		MTL{80, 0xfcfcfcff, "Snow Block", Voxel},
		MTL{81, 0x11801eff, "Cactus", Voxel},
		MTL{82, 0xaaaebeff, "Clay", Voxel},
		MTL{83, 0x3c6e0aff, "Sugar Cane", Voxel},
		MTL{84, 0x9b664bff, "Jukebox", Voxel},
		MTL{85, 0x7a6340ff, "Fence", Voxel},
		MTL{86, 0xa05a0bff, "Pumpkin", Voxel},
		MTL{87, 0xb66b6bff, "Netherrack", Voxel},
		MTL{88, 0x453125ff, "Soul Sand", Voxel},
		MTL{89, 0xfff894ff, "Glowstone", Voxel},
		MTL{90, 0x381d55ff, "Portal", Voxel},
		MTL{91, 0xe9b416ff, "Jack-O-Lantern", Voxel},
		MTL{92, 0xbd9075ff, "Cake Block", Voxel},
	}
)
