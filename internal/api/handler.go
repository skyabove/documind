package api

import (
	"encoding/json"
	"net/http"
)

// Handler holds dependencies for HTTP handlers.
type Handler struct {
	// TODO: inject document store, extractor, claude client
}

func NewHandler() *Handler {
	return &Handler{}
}

// RegisterRoutes registers all HTTP routes on mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", h.Health)
	// TODO: document routes
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func respondJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func respondError(w http.ResponseWriter, status int, msg string) {
	respondJSON(w, status, map[string]string{"error": msg})
}
