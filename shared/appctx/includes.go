package appctx

import (
	"context"
	"strings"
)

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
//
// Nested include keys (dotted paths) implicitly pull in their ancestors so
// backends receive a complete request: include="definition.config" alone is
// expanded to ["definition", "definition.config"], ensuring the backend
// fetches the parent `definition` even when the caller only named the child.
func GetRequestedIncludeKeys(ctx context.Context) []string {
	includes := GetRequestedIncludes(ctx)
	if includes == nil {
		return nil
	}
	expanded := make(map[string]bool, len(includes))
	for k := range includes {
		expanded[k] = true
		for _, ancestor := range ancestorIncludeKeys(k) {
			expanded[ancestor] = true
		}
	}
	keys := make([]string, 0, len(expanded))
	for k := range expanded {
		keys = append(keys, k)
	}
	return keys
}

// ancestorIncludeKeys returns every dot-path prefix of key, excluding the key
// itself. For "definition.config.model" it yields ["definition",
// "definition.config"]. For a leaf key like "actions" it returns nil.
func ancestorIncludeKeys(key string) []string {
	if !strings.Contains(key, ".") {
		return nil
	}
	parts := strings.Split(key, ".")
	out := make([]string, 0, len(parts)-1)
	for i := 1; i < len(parts); i++ {
		out = append(out, strings.Join(parts[:i], "."))
	}
	return out
}
