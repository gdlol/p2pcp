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
	Use:   "receive topic [path]",
	Short: "Receives file/directory from remote peer to specified directory.",
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.RangeArgs(1, 2)(cmd, args); err != nil {
			cmd.Usage()
			os.Exit(1)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		topic := args[0]
		if len(topic) < 7 {
			return fmt.Errorf("topic: must be at least 7 characters long")
		}

		var path string
		var err error
		if len(args) == 1 {
			path, err = os.Getwd()
			if err != nil {
				slog.Error("Error getting current working directory.", "error", err)
				return err
			}
		} else {
			path, err = filepath.Abs(args[1])
			if err != nil {
				slog.Error("Error getting absolute path.", "path", args[1], "error", err)
				return err
			}
		}

		slog.Debug("Receiving...", "topic", topic, "path", path)

		return receive.Receive(ctx, topic, path)
	},
}
