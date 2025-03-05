package integration

import (
	"context"
	"path/filepath"
	"project/pkg/workspace"
)

const receiverDataPath = "/tmp/p2pcp/integration/receiver/data"

func getTestDataPath() string {
	projectPath := workspace.GetProjectPath()
	return filepath.Join(projectPath, "test/testdata/integration")
}

func RunTests(ctx context.Context) {
	TestPublicNetwork(ctx)
}
