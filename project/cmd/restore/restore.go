package restore

import (
	"fmt"
	"project/pkg/project"
	"project/pkg/workspace"

	"github.com/spf13/cobra"
)

const pythonVersion = "3.12"
const codecovVersion = "10"

var RestoreCmd = &cobra.Command{
	Use: "restore",
	Run: func(cmd *cobra.Command, args []string) {
		workspace.Run("uv", "python", "install", "--preview", "--default", pythonVersion)
		workspace.Run("uv", "tool", "install", fmt.Sprintf("codecov-cli@%s", codecovVersion))
		for _, module := range workspace.GetModules() {
			workspace.RunWithChdir(module, "go", "mod", "download")
		}

		// Create buildx builder
		create := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					create = true
				}
			}()
			workspace.Run("docker", "buildx", "use", project.Name)
		}()
		if create {
			workspace.Run("docker", "buildx", "create",
				"--name", project.Name,
				"--use",
				"--bootstrap",
				"--driver-opt", "network=host")
		}
	},
}
