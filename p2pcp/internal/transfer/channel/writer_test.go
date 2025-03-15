package channel

import (
	"context"
	"errors"
	"io"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteEmpty(t *testing.T) {
	t.Parallel()

	channel1, channel2 := newChannelPair(math.MaxInt32)
	channel1.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	writer := NewChannelWriter(ctx, func(ctx context.Context) (io.ReadWriteCloser, error) {
		time.Sleep(100 * time.Millisecond)
		return channel2, nil
	})

	n, err := writer.Write([]byte{})
	assert.NoError(t, err)
	assert.Zero(t, n)
}

func TestWriteGetStreamError(t *testing.T) {
	t.Parallel()

	channel1, channel2 := newChannelPair(math.MaxInt32)
	channel1.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	cancelCtx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	writer := NewChannelWriter(ctx, func(ctx context.Context) (io.ReadWriteCloser, error) {
		time.Sleep(100 * time.Millisecond)
		return channel2, cancelCtx.Err()
	})

	n, err := writer.Write(make([]byte, 1))
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
	assert.Zero(t, n)
	assert.NoError(t, ctx.Err())
}

func TestCancelWrite(t *testing.T) {
	t.Parallel()

	channel1, channel2 := newChannelPair(math.MaxInt32)
	channel1.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()

	writer := NewChannelWriter(ctx, func(ctx context.Context) (io.ReadWriteCloser, error) {
		time.Sleep(100 * time.Millisecond)
		return channel2, nil
	})

	n, err := writer.Write(make([]byte, 1))
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
	assert.Zero(t, n)
	assert.Error(t, ctx.Err())
	assert.Equal(t, context.DeadlineExceeded, ctx.Err())
}

func TestWriteSync(t *testing.T) {
	t.Parallel()

	channel1, channel2 := newChannelPair(math.MaxInt32)
	channel3, channel4 := newChannelPair(math.MaxInt32)
	channel5, channel6 := newChannelPair(math.MaxInt32)
	channel7, channel8 := newChannelPair(math.MaxInt32)
	channel9, channel10 := newChannelPair(math.MaxInt32)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	count := 0
	writer := NewChannelWriter(ctx, func(ctx context.Context) (io.ReadWriteCloser, error) {
		if count == 0 {
			count++
			return channel2, nil
		} else if count == 1 {
			count++
			return channel4, nil
		} else if count == 2 {
			count++
			return channel6, nil
		} else if count == 3 {
			count++
			return channel8, nil
		} else {
			return channel10, nil
		}
	})

	doneWrite := make(chan struct{})
	go func() {
		n, err := writer.Write(make([]byte, 1))
		assert.NoError(t, err)
		assert.Equal(t, 1, n)

		n, err = writer.Write(make([]byte, 1))
		assert.NoError(t, err)
		assert.Equal(t, 1, n)

		doneWrite <- struct{}{}
	}()

	// Read some data
	buffer := [readBufferSize]byte{}
	ack, n, err := readPacket(channel1, &buffer)
	require.NoError(t, err)
	require.False(t, ack)
	require.Equal(t, 1, n)

	// Disconnect
	channel1.Close()

	// Disconnect again
	channel3.Close()

	// Sync
	ack, n, err = readPacket(channel5, &buffer)
	require.NoError(t, err)
	require.True(t, ack)
	require.Equal(t, 0, n)

	// Disconnect
	channel5.Close()

	// Sync
	ack, n, err = readPacket(channel7, &buffer)
	require.NoError(t, err)
	require.True(t, ack)
	require.Equal(t, 0, n)
	writeAckResponse(channel7, 0)

	// Read resent data
	ack, n, err = readPacket(channel7, &buffer)
	require.NoError(t, err)
	require.False(t, ack)
	require.Equal(t, 1, n)

	// Disconnect
	channel7.Close()

	// Sync
	ack, n, err = readPacket(channel9, &buffer)
	require.NoError(t, err)
	require.True(t, ack)
	require.Equal(t, 0, n)
	err = writeAckResponse(channel9, 1)
	require.NoError(t, err)

	// Read more data
	ack, n, err = readPacket(channel9, &buffer)
	require.NoError(t, err)
	require.False(t, ack)
	require.Equal(t, 1, n)

	select {
	case <-doneWrite:
	case <-ctx.Done():
		t.Fail()
	}
}

func TestCancelFlush(t *testing.T) {
	t.Parallel()

	channel1, channel2 := newChannelPair(math.MaxInt32)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	cancelCtx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	writer := NewChannelWriter(ctx, func(ctx context.Context) (io.ReadWriteCloser, error) {
		return channel2, cancelCtx.Err()
	})

	doneWrite := make(chan struct{})
	go func() {
		n, err := writer.Write(make([]byte, 1))
		assert.NoError(t, err)
		assert.Equal(t, 1, n)

		err = writer.Flush(true)
		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)

		doneWrite <- struct{}{}
	}()

	// Read some data
	buffer := [readBufferSize]byte{}
	ack, n, err := readPacket(channel1, &buffer)
	require.NoError(t, err)
	require.False(t, ack)
	require.Equal(t, 1, n)
	channel1.Close()

	select {
	case <-doneWrite:
	case <-ctx.Done():
		t.Fail()
	}
}

