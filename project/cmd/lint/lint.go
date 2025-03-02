package lint

import (
	"project/internal/tasks"

	"github.com/spf13/cobra"
)

var LintCmd = &cobra.Command{
	Use: "lint",
	RunE: func(cmd *cobra.Command, args []string) error {
		err := tasks.CSpell()
		if err != nil {
			return err
		}
		err = tasks.PrettierCheck()
		if err != nil {
			return err
		}
		err = tasks.GoFormatCheck()
		return err
	},
}
