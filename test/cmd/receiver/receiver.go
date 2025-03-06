package receiver

import (
	"os"
	"test/internal/receiver"

	"github.com/spf13/cobra"
)

var ReceiverCmd = &cobra.Command{
	Use: "receiver",
	RunE: func(cmd *cobra.Command, args []string) error {
		receiverDir := os.Getenv("RECEIVER_DIR")
		stdin := os.Getenv("RECEIVER_STDIN")
		targetPath := os.Getenv("RECEIVER_TARGET_PATH")
		receiverSecret := os.Getenv("RECEIVER_SECRET")
		return receiver.Run(cmd.Context(), receiverDir, stdin, targetPath, receiverSecret)
	},
}
