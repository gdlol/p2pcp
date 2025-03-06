package sync

import (
	"project/pkg/workspace"

	"github.com/spf13/cobra"
)

func Run() {
	projectPath := workspace.GetProjectPath()
	workspace.RunWithChdir(projectPath, "go", "work", "sync")

	for _, module := range workspace.GetModules() {
		workspace.RunWithChdir(module, "go", "mod", "tidy")
	}
}

var SyncCmd = &cobra.Command{
	Use: "sync",
	Run: func(cmd *cobra.Command, args []string) {
		Run()
	},
}
