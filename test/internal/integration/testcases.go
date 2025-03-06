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
	"time"
)

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

func runTest(ctx context.Context, composeFilePath string, expectedPath string, receiverPath string) {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		panic("Failed to get caller info.")
	}
	testName := filepath.Base(runtime.FuncForPC(pc).Name())
	testName = testName[strings.LastIndex(testName, ".")+1:]
	logger := slog.With("test", testName)
	logger.Info("Starting test...")

	sudoResetDir(receiverDataPath)
	cleanup := runCompose(ctx, composeFilePath, testName)
	defer cleanup()

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	logger.Info("Waiting for receiver to exit...")
	err := docker.WaitContainer(ctx, "receiver")
	workspace.Check(err)
	docker.AssertContainerLogContains(ctx, "receiver", "Done.")

	logger.Info("Waiting for sender to exit...")
	err = docker.WaitContainer(ctx, "sender")
	workspace.Check(err)
	docker.AssertContainerLogContains(ctx, "sender", "Done.")

	info, err := os.Stat(expectedPath)
	workspace.Check(err)
	if info.IsDir() {
		asserts.AreDirsEqual(expectedPath, receiverPath)
	} else {
		asserts.AreFilesEqual(expectedPath, receiverPath)
	}
	logger.Info("Data directories are equal.")
}

func testPrivateNetwork(ctx context.Context) {
	// Sender sends empty /data (${PWD}) to receiver, receiver gets empty /data/data (${PWD}/data)
	expectedPath := filepath.Join(os.TempDir(), "p2pcp/test/integration/public_network/data")
	workspace.ResetDir(expectedPath)
	receiverPath := filepath.Join(receiverDataPath, "data")

	composeFilePath := filepath.Join(getTestDataPath(), "private_network/compose.yaml")
	runTest(ctx, composeFilePath, expectedPath, receiverPath)
}

func testPrivateNetwork_SendDir(ctx context.Context) {
	restore := setEnv("SENDER_ARGS", "send /testdata/transfer_file_with_subdir/subdir --strict --private")
	defer restore()

	expectedPath := filepath.Join(workspace.GetTestDataPath(), "transfer_file_with_subdir/subdir")
	receiverPath := filepath.Join(receiverDataPath, "subdir")

	composeFilePath := filepath.Join(getTestDataPath(), "private_network/compose.yaml")
	runTest(ctx, composeFilePath, expectedPath, receiverPath)
}

func testPrivateNetwork_SendFile(ctx context.Context) {
	restore := setEnv("SENDER_ARGS", "send /testdata/transfer_file_with_subdir/file --strict --private")
	defer restore()

	expectedPath := filepath.Join(workspace.GetTestDataPath(), "transfer_file_with_subdir/file")
	receiverPath := filepath.Join(receiverDataPath, "file")

	composeFilePath := filepath.Join(getTestDataPath(), "private_network/compose.yaml")
	runTest(ctx, composeFilePath, expectedPath, receiverPath)
}

func testPrivateNetwork_SendFileWithAbsPath(ctx context.Context) {
	restoreSenderArgs := setEnv("SENDER_ARGS", "send /testdata/transfer_file_with_subdir/file --strict --private")
	defer restoreSenderArgs()
	restoreReceiverTargetPath := setEnv("RECEIVER_TARGET_PATH", "/data/test1/test2")
	defer restoreReceiverTargetPath()

	expectedPath := filepath.Join(workspace.GetTestDataPath(), "transfer_file_with_subdir/file")
	receiverPath := filepath.Join(receiverDataPath, "test1/test2/file")

	composeFilePath := filepath.Join(getTestDataPath(), "private_network/compose.yaml")
	runTest(ctx, composeFilePath, expectedPath, receiverPath)
}

func testPrivateNetwork_SendFileWithRelativePath(ctx context.Context) {
	restoreSenderArgs := setEnv("SENDER_ARGS", "send /testdata/transfer_file_with_subdir/file --strict --private")
	defer restoreSenderArgs()
	restoreReceiverTargetPath := setEnv("RECEIVER_TARGET_PATH", "test1/test2")
	defer restoreReceiverTargetPath()

	expectedPath := filepath.Join(workspace.GetTestDataPath(), "transfer_file_with_subdir/file")
	receiverPath := filepath.Join(receiverDataPath, "test1/test2/file")

	composeFilePath := filepath.Join(getTestDataPath(), "private_network/compose.yaml")
	runTest(ctx, composeFilePath, expectedPath, receiverPath)
}

func testPrivateNetwork_SendFileWithRelativePath_Chdir(ctx context.Context) {
	restoreSenderDir := setEnv("SENDER_DIR", "/tmp")
	defer restoreSenderDir()
	restoreSenderArgs := setEnv("SENDER_ARGS", "send ../testdata/transfer_file_with_subdir/file --strict --private")
	defer restoreSenderArgs()
	restoreReceiverDir := setEnv("RECEIVER_DIR", "/tmp")
	defer restoreReceiverDir()
	restoreReceiverTargetPath := setEnv("RECEIVER_TARGET_PATH", "../data/test1/test2")
	defer restoreReceiverTargetPath()

	expectedPath := filepath.Join(workspace.GetTestDataPath(), "transfer_file_with_subdir/file")
	receiverPath := filepath.Join(receiverDataPath, "test1/test2/file")

	composeFilePath := filepath.Join(getTestDataPath(), "private_network/compose.yaml")
	runTest(ctx, composeFilePath, expectedPath, receiverPath)
}

func testPrivateNetwork_SendFile_Confirm(ctx context.Context) {
	restoreSenderArgs := setEnv("SENDER_ARGS", "send /testdata/transfer_file_with_subdir/file --private")
	defer restoreSenderArgs()
	restoreReceiverStdin := setEnv("RECEIVER_STDIN", "y\n")
	defer restoreReceiverStdin()

	expectedPath := filepath.Join(workspace.GetTestDataPath(), "transfer_file_with_subdir/file")
	receiverPath := filepath.Join(receiverDataPath, "file")

	composeFilePath := filepath.Join(getTestDataPath(), "private_network/compose.yaml")
	runTest(ctx, composeFilePath, expectedPath, receiverPath)
}

func testPublicNetwork(ctx context.Context) {
	// Sender sends empty /data (${PWD}) to receiver, receiver gets empty /data/data (${PWD}/data)
	expectedPath := filepath.Join(os.TempDir(), "p2pcp/test/integration/public_network/data")
	workspace.ResetDir(expectedPath)
	receiverPath := filepath.Join(receiverDataPath, "data")

	composeFilePath := filepath.Join(getTestDataPath(), "public_network/compose.yaml")
	runTest(ctx, composeFilePath, expectedPath, receiverPath)
}
