package restore

import (
	"project/pkg/workspace"

	"github.com/spf13/cobra"
)

var RestoreCmd = &cobra.Command{
	Use: "restore",
	Run: func(cmd *cobra.Command, args []string) {
		for _, module := range workspace.GetModules() {
			workspace.RunWithChdir(module, "go", "mod", "download")
		}
	},
}
