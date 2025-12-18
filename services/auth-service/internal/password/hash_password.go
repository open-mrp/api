package password

import (
	"context"

	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/tracing"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes a plaintext password using bcrypt.
func HashPassword(ctx context.Context, plaintextPassword string) (string, *contracts.APIError) {
	_, span := passwordTracer.Start(ctx, "password.hash_password")
	defer span.End()

	hash, err := bcrypt.GenerateFromPassword([]byte(plaintextPassword), 12)
	if err != nil {
		return "", tracing.Trace(span, contracts.NewInternalError(err, "Failed to hash password."))
	}

	return string(hash), nil
}
