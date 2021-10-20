package nativeGit

import (
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/otiai10/copy"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"time"
)

var fetchRefSpec = []config.RefSpec{
	"refs/heads/*:refs/heads/*",
}

type GitopsRepoCache struct {
	gitopsRepo              string
	gitopsRepoDeployKeyPath string
	repo                    *git.Repository
	cachePath               string
	stopCh                  chan struct{}
}

func NewGitopsRepoCache(
	gitopsRepo string,
	gitopsRepoDeployKeyPath string,
	stopCh chan struct{},
) (*GitopsRepoCache, error) {
	cachePath, repo, err := CloneToTmpFs(gitopsRepo, gitopsRepoDeployKeyPath)
	if err != nil {
		return nil, err
	}

	return &GitopsRepoCache{
		gitopsRepo:              gitopsRepo,
		gitopsRepoDeployKeyPath: gitopsRepoDeployKeyPath,
		repo:                    repo,
		cachePath:               cachePath,
		stopCh:                  stopCh,
	}, nil
}

func (r *GitopsRepoCache) Run() {
	for {
		r.syncGitRepo()

		select {
		case <-r.stopCh:
			logrus.Infof("cleaning up git repo cache at %s", r.cachePath)
			TmpFsCleanup(r.cachePath)
			return
		case <-time.After(30 * time.Second):
		}
	}
}

func (r *GitopsRepoCache) syncGitRepo() {
	publicKeys, err := ssh.NewPublicKeysFromFile("git", r.gitopsRepoDeployKeyPath, "")
	if err != nil {
		logrus.Errorf("cannot generate public key from private: %s", err.Error())
	}

	w, err := r.repo.Worktree()
	if err != nil {
		logrus.Errorf("could not get worktree: %s", err)
		return
	}

	w.Pull(&git.PullOptions{
		Auth:       publicKeys,
		RemoteName: "origin",
	})
	if err == git.NoErrAlreadyUpToDate {
		return
	}
	if err != nil {
		logrus.Errorf("could not fetch: %s", err)
	}
}

func (r *GitopsRepoCache) InstanceForRead() *git.Repository {
	return r.repo
}

func (r *GitopsRepoCache) InstanceForWrite() (*git.Repository, string, error) {
	tmpPath, err := ioutil.TempDir("", "gitops-")
	if err != nil {
		errors.WithMessage(err, "couldn't get temporary directory")
	}

	err = copy.Copy(r.cachePath, tmpPath)
	if err != nil {
		errors.WithMessage(err, "could not make copy of repo")
	}

	copiedRepo, err := git.PlainOpen(tmpPath)
	if err != nil {
		return nil, "", fmt.Errorf("cannot open git repository at %s: %s", tmpPath, err)
	}

	return copiedRepo, tmpPath, nil
}

func (r *GitopsRepoCache) CleanupWrittenRepo(path string) error {
	return os.RemoveAll(path)
}

func (r *GitopsRepoCache) Invalidate() {
	r.syncGitRepo()
}
