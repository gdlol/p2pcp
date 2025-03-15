package node

import (
	"crypto/ed25519"
	"crypto/rand"
	"p2pcp/internal/auth"
	"testing"

	"github.com/libp2p/go-libp2p"
	libp2pCrypto "github.com/libp2p/go-libp2p/core/crypto"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetNodeID(t *testing.T) {
	privKey, _, err := libp2pCrypto.GenerateEd25519Key(rand.Reader)
	require.NoError(t, err)
	bytes, err := privKey.Raw()
	require.NoError(t, err)
	pubKey := ed25519.PrivateKey(bytes).Public().(ed25519.PublicKey)
	node := NewNode(t.Context(), true, libp2p.Identity(privKey))
	defer node.Close()
	assert.Equal(t, auth.ComputeHash(pubKey), node.ID().Bytes())
}

func TestNodeIDRandomArt(t *testing.T) {
	net := mocknet.New()
	defer net.Close()

	randomArts := make(map[string]bool)
	for range 1000 {
		psk := make([]byte, 32)
		_, err := rand.Read(psk)
		require.NoError(t, err)
		host, err := libp2p.New(libp2p.PrivateNetwork(psk))
		require.NoError(t, err)
		id := GetNodeID(host.ID())
		randomArt := auth.RandomArt(id.Bytes())
		randomArts[randomArt] = true
	}
	assert.Equal(t, 1000, len(randomArts))
}

func TestNewNode(t *testing.T) {
	node1 := NewNode(t.Context(), true)
	defer node1.Close()
	assert.Equal(t, true, node1.(*node).privateMode)
}
