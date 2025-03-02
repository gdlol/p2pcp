package test

import (
	"path/filepath"
	"project"

	"github.com/spf13/cobra"
)

var TestCmd = &cobra.Command{
	Use: "test",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectPath := project.GetProjectPath()
		err := project.Run("go", "test", filepath.Join(projectPath, "p2pcp", "..."))
		if err != nil {
			return err
		}
		err = project.Run("go", "test", filepath.Join(projectPath, "test", "..."))
		return err
	},
}
