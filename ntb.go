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
    tagStructEnd = 0 // No name. Single zero byte.
    tagInt8 = 1 // A single signed byte (8 bits)
    tagInt16 = 2 // A signed short (16 bits, big endian)
    tagInt32 = 3 // A signed int (32 bits, big endian)
    tagInt64 = 4 // A signed long (64 bits, big endian)
    tagFloat32 = 5 // A floating point value (32 bits, big endian, IEEE 754-2008, binary32)
    tagFloat64 = 6 // A floating point value (64 bits, big endian, IEEE 754-2008, binary64)
    tagByteArray = 7 // { TAG_Int length; An array of bytes of unspecified format. The length of this array is <length> bytes }
    tagString = 8 // { TAG_Short length; An array of bytes defining a string in UTF-8 format. The length of this array is <length> bytes }
    tagList = 9 // { TAG_Byte tagId; TAG_Int length; A sequential list of Tags (not Named Tags), of type <typeId>. The length of this array is <length> Tags. } Notes: All tags share the same type.
    tagStruct = 10 // { A sequential list of Named Tags. This array keeps going until a TAG_End is found.; TAG_End end } Notes: If there's a nested TAG_Compound within this tag, that one will also have a TAG_End, so simply reading until the next TAG_End will not work. The names of the named tags have to be unique within each TAG_Compound The order of the tags is not guaranteed.
)

const (
    trace = false
)

var (
    out *bufio.Writer
)

