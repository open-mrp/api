package appctx

import (
	"context"

	"github.com/augno/api/shared/version"
)

const apiVersionKey contextKey = "api_version"

// WithAPIVersion returns a child context carrying the given API version.
func WithAPIVersion(ctx context.Context, v version.APIVersion) context.Context {
	return context.WithValue(ctx, apiVersionKey, v)
}

// GetAPIVersionFromContext retrieves the API version from the context.
func GetAPIVersionFromContext(ctx context.Context) (version.APIVersion, bool) {
	v, ok := ctx.Value(apiVersionKey).(version.APIVersion)
	return v, ok
}
