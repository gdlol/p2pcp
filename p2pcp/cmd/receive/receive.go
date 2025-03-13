package receive

import (
	"fmt"
	"log/slog"
	"os"
	"p2pcp/internal/receive"
	"path/filepath"

	"github.com/spf13/cobra"
)

var ReceiveCmd = &cobra.Command{
	Use:   "receive id [path]",
	Short: "Receives file/directory from remote peer to specified directory",
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.RangeArgs(1, 2)(cmd, args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			fmt.Println()
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

		var path string
		var err error
		if len(args) == 1 {
			path, err = os.Getwd()
			if err != nil {
				return fmt.Errorf("error getting current working directory: %w", err)
			}
		} else {
			path, err = filepath.Abs(args[1])
			if err != nil {
				return fmt.Errorf("error getting absolute path: %w", err)
			}
		}
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			return fmt.Errorf("path: directory %s does not exist", path)
		}
		if !info.IsDir() {
			return fmt.Errorf("path: %s is not a directory", path)
		}

		fmt.Printf("Enter PIN/token: ")
		var secret string
		fmt.Scanln(&secret)
		if len(secret) < 6 {
			return fmt.Errorf("PIN/token: must be at least 6 characters long")
		}

		private, _ := cmd.Flags().GetBool("private")

		slog.Debug("Receiving...", "id", id, "path", path, "private", private)
		return receive.Receive(ctx, id, secret, path, private)
	},
}
