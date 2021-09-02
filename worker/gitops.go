package worker

import (
	"encoding/json"
	"fmt"
	"github.com/gimlet-io/gimletd/dx"
	"github.com/gimlet-io/gimletd/dx/helm"
	"github.com/gimlet-io/gimletd/githelper"
	"github.com/gimlet-io/gimletd/model"
	"github.com/gimlet-io/gimletd/notifications"
	"github.com/gimlet-io/gimletd/store"
	"github.com/gimlet-io/gimletd/worker/events"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/gobwas/glob"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
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
	eventsProcessed                prometheus.Counter
	repoCache                      *githelper.RepoCache
}

func NewGitopsWorker(
	store *store.Store,
	gitopsRepo string,
	gitopsRepoDeployKeyPath string,
	githubChartAccessDeployKeyPath string,
	notificationsManager notifications.Manager,
	eventsProcessed prometheus.Counter,
	repoCache *githelper.RepoCache,
) *GitopsWorker {
	return &GitopsWorker{
		store:                          store,
		gitopsRepo:                     gitopsRepo,
		gitopsRepoDeployKeyPath:        gitopsRepoDeployKeyPath,
		notificationsManager:           notificationsManager,
		githubChartAccessDeployKeyPath: githubChartAccessDeployKeyPath,
		eventsProcessed:                eventsProcessed,
		repoCache:                      repoCache,
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
			w.eventsProcessed.Inc()
			processEvent(w.store,
				w.gitopsRepo,
				w.gitopsRepoDeployKeyPath,
				w.githubChartAccessDeployKeyPath,
				event,
				w.notificationsManager,
				w.repoCache,
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
	repoCache *githelper.RepoCache,
) {
	// process event based on type
	var err error
	var gitopsEvents []*events.DeployEvent
	var rollbackEvent *events.RollbackEvent
	switch event.Type {
	case model.TypeArtifact:
		gitopsEvents, err = processArtifactEvent(gitopsRepo, gitopsRepoDeployKeyPath, githubChartAccessDeployKeyPath, event, repoCache)
		if len(gitopsEvents) > 0 {
			repoCache.Invalidate()
		}
	case model.TypeRelease:
		gitopsEvents, err = processReleaseEvent(store, gitopsRepo, gitopsRepoDeployKeyPath, githubChartAccessDeployKeyPath, event)
		repoCache.Invalidate()
	case model.TypeRollback:
		rollbackEvent, err = processRollbackEvent(gitopsRepo, gitopsRepoDeployKeyPath, event)
		notificationsManager.Broadcast(notifications.MessageFromRollbackEvent(rollbackEvent))
		for _, sha := range rollbackEvent.GitopsRefs {
			setGitopsHashOnEvent(event, sha)
		}
		repoCache.Invalidate()
	}

	// send out notifications based on gitops events
	for _, gitopsEvent := range gitopsEvents {
		notificationsManager.Broadcast(notifications.MessageFromGitOpsEvent(gitopsEvent))
	}

	// record gitops hashes on events
	for _, gitopsEvent := range gitopsEvents {
		setGitopsHashOnEvent(event, gitopsEvent.GitopsRef)
	}

	// store event state
	if err != nil {
		logrus.Errorf("error in processing event: %s", err.Error())
		event.Status = model.StatusError
		event.StatusDesc = err.Error()
		err := updateEvent(store, event)
		if err != nil {
			logrus.Warnf("could not update event status %v", err)
		}
	} else {
		event.Status = model.StatusProcessed
		err := updateEvent(store, event)
		if err != nil {
			logrus.Warnf("could not update event status %v", err)
		}
	}
}

func setGitopsHashOnEvent(event *model.Event, gitopsSha string) {
	if gitopsSha == "" {
		return
	}

	if event.GitopsHashes == nil {
		event.GitopsHashes = []string{}
	}

	event.GitopsHashes = append(event.GitopsHashes, gitopsSha)
}

func processReleaseEvent(
	store *store.Store,
	gitopsRepo string,
	gitopsRepoDeployKeyPath string,
	githubChartAccessDeployKeyPath string,
	event *model.Event,
) ([]*events.DeployEvent, error) {
	var gitopsEvents []*events.DeployEvent
	var releaseRequest dx.ReleaseRequest
	err := json.Unmarshal([]byte(event.Blob), &releaseRequest)
	if err != nil {
		return gitopsEvents, fmt.Errorf("cannot parse release request with id: %s", event.ID)
	}

	artifactEvent, err := store.Artifact(releaseRequest.ArtifactID)
	if err != nil {
		return gitopsEvents, fmt.Errorf("cannot find artifact with id: %s", event.ArtifactID)
	}
	artifact, err := model.ToArtifact(artifactEvent)
	if err != nil {
		return gitopsEvents, fmt.Errorf("cannot parse artifact %s", err.Error())
	}

	for _, env := range artifact.Environments {
		if env.Env != releaseRequest.Env {
			continue
		}
		if releaseRequest.App != "" &&
			env.App != releaseRequest.App {
			continue
		}

		gitopsEvent, err := cloneTemplateWriteAndPush(
			gitopsRepo,
			gitopsRepoDeployKeyPath,
			githubChartAccessDeployKeyPath,
			artifact,
			env,
			releaseRequest.TriggeredBy,
		)
		gitopsEvents = append(gitopsEvents, gitopsEvent)
		if err != nil {
			return gitopsEvents, err
		}
	}

	return gitopsEvents, nil
}

func processRollbackEvent(
	gitopsRepo string,
	gitopsRepoDeployKeyPath string,
	event *model.Event,
) (*events.RollbackEvent, error) {
	var rollbackRequest dx.RollbackRequest
	err := json.Unmarshal([]byte(event.Blob), &rollbackRequest)
	if err != nil {
		return nil, fmt.Errorf("cannot parse release request with id: %s", event.ID)
	}

	rollbackEvent := &events.RollbackEvent{
		RollbackRequest: &rollbackRequest,
		GitopsRepo:      gitopsRepo,
	}

	repoTmpPath, repo, err := githelper.CloneToTmpFs(gitopsRepo, gitopsRepoDeployKeyPath)
	if err != nil {
		rollbackEvent.Status = events.Failure
		return rollbackEvent, err
	}
	defer githelper.TmpFsCleanup(repoTmpPath)

	headSha, _ := repo.Head()

	err = revertTo(
		rollbackRequest.Env,
		rollbackRequest.App,
		repo,
		repoTmpPath,
		rollbackRequest.TargetSHA,
	)
	if err != nil {
		rollbackEvent.Status = events.Failure
		return rollbackEvent, err
	}

	hashes, err := shasSince(repo, headSha.Hash().String())
	if err != nil {
		rollbackEvent.Status = events.Failure
		return rollbackEvent, err
	}

	err = githelper.Push(repo, gitopsRepoDeployKeyPath)
	if err != nil {
		rollbackEvent.Status = events.Failure
		return rollbackEvent, err
	}

	rollbackEvent.GitopsRefs = hashes
	rollbackEvent.Status = events.Success
	return rollbackEvent, nil
}

func shasSince(repo *git.Repository, since string) ([]string, error) {
	var hashes []string
	commitWalker, err := repo.Log(&git.LogOptions{})
	if err != nil {
		return hashes, fmt.Errorf("cannot walk commits: %s", err)
	}

	err = commitWalker.ForEach(func(c *object.Commit) error {
		if c.Hash.String() == since {
			return fmt.Errorf("%s", "FOUND")
		}
		hashes = append(hashes, c.Hash.String())
		return nil
	})
	if err != nil &&
		err.Error() != "EOF" &&
		err.Error() != "FOUND" {
		return hashes, fmt.Errorf("cannot walk commits: %s", err)
	}

	return hashes, nil
}

func processArtifactEvent(
	gitopsRepo string,
	gitopsRepoDeployKeyPath string,
	githubChartAccessDeployKeyPath string,
	event *model.Event,
	repoCache *githelper.RepoCache,
) ([]*events.DeployEvent, error) {
	var gitopsEvents []*events.DeployEvent
	artifact, err := model.ToArtifact(event)
	if err != nil {
		return gitopsEvents, fmt.Errorf("cannot parse artifact %s", err.Error())
	}

	for _, env := range artifact.Environments {
		if !deployTrigger(artifact, env.Deploy) {
			continue
		}

		gitopsEvent, err := cloneTemplateWriteAndPush(
			gitopsRepo,
			gitopsRepoDeployKeyPath,
			githubChartAccessDeployKeyPath,
			artifact,
			env,
			"policy",
		)
		gitopsEvents = append(gitopsEvents, gitopsEvent)
		if err != nil {
			return gitopsEvents, err
		}

		repoCache.Invalidate()
	}

	return gitopsEvents, nil
}

func cloneTemplateWriteAndPush(
	gitopsRepo string,
	gitopsRepoDeployKeyPath string,
	githubChartAccessDeployKeyPath string,
	artifact *dx.Artifact,
	env *dx.Manifest,
	triggeredBy string,
) (*events.DeployEvent, error) {
	gitopsEvent := &events.DeployEvent{
		Manifest:    env,
		Artifact:    artifact,
		TriggeredBy: triggeredBy,
		Status:      events.Success,
		GitopsRepo:  gitopsRepo,
	}

	repoTmpPath, repo, err := githelper.CloneToTmpFs(gitopsRepo, gitopsRepoDeployKeyPath)
	defer githelper.TmpFsCleanup(repoTmpPath)
	if err != nil {
		gitopsEvent.Status = events.Failure
		gitopsEvent.StatusDesc = err.Error()
		return gitopsEvent, err
	}

	err = env.ResolveVars(artifact.Context)
	if err != nil {
		err = fmt.Errorf("cannot resolve manifest vars %s", err.Error())
		gitopsEvent.Status = events.Failure
		gitopsEvent.StatusDesc = err.Error()
		return gitopsEvent, err
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
		env,
		releaseMeta,
		githubChartAccessDeployKeyPath,
	)
	if err != nil {
		gitopsEvent.Status = events.Failure
		gitopsEvent.StatusDesc = err.Error()
		return gitopsEvent, err
	}

	err = githelper.Push(repo, gitopsRepoDeployKeyPath)
	if err != nil {
		gitopsEvent.Status = events.Failure
		gitopsEvent.StatusDesc = err.Error()
		return gitopsEvent, err
	}

	if sha != "" { // if there is a change to push
		gitopsEvent.GitopsRef = sha
	}

	return gitopsEvent, nil
}

func revertTo(env string, app string, repo *git.Repository, repoTmpPath string, sha string) error {
	path := fmt.Sprintf("%s/%s", env, app)
	commits, err := repo.Log(
		&git.LogOptions{
			Path: &path,
		},
	)
	if err != nil {
		return errors.WithMessage(err, "could not walk commits")
	}

	hashesToRevert := []string{}
	err = commits.ForEach(func(c *object.Commit) error {
		if c.Hash.String() == sha {
			return fmt.Errorf("EOF")
		}

		if !githelper.RollbackCommit(c) {
			hashesToRevert = append(hashesToRevert, c.Hash.String())
		}
		return nil
	})
	if err != nil && err.Error() != "EOF" {
		return err
	}

	for _, hash := range hashesToRevert {
		hasBeenReverted, err := githelper.HasBeenReverted(repo, hash, env, app)
		if !hasBeenReverted {
			logrus.Infof("reverting %s", hash)
			err = githelper.NativeRevert(repoTmpPath, hash)
			if err != nil {
				return errors.WithMessage(err, "could not revert")
			}
		}
	}
	return nil
}

func updateEvent(store *store.Store, event *model.Event) error {
	gitopsHashesString, err := json.Marshal(event.GitopsHashes)
	if err != nil {
		return err
	}
	return store.UpdateEventStatus(event.ID, event.Status, event.StatusDesc, string(gitopsHashesString))
}

func gitopsTemplateAndWrite(
	repo *git.Repository,
	env *dx.Manifest,
	release *dx.Release,
	sshPrivateKeyPathForChartClone string,
) (string, error) {
	if strings.HasPrefix(env.Chart.Name, "git@") {
		tmpChartDir, err := helm.CloneChartFromRepo(*env, sshPrivateKeyPathForChartClone)
		if err != nil {
			return "", fmt.Errorf("cannot fetch chart from git %s", err.Error())
		}
		env.Chart.Name = tmpChartDir
		defer os.RemoveAll(tmpChartDir)
	}

	templatedManifests, err := helm.HelmTemplate(*env)
	if err != nil {
		return "", fmt.Errorf("cannot run helm template %s", err.Error())
	}
	files := helm.SplitHelmOutput(map[string]string{"manifest.yaml": templatedManifests})

	releaseString, err := json.Marshal(release)
	if err != nil {
		return "", fmt.Errorf("cannot marshal release meta data %s", err.Error())
	}

	sha, err := githelper.CommitFilesToGit(repo, files, env.Env, env.App, "automated deploy", string(releaseString))
	if err != nil {
		return "", fmt.Errorf("cannot write to git: %s", err.Error())
	}

	return sha, nil
}

func deployTrigger(artifactToCheck *dx.Artifact, deployPolicy *dx.Deploy) bool {
	if deployPolicy == nil {
		return false
	}

	if deployPolicy.Branch == "" &&
		deployPolicy.Event == nil &&
		deployPolicy.Tag == "" {
		return false
	}

	if deployPolicy.Branch != "" &&
		(deployPolicy.Event == nil || *deployPolicy.Event != *dx.PushPtr() && *deployPolicy.Event != *dx.PRPtr()) {
		return false
	}

	if deployPolicy.Tag != "" &&
		(deployPolicy.Event == nil || *deployPolicy.Event != *dx.TagPtr()) {
		return false
	}

	if deployPolicy.Tag != "" {
		g := glob.MustCompile(deployPolicy.Tag)

		exactMatch := deployPolicy.Tag == artifactToCheck.Version.Tag
		patternMatch := g.Match(artifactToCheck.Version.Tag)

		if !exactMatch && !patternMatch {
			return false
		}
	}

	if deployPolicy.Branch != "" {
		g := glob.MustCompile(deployPolicy.Branch)

		exactMatch := deployPolicy.Branch == artifactToCheck.Version.Branch
		patternMatch := g.Match(artifactToCheck.Version.Branch)

		if !exactMatch && !patternMatch {
			return false
		}
	}

	if deployPolicy.Event != nil {
		if *deployPolicy.Event != artifactToCheck.Version.Event {
			return false
		}
	}

	return true
}
