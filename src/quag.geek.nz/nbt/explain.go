package nbt

import (
	"fmt"
	"io"
	"os"
)

func Explain(r io.Reader, w io.Writer) os.Error {
	e := &explainer{w, pathStack{make([]string, 0, 8)}}

	nr := NewReader(r)
	for {
		err := e.parseStruct(nr, false)
		if err == os.EOF {
			break
		} else if err != nil {
			return err
		}
	}

	return nil
}

type explainer struct {
	w     io.Writer
	stack pathStack
}

func (e *explainer) parseStruct(nr *Reader, listStruct bool) os.Error {
	structDepth := 0
	if listStruct {
		structDepth = 1
	}

	for {
		typeId, name, err := nr.ReadTag()
		if err != nil {
			return err
		}

		if typeId != TagStructEnd {
			e.stack.push(name)
		}

		e.RecordTag(typeId, name)
		switch typeId {
		case TagStruct:
			e.RecordNoValue()
			structDepth++
		case TagStructEnd:
			e.RecordNoValue()
			structDepth--
			if structDepth == 0 {
				return nil
			}
		case TagByteArray:
			e.RecordValue(nr.ReadBytes())
		case TagInt8:
			e.RecordValue(nr.ReadInt8())
		case TagInt16:
			e.RecordValue(nr.ReadInt16())
		case TagInt32:
			e.RecordValue(nr.ReadInt32())
		case TagInt64:
			e.RecordValue(nr.ReadInt64())
		case TagFloat32:
			e.RecordValue(nr.ReadFloat32())
		case TagFloat64:
			e.RecordValue(nr.ReadFloat64())
		case TagString:
			e.RecordValue(nr.ReadString())
		case TagList:
			itemTypeId, length, err := nr.ReadListHeader()
			if err != nil {
				return err
			}
			e.RecordList(itemTypeId, length)
			switch TypeId(itemTypeId) {
			case TagInt8:
				list := make([]int, length)
				for i := 0; i < length; i++ {
					x, err := nr.ReadInt8()
					list[i] = x
					if err != nil {
						return err
					}
				}
				e.RecordValue(list, nil)
			case TagFloat32:
				list := make([]float32, length)
				for i := 0; i < length; i++ {
					x, err := nr.ReadFloat32()
					list[i] = x
					if err != nil {
						return err
					}
				}
				e.RecordValue(list, nil)
			case TagFloat64:
				list := make([]float64, length)
				for i := 0; i < length; i++ {
					x, err := nr.ReadFloat64()
					list[i] = x
					if err != nil {
						return err
					}
				}
				e.RecordValue(list, nil)
			case TagStruct:
				e.RecordNoValue()

				for i := 0; i < length; i++ {
					e.stack.push(fmt.Sprintf("[%d]", i))

					err := e.parseStruct(nr, true)
					if err != nil {
						return err
					}

					e.stack.pop()
				}
			default:
				return os.NewError(fmt.Sprintf("reading lists of typeId %d not supported. length:%d", itemTypeId, length))
			}
		}

		if typeId != TagStruct {
			e.stack.pop()
		}
	}

	return nil
}

func (e *explainer) RecordTag(typeId TypeId, name string) {
	fmt.Fprintf(e.w, "[%2d %-20s] ", typeId, name)
	for i, name := range e.stack.path {
		if i != 0 {
			fmt.Fprintf(e.w, ".")
		}
		fmt.Fprintf(e.w, "%s", name)
	}

	if len(e.stack.path) != 0 {
		fmt.Fprintf(e.w, " ")
	}
}

func (e *explainer) RecordList(itemTypeId TypeId, length int) {
	fmt.Fprintf(e.w, " (%2d, %d) ", itemTypeId, length)
}

func (e *explainer) RecordValue(value interface{}, err os.Error) {
	if err != nil {
		fmt.Fprintln(e.w, err)
	}
	fmt.Fprintf(e.w, "'%v'\n", value)
	//fmt.Fprintln(e.w)
}

func (e *explainer) RecordNoValue() {
	fmt.Fprintln(e.w)
}

type pathStack struct {
	path []string
}

func (ps *pathStack) push(name string) {
	ps.path = append(ps.path, name)
}

func (ps *pathStack) pop() {
	if len(ps.path) != 0 {
		ps.path = ps.path[0 : len(ps.path)-1]
	}
}
