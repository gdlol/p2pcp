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
	"testing"
	"time"
)

const receiverConfirmMessage = "Please verify that the following random art " +
	"matches the one displayed on the sender's side."

func runTestNegative(ctx context.Context, composeFilePath string, assertions func()) {
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

	assertions()
}

func TestPrivateNetwork_DefaultDenyConfirm(t *testing.T) {
	t.Cleanup(cleanup)

	ctx := t.Context()

	restoreSenderArgs := workspace.SetEnv("SENDER_ARGS", "send --private")
	defer restoreSenderArgs()
	restoreReceiverStdin := workspace.SetEnv("RECEIVER_STDIN", "\n")
	defer restoreReceiverStdin()

	composeFilePath := filepath.Join(getTestDataPath(), "private_network/compose.yaml")
	runTestNegative(ctx, composeFilePath, func() {
		docker.WaitContainer(ctx, "receiver")
		docker.AssertContainerLogContains(ctx, "receiver", receiverConfirmMessage)
		docker.AssertContainerLogNotContains(ctx, "receiver", "Done.")
		docker.AssertContainerLogNotContains(ctx, "sender", "Receiver ID:", "Sending...")
	})
}

func TestPrivateNetwork_SenderNonExistPath(t *testing.T) {
	t.Cleanup(cleanup)

	ctx := t.Context()

	restoreSenderArgs := workspace.SetEnv("SENDER_ARGS", "send file --strict --private")
	defer restoreSenderArgs()

	composeFilePath := filepath.Join(getTestDataPath(), "private_network/compose.yaml")
	runTestNegative(ctx, composeFilePath, func() {
		docker.WaitContainer(ctx, "sender")
		docker.AssertContainerLogContains(ctx, "sender", "path: path /data/file does not exist")
		docker.AssertContainerLogNotContains(ctx, "receiver", "Done.", receiverConfirmMessage)
		docker.AssertContainerLogNotContains(ctx, "sender", "Done.", "Sending...")
	})
}

func TestPrivateNetwork_ReceiverNonExistDir(t *testing.T) {
	t.Cleanup(cleanup)

	ctx := t.Context()

	restoreReceiverTargetPath := workspace.SetEnv("RECEIVER_TARGET_PATH", "/data/test1/test2")
	defer restoreReceiverTargetPath()

	composeFilePath := filepath.Join(getTestDataPath(), "private_network/compose.yaml")
	runTestNegative(ctx, composeFilePath, func() {
		docker.WaitContainer(ctx, "receiver")
		docker.AssertContainerLogContains(ctx, "receiver", "path: directory /data/test1/test2 does not exist")
		docker.AssertContainerLogNotContains(ctx, "receiver", "Done.", receiverConfirmMessage)
		docker.AssertContainerLogNotContains(ctx, "sender", "Receiver ID:", "Sending...")
	})
}

func TestPrivateNetwork_ReceiverInvalidPath(t *testing.T) {
	t.Cleanup(cleanup)

	ctx := t.Context()

	restoreReceiverTargetPath := workspace.SetEnv("RECEIVER_TARGET_PATH", "/data/file")
	defer restoreReceiverTargetPath()
	receiverPath := filepath.Join(receiverDataPath, "file")
	generateFile(receiverPath, 1024)

	composeFilePath := filepath.Join(getTestDataPath(), "private_network/compose.yaml")
	runTestNegative(ctx, composeFilePath, func() {
		docker.WaitContainer(ctx, "receiver")
		docker.AssertContainerLogContains(ctx, "receiver", "path: /data/file is not a directory")
		docker.AssertContainerLogNotContains(ctx, "receiver", "Done.")
		docker.AssertContainerLogNotContains(ctx, "sender", "Sending...")
	})
}

