package nbt

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
)

type TypeId byte

const (
	TagStructEnd TypeId = 0  // No name. Single zero byte.
	TagInt8      TypeId = 1  // A single signed byte (8 bits)
	TagInt16     TypeId = 2  // A signed short (16 bits, big endian)
	TagInt32     TypeId = 3  // A signed int (32 bits, big endian)
	TagInt64     TypeId = 4  // A signed long (64 bits, big endian)
	TagFloat32   TypeId = 5  // A floating point value (32 bits, big endian, IEEE 754-2008, binary32)
	TagFloat64   TypeId = 6  // A floating point value (64 bits, big endian, IEEE 754-2008, binary64)
	TagByteArray TypeId = 7  // { TAG_Int length; An array of bytes of unspecified format. The length of this array is <length> bytes }
	TagString    TypeId = 8  // { TAG_Short length; An array of bytes defining a string in UTF-8 format. The length of this array is <length> bytes }
	TagList      TypeId = 9  // { TAG_Byte tagId; TAG_Int length; A sequential list of Tags (not Named Tags), of type <typeId>. The length of this array is <length> Tags. } Notes: All tags share the same type.
	TagStruct    TypeId = 10 // { A sequential list of Named Tags. This array keeps going until a TAG_End is found.; TAG_End end } Notes: If there's a nested TAG_Compound within this tag, that one will also have a TAG_End, so simply reading until the next TAG_End will not work. The names of the named tags have to be unique within each TAG_Compound The order of the tags is not guaranteed.
	TagIntArray  TypeId = 11 // { TAG_Int length; An array of ints. The length of this array is <length> ints }
)

type Reader struct {
	r *bufio.Reader
}

func Parse(r io.Reader) (map[string]interface{}, error) {
	nr := NewReader(r)
	typeId, _, err := nr.ReadTag()
	if err != nil {
		return nil, err
	}

	value, err := nr.ReadValue(typeId)
	if err != nil {
		return nil, err
	}
	return value.(map[string]interface{}), nil
}

func NewReader(r io.Reader) *Reader {
	return &Reader{bufio.NewReader(r)}
}

func (r *Reader) ReadTag() (typeId TypeId, name string, err error) {
	typeId, err = r.readTypeId()
	if err != nil || typeId == 0 {
		return typeId, "", err
	}

	name, err = r.ReadString()
	if err != nil {
		return typeId, name, err
	}

	return typeId, name, nil
}

func (r *Reader) ReadListHeader() (itemTypeId TypeId, length int, err error) {
	length = 0

	itemTypeId, err = r.readTypeId()
	if err == nil {
		length, err = r.ReadInt32()
	}

	return
}

func (r *Reader) ReadString() (string, error) {
	var length, err1 = r.ReadInt16()
	if err1 != nil {
		return "", err1
	}

	var bytes = make([]byte, length)
	var _, err = io.ReadFull(r.r, bytes)
	return string(bytes), err
}

func (r *Reader) ReadBytes() ([]byte, error) {
	var length, err1 = r.ReadInt32()
	if err1 != nil {
		return nil, err1
	}

	var bytes = make([]byte, length)
	var _, err = io.ReadFull(r.r, bytes)
	return bytes, err
}

func (r *Reader) ReadInts() ([]int, error) {
	length, err := r.ReadInt32()
	if err != nil {
		return nil, err
	}

	ints := make([]int, length)
	for i := 0; i < length; i++ {
		ints[i], err = r.ReadInt32()
		if err != nil {
			return nil, err
		}
	}
	return ints, nil
}

func (r *Reader) ReadInt8() (int, error) {
	return r.readIntN(1)
}

func (r *Reader) ReadInt16() (int, error) {
	return r.readIntN(2)
}

func (r *Reader) ReadInt32() (int, error) {
	return r.readIntN(4)
}

func (r *Reader) ReadInt64() (int, error) {
	return r.readIntN(8)
}

func (r *Reader) ReadFloat32() (float32, error) {
	x, err := r.readUintN(4)
	return math.Float32frombits(uint32(x)), err
}

func (r *Reader) ReadFloat64() (float64, error) {
	x, err := r.readUintN(8)
	return math.Float64frombits(x), err
}

func (r *Reader) readTypeId() (TypeId, error) {
	id, err := r.r.ReadByte()
	return TypeId(id), err
}

func (r *Reader) readIntN(n int) (int, error) {
	var a int = 0

	for i := 0; i < n; i++ {
		var b, err = r.r.ReadByte()
		if err != nil {
			return a, err
		}
		a = a<<8 + int(b)
	}

	return a, nil
}

func (r *Reader) readUintN(n int) (uint64, error) {
	var a uint64 = 0

	for i := 0; i < n; i++ {
		var b, err = r.r.ReadByte()
		if err != nil {
			return a, err
		}
		a = a<<8 + uint64(b)
	}

	return a, nil
}

func (r *Reader) ReadStruct() (map[string]interface{}, error) {
	s := make(map[string]interface{})
	for {
		typeId, name, err := r.ReadTag()
		if err != nil {
			return s, err
		}
		if typeId == TagStructEnd {
			break
		}
		x, err := r.ReadValue(typeId)
		s[name] = x
		if err != nil {
			return s, err
		}
	}
	return s, nil
}

func (r *Reader) ReadValue(typeId TypeId) (interface{}, error) {
	switch typeId {
	case TagStruct:
		return r.ReadStruct()
	case TagStructEnd:
		return nil, nil
	case TagByteArray:
		return r.ReadBytes()
	case TagInt8:
		return r.ReadInt8()
	case TagInt16:
		return r.ReadInt16()
	case TagInt32:
		return r.ReadInt32()
	case TagInt64:
		return r.ReadInt64()
	case TagFloat32:
		return r.ReadFloat32()
	case TagFloat64:
		return r.ReadFloat64()
	case TagString:
		return r.ReadString()
	case TagList:
		itemTypeId, length, err := r.ReadListHeader()
		if err != nil {
			return nil, err
		}
		switch TypeId(itemTypeId) {
		case TagInt8:
			list := make([]int, length)
			for i := 0; i < length; i++ {
				x, err := r.ReadInt8()
				list[i] = x
				if err != nil {
					return list, err
				}
			}
			return list, nil
		case TagFloat32:
			list := make([]float32, length)
			for i := 0; i < length; i++ {
				x, err := r.ReadFloat32()
				list[i] = x
				if err != nil {
					return list, err
				}
			}
			return list, nil
		case TagFloat64:
			list := make([]float64, length)
			for i := 0; i < length; i++ {
				x, err := r.ReadFloat64()
				list[i] = x
				if err != nil {
					return list, err
				}
			}
			return list, nil
		case TagStruct:
			list := make([]interface{}, length)
			for i := 0; i < length; i++ {
				s := make(map[string]interface{})
				s, err := r.ReadStruct()
				list[i] = s
				if err != nil {
					return list, err
				}
			}
			return list, nil
		default:
			return nil, errors.New(fmt.Sprintf("reading lists of typeId %d not supported. length:%d", itemTypeId, length))
		}
	}

	return nil, errors.New(fmt.Sprintf("reading typeId %d not supported", typeId))
}
