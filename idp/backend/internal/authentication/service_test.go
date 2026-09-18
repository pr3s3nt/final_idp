package authentication

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

type memoryRepository struct {
	credential *CredentialRecord
	session    *SessionRecord
	failures   map[string]int
	blocked    map[string]time.Time
	revoked    bool
	created    *CredentialRecord
	resetHash  string
	status     string
}

func (r *memoryRepository) FindCredential(_ context.Context, username string) (*CredentialRecord, error) {
	if r.credential != nil && r.credential.Username == username {
		copy := *r.credential
		return &copy, nil
	}
	return nil, nil
}

func (r *memoryRepository) BlockedUntil(_ context.Context, accountKey, sourceKey []byte, _ time.Time) (time.Time, error) {
	var result time.Time
	for _, key := range [][]byte{accountKey, sourceKey} {
		if until := r.blocked[string(key)]; until.After(result) {
			result = until
		}
	}
	return result, nil
}

func (r *memoryRepository) RecordFailure(_ context.Context, scope string, key []byte, limit int, now time.Time, window time.Duration) (time.Time, error) {
	if r.failures == nil {
		r.failures = map[string]int{}
	}
	if r.blocked == nil {
		r.blocked = map[string]time.Time{}
	}
	id := scope + ":" + string(key)
	r.failures[id]++
	if r.failures[id] >= limit {
		r.blocked[string(key)] = now.Add(window)
		return r.blocked[string(key)], nil
	}
	return time.Time{}, nil
}

func (r *memoryRepository) CreateSessionAndClearFailures(_ context.Context, session SessionRecord, _ []byte) error {
	r.session = &session
	return nil
}

func (r *memoryRepository) AuthenticateSession(_ context.Context, tokenHash []byte, now, idleCutoff time.Time) (*AuthenticatedSession, error) {
	if r.session == nil || r.revoked || !bytes.Equal(r.session.TokenHash, tokenHash) ||
		!r.session.ExpiresAt.After(now) || !r.session.LastSeenAt.After(idleCutoff) {
		return nil, nil
	}
	r.session.LastSeenAt = now
	return &AuthenticatedSession{SessionID: r.session.SessionID, UserID: r.session.UserID,
		Username: r.credential.Username, DisplayName: r.credential.DisplayName,
		CSRFHash: append([]byte(nil), r.session.CSRFHash...)}, nil
}

func (r *memoryRepository) RevokeSession(_ context.Context, tokenHash []byte, _ time.Time) error {
	if r.session != nil && bytes.Equal(r.session.TokenHash, tokenHash) {
		r.revoked = true
	}
	return nil
}

func (r *memoryRepository) CreateLocalUser(_ context.Context, username, displayName, passwordHash string, _ time.Time) error {
	r.created = &CredentialRecord{Username: username, DisplayName: displayName, PasswordHash: passwordHash, Status: "ACTIVE"}
	return nil
}

func (r *memoryRepository) ResetLocalPassword(_ context.Context, username, passwordHash string, _ time.Time) error {
	if r.credential == nil || r.credential.Username != username {
		return ErrUserNotFound
	}
	r.resetHash = passwordHash
	return nil
}

func (r *memoryRepository) SetLocalUserStatus(_ context.Context, username, status string, _ time.Time) error {
	if r.credential == nil || r.credential.Username != username {
		return ErrUserNotFound
	}
	r.status = status
	return nil
}

func newTestService(t *testing.T, repo *memoryRepository) *Service {
	t.Helper()
	service, err := NewService(repo, bytes.Repeat([]byte{0x42}, 32))
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	service.Now = func() time.Time { return time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC) }
	return service
}

