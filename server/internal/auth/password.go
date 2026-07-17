package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

const (
	passwordAlgorithm   = "argon2id"
	passwordVersion     = 19
	passwordMemoryKiB   = 64 * 1024
	passwordIterations  = 3
	passwordParallelism = 4
	passwordKeyLength   = 32
	passwordSaltLength  = 16
	passwordMinLength   = 8
	passwordMaxLength   = 128
)

var (
	ErrInvalidPassword       = errors.New("invalid password")
	ErrInvalidPasswordHash   = errors.New("invalid password hash")
	ErrPasswordPepperMissing = errors.New("password pepper is not configured")
)

type PepperSet struct {
	CurrentID  string
	Current    []byte
	PreviousID string
	Previous   []byte
}

type passwordParameters struct {
	MemoryKiB   uint32
	Iterations  uint32
	Parallelism uint8
	PepperID    string
}

func HashPassword(password string, peppers PepperSet) (string, error) {
	if err := ValidatePassword(password); err != nil {
		return "", err
	}
	if err := ValidateCurrentPepper(peppers); err != nil {
		return "", err
	}

	salt := make([]byte, passwordSaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}

	params := passwordParameters{
		MemoryKiB:   passwordMemoryKiB,
		Iterations:  passwordIterations,
		Parallelism: passwordParallelism,
		PepperID:    strings.TrimSpace(peppers.CurrentID),
	}
	hash := deriveArgon2id(password, peppers.Current, salt, params)

	return fmt.Sprintf(
		"$%s$v=%d$m=%d,t=%d,p=%d,pepper=%s$%s$%s",
		passwordAlgorithm,
		passwordVersion,
		params.MemoryKiB,
		params.Iterations,
		params.Parallelism,
		params.PepperID,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

// ValidateCurrentPepper validates the deployment-owned password pepper before
// a one-time account token is consumed.
func ValidateCurrentPepper(peppers PepperSet) error {
	if !validPepperID(peppers.CurrentID) || len(peppers.Current) == 0 {
		return ErrPasswordPepperMissing
	}
	return nil
}

func VerifyPassword(password, encoded string, peppers PepperSet) (bool, bool, error) {
	if err := ValidatePassword(password); err != nil {
		return false, false, err
	}

	params, salt, expectedHash, err := parsePasswordHash(encoded)
	if err != nil {
		return false, false, err
	}
	pepper, currentPepper, err := peppers.lookup(params.PepperID)
	if err != nil {
		return false, false, err
	}

	actualHash := deriveArgon2id(password, pepper, salt, params)
	if subtle.ConstantTimeCompare(expectedHash, actualHash) != 1 {
		return false, false, nil
	}

	needsRehash := !currentPepper || params.MemoryKiB != passwordMemoryKiB || params.Iterations != passwordIterations || params.Parallelism != passwordParallelism || len(expectedHash) != passwordKeyLength
	return true, needsRehash, nil
}

func (p PepperSet) lookup(id string) ([]byte, bool, error) {
	if id == strings.TrimSpace(p.CurrentID) && len(p.Current) > 0 {
		return p.Current, true, nil
	}
	if id == strings.TrimSpace(p.PreviousID) && len(p.Previous) > 0 {
		return p.Previous, false, nil
	}
	return nil, false, ErrPasswordPepperMissing
}

// ValidatePassword performs the inexpensive password-format validation used
// before a one-time token is consumed or an Argon2id operation is started.
func ValidatePassword(password string) error {
	if !utf8.ValidString(password) {
		return ErrInvalidPassword
	}
	length := utf8.RuneCountInString(password)
	if length < passwordMinLength || length > passwordMaxLength {
		return ErrInvalidPassword
	}
	return nil
}

func deriveArgon2id(password string, pepper, salt []byte, params passwordParameters) []byte {
	mac := hmac.New(sha256.New, pepper)
	_, _ = mac.Write([]byte(password))
	prehashedPassword := mac.Sum(nil)
	return argon2.IDKey(prehashedPassword, salt, params.Iterations, params.MemoryKiB, params.Parallelism, passwordKeyLength)
}

func parsePasswordHash(encoded string) (passwordParameters, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != passwordAlgorithm || parts[2] != "v=19" {
		return passwordParameters{}, nil, nil, ErrInvalidPasswordHash
	}

	params, err := parsePasswordParameters(parts[3])
	if err != nil {
		return passwordParameters{}, nil, nil, err
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) != passwordSaltLength {
		return passwordParameters{}, nil, nil, ErrInvalidPasswordHash
	}
	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(expectedHash) != passwordKeyLength {
		return passwordParameters{}, nil, nil, ErrInvalidPasswordHash
	}

	return params, salt, expectedHash, nil
}

func parsePasswordParameters(raw string) (passwordParameters, error) {
	values := make(map[string]string, 4)
	for _, entry := range strings.Split(raw, ",") {
		key, value, ok := strings.Cut(entry, "=")
		if !ok || key == "" || value == "" {
			return passwordParameters{}, ErrInvalidPasswordHash
		}
		if _, exists := values[key]; exists {
			return passwordParameters{}, ErrInvalidPasswordHash
		}
		values[key] = value
	}
	if len(values) != 4 {
		return passwordParameters{}, ErrInvalidPasswordHash
	}

	memory, err := parseUint32Parameter(values, "m")
	if err != nil || memory < 8*1024 || memory > 256*1024 {
		return passwordParameters{}, ErrInvalidPasswordHash
	}
	iterations, err := parseUint32Parameter(values, "t")
	if err != nil || iterations == 0 || iterations > 10 {
		return passwordParameters{}, ErrInvalidPasswordHash
	}
	parallelism, err := parseUint32Parameter(values, "p")
	if err != nil || parallelism == 0 || parallelism > 8 {
		return passwordParameters{}, ErrInvalidPasswordHash
	}
	pepperID := values["pepper"]
	if !validPepperID(pepperID) {
		return passwordParameters{}, ErrInvalidPasswordHash
	}

	return passwordParameters{
		MemoryKiB:   memory,
		Iterations:  iterations,
		Parallelism: uint8(parallelism),
		PepperID:    pepperID,
	}, nil
}

func validPepperID(raw string) bool {
	id := strings.TrimSpace(raw)
	if id == "" || len(id) > 32 {
		return false
	}
	for _, char := range id {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '-' || char == '_' || char == '.' {
			continue
		}
		return false
	}
	return true
}

func parseUint32Parameter(values map[string]string, key string) (uint32, error) {
	raw, ok := values[key]
	if !ok {
		return 0, ErrInvalidPasswordHash
	}
	parsed, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(parsed), nil
}
