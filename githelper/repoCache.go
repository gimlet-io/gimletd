package githelper

import (
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/sirupsen/logrus"
	"time"
)

type RepoCache struct {
	gitopsRepo              string
	gitopsRepoDeployKeyPath string
	repo                    *git.Repository
	repoTmpPath             string
	stopCh                  chan struct{}
}

func NewRepoCache(
	gitopsRepo string,
	gitopsRepoDeployKeyPath string,
	stopCh chan struct{},
) (*RepoCache, error) {
	repoTmpPath, repo, err := CloneToTmpFs(gitopsRepo, gitopsRepoDeployKeyPath)
	if err != nil {
		return nil, err
	}

	return &RepoCache{
		gitopsRepo:              gitopsRepo,
		gitopsRepoDeployKeyPath: gitopsRepoDeployKeyPath,
		repo:                    repo,
		repoTmpPath:             repoTmpPath,
		stopCh:                  stopCh,
	}, nil
}

func (w *RepoCache) Run() {
	for {
		w.syncGitRepo()

		select {
		case <-w.stopCh:
			logrus.Infof("cleaning up git repo cache at %s", w.repoTmpPath)
			TmpFsCleanup(w.repoTmpPath)
			return
		case <-time.After(30 * time.Second):
			fmt.Println("timeout 1")
		}
	}
}

func (w *RepoCache) syncGitRepo() {
	hasChanges, err := RemoteHasChanges(w.repo, w.gitopsRepoDeployKeyPath)

	if hasChanges || err != nil {
		logrus.Info("repo cache is stale, updating")
		err := w.updateRepo()
		if err != nil {
			logrus.Errorf("could not update git repo %s", err)
		}
	}
}

func (w *RepoCache) updateRepo() error {
	defer TmpFsCleanup(w.repoTmpPath)

	repoTmpPath, repo, err := CloneToTmpFs(w.gitopsRepo, w.gitopsRepoDeployKeyPath)
	if err != nil {
		return err
	}

	w.repoTmpPath = repoTmpPath
	w.repo = repo

	return nil
}

func (w *RepoCache) InstanceForRead() *git.Repository {
	return w.repo
}
