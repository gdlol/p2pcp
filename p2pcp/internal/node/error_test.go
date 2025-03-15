package node

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendErrorTimeout(t *testing.T) {
	t.Parallel()

	net := mocknet.New()
	defer net.Close()

	h1, err := net.GenPeer()
	require.NoError(t, err)
	h2, err := net.GenPeer()
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	timer := time.AfterFunc(5*time.Second, func() {})

	done := make(chan struct{})
	go func() {
		sendError(ctx, h1, h2.ID(), "test")
		done <- struct{}{}
	}()

	select {
	case <-done:
		stopped := timer.Stop()
		assert.False(t, stopped)
		assert.NoError(t, ctx.Err())
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestSendErrorRetry(t *testing.T) {
	t.Parallel()

	net := mocknet.New()
	defer net.Close()

	h1, err := net.GenPeer()
	require.NoError(t, err)
	h2, err := net.GenPeer()
	require.NoError(t, err)

	handled := make(chan struct{})
	registerErrorHandler(h2, h1.ID(), func(errStr string) {
		handled <- struct{}{}
	})

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		sendError(ctx, h1, h2.ID(), "test")
		done <- struct{}{}
	}()

	time.Sleep(1 * time.Second)
	err = net.LinkAll()
	require.NoError(t, err)

	select {
	case <-done:
		assert.NoError(t, ctx.Err())
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
		defer cancel()
		select {
		case <-handled:
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestSendErrorDisconnect(t *testing.T) {
	t.Parallel()

	net := mocknet.New()
	defer net.Close()

	h1, err := net.GenPeer()
	require.NoError(t, err)
	h2, err := net.GenPeer()
	require.NoError(t, err)
	err = net.LinkAll()
	require.NoError(t, err)

	h2.SetStreamHandler(errorProtocol, func(stream network.Stream) {
		stream.Close()
	})

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	sendError(ctx, h1, h2.ID(), "test")
	assert.Error(t, ctx.Err())
	assert.Equal(t, context.DeadlineExceeded, ctx.Err())
}

func TestReceiveErrorDisconnect(t *testing.T) {
	t.Parallel()

	net := mocknet.New()
	defer net.Close()

	h1, err := net.GenPeer()
	require.NoError(t, err)
	h2, err := net.GenPeer()
	require.NoError(t, err)

	handled := make(chan struct{})
	registerErrorHandler(h2, h1.ID(), func(errStr string) {
		handled <- struct{}{}
	})
	err = net.LinkAll()
	require.NoError(t, err)

	for range 10 {
		time.Sleep(100 * time.Millisecond)
		stream, err := h1.NewStream(t.Context(), h2.ID(), errorProtocol)
		require.NoError(t, err)
		stream.Close()
	}
	select {
	case <-handled:
		t.Fail()
	default:
	}

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	sendError(ctx, h1, h2.ID(), "test")
	select {
	case <-handled:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}