func TestPrivateNetwork_WrongSecret(t *testing.T) {
	t.Cleanup(cleanup)

	ctx := t.Context()

	restoreSenderArgs := workspace.SetEnv("SENDER_ARGS", "send --private")
	defer restoreSenderArgs()
	restoreReceiverStdin := workspace.SetEnv("RECEIVER_STDIN", "y\n")
	defer restoreReceiverStdin()
	restoreReceiveSecret := workspace.SetEnv("RECEIVER_SECRET", "abcdef")
	defer restoreReceiveSecret()

	composeFilePath := filepath.Join(getTestDataPath(), "private_network/compose.yaml")
	runTestNegative(ctx, composeFilePath, func() {
		docker.WaitContainer(ctx, "receiver")
		docker.WaitContainer(ctx, "sender")
		docker.AssertContainerLogContains(ctx, "receiver", "authentication failed")
		docker.AssertContainerLogNotContains(ctx, "receiver", "Done.")
		docker.AssertContainerLogContains(ctx, "sender", "failed to authenticate receiver")
		docker.AssertContainerLogNotContains(ctx, "sender", "Receiver ID:", "Sending...")
	})
}

func TestPrivateNetwork_WrongSecret_Strict(t *testing.T) {
	t.Cleanup(cleanup)

	ctx := t.Context()

	restoreSenderArgs := workspace.SetEnv("SENDER_ARGS", "send --private --strict")
	defer restoreSenderArgs()
	restoreReceiveSecret := workspace.SetEnv("RECEIVER_SECRET", "abcdef")
	defer restoreReceiveSecret()

	composeFilePath := filepath.Join(getTestDataPath(), "private_network/compose.yaml")
	runTestNegative(ctx, composeFilePath, func() {
		docker.WaitContainer(ctx, "receiver")
		docker.AssertContainerLogContains(ctx, "receiver", "authentication failed")
		docker.AssertContainerLogNotContains(ctx, "receiver", "Done.")
		docker.AssertContainerLogNotContains(ctx, "sender",
			"Receiver ID:", "Sending...", "failed to authenticate receiver")

		restoreReceiveSecret := workspace.SetEnv("RECEIVER_SECRET", "")
		defer restoreReceiveSecret()

		// Retry with correct secret
		workspace.RunCtxWithChdir(ctx, filepath.Dir(composeFilePath), "docker", "compose", "run", "receiver")

		docker.WaitContainer(ctx, "sender")
		docker.AssertContainerLogContains(ctx, "sender", "Sending...", "Done.")
	})
}

func TestPrivateNetwork_SenderError(t *testing.T) {
	t.Cleanup(cleanup)

	ctx := t.Context()

	restoreSenderArgs := workspace.SetEnv("SENDER_ARGS", "send file --strict --private")
	defer restoreSenderArgs()

	senderPath := filepath.Join(senderDataPath, "file")
	err := os.Symlink("not/exist", senderPath)
	workspace.Check(err)

	composeFilePath := filepath.Join(getTestDataPath(), "private_network/compose.yaml")
	runTestNegative(ctx, composeFilePath, func() {
		docker.WaitContainer(ctx, "receiver")
		docker.WaitContainer(ctx, "sender")
		docker.AssertContainerLogContains(ctx, "receiver", "Sender error error=\"\"")
		docker.AssertContainerLogNotContains(ctx, "receiver", "Done.")
		docker.AssertContainerLogContains(ctx, "sender", "Sending...", "unsupported file type: /data/file")
		docker.AssertContainerLogNotContains(ctx, "sender", "Done.")
	})
}

func TestPrivateNetwork_ReceiverError(t *testing.T) {
	t.Cleanup(cleanup)

	ctx := t.Context()

	restoreSenderArgs := workspace.SetEnv(
		"SENDER_ARGS",
		"send /testdata/transfer_file_with_subdir/file --strict --private")
	defer restoreSenderArgs()
	restoreReceiverTargetPath := workspace.SetEnv("RECEIVER_TARGET_PATH", "/data/test1/test2")
	defer restoreReceiverTargetPath()

	receiverPath := filepath.Join(receiverDataPath, "test1/test2/file")
	sudoResetDir(filepath.Dir(receiverPath))
	err := os.MkdirAll(receiverPath, 0555)
	workspace.Check(err)

	composeFilePath := filepath.Join(getTestDataPath(), "private_network/compose.yaml")
	runTestNegative(ctx, composeFilePath, func() {
		docker.WaitContainer(ctx, "receiver")
		docker.WaitContainer(ctx, "sender")
		docker.AssertContainerLogContains(ctx, "receiver", "/data/test1/test2/file: is a directory")
		docker.AssertContainerLogNotContains(ctx, "receiver", "Done.")
		docker.AssertContainerLogContains(ctx, "sender", "Sending...", "Receiver error error=\"\"")
		docker.AssertContainerLogNotContains(ctx, "sender", "Done.")
	})
}

