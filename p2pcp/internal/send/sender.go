package send

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"p2pcp/internal/auth"
	"p2pcp/internal/node"
	"p2pcp/internal/transfer"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoremem"
)

type Sender interface {
	GetNode() node.Node
	GetAdvertiseTopic() string
	AdvertiseWAN(ctx context.Context)
	Send(ctx context.Context, secretHash []byte, path string, strict bool) error
	Close()
}

type sender struct {
	node node.Node
}

func (s *sender) GetNode() node.Node {
	return s.node
}

func (s *sender) GetAdvertiseTopic() string {
	id := s.node.ID().String()
	return id[len(id)-7:]
}

func (s *sender) Close() {
	s.node.Close()
}

func (s *sender) AdvertiseWAN(ctx context.Context) {
	node := s.node
	topic := s.GetAdvertiseTopic()

	// Advertise self to DHT until success/cancel.
	for ctx.Err() == nil {
		time.Sleep(3 * time.Second)
		slog.Debug("Advertising to WAN DHT...", "topic", topic)
		err := node.AdvertiseWAN(ctx, topic)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			slog.Debug("Error advertising to WAN DHT, retrying...", "error", err)
		} else {
			slog.Debug("Advertised to WAN DHT.")
			break
		}
	}
}

func (s *sender) Send(ctx context.Context, secretHash []byte, path string, strict bool) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	n := s.node
	host := n.GetHost()
	var authenticatedPeer peer.ID = ""
	authenticate := make(chan bool, 1)

	streams := make(chan transfer.ChannelStream)
	channel := transfer.NewChannel(ctx, streams, transfer.DefaultPayloadSize)
	defer channel.Close()

	mutex := sync.Mutex{}
	host.SetStreamHandler(node.Protocol, func(stream network.Stream) {
		slog.Debug("New stream received.")
		if authenticatedPeer.Validate() == nil && stream.Conn().RemotePeer() != authenticatedPeer {
			slog.Warn("Received request from other peer after authentication.", "peer", stream.Conn().RemotePeer())
			return
		}

		mutex.Lock()
		defer mutex.Unlock()
		if authenticatedPeer.Validate() != nil {
			defer stream.Close()

			buffer := make([]byte, len(secretHash))
			_, err := io.ReadFull(stream, buffer)
			if err != nil {
				slog.Error("Error reading secret hash.", "error", err)
				stream.Write([]byte{0})
				if !strict {
					authenticate <- false
				}
				return
			}
			if !auth.VerifyHash(buffer, secretHash) {
				slog.Warn("Invalid secret hash received.")
				stream.Write([]byte{0})
				if !strict {
					authenticate <- false
				}
				return
			}

			stream.Write([]byte{1})
			authenticatedPeer = stream.Conn().RemotePeer()
			authenticate <- true
			return
		}

		channelStream := transfer.NewChannelStream(stream)
		select {
		case streams <- *channelStream:
			<-channelStream.Done
		case <-ctx.Done():
		}
	})
	defer host.RemoveStreamHandler(node.Protocol)

	if !<-authenticate {
		cancel()
		return fmt.Errorf("failed to authenticate receiver")
	}
	slog.Info("Authenticated receiver.", "id", authenticatedPeer)

	pipeReader, pipeWriter := io.Pipe()
	done := make(chan struct{}, 1)
	go func() {
		defer pipeReader.Close()
		buffer := make([]byte, transfer.DefaultPayloadSize)
		for ctx.Err() == nil {
			n, err := pipeReader.Read(buffer)
			if err != nil {
				if err != io.EOF {
					slog.Error("Error reading from pipe.", "error", err)
				}
				break
			}
			if n > 0 {
				_, err = channel.Write(buffer[:n])
				if err != nil {
					slog.Error("Error writing to channel.", "error", err)
					break
				}
			}
			if err == io.EOF {
				slog.Debug("Read EOF from pipe.")
				break
			}
		}
		done <- struct{}{}
	}()

	err := writeTar(pipeWriter, path)
	if err != nil {
		return fmt.Errorf("error sending path %s: %w", path, err)
	}

	pipeWriter.Close()
	<-done
	if err = channel.Close(); err != nil {
		slog.Debug("Error closing channel.", "error", err)
	}
	slog.Info("Transfer complete.")
	return nil
}

func NewSender(ctx context.Context, options ...libp2p.Option) (Sender, error) {
	node, err := node.NewNode(ctx, options...)
	if err != nil {
		return nil, fmt.Errorf("error creating sender: %w", err)
	}
	return &sender{node: node}, nil
}

// Create new sender every 6 seconds. until 1 successfully advertised itself to WAN DHT.
func NewAdvertisedSender(ctx context.Context, options ...libp2p.Option) (Sender, error) {
	groupCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	ps, err := pstoremem.NewPeerstore()
	if err != nil {
		return nil, fmt.Errorf("error creating peerstore: %w", err)
	}

	resultChan := make(chan Sender, 1)
	var wg sync.WaitGroup

	// Try advertising up to 1 minute.
	launchNode := func() error {
		if groupCtx.Err() != nil {
			return groupCtx.Err()
		}

		sender, err := NewSender(ctx, append([]libp2p.Option{libp2p.Peerstore(ps)}, options...)...)
		if err != nil {
			return err
		}
		success := false
		defer func() {
			if !success {
				sender.Close()
			}
		}()

		timeoutCtx, cancel := context.WithTimeout(groupCtx, time.Minute)
		defer cancel()
		sender.AdvertiseWAN(timeoutCtx)

		select {
		case resultChan <- sender:
			success = true
		case <-groupCtx.Done():
		}
		return nil
	}

	go func() {
		for i := 0; ; i++ {
			if groupCtx.Err() == context.Canceled {
				return
			}
			// Create 3 nodes at once at the beginning.
			if i >= 3 {
				time.Sleep(6 * time.Second)
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := launchNode()
				if err != nil && groupCtx.Err() != context.Canceled {
					slog.Debug("Error creating advertised node.", "error", err)
				}
			}()
		}
	}()

	result := <-resultChan
	cancel()
	wg.Wait()

	go func() {
		for ctx.Err() == nil {
			time.Sleep(6 * time.Second)
			result.GetNode().AdvertiseWAN(ctx, result.GetAdvertiseTopic())
		}
	}()

	return result, nil
}
