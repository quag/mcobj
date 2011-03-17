package main

import (
	"fmt"
	"io"
	"os"
)

func printMtl(w io.Writer, blockId uint16) {
	if !noColor {
		var idByte = byte(blockId & 0xff)
		var extraValue, extraPresent = extraData[idByte]
		if extraValue && extraPresent {
			fmt.Fprintf(w, "usemtl %d_%d", idByte, blockId>>8)
		} else {
			fmt.Fprintln(w, "usemtl", idByte)
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

func init() {
	extraData = make(map[byte]bool)
	for _, mtl := range colors {
		if mtl.metadata != 255 {
			extraData[mtl.blockId] = true
		}
	}
}

var (
	extraData map[byte]bool

	colors = []MTL{
		MTL{0, 255, 0xfefeff01, "Air"},
		MTL{1, 255, 0x7d7d7dff, "Stone"},
		MTL{2, 255, 0x52732cff, "Grass"},
		MTL{3, 255, 0x866043ff, "Dirt"},
		MTL{4, 255, 0x757575ff, "Cobblestone"},
		MTL{5, 255, 0x9d804fff, "Wooden Plank"},
		MTL{6, 255, 0x5d7e1eff, "Sapling"},
		MTL{7, 255, 0x545454ff, "Bedrock"},
		MTL{8, 255, 0x009aff50, "Water"},
		MTL{9, 255, 0x009aff50, "Stationary water"},
		MTL{10, 255, 0xf54200ff, "Lava"},
		MTL{11, 255, 0xf54200ff, "Stationary lava"},
		MTL{12, 255, 0xdad29eff, "Sand"},
		MTL{13, 255, 0x887f7eff, "Gravel"},
		MTL{14, 255, 0x908c7dff, "Gold ore"},
		MTL{15, 255, 0x88837fff, "Iron ore"},
		MTL{16, 255, 0x737373ff, "Coal ore"},
		MTL{17, 255, 0x665132ff, "Wood"},
		MTL{18, 255, 0x1c4705ff, "Leaves"},
		MTL{19, 255, 0xb7b739ff, "Sponge"},
		MTL{20, 255, 0xffffff01, "Glass"},
		MTL{21, 255, 0x667087ff, "Lapis Lazuli Ore"},
		MTL{22, 255, 0x1d47a6ff, "Lapis Lazuli Block"},
		MTL{23, 255, 0x6c6c6cff, "Dispenser"},
		MTL{24, 255, 0xd5cd94ff, "Sandstone"},
		MTL{25, 255, 0x654433ff, "Note Block"},
		MTL{26, 0, 0x8f1717ff, "Foot of bed pointing West"},
		MTL{26, 1, 0x8f1717ff, "Foot of bed pointing North"},
		MTL{26, 2, 0x8f1717ff, "Foot of bed pointing East"},
		MTL{26, 3, 0x8f1717ff, "Foot of bed pointing South"},
		MTL{26, 8, 0xaf7475ff, "Head of bed pointing West"},
		MTL{26, 9, 0xaf7475ff, "Head of bed pointing North"},
		MTL{26, 10, 0xaf7475ff, "Head of bed pointing East"},
		MTL{26, 11, 0xaf7475ff, "Head of bed pointing South"},
		MTL{35, 0, 0xdededeff, "Wool - White"},
		MTL{35, 1, 0xea8037ff, "Wool - Orange"},
		MTL{35, 2, 0xbf4cc9ff, "Wool - Magenta"},
		MTL{35, 3, 0x688bd4ff, "Wool - Light Blue"},
		MTL{35, 4, 0xc2b51cff, "Wool - Yellow"},
		MTL{35, 5, 0x3bbd30ff, "Wool - Light Green"},
		MTL{35, 6, 0xd9849bff, "Wool - Pink"},
		MTL{35, 7, 0x434343ff, "Wool - Gray"},
		MTL{35, 8, 0x9ea6a6ff, "Wool - Light Gray"},
		MTL{35, 9, 0x277596ff, "Wool - Cyan"},
		MTL{35, 10, 0x8136c4ff, "Wool - Purple"},
		MTL{35, 11, 0x27339aff, "Wool - Blue"},
		MTL{35, 12, 0x56331cff, "Wool - Brown"},
		MTL{35, 13, 0x384d18ff, "Wool - Dark Green"},
		MTL{35, 14, 0xa42d29ff, "Wool - Red"},
		MTL{35, 15, 0x1b1717ff, "Wool - Black"},
		MTL{37, 255, 0xc1c702ff, "Yellow flower"},
		MTL{38, 255, 0xcb060aff, "Red rose"},
		MTL{39, 255, 0x967158ff, "Brown Mushroom"},
		MTL{40, 255, 0xc53c3fff, "Red Mushroom"},
		MTL{41, 255, 0xfaec4eff, "Gold Block"},
		MTL{42, 255, 0xe6e6e6ff, "Iron Block"},
		MTL{43, 255, 0xa7a7a7ff, "Double Stone Slab"},
		MTL{44, 255, 0xa7a7a7ff, "Stone Slab"},
		MTL{45, 255, 0x9c6e62ff, "Brick"},
		MTL{46, 255, 0xa6553fff, "TNT"},
		MTL{47, 255, 0x6c583aff, "Bookshelf"},
		MTL{48, 255, 0x5b6c5bff, "Moss Stone"},
		MTL{49, 255, 0x14121eff, "Obsidian"},
		MTL{50, 255, 0xffda6699, "Torch"},
		MTL{51, 255, 0xff770099, "Fire"},
		MTL{52, 255, 0x1d4f72ff, "Monster Spawner"},
		MTL{53, 255, 0x9d804fff, "Wooden Stairs"},
		MTL{54, 255, 0x835e25ff, "Chest"},
		MTL{55, 255, 0xcb0000ff, "Redstone Wire"},
		MTL{56, 255, 0x828c8fff, "Diamond Ore"},
		MTL{57, 255, 0x64dcd6ff, "Diamond Block"},
		MTL{58, 255, 0x6b472bff, "Workbench"},
		MTL{59, 255, 0x83c144ff, "Crops"},
		MTL{60, 255, 0x4b290eff, "Soil"},
		MTL{61, 255, 0x4e4e4eff, "Furnace"},
		MTL{62, 255, 0x7d6655ff, "Burning Furnace"},
		MTL{63, 255, 0x9d804fff, "Sign Post"},
		MTL{64, 255, 0x9d804fff, "Wooden Door"},
		MTL{65, 255, 0x9d804fff, "Ladder"},
		MTL{66, 255, 0x75664cff, "Minecart Tracks"},
		MTL{67, 255, 0x757575ff, "Cobblestone Stairs"},
		MTL{68, 255, 0x9d804fff, "Wall Sign"},
		MTL{69, 255, 0x9d804fff, "Lever"},
		MTL{70, 255, 0x7d7d7dff, "Stone Pressure Plate"},
		MTL{71, 255, 0xb2b2b2ff, "Iron Door"},
		MTL{72, 255, 0x9d804fff, "Wooden Pressure Plate"},
		MTL{73, 255, 0x856b6bff, "Redstone Ore"},
		MTL{74, 255, 0xbd6b6bff, "Glowing Redstone Ore"},
		MTL{75, 255, 0x44000099, "Redstone torch (\"off\" state)"},
		MTL{76, 255, 0xfe000099, "Redstone torch (\"on\" state)"},
		MTL{77, 255, 0x7d7d7dff, "Stone Button"},
		MTL{78, 255, 0xf0fbfbff, "Snow"},
		MTL{79, 255, 0x7daeff77, "Ice"},
		MTL{80, 255, 0xf0fbfbff, "Snow Block"},
		MTL{81, 255, 0x0d6418ff, "Cactus"},
		MTL{82, 255, 0x9fa5b1ff, "Clay"},
		MTL{83, 255, 0x83c447ff, "Sugar Cane"},
		MTL{84, 255, 0x6b4937ff, "Jukebox"},
		MTL{85, 255, 0x9d804fff, "Fence"},
		MTL{86, 255, 0xc57918ff, "Pumpkin"},
		MTL{87, 255, 0x6e3533ff, "Netherrack"},
		MTL{88, 255, 0x554134ff, "Soul Sand"},
		MTL{89, 255, 0x897141ff, "Glowstone"},
		MTL{90, 255, 0x381d55bb, "Portal"},
		MTL{91, 255, 0xb9861dff, "Jack-O-Lantern"},
		MTL{92, 255, 0xe5cecfff, "Cake Block"},
		MTL{93, 255, 0x989494ff, "Redstone Repeater (\"off\" state)"},
		MTL{94, 255, 0xa19494ff, "Redstone Repeater (\"on\" state)"},
	}
)
