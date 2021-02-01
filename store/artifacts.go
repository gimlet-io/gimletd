package store

import (
	"encoding/json"
	"fmt"
	"github.com/gimlet-io/gimletd/artifact"
	"github.com/gimlet-io/gimletd/model"
	"github.com/russross/meddler"
	"strings"
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
func (db *Store) Artifacts(
	app, branch string,
	pr bool,
	sourceBranch string,
	sha string,
	limit, offset int,
	since,until *time.Time) ([]*model.Artifact, error) {

	filters := []string{}
	args := []interface{}{}

	if since != nil {
		filters = addFilter(filters, "created >= ?")
		args = append(args, since.Unix())
	}
	if until != nil {
		filters = addFilter(filters, "created < ?")
		args = append(args, until.Unix())
	}

	if app != "" {
		filters = addFilter(filters, "repository = ?")
		args = append(args, app)
	}
	if branch != "" {
		filters = addFilter(filters, "branch = ?")
		args = append(args, branch)
	}
	if sourceBranch != "" {
		filters = addFilter(filters, "branch = ?")
		args = append(args, sourceBranch)
	}
	if sha != "" {
		filters = addFilter(filters, fmt.Sprintf("branch = %s", sha))
		args = append(args, sha)
	}

	if pr {
		filters = addFilter(filters, fmt.Sprintf(" pr = %t", pr))
	}

	if limit == 0 || offset == 0 {
		limit = 10
	}
	limitAndOffset := fmt.Sprintf("LIMIT %d OFFSET %d", limit, offset)

	query := fmt.Sprintf(`
SELECT id, repository, branch, pr, source_branch, created, blob
FROM artifacts
%s
ORDER BY created desc
%s;`, strings.Join(filters, " "), limitAndOffset)

	var data []*model.Artifact
	err := meddler.QueryAll(db, &data, query, args...)
	return data, err
}

func addFilter(filters []string, filter string) []string {
	if len(filters) == 0 {
		return append(filters, "WHERE " + filter)
	}

	return append(filters, "AND " + filter)
}
