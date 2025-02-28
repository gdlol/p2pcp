package node

import (
	"crypto/ed25519"
	"crypto/rand"
	"p2pcp/internal/auth"
	"testing"

	"github.com/libp2p/go-libp2p"
	libp2pCrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/stretchr/testify/assert"
)

func TestGetNodeID(t *testing.T) {
	privKey, _, err := libp2pCrypto.GenerateEd25519Key(rand.Reader)
	assert.NoError(t, err)
	bytes, err := privKey.Raw()
	assert.NoError(t, err)
	pubKey := ed25519.PrivateKey(bytes).Public().(ed25519.PublicKey)
	node, err := NewNode(t.Context(), true, libp2p.Identity(privKey))
	assert.NoError(t, err)
	defer node.Close()
	assert.Equal(t, auth.ComputeHash(pubKey), node.ID().Bytes())
}

func TestNewNode(t *testing.T) {
	node1, err := NewNode(t.Context(), true)
	assert.NoError(t, err)
	defer node1.Close()
	assert.Equal(t, true, node1.IsPrivateMode())
}
