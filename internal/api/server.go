package api

import (
	"github.com/skyabove/documind/internal/extraction"
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
	cfg       Config
	extractor *extraction.Extractor
}

// NewServer constructs a Server with the given configuration.
func NewServer(cfg Config) *Server {
	return &Server{
		cfg:       cfg,
		extractor: extraction.NewExtractor(cfg.Claude),
	}
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
	r.Post("/documents/{id}/extract", s.handleExtractDocument)

	return r
}
