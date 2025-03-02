package sync

import (
	"project/pkg/workspace"

	"github.com/spf13/cobra"
)

var SyncCmd = &cobra.Command{
	Use: "sync",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectPath := workspace.GetProjectPath()
		err := workspace.RunWithChdir(projectPath, "go", "work", "sync")
		if err != nil {
			return err
		}

		for _, module := range workspace.GetModules() {
			err := workspace.RunWithChdir(module, "go", "mod", "tidy")
			if err != nil {
				return err
			}
		}

		return nil
	},
}
