package password

import (
	"context"
	"errors"

	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/tracing"

	"golang.org/x/crypto/bcrypt"
)

var passwordTracer = tracing.GetTracer("auth-service.password")

// CompareHashAndPassword compares a plaintext password against a hashed password.
func CompareHashAndPassword(ctx context.Context, plaintextPassword, hash string) (bool, *contracts.APIError) {
	_, span := passwordTracer.Start(ctx, "password.compare_hash_and_password")
	defer span.End()

	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plaintextPassword))
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil // The passwords do not match
		default:
			return false, tracing.Trace(span, contracts.NewInternalError(err, "Failed to compare hash and password.")) // An unexpected error occurred
		}
	}

	return true, nil
}
