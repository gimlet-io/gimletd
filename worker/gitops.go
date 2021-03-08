package worker

import (
	"encoding/json"
	"fmt"
	"github.com/gimlet-io/gimletd/dx"
	"github.com/gimlet-io/gimletd/githelper"
	"github.com/gimlet-io/gimletd/model"
	"github.com/gimlet-io/gimletd/notifications"
	"github.com/gimlet-io/gimletd/store"
	"github.com/go-git/go-git/v5"
	"github.com/sirupsen/logrus"
	"os"
	"strings"
	"time"
)

type GitopsWorker struct {
	store                          *store.Store
	gitopsRepo                     string
	gitopsRepoDeployKeyPath        string
	githubChartAccessDeployKeyPath string
	notificationsManager           notifications.Manager
}

func NewGitopsWorker(
	store *store.Store,
	gitopsRepo string,
	gitopsRepoDeployKeyPath string,
	githubChartAccessDeployKeyPath string,
	notificationsManager notifications.Manager,
) *GitopsWorker {
	return &GitopsWorker{
		store:                          store,
		gitopsRepo:                     gitopsRepo,
		gitopsRepoDeployKeyPath:        gitopsRepoDeployKeyPath,
		notificationsManager:           notificationsManager,
		githubChartAccessDeployKeyPath: githubChartAccessDeployKeyPath,
	}
}

func (w *GitopsWorker) Run() {
	for {
		events, err := w.store.UnprocessedEvents()
		if err != nil {
			logrus.Errorf("Could not fetch unprocessed events %s", err.Error())
			time.Sleep(1 * time.Second)
			continue
		}

		for _, event := range events {
			processEvent(w.store,
				w.gitopsRepo,
				w.gitopsRepoDeployKeyPath,
				w.githubChartAccessDeployKeyPath,
				event,
				w.notificationsManager,
			)
		}

		time.Sleep(100 * time.Millisecond)
	}
}

func processEvent(
	store *store.Store,
	gitopsRepo string,
	gitopsRepoDeployKeyPath string,
	githubChartAccessDeployKeyPath string,
	event *model.Event,
	notificationsManager notifications.Manager,
) {
	var artifact *dx.Artifact
	var err error

	switch event.Type {
	case model.TypeArtifact:
		artifact, err = model.ToArtifact(event)
		if err != nil {
			administerError(fmt.Errorf("cannot parse artifact %s", err.Error()), event, store)
			return
		}

		for _, env := range artifact.Environments {
			if !deployTrigger(artifact, env.Deploy) {
				continue
			}

			err = cloneTemplateWriteAndPush(
				gitopsRepo,
				gitopsRepoDeployKeyPath,
				githubChartAccessDeployKeyPath,
				notificationsManager,
				artifact,
				env,
				"policy",
			)
			if err != nil {
				administerError(err, event, store)
				return
			}
		}
	case model.TypeRelease:
		var releaseRequest dx.ReleaseRequest
		json.Unmarshal([]byte(event.Blob), &releaseRequest)
		if err != nil {
			administerError(fmt.Errorf("cannot parse release request with id: %s", event.ID), event, store)
			return
		}

		artifactEvent, err := store.Artifact(releaseRequest.ArtifactID)
		if err != nil {
			administerError(fmt.Errorf("cannot find artifact with id: %s", event.ArtifactID), event, store)
			return
		}
		artifact, err = model.ToArtifact(artifactEvent)
		if err != nil {
			administerError(fmt.Errorf("cannot parse artifact %s", err.Error()), event, store)
			return
		}

		for _, env := range artifact.Environments {
			if env.Env != releaseRequest.Env {
				continue
			}

			err = cloneTemplateWriteAndPush(
				gitopsRepo,
				gitopsRepoDeployKeyPath,
				githubChartAccessDeployKeyPath,
				notificationsManager,
				artifact,
				env,
				releaseRequest.TriggeredBy,
			)
			if err != nil {
				administerError(err, event, store)
				return
			}
		}
	}

	administerSuccess(store, event)
}

