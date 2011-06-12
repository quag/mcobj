package main

import (
	"bufio"
	"io"
	"regexp"
	"strconv"
	"fmt"
	"strings"
	"os"
)

type ObjVec2 struct {
	x float32
	y float32
}
type ObjVec3 struct {
	x float32
	y float32
	z float32
}
type ObjReference struct {
	vertex         int
	texcoord       int //-1 for missing
	normal         int // -1 for missing
	materialNumber int
}
type Material struct {
	texture string
}
type ObjFile struct {
	texcoords []ObjVec2
	vertices  []ObjVec3
	normals   []ObjVec3
	faces     [][]ObjReference
	materials []Material
}

var floatingPointNumber string = "([\\-+]?[0-9]*\\.?[0-9]+([eE][\\-+]?[0-9]+)?)"

func parseMaterialLibrary(mtllib string, materialMap map[string]int, maxMaterial int) (retval []Material) {
	if maxMaterial == 0 {
		maxMaterial++
	}
	materialRE := regexp.MustCompile("^newmtl[ \t]+([^#]*)")
	textureRE := regexp.MustCompile("^map_([^# \t]*)[ \t]+([^#]*)")
	specRE := regexp.MustCompile("^Ns[ \t](" + floatingPointNumber + ")")
	colorRE := regexp.MustCompile("^(K[a-z])[ \t]+(" + floatingPointNumber + "[ \t]+" + floatingPointNumber + "[ \t]+" + floatingPointNumber + ")")
	retval = make([]Material, maxMaterial)
	for materialName, index := range materialMap {
		if strings.Index(materialName, "articles") != -1 {
			retval[index].texture = "particles.png"
		} else if strings.Index(materialName, "Lava") != -1 || strings.Index(materialName, "lava") != -1 {
			if strings.Index(materialName, "lowing") != -1 {
				retval[index].texture = "custom_lava_flowing.png"
			} else {
				retval[index].texture = "custom_lava_still.png"
			}
		} else if strings.Index(materialName, "Water") != -1 || strings.Index(materialName, "water") != -1 {
			if strings.Index(materialName, "lowing") != -1 {
				retval[index].texture = "custom_water_flowing.png"
			} else {
				retval[index].texture = "custom_water_still.png"
			}
		} else if strings.Index(materialName, "ortal") != -1 {
			retval[index].texture = "custom_portal.png"
		} else if strings.Index(materialName, "lternate") != -1 || strings.Index(materialName, "LTERNATE") != -1 {
			retval[index].texture = "ALTERNATES.png"
		} else {
			retval[index].texture = "terrain.png"
		}
	}
	var file, fileErr = os.Open(mtllib)
	reader, _ := bufio.NewReaderSize(file, 16384)
	if fileErr == nil {
		var dummyMaterial Material
		curMaterial := &dummyMaterial
		for linenum := 0; true; linenum++ {
			line, _, err := reader.ReadLine()
			if err != nil { //EOF?
				break
			}
			material := materialRE.FindSubmatch(line)
			texture := textureRE.FindSubmatch(line)
			spec := specRE.FindSubmatch(line)
			color := colorRE.FindSubmatch(line)
			if material != nil {
				index, ok := materialMap[string(material[1])]
				if ok {
					curMaterial = &retval[index]
				} else {
					curMaterial = &dummyMaterial
				}
			}
			if texture != nil {
				curMaterial.texture = string(texture[2]) //FIXME need to look for Kd
			}
			if spec != nil {

			}
			if color != nil {

			}
		}
	} else {
		fmt.Println("Warning: unable to open material file ", mtllib, " using terrain.png as default texture")
	}
	return
}
func parseObj(f io.Reader) (retval ObjFile) {
	curMaterial := 0
	maxMaterial := 0
	materialMap := make(map[string]int)
	//integerNumber:="-{0,1}[0-9]+";
	polygonVertex := "(\\-*[0-9]+)(/\\-*[0-9]*)?(/\\-*[0-9]*)?"
	mtllibRE := regexp.MustCompile("^mtllib[ \t]+([^#]*)")
	usemtlRE := regexp.MustCompile("^usemtl[ \t]+([^#]*)")
	vertexRE := regexp.MustCompile("^v[ \t]+" + floatingPointNumber + "[ \t]+" + floatingPointNumber + "[ \t]+" + floatingPointNumber)
	normalRE := regexp.MustCompile("^vn[ \t]+" + floatingPointNumber + "[ \t]+" + floatingPointNumber + "[ \t]+" + floatingPointNumber)
	texCoordRE := regexp.MustCompile("^vt[ \t]+" + floatingPointNumber + "[ \t]+" + floatingPointNumber)
	polygonRE := regexp.MustCompile("^f[ \t]+(" + polygonVertex + "([ \t]+" + polygonVertex + ")*" + ")")
	reader, _ := bufio.NewReaderSize(f, 16384)
	var materialFileName string
	for linenum := 0; true; linenum++ {
		line, _, err := reader.ReadLine()
		if err != nil { //EOF?
			break
		}
		vertex := vertexRE.FindSubmatch(line)
		normal := normalRE.FindSubmatch(line)
		texCoord := texCoordRE.FindSubmatch(line)
		polygon := polygonRE.FindSubmatch(line)
		usemtl := usemtlRE.FindSubmatch(line)
		mtllib := mtllibRE.FindSubmatch(line)
		if mtllib != nil {
			materialFileName = string(mtllib[1])

		}
		if usemtl != nil {
			desiredMaterialName := string(usemtl[1])
			desiredMaterialNumber, ok := materialMap[desiredMaterialName]
			if !ok {
				desiredMaterialNumber = maxMaterial
				maxMaterial++
				materialMap[desiredMaterialName] = desiredMaterialNumber
			}
			curMaterial = desiredMaterialNumber
		}
		if vertex != nil {
			x, err0 := strconv.Atof32(string(vertex[1]))
			y, err1 := strconv.Atof32(string(vertex[3]))
			z, err2 := strconv.Atof32(string(vertex[5]))
			if err0 != nil || err1 != nil || err2 != nil {
				fmt.Println("Error parsing obj vertex file, line: ", linenum, err0, err1, err2)
			}
			retval.vertices = append(retval.vertices, ObjVec3{x, y, z})
		}
		if normal != nil {
			x, err0 := strconv.Atof32(string(normal[1]))
			y, err1 := strconv.Atof32(string(normal[3]))
			z, err2 := strconv.Atof32(string(normal[5]))
			if err0 != nil || err1 != nil || err2 != nil {
				fmt.Println("Error parsing obj normal file, line: ", linenum, err0, err1, err2)
			}
			retval.normals = append(retval.normals, ObjVec3{x, y, z})
		}
		if texCoord != nil {
			u, err0 := strconv.Atof32(string(texCoord[1]))
			v, err1 := strconv.Atof32(string(texCoord[3]))
			if err0 != nil || err1 != nil {
				fmt.Println("Error parsing obj texcoord file, line: ", linenum, err0, err1)
			}
			retval.texcoords = append(retval.texcoords, ObjVec2{u, v})
		}
		if polygon != nil {
			faces := strings.Split(strings.Replace(string(polygon[1]), "\t", " ", -1), " ", -1)
			face := make([]ObjReference, 0)
			for _, facestr := range faces {
				references := strings.Split(facestr, "/", 3)
				if len(references) != 0 {
					v, errv := strconv.Atoi(references[0])
					if errv == nil {
						curFaceVertex := ObjReference{-1, -1, -1, curMaterial}
						if v < 0 {
							curFaceVertex.vertex = len(retval.vertices) + v
						} else {
							curFaceVertex.vertex = v
						}
						if len(references) > 1 {
							t, errt := strconv.Atoi(references[1])
							if errt == nil {
								if t < 0 {
									curFaceVertex.texcoord = len(retval.texcoords) + t
								} else {
									curFaceVertex.texcoord = t
								}
							}
						}
						if len(references) > 2 {
							n, errn := strconv.Atoi(references[2])
							if errn == nil {
								if n < 0 {
									curFaceVertex.normal = len(retval.normals) + n
								} else {
									curFaceVertex.normal = n
								}
							}
						}
						face = append(face, curFaceVertex)
					}
				}
			}
			if len(face) > 2 {
				retval.faces = append(retval.faces, face)
			}
		}
	}
	retval.materials = parseMaterialLibrary(materialFileName, materialMap, maxMaterial)
	return
}
