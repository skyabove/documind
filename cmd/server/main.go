// Package main starts the DocuMind HTTP server.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/skyabove/documind/internal/api"
	"github.com/skyabove/documind/internal/claude"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := loadConfig()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	claudeClient := claude.NewClient(cfg.AnthropicAPIKey)

	srv := api.NewServer(api.Config{
		StoragePath: cfg.StoragePath,
		MaxUploadMB: cfg.MaxUploadMB,
		Claude:      claudeClient,
	})

	httpServer := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           srv.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("server starting", "port", cfg.Port)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http server failed", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	slog.Info("shutdown initiated")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown failed", "error", err)
		os.Exit(1)
	}
	slog.Info("shutdown complete")
}

type config struct {
	Port            string
	StoragePath     string
	MaxUploadMB     int64
	AnthropicAPIKey string
}

func loadConfig() (config, error) {
	cfg := config{
		Port:            getEnv("PORT", "8080"),
		StoragePath:     getEnv("STORAGE_PATH", "./data/documents"),
		MaxUploadMB:     20,
		AnthropicAPIKey: os.Getenv("ANTHROPIC_API_KEY"),
	}
	if cfg.AnthropicAPIKey == "" {
		return config{}, errors.New("ANTHROPIC_API_KEY environment variable is required")
	}
	if err := os.MkdirAll(cfg.StoragePath, 0o755); err != nil {
		return config{}, err
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
