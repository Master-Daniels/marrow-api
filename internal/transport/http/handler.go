package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Master-Daniels/marrow/internal/feed"
	"github.com/Master-Daniels/marrow/internal/storage"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	repo storage.Repository
	feed *feed.Service
}

// New returns an http.Handler with the repo attached
func NewHandler(repo storage.Repository, feed *feed.Service) *Handler {
	return &Handler{repo: repo, feed: feed}
}

// GetFeed responds with the latest N items (simple JSON for now)
func (h *Handler) GetFeed(w http.ResponseWriter, r *http.Request) {
	// query param ?limit=20 (default 20)
	limit := 20
	if q := r.URL.Query().Get("limit"); q != "" {
		if v, _ := strconv.Atoi(q); v > 0 {
			limit = v
		}
	}
	items, err := h.repo.Latest(r.Context(), limit)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(items)
}

// SSEHandler wires the Feed Service’s notification channel into an
// HTTP endpoint that streams JSON objects separated by a newline.
func (h *Handler) SSEHandler(w http.ResponseWriter, r *http.Request) {

	// Set the SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Subscribe to the service
	ch := h.feed.Subscribe()
	enc := json.NewEncoder(w)

	// Loop until client disconnects
	for {
		select {
		case <-r.Context().Done():
			return // client closed connection
		case itm := <-ch:
			// SSE format: `data: <json>\n\n`
			if _, err := w.Write([]byte("data: ")); err != nil {
				return
			}
			if err := enc.Encode(itm); err != nil {
				return
			}
			if _, err := w.Write([]byte("\n")); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

// SetSourcesHandler - new endpoint to list all sources with their state
func (h *Handler) GetSourceHandler(w http.ResponseWriter, r *http.Request) {
	sourceID := chi.URLParam(r, "sourceID")
	if sourceID == "" {
		http.Error(w, "missing sourceID", http.StatusBadRequest)
		return
	}

	sources, err := h.repo.GetSource(r.Context(), sourceID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sources)
}

func (h *Handler) GetSourcesHandler(w http.ResponseWriter, r *http.Request) {
	sources, err := h.repo.GetAllSources(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sources)
}

// SetSourcesHandler - new endpoint to list all sources with their state
func (h *Handler) GetSourceAndPollingStateHandler(w http.ResponseWriter, r *http.Request) {
	sourceID := chi.URLParam(r, "sourceID")
	if sourceID == "" {
		http.Error(w, "missing sourceID", http.StatusBadRequest)
		return
	}

	sources, err := h.repo.GetSourceAndPollingState(r.Context(), sourceID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sources)
}

func (h *Handler) GetSourcesAndPollingStateHandler(w http.ResponseWriter, r *http.Request) {
	sources, err := h.repo.GetSourcesAndPollingState(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sources)
}
