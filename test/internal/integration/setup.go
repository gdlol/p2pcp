package integration

import (
	"context"
	"crypto/rand"
	"os"
	"path/filepath"
	"project/pkg/workspace"
	"test/internal/docker"
)

const senderDataPath = "/tmp/p2pcp/integration/sender/data"
const receiverDataPath = "/tmp/p2pcp/integration/receiver/data"

func getTestDataPath() string {
	projectPath := workspace.GetProjectPath()
	return filepath.Join(projectPath, "test/testdata/integration")
}

func sudoResetDir(path string) {
	if _, err := os.Lstat(path); err == nil {
		workspace.Run("sudo", "rm", "--recursive", path)
	}
	workspace.Run("mkdir", "--parents", path)
}

func setEnv(key, value string) (restore func()) {
	originalValue := os.Getenv(key)
	os.Setenv(key, value)
	return func() {
		os.Setenv(key, originalValue)
	}
}

func generateFile(path string, size int64) {
	err := os.MkdirAll(filepath.Dir(path), 0755)
	workspace.Check(err)
	file, err := os.Create(path)
	workspace.Check(err)
	defer file.Close()

	buf := make([]byte, 1024)
	for size > 0 {
		n, err := rand.Read(buf)
		workspace.Check(err)
		if size < int64(n) {
			n = int(size)
		}
		_, err = file.Write(buf[:n])
		workspace.Check(err)
		size -= int64(n)
	}
}

func cleanupCompose(ctx context.Context, composeFilePath string, testName string) {
	defer docker.ComposeDown(ctx, composeFilePath)
	defer docker.DumpComposeLogs(ctx, composeFilePath, testName)
	defer docker.ComposeCollectCoverage(ctx)
	defer docker.ComposeStop(ctx, composeFilePath)
}

func runCompose(ctx context.Context, composeFilePath string, testName string) (cleanup func()) {
	docker.ComposeUp(ctx, composeFilePath)
	return func() {
		cleanupCompose(ctx, composeFilePath, testName)
	}
}

func cleanup() {
	ctx := context.Background()
	docker.ComposeDown(ctx, filepath.Join(getTestDataPath(), "public_network/compose.yaml"))
	docker.ComposeDown(ctx, filepath.Join(getTestDataPath(), "private_network/compose.yaml"))
	docker.ComposeDown(ctx, filepath.Join(getTestDataPath(), "relay_network/compose.yaml"))
	workspace.ResetDir(senderDataPath)
	sudoResetDir(receiverDataPath)
}
