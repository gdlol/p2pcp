package transfer

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/libp2p/go-libp2p/core/protocol"
)

const Protocol protocol.ID = "/p2pcp/transfer/0.1.0"

// Transfer data over unstable streams.
type Channel io.ReadWriteCloser

const DefaultPayloadSize = 8192

// Buffer large enough to hold payload returned by readPacket.
type readBuffer struct {
	buffer []byte
	offset int
	length int
}

func newReadBuffer(size int) *readBuffer {
	return &readBuffer{
		buffer: make([]byte, size),
		offset: 0,
		length: 0,
	}
}

func (r *readBuffer) read(p []byte) int {
	buffer := r.buffer[r.offset:r.length]
	n := copy(p, buffer)
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

type GetStream func(ctx context.Context) (io.ReadWriteCloser, error)

type channel struct {
	ctx           context.Context
	logger        *slog.Logger
	getStream     GetStream
	payloadSize   int
	currentStream io.ReadWriteCloser
	readSeq       uint64
	writeSeq      uint64
	readBuffer    *readBuffer
	readClosed    bool
	writeClosed   bool
}

func (c *channel) getCurrentStream(ctx context.Context) (io.ReadWriteCloser, error) {
	if c.currentStream == nil {
		stream, err := c.getStream(ctx)
		if err != nil {
			return nil, err
		}
		c.currentStream = stream
	}
	return c.currentStream, nil
}

func (c *channel) closeStream() {
	if c.currentStream != nil {
		c.currentStream.Close()
		c.currentStream = nil
	}
}

func (c *channel) writeAck(seq uint64) error {
	packet := newAckPacket(seq)
	stream, err := c.getCurrentStream(c.ctx)
	if err != nil {
		return err
	}
	err = writePacket(stream, packet)
	if err != nil {
		c.logger.Debug("Error writing ACK packet.", "error", err)
		c.closeStream()
	}
	return err
}

func (c *channel) read(p []byte) (n int, err error) {
	if c.readClosed {
		return 0, io.EOF
	}
	for c.ctx.Err() == nil {
		stream, err := c.getCurrentStream(c.ctx)
		if err != nil {
			return 0, err
		}

		// Read ACK.
		packet, err := readPacket(stream, p)
		if err != nil {
			c.logger.Debug("Error reading packet.", "error", err)
			c.closeStream()
			continue
		}

		if packet.header.ack {
			return 0, fmt.Errorf("unexpected ACK packet")
		}
		if packet.header.seq != c.readSeq {
			if packet.header.seq+1 == c.readSeq { // Previous ACK was lost.
				c.logger.Debug("Resending ACK.", "seq", c.readSeq)
				c.writeAck(packet.header.seq)
				continue
			} else {
				return 0, fmt.Errorf(
					"packet ID mismatch: expected %d, received %d",
					c.readSeq, packet.header.seq)
			}
		}

		if packet.header.payloadLength > uint16(len(p)) {
			return 0, io.ErrShortBuffer
		}

		err = c.writeAck(c.readSeq)
		if err != nil {
			continue
		}

		c.readSeq++
		if packet.header.payloadLength == 0 { // FIN
			c.readClosed = true
			return 0, io.EOF
		} else {
			return int(packet.header.payloadLength), nil
		}
	}
	return 0, c.ctx.Err()
}

func (c *channel) Read(p []byte) (n int, err error) {
	if c.readBuffer.isEmpty() {
		c.readBuffer.reset()
		n, err := c.read(c.readBuffer.buffer)
		if err != nil {
			return n, err
		}
		c.readBuffer.length = n
	}
	return c.readBuffer.read(p), nil
}

func (c *channel) write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	for c.ctx.Err() == nil {
		stream, err := c.getCurrentStream(c.ctx)
		if err != nil {
			return 0, err
		}

		packet := newPacket(c.writeSeq, p)

		err = writePacket(stream, packet)
		if err != nil {
			c.logger.Debug("Error writing packet.", "error", err)
			c.closeStream()
			continue
		}

		// Read ACK.
		packet, err = readPacket(stream, nil)
		if err != nil {
			c.logger.Debug("Error reading ACK packet.", "error", err)
			c.closeStream()
			continue
		}
		if !packet.header.ack {
			return 0, fmt.Errorf("unexpected data packet")
		}
		if packet.header.seq != c.writeSeq {
			return 0, fmt.Errorf(
				"packet ID mismatch: expected %d, received %d",
				c.writeSeq, packet.header.seq)
		}

		c.writeSeq++
		return len(p), nil
	}
	return 0, c.ctx.Err()
}

func (c *channel) Write(p []byte) (n int, err error) {
	n = 0
	for len(p) > 0 {
		buffer := p
		if len(buffer) > c.payloadSize {
			buffer = buffer[:c.payloadSize]
		}
		m, err := c.write(buffer)
		if err != nil {
			return n, err
		}
		n += m
		p = p[m:]
	}
	return n, nil
}

func (c *channel) close() error {
	if c.writeClosed && c.readClosed {
		return nil
	}
	defer func() {
		c.writeClosed = true
		c.readClosed = true
	}()

	ctx, cancel := context.WithTimeout(c.ctx, 3*time.Second)
	defer cancel()
	buffer := make([]byte, c.payloadSize)

	defer c.closeStream()
	for ctx.Err() == nil {
		stream, err := c.getCurrentStream(ctx)
		if err != nil {
			return err
		}

		if !c.writeClosed {
			// Write FIN
			packet := newPacket(c.writeSeq, nil)
			err := writePacket(stream, packet)
			if err != nil {
				c.logger.Debug("Error writing FIN packet.", "error", err)
				c.closeStream()
				continue
			}
		}

		packet, err := readPacket(stream, buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			c.logger.Debug("Error reading ACK/FIN packet.", "error", err)
			c.closeStream()
			continue
		}

		if packet.header.ack {
			if packet.header.seq != c.writeSeq {
				return fmt.Errorf(
					"packet ID for FIN ACK mismatch: expected %d, received %d",
					c.writeSeq, packet.header.seq)
			}
			c.writeClosed = true
		} else {
			if packet.header.seq != c.readSeq {
				return fmt.Errorf(
					"FIN packet ID mismatch: expected %d, received %d",
					c.writeSeq, packet.header.seq)
			}
			if len(packet.payload) != 0 {
				return fmt.Errorf("unexpected data packet during close")
			}
			c.readClosed = true
			err = c.writeAck(c.readSeq)
			if err != nil {
				c.closeStream()
				continue
			}
		}

		if c.writeClosed && c.readClosed {
			return nil
		}
	}
	return ctx.Err()
}

func (c *channel) Close() error {
	err := c.close()
	if err != nil {
		c.logger.Debug("Error closing channel.", "error", err)
	}
	return err
}

func NewChannel(ctx context.Context, getStream GetStream, payloadSize int) Channel {
	return &channel{
		ctx:         ctx,
		logger:      slog.With("source", "p2pcp/transfer/channel", "payloadSize", payloadSize),
		getStream:   getStream,
		payloadSize: payloadSize,
		readBuffer:  newReadBuffer(payloadSize),
	}
}
