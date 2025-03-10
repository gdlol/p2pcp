package test

import (
	"os"
	"path/filepath"
	"project/pkg/project"
	"project/pkg/workspace"

	"github.com/spf13/cobra"
)

func setup() {
	projectPath := workspace.GetProjectPath()
	workspace.ResetDir(filepath.Join(projectPath, "bin", "integration"))
	os.Setenv("CGO_ENABLED", "0")

	// Build binary with coverage
	workspace.Run("go", "build",
		"-cover",
		"-o", filepath.Join(projectPath, "bin", "integration", project.Name),
		filepath.Join(projectPath, project.Name))

	// Build test cmd tool
	workspace.Run("go", "build",
		"-o", filepath.Join(projectPath, "bin", "integration", "test"),
		filepath.Join(projectPath, "test"))

	// Build test image
	workspace.Run(
		"docker", "build",
		"--file", filepath.Join(projectPath, "test/testdata/integration/test.Dockerfile"),
		"--tag", "local/test",
		projectPath)
}

var setupCmd = &cobra.Command{
	Use: "setup",
	Run: func(cmd *cobra.Command, args []string) {
		setup()
	},
}
