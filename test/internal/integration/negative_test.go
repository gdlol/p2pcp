package integration

import (
	"context"
	"log/slog"
	"path/filepath"
	"project/pkg/workspace"
	"runtime"
	"strings"
	"test/internal/docker"
	"testing"
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

	sudoResetDir(receiverDataPath)
	cleanup := runCompose(ctx, composeFilePath, testName)
	defer cleanup()

	logger.Info("Waiting for receiver to exit...")
	err := docker.WaitContainer(ctx, "receiver")
	workspace.Check(err)

	assertions()
}

func TestPrivateNetwork_DefaultDenyConfirm(t *testing.T) {
	t.Cleanup(cleanup)

	ctx := t.Context()

	restoreSenderArgs := setEnv("SENDER_ARGS", "send --private")
	defer restoreSenderArgs()
	restoreReceiverStdin := setEnv("RECEIVER_STDIN", "\n")
	defer restoreReceiverStdin()

	composeFilePath := filepath.Join(getTestDataPath(), "private_network/compose.yaml")
	runTestNegative(ctx, composeFilePath, func() {
		docker.AssertContainerLogContains(ctx, "receiver", receiverConfirmMessage)
		docker.AssertContainerLogNotContains(ctx, "receiver", "Done.")
		docker.AssertContainerLogNotContains(ctx, "sender", "Receiver ID:", "Sending...")
	})
}

func TestPrivateNetwork_NonExistDir(t *testing.T) {
	t.Cleanup(cleanup)

	ctx := t.Context()

	restoreReceiverTargetPath := setEnv("RECEIVER_TARGET_PATH", "/data/test1/test2")
	defer restoreReceiverTargetPath()

	composeFilePath := filepath.Join(getTestDataPath(), "private_network/compose.yaml")
	runTestNegative(ctx, composeFilePath, func() {
		docker.AssertContainerLogContains(ctx, "receiver", "path: directory /data/test1/test2 does not exist")
		docker.AssertContainerLogNotContains(ctx, "receiver", "Done.", receiverConfirmMessage)
		docker.AssertContainerLogNotContains(ctx, "sender", "Receiver ID:", "Sending...")
	})
}

func TestPrivateNetwork_WrongSecret(t *testing.T) {
	t.Cleanup(cleanup)

	ctx := t.Context()

	restoreSenderArgs := setEnv("SENDER_ARGS", "send --private")
	defer restoreSenderArgs()
	restoreReceiverStdin := setEnv("RECEIVER_STDIN", "y\n")
	defer restoreReceiverStdin()
	restoreReceiveSecret := setEnv("RECEIVER_SECRET", "abcd")
	defer restoreReceiveSecret()

	composeFilePath := filepath.Join(getTestDataPath(), "private_network/compose.yaml")
	runTestNegative(ctx, composeFilePath, func() {
		docker.WaitContainer(ctx, "sender")
		docker.AssertContainerLogContains(ctx, "receiver", "authentication failed")
		docker.AssertContainerLogNotContains(ctx, "receiver", "Done.")
		docker.AssertContainerLogContains(ctx, "sender", "error waiting for receiver: failed to authenticate receiver")
		docker.AssertContainerLogNotContains(ctx, "sender", "Receiver ID:", "Sending...")
	})
}

func TestPrivateNetwork_WrongSecret_Strict(t *testing.T) {
	t.Cleanup(cleanup)

	ctx := t.Context()

	restoreSenderArgs := setEnv("SENDER_ARGS", "send --private --strict")
	defer restoreSenderArgs()
	restoreReceiveSecret := setEnv("RECEIVER_SECRET", "abcd")
	defer restoreReceiveSecret()

	composeFilePath := filepath.Join(getTestDataPath(), "private_network/compose.yaml")
	runTestNegative(ctx, composeFilePath, func() {
		docker.AssertContainerLogContains(ctx, "receiver", "authentication failed")
		docker.AssertContainerLogNotContains(ctx, "receiver", "Done.")

		docker.AssertContainerLogNotContains(ctx, "sender",
			"Receiver ID:", "Sending...", "failed to authenticate receiver")

		restoreReceiveSecret := setEnv("RECEIVER_SECRET", "")
		defer restoreReceiveSecret()

		// Retry with correct secret
		workspace.RunCtxWithChdir(ctx, filepath.Dir(composeFilePath), "docker", "compose", "run", "receiver")

		docker.WaitContainer(ctx, "sender")
		docker.AssertContainerLogContains(ctx, "sender", "Sending...", "Done.")
	})
}
