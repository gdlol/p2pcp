package channel

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadHeader(t *testing.T) {
	reader := strings.NewReader("")
	ack, err := readHeader(reader)
	assert.Error(t, err)
	assert.Equal(t, err, io.EOF)
	assert.False(t, ack)
}

func TestReadPayload(t *testing.T) {
	reader := strings.NewReader("")
	buffer := [readBufferSize]byte{}
	n, err := readPayload(reader, &buffer)
	assert.Error(t, err)
	assert.Equal(t, err, io.EOF)
	assert.Zero(t, n)

	reader = strings.NewReader(string([]byte{0x00}))
	n, err = readPayload(reader, &buffer)
	assert.Error(t, err)
	assert.Equal(t, io.ErrUnexpectedEOF, err)
	assert.Zero(t, n)
}

func TestReadPacket(t *testing.T) {
	reader := strings.NewReader("")
	buffer := [readBufferSize]byte{}
	ack, n, err := readPacket(reader, &buffer)
	assert.Error(t, err)
	assert.Equal(t, err, io.EOF)
	assert.False(t, ack)
	assert.Zero(t, n)
}

func TestWriteHeader(t *testing.T) {
	reader, writer := io.Pipe()
	defer reader.Close()
	writer.Close()
	err := writeHeader(writer, false)
	assert.Error(t, err)
	assert.Equal(t, err, io.ErrClosedPipe)
}

func TestWritePayload(t *testing.T) {
	reader, writer := io.Pipe()
	defer reader.Close()
	writer.Close()
	err := writePayload(writer, []byte{})
	assert.Error(t, err)
	assert.Equal(t, err, io.ErrClosedPipe)
}

func TestWriteData(t *testing.T) {
	reader, writer := io.Pipe()
	defer reader.Close()
	writer.Close()
	err := writeData(writer, []byte{})
	assert.Error(t, err)
	assert.Equal(t, err, io.ErrClosedPipe)
}

func TestWriteAckResponse(t *testing.T) {
	reader, writer := io.Pipe()
	defer reader.Close()
	writer.Close()
	err := writeAckResponse(writer, 0)
	assert.Error(t, err)
	assert.Equal(t, err, io.ErrClosedPipe)
}

func TestReadAckResponse(t *testing.T) {
	reader := strings.NewReader("")
	offset, err := readAckResponse(reader)
	assert.Error(t, err)
	assert.Equal(t, err, io.EOF)
	assert.Zero(t, offset)

	reader = strings.NewReader(string([]byte{0x00}))
	offset, err = readAckResponse(reader)
	assert.Error(t, err)
	assert.Equal(t, io.ErrUnexpectedEOF, err)
	assert.Zero(t, offset)
}
