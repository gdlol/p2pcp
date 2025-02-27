package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"fmt"
	"io"
	"math/big"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/protocol"
	"golang.org/x/crypto/blake2b"
)

const Protocol protocol.ID = "/p2pcp/auth/0.1.0"

const authenticationTimeout = 10 * time.Second

func ComputeHash(input []byte) []byte {
	hash := blake2b.Sum256(input)
	return hash[:]
}

// 4-digit PIN
func GetOneTimeSecret() string {
	digits := make([]string, 4)
	for i := range digits {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			panic(err)
		}
		digits[i] = n.String()
	}
	return strings.Join(digits, "")
}

func GetStrongSecret() string {
	return rand.Text()
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
