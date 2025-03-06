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
	err := workspace.RunCtxWithChdir(ctx, filepath.Dir(composeFilePath), "docker", "compose", "up", "--detach")
	workspace.Check(err)
}

func ComposeStop(ctx context.Context, composeFilePath string) {
	err := workspace.RunCtxWithChdir(ctx, filepath.Dir(composeFilePath), "docker", "compose", "stop")
	workspace.Check(err)
}

func ComposeDown(ctx context.Context, composeFilePath string) {
	err := workspace.RunCtxWithChdir(ctx, filepath.Dir(composeFilePath), "docker", "compose", "down", "--volumes")
	workspace.Check(err)
}

func DumpComposeLogs(ctx context.Context, composeFilePath string, testName string) {
	cli, err := getClient()
	workspace.Check(err)

	composeProjectName := filepath.Base(filepath.Dir(composeFilePath))
	logsPath := filepath.Join(workspace.GetProjectPath(), "logs", "integration", testName)
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
