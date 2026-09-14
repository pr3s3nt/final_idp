package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/thanhnt1/final-idp/mvp-codex/internal/artifact"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/delivery"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/identity"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/platform"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/processlock"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/provisioner"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/render"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/store/postgres"
	workerpkg "github.com/thanhnt1/final-idp/mvp-codex/internal/worker"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	repositoryRoot, err := filepath.Abs(envOr("IDP_REPOSITORY_ROOT", "."))
	if err != nil {
		logger.Error("resolve repository root", "error", err)
		os.Exit(2)
	}
	lock, err := processlock.Acquire(envOr("IDP_WORKER_LOCK", filepath.Join(repositoryRoot, ".runtime", "idp-worker.lock")))
	if err != nil {
		logger.Error("acquire exclusive worker lock", "error", err)
		os.Exit(1)
	}
	defer lock.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	store, err := postgres.Open(ctx, required("DATABASE_URL", logger))
	if err != nil {
		logger.Error("open metadata database", "error", err)
		os.Exit(1)
	}
	defer store.Close()
	if err := store.Migrate(ctx); err != nil {
		logger.Error("migrate metadata database", "error", err)
		os.Exit(1)
	}

	terraform := provisioner.NewTerraform(repositoryRoot, repositoryRoot)
	registry := provisioner.NewRegistry()
	if err := registry.Register("terraform", terraform); err != nil {
		logger.Error("register Terraform provisioner", "error", err)
		os.Exit(1)
	}
	runner := &workerpkg.Worker{
		Store:    store,
		Engine:   platform.NewEngine(envOr("IDP_STATE_ROOT", ".state/resources")),
		Registry: registry,
		Renderer: render.NewScore(),
		Artifacts: &artifact.OCI{
			Repository:       required("IDP_ARTIFACT_REPOSITORY", logger),
			SourceRepository: os.Getenv("IDP_ARTIFACT_SOURCE_REPOSITORY"),
			PlainHTTP:        strings.EqualFold(os.Getenv("IDP_ARTIFACT_PLAIN_HTTP"), "true"),
		},
		Delivery: delivery.NewArgoCD(envOr("IDP_ARGO_NAMESPACE", "argocd")),
	}
	workerRunID, err := identity.NewUUID()
	if err != nil {
		logger.Error("create worker run ID", "error", err)
		os.Exit(1)
	}
	logger.Info("IDP worker started", "workerRunId", workerRunID)
	for {
		worked, err := runner.RunOnce(ctx, workerRunID)
		if err != nil {
			logger.Error("deployment execution stopped", "error", err)
			os.Exit(1)
		}
		if worked {
			continue
		}
		select {
		case <-ctx.Done():
			logger.Info("IDP worker stopped")
			return
		case <-time.After(2 * time.Second):
		}
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
