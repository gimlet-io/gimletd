package store

import (
	"encoding/json"
	"github.com/gimlet-io/gimletd/artifact"
	"github.com/gimlet-io/gimletd/model"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArtifactCRUD(t *testing.T) {
	s := NewTest()
	defer func() {
		s.Close()
	}()

	artifactStr := `
{
  "id": "my-app-b2ab0f7a-ca0e-45cf-83a0-cadd94dddeac",
  "version": {
    "repositoryName": "my-app",
    "sha": "ea9ab7cc31b2599bf4afcfd639da516ca27a4780",
    "branch": "master",
    "authorName": "Jane Doe",
    "authorEmail": "jane@doe.org",
    "committerName": "Jane Doe",
    "committerEmail": "jane@doe.org",
    "message": "Bugfix 123",
    "url": "https://github.com/gimlet-io/gimlet-cli/commit/ea9ab7cc31b2599bf4afcfd639da516ca27a4780"
  },
  "items": [
    {
      "name": "CI",
      "url": "https://jenkins.example.com/job/dev/84/display/redirect"
    }
  ]
}
`

	var a artifact.Artifact
	json.Unmarshal([]byte(artifactStr), &a)

	aModel, err := model.ToArtifactModel(a)
	assert.Nil(t, err)

	savedArtifact, err := s.CreateArtifact(aModel)
	assert.Nil(t, err)
	assert.NotEqual(t, savedArtifact.Created, 0)

	artifacts, err := s.Artifacts()
	assert.Nil(t, err)
	assert.Equal(t, len(artifacts), 1)
}
