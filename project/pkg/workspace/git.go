package workspace

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

func GetRepoInfo() (owner string, repoName string) {
	projectPath := GetProjectPath()
	repo, err := git.PlainOpen(projectPath)
	Check(err)
	remote, err := repo.Remote("origin")
	Check(err)
	if len(remote.Config().URLs) == 0 {
		panic("no remote URL found")
	}
	remoteURL, err := url.Parse(remote.Config().URLs[0])
	Check(err)
	parts := strings.Split(remoteURL.Path, "/")
	owner = parts[1]
	repoName = parts[2]
	return owner, repoName
}

func GetCurrentBranch() string {
	projectPath := GetProjectPath()
	repo, err := git.PlainOpen(projectPath)
	Check(err)
	head, err := repo.Head()
	Check(err)
	return head.Name().Short()
}

func Push(ctx context.Context, branch string, token string) {
	projectPath := GetProjectPath()
	repo, err := git.PlainOpen(projectPath)
	Check(err)

	refSpec := fmt.Sprintf("+refs/heads/%s:refs/heads/%s", branch, branch)
	err = repo.PushContext(ctx, &git.PushOptions{
		RemoteName: "origin",
		RefSpecs:   []config.RefSpec{config.RefSpec(refSpec)},
		Force:      true,
		Auth: &http.BasicAuth{
			Username: "git",
			Password: token,
		},
	})
	if err != git.NoErrAlreadyUpToDate {
		Check(err)
	}
}
