package appctx

import (
	"context"

	"github.com/open-mrp/api/shared/constants"
)

const platformKey contextKey = "platform"

// WithPlatform returns a child context carrying the given platform mode.
func WithPlatform(ctx context.Context, platform constants.PlatformMode) context.Context {
	return context.WithValue(ctx, platformKey, platform)
}

// GetPlatformFromContext retrieves the platform mode from the context.
func GetPlatformFromContext(ctx context.Context) (constants.PlatformMode, bool) {
	v, ok := ctx.Value(platformKey).(constants.PlatformMode)
	return v, ok
}
