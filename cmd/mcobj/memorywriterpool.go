package main

type MemoryWriter struct {
	buf []byte
}

func (m *MemoryWriter) Clean() {
	if m.buf != nil {
		m.buf = m.buf[:0]
	}
}

func (m *MemoryWriter) Write(p []byte) (n int, err error) {
	m.buf = append(m.buf, p...)
	return len(p), nil
}

type MemoryWriterPool struct {
	freelist          chan *MemoryWriter
	initialBufferSize int
}

func NewMemoryWriterPool(capacity int, initialBufferSize int) *MemoryWriterPool {
	return &MemoryWriterPool{make(chan *MemoryWriter, capacity), initialBufferSize}
}

func (p *MemoryWriterPool) GetWriter() *MemoryWriter {
	var b *MemoryWriter
	select {
	case b = <-p.freelist:
		// Got a buffer
	default:
		b = &MemoryWriter{make([]byte, 0, p.initialBufferSize)}
	}
	return b
}

func (p *MemoryWriterPool) ReuseWriter(b *MemoryWriter) {
	b.Clean()
	select {
	case p.freelist <- b:
		// buffer added to free list
	default:
		// free list is full, discard the buffer
	}
}
