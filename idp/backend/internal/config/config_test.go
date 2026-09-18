package config

import (
	"testing"
	"time"
)

func TestAuthenticationDefaultsAndOverrides(t *testing.T) {
	t.Setenv("IDP_SECRET_KEY", "test-only-secret")
	t.Setenv("IDP_AUTH_HMAC_KEY", "")
	t.Setenv("IDP_RUNTIME_PROFILE", "")
	t.Setenv("IDP_AUTH_COOKIE_MODE", "")
	t.Setenv("IDP_AUTH_ACCOUNT_LIMIT", "")
	t.Setenv("IDP_AUTH_SOURCE_LIMIT", "")
	t.Setenv("IDP_AUTH_ATTEMPT_WINDOW", "")
	config, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if config.AuthHMACKey != "test-only-secret" || config.AuthDevelopmentCookies ||
		config.AuthAccountLimit != 5 || config.AuthSourceLimit != 20 || config.AuthAttemptWindow != 15*time.Minute {
		t.Fatalf("authentication defaults = %#v", config)
	}

	t.Setenv("IDP_AUTH_HMAC_KEY", "separate-auth-key")
	t.Setenv("IDP_RUNTIME_PROFILE", "development")
	t.Setenv("IDP_AUTH_COOKIE_MODE", "development")
	t.Setenv("IDP_AUTH_ACCOUNT_LIMIT", "7")
	t.Setenv("IDP_AUTH_SOURCE_LIMIT", "30")
	t.Setenv("IDP_AUTH_ATTEMPT_WINDOW", "10m")
	config, err = Load()
	if err != nil {
		t.Fatalf("Load() with overrides error = %v", err)
	}
	if config.AuthHMACKey != "separate-auth-key" || !config.AuthDevelopmentCookies ||
		config.AuthAccountLimit != 7 || config.AuthSourceLimit != 30 || config.AuthAttemptWindow != 10*time.Minute {
		t.Fatalf("authentication overrides = %#v", config)
	}
}

func TestInvalidAuthenticationConfiguration(t *testing.T) {
	for key, value := range map[string]string{
		"IDP_AUTH_COOKIE_MODE":    "maybe",
		"IDP_AUTH_ACCOUNT_LIMIT":  "0",
		"IDP_AUTH_SOURCE_LIMIT":   "many",
		"IDP_AUTH_ATTEMPT_WINDOW": "never",
	} {
		t.Run(key, func(t *testing.T) {
			t.Setenv("IDP_SECRET_KEY", "test-only-secret")
			t.Setenv("IDP_RUNTIME_PROFILE", "production")
			t.Setenv("IDP_AUTH_COOKIE_MODE", "secure")
			t.Setenv("IDP_AUTH_ACCOUNT_LIMIT", "5")
			t.Setenv("IDP_AUTH_SOURCE_LIMIT", "20")
			t.Setenv("IDP_AUTH_ATTEMPT_WINDOW", "15m")
			t.Setenv(key, value)
			if _, err := Load(); err == nil {
				t.Fatalf("Load() accepted %s=%q", key, value)
			}
		})
	}
	t.Run("development cookies in production", func(t *testing.T) {
		t.Setenv("IDP_SECRET_KEY", "test-only-secret")
		t.Setenv("IDP_RUNTIME_PROFILE", "production")
		t.Setenv("IDP_AUTH_COOKIE_MODE", "development")
		if _, err := Load(); err == nil {
			t.Fatal("Load() accepted development cookies in the production profile")
		}
	})
	t.Run("invalid runtime profile", func(t *testing.T) {
		t.Setenv("IDP_SECRET_KEY", "test-only-secret")
		t.Setenv("IDP_RUNTIME_PROFILE", "staging")
		t.Setenv("IDP_AUTH_COOKIE_MODE", "secure")
		if _, err := Load(); err == nil {
			t.Fatal("Load() accepted an unknown runtime profile")
		}
	})
}
