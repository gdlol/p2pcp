package build

import (
	"os"
	"path/filepath"
	"project/pkg/project"
	"project/pkg/workspace"

	"github.com/spf13/cobra"
)

var vendoredGoWorkPath = filepath.Join(workspace.GetProjectPath(), "go.vendored.work")

func Run() {
	os.Setenv("CGO_ENABLED", "0")
	projectPath := workspace.GetProjectPath()
	binPath := filepath.Join(projectPath, ".local/bin")

	// Output binaries for the main module.
	for _, module := range workspace.GetModules() {
		moduleName := filepath.Base(module)
		output := "/dev/null"
		if moduleName == project.Name {
			output = filepath.Join(binPath, moduleName)
		}
		restore := workspace.SetEnv("GOWORK", vendoredGoWorkPath)
		defer restore()
		workspace.Run("go", "build", "-o", output, module)
	}
}

var BuildCmd = &cobra.Command{
	Use: "build",
	RunE: func(cmd *cobra.Command, args []string) error {
		Run()
		return generateDocs()
	},
}

func init() {
	BuildCmd.AddCommand(docsCmd)
	BuildCmd.AddCommand(dockerCmd)
	BuildCmd.AddCommand(releaseCmd)
}
