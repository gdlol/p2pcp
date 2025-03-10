package publish

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"project/cmd/build"
	"project/pkg/project"
	"project/pkg/workspace"
	"strings"

	"github.com/spf13/cobra"
)

func Run() error {
	registry := os.Getenv("CR_REGISTRY")
	imageName := os.Getenv("CR_IMAGE_NAME")
	version := os.Getenv("CR_VERSION")
	token := os.Getenv("CR_PAT")

	if version != project.Version {
		return fmt.Errorf("version mismatch: project.Version=%s CR_VERSION=%s", project.Version, version)
	}

	slog.Info("Logging in to the container registry...")
	cmd := exec.Command("docker", "login", registry, "-u", "USERNAME", "--password-stdin")
	cmd.Stdin = strings.NewReader(token)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	workspace.Check(err)

	slog.Info("Publishing multi-arch image...")
	build.BuildImage(registry, imageName, version, true)
	return nil
}

var PublishCmd = &cobra.Command{
	Use: "publish",
	RunE: func(cmd *cobra.Command, args []string) error {
		return Run()
	},
}
