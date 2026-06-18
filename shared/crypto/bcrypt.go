package crypto

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// HashBcrypt hashes a plaintext password using bcrypt with the default cost factor.
func HashBcrypt(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to generate bcrypt hash: %w", err)
	}
	return string(hash), nil
}

// CompareBcryptHash compares a plaintext password against a bcrypt hash. Returns (false, nil) when the password does not match, and (false, error) for unexpected errors (e.g. malformed hash).
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
