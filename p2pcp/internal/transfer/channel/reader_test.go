package channel

import (
	"context"
	"io"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadClosed(t *testing.T) {
	t.Parallel()

	channel1, channel2 := newChannelPair(math.MaxInt32)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	reader := NewChannelReader(ctx, func(ctx context.Context) (io.ReadWriteCloser, error) {
		time.Sleep(100 * time.Millisecond)
		return channel1, nil
	})
	writer := NewChannelWriter(ctx, func(ctx context.Context) (io.ReadWriteCloser, error) {
		time.Sleep(100 * time.Millisecond)
		return channel2, nil
	})

	writerClosed := make(chan struct{})
	go func() {
		err := writer.Close()
		require.NoError(t, err)
		writerClosed <- struct{}{}
	}()

	n, err := reader.Read(make([]byte, 1))
	assert.Error(t, err)
	assert.Equal(t, io.EOF, err)
	assert.Zero(t, n)

	n, err = reader.Read(make([]byte, 1))
	assert.Error(t, err)
	assert.Equal(t, io.EOF, err)
	assert.Zero(t, n)

	err = reader.Close()
	require.NoError(t, err)
	<-writerClosed
	assert.NoError(t, ctx.Err())

	n, err = writer.Write([]byte{0})
	assert.Error(t, err)
	assert.Equal(t, io.ErrClosedPipe, err)
	assert.Zero(t, n)
}

func TestReadEmpty(t *testing.T) {
	t.Parallel()

	channel1, channel2 := newChannelPair(math.MaxInt32)
	channel2.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	reader := NewChannelReader(ctx, func(ctx context.Context) (io.ReadWriteCloser, error) {
		time.Sleep(100 * time.Millisecond)
		return channel1, nil
	})

	n, err := reader.Read(nil)
	assert.NoError(t, err)
	assert.Zero(t, n)
}

func TestReadGetStreamError(t *testing.T) {
	t.Parallel()

	channel1, channel2 := newChannelPair(math.MaxInt32)
	channel2.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cancelCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	reader := NewChannelReader(ctx, func(ctx context.Context) (io.ReadWriteCloser, error) {
		time.Sleep(100 * time.Millisecond)
		return channel1, cancelCtx.Err()
	})

	n, err := reader.Read(make([]byte, 1))
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
	assert.Zero(t, n)
	assert.NoError(t, ctx.Err())
}

func TestCancelRead(t *testing.T) {
	t.Parallel()

	channel1, channel2 := newChannelPair(math.MaxInt32)
	channel2.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	reader := NewChannelReader(ctx, func(ctx context.Context) (io.ReadWriteCloser, error) {
		time.Sleep(100 * time.Millisecond)
		return channel1, nil
	})

	n, err := reader.Read(make([]byte, 1))
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
	assert.Zero(t, n)
	assert.Error(t, ctx.Err())
	assert.Equal(t, context.DeadlineExceeded, ctx.Err())
}

func TestReadAckError(t *testing.T) {
	t.Parallel()

	channel1, channel2 := newChannelPair(math.MaxInt32)
	channel3, channel4 := newChannelPair(math.MaxInt32)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	count := 0
	reader := NewChannelReader(ctx, func(ctx context.Context) (io.ReadWriteCloser, error) {
		if count == 0 {
			count++
			return channel1, nil
		} else {
			return channel3, nil
		}
	})

	doneRead := make(chan struct{})
	go func() {
		buffer := make([]byte, 1)
		n, err := reader.Read(buffer)
		assert.NoError(t, err)
		assert.Equal(t, 1, n)
		assert.Equal(t, byte(1), buffer[0])
		doneRead <- struct{}{}
	}()

	// disconnect after request
	err := writeAckRequest(channel2)
	require.NoError(t, err)
	channel2.Close()

	select {
	case <-doneRead:
		t.Fail()
	default:
	}

	// resume connection and send data
	err = writeAckRequest(channel4)
	require.NoError(t, err)
	n, err := readAckResponse(channel4)
	require.NoError(t, err)
	assert.Equal(t, uint64(0), n)
	err = writeData(channel4, []byte{1})
	require.NoError(t, err)

	select {
	case <-doneRead:
	case <-ctx.Done():
		t.Fail()
	}
}

func TestDoubleCloseRead(t *testing.T) {
	t.Parallel()

	channel1, channel2 := newChannelPair(math.MaxInt32)
	channel2.Close()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	reader := NewChannelReader(ctx, func(ctx context.Context) (io.ReadWriteCloser, error) {
		time.Sleep(100 * time.Millisecond)
		return channel1, ctx.Err()
	})

	err := reader.Close()
	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
	err = reader.Close()
	assert.NoError(t, err)
}

func TestCloseReadGetStreamError(t *testing.T) {
	t.Parallel()

	channel1, channel2 := newChannelPair(math.MaxInt32)
	channel2.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cancelCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	reader := NewChannelReader(ctx, func(ctx context.Context) (io.ReadWriteCloser, error) {
		time.Sleep(100 * time.Millisecond)
		return channel1, cancelCtx.Err()
	})

	err := reader.Close()
	require.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
	assert.NoError(t, ctx.Err())
}

func TestCloseReadTimeout(t *testing.T) {
	t.Parallel()

	channel1, channel2 := newChannelPair(math.MaxInt32)
	channel2.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	reader := NewChannelReader(ctx, func(ctx context.Context) (io.ReadWriteCloser, error) {
		time.Sleep(100 * time.Millisecond)
		return channel1, nil
	})

	err := reader.Close()
	require.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
	assert.NoError(t, ctx.Err())
}

func TestCloseReadAfterReadClosed(t *testing.T) {
	t.Parallel()

	channel1, channel2 := newChannelPair(math.MaxInt32)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	reader := NewChannelReader(ctx, func(ctx context.Context) (io.ReadWriteCloser, error) {
		time.Sleep(100 * time.Millisecond)
		return channel1, nil
	})

	closed := make(chan struct{})
	go func() {
		err := reader.Close()
		require.NoError(t, err)
		assert.NoError(t, ctx.Err())
		closed <- struct{}{}
	}()

	writeData(channel2, nil) // FIN
	time.Sleep(1 * time.Second)
	select {
	case <-closed:
		t.Fail()
	default:
	}
	channel2.Close()

	<-closed
	assert.NoError(t, ctx.Err())
}

func TestReadAckErrorClosing(t *testing.T) {
	t.Parallel()

	channel1, channel2 := newChannelPair(math.MaxInt32)
	channel3, channel4 := newChannelPair(math.MaxInt32)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	count := 0
	reader := NewChannelReader(ctx, func(ctx context.Context) (io.ReadWriteCloser, error) {
		if count == 0 {
			count++
			return channel1, nil
		} else {
			return channel3, nil
		}
	})

	closed := make(chan struct{})
	go func() {
		err := reader.Close()
		require.NoError(t, err)
		assert.NoError(t, ctx.Err())
		closed <- struct{}{}
	}()

	// disconnect after request
	err := writeAckRequest(channel2)
	require.NoError(t, err)
	channel2.Close()

	select {
	case <-closed:
		t.Fail()
	default:
	}

	// resume connection and sync
	err = writeAckRequest(channel4)
	require.NoError(t, err)
	n, err := readAckResponse(channel4)
	require.NoError(t, err)
	assert.Equal(t, uint64(0), n)

	select {
	case <-closed:
		t.Fail()
	default:
	}

	writeData(channel4, nil) // FIN
	time.Sleep(1 * time.Second)
	select {
	case <-closed:
		t.Fail()
	default:
	}

	// Sync again
	err = writeAckRequest(channel4)
	require.NoError(t, err)
	n, err = readAckResponse(channel4)
	require.NoError(t, err)
	assert.Equal(t, uint64(math.MaxUint64), n)

	select {
	case <-closed:
		t.Fail()
	default:
	}
	channel4.Close()

	<-closed
	assert.NoError(t, ctx.Err())
}

func TestReadDataClosing(t *testing.T) {
	t.Parallel()

	channel1, channel2 := newChannelPair(math.MaxInt32)
	defer channel2.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	reader := NewChannelReader(ctx, func(ctx context.Context) (io.ReadWriteCloser, error) {
		time.Sleep(100 * time.Millisecond)
		return channel1, nil
	})

	closed := make(chan struct{})
	go func() {
		err := reader.Close()
		require.Error(t, err)
		assert.Equal(t, "unexpected data packet during close", err.Error())
		closed <- struct{}{}
	}()

	writeData(channel2, []byte{1})
	<-closed
	assert.NoError(t, ctx.Err())
}
