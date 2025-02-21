package send

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"p2pcp/internal/node"
	"path/filepath"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoremem"
	"github.com/pkg/errors"
	progress "github.com/schollz/progressbar/v3"
)

type Sender interface {
	GetNode() node.Node
	GetAdvertiseTopic() string
	AdvertiseWAN(ctx context.Context)
	Send(path string) error
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

func (s *sender) AdvertiseWAN(ctx context.Context) {
	node := s.node
	topic := s.GetAdvertiseTopic()

	// Advertise self to DHT until success/cancel.
	for {
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

// Builds the path structure for the tar archive - this will be the structure as it is received.
func relativePath(basePath string, baseIsDir bool, targetPath string) (string, error) {
	if baseIsDir {
		rel, err := filepath.Rel(basePath, targetPath)
		if err != nil {
			return "", err
		}
		return filepath.Clean(filepath.Join(filepath.Base(basePath), rel)), nil
	} else {
		return filepath.Base(basePath), nil
	}
}

func transfer(s network.Stream, root string) error {
	rootInfo, err := os.Stat(root)
	if err != nil {
		return err
	}

	writer := tar.NewWriter(s)
	defer writer.Close()
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		slog.Debug("Preparing file/dir for transmission.", "path", path)
		if err != nil {
			slog.Debug("Error walking file.", "path", path, "err", err)
			return err
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return errors.Wrapf(err, "error writing tar file info header %s: %s.", path, err)
		}

		// To preserve directory structure in the tar ball.
		header.Name, err = relativePath(root, rootInfo.IsDir(), path)
		if err != nil {
			return errors.Wrapf(err, "error building relative path: %s (IsDir: %v) %s", root, rootInfo.IsDir(), path)
		}

		if err = writer.WriteHeader(header); err != nil {
			return errors.Wrap(err, "error writing tar header")
		}

		// Continue as all information was written above with WriteHeader.
		if info.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return errors.Wrapf(err, "error opening file for taring at: %s", path)
		}
		defer f.Close()

		bar := progress.DefaultBytes(info.Size(), info.Name())
		if _, err = io.Copy(io.MultiWriter(writer, bar), f); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	if err = writer.Close(); err != nil {
		slog.Warn("Error closing tar ball", "err", err)
	}

	return nil
}

func (s *sender) Send(path string) error {
	node := s.node
	host := node.GetHost()

	result := make(chan error)
	defer close(result)
	host.SetStreamHandler(node.GetProtocol(), func(s network.Stream) {
		defer s.Close()
		result <- transfer(s, path)
		if result == nil {
			buffer := make([]byte, 1024)
			for {
				_, err := s.Read(buffer)
				if err != nil {
					if err == io.EOF {
						slog.Info("Transfer complete.")
					} else {
						slog.Warn("Error reading from stream.", "error", err)
					}
					break
				}
			}
		}
	})
	defer host.RemoveStreamHandler(node.GetProtocol())

	return <-result
}

func NewSender(node node.Node) Sender {
	return &sender{node: node}
}

// Create new sender every 6 seconds. until 1 successfully advertised itself to WAN DHT.
func NewAdvertisedSender(ctx context.Context) (Sender, error) {
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

		n, err := node.NewNode(ctx, libp2p.Peerstore(ps))
		if err != nil {
			return err
		}
		success := false
		defer func() {
			if !success {
				n.Close()
			}
		}()
		sender := NewSender(n)

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
	return result, nil
}
