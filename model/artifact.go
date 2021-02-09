package model

import (
	"encoding/json"
	"github.com/gimlet-io/gimletd/dx"
)

const StatusNew = "new"
const StatusProcessed = "processed"
const StatusError = "error"

type Artifact struct {
	ID           string      `json:"id,omitempty"  meddler:"id"`
	Repository   string      `json:"repository,omitempty"  meddler:"repository"`
	Branch       string      `json:"branch,omitempty"  meddler:"branch"`
	Event        dx.GitEvent `json:"event,omitempty"  meddler:"event"`
	SourceBranch string      `json:"sourceBranch,omitempty"  meddler:"source_branch"`
	TargetBranch string      `json:"targetBranch,omitempty"  meddler:"target_branch"`
	Tag          string      `json:"tag,omitempty"  meddler:"tag"`
	Created      int64       `json:"created,omitempty"  meddler:"created"`
	Blob         string      `json:"blob,omitempty"  meddler:"blob"`
	Status       string      `json:"status"  meddler:"status"`
	StatusDesc   string      `json:"statusDesc"  meddler:"status_desc"`
	SHA          string      `json:"sha"  meddler:"sha"`
}

func ToArtifactModel(artifact dx.Artifact) (*Artifact, error) {
	artifactStr, err := json.Marshal(artifact)
	if err != nil {
		return nil, err
	}

	return &Artifact{
		ID:           artifact.ID,
		Repository:   artifact.Version.RepositoryName,
		Branch:       artifact.Version.Branch,
		Event:        artifact.Version.Event,
		TargetBranch: artifact.Version.TargetBranch,
		SourceBranch: artifact.Version.SourceBranch,
		Tag:          artifact.Version.Tag,
		Blob:         string(artifactStr),
		SHA:          artifact.Version.SHA,
	}, nil
}

func ToArtifact(a *Artifact) (*dx.Artifact, error) {
	var artifact dx.Artifact
	json.Unmarshal([]byte(a.Blob), &artifact)
	return &artifact, nil
}
