package send

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"p2pcp/internal/auth"
	"p2pcp/internal/errors"
	"p2pcp/internal/interrupt"
	"p2pcp/internal/node"
	"p2pcp/internal/transfer"
	"p2pcp/internal/transfer/channel"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoremem"
)

type Sender interface {
	GetNode() node.Node
	GetAdvertiseTopic() string
	WaitForReceiver(ctx context.Context, secretHash []byte) (peer.ID, error)
	Send(ctx context.Context, receiver peer.ID, basePath string) error
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

func authenticateReceiver(ctx context.Context, host host.Host, secretHash []byte, strict bool) (peer.ID, error) {
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
						host.ConnManager().Protect(remotePeer, "receiver")
						// Mark receiver as candidate for DHT routing.
						host.Peerstore().Put(remotePeer, node.DhtRoutingTag, struct{}{})
					default:
					}
				}
			} else {
				if !strict {
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

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case authenticatedPeer = <-authenticate:
		host.RemoveStreamHandler(auth.Protocol)
		if authenticatedPeer == "" {
			return "", fmt.Errorf("failed to authenticate receiver")
		} else {
			return authenticatedPeer, nil
		}
	}
}

func (s *sender) WaitForReceiver(ctx context.Context, secretHash []byte) (peer.ID, error) {
	return authenticateReceiver(ctx, s.node.GetHost(), secretHash, s.strictMode)
}

func getAuthorizedStreams(host host.Host, receiver peer.ID) (chan io.ReadWriteCloser, func()) {
	streams := make(chan io.ReadWriteCloser, 1)
	cancel := func() {
		host.RemoveStreamHandler(transfer.Protocol)
	}
	host.SetStreamHandler(transfer.Protocol, func(stream network.Stream) {
		slog.Debug("Received new transfer stream.")
		remotePeer := stream.Conn().RemotePeer()
		if receiver != remotePeer {
			slog.Warn("Unauthorized transfer stream.")
			stream.Close()
		} else {
			streams <- stream
		}
	})
	return streams, cancel
}

func (s *sender) Send(ctx context.Context, receiver peer.ID, basePath string) (err error) {
	n := s.node
	host := n.GetHost()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	n.RegisterErrorHandler(receiver, func(errStr string) {
		slog.Error("Receiver error", "error", errStr)
		cancel()
	})
	streams, cancelStreams := getAuthorizedStreams(host, receiver)
	interrupt.RegisterInterruptHandler(ctx, func() {
		cancelStreams()
		n.SendError(ctx, receiver, "Transfer canceled.")
		cancel()
	})

	writer := channel.NewChannelWriter(ctx, func(ctx context.Context) (io.ReadWriteCloser, error) {
		select {
		case stream := <-streams:
			return stream, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	})
	defer func() {
		if err := writer.Close(); err != nil {
			slog.Debug("Error closing channel.", "error", err)
		}
	}()

	err = transfer.WriteZip(writer, basePath)
	if err == nil {
		err = writer.Flush(true)
	}
	if err != nil {
		n.SendError(ctx, receiver, "")
		cancel()
		return fmt.Errorf("error sending path %s: %w", basePath, err)
	}

	slog.Info("Transfer complete.")
	return nil
}

func advertiseToWAN(sender Sender, ctx context.Context) error {
	node := sender.GetNode()
	topic := sender.GetAdvertiseTopic()

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

func newSender(ctx context.Context, strictMode bool, privateMode bool, options ...libp2p.Option) Sender {
	node := node.NewNode(ctx, privateMode, options...)
	return &sender{node: node, strictMode: strictMode}
}

func NewAdvertisedSender(ctx context.Context, strictMode bool, privateMode bool) (Sender, error) {
	var sender Sender
	if privateMode {
		sender = newSender(ctx, strictMode, privateMode)
	} else {
		// Create new sender every 6 seconds. until 1 successfully advertised itself to WAN DHT.
		groupCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		ps, err := pstoremem.NewPeerstore()
		errors.Unexpected(err, "create peerstore")

		resultChan := make(chan Sender, 1)
		var wg sync.WaitGroup

		// Try advertising up to 1 minute.
		launchNode := func() error {
			if groupCtx.Err() != nil {
				return groupCtx.Err()
			}

			candidate := newSender(ctx, strictMode, privateMode, libp2p.Peerstore(ps))
			success := false
			defer func() {
				if !success {
					candidate.Close()
				}
			}()

			timeoutCtx, cancel := context.WithTimeout(groupCtx, time.Minute)
			defer cancel()
			err := advertiseToWAN(candidate, timeoutCtx)
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
			for i := 0; groupCtx.Err() == nil; i++ {
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

		select {
		case sender = <-resultChan:
		case <-ctx.Done():
			return nil, ctx.Err()
		}

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

	return sender, ctx.Err()
}
