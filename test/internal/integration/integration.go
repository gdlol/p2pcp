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
	testPrivateNetwork(ctx)
	testPrivateNetwork_SendDir(ctx)
	testPrivateNetwork_SendFile(ctx)
	testPrivateNetwork_SendFileWithAbsPath(ctx)
	testPrivateNetwork_SendFileWithRelativePath(ctx)
	testPrivateNetwork_SendFileWithRelativePath_Chdir(ctx)
	testPrivateNetwork_SendFile_Confirm(ctx)
	testPublicNetwork(ctx)
}

func sudoResetDir(path string) {
	workspace.Run("sudo", "rm", "--recursive", path)
	err := workspace.Run("mkdir", "--parents", path)
	workspace.Check(err)
}
