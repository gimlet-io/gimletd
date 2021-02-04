package worker

import (
	"fmt"
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

		for _, artifact := range artifacts {
			err = process(artifact)

			if err != nil {
				logrus.Errorf("error in processing artifact: %s", err.Error())
				artifact.Status = model.StatusError
				artifact.StatusDesc = err.Error()
			} else {
				artifact.Status = model.StatusProcessed
			}

			err = w.store.UpdateArtifactStatus(
				artifact.ID,
				artifact.Status,
				artifact.StatusDesc,
			)
			if err != nil {
				logrus.Warnf("could not update event status %v", err)
			}
		}

		time.Sleep(100 * time.Millisecond)
	}
}

func process(artifactModel *model.Artifact) error {
	artifact, err := model.ToArtifact(artifactModel)
	if err != nil {
		return fmt.Errorf("cannot parse artifact %s", err.Error())
	}

	for _, env := range artifact.Environments {
		logrus.Info(env.Deploy)
	}
	return nil
}
