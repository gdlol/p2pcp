package integration

import (
	"test/internal/integration"

	"github.com/spf13/cobra"
)

var IntegrationCmd = &cobra.Command{
	Use: "integration",
	RunE: func(cmd *cobra.Command, args []string) error {
		integration.RunTests(cmd.Context())
		return nil
	},
}
