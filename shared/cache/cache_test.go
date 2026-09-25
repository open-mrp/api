package cache

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	apierror "github.com/open-mrp/api/shared/errors"

	"github.com/alicebob/miniredis/v2"
)

type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time { return c.now }

func newTestCache[T any](t *testing.T, store Store) *Cache[T] {
	t.Helper()
	c, err := New[T](&Config{Name: "test", Store: store, TTL: time.Minute})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func newMemory(t *testing.T, cfg *MemoryStoreConfig) *MemoryStore {
	t.Helper()
	s, err := NewMemoryStore(cfg)
	if err != nil {
		t.Fatalf("NewMemoryStore: %v", err)
	}
	return s
}

func countingLoader(calls *atomic.Int32, value string) func(context.Context) (string, *apierror.APIError) {
	return func(context.Context) (string, *apierror.APIError) {
		calls.Add(1)
		return value, nil
	}
}

func TestMemoryStore_ExpiresEntries(t *testing.T) {
	clock := &fakeClock{now: time.Unix(0, 0)}
	s := newMemory(t, &MemoryStoreConfig{Now: clock.Now})
	ctx := context.Background()

	_ = s.Set(ctx, "k", []byte("v"), time.Second)
	if _, ok, _ := s.Get(ctx, "k"); !ok {
		t.Fatal("expected hit before expiry")
	}
	clock.now = clock.now.Add(time.Second)
	if _, ok, _ := s.Get(ctx, "k"); ok {
		t.Fatal("expected miss at expiry")
	}
	if s.Len() != 0 {
		t.Fatalf("expired entry not removed, len=%d", s.Len())
	}
}

func TestMemoryStore_EvictsLeastRecentlyUsed(t *testing.T) {
	s := newMemory(t, &MemoryStoreConfig{MaxEntries: 2})
	ctx := context.Background()

	_ = s.Set(ctx, "a", []byte("1"), time.Minute)
	_ = s.Set(ctx, "b", []byte("2"), time.Minute)
	_, _, _ = s.Get(ctx, "a")
	_ = s.Set(ctx, "c", []byte("3"), time.Minute)

	if _, ok, _ := s.Get(ctx, "b"); ok {
		t.Fatal("b is least recently used and should have been evicted")
	}
	for _, k := range []string{"a", "c"} {
		if _, ok, _ := s.Get(ctx, k); !ok {
			t.Fatalf("%s should still be cached", k)
		}
	}
}

func TestCache_HitsAfterFirstLoad(t *testing.T) {
	c := newTestCache[string](t, newMemory(t, nil))
	var calls atomic.Int32
	key := Key{Scopes: []string{"account:a"}, ID: "x"}

	for range 3 {
		v, err := c.GetOrLoad(context.Background(), key, countingLoader(&calls, "v"))
		if err != nil || v != "v" {
			t.Fatalf("got (%q, %v)", v, err)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("loader ran %d times, want 1", calls.Load())
	}
}

func TestCache_InvalidateOrphansScopeOnly(t *testing.T) {
	store := newMemory(t, nil)
	c := newTestCache[string](t, store)
	ctx := context.Background()
	var aCalls, bCalls atomic.Int32
	a := Key{Scopes: []string{"account:a"}, ID: "x"}
	b := Key{Scopes: []string{"account:b"}, ID: "x"}

	_, _ = c.GetOrLoad(ctx, a, countingLoader(&aCalls, "a1"))
	_, _ = c.GetOrLoad(ctx, b, countingLoader(&bCalls, "b1"))

	if err := Invalidate(ctx, store, "account:a"); err != nil {
		t.Fatalf("Invalidate: %v", err)
	}

	v, _ := c.GetOrLoad(ctx, a, countingLoader(&aCalls, "a2"))
	if v != "a2" || aCalls.Load() != 2 {
		t.Fatalf("invalidated scope served %q after %d loads", v, aCalls.Load())
	}
	v, _ = c.GetOrLoad(ctx, b, countingLoader(&bCalls, "b2"))
	if v != "b1" || bCalls.Load() != 1 {
		t.Fatalf("untouched scope reloaded: %q after %d loads", v, bCalls.Load())
	}
}

func TestCache_InvalidationDuringLoadIsNotOverwritten(t *testing.T) {
	store := newMemory(t, nil)
	c := newTestCache[string](t, store)
	ctx := context.Background()
	key := Key{Scopes: []string{"role:r"}, ID: "perms"}

	_, _ = c.GetOrLoad(ctx, key, func(ctx context.Context) (string, *apierror.APIError) {
		if err := Invalidate(ctx, store, "role:r"); err != nil {
			t.Fatalf("Invalidate: %v", err)
		}
		return "stale", nil
	})

	v, _ := c.GetOrLoad(ctx, key, func(context.Context) (string, *apierror.APIError) { return "fresh", nil })
	if v != "fresh" {
		t.Fatalf("a load that raced an invalidation was served afterwards: %q", v)
	}
}

func TestCache_ConcurrentMissesShareOneLoad(t *testing.T) {
	c := newTestCache[string](t, newMemory(t, nil))
	var calls atomic.Int32
	release := make(chan struct{})
	load := func(context.Context) (string, *apierror.APIError) {
		calls.Add(1)
		<-release
		return "v", nil
	}

	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			if v, err := c.GetOrLoad(context.Background(), Key{Scopes: []string{"s"}, ID: "k"}, load); err != nil || v != "v" {
				t.Errorf("got (%q, %v)", v, err)
			}
		})
	}
	time.Sleep(20 * time.Millisecond)
	close(release)
	wg.Wait()

	if calls.Load() != 1 {
		t.Fatalf("loader ran %d times, want 1", calls.Load())
	}
}

