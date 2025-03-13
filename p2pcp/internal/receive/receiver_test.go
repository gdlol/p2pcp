package receive

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnectTimeout(t *testing.T) {
	t.Parallel()

	net := mocknet.New()
	defer net.Close()

	h1, err := net.GenPeer()
	require.NoError(t, err)
	h2, err := net.GenPeer()
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
