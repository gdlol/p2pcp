package auth

import (
	"crypto"
	"crypto/rand"
	_ "crypto/sha256"
	"crypto/subtle"
	"fmt"
	"io"
	"math/big"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/protocol"
)

const Protocol protocol.ID = "/p2pcp/auth/0.1.0"

const authenticationTimeout = 10 * time.Second

func ComputeHash(input []byte) []byte {
	hash := crypto.SHA256.New()
	_, err := hash.Write(input)
	if err != nil {
		panic(err)
	}
	return hash.Sum(nil)
}

// 4-digit PIN
func GetPin() (string, error) {
	digits := make([]string, 4)
	for i := range digits {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		digits[i] = n.String()
	}
	return strings.Join(digits, ""), nil
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
