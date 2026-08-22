package resourcekit

import (
	"context"
	"sync"

	"github.com/open-mrp/api/shared/constants"
)

// LoadMeta is a request-scoped store of loader-side metadata that does NOT
// belong on the public apiresource struct. Carriers attach their account_id
// here, the list of service_level IDs they own, "has more" flags for truncated
// sub-lists, etc. SubField closures read this metadata via the request ctx so
// that apiresource Go structs stay untouched.
//
// Keyed by (ObjectType, resource ID, metadata key). Concurrent-safe.
type LoadMeta struct {
	mu   sync.Mutex
	data map[constants.ObjectType]map[string]map[string]any
}

type metaCtxKey struct{}

// WithLoadMeta attaches a fresh LoadMeta to ctx if one isn't already present.
// Idempotent — repeated calls keep the first attachment. Mirrors WithLoadCache.
func WithLoadMeta(ctx context.Context) context.Context {
	if _, ok := ctx.Value(metaCtxKey{}).(*LoadMeta); ok {
		return ctx
	}
	return context.WithValue(ctx, metaCtxKey{}, &LoadMeta{
		data: map[constants.ObjectType]map[string]map[string]any{},
	})
}

// GetLoadMeta returns the LoadMeta attached to ctx, or a fresh detached one
// when none was attached. The detached fallback means callers don't have to
// guard against nil — they just lose cross-call state.
func GetLoadMeta(ctx context.Context) *LoadMeta {
	if m, ok := ctx.Value(metaCtxKey{}).(*LoadMeta); ok {
		return m
	}
	return &LoadMeta{data: map[constants.ObjectType]map[string]map[string]any{}}
}

// Set stores a metadata value for a (ObjectType, id, key) triple. Overwrites
// any previous value for the same triple.
func (m *LoadMeta) Set(ot constants.ObjectType, id, key string, value any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.data[ot] == nil {
		m.data[ot] = map[string]map[string]any{}
	}
	if m.data[ot][id] == nil {
		m.data[ot][id] = map[string]any{}
	}
	m.data[ot][id][key] = value
}

// Get returns the raw stored value and whether it was present.
func (m *LoadMeta) Get(ot constants.ObjectType, id, key string) (any, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if byID, ok := m.data[ot]; ok {
		if byKey, ok := byID[id]; ok {
			v, ok := byKey[key]
			return v, ok
		}
	}
	return nil, false
}

// GetString returns the stored value as a string. Missing key or wrong type
// both yield ("", false).
func (m *LoadMeta) GetString(ot constants.ObjectType, id, key string) (string, bool) {
	v, ok := m.Get(ot, id, key)
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// GetStrings returns the stored value as a []string. Missing key or wrong type
// both yield (nil, false).
func (m *LoadMeta) GetStrings(ot constants.ObjectType, id, key string) ([]string, bool) {
	v, ok := m.Get(ot, id, key)
	if !ok {
		return nil, false
	}
	s, ok := v.([]string)
	return s, ok
}

// GetBool returns the stored value as a bool. Missing key or wrong type both
// yield (false, false).
func (m *LoadMeta) GetBool(ot constants.ObjectType, id, key string) (bool, bool) {
	v, ok := m.Get(ot, id, key)
	if !ok {
		return false, false
	}
	b, ok := v.(bool)
	return b, ok
}
