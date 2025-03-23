package github

import (
	"project/pkg/github"

	"github.com/spf13/cobra"
)

var pullRequestCmd = &cobra.Command{
	Use: "pr",
	Run: func(cmd *cobra.Command, args []string) {
		github.CreatePullRequest(cmd.Context())
	},
}

var GithubCommand = &cobra.Command{
	Use: "github",
}

func init() {
	GithubCommand.AddCommand(pullRequestCmd)
}
