package restore

import (
	"project/pkg/workspace"

	"github.com/spf13/cobra"
)

var RestoreCmd = &cobra.Command{
	Use: "restore",
	RunE: func(cmd *cobra.Command, args []string) error {
		for _, module := range workspace.GetModules() {
			err := workspace.RunWithChdir(module, "go", "mod", "download")
			if err != nil {
				return err
			}
		}
		return nil
	},
}
