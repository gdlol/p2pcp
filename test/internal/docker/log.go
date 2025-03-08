package docker

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"project/pkg/workspace"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
)

func WaitForContainerLog(ctx context.Context, containerName string, timeout time.Duration, substring string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cli, err := getClient()
	if err != nil {
		return "", err
	}

	reader, err := cli.ContainerLogs(ctx, containerName, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	})
	if err != nil {
		return "", err
	}
	defer reader.Close()

	stdoutReader, stdoutWriter := io.Pipe()
	stderrReader, stderrWriter := io.Pipe()
	copyErr := make(chan error, 1)
	scanErr := make(chan error, 1)
	result := make(chan string, 1)

	// De-multiplex stdout/stderr logs
	go func() {
		defer stdoutReader.Close()
		_, err = stdcopy.StdCopy(stdoutWriter, stderrWriter, reader)
		if err != nil {
			select {
			case copyErr <- err:
			case <-ctx.Done():
			}
		}
	}()

	// Scan for a line starting containing the substring
	for _, reader := range []io.Reader{stdoutReader, stderrReader} {
		go func() {
			scanner := bufio.NewScanner(reader)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.Contains(line, substring) {
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
	}

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

func getContainerLogs(ctx context.Context, containerName string) (stdout string, stderr string, err error) {
	cli, err := getClient()
	if err != nil {
		return stdout, stderr, err
	}

	reader, err := cli.ContainerLogs(ctx, containerName, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     false,
	})
	if err != nil {
		return stdout, stderr, err
	}
	defer reader.Close()

	stdoutWriter := bytes.NewBuffer(nil)
	stderrWriter := bytes.NewBuffer(nil)
	_, err = stdcopy.StdCopy(stdoutWriter, stderrWriter, reader)
	if err != nil {
		return stdout, stderr, err
	}

	return stdoutWriter.String(), stderrWriter.String(), nil
}

func AssertContainerLogContains(ctx context.Context, containerName string, substrings ...string) {
	stdout, stderr, err := getContainerLogs(ctx, containerName)
	workspace.Check(err)

	stdoutLines := strings.Split(stdout, "\n")
	stderrLines := strings.Split(stderr, "\n")
	allLines := make(map[string]bool)
	for _, line := range append(stdoutLines, stderrLines...) {
		if line == "" {
			continue
		}
		allLines[line] = true
	}
	for _, substring := range substrings {
		found := false
		for line := range allLines {
			if strings.Contains(line, substring) {
				found = true
				break
			}
		}
		if !found {
			panic(fmt.Sprintf("Missing substring: %s", substring))
		}
	}
}

func AssertContainerLogNotContains(ctx context.Context, containerName string, substrings ...string) {
	stdout, stderr, err := getContainerLogs(ctx, containerName)
	workspace.Check(err)

	stdoutLines := strings.Split(stdout, "\n")
	stderrLines := strings.Split(stderr, "\n")
	allLines := make(map[string]bool)
	for _, line := range append(stdoutLines, stderrLines...) {
		if line == "" {
			continue
		}
		allLines[line] = true
	}
	for _, substring := range substrings {
		for line := range allLines {
			if strings.Contains(line, substring) {
				panic(fmt.Sprintf("Found substring: %s", substring))
			}
		}
	}
}
