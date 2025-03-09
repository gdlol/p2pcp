package build

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"project/pkg/project"
	"project/pkg/workspace"
	"strings"
	"sync"
)

type platformEnv struct {
	goOS   string
	goARCH string
	goARM  string
}

var platformEnvs = map[string]platformEnv{
	"darwin/amd64":  {"darwin", "amd64", ""},
	"darwin/arm64":  {"darwin", "arm64", ""},
	"freebsd/amd64": {"freebsd", "amd64", ""},
	"linux/amd64":   {"linux", "amd64", ""},
	"linux/arm64":   {"linux", "arm64", ""},
	"linux/arm/v7":  {"linux", "arm", "7"},
	"linux/riscv64": {"linux", "riscv64", ""},
	"windows/amd64": {"windows", "amd64", ""},
	"windows/arm64": {"windows", "arm64", ""},
}

func buildBinaries() {
	slog.Info("Building multi-arch binaries...")
	restore := workspace.SetEnv("CGO_ENABLED", "0")
	defer restore()

	projectPath := workspace.GetProjectPath()
	binariesPath := filepath.Join(projectPath, "bin/docker")
	workspace.ResetDir(binariesPath)
	var wg sync.WaitGroup
	wg.Add(len(platformEnvs))
	for platform, platformEnv := range platformEnvs {
		go func() {
			defer wg.Done()
			outputPath := filepath.Join(binariesPath, platform, project.Name)
			if strings.HasPrefix(platform, "windows") {
				outputPath += ".exe"
			}
			workspace.ResetDir(filepath.Dir(outputPath))

			cmd := []string{
				"go", "build",
				"-ldflags", "-s -w",
				"-trimpath",
				"-o", outputPath,
				filepath.Join(projectPath, project.Name),
			}
			log.Println(strings.Join(cmd, " "))
			c := exec.Command(cmd[0], cmd[1:]...)
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			c.Env = os.Environ()
			c.Env = append(c.Env, "GOOS="+platformEnv.goOS)
			c.Env = append(c.Env, "GOARCH="+platformEnv.goARCH)
			c.Env = append(c.Env, "GOARM="+platformEnv.goARM)
			err := c.Run()
			workspace.Check(err)
		}()
	}
	wg.Wait()
}

func BuildImage(publish bool) {
	buildBinaries()

	owner, repoName := workspace.GetRepoInfo()

	// Build multi-arch image.
	projectPath := workspace.GetProjectPath()
	platforms := []string{}
	for platform := range platformEnvs {
		platforms = append(platforms, platform)
	}
	args := []string{
		"buildx", "build",
		"--file", filepath.Join(projectPath, "docker/Dockerfile"),
		"--tag", fmt.Sprintf("%s/%s/%s:latest", project.Registry, owner, repoName),
		"--tag", fmt.Sprintf("%s/%s/%s:%s", project.Registry, owner, repoName, project.Version),
		"--platform", strings.Join(platforms, ",")}
	if publish {
		args = append(args, "--push")
	}
	args = append(args, projectPath)
	workspace.Run("docker", args...)
}
