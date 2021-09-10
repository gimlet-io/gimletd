package worker

import (
	"encoding/json"
	"fmt"
	"github.com/gimlet-io/gimletd/dx"
	"github.com/gimlet-io/gimletd/git/customScm"
	"github.com/gimlet-io/gimletd/model"
	"github.com/gimlet-io/gimletd/store"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const Dir_RWX_RX_R = 0754

var fetchRefSpec = []config.RefSpec{
	"refs/heads/*:refs/heads/*",
}

type BranchDeleteEventWorker struct {
	tokenManager customScm.NonImpersonatedTokenManager
	cachePath    string
	dao          *store.Store
}

func NewBranchDeleteEventWorker(
	tokenManager customScm.NonImpersonatedTokenManager,
	cachePath string,
	dao *store.Store,
) *BranchDeleteEventWorker {
	branchDeleteEventWorker := &BranchDeleteEventWorker{
		tokenManager: tokenManager,
		cachePath:    cachePath,
		dao:          dao,
	}

	return branchDeleteEventWorker
}

func (r *BranchDeleteEventWorker) Run() {
	for {
		reposWithCleanupPolicy, err := r.dao.ReposWithCleanupPolicy()
		if err != nil {
			logrus.Warnf("could not load repos with cleanup policy: %s", err)
		}

		for _, r := range reposWithCleanupPolicy {
			repoPath := filepath.Join(r.cachePath, strings.ReplaceAll(r, "/", "%"))
			if _, err := os.Stat(repoPath); err == nil { // repo exist
				repo, err := git.PlainOpen(repoPath)
				if err != nil {
					logrus.Warnf("could not open %s: %s", repoPath, err)
					continue
				}
				deletedBranches := r.detectDeletedBranches(r)
				for _, deletedBranch := range deletedBranches {

					branchDeletedEventStr, err := json.Marshal(BranchDeletedEvent{
						Branch: deletedBranch,
						Manifests: extractManifestsFromBranch(repo, deletedBranch),
					})
					if err != nil {
						logrus.Warnf("could not serialize branch deleted event: %s", err)
						continue
					}

					// store branch deleted event
					event, err := r.dao.CreateEvent(&model.Event{
						Type:         model.TypeBranchDeleted,
						Blob:         string(branchDeletedEventStr),
						Repository:   r,
						GitopsHashes: []string{},
					})
					if err != nil {
						logrus.Warnf("could not store branch deleted event: %s", err)
						continue
					}
				}
			} else if os.IsNotExist(err) {
				err := r.clone(r)
				if err != nil {
					logrus.Warnf("could not clone: %s", err)
				}
			} else {
				logrus.Warn(err)
			}
		}

		time.Sleep(30 * time.Second)
	}
}

func (r *BranchDeleteEventWorker) detectDeletedBranches(repo *git.Repository) []string {
	token, user, err := r.tokenManager.Token()
	if err != nil {
		logrus.Errorf("couldn't get scm token: %s", err)
	}

	err = repo.Fetch(&git.FetchOptions{
		RefSpecs: fetchRefSpec,
		Auth: &http.BasicAuth{
			Username: user,
			Password: token,
		},
		Depth: 100,
		Tags:  git.NoTags,
	})
	if err == git.NoErrAlreadyUpToDate {
		return []string{}
	}
	if err != nil {
		logrus.Errorf("could not fetch: %s", err)
	}

	return []string{}
}

var mutex = &sync.Mutex{}

func (r *BranchDeleteEventWorker) clone(repoName string) error {
	repoPath := filepath.Join(r.cachePath, strings.ReplaceAll(repoName, "/", "%"))

	err := os.MkdirAll(repoPath, Dir_RWX_RX_R)
	if err != nil {
		return errors.WithMessage(err, "couldn't create folder")
	}

	token, user, err := r.tokenManager.Token()
	if err != nil {
		return errors.WithMessage(err, "couldn't get scm token")
	}

	opts := &git.CloneOptions{
		URL: fmt.Sprintf("%s/%s", "https://github.com", repoName),
		Auth: &http.BasicAuth{
			Username: user,
			Password: token,
		},
		Depth: 100,
		Tags:  git.NoTags,
	}

	repo, err := git.PlainClone(repoPath, false, opts)
	if err != nil {
		return errors.WithMessage(err, "couldn't clone")
	}

	err = repo.Fetch(&git.FetchOptions{
		RefSpecs: fetchRefSpec,
		Auth: &http.BasicAuth{
			Username: user,
			Password: token,
		},
		Depth: 100,
		Tags:  git.NoTags,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return errors.WithMessage(err, "couldn't fetch")
	}

	return nil
}

// BranchDeletedEvent contains all metadata about the deleted branch
type BranchDeletedEvent struct {
	Manifests []*dx.Manifest
	Branch    string
}
