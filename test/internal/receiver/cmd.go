package receiver

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"project/pkg/workspace"
	"strings"
	"test/internal/docker"
	"time"
)

func Run(ctx context.Context, receiverDir string, stdin string, targetPath string, secret string) error {
	line, err := docker.WaitForContainerLog(ctx, "sender", time.Minute, "p2pcp receive")
	if err != nil {
		return err
	}
	slog.Info(fmt.Sprintf("Received command: %s", line))

	cmd := strings.Split(line, " ")
	if len(cmd) < 4 {
		return fmt.Errorf("invalid command: %s", line)
	}

	args := append(cmd[1:], "--debug")
	if len(secret) > 0 {
		args[2] = secret
	}
	if len(targetPath) > 0 {
		args = append(args, targetPath)
	}

	fmt.Println(cmd[0], strings.Join(args, " "))
	c := exec.CommandContext(ctx, "/p2pcp", args...)
	if len(receiverDir) > 0 {
		c.Dir = receiverDir
	}
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	stdinPipe, err := c.StdinPipe()
	workspace.Check(err)
	defer stdinPipe.Close()

	err = c.Start()
	workspace.Check(err)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-sigChan:
			c.Process.Signal(os.Interrupt)
		}
	}()

	// Confirmation of sender ID.
	go func() {
		if len(stdin) > 0 {
			_, err := stdinPipe.Write([]byte(stdin))
			workspace.Check(err)
		}
		workspace.Check(stdinPipe.Close())
	}()

	return c.Wait()
}
