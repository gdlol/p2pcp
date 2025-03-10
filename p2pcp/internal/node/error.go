package node

import (
	"context"
	"encoding/gob"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
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
			if err == nil {
				_, err = stream.Write([]byte{1})
			}
			if err != nil {
				slog.Error(fmt.Sprintf("Error processing error message: %v", err))
			} else {
				handler(errStr)
			}
		}
	})
}

func sendError(ctx context.Context, host host.Host, peerID peer.ID, errStr string) error {
	ctx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	for ctx.Err() == nil {
		stream, err := host.NewStream(ctx, peerID, errorProtocol)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			slog.Debug("Error creating stream for error notification", "error", err)
			time.Sleep(100 * time.Millisecond)
			continue
		}
		err = func() error {
			defer stream.Close()
			err := writeString(stream, errStr)
			if err == nil {
				n, err := stream.Read(make([]byte, 1))
				if n == 1 {
					return nil
				} else {
					return fmt.Errorf("error reading error ack: %v", err)
				}
			}
			return err
		}()
		if err == nil {
			return nil
		} else {
			slog.Debug("Error sending error message", "error", err)
		}
	}
	return ctx.Err()
}
