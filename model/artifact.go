package model

import (
	"encoding/json"
	"github.com/gimlet-io/gimletd/artifact"
)

const StatusNew = "new"
const StatusProcessed = "processed"
const StatusError = "error"

type Artifact struct {
	ID           string `json:"id,omitempty"  meddler:"id"`
	Repository   string `json:"repository,omitempty"  meddler:"repository"`
	Branch       string `json:"branch,omitempty"  meddler:"branch"`
	PR           bool   `json:"pr,omitempty"  meddler:"pr"`
	SourceBranch string `json:"sourceBranch,omitempty"  meddler:"source_branch"`
	Created      int64  `json:"created,omitempty"  meddler:"created"`
	Blob         string `json:"blob,omitempty"  meddler:"blob"`
	Status       string `json:"status"  meddler:"status"`
	StatusDesc   string `json:"statusDesc"  meddler:"status_desc"`
	SHA          string `json:"sha"  meddler:"sha"`
}

func ToArtifactModel(artifact artifact.Artifact) (*Artifact, error) {
	artifactStr, err := json.Marshal(artifact)
	if err != nil {
		return nil, err
	}

	return &Artifact{
		ID:           artifact.ID,
		Repository:   artifact.Version.RepositoryName,
		Branch:       artifact.Version.Branch,
		PR:           artifact.Version.PR,
		SourceBranch: artifact.Version.SourceBranch,
		Blob:         string(artifactStr),
		SHA:          artifact.Version.SHA,
	}, nil
}

func ToArtifact(a *Artifact) (*artifact.Artifact, error) {
	var artifact artifact.Artifact
	json.Unmarshal([]byte(a.Blob), &artifact)
	return &artifact, nil
}
