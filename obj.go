package main

import (
	"bufio"
	"fmt"
	"os"
	"path"
)

type ObjGenerator struct {
	enclosedsChan  chan *EnclosedChunkJob
	writeFacesChan chan *WriteFacesJob
	completeChan   chan bool

	freelist chan *MemoryWriter

	total int

	outFile *os.File
	out     *bufio.Writer
}

func (o *ObjGenerator) Start(outFilename string, total int, maxProcs int, boundary *BoundaryLocator) {
	o.enclosedsChan = make(chan *EnclosedChunkJob, maxProcs*2)
	o.writeFacesChan = make(chan *WriteFacesJob, maxProcs*2)
	o.completeChan = make(chan bool)

	o.freelist = make(chan *MemoryWriter, maxProcs*2)
	o.total = total

	for i := 0; i < maxProcs; i++ {
		go func() {
			var faces Faces
			faces.boundary = boundary
			for {
				var job = <-o.enclosedsChan

				var b *MemoryWriter
				select {
				case b = <-o.freelist:
					// Got a buffer
				default:
					b = &MemoryWriter{make([]byte, 0, 128*1024)}
				}

				var faceCount = faces.ProcessChunk(job.enclosed, b)
				fmt.Fprintln(b)

				o.writeFacesChan <- &WriteFacesJob{job.enclosed.xPos, job.enclosed.zPos, faceCount, b, job.last}
			}
		}()
	}

	go func() {
		var chunkCount = 0
		var size = 0
		for {
			var job = <-o.writeFacesChan
			chunkCount++
			o.out.Write(job.b.buf)
			o.out.Flush()

			size += len(job.b.buf)
			fmt.Printf("%4v/%-4v (%3v,%3v) Faces: %4d Size: %4.1fMB\n", chunkCount, o.total, job.xPos, job.zPos, job.faceCount, float64(size)/1024/1024)

			job.b.Clean()
			select {
			case o.freelist <- job.b:
				// buffer added to free list
			default:
				// free list is full, discard the buffer
			}

			if job.last {
				o.completeChan <- true
			}
		}
	}()

	var mtlFilename = fmt.Sprintf("%s.mtl", outFilename[:len(outFilename)-len(path.Ext(outFilename))])
	var mtlErr = writeMtlFile(mtlFilename)
	if mtlErr != nil {
		fmt.Fprintln(os.Stderr, mtlErr)
		return
	}

	var outFile, outErr = os.Create(outFilename)
	if outErr != nil {
		fmt.Fprintln(os.Stderr, outErr)
		return
	}
	defer func() {
		if outFile != nil {
			outFile.Close()
		}
	}()
	var bufErr os.Error
	o.out, bufErr = bufio.NewWriterSize(outFile, 1024*1024)
	if bufErr != nil {
		fmt.Fprintln(os.Stderr, bufErr)
		return
	}

	fmt.Fprintln(o.out, "mtllib", path.Base(mtlFilename))

	o.outFile, outFile = outFile, nil
}

func (o *ObjGenerator) GetEnclosedJobsChan() chan *EnclosedChunkJob {
	return o.enclosedsChan
}

func (o *ObjGenerator) GetCompleteChan() chan bool {
	return o.completeChan
}

func (o *ObjGenerator) Close() {
	o.out.Flush()
	o.outFile.Close()
}

type WriteFacesJob struct {
	xPos, zPos, faceCount int
	b                     *MemoryWriter
	last                  bool
}

type MemoryWriter struct {
	buf []byte
}

func (m *MemoryWriter) Clean() {
	if m.buf != nil {
		m.buf = m.buf[:0]
	}
}

func (m *MemoryWriter) Write(p []byte) (n int, err os.Error) {
	m.buf = append(m.buf, p...)
	return len(p), nil
}
