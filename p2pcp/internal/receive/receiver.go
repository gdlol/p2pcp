package receive

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/rand"
	"p2pcp/internal/node"
	"p2pcp/internal/transfer"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
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
				if addrInfo.ID.Validate() != nil {
					slog.Warn("Found sender with invalid ID.", "sender", addrInfo.ID)
					continue
				}
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
	authenticated := false
	authenticate := make(chan bool, 1)

	for ctx.Err() == nil {
		slog.Debug("Connecting to sender...", "sender", sender)
		err := host.Connect(ctx, sender)
		if err != nil {
			slog.Debug("Error connecting to sender.", "error", err)
			time.Sleep(time.Second)
			continue
		}
		slog.Debug("Connected to sender.", "sender", sender)
		host.ConnManager().Protect(sender.ID, "sender")
		break
	}

	streams := make(chan transfer.ChannelStream)
	channel := transfer.NewChannel(ctx, streams, transfer.DefaultPayloadSize)
	defer channel.Close()

	newStreamCtx, cancelNewStream := context.WithCancel(ctx)
	defer cancelNewStream()
	go func() {
		b := backoff.NewExponentialBackoff(
			0, 3*time.Second, backoff.FullJitter,
			100*time.Millisecond, math.Sqrt2, -100*time.Millisecond,
			rand.NewSource(0))()
		for newStreamCtx.Err() == nil {
			stream, err := host.NewStream(newStreamCtx, sender.ID, node.Protocol)
			if err != nil {
				slog.Debug("Error creating stream", "error", err)
				time.Sleep(b.Delay())
				continue
			}
			b.Reset()

			if !authenticated {
				defer stream.Close()

				_, err = stream.Write(secretHash)
				if err != nil {
					slog.Error("Error writing secret hash.", "error", err)
					authenticate <- false
					return
				}
				buffer := make([]byte, 1)
				_, err = io.ReadFull(stream, buffer)
				if err != nil {
					slog.Error("Error reading authentication response.", "error", err)
					authenticate <- false
					return
				}
				if buffer[0] == 0 {
					slog.Debug("Authentication failed.")
					authenticate <- false
					return
				}

				authenticated = true
				authenticate <- true
				continue
			}

			channelStream := transfer.NewChannelStream(stream)
			select {
			case streams <- *channelStream:
				<-channelStream.Done
			case <-newStreamCtx.Done():
			}
		}
	}()

	if !<-authenticate {
		cancel()
		return fmt.Errorf("authentication failed")
	}

	pipeReader, pipeWriter := io.Pipe()
	done := make(chan struct{}, 1)
	go func() {
		defer pipeWriter.Close()
		buffer := make([]byte, transfer.DefaultPayloadSize)
		for ctx.Err() == nil {
			n, err := channel.Read(buffer)
			if err != nil {
				if err != io.EOF {
					slog.Debug("Error reading from channel.", "error", err)
				}
				break
			}
			if n > 0 {
				_, err = pipeWriter.Write(buffer[:n])
				if err != nil {
					slog.Debug("Error writing to pipe.", "error", err)
					break
				}
			}
			if err == io.EOF {
				slog.Debug("Read EOF from channel.")
				break
			}
		}
		done <- struct{}{}
	}()

	reader := tar.NewReader(pipeReader)
	err := readTar(reader, basePath)
	if err != nil {
		return fmt.Errorf("error receiving tar: %w", err)
	}

	// Drain padding
	buffer := make([]byte, 8192)
	for ctx.Err() != nil {
		_, err := pipeReader.Read(buffer)
		if err != nil {
			break
		}
	}

	<-done
	if err = channel.Close(); err != nil {
		slog.Debug("Error closing channel.", "error", err)
	}
	slog.Info("Transfer complete.")
	return nil
}

func NewReceiver(node node.Node) Receiver {
	return &receiver{node: node}
}
