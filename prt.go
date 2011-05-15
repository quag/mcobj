package main

import (
	"bufio"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

type PrtGenerator struct {
	enclosedsChan chan *EnclosedChunkJob
	completeChan  chan bool

	outFile *os.File
	w       *bufio.Writer
	zw      io.WriteCloser

	particleCount int64
	total         int
	boundary      *BoundaryLocator
}

func (o *PrtGenerator) Start(outFilename string, total int, maxProcs int, boundary *BoundaryLocator) {
	o.enclosedsChan = make(chan *EnclosedChunkJob, maxProcs*2)
	o.completeChan = make(chan bool)
	o.total = total
	o.boundary = boundary

	maxProcs = 1
	for i := 0; i < maxProcs; i++ {
		go o.chunkProcessor()
	}

	var openErr os.Error

	o.outFile, openErr = os.Create(outFilename)
	if openErr != nil {
		fmt.Fprintln(os.Stderr, openErr) // TODO: return openErr
		return
	}

	o.w = bufio.NewWriter(o.outFile)
	WriteHeader(o.w, -1, []ChannelDefinition{{"Position", 4, 3, 0}, {"BlockID", 1, 1, 12}})

	var zErr os.Error
	o.zw, zErr = zlib.NewWriterLevel(o.w, zlib.NoCompression)
	if zErr != nil {
		fmt.Fprintln(os.Stderr, zErr) // TODO: return zErr
		return
	}
}

func (o *PrtGenerator) chunkProcessor() {
	var chunkCount = 0
	for {
		var job = <-o.enclosedsChan

		var e = job.enclosed

		for i := 0; i < len(e.blocks); i += 128 {
			var x, z = (i / 128) / 16, (i / 128) % 16

			var column = BlockColumn(e.blocks[i : i+128])
			for y, blockId := range column {
				if y < yMin {
					continue
				}

				switch {
				case o.boundary.IsBoundary(blockId, e.Get(x, y-1, z)):
					fallthrough
				case o.boundary.IsBoundary(blockId, e.Get(x, y+1, z)):
					fallthrough
				case o.boundary.IsBoundary(blockId, e.Get(x-1, y, z)):
					fallthrough
				case o.boundary.IsBoundary(blockId, e.Get(x+1, y, z)):
					fallthrough
				case o.boundary.IsBoundary(blockId, e.Get(x, y, z-1)):
					fallthrough
				case o.boundary.IsBoundary(blockId, e.Get(x, y, z+1)):
					o.particleCount++
					var (
						xa = -(x + e.xPos*16)
						ya = y - 64
						za = z + e.zPos*16
					)
					binary.Write(o.zw, binary.LittleEndian, float32(xa*2))
					binary.Write(o.zw, binary.LittleEndian, float32(za*2))
					binary.Write(o.zw, binary.LittleEndian, float32(ya*2))
					binary.Write(o.zw, binary.LittleEndian, int32(blockId))
				}
			}
		}

		chunkCount++
		fmt.Printf("%4v/%-4v (%3v,%3v) Particles: %d\n", chunkCount, o.total, job.enclosed.xPos, job.enclosed.zPos, o.particleCount)

		if job.last {
			o.completeChan <- true
			break
		}
	}
}

func (o *PrtGenerator) Close() {
	o.zw.Close()
	o.w.Flush()
	UpdateParticleCount(o.outFile, o.particleCount)
	o.outFile.Close()
}

func (o *PrtGenerator) GetEnclosedJobsChan() chan *EnclosedChunkJob {
	return o.enclosedsChan
}

func (o *PrtGenerator) GetCompleteChan() chan bool {
	return o.completeChan
}

type ChannelDefinition struct {
	Name                    string // max of 31 characters and must meet the regex [a-zA-Z_][0-9a-zA-Z_]*
	DataType, Arity, Offset int32
}

// http://software.primefocusworld.com/software/support/krakatoa/prt_file_format.php
func WriteHeader(w io.Writer, particleCount int64, channels []ChannelDefinition) {
	// Header (56 bytes)
	var magic = []byte{192, 'P', 'R', 'T', '\r', '\n', 26, '\n'}
	w.Write(magic)

	var headerLength = uint32(56)
	binary.Write(w, binary.LittleEndian, headerLength)

	var signature = make([]byte, 32)
	copy(signature, []byte("Extensible Particle Format"))
	w.Write(signature)

	var version = uint32(1)
	binary.Write(w, binary.LittleEndian, version)

	binary.Write(w, binary.LittleEndian, particleCount)

	// Reserved bytes (4 bytes)
	var reserved = int32(4)
	binary.Write(w, binary.LittleEndian, reserved)

	// Channel definition header (8 bytes)
	binary.Write(w, binary.LittleEndian, int32(len(channels)))

	var channelDefinitionSize = int32(44)
	binary.Write(w, binary.LittleEndian, channelDefinitionSize)

	for _, channel := range channels {
		var nameBytes = make([]byte, 32)
		copy(nameBytes, []byte(channel.Name))
		w.Write(nameBytes)

		binary.Write(w, binary.LittleEndian, channel.DataType)
		binary.Write(w, binary.LittleEndian, channel.Arity)
		binary.Write(w, binary.LittleEndian, channel.Offset)
	}
}

func UpdateParticleCount(file *os.File, particleCount int64) os.Error {
	var storedOffset, err = file.Seek(0, 1)
	if err != nil {
		return err
	}
	_, err = file.Seek(0x30, 0)
	if err != nil {
		return err
	}
	err = binary.Write(file, binary.LittleEndian, particleCount)
	if err != nil {
		return err
	}
	_, err = file.Seek(storedOffset, 0)
	return err
}
