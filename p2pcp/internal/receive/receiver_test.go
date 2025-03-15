package receive

import (
	"context"
	"crypto/rand"
	"fmt"
	"p2pcp/internal/auth"
	"p2pcp/internal/node"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSenderSuffixCheck(t *testing.T) {
	t.Parallel()

	psk := make([]byte, 32)
	_, err := rand.Read(psk)
	require.NoError(t, err)

	node1, err := libp2p.New(libp2p.PrivateNetwork(psk))
	require.NoError(t, err)
	defer node1.Close()
	node2, err := libp2p.New(libp2p.PrivateNetwork(psk))
	require.NoError(t, err)
	defer node2.Close()
	id := node.GetNodeID(node1.ID()).String()
	assert.True(t, isValidPeer(node1.Peerstore().PeerInfo(node1.ID()), id))
	assert.False(t, isValidPeer(node2.Peerstore().PeerInfo(node2.ID()), id))
}

type mockNode struct {
	host           host.Host
	peers          chan peer.AddrInfo
	findPeerCalled int
}

func (m *mockNode) AdvertiseLAN(ctx context.Context, topic string) error { return nil }

func (m *mockNode) AdvertiseWAN(ctx context.Context, topic string) error { return nil }

func (m *mockNode) Close() { m.host.Close() }

// FindPeers implements node.Node.
func (m *mockNode) FindPeers(ctx context.Context, topic string) (<-chan peer.AddrInfo, error) {
	m.findPeerCalled += 1
	if m.peers == nil {
		return nil, fmt.Errorf("test")
	} else {
		return m.peers, nil
	}
}

func (m *mockNode) GetHost() host.Host { return m.host }

func (m *mockNode) ID() node.NodeID { return node.GetNodeID(m.host.ID()) }

func (m *mockNode) RegisterErrorHandler(peerID peer.ID, handler func(string)) {}

func (m *mockNode) SendError(ctx context.Context, peerID peer.ID, errStr string) {}

func (m *mockNode) StartMdns() {}

var _ node.Node = (*mockNode)(nil)

func TestFindPeers(t *testing.T) {
	t.Parallel()

	psk := make([]byte, 32)
	_, err := rand.Read(psk)
	require.NoError(t, err)

	host, err := libp2p.New(libp2p.PrivateNetwork(psk))
	require.NoError(t, err)
	defer host.Close()
	n := &mockNode{host: host}
	defer n.Close()
	receiver := &receiver{node: n}
	id := node.GetNodeID(host.ID()).String()

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		peer, err := receiver.FindPeer(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, host.ID(), peer)
		done <- struct{}{}
	}()

	time.Sleep(2 * time.Second)
	select {
	case <-done:
		t.Fatal("unexpected completion")
	default:
	}
	peers := make(chan peer.AddrInfo, 1)
	n.peers = peers
	peers <- peer.AddrInfo{ID: host.ID()}

	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("unexpected timeout")
	}

	assert.Greater(t, n.findPeerCalled, 1)
	assert.Less(t, n.findPeerCalled, 5)
}

func TestFindPeersTimeout(t *testing.T) {
	t.Parallel()

	psk := make([]byte, 32)
	_, err := rand.Read(psk)
	require.NoError(t, err)

	host, err := libp2p.New(libp2p.PrivateNetwork(psk))
	require.NoError(t, err)
	defer host.Close()
	n := &mockNode{host: host}
	defer n.Close()
	receiver := &receiver{node: n}
	id := node.GetNodeID(host.ID()).String()

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()

	peer, err := receiver.FindPeer(ctx, id)
	require.Error(t, err)
	assert.Empty(t, peer)
	assert.Equal(t, context.DeadlineExceeded, err)
	assert.Equal(t, context.DeadlineExceeded, ctx.Err())
}

func TestReceiveTimeout(t *testing.T) {
	t.Parallel()

	psk := make([]byte, 32)
	_, err := rand.Read(psk)
	require.NoError(t, err)

	host1, err := libp2p.New(libp2p.PrivateNetwork(psk))
	require.NoError(t, err)
	defer host1.Close()
	host2, err := libp2p.New(libp2p.PrivateNetwork(psk))
	require.NoError(t, err)
	defer host2.Close()
	n := &mockNode{host: host1}
	defer n.Close()
	receiver := &receiver{node: n}

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()

	err = receiver.Receive(ctx, host2.ID(), nil, "")
	require.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
	assert.Equal(t, context.DeadlineExceeded, ctx.Err())
}

