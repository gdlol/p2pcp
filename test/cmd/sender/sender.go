package sender

import (
	"os"
	"test/internal/sender"

	"github.com/spf13/cobra"
)

var SenderCmd = &cobra.Command{
	Use:                "sender",
	DisableFlagParsing: true,
	Run: func(cmd *cobra.Command, args []string) {
		senderDir := os.Getenv("SENDER_DIR")
		sender.Run(cmd.Context(), senderDir, args)
	},
}
