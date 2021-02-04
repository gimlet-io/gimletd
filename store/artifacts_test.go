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

	artifacts, err := s.Artifacts("", "", false, "", "", 0, 0, nil, nil)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(artifacts))
	assert.Equal(t, "ea9ab7cc31b2599bf4afcfd639da516ca27a4780", artifacts[0].SHA)
}
