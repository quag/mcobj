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
			var mtl = MTL{byte(i), 255, 0x7f7f7f, "Unknown"}
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
}

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

var (
	colors = []MTL{
		MTL{0, 255, 0xff0000ff, "Air"},
		MTL{1, 255, 0x7f7f7fff, "Stone"},
		MTL{2, 255, 0x509026ff, "Grass"},
		MTL{3, 255, 0x79553aff, "Dirt"},
		MTL{4, 255, 0xa8a8a8ff, "Cobblestone"},
		MTL{5, 255, 0x9e8052ff, "Wooden Plank"},
		MTL{6, 255, 0x006400ff, "Sapling"},
		MTL{7, 255, 0x575757ff, "Bedrock"},
		MTL{8, 255, 0x009affaa, "Water"},
		MTL{9, 255, 0x002affff, "Stationary water"},
		MTL{10, 255, 0x002affff, "Lava"},
		MTL{11, 255, 0xfc4a00ff, "Stationary lava"},
		MTL{12, 255, 0xcfc393ff, "Sand"},
		MTL{13, 255, 0x9a9b9fff, "Gravel"},
		MTL{14, 255, 0xfff144ff, "Gold ore"},
		MTL{15, 255, 0xe2c0aaff, "Iron ore"},
		MTL{16, 255, 0x454545ff, "Coal ore"},
		MTL{17, 255, 0x695433ff, "Wood"},
		MTL{18, 255, 0x2e6d05ff, "Leaves"},
		MTL{19, 255, 0xe5e54eff, "Sponge"},
		MTL{20, 255, 0xffffff10, "Glass"},
		MTL{21, 255, 0x224da1ff, "Lapis Lazuli Ore"},
		MTL{22, 255, 0x224dffff, "Lapis Lazuli Block"},
		MTL{23, 255, 0x7f7f7fff, "Dispenser"},
		MTL{24, 255, 0xdad2abff, "Sandstone"},
		MTL{25, 255, 0x9b664bff, "Note Block"},
		MTL{35, 0, 0xd1d1d1ff, "Wool - White"},
		MTL{35, 1, 0xe97a2eff, "Wool - Orange"},
		MTL{35, 2, 0xbc3ec7ff, "Wool - Magenta"},
		MTL{35, 3, 0x5980d0ff, "Wool - Light Blue"},
		MTL{35, 4, 0xb6a918ff, "Wool - Yellow"},
		MTL{35, 5, 0x37b32cff, "Wool - Light Green"},
		MTL{35, 6, 0xd57690ff, "Wool - Pink"},
		MTL{35, 7, 0x3f3f3fff, "Wool - Gray"},
		MTL{35, 8, 0x949d9dff, "Wool - Light Gray"},
		MTL{35, 9, 0xbc3ec7ff, "Wool - Cyan"},
		MTL{35, 10, 0x7a2fbcff, "Wool - Purple"},
		MTL{35, 11, 0x243091ff, "Wool - Blue"},
		MTL{35, 12, 0x51301aff, "Wool - Brown"},
		MTL{35, 13, 0x344817ff, "Wool - Dark Green"},
		MTL{35, 14, 0x9b2a26ff, "Wool - Red"},
		MTL{35, 15, 0x171313ff, "Wool - Black"},
		MTL{37, 255, 0xf1f902ff, "Yellow flower"},
		MTL{38, 255, 0xf11102ff, "Red rose"},
		MTL{39, 255, 0xba7038ff, "Brown Mushroom"},
		MTL{40, 255, 0xcf3626ff, "Red Mushroom"},
		MTL{41, 255, 0xfff199ff, "Gold Block"},
		MTL{42, 255, 0xdededeff, "Iron Block"},
		MTL{43, 255, 0xc8c8c8ff, "Double Stone Slab"},
		MTL{44, 255, 0xa8a8a8ff, "Stone Slab"},
		MTL{45, 255, 0xa14f38ff, "Brick"},
		MTL{46, 255, 0xdb441aff, "TNT"},
		MTL{47, 255, 0x9f844dff, "Bookshelf"},
		MTL{48, 255, 0xa7a8a7ff, "Moss Stone"},
		MTL{49, 255, 0x101019ff, "Obsidian"},
		MTL{50, 255, 0xffae0c99, "Torch"},
		MTL{51, 255, 0xff000099, "Fire"},
		MTL{52, 255, 0x929292ff, "Monster Spawner"},
		MTL{53, 255, 0x9c743aff, "Wooden Stairs"},
		MTL{54, 255, 0xab792dff, "Chest"},
		MTL{55, 255, 0xcc0000ff, "Redstone Wire"},
		MTL{56, 255, 0x50aba6ff, "Diamond Ore"},
		MTL{57, 255, 0x69dfdaff, "Diamond Block"},
		MTL{58, 255, 0xab9472ff, "Workbench"},
		MTL{59, 255, 0x1a7508ff, "Crops"},
		MTL{60, 255, 0x573d2aff, "Soil"},
		MTL{61, 255, 0x919191ff, "Furnace"},
		MTL{62, 255, 0x919191ff, "Burning Furnace"},
		MTL{63, 255, 0xd6b88bff, "Sign Post"},
		MTL{64, 255, 0x82592cff, "Wooden Door"},
		MTL{65, 255, 0xab8944ff, "Ladder"},
		MTL{66, 255, 0xc7c7c7ff, "Minecart Tracks"},
		MTL{67, 255, 0x919191ff, "Cobblestone Stairs"}, // Should be Mesh
		MTL{68, 255, 0xd6b88bff, "Wall Sign"},
		MTL{69, 255, 0xd6b88bff, "Lever"},
		MTL{70, 255, 0x919191ff, "Stone Pressure Plate"},
		MTL{71, 255, 0xbababaff, "Iron Door"},
		MTL{72, 255, 0x7a6340ff, "Wooden Pressure Plate"},
		MTL{73, 255, 0xbd0000ff, "Redstone Ore"},
		MTL{74, 255, 0xff0000ff, "Glowing Redstone Ore"},
		MTL{75, 255, 0xbd0000ff, "Redstone torch (\"off\" state)"},
		MTL{76, 255, 0xff0000ff, "Redstone torch (\"on\" state)"},
		MTL{77, 255, 0x919191ff, "Stone Button"},
		MTL{78, 255, 0xfefefeff, "Snow"},
		MTL{79, 255, 0x77a9ffff, "Ice"},
		MTL{80, 255, 0xfcfcfcff, "Snow Block"},
		MTL{81, 255, 0x11801eff, "Cactus"},
		MTL{82, 255, 0xaaaebeff, "Clay"},
		MTL{83, 255, 0x3c6e0aff, "Sugar Cane"},
		MTL{84, 255, 0x9b664bff, "Jukebox"},
		MTL{85, 255, 0x7a6340ff, "Fence"},
		MTL{86, 255, 0xa05a0bff, "Pumpkin"},
		MTL{87, 255, 0xb66b6bff, "Netherrack"},
		MTL{88, 255, 0x453125ff, "Soul Sand"},
		MTL{89, 255, 0xfff894ff, "Glowstone"},
		MTL{90, 255, 0x381d55ff, "Portal"},
		MTL{91, 255, 0xe9b416ff, "Jack-O-Lantern"},
		MTL{92, 255, 0xbd9075ff, "Cake Block"},
	}
)
