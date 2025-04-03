package lint

import (
	"project/internal/tasks"

	"github.com/spf13/cobra"
)

func Run() {
	tasks.CSpell()
	tasks.PrettierCheck()
	tasks.GoFormatCheck()
}

var LintCmd = &cobra.Command{
	Use: "lint",
	Run: func(cmd *cobra.Command, args []string) {
		Run()
	},
}
