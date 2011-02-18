package main

import (
	"fmt"
	"io"
	"os"
)

func printMtl(w io.Writer, blockId uint16) {
	if !noColor {
		if blockId&0xff == blockId {
			fmt.Fprintln(w, "usemtl", blockId)
		} else {
			fmt.Fprintf(w, "usemtl %d_%d", blockId&0xff, blockId>>8)
		}
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
		var matched = false

		for p < len(colors) && colors[p].blockId == byte(i) {
			matched = true
			colors[p].Print(outFile)
			p++
		}

		if !matched {
			var mtl = MTL{byte(i), 255, 0x7f7f7f, "Unknown", Voxel}
			mtl.Print(outFile)
		}
	}

	return nil
}

type MTL struct {
	blockId  byte
	metadata byte
	color    uint32
	name     string
	model    BlockModel
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

	if mtl.metadata == 255 {
		fmt.Fprintf(w, "# %s\nnewmtl %d\nKd %.4f %.4f %.4f\nd %.4f\nillum 1\n\n", mtl.name, mtl.blockId, float64(r)/255, float64(g)/255, float64(b)/255, float64(a)/255)
	} else {
		fmt.Fprintf(w, "# %s\nnewmtl %d_%d\nKd %.4f %.4f %.4f\nd %.4f\nillum 1\n\n", mtl.name, mtl.blockId, mtl.metadata, float64(r)/255, float64(g)/255, float64(b)/255, float64(a)/255)
	}
}

func (mtl *MTL) colorId() uint16 {
	var id = uint16(mtl.blockId)
	if mtl.metadata != 255 {
		id += uint16(mtl.metadata) << 8
	}
	return id
}

func IsMeshBlockId(blockId uint16) bool {
	var value, present = nonVoxelBlocks[blockId]
	return value && present
}

func init() {
	nonVoxelBlocks = make(map[uint16]bool)
	for _, mtl := range colors {
		if mtl.model != Voxel {
			nonVoxelBlocks[mtl.colorId()] = true
		}
	}
}

