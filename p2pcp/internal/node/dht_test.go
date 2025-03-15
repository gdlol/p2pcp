package node

import (
	"fmt"
	"testing"

	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetBootstrapPeers(t *testing.T) {
	peers := getBootstrapPeers(nil)
	assert.NotEmpty(t, peers)
}

func TestInvalidBootstrapPeers(t *testing.T) {
	bootstrapPeers := getBootstrapPeers([]string{"invalid"})
	assert.Empty(t, bootstrapPeers)
}

func TestValidBootstrapPeers(t *testing.T) {
	net := mocknet.New()
	defer net.Close()

	host, err := net.GenPeer()
	require.NoError(t, err)

	p2pAddr := fmt.Sprintf("/ip4/127.0.0.1/tcp/12345/p2p/%s", host.ID())
	bootstrapPeers := getBootstrapPeers([]string{p2pAddr, "invalid"})
	require.Len(t, bootstrapPeers, 1)
	assert.Equal(t, host.ID(), bootstrapPeers[0].ID)
	addr, err := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/12345")
	require.NoError(t, err)
	assert.Equal(t, bootstrapPeers[0].Addrs, []multiaddr.Multiaddr{addr})
}
