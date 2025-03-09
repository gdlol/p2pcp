package test

import (
	"fmt"
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

var SetupCmd = &cobra.Command{
	Use: "setup",
	Run: func(cmd *cobra.Command, args []string) {
		setup()
	},
}

var TestCmd = &cobra.Command{
	Use: "test",
	Run: func(cmd *cobra.Command, args []string) {
		clean, err := cmd.Flags().GetBool("clean")
		workspace.Check(err)

		if clean {
			// spell-checker: ignore testcache
			workspace.Run("go", "clean", "-testcache")
		}

		setup()

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
			workspace.Run("go", args...)
		}

		// Generate coverage report
		// spell-checker: ignore covdata textfmt
		mergedPath := filepath.Join(coveragePath, "merged")
		workspace.ResetDir(mergedPath)
		workspace.Run("go", "tool", "covdata", "merge",
			"-i", fmt.Sprintf("%s/integration,%s/unit", coveragePath, coveragePath),
			"-o", mergedPath)
		workspace.Run("go", "tool", "covdata", "textfmt",
			"-i", mergedPath,
			"-o", filepath.Join(coveragePath, "coverage.txt"))
		workspace.Run("go", "tool", "cover",
			"-html", filepath.Join(coveragePath, "coverage.txt"),
			"-o", filepath.Join(coveragePath, "coverage.html"))
	},
}

func init() {
	TestCmd.Flags().Bool("clean", false, "Clean test cache before running tests.")

	TestCmd.AddCommand(SetupCmd)
}
