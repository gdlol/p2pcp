package publish

import (
	"log/slog"
	"os"
	"os/exec"
	"project/cmd/build"
	"project/pkg/workspace"
	"strings"

	"github.com/spf13/cobra"
)

const registry = "ghcr.io"

func Run() {
	slog.Info("Logging in to the container registry...")
	token := os.Getenv("CR_PAT")
	cmd := exec.Command("docker", "login", registry, "-u", "USERNAME", "--password-stdin")
	cmd.Stdin = strings.NewReader(token)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	workspace.Check(err)

	slog.Info("Publishing multi-arch image...")
	build.BuildImage(true)
}

var PublishCmd = &cobra.Command{
	Use: "publish",
	Run: func(cmd *cobra.Command, args []string) {
		Run()
	},
}
