package github

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"project/pkg/workspace"

	"github.com/google/go-github/v69/github"
)

func getGitHubClient() *github.Client {
	token := os.Getenv("GITHUB_TOKEN")
	return github.NewClient(nil).WithAuthToken(token)
}

func GetDefaultBranch(ctx context.Context) string {
	client := getGitHubClient()
	owner, repoName := workspace.GetRepoInfo()
	repo, _, err := client.Repositories.Get(ctx, owner, repoName)
	workspace.Check(err)
	return repo.GetDefaultBranch()
}

// Creates pull request from current branch to default branch
func CreatePullRequest(ctx context.Context) {
	client := getGitHubClient()
	defaultBranch := GetDefaultBranch(ctx)
	currentBranch := workspace.GetCurrentBranch()
	if defaultBranch == currentBranch {
		panic("current branch is the default branch")
	}
	owner, repoName := workspace.GetRepoInfo()
	pr := &github.NewPullRequest{
		Title: &currentBranch,
		Head:  &currentBranch,
		Base:  &defaultBranch,
		Draft: github.Ptr(true),
	}
	_, _, err := client.PullRequests.Create(ctx, owner, repoName, pr)
	workspace.Check(err)
}

func CreateRelease(ctx context.Context, token string, tag string, name string, filePaths []string) {
	client := github.NewClient(nil).WithAuthToken(token)
	owner, repoName := workspace.GetRepoInfo()

	// Create the release
	slog.Info(fmt.Sprintf("Creating release %s", tag))
	release := &github.RepositoryRelease{
		TagName: github.Ptr(tag),
		Name:    github.Ptr(name),
		Draft:   github.Ptr(true),
	}
	createdRelease, _, err := client.Repositories.CreateRelease(ctx, owner, repoName, release)
	workspace.Check(err)

	// Upload assets
	for _, filePath := range filePaths {
		slog.Info(fmt.Sprintf("Uploading asset %s", filePath))
		file, err := os.Open(filePath)
		workspace.Check(err)
		defer file.Close()

		_, _, err = client.Repositories.UploadReleaseAsset(ctx, owner, repoName, *createdRelease.ID, &github.UploadOptions{
			Name: filepath.Base(filePath),
		}, file)
		workspace.Check(err)
	}
}
