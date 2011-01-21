package main

import (
	"bufio"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
)

const (
	tagStructEnd = 0  // No name. Single zero byte.
	tagInt8      = 1  // A single signed byte (8 bits)
	tagInt16     = 2  // A signed short (16 bits, big endian)
	tagInt32     = 3  // A signed int (32 bits, big endian)
	tagInt64     = 4  // A signed long (64 bits, big endian)
	tagFloat32   = 5  // A floating point value (32 bits, big endian, IEEE 754-2008, binary32)
	tagFloat64   = 6  // A floating point value (64 bits, big endian, IEEE 754-2008, binary64)
	tagByteArray = 7  // { TAG_Int length; An array of bytes of unspecified format. The length of this array is <length> bytes }
	tagString    = 8  // { TAG_Short length; An array of bytes defining a string in UTF-8 format. The length of this array is <length> bytes }
	tagList      = 9  // { TAG_Byte tagId; TAG_Int length; A sequential list of Tags (not Named Tags), of type <typeId>. The length of this array is <length> Tags. } Notes: All tags share the same type.
	tagStruct    = 10 // { A sequential list of Named Tags. This array keeps going until a TAG_End is found.; TAG_End end } Notes: If there's a nested TAG_Compound within this tag, that one will also have a TAG_End, so simply reading until the next TAG_End will not work. The names of the named tags have to be unique within each TAG_Compound The order of the tags is not guaranteed.
)

const (
	trace = false
)

var (
	out         *bufio.Writer
	faces       *Faces
	yMin        int
	blockFaces  bool
	showTunnels bool
	hideStone   bool
	noColor     bool
)

func main() {
	var filename string
	flag.IntVar(&yMin, "y", 0, "Omit all blocks below this height. 63 is sea level")
	flag.BoolVar(&blockFaces, "bf", false, "Don't combine adjacent faces of the same block within a column")
	flag.BoolVar(&showTunnels, "t", false, "Show tunnels")
	flag.BoolVar(&hideStone, "hs", false, "Hide stone")
	flag.BoolVar(&noColor, "g", false, "Omit materials")
	flag.StringVar(&filename, "o", "a.obj", "Name for output file")
	flag.Parse()

	if flag.NArg() != 0 {
		var mtlFilename = fmt.Sprintf("%s.mtl", filename[:len(filename)-len(path.Ext(filename))])
		var mtlErr = writeMtlFile(mtlFilename)
		if mtlErr != nil {
			fmt.Fprintln(os.Stderr, mtlErr)
			return
		}

		var outFile, outErr = os.Open(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
		if outErr != nil {
			fmt.Fprintln(os.Stderr, outErr)
			return
		}
		defer outFile.Close()
		var bufErr os.Error
		out, bufErr = bufio.NewWriterSize(outFile, 1024*1024)
		if bufErr != nil {
			fmt.Fprintln(os.Stderr, bufErr)
			return
		}
		defer out.Flush()

		faces = NewFaces()

		fmt.Fprintln(out, "mtllib", path.Base(mtlFilename))

		for i := 0; i < flag.NArg(); i++ {
			var filepath = flag.Arg(i)
			var fi, err = os.Stat(filepath)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
			}

			switch {
			case fi.IsRegular():
				processChunk(filepath)
			case fi.IsDirectory():
				var errors = make(chan os.Error, 5)
				var done = make(chan bool)
				go func() {
					for error := range errors {
						fmt.Fprintln(os.Stderr, error)
					}
					done <- true
				}()
				path.Walk(filepath, &visitor{}, errors)
				close(errors)
				<-done
			}
		}
	}
}

type visitor struct{}

func (v *visitor) VisitDir(dir string, f *os.FileInfo) bool {
	return true
}

func (v *visitor) VisitFile(file string, f *os.FileInfo) {
	var match, err = path.Match("c.*.dat", path.Base(file))
	if match && err == nil {
		processChunk(file)
	}
}

