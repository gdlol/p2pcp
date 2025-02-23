package cmd

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
				slog.Error("Error getting current working directory.", "error", err)
				return err
			}
		} else {
			path, err = filepath.Abs(args[0])
			if err != nil {
				slog.Error("Error getting absolute path.", "path", args[0], "error", err)
				return err
			}
		}
		strict, _ := cmd.Flags().GetBool("strict")

		slog.Debug(fmt.Sprintf("Sending %s...", path))

		return send.Send(ctx, path, strict)
	},
}

func init() {
	SendCmd.Flags().Bool("strict", false, "Use strict mode, this will generate a long token for authentication")
}
