package test

import (
	"fmt"
	"path/filepath"
	"project/pkg/project"
	"project/pkg/workspace"

	"github.com/spf13/cobra"
)

func setup() error {
	projectPath := workspace.GetProjectPath()
	workspace.ResetDir(filepath.Join(projectPath, "bin", "integration"))

	// Build binary with coverage
	err := workspace.Run("go", "build",
		"-cover",
		"-o", filepath.Join(projectPath, "bin", "integration", project.Name),
		filepath.Join(projectPath, project.Name))
	if err != nil {
		return err
	}

	// Build test cmd tool
	err = workspace.Run("go", "build",
		"-o", filepath.Join(projectPath, "bin", "integration", "test"),
		filepath.Join(projectPath, "test"))
	if err != nil {
		return err
	}

	// Build test image
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
		workspace.Check(err)
		cover, err := cmd.Flags().GetBool("cover")
		workspace.Check(err)

		if clean || cover {
			// spell-checker: ignore testcache
			err = workspace.Run("go", "clean", "-testcache")
			workspace.Check(err)
		}

		err = setup()
		if err != nil {
			return err
		}

		projectPath := workspace.GetProjectPath()
		logsPath := filepath.Join(projectPath, "logs/integration")
		coveragePath := filepath.Join(projectPath, "coverage")
		workspace.ResetDir(coveragePath)
		workspace.ResetDir(logsPath)

		// Run tests
		for _, module := range workspace.GetModules() {
			args := []string{"test", "-v", filepath.Join(module, "...")}
			if filepath.Base(module) == project.Name {
				// spell-checker: ignore covermode coverprofile
				coverprofile := filepath.Join(coveragePath, "unit/coverage.out")
				workspace.ResetDir(filepath.Dir(coverprofile))
				args = append(args, "-cover",
					fmt.Sprintf("-test.gocoverdir=%s", filepath.Join(coveragePath, "unit")))
			}
			if filepath.Base(module) == "test" {
				// Stream logs for integration tests
				args = append(args, "-p", "1")
			}
			err := workspace.Run("go", args...)
			workspace.Check(err)
		}

		// Generate coverage report
		if cover {
			// spell-checker: ignore covdata textfmt
			mergedPath := filepath.Join(coveragePath, "merged")
			workspace.ResetDir(mergedPath)
			err = workspace.Run("go", "tool", "covdata", "merge",
				"-i", fmt.Sprintf("%s/integration,%s/unit", coveragePath, coveragePath),
				"-o", mergedPath)
			workspace.Check(err)
			err = workspace.Run("go", "tool", "covdata", "textfmt",
				"-i", mergedPath,
				"-o", filepath.Join(coveragePath, "coverage.txt"))
			workspace.Check(err)
			err = workspace.Run("go", "tool", "cover",
				"-html", filepath.Join(coveragePath, "coverage.txt"),
				"-o", filepath.Join(coveragePath, "coverage.html"))
			workspace.Check(err)
		}

		return nil
	},
}

func init() {
	TestCmd.Flags().Bool("clean", false, "Clean test cache before running tests.")
	TestCmd.Flags().Bool("cover", false, "Run integration tests.")

	TestCmd.AddCommand(SetupCmd)
}
