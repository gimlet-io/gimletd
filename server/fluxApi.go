package server

import (
	"encoding/json"
	"github.com/fluxcd/pkg/recorder"
	"github.com/gimlet-io/gimletd/notifications"
	"net/http"
)

func fluxEvent(w http.ResponseWriter, r *http.Request) {
	var event recorder.Event
	json.NewDecoder(r.Body).Decode(&event)

	ctx := r.Context()
	notificationsManager := ctx.Value("notificationsManager").(notifications.Manager)
	notificationsManager.Broadcast(notifications.MessageFromFluxEvent(&event))

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(""))
}
