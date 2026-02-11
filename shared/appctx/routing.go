package appctx

import "context"

const (
	routePatternKey   contextKey = "route_pattern"
	allowedMethodsKey contextKey = "allowed_methods"
	pathParamsKey     contextKey = "path_params"
)

// WithRoutePattern returns a child context carrying the matched route pattern.
func WithRoutePattern(ctx context.Context, pattern string) context.Context {
	return context.WithValue(ctx, routePatternKey, pattern)
}

// GetRoutePattern retrieves the route pattern from the context.
func GetRoutePattern(ctx context.Context) (string, bool) {
	val, ok := ctx.Value(routePatternKey).(string)
	return val, ok
}

// WithAllowedMethods returns a child context carrying the allowed HTTP methods.
func WithAllowedMethods(ctx context.Context, methods []string) context.Context {
	return context.WithValue(ctx, allowedMethodsKey, methods)
}

// GetAllowedMethods retrieves the allowed methods from the context.
func GetAllowedMethods(ctx context.Context) ([]string, bool) {
	methods, ok := ctx.Value(allowedMethodsKey).([]string)
	return methods, ok
}

// WithPathParams returns a child context carrying the path parameters.
func WithPathParams(ctx context.Context, params map[string]string) context.Context {
	return context.WithValue(ctx, pathParamsKey, params)
}

// GetPathParams retrieves the path parameters from the context.
func GetPathParams(ctx context.Context) (map[string]string, bool) {
	params, ok := ctx.Value(pathParamsKey).(map[string]string)
	return params, ok
}
