package db

import (
	"context"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"

	"github.com/open-mrp/api/shared/retry"
)

func fastConnRetryConfig() *retry.Config {
	return (&retry.Config{
		MaxRetries:     2,
		InitialWait:    time.Millisecond,
		MaxWait:        5 * time.Millisecond,
		Multiplier:     2.0,
		JitterFraction: 0.2,
	}).WithDefaults()
}

func TestWithConnRetry(t *testing.T) {
	t.Parallel()

	lostConn := &mysql.MySQLError{Number: 2013, Message: "Lost connection to MySQL server during query"}

	t.Run("succeeds without retry", func(t *testing.T) {
		t.Parallel()
		calls := 0
		err := WithConnRetry(context.Background(), fastConnRetryConfig(), "test.op", func() error {
			calls++
			return nil
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, calls)
	})

	t.Run("retries transient connection error until success", func(t *testing.T) {
		t.Parallel()
		calls := 0
		err := WithConnRetry(context.Background(), fastConnRetryConfig(), "test.op", func() error {
			calls++
			if calls < 2 {
				return lostConn
			}
			return nil
		})
		assert.NoError(t, err)
		assert.Equal(t, 2, calls)
	})

	t.Run("returns last error when retries are exhausted", func(t *testing.T) {
		t.Parallel()
		calls := 0
		err := WithConnRetry(context.Background(), fastConnRetryConfig(), "test.op", func() error {
			calls++
			return lostConn
		})
		assert.ErrorIs(t, err, lostConn)
		assert.Equal(t, 3, calls) // 1 initial + 2 retries
	})

	t.Run("does not retry non-connection errors", func(t *testing.T) {
		t.Parallel()
		calls := 0
		dup := &mysql.MySQLError{Number: 1062, Message: "Duplicate entry"}
		err := WithConnRetry(context.Background(), fastConnRetryConfig(), "test.op", func() error {
			calls++
			return dup
		})
		assert.ErrorIs(t, err, dup)
		assert.Equal(t, 1, calls)
	})

	t.Run("stops when context is canceled between attempts", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())
		calls := 0
		err := WithConnRetry(ctx, fastConnRetryConfig(), "test.op", func() error {
			calls++
			cancel()
			return lostConn
		})
		assert.ErrorIs(t, err, context.Canceled)
		assert.Equal(t, 1, calls)
	})

	t.Run("nil config uses default", func(t *testing.T) {
		t.Parallel()
		calls := 0
		err := WithConnRetry(context.Background(), nil, "test.op", func() error {
			calls++
			if calls < 2 {
				return lostConn
			}
			return nil
		})
		assert.NoError(t, err)
		assert.Equal(t, 2, calls)
	})

	t.Run("partially filled config honors MaxRetries", func(t *testing.T) {
		t.Parallel()
		calls := 0
		err := WithConnRetry(context.Background(), &retry.Config{MaxRetries: 1, InitialWait: time.Millisecond}, "test.op", func() error {
			calls++
			return lostConn
		})
		assert.ErrorIs(t, err, lostConn)
		assert.Equal(t, 2, calls) // 1 initial + 1 retry
	})
}
