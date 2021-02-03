package worker

import (
	"github.com/gimlet-io/gimletd/cmd/config"
	"github.com/gimlet-io/gimletd/model"
	"github.com/gimlet-io/gimletd/store"
	"github.com/sirupsen/logrus"
	"time"
)

type GitopsWorker struct {
	store  *store.Store
	config *config.Config
}

func NewGitopsWorker(
	store *store.Store,
	config *config.Config,
) *GitopsWorker {
	return &GitopsWorker{
		store:  store,
		config: config,
	}
}

func (w *GitopsWorker) Run() {
	for {
		artifacts, err := w.store.UnprocessedArtifacts()
		if err != nil {
			logrus.Errorf("Could not fetch unprocessed artifacts %s", err.Error())
			time.Sleep(1 * time.Second)
			continue
		}

		w.process(artifacts, w.store)
		time.Sleep(100 * time.Millisecond)
	}
}

func (w *GitopsWorker) process(artifacts []*model.Artifact, dao *store.Store) {
	for _, artifact := range artifacts {
		logrus.Info(artifact.Created)
	}
}
