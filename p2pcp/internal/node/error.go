package node

import (
	"context"
	"encoding/gob"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/rand"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/backoff"
)

const errorProtocol protocol.ID = "/p2pcp/error/0.1.0"

func writeString(writer io.Writer, str string) error {
	encoder := gob.NewEncoder(writer)
	return encoder.Encode(str)
}

func readString(reader io.Reader) (string, error) {
	decoder := gob.NewDecoder(reader)
	var str string
	err := decoder.Decode(&str)
	return str, err
}

func registerErrorHandler(host host.Host, peerID peer.ID, handler func(string)) {
	host.SetStreamHandler(errorProtocol, func(stream network.Stream) {
		defer stream.Close()
		if stream.Conn().RemotePeer() == peerID {
			errStr, err := readString(stream)
			stream.Write([]byte{1})
			if err != nil {
				handler(fmt.Sprintf("Failed to read error message: %v", err))
			} else {
				handler(errStr)
			}
		}
	})
}

func sendError(ctx context.Context, host host.Host, peerID peer.ID, errStr string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	b := backoff.NewExponentialBackoff(
		0, 3*time.Second, backoff.NoJitter,
		100*time.Millisecond, math.Sqrt2, 0,
		rand.NewSource(0))()
	for ctx.Err() == nil {
		stream, err := host.NewStream(ctx, peerID, errorProtocol)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			slog.Debug("Error creating stream for error notification", "error", err)
			time.Sleep(b.Delay())
			continue
		}
		err = func() error {
			defer stream.Close()
			err := writeString(stream, errStr)
			if err == nil {
				n, _ := stream.Read(make([]byte, 1))
				if n == 1 {
					return nil
				}
			}
			return err
		}()
		if err == nil {
			return nil
		}
	}
	return ctx.Err()
}
