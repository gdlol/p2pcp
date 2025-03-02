package build

import (
	"path/filepath"
	"project/pkg/project"
	"project/pkg/workspace"

	"github.com/spf13/cobra"
)

var BuildCmd = &cobra.Command{
	Use: "build",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectPath := workspace.GetProjectPath()
		for _, module := range workspace.GetModules() {
			output := "/dev/null"
			if filepath.Base(module) == project.Name {
				output = filepath.Join(projectPath, "bin", project.Name)
			}
			err := workspace.Run("go", "build", "-o", output, module)
			if err != nil {
				return err
			}
		}
		return nil
	},
}
