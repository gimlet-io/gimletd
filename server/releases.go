package server

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gimlet-io/gimletd/dx"
	"github.com/gimlet-io/gimletd/githelper"
	"github.com/gimlet-io/gimletd/model"
	"github.com/gimlet-io/gimletd/store"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

func getReleases(w http.ResponseWriter, r *http.Request) {
	var since, until *time.Time
	var app, env, gitRepo string
	limit := 10

	params := r.URL.Query()
	if val, ok := params["limit"]; ok {
		l, err := strconv.Atoi(val[0])
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest)+" - "+err.Error(), http.StatusBadRequest)
			return
		}
		limit = l
	}

	if val, ok := params["since"]; ok {
		t, err := time.Parse(time.RFC3339, val[0])
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest)+" - "+err.Error(), http.StatusBadRequest)
			return
		}
		since = &t
	}
	if val, ok := params["until"]; ok {
		t, err := time.Parse(time.RFC3339, val[0])
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest)+" - "+err.Error(), http.StatusBadRequest)
			return
		}
		until = &t
	}

	if val, ok := params["app"]; ok {
		app = val[0]
	}
	if val, ok := params["env"]; ok {
		env = val[0]
	} else {
		http.Error(w, fmt.Sprintf("%s: %s", http.StatusText(http.StatusBadRequest), "env parameter is mandatory"), http.StatusBadRequest)
		return
	}
	if val, ok := params["git-repo"]; ok {
		gitRepo = val[0]
	}

	ctx := r.Context()
	repoCache := ctx.Value("repoCache").(*githelper.RepoCache)
	gitopsRepo := ctx.Value("gitopsRepo").(string)

	releases, err := githelper.Releases(repoCache.InstanceForRead(), app, env, since, until, limit, gitRepo)
	if err != nil {
		logrus.Errorf("cannot get releases: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	for _, r := range releases {
		r.GitopsRepo = gitopsRepo
	}

	releasesStr, err := json.Marshal(releases)
	if err != nil {
		logrus.Errorf("cannot serialize artifacts: %s", err)
		http.Error(w, http.StatusText(500), 500)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(releasesStr)
}

func getStatus(w http.ResponseWriter, r *http.Request) {
	var app, env string

	params := r.URL.Query()
	if val, ok := params["app"]; ok {
		app = val[0]
	}
	if val, ok := params["env"]; ok {
		env = val[0]
	} else {
		http.Error(w, fmt.Sprintf("%s: %s", http.StatusText(http.StatusBadRequest), "env parameter is mandatory"), http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	repoCache := ctx.Value("repoCache").(*githelper.RepoCache)
	gitopsRepo := ctx.Value("gitopsRepo").(string)
	perf := ctx.Value("perf").(*prometheus.HistogramVec)

	appReleases, err := githelper.Status(repoCache.InstanceForRead(), app, env, perf)
	if err != nil {
		logrus.Errorf("cannot get status: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	for _, release := range appReleases {
		if release != nil {
			release.GitopsRepo = gitopsRepo
			//release.Created = TODO Get githelper.Releases for each app with limit 1 - could be terribly slow
		}
	}

	appReleasesString, err := json.Marshal(appReleases)
	if err != nil {
		logrus.Errorf("cannot serialize app releases: %s", err)
		http.Error(w, http.StatusText(500), 500)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(appReleasesString)
}

func release(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	store := ctx.Value("store").(*store.Store)
	user := ctx.Value("user").(*model.User)

	body, _ := ioutil.ReadAll(r.Body)
	var releaseRequest dx.ReleaseRequest
	err := json.NewDecoder(bytes.NewReader(body)).Decode(&releaseRequest)
	if err != nil {
		logrus.Errorf("cannot decode release request: %s", err)
		http.Error(w, http.StatusText(400), 400)
		return
	}

	if releaseRequest.Env == "" {
		http.Error(w, fmt.Sprintf("%s: %s", http.StatusText(http.StatusBadRequest), "env parameter is mandatory"), http.StatusBadRequest)
		return
	}

	if releaseRequest.ArtifactID == "" {
		http.Error(w, fmt.Sprintf("%s: %s", http.StatusText(http.StatusBadRequest), "artifact parameter is mandatory"), http.StatusBadRequest)
		return
	}

	releaseRequestStr, err := json.Marshal(dx.ReleaseRequest{
		Env:         releaseRequest.Env,
		App:         releaseRequest.App,
		ArtifactID:  releaseRequest.ArtifactID,
		TriggeredBy: user.Login,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("%s - cannot serialize release request: %s", http.StatusText(http.StatusInternalServerError), err), http.StatusInternalServerError)
		return
	}

	artifact, err := store.Artifact(releaseRequest.ArtifactID)
	if err != nil {
		http.Error(w, fmt.Sprintf("%s - cannot find artifact with id %s", http.StatusText(http.StatusNotFound), releaseRequest.ArtifactID), http.StatusNotFound)
		return
	}
	event, err := store.CreateEvent(&model.Event{
		Type:       model.TypeRelease,
		Blob:       string(releaseRequestStr),
		Repository: artifact.Repository,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("%s - cannot save release request: %s", http.StatusText(http.StatusInternalServerError), err), http.StatusInternalServerError)
		return
	}

	eventIDBytes, _ := json.Marshal(map[string]string{
		"id": event.ID,
	})

	w.WriteHeader(http.StatusCreated)
	w.Write(eventIDBytes)
}

func rollback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	store := ctx.Value("store").(*store.Store)
	user := ctx.Value("user").(*model.User)

	params := r.URL.Query()
	var env, app, targetSHA string
	if val, ok := params["env"]; ok {
		env = val[0]
	} else {
		http.Error(w, fmt.Sprintf("%s: %s", http.StatusText(http.StatusBadRequest), "env parameter is mandatory"), http.StatusBadRequest)
		return
	}
	if val, ok := params["app"]; ok {
		app = val[0]
	} else {
		http.Error(w, fmt.Sprintf("%s: %s", http.StatusText(http.StatusBadRequest), "app parameter is mandatory"), http.StatusBadRequest)
		return
	}
	if val, ok := params["sha"]; ok {
		targetSHA = val[0]
	} else {
		http.Error(w, fmt.Sprintf("%s: %s", http.StatusText(http.StatusBadRequest), "sha parameter is mandatory"), http.StatusBadRequest)
		return
	}

	rollbackRequestStr, err := json.Marshal(dx.RollbackRequest{
		Env:         env,
		App:         app,
		TargetSHA:   targetSHA,
		TriggeredBy: user.Login,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("%s - cannot serialize rollback request: %s", http.StatusText(http.StatusInternalServerError), err), http.StatusInternalServerError)
		return
	}

	event, err := store.CreateEvent(&model.Event{
		Type: model.TypeRollback,
		Blob: string(rollbackRequestStr),
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("%s - cannot save rollback request: %s", http.StatusText(http.StatusInternalServerError), err), http.StatusInternalServerError)
		return
	}

	eventIDBytes, _ := json.Marshal(map[string]string{
		"id": event.ID,
	})

	w.WriteHeader(http.StatusCreated)
	w.Write(eventIDBytes)
}

func getEvent(w http.ResponseWriter, r *http.Request) {
	var id string

	params := r.URL.Query()

	if val, ok := params["id"]; ok {
		id = val[0]
	} else {
		http.Error(w, fmt.Sprintf("%s: %s", http.StatusText(http.StatusBadRequest), "id parameter is mandatory"), http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	store := ctx.Value("store").(*store.Store)
	event, err := store.Event(id)
	if err == sql.ErrNoRows {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	} else if err != nil {
		logrus.Errorf("cannot get event: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

	statusBytes, _ := json.Marshal(map[string]string{
		"status": event.Status,
		"desc":   event.StatusDesc,
	})

	w.WriteHeader(http.StatusOK)
	w.Write(statusBytes)
}
