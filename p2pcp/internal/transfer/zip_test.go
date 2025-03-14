package transfer

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

/**
 * For each test case, send the sendPath under testDir to a tmp directory,
 * and compare it with the expectedPath under testDir.
 */
func TestZipReadWrite(t *testing.T) {
	testReadWrite(t, ReadZip, WriteZip)
}

func TestReadEmptyZip(t *testing.T) {
	reader := strings.NewReader("")
	err := ReadZip(reader, "")
	assert.Error(t, err)
	assert.Equal(t, io.EOF, err)

	reader = strings.NewReader(string([]byte{0x12}))
	err = ReadZip(reader, "")
	assert.Error(t, err)
	assert.Equal(t, io.ErrUnexpectedEOF, err)
}
