// Package omrpctx holds the request-context values that are OpenMRP's own: the platform mode, the external host a
// portal request arrived on, the system identity background work acts as, and the request-log body caps. The
// generic request context is apikit's appctx.
package omrpctx

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/appctx"
)

type contextKey string

const (
	platformKey     contextKey = "platform"
	externalHostKey contextKey = "external_host"
)

// WithPlatform returns a child context carrying the platform mode the request runs in.
func WithPlatform(ctx context.Context, platform constants.PlatformMode) context.Context {
	return context.WithValue(ctx, platformKey, platform)
}

// GetPlatformFromContext returns the platform mode ctx carries.
func GetPlatformFromContext(ctx context.Context) (constants.PlatformMode, bool) {
	v, ok := ctx.Value(platformKey).(constants.PlatformMode)
	return v, ok
}

// WithExternalHost returns a child context carrying the host a request arrived on from outside, such as a portal domain.
func WithExternalHost(ctx context.Context, host string) context.Context {
	return context.WithValue(ctx, externalHostKey, host)
}

// GetExternalHost returns the external host ctx carries, if any.
func GetExternalHost(ctx context.Context) (string, bool) {
	host, ok := ctx.Value(externalHostKey).(string)
	return host, ok && host != ""
}

// WithSystemIdentity returns a child context that attributes the writes made under it to the background process source, acting on targetAccountID. It replaces any identity ctx carried. See types.NewSystemIdentity: the identity authorizes nothing.
func WithSystemIdentity(ctx context.Context, source constants.SystemActor, targetAccountID string) context.Context {
	return appctx.WithIdentity(ctx, types.NewSystemIdentity(source, targetAccountID))
}

const (
	// MaxLoggedBodyBytes caps a request or response body kept on a request log. Bodies go to the
	// payloads bucket, so the cap only bounds what the gateway holds in memory per request; a larger
	// body is recorded as a truncation marker. It matches the 8 MiB request body limit.
	MaxLoggedBodyBytes = 8 << 20
	// MaxMessageBodyBytes caps a body carried on an outbox message (a log whose upload failed, or an
	// error alert), which would otherwise bloat message_outbox.
	MaxMessageBodyBytes = 256 << 10
)

// CapBody returns body, or a truncation marker recording its size when it is longer than limit.
func CapBody(body *string, limit int) *string {
	if body == nil || len(*body) <= limit {
		return body
	}
	marker := fmt.Sprintf(`{"_truncated":true,"_original_size":%d}`, len(*body))
	return &marker
}
