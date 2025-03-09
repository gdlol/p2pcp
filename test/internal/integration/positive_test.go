package integration

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"project/pkg/workspace"
	"runtime"
	"strings"
	"test/internal/docker"
	"test/pkg/asserts"
	"testing"

	"github.com/stretchr/testify/assert"
)

func runTestPositive(ctx context.Context, composeFilePath string, assertions func()) {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		panic("Failed to get caller info.")
	}
	testName := filepath.Base(runtime.FuncForPC(pc).Name())
	testName = testName[strings.LastIndex(testName, ".")+1:]
	logger := slog.With("test", testName)
	logger.Info("Starting test...")

	cleanup := runCompose(ctx, composeFilePath, testName)
	defer cleanup()

	logger.Info("Waiting for receiver to exit...")
	err := docker.WaitContainer(ctx, "receiver")
	workspace.Check(err)
	docker.AssertContainerLogContains(ctx, "receiver", "Done.")

	logger.Info("Waiting for sender to exit...")
	err = docker.WaitContainer(ctx, "sender")
	workspace.Check(err)
	docker.AssertContainerLogContains(ctx, "sender", "Sending...")
	docker.AssertContainerLogContains(ctx, "sender", "Done.")

	assertions()
}

func TestPrivateNetwork(t *testing.T) {
	t.Cleanup(cleanup)

	// Sender sends empty /data (${PWD}) to receiver, receiver gets empty /data/data (${PWD}/data)
	expectedPath := filepath.Join(os.TempDir(), "p2pcp/test/integration/public_network/data")
	workspace.ResetDir(expectedPath)
	receiverPath := filepath.Join(receiverDataPath, "data")

	composeFilePath := filepath.Join(getTestDataPath(), "private_network/compose.yaml")
	runTestPositive(t.Context(), composeFilePath, func() {
		asserts.AssertDirsEqual(expectedPath, receiverPath)
	})
}

func TestPrivateNetwork_SendDir(t *testing.T) {
	t.Cleanup(cleanup)

	restore := workspace.SetEnv("SENDER_ARGS", "send /testdata/transfer_file_with_subdir/subdir --strict --private")
	defer restore()

	expectedPath := filepath.Join(workspace.GetTestDataPath(), "transfer_file_with_subdir/subdir")
	receiverPath := filepath.Join(receiverDataPath, "subdir")

	composeFilePath := filepath.Join(getTestDataPath(), "private_network/compose.yaml")
	runTestPositive(t.Context(), composeFilePath, func() {
		asserts.AssertDirsEqual(expectedPath, receiverPath)
	})
}

func TestPrivateNetwork_SendDir_OverwriteDir(t *testing.T) {
	t.Cleanup(cleanup)

	restore := workspace.SetEnv("SENDER_ARGS", "send /testdata/transfer_file_with_subdir/subdir --strict --private")
	defer restore()

	expectedPath := filepath.Join(workspace.GetTestDataPath(), "transfer_file_with_subdir/subdir")
	receiverPath := filepath.Join(receiverDataPath, "subdir")
	err := os.MkdirAll(receiverPath, 0755)
	workspace.Check(err)

	composeFilePath := filepath.Join(getTestDataPath(), "private_network/compose.yaml")
	runTestPositive(t.Context(), composeFilePath, func() {
		asserts.AssertDirsEqual(expectedPath, receiverPath)
	})
}

func TestPrivateNetwork_SendDir_OverwriteDirWithFile(t *testing.T) {
	t.Cleanup(cleanup)

	restore := workspace.SetEnv("SENDER_ARGS", "send /testdata/transfer_file_with_subdir/subdir --strict --private")
	defer restore()

	expectedPath := filepath.Join(workspace.GetTestDataPath(), "transfer_file_with_subdir/subdir")
	receiverPath := filepath.Join(receiverDataPath, "subdir")
	err := os.MkdirAll(receiverPath, 0755)
	workspace.Check(err)
	generateFile(filepath.Join(receiverPath, "extra_file"), 1024)

	composeFilePath := filepath.Join(getTestDataPath(), "private_network/compose.yaml")
	runTestPositive(t.Context(), composeFilePath, func() {
		assert.False(t, asserts.AreDirsEqual(expectedPath, receiverPath))
	})
}

