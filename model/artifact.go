package model

import (
	"encoding/json"
	"github.com/gimlet-io/gimletd/artifact"
)

type Artifact struct {
	ID           string `json:"id,omitempty"  meddler:"id,pk"`
	Repository   string `json:"repository,omitempty"  meddler:"repository"`
	Branch       string `json:"branch,omitempty"  meddler:"branch"`
	PR           bool   `json:"pr,omitempty"  meddler:"pr"`
	SourceBranch string `json:"sourceBranch,omitempty"  meddler:"source_branch"`
	Created      int64  `json:"created,omitempty"  meddler:"created"`
	Blob         string `json:"blob,omitempty"  meddler:"blob"`
}

func ArtifactModel(artifact artifact.Artifact) (*Artifact, error) {
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
	}, nil
}
