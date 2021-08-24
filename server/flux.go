package server

import (
	"encoding/json"
	"fmt"
	"github.com/fluxcd/pkg/runtime/events"
	"github.com/gimlet-io/gimletd/model"
	"github.com/gimlet-io/gimletd/notifications"
	"github.com/gimlet-io/gimletd/store"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strings"
)

func fluxEvent(w http.ResponseWriter, r *http.Request) {
	var event events.Event
	json.NewDecoder(r.Body).Decode(&event)

	gitopsCommit, err := asGitopsCommit(event)
	if err != nil {
		log.Errorf("could not translate to gitops commit: %s", err)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(""))
	}

	ctx := r.Context()
	notificationsManager := ctx.Value("notificationsManager").(notifications.Manager)
	gitopsRepo := ctx.Value("gitopsRepo").(string)
	notificationsManager.Broadcast(notifications.NewMessage(gitopsRepo, gitopsCommit))

	store := ctx.Value("store").(*store.Store)
	err = store.SaveOrUpdateGitopsCommit(*gitopsCommit)
	if err != nil {
		log.Errorf("could not save or update gitops commit: %s", err)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(""))
}

func asGitopsCommit(event events.Event) (*model.GitopsCommit, error) {
	if _, ok := event.Metadata["revision"]; !ok {
		return nil, fmt.Errorf("could not extract gitops sha from Flux message: %s", event)
	}
	sha := parseRev(event.Metadata["revision"])

	var statusDesc string
	if event.Reason == model.ValidationFailed ||
		event.Reason == model.ReconciliationFailed {
		statusDesc = extractValidationError(event.Message)
	}

	return &model.GitopsCommit{
		Sha:        sha,
		Status:     event.Reason,
		StatusDesc: statusDesc,
	}, nil
}

func parseRev(rev string) string {
	parts := strings.Split(rev, "/")
	if len(parts) != 2 {
		return "n/a"
	}

	return parts[1]
}

func extractValidationError(msg string) string {
	errors := ""
	lines := strings.Split(msg, "\n")
	for _, line := range lines {
		if line != "" &&
			!strings.HasSuffix(line, "created") && !strings.HasSuffix(line, "created (dry run)") &&
			!strings.HasSuffix(line, "configured") && !strings.HasSuffix(line, "configured (dry run)") &&
			!strings.HasSuffix(line, "unchanged") && !strings.HasSuffix(line, "unchanged (dry run)") {
			errors += line + "\n"
		}
	}

	return errors
}
