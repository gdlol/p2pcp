package channel

import (
	"context"
	"fmt"
	"io"
	"math"
	"p2pcp/internal/errors"
)

type GetStream func(ctx context.Context) (io.ReadWriteCloser, error)

const readBufferSize = math.MaxUint16
const writeBufferSize = 1024 * 1024 * 4

type readBuffer struct {
	buffer *[readBufferSize]byte
	offset int
	length int
}

func newReadBuffer() *readBuffer {
	return &readBuffer{
		buffer: &[readBufferSize]byte{},
		offset: 0,
		length: 0,
	}
}

func (r *readBuffer) read(p []byte) int {
	n := copy(p, r.buffer[r.offset:r.length])
	r.offset += n
	return n
}

func (r *readBuffer) isEmpty() bool {
	return r.offset >= r.length
}

func (r *readBuffer) reset() {
	r.offset = 0
	r.length = 0
}

type writeBuffer struct {
	buffer    *[writeBufferSize]byte
	length    uint32
	committed uint64
}

func newWriteBuffer() *writeBuffer {
	return &writeBuffer{
		buffer:    &[writeBufferSize]byte{},
		length:    0,
		committed: 0,
	}
}

func (w *writeBuffer) data() []byte {
	return w.buffer[:w.length]
}

func writePrepared(w *writeBuffer, p []byte) {
	copy(w.buffer[w.length:], p)
	w.length += uint32(len(p))
}

func (w *writeBuffer) prepareWrite(p []byte) (write func(w *writeBuffer, p []byte), ok bool) {
	size := uint32(len(p))
	if w.length+size > writeBufferSize {
		return nil, false
	} else {
		return writePrepared, true
	}
}

func (w *writeBuffer) commit(totalOffset uint64) {
	errors.Assert(totalOffset >= w.committed, fmt.Sprintf("invalid offset: %d", totalOffset))
	size := totalOffset - w.committed
	errors.Assert(size <= uint64(w.length), fmt.Sprintf("invalid offset: %d", totalOffset))
	copy(w.buffer[:], w.buffer[size:w.length])
	w.length -= uint32(size)
	w.committed = totalOffset
}
