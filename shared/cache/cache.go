package cache

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	apierror "github.com/open-mrp/api/shared/errors"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/singleflight"
)

const (
	defaultLoadTimeout   = 2 * time.Minute
	defaultMaxValueBytes = 4 << 20
	generationKeyPrefix  = "gen:"

	// generationTTL is how long a scope's generation marker lives. When it lapses the scope's entries are orphaned, costing misses but never staleness.
	generationTTL = 24 * time.Hour
)

// Config configures a Cache.
type Config struct {
	// Name (required) namespaces the cache's keys and labels its trace events, e.g. "auth.role_permissions".
	Name string

	// Store (optional; default: nil) backs the cache. Nil disables caching: every call runs the loader.
	Store Store

	// TTL (required) is how long an entry lives when its Key sets no TTL. It bounds staleness for changes no invalidation reaches.
	TTL time.Duration

	// LoadTimeout (optional; default: 2m) bounds a shared load. Loads outlive the caller that started them so concurrent waiters and the cache still get the result.
	LoadTimeout time.Duration

	// MaxValueBytes (optional; default: 4 MiB) is the largest encoded value stored; larger results are returned but not cached, so one huge report cannot evict everything else.
	MaxValueBytes int
}

// WithDefaults returns a copy of the config with unset fields filled. It is safe to call on a nil receiver.
func (c *Config) WithDefaults() *Config {
	if c == nil {
		c = &Config{}
	}
	out := *c
	if out.LoadTimeout == 0 {
		out.LoadTimeout = defaultLoadTimeout
	}
	if out.MaxValueBytes == 0 {
		out.MaxValueBytes = defaultMaxValueBytes
	}
	return &out
}

func (c *Config) validate() error {
	if c.Name == "" {
		return fmt.Errorf("cache: name is required")
	}
	if c.TTL <= 0 {
		return fmt.Errorf("cache %s: TTL must be positive", c.Name)
	}
	if c.LoadTimeout <= 0 {
		return fmt.Errorf("cache %s: load timeout must be positive", c.Name)
	}
	if c.MaxValueBytes <= 0 {
		return fmt.Errorf("cache %s: max value bytes must be positive", c.Name)
	}
	return nil
}

// Key addresses one cache entry.
type Key struct {
	// Scopes (optional) are the invalidation groups the entry belongs to, e.g. "account:acct_123"; invalidating any of them orphans it. None means the entry expires only by TTL.
	Scopes []string

	// ID (required) identifies the entry within its scopes.
	ID string

	// TTL (optional; default: Config.TTL) overrides the entry's lifetime.
	TTL time.Duration
}

// Cache is a typed read-through cache. Values round-trip through JSON, so every hit is a private copy the caller may mutate.
type Cache[T any] struct {
	cfg     Config
	ttlFunc func(T) time.Duration
	group   singleflight.Group
}

// Option customises a Cache.
type Option[T any] func(*Cache[T])

// WithTTLFunc picks an entry's lifetime from its loaded value, e.g. a shorter one for a negative lookup. A non-positive result falls back to Key.TTL, then Config.TTL.
func WithTTLFunc[T any](fn func(T) time.Duration) Option[T] {
	return func(c *Cache[T]) { c.ttlFunc = fn }
}

// New returns a Cache for cfg.
func New[T any](cfg *Config, opts ...Option[T]) (*Cache[T], error) {
	cfg = cfg.WithDefaults()
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	c := &Cache[T]{cfg: *cfg}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

type loadResult[T any] struct {
	value T
	err   *apierror.APIError
}

// GetOrLoad returns the cached value for key, or runs load, caches its result, and returns it. Concurrent misses for the same key in this process share one load. Loader errors are returned and never cached.
func (c *Cache[T]) GetOrLoad(ctx context.Context, key Key, load func(context.Context) (T, *apierror.APIError)) (T, *apierror.APIError) {
	if c.cfg.Store == nil {
		return load(ctx)
	}

	span := trace.SpanFromContext(ctx)

	storeKey, err := c.storeKey(ctx, key)
	if err != nil {
		c.recordStoreError(span, "generation", err)
		return load(ctx)
	}

	if value, ok := c.get(ctx, span, storeKey); ok {
		c.recordLookup(span, true)
		return value, nil
	}
	c.recordLookup(span, false)

	ch := c.group.DoChan(storeKey, func() (any, error) {
		loadCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), c.cfg.LoadTimeout)
		defer cancel()

		value, apiErr := load(loadCtx)
		if apiErr == nil {
			c.set(loadCtx, span, storeKey, value, key.TTL)
		}
		return loadResult[T]{value: value, err: apiErr}, nil
	})

	select {
	case res := <-ch:
		out := res.Val.(loadResult[T])
		return out.value, out.err
	case <-ctx.Done():
		var zero T
		return zero, apierror.NewRequestTimeoutError("cache: caller gave up waiting on a shared load: " + ctx.Err().Error())
	}
}

