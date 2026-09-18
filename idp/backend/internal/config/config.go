// Package config reads runtime settings from environment variables.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type Config struct {
	DatabaseURL    string
	SecretKey      string
	DataDir        string // Terraform workspaces, secret files, GitOps checkout
	ModulesDir     string // Terraform module sources (terraform://modules/<name>)
	FixturesDir    string
	ListenAddr     string
	FrontendDir    string // built React bundle served under /ui/ (ADR-017)
	RuntimeProfile string // production | development
	AuthHMACKey    string // protected key material for login rate-limit bucket HMACs
	// AuthDevelopmentCookies permits non-Secure cookies with non-__Host names
	// for an explicitly selected local HTTP development profile.
	AuthDevelopmentCookies bool
	AuthAccountLimit       int
	AuthSourceLimit        int
	AuthAttemptWindow      time.Duration

	// Delivery repositories: every application keeps its desired state in its
	// own repository, which the IDP creates on the application's first
	// deployment. Keys and the hosting credential live in the Secret Store.
	DeliveryRepoPattern string // owner/idp-<app>-gitops
	DeliveryBranch      string
	CDProvider          string // which CD system syncs the clusters: fleet | argocd
	GitHostingTokenRef  string // secret reference of the Git hosting credential
	GitHostingAPI       string
	GitHostingSSHHost   string
	KnownHostsFile      string

	HealthTimeout time.Duration
	PollInterval  time.Duration
}

func Load() (*Config, error) {
	root, _ := os.Getwd()
	c := &Config{
		DatabaseURL:         env("IDP_DATABASE_URL", "postgres://idp:idp@127.0.0.1:55433/idp?sslmode=disable"),
		SecretKey:           os.Getenv("IDP_SECRET_KEY"),
		DataDir:             env("IDP_DATA_DIR", filepath.Join(root, "var")),
		ModulesDir:          env("IDP_MODULES_DIR", filepath.Join(root, "terraform", "modules")),
		FixturesDir:         env("IDP_FIXTURES_DIR", filepath.Join(root, "fixtures")),
		ListenAddr:          env("IDP_LISTEN", "127.0.0.1:8088"),
		FrontendDir:         env("IDP_FRONTEND_DIR", filepath.Join(root, "..", "frontend", "dist")),
		RuntimeProfile:      env("IDP_RUNTIME_PROFILE", "production"),
		AuthHMACKey:         os.Getenv("IDP_AUTH_HMAC_KEY"),
		DeliveryRepoPattern: os.Getenv("IDP_DELIVERY_REPO_PATTERN"),
		DeliveryBranch:      env("IDP_DELIVERY_BRANCH", "main"),
		CDProvider:          env("IDP_CD_PROVIDER", "fleet"),
		GitHostingTokenRef:  env("IDP_GIT_HOSTING_TOKEN_SECRET", "idpsecret://platform/git-hosting-token"),
		GitHostingAPI:       env("IDP_GIT_HOSTING_API", "https://api.github.com"),
		GitHostingSSHHost:   env("IDP_GIT_HOSTING_SSH_HOST", "github.com"),
		KnownHostsFile:      env("IDP_GIT_KNOWN_HOSTS", filepath.Join(env("IDP_DATA_DIR", filepath.Join(root, "var")), "delivery", "known_hosts")),
		HealthTimeout:       duration("IDP_HEALTH_TIMEOUT", 6*time.Minute),
		PollInterval:        duration("IDP_WORKER_POLL", 2*time.Second),
	}
	var err error
	if c.AuthAccountLimit, err = positiveInt("IDP_AUTH_ACCOUNT_LIMIT", 5); err != nil {
		return nil, err
	}
	if c.AuthSourceLimit, err = positiveInt("IDP_AUTH_SOURCE_LIMIT", 20); err != nil {
		return nil, err
	}
	if c.AuthAttemptWindow, err = positiveDuration("IDP_AUTH_ATTEMPT_WINDOW", 15*time.Minute); err != nil {
		return nil, err
	}
	if c.SecretKey == "" {
		return nil, errors.New("IDP_SECRET_KEY is required")
	}
	if c.AuthHMACKey == "" {
		c.AuthHMACKey = c.SecretKey
	}
	if c.RuntimeProfile != "production" && c.RuntimeProfile != "development" {
		return nil, errors.New("IDP_RUNTIME_PROFILE must be production or development")
	}
	switch env("IDP_AUTH_COOKIE_MODE", "secure") {
	case "secure":
	case "development":
		if c.RuntimeProfile != "development" {
			return nil, errors.New("IDP_AUTH_COOKIE_MODE=development requires IDP_RUNTIME_PROFILE=development")
		}
		c.AuthDevelopmentCookies = true
	default:
		return nil, errors.New("IDP_AUTH_COOKIE_MODE must be secure or development")
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

func positiveInt(key string, fallback int) (int, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}
	return value, nil
}

func positiveDuration(key string, fallback time.Duration) (time.Duration, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}
	value, err := time.ParseDuration(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration", key)
	}
	return value, nil
}
