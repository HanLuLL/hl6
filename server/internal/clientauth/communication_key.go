package clientauth

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"strings"
)

const (
	CommunicationKeyHashConfigKey = "client_communication_key_hash"
	NativeSessionClaim            = "hl6_native_client"
	NativeSessionKeyHashClaim     = "hl6_client_key_hash"
)

// HashCommunicationKey returns the database representation of a client key.
// The raw key is only present at generation and APK build time.
func HashCommunicationKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return base64.RawStdEncoding.EncodeToString(hash[:])
}

// IsAuthorized verifies a presented key against the stored hash in constant time.
func IsAuthorized(presentedKey, storedHash string) bool {
	presentedKey = strings.TrimSpace(presentedKey)
	if presentedKey == "" || storedHash == "" {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(storedHash)
	if err != nil || len(expected) != sha256.Size {
		return false
	}
	actual := sha256.Sum256([]byte(presentedKey))
	return subtle.ConstantTimeCompare(actual[:], expected) == 1
}

func SameHash(first, second string) bool {
	if first == "" || second == "" || len(first) != len(second) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(first), []byte(second)) == 1
}