func main() {
    var outFile, outErr = os.Open("a.obj", os.O_WRONLY | os.O_CREATE | os.O_TRUNC, 0666)
    if outErr != nil {
        fmt.Fprintln(os.Stderr, outErr)
        return
    }
    defer outFile.Close()
    var bufErr os.Error
    out, bufErr = bufio.NewWriterSize(outFile, 1024 * 1024)
    if bufErr != nil {
        fmt.Fprintln(os.Stderr, bufErr)
        return
    }
    defer out.Flush()

    fmt.Fprintln(out, "mtllib a.mtl")

    flag.Parse()
    for i := 0; i < flag.NArg(); i++ {
        var filepath = flag.Arg(i)
        var fi, err = os.Stat(filepath)
        if err != nil {
            fmt.Fprintln(os.Stderr, err)
        }

        switch {
        case fi.IsRegular():
            processFile(filepath)
        case fi.IsDirectory():
            var errors = make(chan os.Error, 5)
            var done = make(chan bool)
            go func(){
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

type visitor struct {}

func (v *visitor) VisitDir(dir string, f *os.FileInfo) bool {
    return true
}

func (v *visitor) VisitFile(file string, f *os.FileInfo) {
    var _, name = path.Split(file)
    var match, err = path.Match("c.*.dat", name)
    if match && err == nil {
        processFile(file)
    }
}

func processFile(filename string) bool {
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
                processBlocks(bytes, xPos, zPos)
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
        a = a << 8 + int(b)
    }

    return a, nil
}

type Blocks []byte

func (b *Blocks) Get(x, y, z int) byte {
    return (*b)[y + (z*128 + (x * 128 * 16))]
}

func (b *Blocks) IsEmpty(x, y, z int) bool {
    if x < 0 || y < 0 || z < 0 || x >= 16 || y >= 128 || z >= 16 {
        return true
    }
    return isEmptyBlockId(b.Get(x, y, z))
}

func (b *Blocks) IsBoundary(x, y, z int, blockId byte) bool {
    if x < 0 || y < 0 || z < 0 || x >= 16 || y >= 128 || z >= 16 {
        return blockId != 0
    }

    var otherBlockId = b.Get(x, y, z)

    if blockId == 1 && otherBlockId == 0 {
        return true
    }

    return (blockId != 0 && blockId != 1) && (otherBlockId == 0 || blockId == 1)
}

func isEmptyBlockId(blockId byte) bool {
    return blockId == 0 || blockId == 1
}

func processBlocks(bytes []byte, xPos, zPos int) {
    var blocks = Blocks(bytes)

    fmt.Fprintf(out, "# %v,%v\n", xPos, zPos)

    xPos = xPos * 16
    zPos = zPos * 16

    var faces = 0

    for i := 0; i < len(blocks); i += 128 {
        var x, z = (i / 128) / 16, (i / 128) % 16

        var column = blocks[i:i+128]
        for y, blockId := range column {
            if y < 62 { continue }

            if blocks.IsBoundary(x, y-1, z, blockId) {
                printMtl(blockId)
                printFace(v{xPos+x, y, zPos+z}, v{xPos+x+1, y, zPos+z}, v{xPos+x+1, y, zPos+z+1}, v{xPos+x, y, zPos+z+1})
                faces++
            }

            if blocks.IsBoundary(x, y+1, z, blockId) {
                printMtl(blockId)
                printFace(v{xPos+x, y+1, zPos+z}, v{xPos+x, y+1, zPos+z+1}, v{xPos+x+1, y+1, zPos+z+1}, v{xPos+x+1, y+1, zPos+z})
                faces++
            }

            if blocks.IsBoundary(x-1, y, z, blockId) {
                printMtl(blockId)
                printFace(v{xPos+x, y, zPos+z}, v{xPos+x, y, zPos+z+1}, v{xPos+x, y+1, zPos+z+1}, v{xPos+x, y+1, zPos+z})
                faces++
            }

            if blocks.IsBoundary(x+1, y, z, blockId) {
                printMtl(blockId)
                printFace(v{xPos+x+1, y, zPos+z}, v{xPos+x+1, y+1, zPos+z}, v{xPos+x+1, y+1, zPos+z+1}, v{xPos+x+1, y, zPos+z+1})
                faces++
            }

            if blocks.IsBoundary(x, y, z-1, blockId) {
                printMtl(blockId)
                printFace(v{xPos+x, y, zPos+z}, v{xPos+x, y+1, zPos+z}, v{xPos+x+1, y+1, zPos+z}, v{xPos+x+1, y, zPos+z})
                faces++
            }

            if blocks.IsBoundary(x, y, z+1, blockId) {
                printMtl(blockId)
                printFace(v{xPos+x, y, zPos+z+1}, v{xPos+x+1, y, zPos+z+1}, v{xPos+x+1, y+1, zPos+z+1}, v{xPos+x, y+1, zPos+z+1})
                faces++
            }
        }
        fmt.Fprintln(out)
    }
    fmt.Fprintln(os.Stderr, "Faces:", faces)
}

func printMtl(blockId byte) {
    switch blockId {
    case 1:
        fmt.Fprintln(out, "usemtl stone")
    case 2:
        fmt.Fprintln(out, "usemtl grass")
    case 3:
        fmt.Fprintln(out, "usemtl dirt")
    case 4:
        fmt.Fprintln(out, "usemtl cobble")
    case 5:
        fmt.Fprintln(out, "usemtl plank")
    case 7:
        fmt.Fprintln(out, "usemtl bedrock")
    case 8:
    case 9:
        fmt.Fprintln(out, "usemtl water")
    case 12:
        fmt.Fprintln(out, "usemtl sand")
    case 17:
        fmt.Fprintln(out, "usemtl wood")
    case 6: // Sapling
    case 18:
        fmt.Fprintln(out, "usemtl leaves")
    default:
        fmt.Fprintln(out, "usemtl default")
    }
}

func printVertex(x, y, z int) {
    fmt.Fprintf(out, "v %.2f %.2f %.2f\n", float64(x)*0.05, float64(y)*0.05, float64(z)*0.05)
}

func printFaceN(vertexes... int) {
    fmt.Fprintf(out, "f")
    for _, vertex := range vertexes {
        fmt.Fprintf(out, " %v", vertex)
    }
    fmt.Fprintln(out)
}

type v struct { x, y, z int }

func printFace(vertexes... v) {
    for _, vertex := range vertexes {
        printVertex(vertex.x, vertex.y, vertex.z)
    }
    fmt.Fprintf(out, "f")
    for i, _ := range vertexes {
        fmt.Fprintf(out, " -%d", len(vertexes) - i)
    }
    fmt.Fprintln(out)
}
