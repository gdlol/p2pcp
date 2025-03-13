package node

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/routing"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDhtRoutingInitialization(t *testing.T) {
	t.Parallel()

	net := mocknet.New()
	defer net.Close()

	h1, err := net.GenPeer()
	require.NoError(t, err)
	h2, err := net.GenPeer()
	require.NoError(t, err)

	routing := &dhtRouting{
		host: h1,
	}
	assert.Nil(t, routing.dht)

	addrInfo, err := routing.FindPeer(t.Context(), h2.ID())
	assert.Nil(t, err)
	assert.Equal(t, h2.ID(), addrInfo.ID)
	assert.Empty(t, addrInfo.Addrs)
}

type mockDht struct {
	addrInfos map[peer.ID]peer.AddrInfo
}

func (m *mockDht) FindPeer(ctx context.Context, id peer.ID) (peer.AddrInfo, error) {
	addrInfo, ok := m.addrInfos[id]
	if ok {
		return addrInfo, nil
	} else {
		return peer.AddrInfo{ID: id}, routing.ErrNotFound
	}
}

var _ routing.PeerRouting = (*mockDht)(nil)

func TestDhtRoutingSkipUntaggedPeer(t *testing.T) {
	t.Parallel()

	net := mocknet.New()
	defer net.Close()

	h1, err := net.GenPeer()
	require.NoError(t, err)
	h2, err := net.GenPeer()
	require.NoError(t, err)

	r := &dhtRouting{
		host: h1,
		dht: &mockDht{
			addrInfos: map[peer.ID]peer.AddrInfo{},
		},
	}

	addrInfo, err := r.FindPeer(t.Context(), h2.ID())
	assert.Error(t, err)
	assert.Equal(t, peerstore.ErrNotFound, err)
	assert.Equal(t, peer.AddrInfo{ID: h2.ID()}, addrInfo)
}

func TestDhtRoutingTimeout(t *testing.T) {
	t.Parallel()

	net := mocknet.New()
	defer net.Close()

	h1, err := net.GenPeer()
	require.NoError(t, err)
	h2, err := net.GenPeer()
	require.NoError(t, err)

	routing := &dhtRouting{
		host: h1,
		dht: &mockDht{
			addrInfos: map[peer.ID]peer.AddrInfo{},
		},
	}

	h1.Peerstore().Put(h2.ID(), DhtRoutingTag, struct{}{})

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()

	addrInfo, err := routing.FindPeer(ctx, h2.ID())
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
	assert.Equal(t, err, ctx.Err())
	assert.Equal(t, peer.AddrInfo{ID: h2.ID()}, addrInfo)
}

func TestDhtRoutingTaggedPeer(t *testing.T) {
	t.Parallel()

	net := mocknet.New()
	defer net.Close()

	h1, err := net.GenPeer()
	require.NoError(t, err)
	h2, err := net.GenPeer()
	require.NoError(t, err)

	routing := &dhtRouting{
		host: h1,
		dht: &mockDht{
			addrInfos: map[peer.ID]peer.AddrInfo{
				h2.ID(): {ID: h2.ID(), Addrs: h2.Addrs()},
			},
		},
	}
	require.NotEmpty(t, h2.Addrs())

	h1.Peerstore().Put(h2.ID(), DhtRoutingTag, struct{}{})

	addrInfo, err := routing.FindPeer(t.Context(), h2.ID())
	assert.NoError(t, err)
	assert.Equal(t, peer.AddrInfo{ID: h2.ID(), Addrs: h2.Addrs()}, addrInfo)
}
