package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// countingConnector hands out connections that record how often they are opened and pinged.
type countingConnector struct {
	mu       sync.Mutex
	opens    int
	pings    int
	pingFail bool
}

func (c *countingConnector) Connect(context.Context) (driver.Conn, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.opens++
	return &countingConn{connector: c}, nil
}

func (c *countingConnector) Driver() driver.Driver { return nil }

func (c *countingConnector) counts() (opens, pings int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.opens, c.pings
}

type countingConn struct{ connector *countingConnector }

func (c *countingConn) Ping(context.Context) error {
	c.connector.mu.Lock()
	defer c.connector.mu.Unlock()
	c.connector.pings++
	if c.connector.pingFail {
		return driver.ErrBadConn
	}
	return nil
}

func (c *countingConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("unused") }
func (c *countingConn) Close() error                        { return nil }
func (c *countingConn) Begin() (driver.Tx, error)           { return nil, errors.New("unused") }

func newCountingPool(t *testing.T) (*sql.DB, *countingConnector) {
	t.Helper()
	connector := &countingConnector{}
	pool := sql.OpenDB(connector)
	pool.SetMaxIdleConns(10)
	t.Cleanup(func() { _ = pool.Close() })
	return pool, connector
}

func TestWarm(t *testing.T) {
	t.Parallel()

	t.Run("opens the missing connections and pings each", func(t *testing.T) {
		t.Parallel()
		pool, connector := newCountingPool(t)

		require.True(t, warm(pool, 3, time.Second))
		opens, pings := connector.counts()
		assert.Equal(t, 3, opens)
		assert.Equal(t, 3, pings)
		assert.Equal(t, 3, pool.Stats().Idle, "warmed connections go back to the pool for requests")
	})

	t.Run("a warm pool is pinged without reconnecting", func(t *testing.T) {
		t.Parallel()
		pool, connector := newCountingPool(t)

		require.True(t, warm(pool, 3, time.Second))
		require.True(t, warm(pool, 3, time.Second))
		opens, pings := connector.counts()
		assert.Equal(t, 3, opens)
		assert.Equal(t, 6, pings)
	})

	t.Run("a connection that fails its ping is replaced on the next round", func(t *testing.T) {
		t.Parallel()
		pool, connector := newCountingPool(t)

		connector.pingFail = true
		require.True(t, warm(pool, 2, time.Second))
		assert.Equal(t, 0, pool.Stats().Idle, "bad connections are discarded, not returned to the pool")

		connector.pingFail = false
		require.True(t, warm(pool, 2, time.Second))
		opens, _ := connector.counts()
		assert.Equal(t, 4, opens)
		assert.Equal(t, 2, pool.Stats().Idle)
	})

	t.Run("reports false once the pool is closed", func(t *testing.T) {
		t.Parallel()
		pool, _ := newCountingPool(t)
		require.NoError(t, pool.Close())
		assert.False(t, warm(pool, 2, time.Second))
	})
}

func TestKeepWarm_StopsWhenThePoolCloses(t *testing.T) {
	t.Parallel()
	pool, _ := newCountingPool(t)

	done := make(chan struct{})
	go func() {
		keepWarm(pool, 1, 10*time.Millisecond)
		close(done)
	}()
	require.NoError(t, pool.Close())

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("keepWarm kept running after the pool was closed")
	}
}
