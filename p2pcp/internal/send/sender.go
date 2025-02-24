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

	var authenticatedPeer *peer.ID = nil
	authenticate := make(chan *peer.ID, 1)
	host.SetStreamHandler(auth.Protocol, func(stream network.Stream) {
		slog.Debug("Received new auth stream.")
		remotePeer := stream.Conn().RemotePeer()
		if authenticatedPeer == nil {
			success, err := auth.HandleAuthenticate(stream, secretHash)
			if err != nil {
				slog.Warn("Error authenticating receiver.", "error", err)
			}
			if success == nil {
				return
			}
			if *success {
				if err == nil {
					select {
					case authenticate <- &remotePeer:
					default:
					}
				}
			} else {
				if !strict {
					select {
					case authenticate <- nil: // Causes abort if not in strict mode.
					default:
					}
				}
			}
		} else {
			slog.Warn("Received extra auth stream.")
			stream.Close()
		}
	})

	authenticatedPeer = <-authenticate
	host.RemoveStreamHandler(auth.Protocol)
	if authenticatedPeer == nil {
		cancel()
		return fmt.Errorf("failed to authenticate receiver")
	}
	slog.Info("Authenticated receiver.", "peer", authenticatedPeer)

	streams := make(chan io.ReadWriteCloser, 1)
	channel := transfer.NewChannel(ctx, func() (io.ReadWriteCloser, error) {
		select {
		case stream := <-streams:
			return stream, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}, transfer.DefaultPayloadSize)
	defer func() {
		if err := channel.Close(); err != nil {
			slog.Debug("Error closing channel.", "error", err)
		}
	}()
	host.SetStreamHandler(transfer.Protocol, func(stream network.Stream) {
		slog.Debug("Received new transfer stream.")
		remotePeer := stream.Conn().RemotePeer()
		if authenticatedPeer == nil || *authenticatedPeer != remotePeer {
			slog.Warn("Unauthorized transfer stream.")
			stream.Close()
		} else {
			select {
			case streams <- stream:
			case <-ctx.Done():
				stream.Close()
			}
		}
	})
	defer host.RemoveStreamHandler(transfer.Protocol)

	err := writeTar(channel, path)
	if err != nil {
		return fmt.Errorf("error sending path %s: %w", path, err)
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
			if groupCtx.Err() != nil {
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
