package password

import (
	"context"
	"errors"

	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
	"github.com/augno/api/shared/validate"

	"golang.org/x/crypto/bcrypt"
)

var passwordTracer = tracing.GetTracer("auth-service.password")

// CompareHashAndPassword compares a plaintext password against a hashed password.
func CompareHashAndPassword(ctx context.Context, plaintextPassword, hash string) (bool, *apierror.APIError) {
	_, span := passwordTracer.Start(ctx, "password.compare_hash_and_password")
	defer span.End()

	if len(plaintextPassword) > validate.PasswordMaxLength {
		return false, tracing.Trace(span, apierror.NewInvariantViolationError("password length is too long when comparing hash and password"))
	}

	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plaintextPassword))
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil // The passwords do not match
		default:
			return false, tracing.Trace(span, apierror.NewInternalError(err, "Failed to compare hash and password.")) // An unexpected error occurred
		}
	}

	return true, nil
}
