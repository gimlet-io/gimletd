package store

import (
	"fmt"
	"github.com/gimlet-io/gimletd/dx"
	"github.com/gimlet-io/gimletd/model"
	"github.com/gimlet-io/gimletd/store/sql"
	"github.com/google/uuid"
	"github.com/russross/meddler"
	"strings"
	"time"
)

// CreateEvent stores a new event in the database
func (db *Store) CreateEvent(event *model.Event) (*model.Event, error) {
	event.ID = uuid.New().String()
	event.Created = time.Now().Unix()
	event.Status = model.StatusNew
	return event, meddler.Insert(db, "events", event)
}

// Artifacts returns all events in the database within the given constraints
func (db *Store) Artifacts(
	repo, branch string,
	gitEvent *dx.GitEvent,
	sourceBranch string,
	sha []string,
	limit, offset int,
	since, until *time.Time) ([]*model.Event, error) {

	filters := []string{}
	args := []interface{}{}

	filters = addFilter(filters, "type = ?")
	args = append(args, model.TypeArtifact)

	if since != nil {
		filters = addFilter(filters, "created >= ?")
		args = append(args, since.Unix())
	}
	if until != nil {
		filters = addFilter(filters, "created < ?")
		args = append(args, until.Unix())
	}

	if repo != "" {
		filters = addFilter(filters, "repository = ?")
		args = append(args, repo)
	}
	if branch != "" {
		filters = addFilter(filters, "branch = ?")
		args = append(args, branch)
	}
	if sourceBranch != "" {
		filters = addFilter(filters, "branch = ?")
		args = append(args, sourceBranch)
	}
	if len(sha) != 0 {
		filters = addFilter(filters, "sha in (?" + strings.Repeat(",?", len(sha)-1) + ")")
		for _, s := range sha {
			args = append(args, s)
		}
	}

	if gitEvent != nil {
		var intRep int
		intRep = int(*gitEvent)
		filters = addFilter(filters, fmt.Sprintf(" event = %d", intRep))
	}

	if limit == 0 && offset == 0 {
		limit = 10
	}
	limitAndOffset := fmt.Sprintf("LIMIT %d OFFSET %d", limit, offset)

	query := fmt.Sprintf(`
SELECT id, repository, branch, event, source_branch, target_branch, tag, created, blob, status, status_desc, sha, artifact_id
FROM events
%s
ORDER BY created desc
%s;`, strings.Join(filters, " "), limitAndOffset)

	var data []*model.Event
	err := meddler.QueryAll(db, &data, query, args...)
	return data, err
}

// Artifact returns an artifact by id
func (db *Store) Artifact(id string) (*model.Event, error) {
	query := fmt.Sprintf(`
SELECT id, repository, branch, event, source_branch, target_branch, tag, created, blob, status, status_desc, sha, artifact_id
FROM events
WHERE artifact_id = ?;
`)

	var data model.Event
	err := meddler.QueryRow(db, &data, query, id)
	return &data, err
}

// Event returns an event by id
func (db *Store) Event(id string) (*model.Event, error) {
	query := fmt.Sprintf(`
SELECT id, created, blob, status, status_desc, gitops_hashes
FROM events
WHERE id = ?;
`)

	var data model.Event
	err := meddler.QueryRow(db, &data, query, id)
	return &data, err
}

// UnprocessedEvents selects an event timeline
func (db *Store) UnprocessedEvents() (events []*model.Event, err error) {
	stmt := sql.Stmt(db.driver, sql.SelectUnprocessedEvents)
	err = meddler.QueryAll(db, &events, stmt)
	return events, err
}

// UpdateEventStatus updates an event status in the database
func (db *Store) UpdateEventStatus(id string, status string, desc string, gitopsStatusString string) error {
	stmt := sql.Stmt(db.driver, sql.UpdateEventStatus)
	_, err := db.Exec(stmt, status, desc, gitopsStatusString, id)
	return err
}

func addFilter(filters []string, filter string) []string {
	if len(filters) == 0 {
		return append(filters, "WHERE "+filter)
	}

	return append(filters, "AND "+filter)
}
