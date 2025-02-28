package auth

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
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
		assert.Len(t, secret, 4)
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
		assert.GreaterOrEqual(t, len(secret), 26)
		secrets[secret] = true
		for _, char := range secret {
			chars[string(char)] = true
		}
	}
	assert.Len(t, secrets, 1000)
	assert.Len(t, chars, 32)
}
