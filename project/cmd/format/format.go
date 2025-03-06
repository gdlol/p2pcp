package format

import (
	"project/internal/tasks"

	"github.com/spf13/cobra"
)

var FormatCmd = &cobra.Command{
	Use: "format",
	Run: func(cmd *cobra.Command, args []string) {
		tasks.PrettierFormat()
		tasks.GoFormat()
	},
}
