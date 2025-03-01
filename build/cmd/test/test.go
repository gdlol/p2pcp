package test

import (
	project "build/internal"
	"path/filepath"

	"github.com/spf13/cobra"
)

var TestCmd = &cobra.Command{
	Use: "test",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectPath := project.GetProjectPath()
		return project.Run("go", "test", filepath.Join(projectPath, "p2pcp", "..."))
	},
}
