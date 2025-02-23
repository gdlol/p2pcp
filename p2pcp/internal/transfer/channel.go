package transfer

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"
)

// Transfer data over unstable streams.
type Channel io.ReadWriteCloser

const DefaultPayloadSize = 8192

type channel struct {
	logger        *slog.Logger
	ctx           context.Context
	streams       chan ChannelStream
	payloadSize   int
	currentStream *ChannelStream
	readSeq       uint64
	writeSeq      uint64
	readClosed    bool
	writeClosed   bool
}

func (c *channel) getStream() (*ChannelStream, error) {
	if c.currentStream == nil {
		select {
		case <-c.ctx.Done():
			return nil, c.ctx.Err()
		case stream := <-c.streams:
			c.currentStream = &stream
		}
	}

	return c.currentStream, nil
}

func (c *channel) closeStream() {
	if c.currentStream != nil {
		(*c.currentStream).Close()
		c.currentStream = nil
	}
}

func (c *channel) writeAck(seq uint64) error {
	packet := newAckPacket(seq)
	stream, err := c.getStream()
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

func (c *channel) Read(p []byte) (n int, err error) {
	if len(p) < c.payloadSize {
		return 0, io.ErrShortBuffer
	}
	if c.readClosed {
		return 0, io.EOF
	}
	for c.ctx.Err() == nil {
		stream, err := c.getStream()
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

func (c *channel) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	if len(p) > c.payloadSize {
		return 0, fmt.Errorf("payload too large: %d > %d", len(p), c.payloadSize)
	}
	for c.ctx.Err() == nil {
		stream, err := c.getStream()
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

func (c *channel) Close() error {
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

	for ctx.Err() == nil {
		stream, err := c.getStream()
		if err != nil {
			return err
		}
		defer c.closeStream()

		if !c.writeClosed {
			// Write FIN
			packet := newPacket(c.writeSeq, nil)
			err := writePacket(stream, packet)
			if err != nil {
				c.logger.Debug("Error writing FIN packet.", "error", err)
				continue
			}
		}

		packet, err := readPacket(stream, buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			c.logger.Debug("Error reading ACK/FIN packet.", "error", err)
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
					"packet ID mismatch FIN: expected %d, received %d",
					c.writeSeq, packet.header.seq)
			}
			if len(packet.payload) != 0 {
				return fmt.Errorf("unexpected data packet during close")
			}
			c.readClosed = true
			err = c.writeAck(c.readSeq)
			if err != nil {
				continue
			}
		}

		if c.writeClosed && c.readClosed {
			return nil
		}
	}
	return ctx.Err()
}

func NewChannel(ctx context.Context, streams chan ChannelStream, payloadSize int) Channel {
	return &channel{
		ctx:         ctx,
		streams:     streams,
		payloadSize: payloadSize,
		logger:      slog.With("source", "p2pcp/transfer/channel", "payloadSize", payloadSize),
	}
}
