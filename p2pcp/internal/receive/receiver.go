package receive

import (
	"archive/tar"
	"context"
	"io"
	"log/slog"
	"os"
	"p2pcp/internal/node"
	"path/filepath"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	progress "github.com/schollz/progressbar/v3"
)

type Receiver interface {
	GetNode() node.Node
	FindPeer(ctx context.Context, topic string) (*peer.AddrInfo, error)
	Receive(ctx context.Context, peer peer.AddrInfo, path string) error
}

type receiver struct {
	node node.Node
}

func (r *receiver) GetNode() node.Node {
	return r.node
}

func (r *receiver) FindPeer(ctx context.Context, topic string) (*peer.AddrInfo, error) {
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
			found := false
			for addrInfo := range peers {
				if addrInfo.ID.Validate() != nil || len(addrInfo.Addrs) == 0 {
					slog.Warn("Found sender with invalid ID or no addresses.", "sender", addrInfo)
					continue
				}
				nodeID, err := node.GetNodeID(addrInfo.ID)
				if err != nil {
					slog.Warn("Error getting node ID.", "sender", addrInfo)
					continue
				}
				if !strings.HasSuffix(nodeID.String(), topic) {
					slog.Warn("Found invalid sender advertising topic.", "topic", topic, "sender", addrInfo)
					continue
				}
				found = true
				senderAddrInfo = addrInfo
				break
			}
			if found {
				slog.Info("Found listener.", "listener", senderAddrInfo)
				break
			}
		}
	}
	return &senderAddrInfo, nil
}

func handleFile(header *tar.Header, reader io.Reader, basePath string) error {
	fileInfo := header.FileInfo()
	joined := filepath.Join(basePath, header.Name)
	if fileInfo.IsDir() {
		err := os.MkdirAll(joined, fileInfo.Mode())
		if err != nil {
			slog.Error("error creating directory", "path", joined, "err", err)
			return err
		}
		return nil
	}

	newFile, err := os.OpenFile(joined, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fileInfo.Mode().Perm())
	if err != nil {
		slog.Error("error creating file", "path", joined, "err", err)
		return err
	}

	bar := progress.DefaultBytes(header.Size, filepath.Base(header.Name))
	_, err = io.Copy(io.MultiWriter(newFile, bar), reader)
	if err != nil {
		slog.Error("error writing file content", "path", joined, "err", err)
		return err
	}
	return nil
}

func (r *receiver) Receive(ctx context.Context, peer peer.AddrInfo, basePath string) error {
	node := r.node
	host := node.GetHost()

	slog.Debug("Connecting to sender...", "sender", peer)
	err := host.Connect(ctx, peer)
	if err != nil {
		slog.Error("Error connecting to sender.", "error", err)
		return err
	}
	slog.Info("Connected to sender.", "sender", peer)

	stream, err := host.NewStream(ctx, peer.ID, node.GetProtocol())
	if err != nil {
		slog.Error("Error creating stream", "error", err)
		return err
	}
	defer stream.Close()

	reader := tar.NewReader(stream)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break // End of archive
		} else if err != nil {
			slog.Error("Error reading next tar element", "err", err)
			return err
		}
		err = handleFile(header, reader, basePath)
		if err != nil {
			return err
		}
	}

	stream.CloseWrite()
	return nil
}

func NewReceiver(node node.Node) Receiver {
	return &receiver{node: node}
}
