package worker

import (
	"fmt"
	"github.com/gimlet-io/gimletd/githelper"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"path/filepath"
	"time"
)

type ReleaseStateWorker struct {
	GitopsRepo string
	RepoCache  *githelper.RepoCache
	Releases   *prometheus.GaugeVec
	Perf       *prometheus.HistogramVec
}

func (w *ReleaseStateWorker) Run() {
	for {
		t0 := time.Now()
		repo := w.RepoCache.InstanceForRead()
		logrus.Infof("releaseState_clone: %f", time.Since(t0).Seconds())
		w.Perf.WithLabelValues("releaseState_clone").Observe(time.Since(t0).Seconds())

		envs, err := githelper.Envs(repo)
		if err != nil {
			logrus.Errorf("cannot get envs: %s", err)
			time.Sleep(30 * time.Second)
			continue
		}

		w.Releases.Reset()
		for _, env := range envs {
			t1 := time.Now()
			appReleases, err := githelper.Status(repo, "", env)
			if err != nil {
				logrus.Errorf("cannot get status: %s", err)
				time.Sleep(30 * time.Second)
				continue
			}
			logrus.Infof("releaseState_appReleases: %f", time.Since(t1).Seconds())
			w.Perf.WithLabelValues("releaseState_appReleases").Observe(time.Since(t1).Seconds())

			for app, release := range appReleases {
				t2 := time.Now()
				commit, err := lastCommitThatTouchedAFile(repo, filepath.Join(env, app))
				if err != nil {
					logrus.Errorf("cannot find last commit: %s", err)
					time.Sleep(30 * time.Second)
					continue
				}
				logrus.Infof("releaseState_appRelease: %f", time.Since(t2).Seconds())
				w.Perf.WithLabelValues("releaseState_appRelease").Observe(time.Since(t2).Seconds())

				gitopsRef := fmt.Sprintf("https://github.com/%s/commit/%s", w.GitopsRepo, commit.Hash.String())
				created := commit.Committer.When

				if release != nil {
					w.Releases.WithLabelValues(
						env,
						app,
						release.Version.URL,
						release.Version.Message,
						gitopsRef,
						created.Format(time.RFC3339),
					).Set(1.0)
				} else {
					w.Releases.WithLabelValues(
						env,
						app,
						"",
						"",
						gitopsRef,
						created.Format(time.RFC3339),
					).Set(1.0)
				}
			}
		}
		logrus.Infof("releaseState_run: %f", time.Since(t0).Seconds())
		w.Perf.WithLabelValues("releaseState_run").Observe(time.Since(t0).Seconds())
		time.Sleep(30 * time.Second)
	}
}

func lastCommitThatTouchedAFile(repo *git.Repository, path string) (*object.Commit, error) {
	commits, err := repo.Log(
		&git.LogOptions{
			Path: &path,
		},
	)
	if err != nil {
		return nil, err
	}

	var commit *object.Commit
	err = commits.ForEach(func(c *object.Commit) error {
		commit = c
		return fmt.Errorf("%s", "FOUND")
	})
	if err != nil &&
		err.Error() != "EOF" &&
		err.Error() != "FOUND" {
		return nil, err
	}

	return commit, nil
}
