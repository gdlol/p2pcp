package receive

import (
	"fmt"
	"log/slog"
	"os"
	"p2pcp/internal/path"
	"p2pcp/internal/receive"

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

		var basePath string
		var err error
		if len(args) == 1 {
			basePath = path.GetCurrentDirectory()
		} else {
			basePath = path.GetAbsolutePath(args[1])
		}
		info, err := os.Lstat(basePath)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return fmt.Errorf("path: %s is not a directory", basePath)
		}

		fmt.Printf("Enter PIN/token: ")
		var secret string
		fmt.Scanln(&secret)
		if len(secret) < 6 {
			return fmt.Errorf("PIN/token: must be at least 6 characters long")
		}

		private, _ := cmd.Flags().GetBool("private")

		slog.Debug("Receiving...", "id", id, "path", basePath, "private", private)
		return receive.Receive(ctx, id, secret, basePath, private)
	},
}
