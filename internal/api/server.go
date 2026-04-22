package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/skyabove/documind/internal/claude"
)

// Config holds runtime configuration for the API server.
type Config struct {
	StoragePath string
	MaxUploadMB int64
	Claude      *claude.Client
}

// Server is the root HTTP handler container.
type Server struct {
	cfg Config
}

// NewServer constructs a Server with the given configuration.
func NewServer(cfg Config) *Server {
	return &Server{cfg: cfg}
}

// Routes returns the HTTP handler wired with all routes and middleware.
func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	r.Get("/health", s.handleHealth)
	r.Post("/documents", s.handleUploadDocument)
	r.Post("/documents/{id}/analyze", s.handleAnalyzeDocument)

	return r
}