func TestPrivateNetwork_SendDir_OverwriteLink(t *testing.T) {
	t.Cleanup(cleanup)

	expectedPath := filepath.Join(senderDataPath, "dir")
	receiverPath := filepath.Join(receiverDataPath, "data/dir")
	err := os.MkdirAll(expectedPath, 0755)
	workspace.Check(err)
	generateFile(filepath.Join(expectedPath, "file"), 1024)
	generateFile(filepath.Join(receiverPath, "file"), 1024)
	err = os.Symlink("./file", filepath.Join(expectedPath, "link"))
	workspace.Check(err)
	err = os.Symlink("..", filepath.Join(receiverPath, "link"))
	workspace.Check(err)

	composeFilePath := filepath.Join(getTestDataPath(), "private_network/compose.yaml")
	runTestPositive(t.Context(), composeFilePath, func() {
		asserts.AssertFilesEqual(filepath.Join(expectedPath, "file"), filepath.Join(receiverPath, "file"))
		destination, err := os.Readlink(filepath.Join(receiverPath, "link"))
		workspace.Check(err)
		assert.Equal(t, "file", destination)
	})
}

func TestPrivateNetwork_SendFile(t *testing.T) {
	t.Cleanup(cleanup)

	restore := workspace.SetEnv("SENDER_ARGS", "send /testdata/transfer_file_with_subdir/file --strict --private")
	defer restore()

	expectedPath := filepath.Join(workspace.GetTestDataPath(), "transfer_file_with_subdir/file")
	receiverPath := filepath.Join(receiverDataPath, "file")

	composeFilePath := filepath.Join(getTestDataPath(), "private_network/compose.yaml")
	runTestPositive(t.Context(), composeFilePath, func() {
		asserts.AssertFilesEqual(expectedPath, receiverPath)
	})
}

func TestPrivateNetwork_SendFile_Overwrite(t *testing.T) {
	t.Cleanup(cleanup)

	restore := workspace.SetEnv("SENDER_ARGS", "send /testdata/transfer_file_with_subdir/file --strict --private")
	defer restore()

	expectedPath := filepath.Join(workspace.GetTestDataPath(), "transfer_file_with_subdir/file")
	receiverPath := filepath.Join(receiverDataPath, "file")
	generateFile(receiverPath, 1024)

	composeFilePath := filepath.Join(getTestDataPath(), "private_network/compose.yaml")
	runTestPositive(t.Context(), composeFilePath, func() {
		asserts.AssertFilesEqual(expectedPath, receiverPath)
	})
}

func TestPrivateNetwork_SendFileWithAbsPath(t *testing.T) {
	t.Cleanup(cleanup)

	restoreSenderArgs := workspace.SetEnv(
		"SENDER_ARGS",
		"send /testdata/transfer_file_with_subdir/file --strict --private")
	defer restoreSenderArgs()
	restoreReceiverTargetPath := workspace.SetEnv("RECEIVER_TARGET_PATH", "/data/test1/test2")
	defer restoreReceiverTargetPath()

	expectedPath := filepath.Join(workspace.GetTestDataPath(), "transfer_file_with_subdir/file")
	receiverPath := filepath.Join(receiverDataPath, "test1/test2/file")
	sudoResetDir(filepath.Dir(receiverPath))

	composeFilePath := filepath.Join(getTestDataPath(), "private_network/compose.yaml")
	runTestPositive(t.Context(), composeFilePath, func() {
		asserts.AssertFilesEqual(expectedPath, receiverPath)
	})
}

func TestPrivateNetwork_SendFileWithRelativePath(t *testing.T) {
	t.Cleanup(cleanup)

	restoreSenderArgs := workspace.SetEnv(
		"SENDER_ARGS",
		"send ../testdata/transfer_file_with_subdir/file --strict --private")
	defer restoreSenderArgs()
	restoreReceiverTargetPath := workspace.SetEnv("RECEIVER_TARGET_PATH", "test1/test2")
	defer restoreReceiverTargetPath()

	expectedPath := filepath.Join(workspace.GetTestDataPath(), "transfer_file_with_subdir/file")
	receiverPath := filepath.Join(receiverDataPath, "test1/test2/file")
	sudoResetDir(filepath.Dir(receiverPath))

	composeFilePath := filepath.Join(getTestDataPath(), "private_network/compose.yaml")
	runTestPositive(t.Context(), composeFilePath, func() {
		asserts.AssertFilesEqual(expectedPath, receiverPath)
	})
}

