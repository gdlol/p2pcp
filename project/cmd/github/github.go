package github

import (
	"project/pkg/workspace"

	"github.com/spf13/cobra"
)

var pullRequestCmd = &cobra.Command{
	Use: "pr",
	Run: func(cmd *cobra.Command, args []string) {
		workspace.CreatePullRequest(cmd.Context())
	},
}

var GithubCommand = &cobra.Command{
	Use: "github",
}

func init() {
	GithubCommand.AddCommand(pullRequestCmd)
}
