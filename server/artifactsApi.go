package server

import (
	"encoding/json"
	"github.com/gimlet-io/gimletd/artifact"
	"github.com/gimlet-io/gimletd/model"
	"github.com/gimlet-io/gimletd/store"
	"github.com/sirupsen/logrus"
	"net/http"
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

	artifactModels, err := store.Artifacts()
	if err != nil {
		logrus.Errorf("cannot get artifacts: %s", err)
		http.Error(w, http.StatusText(500), 500)
		return
	}

	var artifacts []*artifact.Artifact
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
