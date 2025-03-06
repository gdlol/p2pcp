package build

import (
	"os"
	"path/filepath"
	"project/pkg/project"
	"project/pkg/workspace"

	"github.com/spf13/cobra"
)

func Run() {
	os.Setenv("CGO_ENABLED", "0")
	projectPath := workspace.GetProjectPath()
	workspace.ResetDir(filepath.Join(projectPath, "bin"))
	for _, module := range workspace.GetModules() {
		moduleName := filepath.Base(module)
		output := "/dev/null"
		if moduleName == project.Name {
			output = filepath.Join(projectPath, "bin", moduleName)
		}
		workspace.Run("go", "build", "-o", output, module)
	}
}

var BuildCmd = &cobra.Command{
	Use: "build",
	Run: func(cmd *cobra.Command, args []string) {
		Run()
	},
}
