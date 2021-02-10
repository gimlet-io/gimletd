package worker

import (
	"fmt"
	"github.com/gimlet-io/gimletd/dx"
	"github.com/gimlet-io/gimletd/githelper"
	"github.com/gimlet-io/gimletd/model"
	"github.com/gimlet-io/gimletd/store"
	"github.com/go-git/go-git/v5"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/yaml"
	"time"
)

type GitopsWorker struct {
	store                   *store.Store
	gitopsRepoUrl           string
	gitopsRepoDeployKeyPath string
}

func NewGitopsWorker(
	store *store.Store,
	gitopsRepoUrl string,
	gitopsRepoDeployKeyPath string,
) *GitopsWorker {
	return &GitopsWorker{
		store:                   store,
		gitopsRepoUrl:           gitopsRepoUrl,
		gitopsRepoDeployKeyPath: gitopsRepoDeployKeyPath,
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
			repo, err := githelper.CloneToMemory(w.gitopsRepoUrl, w.gitopsRepoDeployKeyPath)

			if err == nil {
				err = process(repo, artifact)
				if err == nil {
					err = githelper.Push(repo, w.gitopsRepoDeployKeyPath)
				}
			}

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

func process(repo *git.Repository, artifactModel *model.Artifact) error {
	artifact, err := model.ToArtifact(artifactModel)
	if err != nil {
		return fmt.Errorf("cannot parse artifact %s", err.Error())
	}

	for _, env := range artifact.Environments {
		if deployTrigger(artifact, env.Deploy) {
			manifestString, err := yaml.Marshal(env)
			if err != nil {
				return fmt.Errorf("cannot serialize manifest %s", err.Error())
			}

			templatedManifests, err := dx.HelmTemplate(string(manifestString), artifact.Context)
			files := dx.SplitHelmOutput(map[string]string{"manifest.yaml": templatedManifests})
			githelper.CommitFilesToGit(repo, files, env.Env, env.App, "automated deploy")
		}
	}
	return nil
}

func deployTrigger(artifactToCheck *dx.Artifact, deployPolicy *dx.Deploy) bool {
	if deployPolicy == nil {
		return false
	}

	if deployPolicy.Branch == "" &&
		deployPolicy.Event == nil {
		return false
	}

	if deployPolicy.Branch != "" &&
		deployPolicy.Branch != artifactToCheck.Version.Branch {
		return false
	}

	if deployPolicy.Event != nil {
		if *deployPolicy.Event != artifactToCheck.Version.Event {
			return false
		}
	}

	return true
}
