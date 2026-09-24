package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	defaultRedisOpTimeout = 150 * time.Millisecond
	defaultRedisPoolSize  = 20
)

// RedisStoreConfig configures a RedisStore.
type RedisStoreConfig struct {
	// URL (required) is the redis:// or rediss:// connection URL, including any password.
	URL string

	// KeyPrefix (optional; default: "") is prepended to every key so services sharing one Redis cannot collide.
	KeyPrefix string

	// OpTimeout (optional; default: 150ms) bounds each command; a slower Redis is treated as a miss rather than stalling the request.
	OpTimeout time.Duration

	// PoolSize (optional; default: 20) is the maximum number of connections per process.
	PoolSize int
}

// WithDefaults returns a copy of the config with unset fields filled. It is safe to call on a nil receiver.
func (c *RedisStoreConfig) WithDefaults() *RedisStoreConfig {
	if c == nil {
		c = &RedisStoreConfig{}
	}
	out := *c
	if out.OpTimeout == 0 {
		out.OpTimeout = defaultRedisOpTimeout
	}
	if out.PoolSize == 0 {
		out.PoolSize = defaultRedisPoolSize
	}
	return &out
}

func (c *RedisStoreConfig) validate() error {
	if c.URL == "" {
		return fmt.Errorf("cache: redis URL is required")
	}
	if c.OpTimeout <= 0 {
		return fmt.Errorf("cache: redis op timeout must be positive")
	}
	if c.PoolSize <= 0 {
		return fmt.Errorf("cache: redis pool size must be positive")
	}
	return nil
}

// RedisStore is a Store shared by every replica through one Redis. It does not connect until first use, so a Redis that is down at startup does not block the service.
type RedisStore struct {
	cfg    RedisStoreConfig
	client *redis.Client
}

// NewRedisStore returns a RedisStore for cfg.URL.
func NewRedisStore(cfg *RedisStoreConfig) (*RedisStore, error) {
	cfg = cfg.WithDefaults()
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	opts, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("cache: invalid redis URL: %w", err)
	}
	opts.PoolSize = cfg.PoolSize
	opts.ReadTimeout = cfg.OpTimeout
	opts.WriteTimeout = cfg.OpTimeout
	opts.DialTimeout = 2 * cfg.OpTimeout
	opts.MaxRetries = 1
	opts.ContextTimeoutEnabled = true

	return &RedisStore{cfg: *cfg, client: redis.NewClient(opts)}, nil
}

func (s *RedisStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, s.cfg.OpTimeout)
	defer cancel()

	value, err := s.client.Get(ctx, s.cfg.KeyPrefix+key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return value, true, nil
}

func (s *RedisStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, s.cfg.OpTimeout)
	defer cancel()

	return s.client.Set(ctx, s.cfg.KeyPrefix+key, value, ttl).Err()
}

func (s *RedisStore) Add(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, s.cfg.OpTimeout)
	defer cancel()

	return s.client.SetNX(ctx, s.cfg.KeyPrefix+key, value, ttl).Result()
}

// Ping reports whether Redis is reachable.
func (s *RedisStore) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*s.cfg.OpTimeout)
	defer cancel()
	return s.client.Ping(ctx).Err()
}

func (s *RedisStore) Close() error {
	return s.client.Close()
}
