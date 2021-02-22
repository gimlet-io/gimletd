package server

import (
	"github.com/gimlet-io/gimletd/cmd/config"
	"github.com/gimlet-io/gimletd/notifications"
	"github.com/gimlet-io/gimletd/server/session"
	"github.com/gimlet-io/gimletd/store"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"net/http"
	"time"
)

func SetupRouter(
	config *config.Config,
	store *store.Store,
	notificationsManager notifications.Manager,
) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.NoCache)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Use(middleware.WithValue("store", store))
	r.Use(middleware.WithValue("notificationsManager", notificationsManager))
	r.Use(middleware.WithValue("gitopsRepo", config.GitopsRepo))

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:8888", config.Host},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Group(func(r chi.Router) {
		r.Use(session.SetUser())
		r.Use(session.MustUser())
		r.Post("/api/artifact", saveArtifact)
		r.Get("/api/artifacts", getArtifacts)
		r.Post("/api/flux-events", fluxEvent)
	})

	r.Group(func(r chi.Router) {
		r.Use(session.SetUser())
		r.Use(session.MustAdmin())
		r.Get("/api/user/{login}", getUser)
		r.Post("/api/user", saveUser)
		r.Delete("/api/user/{login}", deleteUser)
		r.Get("/api/users", getUsers)
	})

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	return r
}
