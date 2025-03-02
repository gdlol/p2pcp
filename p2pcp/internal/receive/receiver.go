package receive

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/rand"
	"p2pcp/internal/auth"
	"p2pcp/internal/config"
	"p2pcp/internal/node"
	"p2pcp/internal/transfer"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/backoff"
)

type Receiver interface {
	GetNode() node.Node
	FindPeer(ctx context.Context, id string) (peer.ID, error)
	Receive(ctx context.Context, sender peer.ID, secretHash []byte, basePath string) error
}

type receiver struct {
	node node.Node
}

func (r *receiver) GetNode() node.Node {
	return r.node
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
				if len(addrInfo.Addrs) == 0 {
					slog.Warn("Found sender with no addresses.", "sender", addrInfo.ID)
					continue
				}
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
				break
			} else if len(validPeers) > 1 {
				slog.Warn("Found multiple peers advertising topic.", "topic", id, "peers", validPeers)
				return "", fmt.Errorf("found multiple peers advertising topic %s", id)
			}
		}
	}
	return sender, nil
}

func (r *receiver) Receive(ctx context.Context, sender peer.ID, secretHash []byte, basePath string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	n := r.node
	host := n.GetHost()

	for ctx.Err() == nil {
		slog.Debug("Connecting to sender...", "sender", sender)
		addrs := host.Peerstore().Addrs(sender)
		err := host.Connect(ctx, peer.AddrInfo{ID: sender, Addrs: addrs})
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			slog.Debug("Error connecting to sender.", "error", err)
			time.Sleep(time.Second)
			continue
		}
		slog.Info("Connected to sender.", "sender", sender)
		host.ConnManager().Protect(sender, "sender")
		break
	}

	backoffStrategy := backoff.NewExponentialBackoff(
		0, 3*time.Second, backoff.FullJitter,
		100*time.Millisecond, math.Sqrt2, 0,
		rand.NewSource(0))
	getStream := func(protocol protocol.ID) (io.ReadWriteCloser, error) {
		b := backoffStrategy()
		for ctx.Err() == nil {
			stream, err := host.NewStream(ctx, sender, protocol)
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

	authStream, err := getStream(auth.Protocol)
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

	cfg := config.GetConfig()
	channel := transfer.NewChannel(ctx, func(ctx context.Context) (io.ReadWriteCloser, error) {
		return getStream(transfer.Protocol)
	}, int(cfg.PayloadSize))
	defer func() {
		if err := channel.Close(); err != nil {
			slog.Debug("Error closing channel.", "error", err)
		}
	}()

	err = transfer.ReadTar(channel, basePath)
	if err != nil {
		return fmt.Errorf("error receiving tar: %w", err)
	}

	slog.Info("Transfer complete.")
	return nil
}

func NewReceiver(node node.Node) Receiver {
	return &receiver{node: node}
}
