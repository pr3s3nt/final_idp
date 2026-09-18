package authentication

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argonMemory      = 19 * 1024
	argonIterations  = 2
	argonParallelism = 1
	argonSaltLength  = 16
	argonKeyLength   = 32
)

type PasswordHasher struct{}

func (PasswordHasher) Hash(password string) (string, error) {
	salt := make([]byte, argonSaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, argonIterations, argonMemory, argonParallelism, argonKeyLength)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, argonMemory,
		argonIterations, argonParallelism, base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key)), nil
}

func (PasswordHasher) Verify(password, encoded string) bool {
	params, salt, expected, err := parseEncodedHash(encoded)
	if err != nil {
		return false
	}
	actual := argon2.IDKey([]byte(password), salt, params.iterations, params.memory, params.parallelism, uint32(len(expected)))
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

func (PasswordHasher) NeedsRehash(encoded string) bool {
	params, salt, key, err := parseEncodedHash(encoded)
	return err != nil || params.memory < argonMemory || params.iterations < argonIterations ||
		params.parallelism < argonParallelism || len(salt) < argonSaltLength || len(key) < argonKeyLength
}

type hashParams struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
}

func parseEncodedHash(encoded string) (hashParams, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return hashParams{}, nil, nil, errors.New("invalid encoded hash")
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return hashParams{}, nil, nil, errors.New("unsupported argon2 version")
	}
	var p hashParams
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.memory, &p.iterations, &p.parallelism); err != nil ||
		p.memory == 0 || p.iterations == 0 || p.parallelism == 0 {
		return hashParams{}, nil, nil, errors.New("invalid argon2 parameters")
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) < 8 {
		return hashParams{}, nil, nil, errors.New("invalid argon2 salt")
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(key) < 16 {
		return hashParams{}, nil, nil, errors.New("invalid argon2 key")
	}
	return p, salt, key, nil
}