func processChunk(filename string) bool {
	fmt.Fprintln(out, "#", filename)
	fmt.Fprintln(os.Stderr, filename)
	var file, fileErr = os.Open(filename, os.O_RDONLY, 0666)
	if fileErr != nil {
		fmt.Println(fileErr)
		return true
	}

	var r, rErr = gzip.NewReader(file)
	if rErr != nil {
		fmt.Println(rErr)
		return true
	}

	var xPos, zPos int

	var abort = false

	var br = bufio.NewReader(r)

	for {
		var typeId, name, err = readTag(br)
		if err != nil {
			break
		}

		switch typeId {
		case tagStruct:
			if trace {
				fmt.Printf("%s struct start\n", name)
			}
		case tagStructEnd:
			if trace {
				fmt.Printf("struct end\n")
			}
		case tagByteArray:
			var bytes, err2 = readBytes(br)
			if err2 != nil {
				fmt.Println(err2)
				return true
			}
			if trace {
				fmt.Printf("%s bytes(%d) %v\n", name, len(bytes), bytes)
			}
			if name == "Blocks" {
				faces.Clean(xPos, zPos)
				processBlocks(bytes, faces)
				faces.Process()
			}
		case tagInt8:
			var number, err2 = readInt8(br)
			if err2 != nil {
				fmt.Println(err2)
				return true
			}
			if trace {
				fmt.Printf("%s int8 %v\n", name, number)
			}
		case tagInt16:
			var number, err2 = readInt16(br)
			if err2 != nil {
				fmt.Println(err2)
				return true
			}
			if trace {
				fmt.Printf("%s int16 %v\n", name, number)
			}
		case tagInt32:
			var number, err2 = readInt32(br)
			if err2 != nil {
				fmt.Println(err2)
				return true
			}
			if trace {
				fmt.Printf("%s int32 %v\n", name, number)
			}

			if name == "xPos" {
				xPos = number
			}
			if name == "zPos" {
				zPos = number
			}
		case tagInt64:
			var number, err2 = readInt64(br)
			if err2 != nil {
				fmt.Println(err2)
				return true
			}
			if trace {
				fmt.Printf("%s int64 %v\n", name, number)
			}
		case tagFloat32:
			var number, err2 = readInt32(br) // TODO(jw): read floats not ints
			if err2 != nil {
				fmt.Println(err2)
				return true
			}
			if trace {
				fmt.Printf("%s int32 %v\n", name, number)
			}
		case tagFloat64:
			var number, err2 = readInt64(br) // TODO(jw): read floats not ints
			if err2 != nil {
				fmt.Println(err2)
				return true
			}
			if trace {
				fmt.Printf("%s int64 %v\n", name, number)
			}
		case tagString:
			var s, err2 = readString(br)
			if err2 != nil {
				fmt.Println(err2)
				return true
			}
			if trace {
				fmt.Printf("%s string \"%s\"", name, s)
			}
		case tagList:
			var itemTypeId, length, err2 = readListHeader(br)
			if err2 != nil {
				fmt.Println(err2)
				return true
			}
			switch itemTypeId {
			case tagInt8:
				if trace {
					fmt.Printf("%s list int8 (%d)\n", name, length)
				}
				for i := 0; i < length; i++ {
					var v, err3 = readInt8(br)
					if err3 != nil {
						fmt.Println(err3)
						return true
					}
					if trace {
						fmt.Println("  ", v)
					}
				}
			case tagFloat32:
				if trace {
					fmt.Printf("%s list float64 (%d)\n", name, length)
				}
				for i := 0; i < length; i++ {
					var v, err3 = readInt32(br) // TODO(jw) read float32 instead of int32
					if err3 != nil {
						fmt.Println(err3)
						return true
					}
					if trace {
						fmt.Println("  ", v)
					}
				}
			case tagFloat64:
				if trace {
					fmt.Printf("%s list float64 (%d)\n", name, length)
				}
				for i := 0; i < length; i++ {
					var v, err3 = readInt64(br) // TODO(jw) read float64 instead of int64
					if err3 != nil {
						fmt.Println(err3)
						return true
					}
					if trace {
						fmt.Println("  ", v)
					}
				}
			case tagStruct:
				if trace {
					fmt.Printf("%s list struct (%d)\n", name, length)
				}
				abort = true
			default:
				fmt.Printf("# %s list todo(%v) %v\n", name, itemTypeId, length)
			}
		default:
			fmt.Printf("# %s todo(%d)\n", name, typeId)
		}
	}
	fmt.Fprintln(out)

	out.Flush()

	return abort
}

func readTag(r *bufio.Reader) (byte, string, os.Error) {
	var typeId, err = r.ReadByte()
	if err != nil || typeId == 0 {
		return typeId, "", err
	}

	var name, err2 = readString(r)
	if err2 != nil {
		return typeId, name, err2
	}

	return typeId, name, nil
}

func readListHeader(r *bufio.Reader) (itemTypeId byte, length int, err os.Error) {
	length = 0

	itemTypeId, err = r.ReadByte()
	if err == nil {
		length, err = readInt32(r)
	}

	return
}

func readString(r *bufio.Reader) (string, os.Error) {
	var length, err1 = readInt16(r)
	if err1 != nil {
		return "", err1
	}

	var bytes = make([]byte, length)
	var _, err2 = io.ReadFull(r, bytes)
	return string(bytes), err2
}

func readBytes(r *bufio.Reader) ([]byte, os.Error) {
	var length, err1 = readInt32(r)
	if err1 != nil {
		return nil, err1
	}

	var bytes = make([]byte, length)
	var _, err2 = io.ReadFull(r, bytes)
	return bytes, err2
}

