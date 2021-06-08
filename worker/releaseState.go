package worker

import (
	"github.com/gimlet-io/gimletd/githelper"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"time"
)

type ReleaseStateWorker struct {
	GitopsRepo              string
	GitopsRepoDeployKeyPath string
	Releases                *prometheus.GaugeVec
}

func (w *ReleaseStateWorker) Run() {
	for {
		repoTmpPath, repo, err := githelper.CloneToTmpFs(w.GitopsRepo, w.GitopsRepoDeployKeyPath)
		if err != nil {
			logrus.Errorf("cannot clone gitops repo: %s", err)
			time.Sleep(30 * time.Second)
			continue
		}
		defer githelper.TmpFsCleanup(repoTmpPath)

		envs, err := githelper.Envs(repo)
		if err != nil {
			logrus.Errorf("cannot get envs: %s", err)
			time.Sleep(30 * time.Second)
			continue
		}

		w.Releases.Reset()
		for _, env := range envs {
			appReleases, err := githelper.Status(repo, "", env)
			if err != nil {
				logrus.Errorf("cannot get status: %s", err)
				time.Sleep(30 * time.Second)
				continue
			}

			for app, release := range appReleases {
				created := time.Unix(release.Created, 0)
				if release != nil {
					w.Releases.WithLabelValues(
						env,
						app,
						release.Version.URL,
						release.Version.Message,
						release.GitopsRef,
						created.Format(time.RFC3339),
					).Set(1.0)
				} else {
					w.Releases.WithLabelValues(
						env,
						app,
						release.Version.URL,
						release.Version.Message,
						release.GitopsRef,
						created.Format(time.RFC3339),
					).Set(1.0)
				}
			}
		}

		time.Sleep(30 * time.Second)
	}
}
