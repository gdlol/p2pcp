package cmd

import (
	"test/cmd/integration"
	"test/cmd/receiver"
	"test/cmd/sender"
	"test/cmd/server"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:                "test",
	SilenceUsage:       true,
	SilenceErrors:      true,
	DisableFlagParsing: true,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	rootCmd.AddCommand(server.ServerCmd)
	rootCmd.AddCommand(sender.SenderCmd)
	rootCmd.AddCommand(receiver.ReceiverCmd)
	rootCmd.AddCommand(integration.IntegrationCmd)
}