func readInt8(r *bufio.Reader) (int, os.Error) {
	return readIntN(r, 1)
}

func readInt16(r *bufio.Reader) (int, os.Error) {
	return readIntN(r, 2)
}

func readInt32(r *bufio.Reader) (int, os.Error) {
	return readIntN(r, 4)
}

func readInt64(r *bufio.Reader) (int, os.Error) {
	return readIntN(r, 8)
}

func readIntN(r *bufio.Reader, n int) (int, os.Error) {
	var a int = 0

	for i := 0; i < n; i++ {
		var b, err = r.ReadByte()
		if err != nil {
			return a, err
		}
		a = a<<8 + int(b)
	}

	return a, nil
}

type Blocks []byte

func (b *Blocks) Get(x, y, z int) byte {
	return (*b)[y+(z*128+(x*128*16))]
}

func (b *Blocks) IsEmpty(x, y, z int) bool {
	if x < 0 || y < 0 || z < 0 || x >= 16 || y >= 128 || z >= 16 {
		return true
	}
	return isEmptyBlockId(b.Get(x, y, z))
}

func (b *Blocks) IsBoundary(x, y, z int, blockId byte) bool {
	if x < 0 || y < 0 || z < 0 || x >= 16 || y >= 128 || z >= 16 {
		if showTunnels {
			return blockId == 18 /* leaves */ || blockId == 17 /* wood */
		} else {
			return blockId != 0 && blockId != 9 && (!hideStone || blockId != 1)
		}
	}

	var otherBlockId = b.Get(x, y, z)

	if (blockId == 9 || (hideStone && blockId == 1)) && otherBlockId == 0 {
		return true
	}

	return (blockId != 0 && blockId != 9 && (!hideStone || blockId != 1)) && (otherBlockId == 0 || otherBlockId == 9 || (hideStone && otherBlockId == 1))
}

func isEmptyBlockId(blockId byte) bool {
	return blockId == 0 || blockId == 1
}

type AddFacer interface {
	AddFace(blockId byte, v1, v2, v3, v4 Vertex)
}

type Faces struct {
	xPos, zPos int
	count      int

	vertexes Vertexes
	faces    []Face
}

type Face struct {
	blockId byte
	indexes [4]int
}

func NewFaces() *Faces {
	return &Faces{0, 0, 0, make([]int16, (128+1)*(16+1)*(16+1)), make([]Face, 0, 8192)}
}

func (fs *Faces) Clean(xPos, zPos int) {
	fs.xPos = xPos
	fs.zPos = zPos
	fs.vertexes.Clear()
	fs.faces = fs.faces[0:0]
}

func (fs *Faces) AddFace(blockId byte, v1, v2, v3, v4 Vertex) {
	var face = Face{blockId, [4]int{fs.vertexes.Use(v1), fs.vertexes.Use(v2), fs.vertexes.Use(v3), fs.vertexes.Use(v4)}}
	fs.faces = append(fs.faces, face)
}

func (fs *Faces) Process() {
	fs.vertexes.Number()
	var vc = int16(fs.vertexes.Print(out, fs.xPos, fs.zPos))

	var blockIds = make([]byte, 0, 16)
	for _, face := range fs.faces {
		var found = false
		for _, id := range blockIds {
			if id == face.blockId {
				found = true
				break
			}
		}

		if !found {
			blockIds = append(blockIds, face.blockId)
		}
	}

	for _, blockId := range blockIds {
		printMtl(out, blockId)
		for _, face := range fs.faces {
			if face.blockId == blockId {
				fmt.Fprintln(out, "f", fs.vertexes.Get(face.indexes[0])-vc-1, fs.vertexes.Get(face.indexes[1])-vc-1, fs.vertexes.Get(face.indexes[2])-vc-1, fs.vertexes.Get(face.indexes[3])-vc-1)
			}
		}
	}

	fmt.Fprintln(os.Stderr, "Faces:", len(fs.faces))
}

type Vertexes []int16

func (vs *Vertexes) Index(x, y, z int) int {
	return y + (z*129 + (x * 129 * 17))
}

func (vs *Vertexes) Use(v Vertex) int {
	var i = vs.Index(v.x, v.y, v.z)
	(*vs)[i]++
	return i
}

func (vs *Vertexes) Release(v Vertex) int {
	var i = vs.Index(v.x, v.y, v.z)
	(*vs)[i]--
	return i
}

func (vs *Vertexes) Get(i int) int16 {
	return (*vs)[i]
}

func (vs *Vertexes) Clear() {
	for i, _ := range *vs {
		(*vs)[i] = 0
	}
}

