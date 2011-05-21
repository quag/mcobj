package main

import (
	"os"
	"io"
	"image"
	"image/png"
	"fmt"
)

type ImageNG interface {
	image.Image
	Set(x, y int, c image.Color)
}

func imgType(im *image.Image, w, h int) ImageNG {
	var scaled ImageNG
	switch (*im).ColorModel() {
	case image.RGBAColorModel:
		scaled = image.NewRGBA(w, h)
	case image.NRGBAColorModel:
		scaled = image.NewNRGBA(w, h)
	case image.GrayColorModel:
		scaled = image.NewGray(w, h)
	case image.Gray16ColorModel:
		scaled = image.NewGray16(w, h)
	default:
		scaled = image.NewRGBA64(w, h)
	}
	return scaled
}
func convertColor(im image.Image, col image.RGBA64Color) image.Color {
	var outCol image.Color
	switch im.ColorModel() {
	case image.RGBAColorModel:
		outCol = image.RGBAColorModel.Convert(col).(image.RGBAColor)
	case image.NRGBAColorModel:
		outCol = image.NRGBAColorModel.Convert(col).(image.NRGBAColor)
	case image.GrayColorModel:
		outCol = image.GrayColorModel.Convert(col).(image.GrayColor)
	case image.Gray16ColorModel:
		outCol = image.Gray16ColorModel.Convert(col).(image.Gray16Color)
	default:
		outCol = col
	}
	return outCol
}
func copyBlock(output ImageNG, input image.Image, dest image.Rectangle, source image.Rectangle) {
	source = source.Canon()
	dest = dest.Canon()
	width := dest.Dx()
	height := dest.Dy()
	for j := 0; j < height; j++ {
		for i := 0; i < width; i++ {
			output.Set(dest.Min.X+i, dest.Min.Y+j, input.At(source.Min.X+i, source.Min.Y+j))
		}
	}
}
func makeTexture(input *image.Image, outputTemplate string, count int) (outputName string, retimg ImageNG) {
	outputName = fmt.Sprintf("%s%d.png", outputTemplate, count)
	retimg = imgType(input, numRepeatingPatternsAcross*((*input).Bounds().Dx()/numBlockPatternsAcross), ((*input).Bounds().Dy() / numBlockPatternsAcross))
	return
}
func writeTexture(outputName string, retimg ImageNG) os.Error {
	outputFile, openErr := os.Create(outputName)
	if openErr != nil {
		return openErr
	}
	png.Encode(outputFile, retimg)
	outputFile.Close()
	return nil
}
func extractTerrainImage(reader io.Reader, outputTemplate string, blockTypeMap map[byte]*BlockType) (retval os.Error) {
	var img, fileErr = png.Decode(reader)
	if fileErr != nil {
		return fileErr
	}
	currentTextureCount := 0
	currentTextureOffset := 0
	currentTextureName, currentTexture := makeTexture(&img, outputTemplate, currentTextureCount)
	var repeatingTextureNames map[string]string = make(map[string]string)
	var repeatingTextureOffsets map[string]int = make(map[string]int)
	for blockName, blockValue := range blockTypeMap {
		for colorName, colorValue := range blockValue.colors {
			tc := colorValue.frontTex
			for side := 0; side < 2; side++ {
				tckey := fmt.Sprint(tc)
				repeatingTextureName, ok := repeatingTextureNames[tckey]
				var repeatingTextureOffset int
				if ok {
					repeatingTextureOffset, ok = repeatingTextureOffsets[tckey]
				} else {
					if (side == 1 && currentTextureOffset < numRepeatingPatternsAcross) || currentTextureOffset+1 < numRepeatingPatternsAcross {
						repeatingTextureOffset = currentTextureOffset
						repeatingTextureName = currentTextureName
						currentTextureOffset++
					} else {
						currentTextureCount++
						currentTextureOffset = 0
						retval = writeTexture(currentTextureName, currentTexture)
						currentTextureName, currentTexture = makeTexture(&img, outputTemplate, currentTextureCount)
						repeatingTextureName = currentTextureName
						repeatingTextureOffset = currentTextureOffset
					}
					repeatingTextureNames[tckey] = repeatingTextureName
					repeatingTextureOffsets[tckey] = repeatingTextureOffset
					blockSizeX := img.Bounds().Dx() / numBlockPatternsAcross
					blockSizeY := img.Bounds().Dy() / numBlockPatternsAcross

					copyBlock(currentTexture,
						img,
						image.Rect(currentTextureOffset*blockSizeX, 0,
							(currentTextureOffset+1)*blockSizeX, blockSizeY),
						image.Rect(int(tc.topLeft.x)*blockSizeX,
							int(tc.topLeft.y)*blockSizeY, int(tc.bottomRight.x)*blockSizeX, int(tc.bottomRight.y)*blockSizeY))
				}
				blockTypeMap[blockName].colors[colorName].repeatingTextureName = repeatingTextureName
				if side == 0 {
					blockTypeMap[blockName].colors[colorName].repeatingFrontOffset = repeatingTextureOffset
				} else {
					blockTypeMap[blockName].colors[colorName].repeatingSideOffset = repeatingTextureOffset
				}
				tc = colorValue.sideTex

			}
		}
	}
	retval = writeTexture(currentTextureName, currentTexture)
	return
}
