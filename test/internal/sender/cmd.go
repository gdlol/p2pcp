package sender

import (
	"context"
	"os"
	"os/exec"
	"os/signal"
	"project/pkg/workspace"
)

func Run(ctx context.Context, senderDir string, args []string) {
	if len(senderDir) > 0 {
		restore := workspace.Chdir(senderDir)
		defer restore()
	}

	c := exec.CommandContext(ctx, "/p2pcp", args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	err := c.Start()
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

	err = c.Wait()
	workspace.Check(err)
}
