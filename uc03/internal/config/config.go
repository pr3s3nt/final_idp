// Package config reads runtime settings from environment variables.
package config

import (
	"errors"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	DatabaseURL string
	SecretKey   string
	DataDir     string // Terraform workspaces, secret files, GitOps checkout
	ModulesDir  string // Terraform module sources (terraform://modules/<name>)
	FixturesDir string
	ListenAddr  string

	GitOpsRepo       string // git@github.com:owner/repo.git
	GitOpsKeyFile    string // write deploy key used by the IDP
	GitOpsReadKeyFile string // read-only deploy key given to Argo CD
	GitOpsBranch     string

	HealthTimeout time.Duration
	PollInterval  time.Duration
}

func Load() (*Config, error) {
	root, _ := os.Getwd()
	c := &Config{
		DatabaseURL:       env("IDP_DATABASE_URL", "postgres://idp:idp@127.0.0.1:55433/idp?sslmode=disable"),
		SecretKey:         os.Getenv("IDP_SECRET_KEY"),
		DataDir:           env("IDP_DATA_DIR", filepath.Join(root, "var")),
		ModulesDir:        env("IDP_MODULES_DIR", filepath.Join(root, "terraform", "modules")),
		FixturesDir:       env("IDP_FIXTURES_DIR", filepath.Join(root, "fixtures")),
		ListenAddr:        env("IDP_LISTEN", "127.0.0.1:8088"),
		GitOpsRepo:        os.Getenv("IDP_GITOPS_REPO"),
		GitOpsKeyFile:     os.Getenv("IDP_GITOPS_SSH_KEY_FILE"),
		GitOpsReadKeyFile: os.Getenv("IDP_GITOPS_READ_SSH_KEY_FILE"),
		GitOpsBranch:      env("IDP_GITOPS_BRANCH", "main"),
		HealthTimeout:     duration("IDP_HEALTH_TIMEOUT", 6*time.Minute),
		PollInterval:      duration("IDP_WORKER_POLL", 2*time.Second),
	}
	if c.SecretKey == "" {
		return nil, errors.New("IDP_SECRET_KEY is required")
	}
	return c, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func duration(key string, fallback time.Duration) time.Duration {
	if v, err := time.ParseDuration(os.Getenv(key)); err == nil {
		return v
	}
	return fallback
}
