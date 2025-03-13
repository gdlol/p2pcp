package node

import (
	"context"
	"testing"
	"time"

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

	errChan := make(chan error)
	go func() {
		errChan <- sendError(ctx, h1, h2.ID(), "test")
	}()

	select {
	case err := <-errChan:
		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
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

	errChan := make(chan error)
	go func() {
		errChan <- sendError(ctx, h1, h2.ID(), "test")
	}()

	time.Sleep(1 * time.Second)
	net.LinkAll()

	select {
	case err := <-errChan:
		assert.NoError(t, err)
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