func TestConnectTimeout(t *testing.T) {
	t.Parallel()

	net := mocknet.New()
	defer net.Close()

	h1, err := net.GenPeer()
	require.NoError(t, err)
	h2, err := net.GenPeer()
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()

	err = connectToSender(ctx, h1, h2.ID())
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
	assert.Error(t, ctx.Err())
	assert.Equal(t, context.DeadlineExceeded, ctx.Err())
}

func TestConnectRetry(t *testing.T) {
	t.Parallel()

	net := mocknet.New()
	defer net.Close()

	h1, err := net.GenPeer()
	require.NoError(t, err)
	h2, err := net.GenPeer()
	require.NoError(t, err)

	connected := false
	connect := make(chan error)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	go func() {
		err = connectToSender(t.Context(), h1, h2.ID())
		connected = true
		connect <- err
	}()

	select {
	case err := <-connect:
		t.Fatalf("unexpected connection: %v", err)
	case <-ctx.Done():
	}

	assert.False(t, connected)
	assert.Error(t, ctx.Err())
	assert.Equal(t, context.DeadlineExceeded, ctx.Err())
	assert.NoError(t, err)

	err = net.LinkAll()
	require.NoError(t, err)

	err = <-connect
	require.NoError(t, err)
	assert.True(t, connected)
}

func TestGetStreamTimeout(t *testing.T) {
	t.Parallel()

	net := mocknet.New()
	defer net.Close()

	h1, err := net.GenPeer()
	require.NoError(t, err)
	h2, err := net.GenPeer()
	require.NoError(t, err)
	_, err = net.LinkPeers(h1.ID(), h2.ID())
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()

	_, err = getStream(ctx, h1, h2.ID(), protocol.TestingID)
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
	assert.Error(t, ctx.Err())
	assert.Equal(t, context.DeadlineExceeded, ctx.Err())
}

func TestGetStreamRetry(t *testing.T) {
	t.Parallel()

	net := mocknet.New()
	defer net.Close()

	h1, err := net.GenPeer()
	require.NoError(t, err)
	h2, err := net.GenPeer()
	require.NoError(t, err)
	_, err = net.LinkPeers(h1.ID(), h2.ID())
	require.NoError(t, err)

	newStream := false
	connect := make(chan error)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	go func() {
		stream, err := getStream(t.Context(), h1, h2.ID(), protocol.TestingID)
		defer func() {
			if err != nil {
				stream.Close()
			}
		}()
		newStream = true
		connect <- err
	}()

	select {
	case err := <-connect:
		t.Fatalf("unexpected connection: %v", err)
	case <-ctx.Done():
	}

	assert.False(t, newStream)
	assert.Error(t, ctx.Err())
	assert.Equal(t, context.DeadlineExceeded, ctx.Err())
	assert.NoError(t, err)

	h2.SetStreamHandler(protocol.TestingID, func(stream network.Stream) {
		stream.Close()
	})

	err = <-connect
	assert.NoError(t, err)
	assert.True(t, newStream)
}

func TestAuthenticateTimeout(t *testing.T) {
	t.Parallel()

	net := mocknet.New()
	defer net.Close()

	h1, err := net.GenPeer()
	require.NoError(t, err)
	h2, err := net.GenPeer()
	require.NoError(t, err)
	_, err = net.LinkPeers(h1.ID(), h2.ID())
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()

	err = authenticate(ctx, h1, h2.ID(), []byte("test"))
	assert.Error(t, err)
	assert.Error(t, ctx.Err())
	assert.Equal(t, context.DeadlineExceeded, ctx.Err())
}

func TestAuthenticateDisconnect(t *testing.T) {
	t.Parallel()

	net := mocknet.New()
	defer net.Close()

	h1, err := net.GenPeer()
	require.NoError(t, err)
	h2, err := net.GenPeer()
	require.NoError(t, err)
	_, err = net.LinkPeers(h1.ID(), h2.ID())
	require.NoError(t, err)

	h2.SetStreamHandler(auth.Protocol, func(stream network.Stream) {
		stream.Close()
	})

	err = authenticate(t.Context(), h1, h2.ID(), []byte("test"))
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "authentication failed")
}
