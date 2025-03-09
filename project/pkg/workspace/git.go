package workspace

import (
	"net/url"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"golang.org/x/mod/semver"
)

func GetOriginURL() string {
	projectPath := GetProjectPath()
	repo, err := git.PlainOpen(projectPath)
	Check(err)
	remote, err := repo.Remote("origin")
	Check(err)
	if len(remote.Config().URLs) == 0 {
		panic("no remote URL found")
	}
	return remote.Config().URLs[0]
}

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

func getTags() []string {
	projectPath := GetProjectPath()
	repo, err := git.PlainOpen(projectPath)
	Check(err)
	tags, err := repo.Tags()
	Check(err)
	var tagList []string
	tags.ForEach(func(ref *plumbing.Reference) error {
		tagList = append(tagList, ref.Name().Short())
		return nil
	})
	return tagList
}

func GetLatestTag() string {
	tags := getTags()
	if len(tags) == 0 {
		return "0.0.0"
	}
	semver.Sort(tags)
	return tags[len(tags)-1]
}
