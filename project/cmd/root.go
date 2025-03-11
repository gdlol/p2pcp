package cmd

import (
	"project/cmd/build"
	"project/cmd/codecov"
	"project/cmd/format"
	"project/cmd/github"
	"project/cmd/install"
	"project/cmd/lint"
	"project/cmd/publish"
	"project/cmd/restore"
	"project/cmd/sync"
	"project/cmd/test"
	"project/cmd/update"

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
	rootCmd.AddCommand(restore.RestoreCmd)
	rootCmd.AddCommand(build.BuildCmd)
	rootCmd.AddCommand(lint.LintCmd)
	rootCmd.AddCommand(format.FormatCmd)
	rootCmd.AddCommand(install.InstallCmd)
	rootCmd.AddCommand(sync.SyncCmd)
	rootCmd.AddCommand(test.TestCmd)
	rootCmd.AddCommand(codecov.CodecovCommand)
	rootCmd.AddCommand(publish.PublishCmd)
	rootCmd.AddCommand(update.UpdateCmd)
	rootCmd.AddCommand(github.GithubCommand)
}