func TestFlushTimeout(t *testing.T) {
	t.Parallel()

	channel1, channel2 := newChannelPair(math.MaxInt32)

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()

	writer := NewChannelWriter(ctx, func(ctx context.Context) (io.ReadWriteCloser, error) {
		return channel2, nil
	})

	doneWrite := make(chan struct{})
	go func() {
		n, err := writer.Write(make([]byte, 1))
		assert.NoError(t, err)
		assert.Equal(t, 1, n)

		err = writer.Flush(true)
		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)

		doneWrite <- struct{}{}
	}()

	// Read some data
	buffer := [readBufferSize]byte{}
	ack, n, err := readPacket(channel1, &buffer)
	require.NoError(t, err)
	require.False(t, ack)
	require.Equal(t, 1, n)
	channel1.Close()

	<-doneWrite
	assert.Error(t, ctx.Err())
	assert.Equal(t, context.DeadlineExceeded, ctx.Err())
}

func TestFlushSync(t *testing.T) {
	t.Parallel()

	channel1, channel2 := newChannelPair(math.MaxInt32)
	channel3, channel4 := newChannelPair(math.MaxInt32)
	channel5, channel6 := newChannelPair(math.MaxInt32)
	channel7, channel8 := newChannelPair(math.MaxInt32)
	channel9, channel10 := newChannelPair(math.MaxInt32)
	defer channel7.Close()
	defer channel9.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	count := 0
	writer := NewChannelWriter(ctx, func(ctx context.Context) (io.ReadWriteCloser, error) {
		if count == 0 {
			count++
			return channel2, nil
		} else if count == 1 {
			count++
			return channel4, nil
		} else if count == 2 {
			count++
			return channel6, nil
		} else if count == 3 {
			count++
			return channel8, nil
		} else {
			return channel10, nil
		}
	})

	doneWrite := make(chan struct{})
	go func() {
		n, err := writer.Write(make([]byte, payloadSize))
		assert.NoError(t, err)
		assert.Equal(t, payloadSize, n)

		n, err = writer.Write(make([]byte, payloadSize))
		assert.NoError(t, err)
		assert.Equal(t, payloadSize, n)

		err = writer.Flush(true)
		assert.NoError(t, err)

		doneWrite <- struct{}{}
	}()

	// Read some data
	buffer := [readBufferSize]byte{}
	ack, n, err := readPacket(channel1, &buffer)
	require.NoError(t, err)
	require.False(t, ack)
	require.Equal(t, payloadSize, n)
	ack, n, err = readPacket(channel1, &buffer)
	require.NoError(t, err)
	require.False(t, ack)
	require.Equal(t, payloadSize, n)

	// Disconnect
	channel1.Close()

	// Sync
	ack, n, err = readPacket(channel3, &buffer)
	require.NoError(t, err)
	require.True(t, ack)
	require.Equal(t, 0, n)
	writeAckResponse(channel3, 0)

	// Read resent data
	ack, n, err = readPacket(channel3, &buffer)
	require.NoError(t, err)
	require.False(t, ack)
	require.Equal(t, payloadSize, n)

	// Disconnect
	channel3.Close()

	// Sync
	ack, n, err = readPacket(channel5, &buffer)
	require.NoError(t, err)
	require.True(t, ack)
	require.Equal(t, 0, n)
	writeAckResponse(channel5, payloadSize)

	// Read resent data
	ack, n, err = readPacket(channel5, &buffer)
	require.NoError(t, err)
	require.False(t, ack)
	require.Equal(t, payloadSize, n)

	// Sync
	ack, n, err = readPacket(channel5, &buffer)
	require.NoError(t, err)
	require.True(t, ack)
	require.Equal(t, 0, n)
	writeAckResponse(channel5, payloadSize+1)

	// Read resent data
	ack, n, err = readPacket(channel5, &buffer)
	require.NoError(t, err)
	require.False(t, ack)
	require.Equal(t, payloadSize-1, n)

	// Sync
	ack, n, err = readPacket(channel5, &buffer)
	require.NoError(t, err)
	require.True(t, ack)
	require.Equal(t, 0, n)
	writeAckResponse(channel5, payloadSize+2)

	// Read resent data
	ack, n, err = readPacket(channel5, &buffer)
	require.NoError(t, err)
	require.False(t, ack)
	require.Equal(t, payloadSize-2, n)

	// Sync
	ack, n, err = readPacket(channel5, &buffer)
	require.NoError(t, err)
	require.True(t, ack)
	require.Equal(t, 0, n)
	writeAckResponse(channel5, payloadSize*2)

	select {
	case <-doneWrite:
	case <-ctx.Done():
		t.Fail()
	}
}

