package apicontext

import (
	"context"
	"net/http"

	"github.com/augno/api/shared/constants"
)

const (
	ResponseWriterCtxKey contextKey = "response_writer"
	PlatformCtxKey       contextKey = "platform"
)

func WithResponseWriter(ctx context.Context, w http.ResponseWriter) context.Context {
	return context.WithValue(ctx, ResponseWriterCtxKey, w)
}

func GetResponseWriterFromContext(ctx context.Context) (http.ResponseWriter, bool) {
	w, ok := ctx.Value(ResponseWriterCtxKey).(http.ResponseWriter)
	return w, ok
}

func WithPlatform(ctx context.Context, platform constants.PlatformMode) context.Context {
	return context.WithValue(ctx, PlatformCtxKey, platform)
}

func GetPlatformFromContext(ctx context.Context) (constants.PlatformMode, bool) {
	v, ok := ctx.Value(PlatformCtxKey).(constants.PlatformMode)
	return v, ok
}