func TestPrivateNetwork_SendFileWithRelativePath_Chdir(t *testing.T) {
	t.Cleanup(cleanup)

	restoreSenderDir := workspace.SetEnv("SENDER_DIR", "/testdata/transfer_file_with_subdir")
	defer restoreSenderDir()
	restoreSenderArgs := workspace.SetEnv("SENDER_ARGS", "send file --strict --private")
	defer restoreSenderArgs()
	restoreReceiverDir := workspace.SetEnv("RECEIVER_DIR", "/data/test1")
	defer restoreReceiverDir()
	restoreReceiverTargetPath := workspace.SetEnv("RECEIVER_TARGET_PATH", "test2")
	defer restoreReceiverTargetPath()

	expectedPath := filepath.Join(workspace.GetTestDataPath(), "transfer_file_with_subdir/file")
	receiverPath := filepath.Join(receiverDataPath, "test1/test2/file")
	sudoResetDir(filepath.Dir(receiverPath))

	composeFilePath := filepath.Join(getTestDataPath(), "private_network/compose.yaml")
	runTestPositive(t.Context(), composeFilePath, func() {
		asserts.AssertFilesEqual(expectedPath, receiverPath)
	})
}

func TestPrivateNetwork_SendFile_Confirm(t *testing.T) {
	t.Cleanup(cleanup)

	restoreSenderArgs := workspace.SetEnv("SENDER_ARGS", "send /testdata/transfer_file_with_subdir/file --private")
	defer restoreSenderArgs()
	restoreReceiverStdin := workspace.SetEnv("RECEIVER_STDIN", "y\n")
	defer restoreReceiverStdin()

	expectedPath := filepath.Join(workspace.GetTestDataPath(), "transfer_file_with_subdir/file")
	receiverPath := filepath.Join(receiverDataPath, "file")

	composeFilePath := filepath.Join(getTestDataPath(), "private_network/compose.yaml")
	runTestPositive(t.Context(), composeFilePath, func() {
		asserts.AssertFilesEqual(expectedPath, receiverPath)
	})
}

func TestPrivateNetwork_LargeFile(t *testing.T) {
	t.Cleanup(cleanup)

	restoreSenderArgs := workspace.SetEnv("SENDER_ARGS", "send test --strict --private")
	defer restoreSenderArgs()

	expectedPath := filepath.Join(senderDataPath, "test")
	generateFile(expectedPath, 1024*1024*10)
	receiverPath := filepath.Join(receiverDataPath, "test")

	composeFilePath := filepath.Join(getTestDataPath(), "private_network/compose.yaml")
	runTestPositive(t.Context(), composeFilePath, func() {
		asserts.AssertFilesEqual(expectedPath, receiverPath)
	})
}

func TestPublicNetwork(t *testing.T) {
	t.Cleanup(cleanup)

	// Sender sends empty /data (${PWD}) to receiver, receiver gets empty /data/data (${PWD}/data)
	expectedPath := filepath.Join(os.TempDir(), "p2pcp/test/integration/public_network/data")
	workspace.ResetDir(expectedPath)
	receiverPath := filepath.Join(receiverDataPath, "data")
	cleanup()
	sudoResetDir(filepath.Dir(receiverPath))

	composeFilePath := filepath.Join(getTestDataPath(), "public_network/compose.yaml")
	runTestPositive(t.Context(), composeFilePath, func() {
		asserts.AssertDirsEqual(expectedPath, receiverPath)
	})
}

func TestRelayNetwork(t *testing.T) {
	t.Cleanup(cleanup)

	restoreSenderArgs := workspace.SetEnv("SENDER_ARGS", "send test -sd")
	defer restoreSenderArgs()

	expectedPath := filepath.Join(senderDataPath, "test")
	generateFile(expectedPath, 1024*1024)
	receiverPath := filepath.Join(receiverDataPath, "test")

	composeFilePath := filepath.Join(getTestDataPath(), "relay_network/compose.yaml")
	runTestPositive(t.Context(), composeFilePath, func() {
		asserts.AssertFilesEqual(expectedPath, receiverPath)
	})
}
