package authentication

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	SessionLifetime = 8 * time.Hour
	IdleTimeout     = 30 * time.Minute
	AttemptWindow   = 15 * time.Minute
	AccountLimit    = 5
	SourceLimit     = 20
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidInput       = errors.New("invalid input")
	ErrUserNotFound       = errors.New("local user not found")
	ErrUsernameTaken      = errors.New("username already exists")
	usernamePattern       = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{2,63}$`)
)

type RateLimitedError struct{ RetryAfter time.Duration }

func (e *RateLimitedError) Error() string { return "login rate limit exceeded" }

type CredentialRecord struct {
	UserID       string
	Username     string
	DisplayName  string
	Status       string
	PasswordHash string
}

type SessionRecord struct {
	SessionID  string
	UserID     string
	TokenHash  []byte
	CSRFHash   []byte
	CreatedAt  time.Time
	LastSeenAt time.Time
	ExpiresAt  time.Time
}

type AuthenticatedSession struct {
	SessionID   string
	UserID      string
	Username    string
	DisplayName string
	TokenHash   []byte
	CSRFHash    []byte
}

type Principal struct {
	UserID      string
	Username    string
	DisplayName string
}

func (s *AuthenticatedSession) Principal() Principal {
	return Principal{UserID: s.UserID, Username: s.Username, DisplayName: s.DisplayName}
}

type Repository interface {
	FindCredential(context.Context, string) (*CredentialRecord, error)
	BlockedUntil(context.Context, []byte, []byte, time.Time) (time.Time, error)
	RecordFailure(context.Context, string, []byte, int, time.Time, time.Duration) (time.Time, error)
	CreateSessionAndClearFailures(context.Context, SessionRecord, []byte) error
	AuthenticateSession(context.Context, []byte, time.Time, time.Time) (*AuthenticatedSession, error)
	RevokeSession(context.Context, []byte, time.Time) error
	CreateLocalUser(context.Context, string, string, string, time.Time) error
	ResetLocalPassword(context.Context, string, string, time.Time) error
	SetLocalUserStatus(context.Context, string, string, time.Time) error
}

type Service struct {
	Repo                Repository
	Hasher              PasswordHasher
	HMACKey             []byte
	Now                 func() time.Time
	AccountAttemptLimit int
	SourceAttemptLimit  int
	AttemptWindow       time.Duration
	dummyHash           string
}

type SessionMaterial struct {
	SessionToken string
	CSRFToken    string
	RedirectTo   string
}

func NewService(repo Repository, hmacKey []byte) (*Service, error) {
	if repo == nil || len(hmacKey) < 32 {
		return nil, errors.New("authentication repository and a 32-byte HMAC key are required")
	}
	hasher := PasswordHasher{}
	dummy, err := hasher.Hash("dummy-password-never-valid")
	if err != nil {
		return nil, err
	}
	return &Service{Repo: repo, Hasher: hasher, HMACKey: append([]byte(nil), hmacKey...), Now: time.Now,
		AccountAttemptLimit: AccountLimit, SourceAttemptLimit: SourceLimit, AttemptWindow: AttemptWindow,
		dummyHash: dummy}, nil
}

func NormalizeUsername(username string) string { return strings.ToLower(strings.TrimSpace(username)) }

func (s *Service) SignIn(ctx context.Context, username, password, source, returnTo string) (*SessionMaterial, error) {
	now := s.now()
	normalized := NormalizeUsername(username)
	accountKey := s.key("account", normalized)
	sourceKey := s.key("source", source)
	blocked, err := s.Repo.BlockedUntil(ctx, accountKey, sourceKey, now)
	if err != nil {
		return nil, err
	}
	if blocked.After(now) {
		return nil, &RateLimitedError{RetryAfter: blocked.Sub(now)}
	}

	record, err := s.Repo.FindCredential(ctx, normalized)
	if err != nil {
		return nil, err
	}
	encoded := s.dummyHash
	if record != nil {
		encoded = record.PasswordHash
	}
	valid := s.Hasher.Verify(password, encoded) && record != nil && record.Status == "ACTIVE"
	if !valid {
		_, accountErr := s.Repo.RecordFailure(ctx, "ACCOUNT", accountKey, s.AccountAttemptLimit, now, s.AttemptWindow)
		_, sourceErr := s.Repo.RecordFailure(ctx, "SOURCE", sourceKey, s.SourceAttemptLimit, now, s.AttemptWindow)
		if accountErr != nil {
			return nil, accountErr
		}
		if sourceErr != nil {
			return nil, sourceErr
		}
		return nil, ErrInvalidCredentials
	}

	rawToken, tokenHash, err := newToken()
	if err != nil {
		return nil, err
	}
	rawCSRF, csrfHash, err := newToken()
	if err != nil {
		return nil, err
	}
	session := SessionRecord{SessionID: uuid.NewString(), UserID: record.UserID, TokenHash: tokenHash,
		CSRFHash: csrfHash, CreatedAt: now, LastSeenAt: now, ExpiresAt: now.Add(SessionLifetime)}
	if err := s.Repo.CreateSessionAndClearFailures(ctx, session, accountKey); err != nil {
		return nil, err
	}
	return &SessionMaterial{SessionToken: rawToken, CSRFToken: rawCSRF, RedirectTo: SafeReturnTo(returnTo)}, nil
}

func (s *Service) Authenticate(ctx context.Context, rawToken string) (*AuthenticatedSession, error) {
	if rawToken == "" {
		return nil, ErrInvalidCredentials
	}
	now := s.now()
	hash := sha256.Sum256([]byte(rawToken))
	session, err := s.Repo.AuthenticateSession(ctx, hash[:], now, now.Add(-IdleTimeout))
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, ErrInvalidCredentials
	}
	session.TokenHash = append([]byte(nil), hash[:]...)
	return session, nil
}

func (s *Service) SignOut(ctx context.Context, tokenHash []byte) error {
	return s.Repo.RevokeSession(ctx, tokenHash, s.now())
}

func (s *Service) CreateLocalUser(ctx context.Context, username, displayName, password string) error {
	normalized, display, err := validateUser(username, displayName, password)
	if err != nil {
		return err
	}
	hash, err := s.Hasher.Hash(password)
	if err != nil {
		return err
	}
	return s.Repo.CreateLocalUser(ctx, normalized, display, hash, s.now())
}

func (s *Service) ResetLocalPassword(ctx context.Context, username, password string) error {
	if err := ValidatePassword(password); err != nil {
		return err
	}
	hash, err := s.Hasher.Hash(password)
	if err != nil {
		return err
	}
	return s.Repo.ResetLocalPassword(ctx, NormalizeUsername(username), hash, s.now())
}

func (s *Service) SetLocalUserStatus(ctx context.Context, username, status string) error {
	status = strings.ToUpper(strings.TrimSpace(status))
	if status != "ACTIVE" && status != "DISABLED" {
		return fmt.Errorf("%w: status must be ACTIVE or DISABLED", ErrInvalidInput)
	}
	return s.Repo.SetLocalUserStatus(ctx, NormalizeUsername(username), status, s.now())
}

func ValidatePassword(password string) error {
	n := utf8.RuneCountInString(password)
	if n < 15 || n > 128 {
		return fmt.Errorf("%w: password must contain 15 to 128 characters", ErrInvalidInput)
	}
	common := map[string]bool{
		"passwordpassword": true, "password123456": true, "qwertyuiop12345": true,
		"administrator123": true, "letmeinletmein": true,
	}
	if common[strings.ToLower(password)] {
		return fmt.Errorf("%w: password is too common", ErrInvalidInput)
	}
	return nil
}

func SafeReturnTo(value string) string {
	const fallback = "/ui/applications"
	if value == "" || strings.Contains(value, "\\") {
		return fallback
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return fallback
		}
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || !strings.HasPrefix(parsed.Path, "/") ||
		strings.HasPrefix(parsed.Path, "//") || strings.HasPrefix(parsed.Path, "/ui/login") ||
		strings.HasPrefix(parsed.Path, "/api/auth/") {
		return fallback
	}
	return value
}

func validateUser(username, displayName, password string) (string, string, error) {
	normalized := NormalizeUsername(username)
	if !usernamePattern.MatchString(normalized) {
		return "", "", fmt.Errorf("%w: username must match %s", ErrInvalidInput, usernamePattern.String())
	}
	display := strings.TrimSpace(displayName)
	if n := utf8.RuneCountInString(display); n < 1 || n > 255 {
		return "", "", fmt.Errorf("%w: display name must contain 1 to 255 characters", ErrInvalidInput)
	}
	for _, r := range display {
		if unicode.IsControl(r) {
			return "", "", fmt.Errorf("%w: display name must not contain control characters", ErrInvalidInput)
		}
	}
	if err := ValidatePassword(password); err != nil {
		return "", "", err
	}
	return normalized, display, nil
}

func (s *Service) key(namespace, value string) []byte {
	h := hmac.New(sha256.New, s.HMACKey)
	h.Write([]byte(namespace))
	h.Write([]byte{0})
	h.Write([]byte(value))
	return h.Sum(nil)
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func newToken() (string, []byte, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, fmt.Errorf("generate authentication token: %w", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(raw)
	hash := sha256.Sum256([]byte(encoded))
	return encoded, hash[:], nil
}
