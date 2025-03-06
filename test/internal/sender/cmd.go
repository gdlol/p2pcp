package sender

import (
	"context"
	"project/pkg/workspace"
)

func Run(ctx context.Context, senderDir string, args []string) {
	if len(senderDir) > 0 {
		workspace.RunCtxWithChdir(ctx, senderDir, "/p2pcp", args...)
	} else {
		workspace.RunCtx(ctx, "/p2pcp", args...)
	}
}
