package test

import (
	"path/filepath"
	"project/cmd/build"
	"project/pkg/workspace"

	"github.com/spf13/cobra"
)

func setup() error {
	err := build.Run()
	if err != nil {
		return err
	}

	projectPath := workspace.GetProjectPath()
	err = workspace.Run(
		"docker", "build",
		"--file", filepath.Join(projectPath, "test/testdata/integration/test.Dockerfile"),
		"--tag", "local/test",
		projectPath)
	if err != nil {
		return err
	}

	return nil
}

var SetupCmd = &cobra.Command{
	Use: "setup",
	RunE: func(cmd *cobra.Command, args []string) error {
		return setup()
	},
}

var TestCmd = &cobra.Command{
	Use: "test",
	RunE: func(cmd *cobra.Command, args []string) error {
		clean, err := cmd.Flags().GetBool("clean")
		if err != nil {
			return err
		}

		if clean {
			// spell-checker: ignore testcache
			err = workspace.Run("go", "clean", "-testcache")
			if err != nil {
				return err
			}
		}

		err = setup()
		if err != nil {
			return err
		}

		for _, module := range workspace.GetModules() {
			args := []string{"test", "-v", filepath.Join(module, "...")}
			if filepath.Base(module) == "test" {
				args = append(args, "-p", "1") // Stream logs for integration tests
			}
			err := workspace.Run("go", args...)
			if err != nil {
				return err
			}
		}
		return nil
	},
}

func init() {
	TestCmd.Flags().Bool("clean", false, "Clean test cache before running tests.")

	TestCmd.AddCommand(SetupCmd)
}
