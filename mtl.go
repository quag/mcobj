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
			fmt.Fprintf(w, "usemtl %d_%d\n", idByte, blockId>>8)
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
	colors = make([]MTL, 256)
	for i, _ := range colors {
		colors[i] = MTL{byte(i), 255, 0x7f7f7f, "Unknown"}
	}

	extraData = make(map[byte]bool)
}

var (
	extraData map[byte]bool

	colors []MTL
)
