package channel

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"time"
)

const payloadSize = 8192

type ChannelWriter interface {
	io.WriteCloser
	Flush(end bool) error
}

type channelWriter struct {
	ctx           context.Context
	logger        *slog.Logger
	getStream     GetStream
	currentStream io.ReadWriteCloser
	writeBuffer   *writeBuffer
	closed        bool
}

func (c *channelWriter) getCurrentStream(ctx context.Context) (stream io.ReadWriteCloser, new bool, err error) {
	if c.currentStream == nil {
		stream, err := c.getStream(ctx)
		if err != nil {
			return nil, false, err
		}
		c.currentStream = stream
		new = true
	}
	return c.currentStream, new, nil
}

func (c *channelWriter) closeStream() {
	if c.currentStream != nil {
		c.currentStream.Close()
		c.currentStream = nil
	}
}

func syncOffset(stream io.ReadWriter) (offset uint64, err error) {
	err = writeAckRequest(stream)
	if err != nil {
		return 0, err
	}
	return readAckResponse(stream)
}

func (c *channelWriter) Flush(end bool) error {
	if c.writeBuffer.length == 0 {
		return nil
	}
	for c.ctx.Err() == nil {
		stream, _, err := c.getCurrentStream(c.ctx)
		if err != nil {
			return err
		}

		offset, err := syncOffset(stream)
		if err != nil {
			c.logger.Debug("Error syncing offset.", "error", err)
			c.closeStream()
			continue
		}
		err = c.writeBuffer.commit(offset)
		if err != nil {
			return fmt.Errorf("error committing write buffer: %w", err)
		}
		data := c.writeBuffer.data()
		if len(data) == 0 {
			return nil
		}
		err = writeData(stream, data)
		if err != nil {
			c.logger.Debug("Error flushing data.", "error", err)
			c.closeStream()
			continue
		}
		if end {
			continue // ensure buffer empty
		}
		return nil
	}
	return c.ctx.Err()
}

func (c *channelWriter) write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	for c.ctx.Err() == nil {
		stream, new, err := c.getCurrentStream(c.ctx)
		if err != nil {
			return 0, err
		}

		writeCache, ok := c.writeBuffer.prepareWrite(p)
		if new || !ok {
			if err := c.Flush(false); err != nil {
				return 0, err
			}
			continue
		}

		err = writeData(stream, p)
		if err != nil {
			c.logger.Debug("Error writing data.", "error", err)
			c.closeStream()
			continue
		}
		writeCache()
		return len(p), nil
	}
	return 0, c.ctx.Err()
}

func (c *channelWriter) Write(p []byte) (n int, err error) {
	n = 0
	for len(p) > 0 {
		buffer := p
		if len(buffer) > payloadSize {
			buffer = buffer[:payloadSize]
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

func (c *channelWriter) Close() error {
	if c.closed {
		return nil
	}
	defer func() {
		c.closed = true
	}()

	err := c.Flush(true)
	if err != nil {
		return fmt.Errorf("error flushing write buffer: %w", err)
	}

	ctx, cancel := context.WithTimeout(c.ctx, 3*time.Second)
	defer cancel()
	defer c.closeStream()
	for c.ctx.Err() == nil {
		stream, _, err := c.getCurrentStream(ctx)
		if err != nil {
			return err
		}

		err = writeData(stream, nil) // FIN
		if err != nil {
			c.logger.Debug("Error writing FIN.", "error", err)
			c.closeStream()
			continue
		}
		offset, err := syncOffset(stream)
		if err != nil {
			c.logger.Debug("Error reading ACK.", "error", err)
			c.closeStream()
			continue
		}
		if offset == math.MaxUint64 {
			return nil
		}
	}
	return c.ctx.Err()
}

func NewChannelWriter(ctx context.Context, getStream GetStream) ChannelWriter {
	return &channelWriter{
		ctx:         ctx,
		logger:      slog.With("source", "transfer/channel"),
		getStream:   getStream,
		writeBuffer: newWriteBuffer(),
	}
}

var _ io.WriteCloser = &channelWriter{}
