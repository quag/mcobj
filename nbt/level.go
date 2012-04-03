package nbt

import (
	"compress/gzip"
	"errors"
	"io"
)

var (
	DataStructNotFound = errors.New("'Data' struct not found")
	SpawnIntNotFound   = errors.New("SpawnX/SpawnY/SpawnZ int32s not found")
)

type Level struct {
	SpawnX, SpawnY, SpawnZ int
}

func ReadLevelDat(reader io.Reader) (*Level, error) {
	r, err := gzip.NewReader(reader)
	defer r.Close()
	if err != nil {
		return nil, err
	}

	return ReadLevelNbt(r)
}

func ReadLevelNbt(reader io.Reader) (*Level, error) {
	root, err := Parse(reader)
	if err != nil {
		return nil, err
	}

	dataValue := root["Data"]
	if dataValue == nil {
		return nil, DataStructNotFound
	}

	data, ok := dataValue.(map[string]interface{})
	if !ok {
		return nil, DataStructNotFound
	}

	xval, xok := data["SpawnX"]
	yval, yok := data["SpawnY"]
	zval, zok := data["SpawnZ"]

	if !xok || !yok || !zok {
		return nil, SpawnIntNotFound
	}

	level := new(Level)
	level.SpawnX, xok = xval.(int)
	level.SpawnY, yok = yval.(int)
	level.SpawnZ, zok = zval.(int)

	if !xok || !yok || !zok {
		return nil, SpawnIntNotFound
	}

	return level, nil
}
