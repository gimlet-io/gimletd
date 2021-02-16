package worker

import (
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
	store                   *store.Store
	gitopsRepoUrl           string
	gitopsRepoDeployKeyPath string
	notificationsManager    notifications.Manager
}

func NewGitopsWorker(
	store *store.Store,
	gitopsRepoUrl string,
	gitopsRepoDeployKeyPath string,
	notificationsManager notifications.Manager,
) *GitopsWorker {
	return &GitopsWorker{
		store:                   store,
		gitopsRepoUrl:           gitopsRepoUrl,
		gitopsRepoDeployKeyPath: gitopsRepoDeployKeyPath,
		notificationsManager:    notificationsManager,
	}
}

func (w *GitopsWorker) Run() {
	for {
		artifactModels, err := w.store.UnprocessedArtifacts()
		if err != nil {
			logrus.Errorf("Could not fetch unprocessed artifactModels %s", err.Error())
			time.Sleep(1 * time.Second)
			continue
		}

		for _, artifactModel := range artifactModels {
			process(w.store,
				w.gitopsRepoUrl,
				w.gitopsRepoDeployKeyPath,
				artifactModel,
				w.notificationsManager,
			)
		}

		time.Sleep(100 * time.Millisecond)
	}
}

func process(
	store *store.Store,
	gitopsRepoUrl string,
	gitopsRepoDeployKeyPath string,
	artifactModel *model.Artifact,
	notificationsManager notifications.Manager,
) {
	artifact, err := model.ToArtifact(artifactModel)
	if err != nil {
		administerError(fmt.Errorf("cannot parse artifact %s", err.Error()), artifactModel, store)
		return
	}

	for _, env := range artifact.Environments {
		if !deployTrigger(artifact, env.Deploy) {
			continue
		}

		repo, err := githelper.CloneToMemory(gitopsRepoUrl, gitopsRepoDeployKeyPath)
		if err != nil {
			administerError(err, artifactModel, store)
			return
		}

		event := &notifications.GitopsEvent{
			Manifest:    env,
			Artifact:    artifact,
			TriggeredBy: "policy",
			Status:      notifications.Success,
			GitopsRepo:  gitopsRepoUrl,
		}

		sha, err := gitopsTemplateAndWrite(repo, artifact.Context, env, gitopsRepoDeployKeyPath)
		if err != nil {
			event.Status = notifications.Failure
			event.StatusDesc = err.Error()
			administerError(err, artifactModel, store)

			event.Status = notifications.Failure
			event.StatusDesc = err.Error()
			notificationsManager.Broadcast(event)
			return
		}

		err = githelper.Push(repo, gitopsRepoDeployKeyPath)
		if err != nil {
			administerError(err, artifactModel, store)
			return
		}

		if sha != "" { // if there was no changes to push
			event.GitopsRef = sha
			notificationsManager.Broadcast(event)
		}
	}

	administerSuccess(store, artifactModel)
}

func administerSuccess(store *store.Store, artifactModel *model.Artifact) {
	artifactModel.Status = model.StatusProcessed
	updateArtifactModel(store, artifactModel)
}

func updateArtifactModel(store *store.Store, artifactModel *model.Artifact) {
	err := store.UpdateArtifactStatus(
		artifactModel.ID,
		artifactModel.Status,
		artifactModel.StatusDesc,
	)
	if err != nil {
		logrus.Warnf("could not update artifactModel status %v", err)
	}
}

func administerError(err error, artifactModel *model.Artifact, store *store.Store) {
	logrus.Errorf("error in processing artifactModel: %s", err.Error())
	artifactModel.Status = model.StatusError
	artifactModel.StatusDesc = err.Error()

	updateArtifactModel(store, artifactModel)
}

func gitopsTemplateAndWrite(repo *git.Repository, context map[string]string, env *dx.Manifest, sshPrivateKeyPathForChartClone string) (string, error) {
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
