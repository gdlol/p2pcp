package integration

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"project/pkg/project"
	"project/pkg/workspace"
	"runtime"
	"strings"
	"test/internal/docker"
	"testing"
)

func runCommand(ctx context.Context, args ...string) (rm func()) {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		panic("Failed to get caller info.")
	}
	testName := filepath.Base(runtime.FuncForPC(pc).Name())
	testName = testName[strings.LastIndex(testName, ".")+1:]
	logger := slog.With("test", testName)
	logger.Info("Starting test...")

	rm = func() {
		workspace.RunCtx(ctx, "docker", "rm", "--force", "--volumes", project.Name)
		workspace.RunCtx(ctx, "docker", "volume", "prune",
			"--all", "--force",
			"--filter", fmt.Sprintf("label=%s", project.Name))
	}
	rm()
	func() {
		defer func() {
			recover()
			coveragePath := filepath.Join(workspace.GetProjectPath(), ".local/coverage/integration")
			err := os.MkdirAll(coveragePath, 0755)
			workspace.Check(err)
			workspace.RunCtx(ctx, "docker", "cp", fmt.Sprintf("%s:/coverage/.", project.Name), coveragePath)
		}()
		workspace.RunCtx(ctx, "docker", "volume", "create", "coverage", "--label", project.Name)
		args = append([]string{args[0], "--private"}, args[1:]...)
		workspace.RunCtx(ctx, "docker", append([]string{"run",
			"--name", project.Name,
			"--entrypoint", "/p2pcp",
			"--volume", "coverage:/coverage",
			"--env", "GOCOVERDIR=/coverage",
			"--env", fmt.Sprintf("RECEIVER_SECRET=%s", os.Getenv("RECEIVER_SECRET")),
			"local/test"},
			args...)...)
	}()
	return rm
}

func TestSenderArgs(t *testing.T) {
	ctx := t.Context()

	func() {
		rm := runCommand(ctx, "send", "a", "b")
		defer rm()
		docker.AssertContainerLogContains(ctx, project.Name, "Usage:")
	}()

}

func TestReceiverArgs(t *testing.T) {
	ctx := t.Context()

	for _, args := range [][]string{
		{"receive"},
		{"receive", "a", "b", "c"},
	} {
		func() {
			rm := runCommand(ctx, args...)
			defer rm()
			docker.AssertContainerLogContains(ctx, project.Name, "Usage:")
		}()
	}

	func() {
		rm := runCommand(ctx, "receive", "short")
		defer rm()
		docker.AssertContainerLogContains(ctx, project.Name, "id: must be at least 7 characters long")
	}()

	func() {
		restore := workspace.SetEnv("RECEIVER_SECRET", "12345")
		defer restore()
		rm := runCommand(ctx, "receive", "abcdefg")
		defer rm()
		docker.AssertContainerLogContains(ctx, project.Name, "PIN/token: must be at least 6 characters long")
	}()
}
