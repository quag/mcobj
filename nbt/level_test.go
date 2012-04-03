package nbt

import (
	"bytes"
	"compress/gzip"
	"io"
	"testing"
)

func TestReadSpawnXYZ(t *testing.T) {
	level, err := readLevelBytes(10, 0, 0, 10, 0, 4, 'D', 'a', 't', 'a', 3, 0, 6, 'S', 'p', 'a', 'w', 'n', 'X', 0, 0, 0, 13, 3, 0, 6, 'S', 'p', 'a', 'w', 'n', 'Y', 0, 0, 0, 14, 3, 0, 6, 'S', 'p', 'a', 'w', 'n', 'Z', 0, 0, 0, 15, 0, 0)

	checkError(t, err, nil)

	if level == nil {
		t.Error("Level is nil")
	} else {
		if level.SpawnX != 13 {
			t.Errorf("SpawnX %d not 13", level.SpawnX)
		}

		if level.SpawnY != 14 {
			t.Errorf("SpawnY %d not 14", level.SpawnY)
		}

		if level.SpawnZ != 15 {
			t.Errorf("SpawnZ %d not 15", level.SpawnZ)
		}
	}
}

func TestLevelParseError(t *testing.T) {
	checkLevelReadError(t, io.EOF, 0xff)
}

func TestLevelDataNotFound(t *testing.T) {
	checkLevelReadError(t, DataStructNotFound, 10, 0, 0, 10, 0, 4, 'Z', 'Z', 'Z', 'Z', 0, 0)
}

func TestLevelDataNotStruct(t *testing.T) {
	checkLevelReadError(t, DataStructNotFound, 10, 0, 0, 3, 0, 4, 'D', 'a', 't', 'a', 1, 2, 3, 4, 0, 0)
}

func TestSpawnNotFound(t *testing.T) {
	checkLevelReadError(t, SpawnIntNotFound, 10, 0, 0, 10, 0, 4, 'D', 'a', 't', 'a', 3, 0, 6, 'S', 'p', 'a', 'w', 'n', 'A', 0, 0, 0, 13, 3, 0, 6, 'S', 'p', 'a', 'w', 'n', 'B', 0, 0, 0, 14, 3, 0, 6, 'S', 'p', 'a', 'w', 'n', 'C', 0, 0, 0, 15, 0, 0)
}

func TestSpawnNotInt(t *testing.T) {
	checkLevelReadError(t, SpawnIntNotFound, 10, 0, 0, 10, 0, 4, 'D', 'a', 't', 'a', 10, 0, 6, 'S', 'p', 'a', 'w', 'n', 'X', 0, 10, 0, 6, 'S', 'p', 'a', 'w', 'n', 'Y', 0, 10, 0, 6, 'S', 'p', 'a', 'w', 'n', 'Z', 0, 0, 0)
}

// TODO: Data/Player/Pos

func readLevelBytes(b ...byte) (*Level, error) {
	r, err := gzipBytesReader(b)
	if err != nil {
		return nil, err
	}
	return ReadLevelDat(r)
}

func gzipBytesReader(b []byte) (io.Reader, error) {
	buffer := new(bytes.Buffer)
	gw := gzip.NewWriter(buffer)
	gw.Write(b)
	gw.Close()
	return buffer, nil
}

func checkLevelReadError(t *testing.T, expectedErr error, bytes ...byte) {
	level, err := readLevelBytes(bytes...)
	checkError(t, err, expectedErr)
	checkLevelNil(t, level)
}

func checkError(t *testing.T, err, expectedErr error) {
	if err != expectedErr {
		t.Errorf("Error was %q not %q", err, expectedErr)
	}
}

func checkLevelNil(t *testing.T, level *Level) {
	if level != nil {
		t.Errorf("Level %v is not nil", level)
	}
}
