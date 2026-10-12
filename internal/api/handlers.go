package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/knorVV/github_sync_go/internal/scheduler"
	"github.com/knorVV/github_sync_go/internal/storage"
)

type Handler struct {
	store     *storage.RepoStore
	scheduler *scheduler.Scheduler
}

func NewHandler(store *storage.RepoStore, sched *scheduler.Scheduler) *Handler {
	return &Handler{store: store, scheduler: sched}
}

func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()
	r.Get("/api/repos", h.list)
	r.Get("/api/repos/{id}", h.getByID)
	r.Post("/api/sync", h.sync)
	return r
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	repos, err := h.store.ListAll(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"count": len(repos),
		"data":  repos,
	})
}

func (h *Handler) getByID(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")

	if id, err := strconv.ParseInt(idParam, 10, 64); err == nil {
		repo, err := h.store.GetByGithubID(r.Context(), id)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if repo == nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
			return
		}
		writeJSON(w, http.StatusOK, repo)
		return
	}

	repo, err := h.store.GetByName(r.Context(), idParam)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if repo == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
		return
	}
	writeJSON(w, http.StatusOK, repo)
}

func (h *Handler) sync(w http.ResponseWriter, r *http.Request) {
	go h.scheduler.ForceSync()
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "sync_started"})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