func TestCache_ErrorsAreNotCached(t *testing.T) {
	c := newTestCache[string](t, newMemory(t, nil))
	ctx := context.Background()
	key := Key{ID: "k"}

	_, err := c.GetOrLoad(ctx, key, func(context.Context) (string, *apierror.APIError) {
		return "", apierror.NewInternalError(nil, "boom")
	})
	if err == nil {
		t.Fatal("expected loader error")
	}
	v, err := c.GetOrLoad(ctx, key, func(context.Context) (string, *apierror.APIError) { return "ok", nil })
	if err != nil || v != "ok" {
		t.Fatalf("error was cached: (%q, %v)", v, err)
	}
}

func TestCache_ReturnsPrivateCopies(t *testing.T) {
	c := newTestCache[map[string]bool](t, newMemory(t, nil))
	ctx := context.Background()
	load := func(context.Context) (map[string]bool, *apierror.APIError) { return map[string]bool{"a": true}, nil }

	first, _ := c.GetOrLoad(ctx, Key{ID: "k"}, load)
	first["mutated"] = true
	second, _ := c.GetOrLoad(ctx, Key{ID: "k"}, load)
	if second["mutated"] {
		t.Fatal("a caller's mutation leaked into the cached value")
	}
}

type failingStore struct{}

func (failingStore) Get(context.Context, string) ([]byte, bool, error) {
	return nil, false, errors.New("down")
}

func (failingStore) Set(context.Context, string, []byte, time.Duration) error {
	return errors.New("down")
}

func (failingStore) Add(context.Context, string, []byte, time.Duration) (bool, error) {
	return false, errors.New("down")
}

func TestCache_FailsOpenWhenStoreErrors(t *testing.T) {
	c := newTestCache[string](t, failingStore{})
	var calls atomic.Int32

	for range 2 {
		v, err := c.GetOrLoad(context.Background(), Key{Scopes: []string{"s"}, ID: "k"}, countingLoader(&calls, "v"))
		if err != nil || v != "v" {
			t.Fatalf("store failure surfaced to caller: (%q, %v)", v, err)
		}
	}
	if calls.Load() != 2 {
		t.Fatalf("loader ran %d times, want 2", calls.Load())
	}
}

