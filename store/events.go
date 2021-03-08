package store

import (
	"encoding/json"
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
	event.ID = fmt.Sprintf("%s-%s", event.Repository, uuid.New().String())
	now := time.Now().Unix()
	event.Created = now
	event.Status = model.StatusNew

	var a dx.Artifact
	err := json.Unmarshal([]byte(event.Blob), &a)
	if err != nil {
		return nil, fmt.Errorf("couldn't deserialize artifact: %s", err)
	}
	a.ID = event.ID
	a.Created = now

	artifactStr, err := json.Marshal(a)
	if err != nil {
		return nil, fmt.Errorf("couldn't serialize artifact: %s", err)
	}
	event.Blob = string(artifactStr)

	return event, meddler.Insert(db, "events", event)
}

// Events returns all events in the database within the given constraints
func (db *Store) Events(
	app, branch string,
	event *dx.GitEvent,
	sourceBranch string,
	sha string,
	limit, offset int,
	since, until *time.Time) ([]*model.Event, error) {

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
		filters = addFilter(filters, fmt.Sprintf("sha = %s", sha))
		args = append(args, sha)
	}

	if event != nil {
		var intRep int
		intRep = int(*event)
		filters = addFilter(filters, fmt.Sprintf(" event = %d", intRep))
	}

	if limit == 0 || offset == 0 {
		limit = 10
	}
	limitAndOffset := fmt.Sprintf("LIMIT %d OFFSET %d", limit, offset)

	query := fmt.Sprintf(`
SELECT id, repository, branch, event, source_branch, target_branch, tag, created, blob, status, status_desc, sha
FROM events
%s
ORDER BY created desc
%s;`, strings.Join(filters, " "), limitAndOffset)

	var data []*model.Event
	err := meddler.QueryAll(db, &data, query, args...)
	return data, err
}

// UnprocessedEvents selects an event timeline
func (db *Store) UnprocessedEvents() (events []*model.Event, err error) {
	stmt := sql.Stmt(db.driver, sql.SelectUnprocessedEvents)
	err = meddler.QueryAll(db, &events, stmt)
	return events, err
}

// UpdateEventStatus updates an event status in the database
func (db *Store) UpdateEventStatus(id string, status string, desc string) error {
	stmt := sql.Stmt(db.driver, sql.UpdateEventStatus)
	_, err := db.Exec(stmt, status, desc, id)
	return err
}

func addFilter(filters []string, filter string) []string {
	if len(filters) == 0 {
		return append(filters, "WHERE "+filter)
	}

	return append(filters, "AND "+filter)
}
