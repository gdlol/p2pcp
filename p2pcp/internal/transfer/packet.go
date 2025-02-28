package transfer

import (
	"encoding/binary"
	"io"
)

type header struct {
	ack           bool
	seq           uint64
	payloadLength uint16
}

const headerSize = 1 + 8 + 2

type packet struct {
	header  header
	payload []byte
}

func readPacket(reader io.Reader, payload []byte) (*packet, error) {
	headerBuffer := make([]byte, headerSize)
	_, err := io.ReadFull(reader, headerBuffer)
	if err != nil {
		return nil, err
	}
	header := header{
		ack:           headerBuffer[0] == 1,
		seq:           binary.BigEndian.Uint64(headerBuffer[1:9]),
		payloadLength: binary.BigEndian.Uint16(headerBuffer[9:]),
	}
	if len(payload) < int(header.payloadLength) {
		return nil, io.ErrShortBuffer
	}
	payload = payload[:header.payloadLength]
	_, err = io.ReadFull(reader, payload)
	if err != nil {
		return nil, err
	}
	return &packet{header: header, payload: payload}, nil
}

func writePacket(writer io.Writer, packet *packet) error {
	headerBuffer := make([]byte, headerSize)
	if packet.header.ack {
		headerBuffer[0] = 1
	}
	binary.BigEndian.PutUint64(headerBuffer[1:], packet.header.seq)
	binary.BigEndian.PutUint16(headerBuffer[9:], packet.header.payloadLength)
	_, err := writer.Write(headerBuffer)
	if err != nil {
		return err
	}
	if len(packet.payload) > 0 {
		_, err = writer.Write(packet.payload)
		if err != nil {
			return err
		}
	}
	return nil
}

func newPacket(seq uint64, payload []byte) *packet {
	return &packet{
		header: header{
			seq:           seq,
			payloadLength: uint16(len(payload)),
		},
		payload: payload,
	}
}

func newAckPacket(seq uint64) *packet {
	return &packet{
		header: header{
			ack: true,
			seq: seq,
		},
	}
}
