package sender

import (
	"context"
	"project/pkg/workspace"
)

func Run(ctx context.Context, senderDir string, args []string) error {
	restore := workspace.Chdir(senderDir)
	defer restore()
	return workspace.Run("/p2pcp", args...)
}
