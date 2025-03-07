package channel

import (
	"encoding/binary"
	"io"
)

func readHeader(reader io.Reader) (ack bool, err error) {
	buffer := make([]byte, 1)
	_, err = io.ReadFull(reader, buffer)
	if err != nil {
		return false, err
	}
	return buffer[0] == 1, nil
}

func readPayload(reader io.Reader, buffer *[readBufferSize]byte) (n int, err error) {
	payloadLengthBuffer := make([]byte, 2)
	_, err = io.ReadFull(reader, payloadLengthBuffer)
	if err != nil {
		return 0, err
	}
	payloadLength := binary.BigEndian.Uint16(payloadLengthBuffer)
	payload := buffer[:payloadLength]
	return io.ReadFull(reader, payload)
}

func readPacket(reader io.Reader, buffer *[readBufferSize]byte) (ack bool, n int, err error) {
	ack, err = readHeader(reader)
	if err != nil {
		return false, 0, err
	}
	if ack {
		return true, 0, nil
	}
	n, err = readPayload(reader, buffer)
	return false, n, err
}

func writeHeader(writer io.Writer, ack bool) error {
	var buffer byte
	if ack {
		buffer = 1
	}
	_, err := writer.Write([]byte{buffer})
	return err
}

func writeAckRequest(writer io.Writer) error {
	return writeHeader(writer, true)
}

func writePayload(writer io.Writer, payload []byte) error {
	payloadLength := make([]byte, 2)
	binary.BigEndian.PutUint16(payloadLength, uint16(len(payload)))
	_, err := writer.Write(payloadLength)
	if err != nil {
		return err
	}
	_, err = writer.Write(payload)
	if err != nil {
		return err
	}
	return nil
}

func writeData(writer io.Writer, payload []byte) error {
	err := writeHeader(writer, false)
	if err != nil {
		return err
	}
	return writePayload(writer, payload)
}

func writeAckResponse(writer io.Writer, offset uint64) error {
	buffer := make([]byte, 8)
	binary.BigEndian.PutUint64(buffer, offset)
	_, err := writer.Write(buffer)
	return err
}

func readAckResponse(reader io.Reader) (uint64, error) {
	buffer := make([]byte, 8)
	_, err := io.ReadFull(reader, buffer)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint64(buffer), nil
}
