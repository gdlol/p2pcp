package cmd

import (
	"build/internal/tasks"

	"github.com/spf13/cobra"
)

var FormatCmd = &cobra.Command{
	Use: "format",
	RunE: func(cmd *cobra.Command, args []string) error {
		err := tasks.PrettierFormat()
		if err != nil {
			return err
		}
		err = tasks.GoFormat()
		return err
	},
}