func TestCache_NilStoreAlwaysLoads(t *testing.T) {
	c := newTestCache[string](t, nil)
	var calls atomic.Int32
	for range 2 {
		_, _ = c.GetOrLoad(context.Background(), Key{ID: "k"}, countingLoader(&calls, "v"))
	}
	if calls.Load() != 2 {
		t.Fatalf("loader ran %d times, want 2", calls.Load())
	}
}

func TestCache_CallerCancellationDoesNotAbortSharedLoad(t *testing.T) {
	store := newMemory(t, nil)
	c := newTestCache[string](t, store)
	release := make(chan struct{})
	done := make(chan struct{})
	key := Key{Scopes: []string{"s"}, ID: "k"}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer close(done)
		_, err := c.GetOrLoad(ctx, key, func(loadCtx context.Context) (string, *apierror.APIError) {
			<-release
			if loadCtx.Err() != nil {
				t.Errorf("load context cancelled with its caller: %v", loadCtx.Err())
			}
			return "v", nil
		})
		if err == nil {
			t.Error("cancelled caller should get an error")
		}
	}()
	time.Sleep(10 * time.Millisecond)
	cancel()
	<-done
	close(release)

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		v, _ := c.GetOrLoad(context.Background(), key, func(context.Context) (string, *apierror.APIError) { return "", nil })
		if v == "v" {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("abandoned load never populated the cache")
}

func TestRedisStore_SharesEntriesAndInvalidationsAcrossClients(t *testing.T) {
	mr := miniredis.RunT(t)
	newStore := func() *RedisStore {
		s, err := NewRedisStore(&RedisStoreConfig{URL: "redis://" + mr.Addr(), KeyPrefix: "core:"})
		if err != nil {
			t.Fatalf("NewRedisStore: %v", err)
		}
		t.Cleanup(func() { _ = s.Close() })
		return s
	}
	replicaA, replicaB := newStore(), newStore()
	cacheA := newTestCache[string](t, replicaA)
	cacheB := newTestCache[string](t, replicaB)
	ctx := context.Background()
	key := Key{Scopes: []string{"account:a"}, ID: "report"}
	var calls atomic.Int32

	_, _ = cacheA.GetOrLoad(ctx, key, countingLoader(&calls, "v1"))
	if v, _ := cacheB.GetOrLoad(ctx, key, countingLoader(&calls, "unused")); v != "v1" {
		t.Fatalf("second replica missed a shared entry: %q", v)
	}

	if err := Invalidate(ctx, replicaB, "account:a"); err != nil {
		t.Fatalf("Invalidate: %v", err)
	}
	if v, _ := cacheA.GetOrLoad(ctx, key, countingLoader(&calls, "v2")); v != "v2" {
		t.Fatalf("invalidation on one replica did not reach the other: %q", v)
	}
	if !mr.Exists("core:gen:account:a") {
		t.Fatal("key prefix not applied")
	}
}

func TestRedisStore_UnreachableIsAMiss(t *testing.T) {
	mr := miniredis.RunT(t)
	store, err := NewRedisStore(&RedisStoreConfig{URL: "redis://" + mr.Addr()})
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	mr.Close()

	c := newTestCache[string](t, store)
	v, apiErr := c.GetOrLoad(context.Background(), Key{Scopes: []string{"s"}, ID: "k"}, func(context.Context) (string, *apierror.APIError) { return "v", nil })
	if apiErr != nil || v != "v" {
		t.Fatalf("Redis outage surfaced to caller: (%q, %v)", v, apiErr)
	}
}

func TestConfig_Validate(t *testing.T) {
	if _, err := New[string](&Config{TTL: time.Minute}); err == nil {
		t.Error("missing name accepted")
	}
	if _, err := New[string](&Config{Name: "n"}); err == nil {
		t.Error("missing TTL accepted")
	}
	if _, err := NewRedisStore(&RedisStoreConfig{}); err == nil {
		t.Error("missing redis URL accepted")
	}
}

