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

// Artifacts returns all artifacts in the database within the given constraints
func (db *Store) Artifacts(limit, offset int, since,until *time.Time) ([]*model.Artifact, error) {
	if (limit != 0 || offset != 0) &&
		since != nil || until != nil {
		return []*model.Artifact{}, fmt.Errorf("use either limit - offset or since - until")
	}

	if since != nil || until != nil {
		return db.artifactsSinceUntil(since, until)
	}

	if limit == 0 || offset == 0 {
		limit = 10
	}

	return db.artifactsLimitOffset(limit, offset)
}

func (db *Store) artifactsLimitOffset(limit, offset int) ([]*model.Artifact, error) {
	stmt := sql.Stmt(db.driver, sql.SelectArtifactsLimitOffset)
	var data []*model.Artifact
	err := meddler.QueryAll(db, &data, stmt, limit, offset)
	return data, err
}

func (db *Store) artifactsSinceUntil(since,until *time.Time) ([]*model.Artifact, error) {
	if since == nil || until == nil {
		return []*model.Artifact{}, fmt.Errorf("you must set both since and until")
	}

	sinceUnix := since.Unix()
	untilUnix := until.Unix()

	stmt := sql.Stmt(db.driver, sql.SelectArtifactsSinceUntil)
	var data []*model.Artifact
	err := meddler.QueryAll(db, &data, stmt, sinceUnix, untilUnix)
	return data, err
}


