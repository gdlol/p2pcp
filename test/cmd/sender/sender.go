package sender

import (
	"os"
	"test/internal/sender"

	"github.com/spf13/cobra"
)

var SenderCmd = &cobra.Command{
	Use:                "sender",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		senderDir := os.Getenv("SENDER_DIR")
		return sender.Run(cmd.Context(), senderDir, args)
	},
}
