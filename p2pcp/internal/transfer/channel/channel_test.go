package channel

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type limitedChannel struct {
	limit  int
	reader io.Reader
	writer io.Writer
	closer func() error
}

func (l *limitedChannel) Close() error {
	return l.closer()
}

func (l *limitedChannel) Read(p []byte) (n int, err error) {
	return l.reader.Read(p)
}

func (l *limitedChannel) Write(p []byte) (n int, err error) {
	return l.writer.Write(p)
}

func newLimitedChannel(limit int, reader io.ReadCloser, writer io.WriteCloser) io.ReadWriteCloser {
	return &limitedChannel{
		limit:  limit,
		reader: io.LimitReader(reader, int64(limit)),
		writer: writer,
		closer: func() error {
			writer.Close()
			reader.Close()
			return nil
		},
	}
}

func newChannelPair(limit int) (io.ReadWriteCloser, io.ReadWriteCloser) {
	reader1, writer1 := io.Pipe()
	reader2, writer2 := io.Pipe()
	return newLimitedChannel(limit, reader1, writer2), newLimitedChannel(limit, reader2, writer1)
}

func TestChannel(t *testing.T) {
	dataLength := payloadSize * 10
	limits := make([]int, 0)
	for i := payloadSize + 10; i < payloadSize+100; i += 10 {
		limits = append(limits, i)
	}
	for i := payloadSize + 100; i < payloadSize+1000; i += 100 {
		limits = append(limits, i)
	}
	for i := payloadSize + 1000; i < payloadSize+10000; i += 1000 {
		limits = append(limits, i)
	}

	for _, limit := range limits {
		limit := limit
		t.Run(fmt.Sprint(limit), func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
			defer cancel()

			streamSource1 := make(chan io.ReadWriteCloser, 1)
			streamSource2 := make(chan io.ReadWriteCloser, 1)
			senderCtx, cancelSender := context.WithCancel(ctx)
			receiverCtx, cancelReceiver := context.WithCancel(ctx)

			go func() {
				for ctx.Err() == nil {
					stream1, stream2 := newChannelPair(limit)
					go func() {
						select {
						case <-receiverCtx.Done():
							stream1.Close()
							stream2.Close()
						case <-senderCtx.Done():
							stream1.Close()
							stream2.Close()
						case <-ctx.Done():
						}
					}()
					select {
					case <-senderCtx.Done():
						return
					case <-receiverCtx.Done():
						return
					case <-ctx.Done():
					case streamSource1 <- stream1:
					}
					select {
					case <-ctx.Done():
					case <-senderCtx.Done():
						return
					case <-receiverCtx.Done():
						return
					case streamSource2 <- stream2:
					}
					time.Sleep(100 * time.Millisecond)
				}
			}()

			sender := NewChannelWriter(ctx, func(ctx context.Context) (io.ReadWriteCloser, error) {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case stream := <-streamSource1:
					return stream, nil
				}
			})
			receiver := NewChannelReader(ctx, func(ctx context.Context) (io.ReadWriteCloser, error) {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case stream := <-streamSource2:
					return stream, nil
				}
			})

			data := make([]byte, dataLength)
			_, err := rand.Read(data)
			require.NoError(t, err)

			var sendErr error
			var receiveErr error

			go func() {
				defer cancelSender()
				defer func() {
					err := sender.Close()
					if err != nil {
						sendErr = err
					}
				}()
				remaining := data
				for len(remaining) > 0 {
					if ctx.Err() != nil {
						return
					}
					lengthB, err := rand.Int(rand.Reader, big.NewInt(int64(payloadSize)*2))
					if err != nil {
						sendErr = err
						return
					}
					length := min(lengthB.Int64(), int64(len(remaining)))
					n, err := sender.Write(remaining[:length])
					if err != nil {
						sendErr = err
						return
					}
					remaining = remaining[n:]
				}
			}()

			received := make([]byte, 0)
			go func() {
				defer cancelReceiver()
				defer func() {
					err := receiver.Close()
					if err != nil {
						receiveErr = err
					}
				}()
				buffer := make([]byte, payloadSize*2)
				for {
					if ctx.Err() != nil {
						return
					}
					length, err := rand.Int(rand.Reader, big.NewInt(int64(payloadSize)*2))
					if err != nil {
						receiveErr = err
						return
					}
					n, err := receiver.Read(buffer[:length.Int64()])
					if err == io.EOF {
						break
					}
					if err != nil {
						receiveErr = err
						return
					}
					received = append(received, buffer[:n]...)
				}
			}()

			done := make(chan struct{})
			go func() {
				<-senderCtx.Done()
				<-receiverCtx.Done()
				done <- struct{}{}
			}()

			select {
			case <-ctx.Done():
			case <-done:
			}
			if sendErr != nil {
				assert.Equal(t, context.DeadlineExceeded, sendErr)
			}
			if receiveErr != nil {
				assert.Equal(t, context.DeadlineExceeded, receiveErr)
			}
			require.NoError(t, ctx.Err())
			assert.Equal(t, data, received)
		})
	}
}
