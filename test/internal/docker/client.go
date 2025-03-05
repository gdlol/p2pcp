package docker

import (
	"bufio"
	"context"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

func getClient() (*client.Client, error) {
	return client.NewClientWithOpts(
		client.WithAPIVersionNegotiation(),
		client.WithHost("unix:///var/run/docker.sock"))
}

func WaitForContainerLog(ctx context.Context, containerName string, timeout time.Duration, prefix string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cli, err := getClient()
	if err != nil {
		return "", err
	}

	reader, err := cli.ContainerLogs(ctx, containerName, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: false,
		Follow:     true,
	})
	if err != nil {
		return "", err
	}
	defer reader.Close()

	pipeReader, pipeWriter := io.Pipe()
	copyErr := make(chan error, 1)
	scanErr := make(chan error, 1)
	result := make(chan string, 1)

	// De-multiplex stdout/stderr logs
	go func() {
		defer pipeReader.Close()
		_, err = stdcopy.StdCopy(pipeWriter, io.Discard, reader)
		if err != nil {
			select {
			case copyErr <- err:
			case <-ctx.Done():
			}
		}
	}()

	// Scan for a line starting with prefix
	go func() {
		scanner := bufio.NewScanner(pipeReader)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, prefix) {
				select {
				case result <- line:
				case <-ctx.Done():
				}
				return
			}
		}
		if err := scanner.Err(); err != nil {
			select {
			case scanErr <- err:
			case <-ctx.Done():
			}
		}
	}()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case err := <-copyErr:
		return "", err
	case err := <-scanErr:
		return "", err
	case line := <-result:
		return line, nil
	}
}

func WaitContainer(ctx context.Context, name string) error {
	cli, err := getClient()
	if err != nil {
		return err
	}

	statusCh, errCh := cli.ContainerWait(ctx, name, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		return err
	case <-statusCh:
		return nil
	}
}

func WaitContainerWithTimeout(ctx context.Context, name string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return WaitContainer(ctx, name)
}

func StopContainer(ctx context.Context, name string) error {
	cli, err := getClient()
	if err != nil {
		return err
	}

	return cli.ContainerStop(ctx, name, container.StopOptions{})
}
