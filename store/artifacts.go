package store

import (
	"encoding/json"
	"fmt"
	"github.com/gimlet-io/gimletd/artifact"
	"github.com/gimlet-io/gimletd/model"
	"github.com/gimlet-io/gimletd/store/sql"
	"github.com/russross/meddler"
	"time"
)

// CreateArtifact stores a new artifact in the database
func (db *Store) CreateArtifact(artifactModel *model.Artifact) (*model.Artifact, error) {
	// setting created on model and in artifact blob
	now := time.Now().Unix()
	artifactModel.Created = now
	var a artifact.Artifact
	err := json.Unmarshal([]byte(artifactModel.Blob), &a)
	if err != nil {
		return nil, fmt.Errorf("couldn't deserialize artifact: %s", err)
	}
	a.Created = now

	artifactStr, err := json.Marshal(a)
	if err != nil {
		return nil, fmt.Errorf("couldn't serialize artifact: %s", err)
	}
	artifactModel.Blob = string(artifactStr)

	return artifactModel, meddler.Insert(db, "artifacts", artifactModel)
}

// Users returns all users in the database
func (db *Store) Artifacts() ([]*model.Artifact, error) {
	stmt := sql.Stmt(db.driver, sql.SelectArtifactsByDate)
	var data []*model.Artifact
	err := meddler.QueryAll(db, &data, stmt)
	return data, err
}
