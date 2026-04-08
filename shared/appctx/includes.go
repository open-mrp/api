package appctx

import "context"

const includesKey contextKey = "requested_includes"

// WithRequestedIncludes returns a child context carrying the requested include set.
func WithRequestedIncludes(ctx context.Context, includes map[string]bool) context.Context {
	return context.WithValue(ctx, includesKey, includes)
}

// GetRequestedIncludes retrieves the requested include set from the context.
// Returns nil when no includes were set (meaning: no enriched data requested).
func GetRequestedIncludes(ctx context.Context) map[string]bool {
	includes, _ := ctx.Value(includesKey).(map[string]bool)
	return includes
}

// IsIncludeRequested returns true when the given key was requested by the client.
// When no includes are in the context (nil), returns false — no enriched data is needed.
func IsIncludeRequested(ctx context.Context, key string) bool {
	includes := GetRequestedIncludes(ctx)
	if includes == nil {
		return false
	}
	return includes[key]
}

// GetRequestedIncludeKeys converts the include set to a string slice suitable for
// proto repeated string fields. Returns nil when no includes are in the context.
func GetRequestedIncludeKeys(ctx context.Context) []string {
	includes := GetRequestedIncludes(ctx)
	if includes == nil {
		return nil
	}
	keys := make([]string, 0, len(includes))
	for k := range includes {
		keys = append(keys, k)
	}
	return keys
}