func TestSignInAuthenticateAndSignOut(t *testing.T) {
	hasher := PasswordHasher{}
	hash, err := hasher.Hash("a sufficiently long password")
	if err != nil {
		t.Fatal(err)
	}
	repo := &memoryRepository{credential: &CredentialRecord{UserID: "user-1", Username: "developer",
		DisplayName: "Developer", Status: "ACTIVE", PasswordHash: hash}}
	service := newTestService(t, repo)

	material, err := service.SignIn(context.Background(), " Developer ", "a sufficiently long password", "127.0.0.1", "/ui/applications/new?copy=1")
	if err != nil {
		t.Fatalf("SignIn() error = %v", err)
	}
	if material.SessionToken == "" || material.CSRFToken == "" || material.RedirectTo != "/ui/applications/new?copy=1" {
		t.Fatalf("SignIn() result = %#v", material)
	}
	if repo.session == nil || repo.session.ExpiresAt.Sub(repo.session.CreatedAt) != SessionLifetime {
		t.Fatalf("persisted session = %#v", repo.session)
	}

	principal, err := service.Authenticate(context.Background(), material.SessionToken)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if principal.UserID != "user-1" || principal.Username != "developer" || principal.DisplayName != "Developer" {
		t.Fatalf("principal = %#v", principal)
	}
	if err := service.SignOut(context.Background(), principal.TokenHash); err != nil {
		t.Fatalf("SignOut() error = %v", err)
	}
	if _, err := service.Authenticate(context.Background(), material.SessionToken); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Authenticate() after logout error = %v", err)
	}
}

func TestSignInUsesGenericFailureAndRateLimit(t *testing.T) {
	repo := &memoryRepository{}
	service := newTestService(t, repo)
	for attempt := 1; attempt <= AccountLimit+1; attempt++ {
		_, err := service.SignIn(context.Background(), "missing", "a sufficiently long password", "127.0.0.1", "")
		if attempt <= AccountLimit && !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("attempt %d error = %v", attempt, err)
		}
		if attempt == AccountLimit+1 {
			var limited *RateLimitedError
			if !errors.As(err, &limited) || limited.RetryAfter != AttemptWindow {
				t.Fatalf("attempt %d error = %v", attempt, err)
			}
		}
	}
	if repo.session != nil {
		t.Fatal("invalid credentials created a session")
	}
}

func TestLocalUserOperationsValidateAndNormalizeInput(t *testing.T) {
	repo := &memoryRepository{credential: &CredentialRecord{Username: "developer"}}
	service := newTestService(t, repo)
	if err := service.CreateLocalUser(context.Background(), " Developer ", " Developer One ", "a sufficiently long password"); err != nil {
		t.Fatalf("CreateLocalUser() error = %v", err)
	}
	if repo.created.Username != "developer" || repo.created.DisplayName != "Developer One" ||
		!service.Hasher.Verify("a sufficiently long password", repo.created.PasswordHash) {
		t.Fatalf("created credential = %#v", repo.created)
	}
	if err := service.ResetLocalPassword(context.Background(), " Developer ", "a different long password"); err != nil {
		t.Fatalf("ResetLocalPassword() error = %v", err)
	}
	if !service.Hasher.Verify("a different long password", repo.resetHash) {
		t.Fatal("reset password was not hashed")
	}
	if err := service.SetLocalUserStatus(context.Background(), " Developer ", "disabled"); err != nil {
		t.Fatalf("SetLocalUserStatus() error = %v", err)
	}
	if repo.status != "DISABLED" {
		t.Fatalf("status = %q", repo.status)
	}
	if err := service.CreateLocalUser(context.Background(), "bad name", "Developer", "a sufficiently long password"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("invalid username error = %v", err)
	}
}

func TestSafeReturnTo(t *testing.T) {
	tests := map[string]string{
		"":                              "/ui/applications",
		"/ui/applications/new?copy=1":   "/ui/applications/new?copy=1",
		"https://attacker.example/path": "/ui/applications",
		"//attacker.example/path":       "/ui/applications",
		"/%2f%2fattacker.example/path":  "/ui/applications",
		"/\\attacker.example/path":      "/ui/applications",
		"/ui/login?return_to=/":         "/ui/applications",
		"/api/auth/logout":              "/ui/applications",
		"/ui/applications\r\nX: y":      "/ui/applications",
	}
	for input, want := range tests {
		t.Run(fmt.Sprintf("%q", input), func(t *testing.T) {
			if got := SafeReturnTo(input); got != want {
				t.Fatalf("SafeReturnTo(%q) = %q, want %q", input, got, want)
			}
		})
	}
}
