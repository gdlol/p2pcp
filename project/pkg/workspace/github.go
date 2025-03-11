package workspace

import (
	"context"
	"os"

	"github.com/google/go-github/v69/github"
)

func getGitHubClient() *github.Client {
	token := os.Getenv("GITHUB_TOKEN")
	return github.NewClient(nil).WithAuthToken(token)
}

func GetDefaultBranch(ctx context.Context) string {
	client := getGitHubClient()
	owner, repoName := GetRepoInfo()
	repo, _, err := client.Repositories.Get(context.Background(), owner, repoName)
	Check(err)
	return repo.GetDefaultBranch()
}

// Creates pull request from current branch to default branch
func CreatePullRequest(ctx context.Context) {
	client := getGitHubClient()
	defaultBranch := GetDefaultBranch(ctx)
	currentBranch := GetCurrentBranch()
	if defaultBranch == currentBranch {
		panic("current branch is the default branch")
	}
	owner, repoName := GetRepoInfo()
	pr := &github.NewPullRequest{
		Title: &currentBranch,
		Head:  &currentBranch,
		Base:  &defaultBranch,
		Draft: github.Ptr(true),
	}
	_, _, err := client.PullRequests.Create(ctx, owner, repoName, pr)
	Check(err)
}