func cloneTemplateWriteAndPush(
	gitopsRepo string,
	gitopsRepoDeployKeyPath string,
	githubChartAccessDeployKeyPath string,
	notificationsManager notifications.Manager,
	artifact *dx.Artifact,
	env *dx.Manifest,
	triggeredBy string,
) error {
	repo, err := githelper.CloneToMemory(gitopsRepo, gitopsRepoDeployKeyPath, true)
	if err != nil {
		return err
	}

	gitopsEvent := &notifications.GitopsEvent{
		Manifest:    env,
		Artifact:    artifact,
		TriggeredBy: "policy",
		Status:      notifications.Success,
		GitopsRepo:  gitopsRepo,
	}

	releaseMeta := &dx.Release{
		App:         env.App,
		Env:         env.Env,
		ArtifactID:  artifact.ID,
		Version:     &artifact.Version,
		TriggeredBy: triggeredBy,
	}

	sha, err := gitopsTemplateAndWrite(
		repo,
		artifact.Context,
		env,
		releaseMeta,
		githubChartAccessDeployKeyPath,
	)
	if err != nil {
		gitopsEvent.Status = notifications.Failure
		gitopsEvent.StatusDesc = err.Error()
		notificationsManager.Broadcast(notifications.MessageFromGitOpsEvent(gitopsEvent))
		return err
	}

	err = githelper.Push(repo, gitopsRepoDeployKeyPath)
	if err != nil {
		return err
	}

	if sha != "" { // if there was changes to push
		gitopsEvent.GitopsRef = sha
		notificationsManager.Broadcast(notifications.MessageFromGitOpsEvent(gitopsEvent))
	}

	return nil
}

func administerSuccess(store *store.Store, event *model.Event) {
	event.Status = model.StatusProcessed
	updateEvent(store, event)
}

func updateEvent(store *store.Store, event *model.Event) {
	err := store.UpdateEventStatus(
		event.ID,
		event.Status,
		event.StatusDesc,
	)
	if err != nil {
		logrus.Warnf("could not update event status %v", err)
	}
}

func administerError(err error, event *model.Event, store *store.Store) {
	logrus.Errorf("error in processing event: %s", err.Error())
	event.Status = model.StatusError
	event.StatusDesc = err.Error()

	updateEvent(store, event)
}

func gitopsTemplateAndWrite(
	repo *git.Repository,
	context map[string]string,
	env *dx.Manifest,
	release *dx.Release,
	sshPrivateKeyPathForChartClone string,
) (string, error) {
	err := env.ResolveVars(context)
	if err != nil {
		return "", fmt.Errorf("cannot resolve manifest vars %s", err.Error())
	}

	if strings.HasPrefix(env.Chart.Name, "git@") {
		tmpChartDir, err := dx.CloneChartFromRepo(*env, sshPrivateKeyPathForChartClone)
		if err != nil {
			return "", fmt.Errorf("cannot fetch chart from git %s", err.Error())
		}
		env.Chart.Name = tmpChartDir
		defer os.RemoveAll(tmpChartDir)
	}

	templatedManifests, err := dx.HelmTemplate(*env)
	if err != nil {
		return "", fmt.Errorf("cannot run helm template %s", err.Error())
	}
	files := dx.SplitHelmOutput(map[string]string{"manifest.yaml": templatedManifests})

	releaseString, err := json.Marshal(release)
	if err != nil {
		return "", fmt.Errorf("cannot marshal release meta data %s", err.Error())
	}
	files["release.json"] = string(releaseString)

	sha, err := githelper.CommitFilesToGit(repo, files, env.Env, env.App, "automated deploy")
	if err != nil {
		return "", fmt.Errorf("cannot write to git %s", err.Error())
	}

	return sha, nil
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
