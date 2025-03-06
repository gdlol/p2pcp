package integration

import (
	"context"
	"os"
	"path/filepath"
	"project/pkg/workspace"
	"test/internal/docker"
)

const receiverDataPath = "/tmp/p2pcp/integration/receiver/data"

func getTestDataPath() string {
	projectPath := workspace.GetProjectPath()
	return filepath.Join(projectPath, "test/testdata/integration")
}

func sudoResetDir(path string) {
	workspace.Run("sudo", "rm", "--recursive", path)
	err := workspace.Run("mkdir", "--parents", path)
	workspace.Check(err)
}

func setEnv(key, value string) (restore func()) {
	originalValue := os.Getenv(key)
	os.Setenv(key, value)
	return func() {
		os.Setenv(key, originalValue)
	}
}

func runCompose(ctx context.Context, composeFilePath string, testName string) (cleanup func()) {
	docker.ComposeDown(ctx, composeFilePath)
	docker.ComposeUp(ctx, composeFilePath)
	return func() {
		defer docker.ComposeDown(ctx, composeFilePath)
		defer docker.DumpComposeLogs(ctx, composeFilePath, testName)
		defer docker.ComposeStop(ctx, composeFilePath)
	}
}
