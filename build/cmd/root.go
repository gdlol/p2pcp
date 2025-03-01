package cmd

import (
	build "build/cmd/build"
	format "build/cmd/format"
	install "build/cmd/install"
	lint "build/cmd/lint"
	test "build/cmd/test"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "build",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.AddCommand(build.BuildCmd)
	rootCmd.AddCommand(install.InstallCmd)
	rootCmd.AddCommand(test.TestCmd)
	rootCmd.AddCommand(lint.LintCmd)
	rootCmd.AddCommand(format.FormatCmd)
}
