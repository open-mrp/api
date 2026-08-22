package password

import (
	"context"

	"github.com/open-mrp/api/shared/crypto"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
	"github.com/open-mrp/api/shared/validate"
)

var passwordTracer = tracing.GetTracer("auth-service.password")

// CompareHashAndPassword compares a plaintext password against a hashed password.
func CompareHashAndPassword(ctx context.Context, hash, plaintextPassword string) (bool, *apierror.APIError) {
	_, span := passwordTracer.Start(ctx, "password.compare_hash_and_password")
	defer span.End()

	if len(plaintextPassword) > validate.PasswordMaxLength {
		return false, tracing.Trace(span, apierror.NewInvariantViolationError("password length is too long when comparing hash and password"))
	}

	match, err := crypto.CompareBcryptHash(hash, plaintextPassword)
	if err != nil {
		return false, tracing.Trace(span, apierror.NewInternalError(err, "Failed to compare hash and password."))
	}

	return match, nil
}
