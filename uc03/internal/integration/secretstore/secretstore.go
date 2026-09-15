// Package secretstore is the Secret Store / Secret Management Adapter
// abstraction and its encrypted-file implementation.
package secretstore

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Store keeps secret values outside the IDP database; callers persist only the
// opaque reference.
type Store interface {
	Put(ctx context.Context, name string, value []byte) (ref string, err error)
	Get(ctx context.Context, ref string) ([]byte, error)
	Delete(ctx context.Context, ref string) error
}

const refPrefix = "idpsecret://"

// EncryptedFile stores AES-256-GCM encrypted values under dir, one file per
// reference. The key comes from IDP_SECRET_KEY and never touches disk here.
type EncryptedFile struct {
	dir  string
	aead cipher.AEAD
}

var safeName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._/-]*$`)

func NewEncryptedFile(dir, key string) (*EncryptedFile, error) {
	if len(key) < 16 {
		return nil, errors.New("IDP_SECRET_KEY must be at least 16 characters")
	}
	sum := sha256.Sum256([]byte(key))
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return &EncryptedFile{dir: dir, aead: aead}, nil
}

func (s *EncryptedFile) path(ref string) (string, error) {
	name, ok := strings.CutPrefix(ref, refPrefix)
	if !ok || !safeName.MatchString(name) || strings.Contains(name, "..") {
		return "", fmt.Errorf("invalid secret reference %q", ref)
	}
	return filepath.Join(s.dir, filepath.FromSlash(name)+".enc"), nil
}

func (s *EncryptedFile) Put(_ context.Context, name string, value []byte) (string, error) {
	ref := refPrefix + name
	p, err := s.path(ref)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := s.aead.Seal(nonce, nonce, value, []byte(ref))
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return "", err
	}
	return ref, os.WriteFile(p, []byte(base64.StdEncoding.EncodeToString(sealed)), 0o600)
}

func (s *EncryptedFile) Get(_ context.Context, ref string) ([]byte, error) {
	p, err := s.path(ref)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("secret %s is not available: %w", ref, err)
	}
	sealed, err := base64.StdEncoding.DecodeString(string(raw))
	if err != nil || len(sealed) < s.aead.NonceSize() {
		return nil, fmt.Errorf("secret %s is corrupt", ref)
	}
	value, err := s.aead.Open(nil, sealed[:s.aead.NonceSize()], sealed[s.aead.NonceSize():], []byte(ref))
	if err != nil {
		return nil, fmt.Errorf("secret %s cannot be decrypted with the configured key", ref)
	}
	return value, nil
}

func (s *EncryptedFile) Delete(_ context.Context, ref string) error {
	p, err := s.path(ref)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// RandomValue returns a random hex string for generated secrets.
func RandomValue(bytes int) string {
	b := make([]byte, bytes)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
