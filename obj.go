package main

import (
	"bufio"
	"fmt"
	"os"
	"path"
)

func GenerateObj(maxProcs int, outFilename string, pool ChunkPool, opener ChunkOpener, cx, cz int) {
	var total = 0

	var (
		enclosedsChan  = make(chan *EnclosedChunkJob, maxProcs*2)
		writeFacesChan = make(chan *WriteFacesJob, maxProcs*2)
		completeChan   = make(chan bool)

		freelist = make(chan *MemoryWriter, maxProcs*2)
	)

	for i := 0; i < maxProcs; i++ {
		go func() {
			var faces Faces
			for {
				var job = <-enclosedsChan

				var b *MemoryWriter
				select {
				case b = <-freelist:
					// Got a buffer
				default:
					b = &MemoryWriter{make([]byte, 0, 128*1024)}
				}

				var faceCount = faces.ProcessChunk(job.enclosed, b)
				fmt.Fprintln(b)

				writeFacesChan <- &WriteFacesJob{job.enclosed.xPos, job.enclosed.zPos, faceCount, b, job.last}
			}
		}()
	}

	go func() {
		var chunkCount = 0
		var size = 0
		for {
			var job = <-writeFacesChan
			chunkCount++
			out.Write(job.b.buf)
			out.Flush()

			size += len(job.b.buf)
			fmt.Printf("%4v/%-4v (%3v,%3v) Faces: %4d Size: %4.1fMB\n", chunkCount, total, job.xPos, job.zPos, job.faceCount, float64(size)/1024/1024)

			job.b.Clean()
			select {
			case freelist <- job.b:
				// buffer added to free list
			default:
				// free list is full, discard the buffer
			}

			if job.last {
				completeChan <- true
			}
		}
	}()

	var mtlFilename = fmt.Sprintf("%s.mtl", outFilename[:len(outFilename)-len(path.Ext(outFilename))])
	var mtlErr = writeMtlFile(mtlFilename)
	if mtlErr != nil {
		fmt.Fprintln(os.Stderr, mtlErr)
		return
	}

	var outFile, outErr = os.Open(outFilename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
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

	fmt.Fprintln(out, "mtllib", path.Base(mtlFilename))

	total = pool.Remaining()

	if WalkEnclosedChunks(pool, opener, cx, cz, enclosedsChan) {
		<-completeChan
	}
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
