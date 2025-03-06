package sender

import (
	"context"
	"project/pkg/workspace"
)

func Run(ctx context.Context, senderDir string, args []string) error {
	if len(senderDir) > 0 {
		return workspace.RunCtxWithChdir(ctx, senderDir, "/p2pcp", args...)
	} else {
		return workspace.RunCtx(ctx, "/p2pcp", args...)
	}
}
