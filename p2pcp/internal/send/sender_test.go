package send

import (
	"context"
	"io"
	"p2pcp/internal/auth"
	"p2pcp/internal/transfer"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCancelAuthentication(t *testing.T) {
	t.Parallel()

	net := mocknet.New()
	defer net.Close()

	h1, err := net.GenPeer()
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	peerID, err := authenticateReceiver(ctx, h1, nil, false)
	require.Error(t, err)
	require.Equal(t, context.Canceled, err)
	require.Empty(t, peerID)
}

func TestConnectionError(t *testing.T) {
	t.Parallel()

	net := mocknet.New()
	defer net.Close()

	h1, err := net.GenPeer()
	require.NoError(t, err)
	h2, err := net.GenPeer()
	require.NoError(t, err)
	err = net.LinkAll()
	require.NoError(t, err)

	secret := "test"
	secretHash := auth.ComputeHash([]byte(secret))

	var peerID peer.ID
	var authenticateErr error
	done := make(chan struct{})
	go func() {
		peerID, authenticateErr = authenticateReceiver(t.Context(), h1, secretHash, false)
		done <- struct{}{}
	}()

	for {
		time.Sleep(100 * time.Microsecond)
		stream, err := h2.NewStream(t.Context(), h1.ID(), auth.Protocol)
		if err == nil {
			stream.Close()
			break
		}
	}

	select {
	case <-done:
		t.Fatal("authentication should not have completed")
	default:
		assert.Empty(t, peerID)
		assert.Nil(t, authenticateErr)
	}
}

func TestUnauthorizedStream(t *testing.T) {
	t.Parallel()

	net := mocknet.New()
	defer net.Close()

	h1, err := net.GenPeer()
	require.NoError(t, err)
	h2, err := net.GenPeer()
	require.NoError(t, err)
	h3, err := net.GenPeer()
	require.NoError(t, err)
	err = net.LinkAll()
	require.NoError(t, err)

	streams, _ := getAuthorizedStreams(t.Context(), h1, h2.ID())

	go func() {
		for stream := range streams {
			func() {
				defer stream.Close()
				stream.Write([]byte{1})
			}()
		}
	}()

	stream1, err := h2.NewStream(t.Context(), h1.ID(), transfer.Protocol)
	assert.NoError(t, err)
	if err == nil {
		n, err := io.ReadFull(stream1, make([]byte, 1))
		assert.NoError(t, err)
		assert.Equal(t, 1, n)
		stream1.Close()
	}

	stream2, err := h3.NewStream(t.Context(), h1.ID(), transfer.Protocol)
	assert.NoError(t, err)
	if err == nil {
		n, err := io.ReadFull(stream2, make([]byte, 1))
		assert.Error(t, err)
		assert.Equal(t, 0, n)
		stream2.Close()
	}
}