func TestCache_EntryDiesWithAnyOfItsScopes(t *testing.T) {
	store := newMemory(t, nil)
	c := newTestCache[string](t, store)
	ctx := context.Background()
	var calls atomic.Int32
	key := Key{Scopes: []string{"account:target", "account:actor"}, ID: "relation"}

	_, _ = c.GetOrLoad(ctx, key, countingLoader(&calls, "v1"))
	_ = Invalidate(ctx, store, "account:actor")
	if v, _ := c.GetOrLoad(ctx, key, countingLoader(&calls, "v2")); v != "v2" {
		t.Fatalf("invalidating the second scope left the entry live: %q", v)
	}
	_ = Invalidate(ctx, store, "account:target")
	if v, _ := c.GetOrLoad(ctx, key, countingLoader(&calls, "v3")); v != "v3" {
		t.Fatalf("invalidating the first scope left the entry live: %q", v)
	}
}

func TestCache_TTLFuncOverridesDefault(t *testing.T) {
	clock := &fakeClock{now: time.Unix(0, 0)}
	store := newMemory(t, &MemoryStoreConfig{Now: clock.Now})
	c, err := New[string](&Config{Name: "test", Store: store, TTL: time.Hour}, WithTTLFunc(func(v string) time.Duration {
		if v == "" {
			return time.Second
		}
		return 0
	}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx := context.Background()
	var calls atomic.Int32

	_, _ = c.GetOrLoad(ctx, Key{ID: "negative"}, countingLoader(&calls, ""))
	_, _ = c.GetOrLoad(ctx, Key{ID: "positive"}, countingLoader(&calls, "v"))
	clock.now = clock.now.Add(2 * time.Second)
	_, _ = c.GetOrLoad(ctx, Key{ID: "negative"}, countingLoader(&calls, ""))
	_, _ = c.GetOrLoad(ctx, Key{ID: "positive"}, countingLoader(&calls, "v"))

	if calls.Load() != 3 {
		t.Fatalf("loader ran %d times, want 3 (only the negative entry expires)", calls.Load())
	}
}

func TestCache_OversizeValuesAreReturnedButNotStored(t *testing.T) {
	c, err := New[string](&Config{Name: "test", Store: newMemory(t, nil), TTL: time.Minute, MaxValueBytes: 8})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	var calls atomic.Int32
	for range 2 {
		v, apiErr := c.GetOrLoad(context.Background(), Key{ID: "k"}, countingLoader(&calls, "much too long"))
		if apiErr != nil || v != "much too long" {
			t.Fatalf("got (%q, %v)", v, apiErr)
		}
	}
	if calls.Load() != 2 {
		t.Fatalf("oversize value was cached: loader ran %d times", calls.Load())
	}
}

func TestCache_PeekNeverLoads(t *testing.T) {
	c := newTestCache[string](t, newMemory(t, nil))
	key := Key{ID: "x"}

	if _, ok := c.Peek(context.Background(), key); ok {
		t.Fatal("Peek hit an empty cache")
	}
	var calls atomic.Int32
	if _, err := c.GetOrLoad(context.Background(), key, countingLoader(&calls, "v")); err != nil {
		t.Fatalf("GetOrLoad: %v", err)
	}
	if v, ok := c.Peek(context.Background(), key); !ok || v != "v" {
		t.Fatalf("Peek = (%q, %v), want the loaded value", v, ok)
	}
	if calls.Load() != 1 {
		t.Fatalf("loader ran %d times, want 1", calls.Load())
	}
}

func TestCache_PeekWithoutStoreMisses(t *testing.T) {
	c := newTestCache[string](t, nil)
	if _, ok := c.Peek(context.Background(), Key{ID: "x"}); ok {
		t.Fatal("a cache with no store must always miss")
	}
}
