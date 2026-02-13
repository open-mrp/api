package crypto

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// HashBcrypt hashes a plaintext password using bcrypt with the given cost factor.
func HashBcrypt(password string, cost int) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CompareBcryptHash compares a plaintext password against a bcrypt hash.
// Returns (false, nil) when the password does not match.
// Returns (false, error) for unexpected errors (e.g. malformed hash).
func CompareBcryptHash(password, hash string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
