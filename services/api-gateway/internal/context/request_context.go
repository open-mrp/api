package apicontext

import (
	"context"

	"github.com/augno/api/services/api-gateway/internal/domain"
)

const (
	RequestLogKey   contextKey = "request_log"
	PathParamsKey   contextKey = "path_params"
	RoutePatternKey contextKey = "route_pattern"
)

func GetRequestLogFromContext(ctx context.Context) (*domain.RequestLog, bool) {
	requestInfo, ok := ctx.Value(RequestLogKey).(*domain.RequestLog)
	return requestInfo, ok
}

func WithRoutePattern(ctx context.Context, pattern string) context.Context {
	return context.WithValue(ctx, RoutePatternKey, pattern)
}

func GetRoutePattern(ctx context.Context) (string, bool) {
	val, ok := ctx.Value(RoutePatternKey).(string)
	return val, ok
}
