package send

import (
	"fmt"
	"log/slog"
	"os"
	"p2pcp/internal/path"
	"p2pcp/internal/send"

	"github.com/spf13/cobra"
)

var SendCmd = &cobra.Command{
	Use:   "send [path]",
	Short: "Sends the specified file/directory to remote peer",
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.MaximumNArgs(1)(cmd, args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			fmt.Println()
			cmd.Usage()
			os.Exit(1)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		var basePath string
		if len(args) == 0 {
			basePath = path.GetCurrentDirectory()
		} else {
			basePath = path.GetAbsolutePath(args[0])
		}
		if _, err := os.Lstat(basePath); err != nil {
			return err
		}

		strict, _ := cmd.Flags().GetBool("strict")
		private, _ := cmd.Flags().GetBool("private")

		slog.Debug(fmt.Sprintf("Sending %s...", basePath), "strict", strict, "private", private)
		return send.Send(ctx, basePath, strict, private)
	},
}

func init() {
	SendCmd.Flags().BoolP("strict", "s", false, "use strict mode, this will generate a long secret for authentication")
}
