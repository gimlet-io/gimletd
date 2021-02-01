package server

import (
	"encoding/json"
	"github.com/gimlet-io/gimletd/artifact"
	"github.com/gimlet-io/gimletd/model"
	"github.com/gimlet-io/gimletd/store"
	"github.com/sirupsen/logrus"
	"net/http"
	"strconv"
	"time"
)

func saveArtifact(w http.ResponseWriter, r *http.Request) {
	var artifact artifact.Artifact
	json.NewDecoder(r.Body).Decode(&artifact)

	ctx := r.Context()
	store := ctx.Value("store").(*store.Store)

	artifactModel, err := model.ToArtifactModel(artifact)
	if err != nil {
		logrus.Errorf("cannot convert to artifact model: %s", err)
		http.Error(w, http.StatusText(500), 500)
		return
	}

	savedArtifactModel, err := store.CreateArtifact(artifactModel)
	if err != nil {
		logrus.Errorf("cannot save artifact: %s", err)
		http.Error(w, http.StatusText(500), 500)
		return
	}

	savedArtifact, err := model.ToArtifact(savedArtifactModel)
	artifactStr, err := json.Marshal(savedArtifact)
	if err != nil {
		logrus.Errorf("cannot serialize artifact: %s", err)
		http.Error(w, http.StatusText(500), 500)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(artifactStr))
}

func getArtifacts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	store := ctx.Value("store").(*store.Store)

	var limit, offset int
	var since, until *time.Time

	var app, branch string
	var pr bool
	var sourceBranch string
	var sha string

	params := r.URL.Query()
	if val, ok := params["limit"]; ok {
		l, err := strconv.Atoi(val[0])
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest)+" - "+err.Error(), http.StatusBadRequest)
			return
		}
		limit = l
	}
	if val, ok := params["offset"]; ok {
		o, err := strconv.Atoi(val[0])
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest)+" - "+err.Error(), http.StatusBadRequest)
			return
		}
		offset = o
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
	if val, ok := params["branch"]; ok {
		branch = val[0]
	}
	if val, ok := params["sourceBranch"]; ok {
		sourceBranch = val[0]
	}
	if val, ok := params["sha"]; ok {
		sha = val[0]
	}
	if val, ok := params["pr"]; ok {
		pr = val[0] == "true"
	}

	artifactModels, err := store.Artifacts(
		app, branch,
		pr,
		sourceBranch,
		sha,
		limit, offset, since, until)
	if err != nil {
		logrus.Errorf("cannot get artifacts: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	artifacts := []*artifact.Artifact{}
	for _, a := range artifactModels {
		artifact, err := model.ToArtifact(a)
		if err != nil {
			logrus.Errorf("cannot deserialize artifact: %s", err)
			http.Error(w, http.StatusText(500), 500)
			return
		}
		artifacts = append(artifacts, artifact)
	}

	artifactsStr, err := json.Marshal(artifacts)
	if err != nil {
		logrus.Errorf("cannot serialize artifacts: %s", err)
		http.Error(w, http.StatusText(500), 500)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(artifactsStr)
}
