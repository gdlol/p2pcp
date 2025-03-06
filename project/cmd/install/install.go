package install

import (
	"path/filepath"
	"project/pkg/project"
	"project/pkg/workspace"

	"github.com/spf13/cobra"
)

var InstallCmd = &cobra.Command{
	Use: "install",
	Run: func(cmd *cobra.Command, args []string) {
		projectPath := workspace.GetProjectPath()
		workspace.Run("go", "install", filepath.Join(projectPath, project.Name))
	},
}
