package apicontext

import (
	"context"

	"github.com/augno/api/services/auth-service/pkg/types"
)

const (
	AuthIdentityKey contextKey = "identity"
)

func GetIdentityFromContext(ctx context.Context) (*types.Identity, bool) {
	identity, ok := ctx.Value(AuthIdentityKey).(*types.Identity)
	return identity, ok
}
