package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/skyabove/documind/internal/claude"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

type healthResponse struct {
	Status string `json:"status"`
	Time   string `json:"time"`
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{
		Status: "ok",
		Time:   time.Now().UTC().Format(time.RFC3339),
	})
}

type uploadResponse struct {
	DocumentID string `json:"document_id"`
	Filename   string `json:"filename"`
	SizeBytes  int64  `json:"size_bytes"`
}

func (s *Server) handleUploadDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	maxBytes := s.cfg.MaxUploadMB * 1024 * 1024
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

	if err := r.ParseMultipartForm(maxBytes); err != nil {
		writeError(w, http.StatusBadRequest, "file too large or malformed multipart")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing 'file' field in form")
		return
	}
	defer file.Close()

	if filepath.Ext(header.Filename) != ".pdf" {
		writeError(w, http.StatusBadRequest, "only .pdf files are accepted")
		return
	}

	docID := uuid.NewString()
	targetPath := filepath.Join(s.cfg.StoragePath, docID+".pdf")

	dst, err := os.Create(targetPath)
	if err != nil {
		slog.ErrorContext(ctx, "create file", "error", err, "path", targetPath)
		writeError(w, http.StatusInternalServerError, "storage error")
		return
	}
	defer dst.Close()

	written, err := io.Copy(dst, file)
	if err != nil {
		slog.ErrorContext(ctx, "write file", "error", err)
		writeError(w, http.StatusInternalServerError, "storage error")
		return
	}

	slog.InfoContext(ctx, "document uploaded", "id", docID, "filename", header.Filename, "bytes", written)

	writeJSON(w, http.StatusCreated, uploadResponse{
		DocumentID: docID,
		Filename:   header.Filename,
		SizeBytes:  written,
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("encode json", "error", err)
	}
}

type errorResponse struct {
	Error string `json:"error"`
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{Error: fmt.Sprintf("%s", message)})
}

// В handlers.go, допиши импорты:
//   "encoding/base64"
//   "github.com/<твой-username>/documind/internal/claude"
//   "github.com/go-chi/chi/v5"

type analyzeResponse struct {
	DocumentID   string `json:"document_id"`
	Summary      string `json:"summary"`
	Iterations   int    `json:"iterations"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
}

func (s *Server) handleAnalyzeDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	docID := chi.URLParam(r, "id")
	if docID == "" {
		writeError(w, http.StatusBadRequest, "missing document id")
		return
	}

	// Load PDF from storage.
	path := filepath.Join(s.cfg.StoragePath, docID+".pdf")
	pdfBytes, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "document not found")
			return
		}
		slog.ErrorContext(ctx, "read pdf", "error", err, "path", path)
		writeError(w, http.StatusInternalServerError, "storage error")
		return
	}

	// Build the initial user content: PDF as document block + instruction text.
	pdfB64 := base64.StdEncoding.EncodeToString(pdfBytes)
	initialContent := []claude.ContentBlock{
		{
			Type: claude.ContentTypeDocument,
			Source: &claude.DocumentSource{
				Type:      "base64",
				MediaType: "application/pdf",
				Data:      pdfB64,
			},
		},
		{
			Type: claude.ContentTypeText,
			Text: "Analyze this document. Produce a concise summary (3-5 sentences) covering: the document's purpose, main topics, and any key data points or conclusions. Return only the summary text, no preamble.",
		},
	}

	result, err := s.cfg.Claude.RunAgent(ctx, claude.RunAgentOptions{
		System:         "You are a document analysis assistant. Your summaries are concise, factual, and preserve key numerical data.",
		InitialContent: initialContent,
		MaxIterations:  3, // no tools yet — should finish in 1 iteration
		MaxTokens:      1024,
	})
	if err != nil {
		slog.ErrorContext(ctx, "agent run", "error", err, "document_id", docID)
		writeError(w, http.StatusBadGateway, "analysis failed")
		return
	}

	writeJSON(w, http.StatusOK, analyzeResponse{
		DocumentID:   docID,
		Summary:      result.FinalText,
		Iterations:   result.Iterations,
		InputTokens:  result.TotalUsage.InputTokens,
		OutputTokens: result.TotalUsage.OutputTokens,
	})
}

func (s *Server) handleExtractDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	docID := chi.URLParam(r, "id")
	if docID == "" {
		writeError(w, http.StatusBadRequest, "missing document id")
		return
	}

	path := filepath.Join(s.cfg.StoragePath, docID+".pdf")
	pdfBytes, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "document not found")
			return
		}
		slog.ErrorContext(ctx, "read pdf", "error", err, "path", path)
		writeError(w, http.StatusInternalServerError, "storage error")
		return
	}

	result, err := s.extractor.Extract(ctx, docID, pdfBytes)
	if err != nil {
		slog.ErrorContext(ctx, "extract", "error", err, "document_id", docID)
		writeError(w, http.StatusBadGateway, "extraction failed")
		return
	}

	slog.InfoContext(ctx, "extraction complete",
		"document_id", docID,
		"iterations", result.Iterations,
		"input_tokens", result.Usage.InputTokens,
		"output_tokens", result.Usage.OutputTokens,
		"summary_extracted", result.Summary != nil,
		"entity_count", len(result.Entities),
	)

	writeJSON(w, http.StatusOK, result)
}
