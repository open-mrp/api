package password

import (
	"context"

	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/crypto"
	"github.com/open-mrp/apikit/tracing"
	"github.com/open-mrp/apikit/validate"
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
