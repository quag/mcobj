package main

import (
	"bufio"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"math"
	"strconv"
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
	out        *bufio.Writer
	faces      *Faces
	sideCache  *SideCache
	yMin       int
	blockFaces bool
	hideBottom bool
	hideStone  bool
	noColor    bool
	cacheSides bool

	faceCount int
	faceLimit int

	chunkCount int
	chunkLimit int
)

func base36(i int) string {
	return strconv.Itob(i, 36)
}

func encodeFolder(i int) string {
	return base36(((i % 64) + 64) % 64)
}

func chunkPath(world string, x, z int) string {
	return path.Join(world, encodeFolder(x), encodeFolder(z), "c."+base36(x)+"."+base36(z)+".dat")
}

func zigzag(n int) int {
	return (n << 1) ^ (n >> 31)
}

func unzigzag(n int) int {
	return (n >> 1) ^ (-(n & 1))
}

func main() {
	var cx, cz int
	var square int

	var filename string
	flag.StringVar(&filename, "o", "a.obj", "Name for output file")
	flag.IntVar(&yMin, "y", 0, "Omit all blocks below this height. 63 is sea level")
	flag.BoolVar(&blockFaces, "bf", false, "Don't combine adjacent faces of the same block within a column")
	flag.BoolVar(&hideBottom, "hb", false, "Hide bottom of world")
	flag.BoolVar(&hideStone, "hs", false, "Hide stone")
	flag.BoolVar(&noColor, "g", false, "Omit materials")
	flag.BoolVar(&cacheSides, "cs", false, "Cache sides. Memory usage will increase as chunks are processed")
	flag.IntVar(&cx, "cx", 0, "Center x coordinate")
	flag.IntVar(&cz, "cz", 0, "Center z coordinate")
	flag.IntVar(&faceLimit, "fk", math.MaxInt32, "Face limit (thousands of faces)")
	flag.IntVar(&square, "s", math.MaxInt32, "Chunk square size")
	flag.Parse()

	if faceLimit != math.MaxInt32 {
		faceLimit *= 1000
	}

	if square != math.MaxInt32 {
		chunkLimit = square * square
	} else {
		chunkLimit = math.MaxInt32
	}

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

		sideCache = new(SideCache)
		faces = NewFaces(sideCache)

		fmt.Fprintln(out, "mtllib", path.Base(mtlFilename))

		for i := 0; i < flag.NArg(); i++ {
			var filepath = flag.Arg(i)
			var fi, err = os.Stat(filepath)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				continue
			}

			switch {
			case fi.IsRegular():
				processChunk(filepath, faces)
			case fi.IsDirectory():
				var errors = make(chan os.Error, 5)
				var done = make(chan bool)
				go func() {
					for error := range errors {
						fmt.Fprintln(os.Stderr, error)
					}
					done <- true
				}()
				var v = &visitor{make(map[string]bool)}
				path.Walk(filepath, v, errors)
				close(errors)
				<-done

				var total = len(v.chunks)

				for i := 0; moreChunks(v.chunks); i++ {
					for x := 0; x < i && moreChunks(v.chunks); x++ {
						for z := 0; z < i && moreChunks(v.chunks); z++ {
							var (
								ax = cx + unzigzag(x)
								az = cz + unzigzag(z)
							)

							var chunk = chunkPath(filepath, ax, az)
							var _, exists = v.chunks[chunk]
							if exists {
								loadSide(sideCache, filepath, v.chunks, ax-1, az)
								loadSide(sideCache, filepath, v.chunks, ax+1, az)
								loadSide(sideCache, filepath, v.chunks, ax, az-1)
								loadSide(sideCache, filepath, v.chunks, ax, az+1)

								v.chunks[chunk] = false, false
								fmt.Printf("%v/%v ", total-len(v.chunks), total)
								processChunk(chunk, faces)
								chunkCount++

								if !cacheSides {
									sideCache.Clear()
								}
							}
						}
					}
				}
			}
		}
	}
}

func moreChunks(chunks map[string]bool) bool {
	return len(chunks) > 0 && faceCount < faceLimit && chunkCount < chunkLimit
}

func loadSide(sideCache *SideCache, world string, chunks map[string]bool, x, z int) {
	if !sideCache.HasSide(x, z) {
		var fileName = chunkPath(world, x, z)
		var fileExists bool
		if cacheSides {
			_, fileExists = chunks[fileName]
		} else {
			var _, err = os.Stat(fileName)
			fileExists = err == nil
		}
		if fileExists {
			processChunk(fileName, sideCache)
		}
	}
}

type visitor struct {
	chunks map[string]bool
}

func (v *visitor) VisitDir(dir string, f *os.FileInfo) bool {
	return true
}

func (v *visitor) VisitFile(file string, f *os.FileInfo) {
	var match, err = path.Match("c.*.dat", path.Base(file))
	if match && err == nil {
		v.chunks[file] = true
	}
}

type ProcessBlocker interface {
	ProcessBlock(xPos, zPos int, blocks Blocks)
}

type Blocks []byte

type BlockColumn []byte

func (b *Blocks) Get(x, y, z int) byte {
	return (*b)[y+(z*128+(x*128*16))]
}

func (b *Blocks) Column(x, z int) BlockColumn {
	var i = 128 * (z + x*16)
	return BlockColumn((*b)[i : i+128])
}

func processChunk(filename string, processor ProcessBlocker) bool {
	fmt.Fprintln(out, "#", filename)
	//fmt.Fprintln(os.Stderr, filename)
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
				processor.ProcessBlock(xPos, zPos, Blocks(bytes))
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