var (
	nonVoxelBlocks map[uint16]bool

	colors = []MTL{
		MTL{0, 255, 0xff0000ff, "Air", Voxel},
		MTL{1, 255, 0x7f7f7fff, "Stone", Voxel},
		MTL{2, 255, 0x509026ff, "Grass", Voxel},
		MTL{3, 255, 0x79553aff, "Dirt", Voxel},
		MTL{4, 255, 0xa8a8a8ff, "Cobblestone", Voxel},
		MTL{5, 255, 0x9e8052ff, "Wooden Plank", Voxel},
		MTL{6, 255, 0x006400ff, "Sapling", Voxel},
		MTL{7, 255, 0x575757ff, "Bedrock", Voxel},
		MTL{8, 255, 0x009affaa, "Water", Voxel},
		MTL{9, 255, 0x002affff, "Stationary water", Voxel},
		MTL{10, 255, 0x002affff, "Lava", Voxel},
		MTL{11, 255, 0xfc4a00ff, "Stationary lava", Voxel},
		MTL{12, 255, 0xcfc393ff, "Sand", Voxel},
		MTL{13, 255, 0x9a9b9fff, "Gravel", Voxel},
		MTL{14, 255, 0xfff144ff, "Gold ore", Voxel},
		MTL{15, 255, 0xe2c0aaff, "Iron ore", Voxel},
		MTL{16, 255, 0x454545ff, "Coal ore", Voxel},
		MTL{17, 255, 0x695433ff, "Wood", Voxel},
		MTL{18, 255, 0x2e6d05ff, "Leaves", Voxel},
		MTL{19, 255, 0xe5e54eff, "Sponge", Voxel},
		MTL{20, 255, 0xffffff10, "Glass", Voxel},
		MTL{21, 255, 0x224da1ff, "Lapis Lazuli Ore", Voxel},
		MTL{22, 255, 0x224dffff, "Lapis Lazuli Block", Voxel},
		MTL{23, 255, 0x7f7f7fff, "Dispenser", Voxel},
		MTL{24, 255, 0xdad2abff, "Sandstone", Voxel},
		MTL{25, 255, 0x9b664bff, "Note Block", Voxel},
		MTL{35, 0, 0xd1d1d1ff, "Wool - White", Voxel},
		MTL{35, 1, 0xe97a2eff, "Wool - Orange", Voxel},
		MTL{35, 2, 0xbc3ec7ff, "Wool - Magenta", Voxel},
		MTL{35, 3, 0x5980d0ff, "Wool - Light Blue", Voxel},
		MTL{35, 4, 0xb6a918ff, "Wool - Yellow", Voxel},
		MTL{35, 5, 0x37b32cff, "Wool - Light Green", Voxel},
		MTL{35, 6, 0xd57690ff, "Wool - Pink", Voxel},
		MTL{35, 7, 0x3f3f3fff, "Wool - Gray", Voxel},
		MTL{35, 8, 0x949d9dff, "Wool - Light Gray", Voxel},
		MTL{35, 9, 0xbc3ec7ff, "Wool - Cyan", Voxel},
		MTL{35, 10, 0x7a2fbcff, "Wool - Purple", Voxel},
		MTL{35, 11, 0x243091ff, "Wool - Blue", Voxel},
		MTL{35, 12, 0x51301aff, "Wool - Brown", Voxel},
		MTL{35, 13, 0x344817ff, "Wool - Dark Green", Voxel},
		MTL{35, 14, 0x9b2a26ff, "Wool - Red", Voxel},
		MTL{35, 15, 0x171313ff, "Wool - Black", Voxel},
		MTL{37, 255, 0xf1f902ff, "Yellow flower", Mesh},
		MTL{38, 255, 0xf11102ff, "Red rose", Mesh},
		MTL{39, 255, 0xba7038ff, "Brown Mushroom", Mesh},
		MTL{40, 255, 0xcf3626ff, "Red Mushroom", Mesh},
		MTL{41, 255, 0xfff199ff, "Gold Block", Voxel},
		MTL{42, 255, 0xdededeff, "Iron Block", Voxel},
		MTL{43, 255, 0xc8c8c8ff, "Double Stone Slab", Voxel},
		MTL{44, 255, 0xa8a8a8ff, "Stone Slab", Voxel},
		MTL{45, 255, 0xa14f38ff, "Brick", Voxel},
		MTL{46, 255, 0xdb441aff, "TNT", Voxel},
		MTL{47, 255, 0x9f844dff, "Bookshelf", Voxel},
		MTL{48, 255, 0xa7a8a7ff, "Moss Stone", Voxel},
		MTL{49, 255, 0x101019ff, "Obsidian", Voxel},
		MTL{50, 255, 0xffae0c99, "Torch", Mesh},
		MTL{51, 255, 0xff000099, "Fire", Voxel},
		MTL{52, 255, 0x929292ff, "Monster Spawner", Voxel},
		MTL{53, 255, 0x9c743aff, "Wooden Stairs", Voxel},
		MTL{54, 255, 0xab792dff, "Chest", Voxel},
		MTL{55, 255, 0xcc0000ff, "Redstone Wire", Voxel},
		MTL{56, 255, 0x50aba6ff, "Diamond Ore", Voxel},
		MTL{57, 255, 0x69dfdaff, "Diamond Block", Voxel},
		MTL{58, 255, 0xab9472ff, "Workbench", Voxel},
		MTL{59, 255, 0x1a7508ff, "Crops", Voxel},
		MTL{60, 255, 0x573d2aff, "Soil", Voxel},
		MTL{61, 255, 0x919191ff, "Furnace", Voxel},
		MTL{62, 255, 0x919191ff, "Burning Furnace", Voxel},
		MTL{63, 255, 0xd6b88bff, "Sign Post", Mesh},
		MTL{64, 255, 0x82592cff, "Wooden Door", Mesh},
		MTL{65, 255, 0xab8944ff, "Ladder", Mesh},
		MTL{66, 255, 0xc7c7c7ff, "Minecart Tracks", Mesh},
		MTL{67, 255, 0x919191ff, "Cobblestone Stairs", Voxel}, // Should be Mesh
		MTL{68, 255, 0xd6b88bff, "Wall Sign", Mesh},
		MTL{69, 255, 0xd6b88bff, "Lever", Mesh},
		MTL{70, 255, 0x919191ff, "Stone Pressure Plate", Mesh},
		MTL{71, 255, 0xbababaff, "Iron Door", Mesh},
		MTL{72, 255, 0x7a6340ff, "Wooden Pressure Plate", Mesh},
		MTL{73, 255, 0xbd0000ff, "Redstone Ore", Voxel},
		MTL{74, 255, 0xff0000ff, "Glowing Redstone Ore", Voxel},
		MTL{75, 255, 0xbd0000ff, "Redstone torch (\"off\" state)", Mesh},
		MTL{76, 255, 0xff0000ff, "Redstone torch (\"on\" state)", Mesh},
		MTL{77, 255, 0x919191ff, "Stone Button", Mesh},
		MTL{78, 255, 0xfefefeff, "Snow", Voxel},
		MTL{79, 255, 0x77a9ffff, "Ice", Voxel},
		MTL{80, 255, 0xfcfcfcff, "Snow Block", Voxel},
		MTL{81, 255, 0x11801eff, "Cactus", Voxel},
		MTL{82, 255, 0xaaaebeff, "Clay", Voxel},
		MTL{83, 255, 0x3c6e0aff, "Sugar Cane", Voxel},
		MTL{84, 255, 0x9b664bff, "Jukebox", Voxel},
		MTL{85, 255, 0x7a6340ff, "Fence", Voxel},
		MTL{86, 255, 0xa05a0bff, "Pumpkin", Voxel},
		MTL{87, 255, 0xb66b6bff, "Netherrack", Voxel},
		MTL{88, 255, 0x453125ff, "Soul Sand", Voxel},
		MTL{89, 255, 0xfff894ff, "Glowstone", Voxel},
		MTL{90, 255, 0x381d55ff, "Portal", Voxel},
		MTL{91, 255, 0xe9b416ff, "Jack-O-Lantern", Voxel},
		MTL{92, 255, 0xbd9075ff, "Cake Block", Voxel},
	}
)
