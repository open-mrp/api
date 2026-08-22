package appctx

import (
	"context"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
)

const identityKey contextKey = "identity"

// WithIdentity returns a child context carrying the given identity.
func WithIdentity(ctx context.Context, identity *types.Identity) context.Context {
	return context.WithValue(ctx, identityKey, identity)
}

// GetIdentityFromContext retrieves the identity from the context.
func GetIdentityFromContext(ctx context.Context) (*types.Identity, bool) {
	identity, ok := ctx.Value(identityKey).(*types.Identity)
	return identity, ok
}
