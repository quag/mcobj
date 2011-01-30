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
			mtl = &MTL{byte(i), 0x7f7f7f, "Unknown"}
		}

		mtl.Print(outFile)
	}

	return nil
}

type MTL struct {
	blockId byte
	color   uint32
	name    string
}

func (mtl *MTL) Print(w io.Writer) {
	var (
		r = mtl.color >> 24
		g = mtl.color >> 16 & 0xff
		b = mtl.color >> 8 & 0xff
		a = mtl.color & 0xff
	)

	fmt.Fprintf(w, "# %s\nnewmtl %d\nKd %.4f %.4f %.4f\nd %.4f\nillum 1\n\n", mtl.name, mtl.blockId, float64(r)/255, float64(g)/255, float64(b)/255, float64(a)/255)
}

var (
	colors = []MTL{
		MTL{0, 0xff0000ff, "Air"},
		MTL{1, 0x757575ff, "Stone"},
		MTL{2, 0x2e6d05ff, "Grass"},
		MTL{3, 0x593d29ff, "Dirt"},
		MTL{4, 0x3d3d3dff, "Cobblestone"},
		MTL{5, 0x9f844dff, "Wooden Plank"},
		MTL{6, 0x7f7f7fff, "Sapling"},
		MTL{7, 0x070707ff, "Bedrock"},
		MTL{8, 0x3d6dff7f, "Water"},
		MTL{9, 0x3d6dff7f, "Stationary water"},
		MTL{10, 0x7f7f7fff, "Lava"},
		MTL{11, 0x7f7f7fff, "Stationary lava"},
		MTL{12, 0xbfb882ff, "Sand"},
		MTL{13, 0x7f7f7fff, "Gravel"},
		MTL{14, 0x7f7f7fff, "Gold ore"},
		MTL{15, 0x7f7f7fff, "Iron ore"},
		MTL{16, 0x7f7f7fff, "Coal ore"},
		MTL{17, 0x675231ff, "Wood"},
		MTL{18, 0x509026ff, "Leaves"},
		MTL{19, 0x7f7f7fff, "Sponge"},
		MTL{20, 0x7f7f7fff, "Glass"},
		MTL{21, 0x7f7f7fff, "Lapis Lazuli Ore"},
		MTL{22, 0x7f7f7fff, "Lapis Lazuli Block"},
		MTL{23, 0x7f7f7fff, "Dispenser"},
		MTL{24, 0xbfb882ff, "Sandstone"},
		MTL{25, 0x7f7f7fff, "Note Block"},
		MTL{35, 0x7f7f7fff, "Wool"},
		MTL{37, 0x7f7f7fff, "Yellow flower"},
		MTL{38, 0x7f7f7fff, "Red rose"},
		MTL{39, 0x7f7f7fff, "Brown Mushroom"},
		MTL{40, 0x7f7f7fff, "Red Mushroom"},
		MTL{41, 0x7f7f7fff, "Gold Block"},
		MTL{42, 0x7f7f7fff, "Iron Block"},
		MTL{43, 0x7f7f7fff, "Double Stone Slab"},
		MTL{44, 0x7f7f7fff, "Stone Slab"},
		MTL{45, 0x7f7f7fff, "Brick"},
		MTL{46, 0x7f7f7fff, "TNT"},
		MTL{47, 0x7f7f7fff, "Bookshelf"},
		MTL{48, 0x7f7f7fff, "Moss Stone"},
		MTL{49, 0x7f7f7fff, "Obsidian"},
		MTL{50, 0x7f7f7fff, "Torch"},
		MTL{51, 0x7f7f7fff, "Fire"},
		MTL{52, 0x7f7f7fff, "Monster Spawner"},
		MTL{53, 0x7f7f7fff, "Wooden Stairs"},
		MTL{54, 0x7f7f7fff, "Chest"},
		MTL{55, 0x7f7f7fff, "Redstone Wire"},
		MTL{56, 0x7f7f7fff, "Diamond Ore"},
		MTL{57, 0x7f7f7fff, "Diamond Block"},
		MTL{58, 0x7f7f7fff, "Workbench"},
		MTL{59, 0x7f7f7fff, "Crops"},
		MTL{60, 0x7f7f7fff, "Soil"},
		MTL{61, 0x7f7f7fff, "Furnace"},
		MTL{62, 0x7f7f7fff, "Burning Furnace"},
		MTL{63, 0x7f7f7fff, "Sign Post"},
		MTL{64, 0x7f7f7fff, "Wooden Door"},
		MTL{65, 0x7f7f7fff, "Ladder"},
		MTL{66, 0x7f7f7fff, "Minecart Tracks"},
		MTL{67, 0x7f7f7fff, "Cobblestone Stairs"},
		MTL{68, 0x7f7f7fff, "Wall Sign"},
		MTL{69, 0x7f7f7fff, "Lever"},
		MTL{70, 0x7f7f7fff, "Stone Pressure Plate"},
		MTL{71, 0x7f7f7fff, "Iron Door"},
		MTL{72, 0x7f7f7fff, "Wooden Pressure Plate"},
		MTL{73, 0x7f7f7fff, "Redstone Ore"},
		MTL{74, 0x7f7f7fff, "Glowing Redstone Ore"},
		MTL{75, 0x7f7f7fff, "Redstone torch (\"off\" state)"},
		MTL{76, 0x7f7f7fff, "Redstone torch (\"on\" state)"},
		MTL{77, 0x7f7f7fff, "Stone Button"},
		MTL{78, 0xffffffff, "Snow"},
		MTL{79, 0x70a0ffff, "Ice"},
		MTL{80, 0xffffffff, "Snow Block"},
		MTL{81, 0x7f7f7fff, "Cactus"},
		MTL{82, 0x7f7f7fff, "Clay"},
		MTL{83, 0x7f7f7fff, "Sugar Cane"},
		MTL{84, 0x7f7f7fff, "Jukebox"},
		MTL{85, 0x7f7f7fff, "Fence"},
		MTL{86, 0x7f7f7fff, "Pumpkin"},
		MTL{87, 0x7f7f7fff, "Netherrack"},
		MTL{88, 0x7f7f7fff, "Soul Sand"},
		MTL{89, 0x7f7f7fff, "Glowstone"},
		MTL{90, 0x7f7f7fff, "Portal"},
		MTL{91, 0x7f7f7fff, "Jack-O-Lantern"},
		MTL{92, 0xffffffff, "Cake Block"},
	}
)