func (vs *Vertexes) Number() {
	var count int16 = 0
	for i, references := range *vs {
		if references != 0 {
			count++
			(*vs)[i] = count
		} else {
			(*vs)[i] = -1
		}
	}
}

func (vs *Vertexes) Print(writer io.Writer, xPos, zPos int) (count int) {
	count = 0
	for i := 0; i < len(*vs); i += 129 {
		var x, z = (i / 129) / 17, (i / 129) % 17

		var column = (*vs)[i : i+129]
		for y, offset := range column {
			if offset != -1 {
				count++
				fmt.Fprintf(writer, "v %.2f %.2f %.2f\n", float64(x+xPos*16)*0.05, float64(y)*0.05, float64(z+zPos*16)*0.05)
			}
		}
	}
	return
}

type Vertex struct {
	x, y, z int
}

func printFace(xPos, zPos int, vertexes ...Vertex) {
	for _, vertex := range vertexes {
		fmt.Fprintf(out, "v %.2f %.2f %.2f\n", float64(vertex.x+xPos*16)*0.05, float64(vertex.y)*0.05, float64(vertex.z+zPos*16)*0.05)
	}
	fmt.Fprintf(out, "f")
	for i, _ := range vertexes {
		fmt.Fprintf(out, " -%d", len(vertexes)-i)
	}
	fmt.Fprintln(out)
}

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
		MTL{35, 0x7f7f7fff, "Wool D B"},
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

type blockRun struct {
	blockId        byte
	v1, v2, v3, v4 Vertex
	dirty          bool
}

func (r *blockRun) AddFace(faces AddFacer) {
	if r.dirty {
		faces.AddFace(r.blockId, r.v1, r.v2, r.v3, r.v4)
		r.dirty = false
	}
}

func (r *blockRun) Update(faces AddFacer, nr *blockRun, flag bool) {
	if !blockFaces {
		if r.dirty {
			if nr.blockId == r.blockId {
				if flag {
					r.v3 = nr.v3
					r.v4 = nr.v4
				} else {
					r.v2 = nr.v2
					r.v3 = nr.v3
				}
			} else {
				r.AddFace(faces)
				*r = *nr
			}
		} else {
			*r = *nr
		}
	} else {
		nr.AddFace(faces)
		nr.dirty = false
	}
}

func processBlocks(bytes []byte, faces AddFacer) {
	var blocks = Blocks(bytes)

	for i := 0; i < len(blocks); i += 128 {
		var x, z = (i / 128) / 16, (i / 128) % 16

		var r1, r2, r3, r4 blockRun

		var column = blocks[i : i+128]
		for y, blockId := range column {
			if y < yMin {
				continue
			}

			if blocks.IsBoundary(x, y-1, z, blockId) {
				faces.AddFace(blockId, Vertex{x, y, z}, Vertex{x + 1, y, z}, Vertex{x + 1, y, z + 1}, Vertex{x, y, z + 1})
			}

			if blocks.IsBoundary(x, y+1, z, blockId) {
				faces.AddFace(blockId, Vertex{x, y + 1, z}, Vertex{x, y + 1, z + 1}, Vertex{x + 1, y + 1, z + 1}, Vertex{x + 1, y + 1, z})
			}

			if blocks.IsBoundary(x-1, y, z, blockId) {
				r1.Update(faces, &blockRun{blockId, Vertex{x, y, z}, Vertex{x, y, z + 1}, Vertex{x, y + 1, z + 1}, Vertex{x, y + 1, z}, true}, true)
			} else {
				r1.AddFace(faces)
			}

			if blocks.IsBoundary(x+1, y, z, blockId) {
				r2.Update(faces, &blockRun{blockId, Vertex{x + 1, y, z}, Vertex{x + 1, y + 1, z}, Vertex{x + 1, y + 1, z + 1}, Vertex{x + 1, y, z + 1}, true}, false)
			} else {
				r2.AddFace(faces)
			}

			if blocks.IsBoundary(x, y, z-1, blockId) {
				r3.Update(faces, &blockRun{blockId, Vertex{x, y, z}, Vertex{x, y + 1, z}, Vertex{x + 1, y + 1, z}, Vertex{x + 1, y, z}, true}, false)
			} else {
				r3.AddFace(faces)
			}

			if blocks.IsBoundary(x, y, z+1, blockId) {
				r4.Update(faces, &blockRun{blockId, Vertex{x, y, z + 1}, Vertex{x + 1, y, z + 1}, Vertex{x + 1, y + 1, z + 1}, Vertex{x, y + 1, z + 1}, true}, true)
			} else {
				r4.AddFace(faces)
			}
		}

		r1.AddFace(faces)
		r2.AddFace(faces)
		r3.AddFace(faces)
		r4.AddFace(faces)
	}
}
