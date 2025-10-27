package test

import (
	"fmt"
	"path/filepath"
	"project/pkg/project"
	"project/pkg/workspace"

	"github.com/spf13/cobra"
)

func Run() {
	setup()

	// spell-checker: ignore testcache
	workspace.Run("go", "clean", "-testcache")

	projectPath := workspace.GetProjectPath()
	logsPath := filepath.Join(projectPath, ".local/logs/integration")
	coveragePath := filepath.Join(projectPath, ".local/coverage")
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
}

var TestCmd = &cobra.Command{
	Use: "test",
	Run: func(cmd *cobra.Command, args []string) {
		Run()
	},
}

func init() {
	TestCmd.AddCommand(setupCmd)
}
