package sync

import (
	"project/pkg/workspace"

	"github.com/spf13/cobra"
)

func Run() {
	projectPath := workspace.GetProjectPath()

	// Unclear behavior of go work sync, need 2 runs to get stable results.
	for range 2 {
		workspace.RunWithChdir(projectPath, "go", "work", "sync")
		for _, module := range workspace.GetModules() {
			workspace.RunWithChdir(module, "go", "mod", "tidy")
		}
	}
}

var SyncCmd = &cobra.Command{
	Use: "sync",
	Run: func(cmd *cobra.Command, args []string) {
		Run()
	},
}
