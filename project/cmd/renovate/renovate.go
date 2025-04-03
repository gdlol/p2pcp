package renovate

import (
	"project/cmd/lint"
	"project/cmd/test"
	"project/cmd/update"
	"project/pkg/github"
	"project/pkg/workspace"

	"github.com/spf13/cobra"
	"golang.org/x/exp/slog"
)

var RenovateCmd = &cobra.Command{
	Use: "renovate",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		projectPath := workspace.GetProjectPath()
		originalBranch := workspace.GetCurrentBranch()
		renovateBranch := "renovate"
		if originalBranch != renovateBranch {
			workspace.RunCtxWithChdir(ctx, projectPath, "git", "switch", "--force-create", "renovate")
			defer workspace.RunCtxWithChdir(ctx, projectPath, "git", "switch", originalBranch)
		}

		update.Run()
		if workspace.CheckDiff() {
			lint.Run()
			test.Run()

			slog.Info("Committing changes...")
			workspace.RunWithChdir(projectPath, "git", "add", ".")
			workspace.RunWithChdir(projectPath, "git", "commit", "--message", "renovate")
			slog.Info("Pushing...")
			github.Push(cmd.Context())
			slog.Info("Creating pull request...")
			github.CreatePullRequest(cmd.Context())
		} else {
			slog.Info("No changes from latest commit.")
		}
	},
}
