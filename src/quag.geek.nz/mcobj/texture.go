package main

import (
	"os"
	"io"
	"image/png"
	"fmt"
)

func extractTerrainImage(reader io.Reader, outputTemplate string, blockTypeMap map[byte]*BlockType) os.Error {
	var _, fileErr = png.Decode(reader)
	if fileErr != nil {
		return fileErr
	}
	currentTextureCount := 0
	currentTextureOffset := 0
	currentTextureName := fmt.Sprintf("%s%d.png", outputTemplate, currentTextureCount)
	var repeatingTextureNames map[string]string = make(map[string]string)
	var repeatingTextureOffsets map[string]int = make(map[string]int)
	for _, blockValue := range blockTypeMap {
		for _, colorValue := range blockValue.colors {
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
						currentTextureOffset = 01
						currentTextureName = fmt.Sprintf("%s%d.png", outputTemplate, currentTextureCount)
						repeatingTextureName = currentTextureName
						repeatingTextureOffset = currentTextureOffset
					}
					repeatingTextureNames[tckey] = repeatingTextureName
					repeatingTextureOffsets[tckey] = repeatingTextureOffset
				}
				colorValue.repeatingTextureName = repeatingTextureName
				if side == 0 {
					colorValue.repeatingFrontOffset = repeatingTextureOffset
				} else {
					colorValue.repeatingSideOffset = repeatingTextureOffset
				}
				tc = colorValue.sideTex

			}
		}
	}
	return nil
}