func (c *Cache[T]) get(ctx context.Context, span trace.Span, storeKey string) (T, bool) {
	var value T
	raw, ok, err := c.cfg.Store.Get(ctx, storeKey)
	if err != nil {
		c.recordStoreError(span, "get", err)
		return value, false
	}
	if !ok {
		return value, false
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		c.recordStoreError(span, "decode", err)
		return value, false
	}
	return value, true
}

func (c *Cache[T]) set(ctx context.Context, span trace.Span, storeKey string, value T, ttl time.Duration) {
	raw, err := json.Marshal(value)
	if err != nil {
		c.recordStoreError(span, "encode", err)
		return
	}
	if len(raw) > c.cfg.MaxValueBytes {
		span.AddEvent("cache.skip_oversize", trace.WithAttributes(
			attribute.String("cache.name", c.cfg.Name),
			attribute.Int("cache.value_bytes", len(raw)),
		))
		return
	}
	if c.ttlFunc != nil {
		if fromValue := c.ttlFunc(value); fromValue > 0 {
			ttl = fromValue
		}
	}
	if ttl <= 0 {
		ttl = c.cfg.TTL
	}
	if err := c.cfg.Store.Set(ctx, storeKey, raw, ttl); err != nil {
		c.recordStoreError(span, "set", err)
	}
}

func (c *Cache[T]) storeKey(ctx context.Context, key Key) (string, error) {
	var b strings.Builder
	b.WriteString(c.cfg.Name)
	for _, scope := range key.Scopes {
		gen, err := generation(ctx, c.cfg.Store, scope)
		if err != nil {
			return "", err
		}
		b.WriteString("|" + scope + "@" + gen)
	}
	b.WriteString("|" + key.ID)
	return b.String(), nil
}

func (c *Cache[T]) recordLookup(span trace.Span, hit bool) {
	span.AddEvent("cache.lookup", trace.WithAttributes(
		attribute.String("cache.name", c.cfg.Name),
		attribute.Bool("cache.hit", hit),
	))
}

func (c *Cache[T]) recordStoreError(span trace.Span, op string, err error) {
	span.RecordError(err, trace.WithAttributes(
		attribute.String("cache.name", c.cfg.Name),
		attribute.String("cache.op", op),
	))
}

// Invalidate rotates the generation of each scope in store, orphaning every entry any Cache on that store wrote under it.
func Invalidate(ctx context.Context, store Store, scopes ...string) error {
	if store == nil {
		return nil
	}
	for _, scope := range scopes {
		if scope == "" {
			continue
		}
		if err := store.Set(ctx, generationKeyPrefix+scope, []byte(newGeneration()), generationTTL); err != nil {
			return fmt.Errorf("cache: invalidate %s: %w", scope, err)
		}
	}
	return nil
}

// generation returns scope's current generation, minting one if none is live. Minting is add-if-absent so concurrent first readers agree on one generation, and no mint can repeat a generation already used.
func generation(ctx context.Context, store Store, scope string) (string, error) {
	key := generationKeyPrefix + scope
	raw, ok, err := store.Get(ctx, key)
	if err != nil {
		return "", err
	}
	if ok {
		return string(raw), nil
	}
	gen := newGeneration()
	added, err := store.Add(ctx, key, []byte(gen), generationTTL)
	if err != nil {
		return "", err
	}
	if added {
		return gen, nil
	}
	raw, ok, err = store.Get(ctx, key)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("cache: generation for %s vanished after a concurrent mint", scope)
	}
	return string(raw), nil
}

func newGeneration() string {
	var b [4]byte
	_, _ = rand.Read(b[:])
	return strconv.FormatInt(time.Now().UnixNano(), 36) + hex.EncodeToString(b[:])
}
