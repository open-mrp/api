package password

import (
	"context"

	"github.com/open-mrp/api/shared/crypto"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
	"github.com/open-mrp/api/shared/validate"
)

// HashPassword hashes a plaintext password using bcrypt.
func HashPassword(ctx context.Context, plaintextPassword string) (string, *apierror.APIError) {
	_, span := passwordTracer.Start(ctx, "password.hash_password")
	defer span.End()

	if len(plaintextPassword) > validate.PasswordMaxLength {
		return "", tracing.Trace(span, apierror.NewInvariantViolationError("password length is too long"))
	}

	hash, err := crypto.HashBcrypt(plaintextPassword)
	if err != nil {
		return "", tracing.Trace(span, apierror.NewInternalError(err, "Failed to hash password."))
	}

	return hash, nil
}
