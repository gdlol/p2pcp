package channel

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"time"
)

type channelReader struct {
	ctx           context.Context
	logger        *slog.Logger
	getStream     GetStream
	currentStream io.ReadWriteCloser
	offset        uint64
	readBuffer    *readBuffer
	readClosed    bool
	closed        bool
}

func (c *channelReader) getCurrentStream(ctx context.Context) (io.ReadWriteCloser, error) {
	if c.currentStream == nil {
		stream, err := c.getStream(ctx)
		if err != nil {
			return nil, err
		}
		c.currentStream = stream
	}
	return c.currentStream, nil
}

func (c *channelReader) closeStream() {
	if c.currentStream != nil {
		c.currentStream.Close()
		c.currentStream = nil
	}
}

func (c *channelReader) read(p *[readBufferSize]byte) (n int, err error) {
	if c.readClosed {
		return 0, io.EOF
	}
	for c.ctx.Err() == nil {
		stream, err := c.getCurrentStream(c.ctx)
		if err != nil {
			return 0, err
		}

		ack, payloadLength, err := readPacket(stream, p)
		if err != nil {
			c.logger.Debug("Error reading packet.", "error", err)
			c.closeStream()
			continue
		}

		if ack {
			err = writeAckResponse(stream, c.offset)
			if err != nil {
				c.logger.Debug("Error writing ACK.", "error", err)
				c.closeStream()
				continue
			}
		} else {
			if payloadLength == 0 { // FIN
				c.readClosed = true
				c.offset = math.MaxUint64
				return 0, io.EOF
			} else {
				c.offset += uint64(payloadLength)
				return payloadLength, nil
			}
		}
	}
	return 0, c.ctx.Err()
}

func (c *channelReader) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
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

func (c *channelReader) Close() error {
	if c.closed {
		return nil
	}
	defer func() {
		c.closed = true
	}()

	ctx, cancel := context.WithTimeout(c.ctx, 3*time.Second)
	defer cancel()
	defer c.closeStream()
	for ctx.Err() == nil {
		stream, err := c.getCurrentStream(ctx)
		if err != nil {
			return err
		}

		ack, payloadLength, err := readPacket(stream, c.readBuffer.buffer)
		if err != nil {
			if err == io.EOF && c.readClosed {
				return nil
			}
			fmt.Println("Error reading FIN.", "error", err)
			c.logger.Debug("Error reading FIN.", "error", err)
			c.closeStream()
			continue
		}

		if ack {
			err = writeAckResponse(stream, uint64(c.offset))
			if err != nil {
				c.logger.Debug("Error writing ACK.", "error", err)
				c.closeStream()
			}
		} else {
			if payloadLength > 0 {
				return fmt.Errorf("unexpected data packet during close")
			}
			c.readClosed = true
			c.offset = math.MaxUint64
		}
	}
	return ctx.Err()
}

func NewChannelReader(ctx context.Context, getStream GetStream) io.ReadCloser {
	return &channelReader{
		ctx:        ctx,
		logger:     slog.With("source", "transfer/channel"),
		getStream:  getStream,
		readBuffer: newReadBuffer(),
	}
}

var _ io.ReadCloser = (*channelReader)(nil)
