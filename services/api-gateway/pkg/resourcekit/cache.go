package resourcekit

import (
	"context"
	"sync"

	"github.com/augno/api/shared/constants"
)

// loadCache is a per-request memo of already-loaded resources, keyed by
// (ObjectType, ID). It exists so that cyclic include paths (e.g.
// child_accounts.parent_account) never re-fetch a resource the request has
// already seen.
type loadCache struct {
	mu   sync.Mutex
	data map[constants.ObjectType]map[string]any
}

type cacheCtxKey struct{}

// WithLoadCache attaches a fresh load-cache to ctx if one isn't already
// present. Subsequent ResolveIncludes calls on this context will share the
// cache. Idempotent — repeated calls keep the first cache.
func WithLoadCache(ctx context.Context) context.Context {
	if _, ok := ctx.Value(cacheCtxKey{}).(*loadCache); ok {
		return ctx
	}
	return context.WithValue(ctx, cacheCtxKey{}, &loadCache{
		data: map[constants.ObjectType]map[string]any{},
	})
}

// getOrCreateCache returns the cache attached to ctx, or a fresh detached
// one when none was attached. Returning a detached cache means callers
// don't have to special-case missing context — the resolver still works,
// it just won't share state across calls.
func getOrCreateCache(ctx context.Context) *loadCache {
	if c, ok := ctx.Value(cacheCtxKey{}).(*loadCache); ok {
		return c
	}
	return &loadCache{data: map[constants.ObjectType]map[string]any{}}
}

func (c *loadCache) get(ot constants.ObjectType, id string) (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if m, ok := c.data[ot]; ok {
		v, ok := m[id]
		return v, ok
	}
	return nil, false
}

func (c *loadCache) set(ot constants.ObjectType, id string, v any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.data[ot] == nil {
		c.data[ot] = map[string]any{}
	}
	c.data[ot][id] = v
}

// PreheatCache inserts a resource into the request-scoped resolver cache so
// that a subsequent ResolveIncludes call finds it without invoking the
// resource's Load function. Use this when the service method already has the
// resource data (e.g. denormalized from a parent proto) and a loader call
// would be redundant.
func PreheatCache(ctx context.Context, ot constants.ObjectType, id string, v any) {
	getOrCreateCache(ctx).set(ot, id, v)
}
