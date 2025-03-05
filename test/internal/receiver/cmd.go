package receiver

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"project/pkg/workspace"
	"strings"
	"test/internal/docker"
	"time"
)

func Run(ctx context.Context, private bool, stdin string) error {
	line, err := docker.WaitForContainerLog(ctx, "sender", time.Minute, "p2pcp receive")
	if err != nil {
		return err
	}
	slog.Info(fmt.Sprintf("Received command: %s", line))

	cmd := strings.Split(line, " ")
	if len(cmd) != 4 {
		return fmt.Errorf("invalid command: %s", line)
	}

	args := append(cmd[1:], "--debug")
	if private {
		args = append(args, "--private")
	}

	fmt.Println(cmd[0], strings.Join(args, " "))
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	c := exec.CommandContext(ctx, "/p2pcp", args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	stdinPipe, err := c.StdinPipe()
	workspace.Check(err)
	defer stdinPipe.Close()

	err = c.Start()
	workspace.Check(err)

	// Confirmation of sender ID.
	go func() {
		if len(stdin) > 0 {
			_, err := stdinPipe.Write([]byte(stdin))
			workspace.Check(err)
		}
		workspace.Check(stdinPipe.Close())
	}()

	time.AfterFunc(30*time.Second, cancel)
	return c.Wait()
}
