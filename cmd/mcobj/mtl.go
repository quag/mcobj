package main

import (
	"fmt"
	"io"
	"os"
)

func printMtl(w io.Writer, blockId uint16) {
	if !noColor {
		fmt.Fprintln(w, "usemtl", MaterialNamer.NameBlockId(blockId))
	}
}

func writeMtlFile(filename string) error {
	if noColor {
		return nil
	}

	var outFile, outErr = os.Create(filename)
	if outErr != nil {
		return outErr
	}
	defer outFile.Close()

	for _, color := range colors {
		color.Print(outFile)
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

	fmt.Fprintf(w, "# %s\nnewmtl %s\nKd %.4f %.4f %.4f\nd %.4f\nillum 1\n\n", mtl.name, MaterialNamer.NameBlockId(uint16(mtl.blockId)+uint16(mtl.metadata)*256), float64(r)/255, float64(g)/255, float64(b)/255, float64(a)/255)
}

func (mtl *MTL) colorId() uint16 {
	var id = uint16(mtl.blockId)
	if mtl.metadata != 255 {
		id += uint16(mtl.metadata) << 8
	}
	return id
}

func init() {
	colors = make([]MTL, 256)
	for i, _ := range colors {
		colors[i] = MTL{byte(i), 255, 0x800000ff, fmt.Sprintf("Unknown.%d", i)}
	}

	extraData = make(map[byte]bool)
}

var (
	extraData map[byte]bool

	colors []MTL

	MaterialNamer BlockIdNamer
)

type BlockIdNamer interface {
	NameBlockId(blockId uint16) string
}

type NumberBlockIdNamer struct{}

func (n *NumberBlockIdNamer) NameBlockId(blockId uint16) (name string) {
	var idByte = byte(blockId & 0xff)
	var extraValue, extraPresent = extraData[idByte]
	if extraValue && extraPresent {
		name = fmt.Sprintf("%d_%d", idByte, blockId>>8)
	} else {
		name = fmt.Sprintf("%d", idByte)
	}
	return
}

type NameBlockIdNamer struct{}

func (n *NameBlockIdNamer) NameBlockId(blockId uint16) (name string) {
	var idByte = byte(blockId & 0xff)
	var extraValue, extraPresent = extraData[idByte]
	if extraValue && extraPresent {
		for _, color := range colors {
			if color.blockId == idByte && color.metadata == uint8(blockId>>8) {
				return color.name
			}
		}
	} else {
		for _, color := range colors {
			if color.blockId == idByte {
				return color.name
			}
		}
	}
	return
}