func TestCancelFlushInWrite(t *testing.T) {
	t.Parallel()

	channel1, channel2 := newChannelPair(math.MaxInt32)
	channel3, channel4 := newChannelPair(math.MaxInt32)
	defer channel3.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	count := 0
	writer := NewChannelWriter(ctx, func(ctx context.Context) (io.ReadWriteCloser, error) {
		if count == 0 {
			count++
			return channel2, nil
		} else {
			return channel4, nil
		}
	})

	doneWrite := make(chan struct{})
	go func() {
		n, err := writer.Write(make([]byte, 1))
		assert.NoError(t, err)
		assert.Equal(t, 1, n)

		n, err = writer.Write(make([]byte, 1))
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
		assert.Equal(t, 0, n)

		doneWrite <- struct{}{}
	}()

	// Read some data
	buffer := [readBufferSize]byte{}
	ack, n, err := readPacket(channel1, &buffer)
	require.NoError(t, err)
	require.False(t, ack)
	require.Equal(t, 1, n)

	// Disconnect
	channel1.Close()

	// Sync
	ack, n, err = readPacket(channel3, &buffer)
	require.NoError(t, err)
	require.True(t, ack)
	require.Equal(t, 0, n)
	channel3.Close()

	cancel()

	<-doneWrite
	assert.Error(t, ctx.Err())
	assert.Equal(t, context.Canceled, ctx.Err())
}

func TestCancelClose(t *testing.T) {
	t.Parallel()

	channel1, channel2 := newChannelPair(math.MaxInt32)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	cancelCtx, cancel := context.WithCancel(ctx)

	writer := NewChannelWriter(cancelCtx, func(ctx context.Context) (io.ReadWriteCloser, error) {
		return channel2, nil
	})

	writerDone := make(chan struct{})
	go func() {
		n, err := writer.Write(make([]byte, 1))
		assert.NoError(t, err)
		assert.Equal(t, 1, n)

		err = writer.Close()
		assert.Error(t, err)
		assert.True(t, errors.Is(err, context.Canceled))

		err = writer.Close()
		assert.NoError(t, err)

		writerDone <- struct{}{}
	}()

	buffer := [readBufferSize]byte{}
	ack, n, err := readPacket(channel1, &buffer)
	require.NoError(t, err)
	require.False(t, ack)
	require.Equal(t, 1, n)
	channel1.Close()
	cancel()

	select {
	case <-writerDone:
	case <-ctx.Done():
		t.Fail()
	}
}

func TestCancelCloseFlushed(t *testing.T) {
	t.Parallel()

	channel1, channel2 := newChannelPair(math.MaxInt32)
	channel1.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	cancelCtx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	writer := NewChannelWriter(ctx, func(ctx context.Context) (io.ReadWriteCloser, error) {
		return channel2, cancelCtx.Err()
	})

	err := writer.Close()
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
	assert.NoError(t, ctx.Err())
}

func TestCancelTimeoutClose(t *testing.T) {
	t.Parallel()

	channel1, channel2 := newChannelPair(math.MaxInt32)
	channel1.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	writer := NewChannelWriter(ctx, func(ctx context.Context) (io.ReadWriteCloser, error) {
		return channel2, nil
	})

	err := writer.Close()
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
	assert.NoError(t, ctx.Err())
}

func TestCloseSync(t *testing.T) {
	t.Parallel()

	channel1, channel2 := newChannelPair(math.MaxInt32)
	channel3, channel4 := newChannelPair(math.MaxInt32)
	channel5, channel6 := newChannelPair(math.MaxInt32)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	count := 0
	writer := NewChannelWriter(ctx, func(ctx context.Context) (io.ReadWriteCloser, error) {
		if count == 0 {
			count++
			return channel2, nil
		} else if count == 1 {
			count++
			return channel4, nil
		} else {
			count++
			return channel6, nil
		}
	})

	closed := make(chan struct{})
	go func() {
		err := writer.Close()
		assert.NoError(t, err)

		closed <- struct{}{}
	}()

	// Read FIN
	buffer := [readBufferSize]byte{}
	ack, n, err := readPacket(channel1, &buffer)
	require.NoError(t, err)
	require.False(t, ack)
	require.Zero(t, n)

	// Disconnect
	channel1.Close()

	// Read FIN
	ack, n, err = readPacket(channel3, &buffer)
	require.NoError(t, err)
	require.False(t, ack)
	require.Zero(t, n)

	// Sync
	ack, n, err = readPacket(channel3, &buffer)
	require.NoError(t, err)
	require.True(t, ack)
	require.Zero(t, n)

	// Disconnect
	channel3.Close()

	// Read FIN
	ack, n, err = readPacket(channel5, &buffer)
	require.NoError(t, err)
	require.False(t, ack)
	require.Zero(t, n)

	// Sync
	ack, n, err = readPacket(channel5, &buffer)
	require.NoError(t, err)
	require.True(t, ack)
	require.Zero(t, n)
	err = writeAckResponse(channel5, math.MaxUint64)
	require.NoError(t, err)

	select {
	case <-closed:
	case <-ctx.Done():
		t.Fail()
	}
}
