package server

import (
	"test/internal/server"

	"github.com/spf13/cobra"
)

var ServerCmd = &cobra.Command{
	Use: "server",
	RunE: func(cmd *cobra.Command, args []string) error {
		return server.Run(cmd.Context())
	},
}

func init() {
	ServerCmd.AddCommand(&cobra.Command{
		Use: "ready",
		RunE: func(cmd *cobra.Command, args []string) error {
			return server.Ready()
		},
	})
}
