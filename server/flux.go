package server

import (
	"encoding/json"
	"github.com/fluxcd/pkg/runtime/events"
	"github.com/gimlet-io/gimletd/notifications"
	"net/http"
)

func fluxEvent(w http.ResponseWriter, r *http.Request) {
	var event events.Event
	json.NewDecoder(r.Body).Decode(&event)

	ctx := r.Context()
	notificationsManager := ctx.Value("notificationsManager").(notifications.Manager)
	gitopsRepo := ctx.Value("gitopsRepo").(string)
	notificationsManager.Broadcast(notifications.MessageFromFluxEvent(gitopsRepo, &event))

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(""))
}
