package cmd

import (
	"project/cmd/build"
	"project/cmd/format"
	"project/cmd/install"
	"project/cmd/lint"
	"project/cmd/restore"
	"project/cmd/sync"
	"project/cmd/test"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:                "project",
	SilenceUsage:       true,
	SilenceErrors:      true,
	DisableFlagParsing: true,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.AddCommand(build.BuildCmd)
	rootCmd.AddCommand(format.FormatCmd)
	rootCmd.AddCommand(install.InstallCmd)
	rootCmd.AddCommand(lint.LintCmd)
	rootCmd.AddCommand(restore.RestoreCmd)
	rootCmd.AddCommand(sync.SyncCmd)
	rootCmd.AddCommand(test.TestCmd)
}
