package docker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"project/pkg/workspace"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/pkg/stdcopy"
)

func ComposeUp(ctx context.Context, composeFilePath string) {
	composePath := filepath.Dir(composeFilePath)
	defer func() {
		if r := recover(); r != nil {
			workspace.RunCtxWithChdir(ctx, composePath, "docker", "compose", "logs", "--no-color")
			panic(r)
		}
	}()
	workspace.RunCtxWithChdir(ctx, composePath, "docker", "compose", "up", "--detach")
}

func ComposeStop(ctx context.Context, composeFilePath string) {
	workspace.RunCtxWithChdir(ctx, filepath.Dir(composeFilePath), "docker", "compose", "stop", "--timeout", "0")
}

func ComposeDown(ctx context.Context, composeFilePath string) {
	workspace.RunCtxWithChdir(ctx, filepath.Dir(composeFilePath),
		"docker", "compose", "down", "--volumes", "--remove-orphans")
}

func ComposeCollectCoverage(ctx context.Context) {
	coveragePath := filepath.Join(workspace.GetProjectPath(), ".local/coverage/integration")
	err := os.MkdirAll(coveragePath, 0755)
	workspace.Check(err)
	workspace.RunCtx(ctx, "docker", "cp", "receiver:/coverage/.", coveragePath)
}

func DumpComposeLogs(ctx context.Context, composeFilePath string, testName string) {
	cli, err := getClient()
	workspace.Check(err)

	composeProjectName := filepath.Base(filepath.Dir(composeFilePath))
	logsPath := filepath.Join(workspace.GetProjectPath(), ".local/logs/integration", testName)
	workspace.ResetDir(logsPath)

	filter := filters.NewArgs()
	filter.Add("label", fmt.Sprintf("com.docker.compose.project=%s", composeProjectName))
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true, Filters: filter})
	workspace.Check(err)
	for _, c := range containers {
		name := c.Labels["com.docker.compose.service"]
		stdOutPath := filepath.Join(logsPath, name+".stdout.log")
		stdErrPath := filepath.Join(logsPath, name+".stderr.log")
		stdOutFile, err := os.Create(stdOutPath)
		workspace.Check(err)
		defer stdOutFile.Close()
		stdErrFile, err := os.Create(stdErrPath)
		workspace.Check(err)
		defer stdErrFile.Close()

		reader, err := cli.ContainerLogs(ctx, c.ID, container.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
		})
		workspace.Check(err)
		defer reader.Close()

		_, err = stdcopy.StdCopy(stdOutFile, stdErrFile, reader)
		workspace.Check(err)
	}
}
