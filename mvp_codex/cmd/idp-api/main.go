package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/thanhnt1/final-idp/mvp-codex/internal/api"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/service"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/store/postgres"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	databaseURL := required("DATABASE_URL", logger)
	token := required("IDP_TOKEN", logger)
	listen := envOr("IDP_LISTEN", "127.0.0.1:8081")
	stateRoot := envOr("IDP_STATE_ROOT", ".state/resources")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	store, err := postgres.Open(ctx, databaseURL)
	if err != nil {
		logger.Error("open database", "error", err)
		os.Exit(1)
	}
	defer store.Close()
	if err := store.Migrate(ctx); err != nil {
		logger.Error("migrate database", "error", err)
		os.Exit(1)
	}
	cancel()
	server := api.New(token, service.NewDeploymentService(store, stateRoot), logger)
	httpServer := &http.Server{Addr: listen, Handler: server.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	logger.Info("IDP API listening", "address", listen)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("http server stopped", "error", err)
		os.Exit(1)
	}
}

func required(name string, logger *slog.Logger) string {
	value := os.Getenv(name)
	if value == "" {
		logger.Error("required environment variable is missing", "name", name)
		os.Exit(2)
	}
	return value
}
func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
