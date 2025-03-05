package receiver

import (
	"os"
	"test/internal/receiver"

	"github.com/spf13/cobra"
)

var ReceiverCmd = &cobra.Command{
	Use: "receiver",
	RunE: func(cmd *cobra.Command, args []string) error {
		private := os.Getenv("RECEIVER_PRIVATE") == "true"
		stdin := os.Getenv("RECEIVER_STDIN")
		return receiver.Run(cmd.Context(), private, stdin)
	},
}
