package codecov

import (
	"path/filepath"
	"project/pkg/workspace"

	"github.com/spf13/cobra"
)

var CodecovCommand = &cobra.Command{
	Use: "codecov",
	Run: func(cmd *cobra.Command, args []string) {
		projectPath := workspace.GetProjectPath()
		coverageFile := filepath.Join(projectPath, "coverage/coverage.txt")
		commitHash := workspace.GitCommitHash()
		workspace.Run("codecovcli", "--verbose", "upload-process",
			"--commit-sha", commitHash,
			"--disable-search", "--fail-on-error",
			"--file", coverageFile)
	},
}
