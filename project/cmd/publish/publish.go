package publish

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"project/cmd/build"
	"project/pkg/project"
	"project/pkg/workspace"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

func Run(ctx context.Context) error {
	registry := os.Getenv("CR_REGISTRY")
	imageName := os.Getenv("CR_IMAGE_NAME")
	version := os.Getenv("CR_VERSION")
	token := os.Getenv("CR_PAT")
	releaseToken := os.Getenv("RELEASE_TOKEN")

	tags := []string{version}
	isTag := semver.IsValid("v" + version)
	if isTag {
		if version != project.Version {
			return fmt.Errorf("version mismatch: project.Version=%s CR_VERSION=%s", project.Version, version)
		}
		tags = append(tags, "latest")
	}

	slog.Info("Logging in to the container registry...")
	cmd := exec.Command("docker", "login", registry, "-u", "USERNAME", "--password-stdin")
	cmd.Stdin = strings.NewReader(token)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	workspace.Check(err)

	slog.Info("Building binaries...")
	build.BuildBinaries()

	slog.Info("Publishing multi-arch image...")
	build.BuildImage(registry, imageName, tags, true)

	if isTag {
		slog.Info("Creating release...")
		createRelease(ctx, releaseToken)
	}
	return nil
}

var PublishCmd = &cobra.Command{
	Use: "publish",
	RunE: func(cmd *cobra.Command, args []string) error {
		return Run(cmd.Context())
	},
}

func init() {
	PublishCmd.AddCommand(releaseCmd)
}
