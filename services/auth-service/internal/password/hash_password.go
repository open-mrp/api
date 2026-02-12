package password

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
	"github.com/augno/api/shared/validate"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes a plaintext password using bcrypt.
func HashPassword(ctx context.Context, plaintextPassword string) (string, *apierror.APIError) {
	_, span := passwordTracer.Start(ctx, "password.hash_password")
	defer span.End()

	if len(plaintextPassword) > validate.PasswordMaxLength {
		return "", tracing.Trace(span, apierror.NewInvariantViolationError("password length is too long"))
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(plaintextPassword), 12)
	if err != nil {
		return "", tracing.Trace(span, apierror.NewInternalError(err, "Failed to hash password."))
	}

	return string(hash), nil
}
