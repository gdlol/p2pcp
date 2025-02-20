package send

import (
	"archive/tar"
	"context"
	"io"
	"log/slog"
	"os"
	"p2pcp/internal/node"
	"path/filepath"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/pkg/errors"
	progress "github.com/schollz/progressbar/v3"
)

type Sender interface {
	GetAdvertiseTopic() string
	StartAdvertise(ctx context.Context)
	Send(path string) error
}

type sender struct {
	node node.Node
}

func NewSender(node node.Node) Sender {
	return &sender{node: node}
}

func (s *sender) GetAdvertiseTopic() string {
	id := s.node.GetHost().ID().String()
	return id[len(id)-7:]
}

func (s *sender) StartAdvertise(ctx context.Context) {
	node := s.node
	discovery := node.GetDiscovery()
	topic := s.GetAdvertiseTopic()

	// Advertise self to DHT periodically.
	go func() {
		for {
			slog.Debug("Advertising to DHT...")
			_, err := discovery.Advertise(ctx, topic)
			if err != nil {
				slog.Warn("Error advertising to DHT.", "error", err)
			} else {
				slog.Debug("Advertised to DHT.")
				time.Sleep(60 * time.Second)
			}
			time.Sleep(3 * time.Second)
		}
	}()
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
		slog.Debug("Preparing file/dir for transmission", "path", path)
		if err != nil {
			slog.Debug("Error walking file", "path", path, "err", err)
			return err
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return errors.Wrapf(err, "error writing tar file info header %s: %s", path, err)
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
	})
	defer host.RemoveStreamHandler(node.GetProtocol())

	return <-result
}
