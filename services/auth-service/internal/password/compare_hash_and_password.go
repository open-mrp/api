package password

import (
	"context"

	"github.com/augno/api/shared/crypto"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
	"github.com/augno/api/shared/validate"
)

var passwordTracer = tracing.GetTracer("auth-service.password")

// CompareHashAndPassword compares a plaintext password against a hashed password.
//
//  1. Compares the plaintext password against the hashed password.
//  2. Returns true if the passwords match, false otherwise.
func CompareHashAndPassword(ctx context.Context, plaintextPassword, hash string) (bool, *apierror.APIError) {
	_, span := passwordTracer.Start(ctx, "password.compare_hash_and_password")
	defer span.End()

	if len(plaintextPassword) > validate.PasswordMaxLength {
		return false, tracing.Trace(span, apierror.NewInvariantViolationError("password length is too long when comparing hash and password"))
	}

	match, err := crypto.CompareBcryptHash(plaintextPassword, hash)
	if err != nil {
		return false, tracing.Trace(span, apierror.NewInternalError(err, "Failed to compare hash and password."))
	}

	return match, nil
}
