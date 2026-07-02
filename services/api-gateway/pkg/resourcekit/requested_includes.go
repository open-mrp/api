package resourcekit

import "context"

type requestedIncludesCtxKey struct{}

// WithRequestedIncludes stores the flat list of include paths the client asked
// for on this request. The list is already validated against the endpoint's
// allowed include set and is flattened so that a nested request like
// "definition.config" yields both "definition" and "definition.config".
//
// Gateway service handlers read it via FilterIncludes to forward only the
// includes the client actually requested to the backend, instead of always
// over-fetching a fixed set.
func WithRequestedIncludes(ctx context.Context, includes []string) context.Context {
	return context.WithValue(ctx, requestedIncludesCtxKey{}, includes)
}

// RequestedIncludes returns the include paths the client requested, or nil when
// none were requested (or the endpoint does not support includes).
func RequestedIncludes(ctx context.Context) []string {
	if v, ok := ctx.Value(requestedIncludesCtxKey{}).([]string); ok {
		return v
	}
	return nil
}

// RequestedIncludeSet returns the client's requested include paths as a set for cheap membership
// checks (e.g. gating per-field name hydration). Nil when nothing was requested.
func RequestedIncludeSet(ctx context.Context) map[string]bool {
	requested := RequestedIncludes(ctx)
	if len(requested) == 0 {
		return nil
	}
	set := make(map[string]bool, len(requested))
	for _, r := range requested {
		set[r] = true
	}
	return set
}

// FilterIncludes returns the subset of supported includes that the client
// requested, preserving the order of supported. supported is the set a given
// backend call understands (historically passed verbatim, which over-fetched);
// the result is what should now be forwarded so the backend only enriches data
// the client asked for. Returns nil when nothing was requested.
func FilterIncludes(ctx context.Context, supported ...string) []string {
	requested := RequestedIncludes(ctx)
	if len(requested) == 0 || len(supported) == 0 {
		return nil
	}
	wanted := make(map[string]struct{}, len(requested))
	for _, r := range requested {
		wanted[r] = struct{}{}
	}
	var out []string
	for _, s := range supported {
		if _, ok := wanted[s]; ok {
			out = append(out, s)
		}
	}
	return out
}
