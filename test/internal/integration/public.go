package integration

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"project/pkg/workspace"
	"test/internal/docker"
	"test/pkg/asserts"
	"time"
)

func TestPublicNetwork(ctx context.Context) {
	logger := slog.With("test", "TestPublicNetwork")
	logger.Info("Starting test...")

	workspace.ResetDir(receiverDataPath)
	composeFilePath := filepath.Join(getTestDataPath(), "public_network/compose.yaml")
	defer docker.ComposeCleanup(ctx, composeFilePath)
	docker.ComposeUp(ctx, composeFilePath)

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	logger.Info("Waiting for receiver to exit...")
	err := docker.WaitContainer(ctx, "receiver")
	workspace.Check(err)

	logger.Info("Waiting for sender to exit...")
	err = docker.WaitContainer(ctx, "sender")
	workspace.Check(err)

	expected := filepath.Join(os.TempDir(), "p2pcp/test/integration/public_network/data")
	workspace.ResetDir(expected)

	// Sender sends empty /data (${PWD}) to receiver, receiver gets empty /data/data (${PWD}/data)
	asserts.AreDirsEqual(expected, filepath.Join(receiverDataPath, "data"))
	logger.Info("Data directories are equal.")
}