func TestRelayNetwork_SenderCancel(t *testing.T) {
	t.Cleanup(cleanup)

	ctx := t.Context()

	restoreSenderArgs := workspace.SetEnv("SENDER_ARGS", "send large_file --strict")
	defer restoreSenderArgs()

	senderPath := filepath.Join(senderDataPath, "large_file")
	generateFile(senderPath, 1024*1024*100)

	composeFilePath := filepath.Join(getTestDataPath(), "relay_network/compose.yaml")
	runTestNegative(ctx, composeFilePath, func() {
		_, err := docker.WaitForContainerLog(ctx, "receiver", time.Minute, "large_file")
		workspace.Check(err)
		workspace.RunCtx(ctx, "docker", "kill", "sender", "--signal", "SIGINT")
		docker.WaitContainer(ctx, "sender")
		docker.WaitContainer(ctx, "receiver")
		docker.AssertContainerLogContains(ctx, "sender", "Canceling...")
		docker.AssertContainerLogContains(ctx, "receiver", "Sender error error=\"Transfer canceled.\"")
		docker.AssertContainerLogNotContains(ctx, "receiver", "Done.")
		docker.AssertContainerLogNotContains(ctx, "sender", "Done.")
	})
}

func TestRelayNetwork_ReceiverCancel(t *testing.T) {
	t.Cleanup(cleanup)

	ctx := t.Context()

	restoreSenderArgs := workspace.SetEnv("SENDER_ARGS", "send large_file --strict")
	defer restoreSenderArgs()

	senderPath := filepath.Join(senderDataPath, "large_file")
	generateFile(senderPath, 1024*1024*100)

	composeFilePath := filepath.Join(getTestDataPath(), "relay_network/compose.yaml")
	runTestNegative(ctx, composeFilePath, func() {
		_, err := docker.WaitForContainerLog(ctx, "receiver", time.Minute, "large_file")
		workspace.Check(err)
		workspace.RunCtx(ctx, "docker", "kill", "receiver", "--signal", "SIGINT")
		docker.WaitContainer(ctx, "receiver")
		docker.WaitContainer(ctx, "sender")
		docker.AssertContainerLogContains(ctx, "sender", "Receiver error error=\"Transfer canceled.\"")
		docker.AssertContainerLogContains(ctx, "receiver", "Canceling...")
		docker.AssertContainerLogNotContains(ctx, "receiver", "Done.")
		docker.AssertContainerLogNotContains(ctx, "sender", "Done.")
	})
}

func TestPrivateNetwork_SendDir_OverwriteFile(t *testing.T) {
	t.Cleanup(cleanup)

	ctx := t.Context()

	restore := workspace.SetEnv("SENDER_ARGS", "send /testdata/transfer_file_with_subdir/subdir --strict --private")
	defer restore()

	receiverPath := filepath.Join(receiverDataPath, "subdir")
	generateFile(receiverPath, 1024)

	composeFilePath := filepath.Join(getTestDataPath(), "private_network/compose.yaml")
	runTestNegative(t.Context(), composeFilePath, func() {
		docker.WaitContainer(ctx, "receiver")
		docker.WaitContainer(ctx, "sender")
		docker.AssertContainerLogContains(ctx, "sender", "Sending...", "Receiver error error=\"\"")
		docker.AssertContainerLogContains(ctx, "receiver", "error creating directory /data/subdir")
		docker.AssertContainerLogNotContains(ctx, "receiver", "Done.")
		docker.AssertContainerLogNotContains(ctx, "sender", "Done.")
	})
}
