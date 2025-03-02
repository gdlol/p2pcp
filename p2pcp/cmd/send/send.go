package send

import (
	"fmt"
	"log/slog"
	"os"
	"p2pcp/internal/send"
	"path/filepath"

	"github.com/spf13/cobra"
)

var SendCmd = &cobra.Command{
	Use:   "send [path]",
	Short: "Sends the specified file/directory to remote peer.",
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

		var path string
		var err error
		if len(args) == 0 {
			path, err = os.Getwd()
			if err != nil {
				return fmt.Errorf("error getting current working directory: %w", err)
			}
		} else {
			path, err = filepath.Abs(args[0])
			if err != nil {
				return fmt.Errorf("error getting absolute path: %w", err)
			}
		}
		strict, _ := cmd.Flags().GetBool("strict")

		private, _ := cmd.Flags().GetBool("private")

		slog.Debug(fmt.Sprintf("Sending %s...", path), "strict", strict, "private", private)
		return send.Send(ctx, path, strict, private)
	},
}

func init() {
	SendCmd.Flags().BoolP("strict", "s", false, "use strict mode, this will generate a long secret for authentication")
}
