package cmd

import (
	"os"
	"path/filepath"
	"project"

	"github.com/spf13/cobra"
)

var BuildCmd = &cobra.Command{
	Use: "build",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectPath := project.GetProjectPath()
		return project.Run("go", "build",
			"-o", filepath.Join(projectPath, "bin")+string(os.PathSeparator),
			filepath.Join(projectPath, "p2pcp"))
	},
}
