package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"p2pcp/internal/receive"
	"path/filepath"

	"github.com/spf13/cobra"
)

var ReceiveCmd = &cobra.Command{
	Use:   "receive id {pin | token} [path]",
	Short: "Receives file/directory from remote peer to specified directory.",
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.RangeArgs(2, 3)(cmd, args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			cmd.Usage()
			os.Exit(1)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		id := args[0]
		if len(id) < 7 {
			return fmt.Errorf("id: must be at least 7 characters long")
		}

		secret := args[1]
		if len(secret) < 4 {
			return fmt.Errorf("pin/token: must be at least 4 characters long")
		}

		var path string
		var err error
		if len(args) == 2 {
			path, err = os.Getwd()
			if err != nil {
				slog.Error("Error getting current working directory.", "error", err)
				return err
			}
		} else {
			path, err = filepath.Abs(args[1])
			if err != nil {
				slog.Error("Error getting absolute path.", "path", args[2], "error", err)
				return err
			}
		}

		slog.Debug("Receiving...", "id", id, "path", path)
		return receive.Receive(ctx, id, secret, path)
	},
}
