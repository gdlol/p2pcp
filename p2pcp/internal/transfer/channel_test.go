package transfer

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
	dataLength := DefaultPayloadSize * 10
	minLimit := DefaultPayloadSize + headerSize
	limits := []int{minLimit - 1, minLimit, minLimit + 1}
	for i := minLimit + 1000; i < DefaultPayloadSize*2; i += 1000 {
		limits = append(limits, i)
	}

	var testWg sync.WaitGroup
	for _, limit := range limits {
		testWg.Add(1)
		go t.Run(fmt.Sprint(limit), func(t *testing.T) {
			defer testWg.Done()
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()

			streamSource1 := make(chan io.ReadWriteCloser, 1)
			streamSource2 := make(chan io.ReadWriteCloser, 1)

			go func() {
				for ctx.Err() == nil {
					stream1, stream2 := newChannelPair(limit)
					select {
					case <-ctx.Done():
						return
					case streamSource1 <- stream1:
					}
					select {
					case <-ctx.Done():
						return
					case streamSource2 <- stream2:
					}
					time.Sleep(100 * time.Millisecond)
				}
			}()

			channel1 := NewChannel(ctx, func(ctx context.Context) (io.ReadWriteCloser, error) {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case stream := <-streamSource1:
					return stream, nil
				}
			}, DefaultPayloadSize)
			channel2 := NewChannel(ctx, func(ctx context.Context) (io.ReadWriteCloser, error) {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case stream := <-streamSource2:
					return stream, nil
				}
			}, DefaultPayloadSize)

			data := make([]byte, dataLength)
			_, err := rand.Read(data)
			assert.NoError(t, err)

			var transferWg sync.WaitGroup
			transferWg.Add(2)

			var sendErr error
			var receiveErr error

			go func() {
				defer transferWg.Done()
				defer func() {
					err := channel1.Close()
					if err != nil {
						sendErr = err
					}
				}()
				remaining := data
				for len(remaining) > 0 {
					if ctx.Err() != nil {
						return
					}
					lengthB, err := rand.Int(rand.Reader, big.NewInt(int64(DefaultPayloadSize)*2))
					if err != nil {
						sendErr = err
						return
					}
					length := min(lengthB.Int64(), int64(len(remaining)))
					n, err := channel1.Write(remaining[:length])
					if err != nil {
						sendErr = err
						return
					}
					remaining = remaining[n:]
				}
			}()

			received := make([]byte, 0)
			go func() {
				defer transferWg.Done()
				defer func() {
					err := channel2.Close()
					if err != nil {
						receiveErr = err
					}
				}()
				buffer := make([]byte, DefaultPayloadSize*2)
				for {
					if ctx.Err() != nil {
						return
					}
					length, err := rand.Int(rand.Reader, big.NewInt(int64(DefaultPayloadSize)*2))
					if err != nil {
						receiveErr = err
						return
					}
					n, err := channel2.Read(buffer[:length.Int64()])
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

			transferWg.Wait()
			if limit < minLimit {
				assert.Error(t, sendErr)
				assert.Error(t, receiveErr)
				assert.Equal(t, ctx.Err(), context.DeadlineExceeded)
			} else {
				assert.NoError(t, sendErr)
				assert.NoError(t, receiveErr)
				assert.ElementsMatch(t, data, received)
			}
		})
	}
	testWg.Wait()
}
