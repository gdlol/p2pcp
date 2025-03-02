package install

import (
	"path/filepath"
	"project/pkg/project"
	"project/pkg/workspace"

	"github.com/spf13/cobra"
)

var InstallCmd = &cobra.Command{
	Use: "install",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectPath := workspace.GetProjectPath()
		return workspace.Run("go", "install", filepath.Join(projectPath, project.Name))
	},
}
