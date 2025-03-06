package docker

import (
	"context"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

func getClient() (*client.Client, error) {
	return client.NewClientWithOpts(
		client.WithAPIVersionNegotiation(),
		client.WithHost("unix:///var/run/docker.sock"))
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
