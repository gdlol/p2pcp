package lint

import (
	"project/internal/tasks"

	"github.com/spf13/cobra"
)

var LintCmd = &cobra.Command{
	Use: "lint",
	Run: func(cmd *cobra.Command, args []string) {
		tasks.CSpell()
		tasks.PrettierCheck()
		tasks.GoFormatCheck()
	},
}
