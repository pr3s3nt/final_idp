//go:build integration

package service_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"idp/internal/authentication"
	"idp/internal/persistence"
)

func TestLocalAuthenticationPersistence(t *testing.T) {
	environment := setup(t)
	repository := &persistence.AuthenticationRepository{DB: environment.db}
	auth, err := authentication.NewService(repository, bytes.Repeat([]byte{0x35}, 32))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := auth.CreateLocalUser(ctx, " Developer ", "Developer One", "a sufficiently long password"); err != nil {
		t.Fatalf("create local user: %v", err)
	}
	if err := auth.CreateLocalUser(ctx, "developer", "Duplicate", "a different long password"); !errors.Is(err, authentication.ErrUsernameTaken) {
		t.Fatalf("duplicate local user error = %v", err)
	}

	material, err := auth.SignIn(ctx, "developer", "a sufficiently long password", "127.0.0.1", "/ui/applications")
	if err != nil {
		t.Fatalf("sign in: %v", err)
	}
	principal, err := auth.Authenticate(ctx, material.SessionToken)
	if err != nil || principal.Username != "developer" || principal.DisplayName != "Developer One" {
		t.Fatalf("authenticate principal=%#v error=%v", principal, err)
	}

	if err := auth.ResetLocalPassword(ctx, "developer", "a different long password"); err != nil {
		t.Fatalf("reset password: %v", err)
	}
	if _, err := auth.Authenticate(ctx, material.SessionToken); !errors.Is(err, authentication.ErrInvalidCredentials) {
		t.Fatalf("old session after password reset error = %v", err)
	}
	if _, err := auth.SignIn(ctx, "developer", "a sufficiently long password", "127.0.0.1", ""); !errors.Is(err, authentication.ErrInvalidCredentials) {
		t.Fatalf("old password sign-in error = %v", err)
	}
	second, err := auth.SignIn(ctx, "developer", "a different long password", "127.0.0.1", "")
	if err != nil {
		t.Fatalf("new password sign in: %v", err)
	}
	if err := auth.SetLocalUserStatus(ctx, "developer", "DISABLED"); err != nil {
		t.Fatalf("disable user: %v", err)
	}
	if _, err := auth.Authenticate(ctx, second.SessionToken); !errors.Is(err, authentication.ErrInvalidCredentials) {
		t.Fatalf("disabled user session error = %v", err)
	}
}
