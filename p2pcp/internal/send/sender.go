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
	Send(ctx context.Context, secretHash []byte, path string) error
	Close()
}

type sender struct {
	node       node.Node
	strictMode bool
}

func (s *sender) GetNode() node.Node {
	return s.node
}

func (s *sender) GetAdvertiseTopic() string {
	id := s.node.ID().String()
	if s.strictMode {
		return id
	} else {
		return id[len(id)-7:]
	}
}

func (s *sender) Close() {
	s.node.Close()
}

func (s *sender) Send(ctx context.Context, secretHash []byte, path string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	n := s.node
	host := n.GetHost()

	var authenticatedPeer peer.ID = ""
	authenticate := make(chan peer.ID, 1)
	host.SetStreamHandler(auth.Protocol, func(stream network.Stream) {
		slog.Debug("Received new auth stream.")
		remotePeer := stream.Conn().RemotePeer()
		if authenticatedPeer == "" {
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
					case authenticate <- remotePeer:
					default:
					}
				}
			} else {
				if !s.strictMode {
					select {
					case authenticate <- "": // Causes abort if not in strict mode.
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
	if authenticatedPeer == "" {
		cancel()
		return fmt.Errorf("failed to authenticate receiver")
	}
	slog.Info("Authenticated receiver.", "peer", authenticatedPeer)

	streams := make(chan io.ReadWriteCloser, 1)
	channel := transfer.NewChannel(ctx, func(ctx context.Context) (io.ReadWriteCloser, error) {
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
		if authenticatedPeer != remotePeer {
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

	err := transfer.WriteTar(channel, path)
	if err != nil {
		return fmt.Errorf("error sending path %s: %w", path, err)
	}

	slog.Info("Transfer complete.")
	return nil
}

func advertiseToWAN(sender Sender, ctx context.Context) error {
	node := sender.GetNode()
	topic := sender.GetAdvertiseTopic()

	slog.Debug("Waiting for WAN connection...")
	err := node.WaitForWAN(ctx)
	if err != nil {
		return fmt.Errorf("error waiting for WAN connection: %w", err)
	}
	slog.Debug("Connected to WAN.")

	// Advertise self to DHT until success/cancel.
	for ctx.Err() == nil {
		time.Sleep(3 * time.Second)
		slog.Debug("Advertising to WAN DHT...", "topic", topic)
		err := node.AdvertiseWAN(ctx, topic)
		if ctx.Err() != nil {
			break
		}
		if err != nil {
			slog.Debug("Error advertising to WAN DHT, retrying...", "error", err)
		} else {
			slog.Debug("Advertised to WAN DHT.")
			break
		}
	}
	return ctx.Err()
}

func newSender(ctx context.Context, strictMode bool, privateMode bool, options ...libp2p.Option) (Sender, error) {
	node, err := node.NewNode(ctx, privateMode, options...)
	if err != nil {
		return nil, fmt.Errorf("error creating sender: %w", err)
	}
	return &sender{node: node, strictMode: strictMode}, nil
}

func NewAdvertisedSender(ctx context.Context, strictMode bool, privateMode bool) (Sender, error) {
	var sender Sender
	var err error
	if privateMode {
		sender, err = newSender(ctx, strictMode, privateMode)
	} else {
		// Create new sender every 6 seconds. until 1 successfully advertised itself to WAN DHT.
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

			candidate, err := newSender(ctx, strictMode, privateMode, libp2p.Peerstore(ps))
			if err != nil {
				return err
			}
			success := false
			defer func() {
				if !success {
					candidate.Close()
				}
			}()

			timeoutCtx, cancel := context.WithTimeout(groupCtx, time.Minute)
			defer cancel()
			err = advertiseToWAN(candidate, timeoutCtx)
			if err != nil {
				return err
			}

			select {
			case resultChan <- candidate:
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

		sender = <-resultChan
		cancel()
		wg.Wait()

		go func() {
			for ctx.Err() == nil {
				time.Sleep(6 * time.Second)
				sender.GetNode().AdvertiseWAN(ctx, sender.GetAdvertiseTopic())
			}
		}()
	}

	go func() {
		for ctx.Err() == nil {
			time.Sleep(3 * time.Second)
			sender.GetNode().AdvertiseLAN(ctx, sender.GetAdvertiseTopic())
		}
	}()

	return sender, err
}
