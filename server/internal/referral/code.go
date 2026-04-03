package referral

import (
	"crypto/rand"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

func GenerateCode(length int) (string, error) {
	if length <= 0 {
		return "", errors.New("invalid referral code length")
	}

	const alphabet = "abcdefghijklmnopqrstuvwxyz"
	const maxUnbiasedByte = byte(26 * 9)

	code := make([]byte, length)
	randomBytes := make([]byte, length*2)
	filled := 0

	for filled < length {
		if _, err := rand.Read(randomBytes); err != nil {
			return "", err
		}
		for _, b := range randomBytes {
			if b >= maxUnbiasedByte {
				continue
			}
			code[filled] = alphabet[b%26]
			filled++
			if filled == length {
				break
			}
		}
	}

	return string(code), nil
}

func IsCodeUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return false
	}
	return strings.Contains(strings.ToLower(pgErr.ConstraintName), "referral_code")
}
