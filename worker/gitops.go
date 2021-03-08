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
			process(w.store,
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

func process(
	store *store.Store,
	gitopsRepo string,
	gitopsRepoDeployKeyPath string,
	githubChartAccessDeployKeyPath string,
	event *model.Event,
	notificationsManager notifications.Manager,
) {
	artifact, err := model.ToArtifact(event)
	if err != nil {
		administerError(fmt.Errorf("cannot parse artifact %s", err.Error()), event, store)
		return
	}

	for _, env := range artifact.Environments {
		if !deployTrigger(artifact, env.Deploy) {
			continue
		}

		repo, err := githelper.CloneToMemory(gitopsRepo, gitopsRepoDeployKeyPath, true)
		if err != nil {
			administerError(err, event, store)
			return
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
			TriggeredBy: "policy",
		}

		sha, err := gitopsTemplateAndWrite(
			repo,
			artifact.Context,
			env,
			releaseMeta,
			githubChartAccessDeployKeyPath,
		)
		if err != nil {
			event.Status = model.StatusError
			event.StatusDesc = err.Error()
			administerError(err, event, store)

			gitopsEvent.Status = notifications.Failure
			gitopsEvent.StatusDesc = err.Error()
			notificationsManager.Broadcast(notifications.MessageFromGitOpsEvent(gitopsEvent))
			return
		}

		err = githelper.Push(repo, gitopsRepoDeployKeyPath)
		if err != nil {
			administerError(err, event, store)
			return
		}

		if sha != "" { // if there was changes to push
			gitopsEvent.GitopsRef = sha
			notificationsManager.Broadcast(notifications.MessageFromGitOpsEvent(gitopsEvent))
		}
	}

	administerSuccess(store, event)
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
