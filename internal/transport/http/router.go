package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func New(h *Handler) http.Handler {
	r := chi.NewRouter()

	// Example endpoint – you’ll flesh it out later
	r.Get("/api/v1/feed", h.GetFeed)
	r.Get("/api/v1/updates", h.SSEHandler)
	r.Get("/api/v1/sources", h.GetSourceHandler)
	r.Get("/api/v1/sources/{sourceID}", h.GetSourceHandler)
	r.Get("/api/v1/sources/polling_state", h.GetSourcesAndPollingStateHandler)
	r.Get("/api/v1/sources/{sourceID}/polling_state", h.GetSourceAndPollingStateHandler)

	// Middleware chain (recovery, logging, etc.) can be added here
	return r
}
