package publish

import (
	"log/slog"
	"os"
	"os/exec"
	"project/cmd/build"
	"project/pkg/workspace"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

func Run() error {
	registry := os.Getenv("CR_REGISTRY")
	imageName := os.Getenv("CR_IMAGE_NAME")
	version := os.Getenv("CR_VERSION")
	token := os.Getenv("CR_PAT")

	tags := []string{version}
	if semver.IsValid(version) {
		tags = append(tags, "latest")
	}

	slog.Info("Logging in to the container registry...")
	cmd := exec.Command("docker", "login", registry, "-u", "USERNAME", "--password-stdin")
	cmd.Stdin = strings.NewReader(token)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	workspace.Check(err)

	slog.Info("Publishing multi-arch image...")
	build.BuildImage(registry, imageName, tags, true)
	return nil
}

var PublishCmd = &cobra.Command{
	Use: "publish",
	RunE: func(cmd *cobra.Command, args []string) error {
		return Run()
	},
}
