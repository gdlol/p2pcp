package auth

import (
	"crypto"
	"crypto/rand"
	"crypto/subtle"
	"math/big"
	"strings"
)

func ComputeHash(input []byte) []byte {
	hash := crypto.SHA256.New()
	_, err := hash.Write(input)
	if err != nil {
		panic(err)
	}
	return hash.Sum(nil)
}

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

func VerifyHash(input []byte, hash []byte) bool {
	return subtle.ConstantTimeCompare(input, hash) == 1
}
