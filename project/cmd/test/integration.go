package test

import (
	"path/filepath"
	"project/cmd/build"
	"project/pkg/workspace"

	"github.com/spf13/cobra"
)

func RunIntegration() error {
	if err := build.Run(); err != nil {
		return err
	}
	projectPath := workspace.GetProjectPath()
	err := workspace.Run(
		"docker", "build",
		"--file", filepath.Join(projectPath, "test/testdata/integration/test.Dockerfile"),
		"--tag", "local/test",
		projectPath)
	if err != nil {
		return err
	}
	return workspace.Run("go", "run", "test", "integration")
}

var IntegrationCmd = &cobra.Command{
	Use: "integration",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunIntegration()
	},
}
