package cmd

import (
	"path/filepath"
	"project"

	"github.com/spf13/cobra"
)

var InstallCmd = &cobra.Command{
	Use: "install",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectPath := project.GetProjectPath()
		return project.Run("go", "install", filepath.Join(projectPath, "p2pcp"))
	},
}
