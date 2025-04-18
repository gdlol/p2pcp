package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"fmt"
	"io"
	"math/big"
	"p2pcp/internal/errors"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/protocol"
	"golang.org/x/crypto/blake2b"
	"moul.io/drunken-bishop/drunkenbishop"
)

const Protocol protocol.ID = "/p2pcp/auth/1.0.0"

const authenticationTimeout = 10 * time.Second

const pinLength = 6

func ComputeHash(input []byte) []byte {
	hash := blake2b.Sum256(input)
	return hash[:]
}

// 6-digit PIN
func GetOneTimeSecret() string {
	digits := make([]string, pinLength)
	for i := range digits {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		errors.Unexpected(err, "GetOneTimeSecret: rand.Int")
		digits[i] = n.String()
	}
	return strings.Join(digits, "")
}

func GetStrongSecret() string {
	return rand.Text()
}

func RandomArt(bytes []byte) string {
	return drunkenbishop.FromBytes(bytes).String()
}

func HandleAuthenticate(stream io.ReadWriteCloser, secretHash []byte) (*bool, error) {
	timer := time.AfterFunc(authenticationTimeout, func() {
		stream.Close()
	})

	buffer := make([]byte, len(secretHash))
	_, err := io.ReadFull(stream, buffer)
	if !timer.Stop() {
		return nil, fmt.Errorf("authentication timed out")
	}
	defer stream.Close()
	if err != nil {
		return nil, err
	}
	result := subtle.ConstantTimeCompare(buffer, secretHash)
	_, err = stream.Write([]byte{byte(result)})
	success := result == 1
	return &success, err
}

func Authenticate(stream io.ReadWriteCloser, secretHash []byte) (bool, error) {
	defer stream.Close()
	_, err := stream.Write(secretHash)
	if err != nil {
		return false, err
	}
	buffer := make([]byte, 1)
	_, err = io.ReadFull(stream, buffer)
	if err != nil {
		return false, err
	}
	return buffer[0] == 1, nil
}
