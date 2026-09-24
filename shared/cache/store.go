// Package cache is a read-through cache for data that is expensive to load and tolerates bounded staleness: the per-request auth lookups and the heavy analytics aggregates.
//
// Entries are grouped into scopes (typically one account, role, or API key). Invalidating a scope rotates its generation, which orphans every entry written under the old one; nothing is deleted. A load that started before an invalidation writes under the generation it read, so it cannot resurrect stale data after the invalidation lands.
//
// The cache fails open: a store error is treated as a miss, and the loader's result is returned uncached. A cache outage costs latency, never availability.
package cache

import (
	"context"
	"time"
)

// Store is the byte-level backend a Cache reads and writes through. Implementations must be safe for concurrent use.
type Store interface {
	// Get returns the value stored under key and whether it was present and unexpired.
	Get(ctx context.Context, key string) ([]byte, bool, error)

	// Set stores value under key for ttl.
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// Add stores value under key for ttl only if key holds no live value, and reports whether it did.
	Add(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error)
}
