package cmd

import (
	project "build/internal"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var BuildCmd = &cobra.Command{
	Use: "build",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectPath := project.GetProjectPath()
		err := project.Run("go", "build",
			"-o", filepath.Join(projectPath, "bin")+string(os.PathSeparator),
			filepath.Join(projectPath, "p2pcp"))
		return err
	},
}
