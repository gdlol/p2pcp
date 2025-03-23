package build

import (
	"path/filepath"
	"project/pkg/project"
	"project/pkg/workspace"
	"strings"
	"sync"

	"github.com/spf13/cobra"
)

func GetReleaseArtifactsPath() string {
	return filepath.Join(workspace.GetProjectPath(), "bin/release")
}

func PackBinaries() {
	binariesPath := GetBinariesPath()
	artifactsPath := GetReleaseArtifactsPath()
	workspace.ResetDir(artifactsPath)

	var wg sync.WaitGroup
	wg.Add(len(platformEnvs))
	for platform := range platformEnvs {
		go func() {
			defer wg.Done()
			outputName := strings.Join([]string{project.Name, project.Version, strings.Replace(platform, "/", "_", -1)}, "_")

			binaryPath := filepath.Join(binariesPath, platform, project.Name)
			if strings.HasPrefix(platform, "windows") {
				binaryPath += ".exe"
				outputPath := filepath.Join(artifactsPath, outputName+".zip")
				workspace.RunWithChdir(filepath.Dir(binaryPath),
					"zip", "-j", outputPath, filepath.Base(binaryPath))
			} else {
				outputPath := filepath.Join(artifactsPath, outputName+".tar.gz")
				workspace.RunWithChdir(filepath.Dir(binaryPath),
					"tar", "czf", outputPath, filepath.Base(binaryPath))
			}
		}()
	}
	wg.Wait()
}

var releaseCmd = &cobra.Command{
	Use: "release",
	Run: func(cmd *cobra.Command, args []string) {
		BuildBinaries()
		PackBinaries()
	},
}
