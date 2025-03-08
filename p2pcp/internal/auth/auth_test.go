package auth

import (
	"encoding/hex"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeHash(t *testing.T) {
	input1 := []byte("test1")
	input2 := []byte("test2")
	hash1 := ComputeHash(input1)
	hash2 := ComputeHash(input2)
	assert.Len(t, hash1, 32)
	assert.Len(t, hash2, 32)
	assert.Equal(t, hex.EncodeToString(hash1), "e56837ccd7a38d6795d30ac2ab63eccc8c2b571289a62dc406bb62767e88b06c")
	assert.Equal(t, hex.EncodeToString(hash2), "62a8b48c4e90c81ed98327b43123849a98997e2aafd19fad7eba3a714f245a65")
}

func TestGetOneTimeSecret(t *testing.T) {
	secrets := make(map[string]bool)
	chars := make(map[string]bool)
	for range 1000 {
		secret := GetOneTimeSecret()
		require.Len(t, secret, 6)
		secrets[secret] = true
		for _, char := range secret {
			chars[string(char)] = true
		}
	}
	assert.Len(t, chars, 10)
	digits := "0123456789"
	for c := range chars {
		assert.Contains(t, digits, c)
	}
}

func TestGetStrongSecret(t *testing.T) {
	secrets := make(map[string]bool)
	chars := make(map[string]bool)
	for range 1000 {
		secret := GetStrongSecret()
		require.GreaterOrEqual(t, len(secret), 26)
		secrets[secret] = true
		for _, char := range secret {
			chars[string(char)] = true
		}
	}
	assert.Len(t, secrets, 1000)
	assert.Len(t, chars, 32)
}

type testStream struct {
	readClosed  bool
	writeClosed bool
}

func (s *testStream) Read(p []byte) (n int, err error) {
	for !s.readClosed {
		time.Sleep(100 * time.Millisecond)
	}
	return 0, io.EOF
}

func (s *testStream) Write(p []byte) (n int, err error) {
	if s.writeClosed {
		return 0, io.ErrClosedPipe
	} else {
		return len(p), nil
	}
}

func (s *testStream) Close() error {
	s.readClosed = true
	s.writeClosed = true
	return nil
}

func TestHandleAuthenticate_Timeout(t *testing.T) {
	stream := &testStream{}
	secretHash := ComputeHash([]byte("test"))
	timer := time.AfterFunc(authenticationTimeout, func() {})

	success, err := HandleAuthenticate(stream, secretHash)
	assert.Nil(t, success)
	assert.Error(t, err)
	assert.True(t, stream.readClosed)
	assert.True(t, stream.writeClosed)
	assert.False(t, timer.Stop())
}

func TestHandleAuthenticate_ReadError(t *testing.T) {
	stream := &testStream{}
	secretHash := ComputeHash([]byte("test"))
	stream.Close()

	timer := time.AfterFunc(authenticationTimeout, func() {})
	success, err := HandleAuthenticate(stream, secretHash)
	assert.Nil(t, success)
	assert.Error(t, err)
	assert.Equal(t, io.EOF, err)
	assert.True(t, timer.Stop())
	assert.True(t, stream.readClosed)
	assert.True(t, stream.writeClosed)
}

func TestAuthenticate_ErrorWrite(t *testing.T) {
	stream := &testStream{}
	stream.writeClosed = true
	secretHash := ComputeHash([]byte("test"))
	success, err := Authenticate(stream, secretHash)
	assert.False(t, success)
	assert.Error(t, err)
	assert.Equal(t, io.ErrClosedPipe, err)
	assert.True(t, stream.readClosed)
	assert.True(t, stream.writeClosed)
}

func TestAuthenticate_ErrorRead(t *testing.T) {
	stream := &testStream{}
	stream.readClosed = true
	secretHash := ComputeHash([]byte("test"))
	success, err := Authenticate(stream, secretHash)
	assert.False(t, success)
	assert.Error(t, err)
	assert.Equal(t, io.EOF, err)
	assert.True(t, stream.readClosed)
	assert.True(t, stream.writeClosed)
}
