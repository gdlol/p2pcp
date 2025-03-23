package publish

import (
	"context"
	"os"
	"path/filepath"
	"project/cmd/build"
	"project/pkg/github"
	"project/pkg/project"
	"project/pkg/workspace"

	"github.com/spf13/cobra"
)

func createRelease(ctx context.Context, token string) {
	build.PackBinaries()

	artifactsPath := build.GetReleaseArtifactsPath()
	files := workspace.ListDir(artifactsPath)
	filePaths := make([]string, len(files))
	for i, file := range files {
		filePaths[i] = filepath.Join(artifactsPath, file)
	}
	github.CreateRelease(ctx, token, project.Version, project.Version, filePaths)
}

var releaseCmd = &cobra.Command{
	Use: "release",
	Run: func(cmd *cobra.Command, args []string) {
		token := os.Getenv("RELEASE_TOKEN")

		build.BuildBinaries()
		createRelease(cmd.Context(), token)
	},
}
