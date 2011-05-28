package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
)

type ObjGenerator struct {
	enclosedsChan  chan *EnclosedChunkJob
	writeFacesChan chan *WriteFacesJob
	completeChan   chan bool

	freelist chan *MemoryWriter
	memoryWriterPool *MemoryWriterPool

	total int

	outFile *os.File
	out     *bufio.Writer
}

func (o *ObjGenerator) Start(outFilename string, total int, maxProcs int, boundary *BoundaryLocator) {
	o.enclosedsChan = make(chan *EnclosedChunkJob, maxProcs*2)
	o.writeFacesChan = make(chan *WriteFacesJob, maxProcs*2)
	o.completeChan = make(chan bool)

	o.memoryWriterPool = NewMemoryWriterPool(maxProcs*2, 128*1024)
	o.total = total

	for i := 0; i < maxProcs; i++ {
		go func() {
			var faces Faces
			faces.boundary = boundary
			for {
				var job = <-o.enclosedsChan

				var b = o.memoryWriterPool.GetWriter()
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

			o.memoryWriterPool.ReuseWriter(job.b)

			if job.last {
				o.completeChan <- true
			}
		}
	}()

	var mtlFilename = fmt.Sprintf("%s.mtl", outFilename[:len(outFilename)-len(filepath.Ext(outFilename))])
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

	fmt.Fprintln(o.out, "mtllib", filepath.Base(mtlFilename))

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

