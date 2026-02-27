package crypto

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// HashBcrypt hashes a plaintext password using bcrypt with the given cost factor.
func HashBcrypt(password string, cost int) (string, error) {
	if cost == 0 {
		cost = bcrypt.DefaultCost
	}

	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		return "", fmt.Errorf("invalid cost: %d", cost)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", fmt.Errorf("failed to generate bcrypt hash: %w", err)
	}
	return string(hash), nil
}

// CompareBcryptHash compares a plaintext password against a bcrypt hash.
// Returns (false, nil) when the password does not match.
// Returns (false, error) for unexpected errors (e.g. malformed hash).
func CompareBcryptHash(hash, password string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return false, nil
		}
		return false, fmt.Errorf("failed to compare bcrypt hash: %w", err)
	}
	return true, nil
}
