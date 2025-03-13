package receive

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/rand"
	"p2pcp/internal/auth"
	"p2pcp/internal/interrupt"
	"p2pcp/internal/node"
	"p2pcp/internal/transfer"
	"p2pcp/internal/transfer/channel"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/backoff"
)

type Receiver interface {
	FindPeer(ctx context.Context, id string) (peer.ID, error)
	Receive(ctx context.Context, sender peer.ID, secretHash []byte, basePath string) error
}

type receiver struct {
	node node.Node
}

func (r *receiver) FindPeer(ctx context.Context, id string) (peer.ID, error) {
	if !r.node.IsPrivateMode() {
		slog.Debug("Waiting for WAN connection...")
		err := r.node.WaitForWAN(ctx)
		if err != nil {
			return "", err
		}
		slog.Debug("Connected to WAN.")
	}

	var sender peer.ID
	for {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		time.Sleep(1 * time.Second)

		slog.Debug("Finding sender from DHT...")
		peers, err := r.node.FindPeers(ctx, id)
		if err != nil {
			slog.Debug("Error finding sender from DHT, retrying...", "error", err)
		} else {
			validPeers := []peer.AddrInfo{}
			for addrInfo := range peers {
				nodeID, err := node.GetNodeID(addrInfo.ID)
				if err != nil {
					slog.Warn("Error getting node ID.", "sender", addrInfo)
					continue
				}
				if !strings.HasSuffix(nodeID.String(), id) {
					slog.Warn("Found invalid sender advertising topic.", "topic", id, "sender", addrInfo)
					continue
				}
				validPeers = append(validPeers, addrInfo)
				break
			}
			if len(validPeers) == 1 {
				sender = validPeers[0].ID
				slog.Info("Found sender.", "sender", sender)
				// Mark sender as candidate for DHT routing.
				r.node.GetHost().Peerstore().Put(sender, node.DhtRoutingTag, struct{}{})
				break
			} else if len(validPeers) > 1 {
				slog.Warn("Found multiple peers advertising topic.", "topic", id, "peers", validPeers)
				return "", fmt.Errorf("found multiple peers advertising topic %s", id)
			}
		}
	}
	return sender, nil
}

func connectToSender(ctx context.Context, host host.Host, peerID peer.ID) error {
	for ctx.Err() == nil {
		slog.Debug("Connecting to sender...", "sender", peerID)
		addrs := host.Peerstore().Addrs(peerID)
		err := host.Connect(ctx, peer.AddrInfo{ID: peerID, Addrs: addrs})
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			slog.Debug("Error connecting to sender.", "error", err)
			time.Sleep(time.Second)
			continue
		}
		slog.Info("Connected to sender.", "sender", peerID)
		host.ConnManager().Protect(peerID, "sender")
		break
	}
	return ctx.Err()
}

func getStream(ctx context.Context, host host.Host, peerID peer.ID, protocol protocol.ID) (io.ReadWriteCloser, error) {
	b := backoff.NewExponentialBackoff(
		0, 3*time.Second, backoff.FullJitter,
		100*time.Millisecond, math.Sqrt2, 0,
		rand.NewSource(0))()
	for ctx.Err() == nil {
		stream, err := host.NewStream(ctx, peerID, protocol)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			slog.Debug("Error creating stream", "error", err)
			time.Sleep(b.Delay())
			continue
		}
		return stream, nil
	}
	return nil, ctx.Err()
}

func (r *receiver) Receive(ctx context.Context, sender peer.ID, secretHash []byte, basePath string) (err error) {
	n := r.node
	host := n.GetHost()

	err = connectToSender(ctx, host, sender)
	if err != nil {
		return err
	}

	authStream, err := getStream(ctx, host, sender, auth.Protocol)
	if err != nil {
		return fmt.Errorf("error creating auth stream: %w", err)
	} else {
		success, err := auth.Authenticate(authStream, secretHash)
		if err != nil {
			slog.Error("Error authenticating.", "error", err)
		}
		if !success {
			return fmt.Errorf("authentication failed")
		}
		slog.Info("Authenticated.")
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	n.RegisterErrorHandler(sender, func(errStr string) {
		slog.Error("Sender error", "error", errStr)
		cancel()
	})
	canceling := false
	interrupt.RegisterInterruptHandler(ctx, func() {
		canceling = true
		n.SendError(ctx, sender, "Transfer canceled.")
		cancel()
	})

	reader := channel.NewChannelReader(ctx, func(ctx context.Context) (io.ReadWriteCloser, error) {
		if canceling {
			<-ctx.Done()
			return nil, ctx.Err()
		} else {
			return getStream(ctx, host, sender, transfer.Protocol)
		}
	})
	defer func() {
		if err := reader.Close(); err != nil {
			slog.Debug("Error closing channel.", "error", err)
		}
	}()

	err = transfer.ReadZip(reader, basePath)
	if err != nil {
		if ctx.Err() == nil {
			n.SendError(context.Background(), sender, "")
			cancel()
		}
		return fmt.Errorf("error receiving zip: %w", err)
	}

	slog.Info("Transfer complete.")
	return nil
}

func NewReceiver(node node.Node) Receiver {
	return &receiver{node: node}
}
