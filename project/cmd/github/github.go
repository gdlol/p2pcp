package github

import (
	"project/pkg/github"

	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use: "push",
	Run: func(cmd *cobra.Command, args []string) {
		github.Push(cmd.Context())
	},
}

var pullRequestCmd = &cobra.Command{
	Use: "pr",
	Run: func(cmd *cobra.Command, args []string) {
		github.Push(cmd.Context())
		github.CreatePullRequest(cmd.Context())
	},
}

var GithubCommand = &cobra.Command{
	Use: "github",
}

func init() {
	GithubCommand.AddCommand(pushCmd)
	GithubCommand.AddCommand(pullRequestCmd)
}
