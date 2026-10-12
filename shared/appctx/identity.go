package appctx

import (
	"context"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
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

// WithSystemIdentity returns a child context that attributes the writes made under it to the background process source, acting on targetAccountID. It replaces any identity ctx carried. See types.NewSystemIdentity: the identity authorizes nothing.
func WithSystemIdentity(ctx context.Context, source constants.SystemActor, targetAccountID string) context.Context {
	return WithIdentity(ctx, types.NewSystemIdentity(source, targetAccountID))
}
