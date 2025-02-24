package receive

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/rand"
	"p2pcp/internal/auth"
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
	FindPeer(ctx context.Context, id string) (*peer.AddrInfo, error)
	Receive(ctx context.Context, sender peer.AddrInfo, secretHash []byte, basePath string) error
}

type receiver struct {
	node node.Node
}

func (r *receiver) GetNode() node.Node {
	return r.node
}

func (r *receiver) FindPeer(ctx context.Context, id string) (*peer.AddrInfo, error) {
	if len(id) < 7 {
		panic("Invalid id length.")
	}
	topic := id[len(id)-7:]
	var senderAddrInfo peer.AddrInfo
	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		time.Sleep(3 * time.Second)

		slog.Debug("Finding sender from DHT...")
		peers, err := r.node.FindPeers(ctx, topic)
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
					slog.Warn("Found invalid sender advertising topic.", "topic", topic, "sender", addrInfo)
					continue
				}
				validPeers = append(validPeers, addrInfo)
				break
			}
			if len(validPeers) == 1 {
				senderAddrInfo = validPeers[0]
				slog.Info("Found sender.", "sender", senderAddrInfo)
				break
			} else if len(validPeers) > 1 {
				slog.Warn("Found multiple peers advertising topic.", "topic", topic, "peers", validPeers)
				return nil, fmt.Errorf("found multiple peers advertising topic %s", topic)
			}
		}
	}
	return &senderAddrInfo, nil
}

func (r *receiver) Receive(ctx context.Context, sender peer.AddrInfo, secretHash []byte, basePath string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	n := r.node
	host := n.GetHost()

	for ctx.Err() == nil {
		slog.Debug("Connecting to sender...", "sender", sender)
		err := host.Connect(ctx, sender)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			slog.Debug("Error connecting to sender.", "error", err)
			time.Sleep(time.Second)
			continue
		}
		slog.Info("Connected to sender.", "sender", sender)
		host.ConnManager().Protect(sender.ID, "sender")
		break
	}

	backoffStrategy := backoff.NewExponentialBackoff(
		0, 3*time.Second, backoff.FullJitter,
		100*time.Millisecond, math.Sqrt2, 0,
		rand.NewSource(0))
	getStream := func(protocol protocol.ID) (io.ReadWriteCloser, error) {
		b := backoffStrategy()
		for ctx.Err() == nil {
			stream, err := host.NewStream(ctx, sender.ID, protocol)
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

	channel := transfer.NewChannel(ctx, func() (io.ReadWriteCloser, error) {
		return getStream(transfer.Protocol)
	}, transfer.DefaultPayloadSize)
	defer func() {
		if err := channel.Close(); err != nil {
			slog.Debug("Error closing channel.", "error", err)
		}
	}()

	err = readTar(channel, basePath)
	if err != nil {
		return fmt.Errorf("error receiving tar: %w", err)
	}

	slog.Info("Transfer complete.")
	return nil
}

func NewReceiver(node node.Node) Receiver {
	return &receiver{node: node}
}
