package db

import (
	"context"
	"database/sql"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var timeoutDirective = regexp.MustCompile(`/\*\+ MAX_EXECUTION_TIME\((\d+)\) \*/ /\*vt\+ QUERY_TIMEOUT_MS=(\d+) \*/`)

func TestWithQueryTimeoutBoundsSelectsOnly(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	const d = " /*+ MAX_EXECUTION_TIME(30000) */ /*vt+ QUERY_TIMEOUT_MS=30000 */"
	cases := map[string]string{
		"SELECT 1":                              "SELECT" + d + " 1",
		"  select a FROM t":                     "  select" + d + " a FROM t",
		"-- name: ListX :many\nSELECT a FROM t": "-- name: ListX :many\nSELECT" + d + " a FROM t",
		"/* lead */ SELECT(1)":                  "/* lead */ SELECT" + d + "(1)",
		"SELECT /*+ NO_SEMIJOIN() */ 1":         "SELECT" + d + " /*+ NO_SEMIJOIN() */ 1",
		"UPDATE t SET a = 1":                    "UPDATE t SET a = 1",
		"INSERT INTO t SELECT 1":                "INSERT INTO t SELECT 1",
		"DELETE FROM t WHERE id IN (SELECT 1)":  "DELETE FROM t WHERE id IN (SELECT 1)",
		"WITH x AS (SELECT 1) SELECT * FROM x":  "WITH x AS (SELECT 1) SELECT * FROM x",
		"SELECTED":                              "SELECTED",
		"-- only a comment":                     "-- only a comment",
	}
	for in, want := range cases {
		assert.Equal(t, want, withQueryTimeout(ctx, in, 30*time.Second), in)
	}
	assert.Equal(t, "SELECT 1", withQueryTimeout(ctx, "SELECT 1", 0), "no limit configured")
}

func directiveMillis(t *testing.T, q string) int64 {
	t.Helper()
	m := timeoutDirective.FindStringSubmatch(q)
	require.NotNil(t, m, q)
	require.Equal(t, m[1], m[2], "both directives carry the same limit")
	ms, err := strconv.ParseInt(m[1], 10, 64)
	require.NoError(t, err)
	return ms
}

func TestWithQueryTimeoutStopsBeforeTheCallersDeadline(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ms := directiveMillis(t, withQueryTimeout(ctx, "SELECT 1", 30*time.Second))
	assert.LessOrEqual(t, ms, int64(10_000-250), "stops before the deadline, with a margin")
	assert.Greater(t, ms, int64(9_000))

	// The configured cap wins when it is sooner than the deadline.
	assert.Equal(t, int64(2_000), directiveMillis(t, withQueryTimeout(ctx, "SELECT 1", 2*time.Second)))

	// A deadline already past still yields a valid, tiny limit rather than zero (which MySQL reads as "no limit").
	past, cancelPast := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancelPast()
	assert.Equal(t, int64(1), directiveMillis(t, withQueryTimeout(past, "SELECT 1", time.Minute)))
}

func TestTaggingConnectorAppliesTheQueryTimeoutWithTags(t *testing.T) {
	t.Parallel()
	rec := &recordingDriver{}
	pool := sql.OpenDB(taggingConnector{base: rec, static: map[string]string{"app": "core-service"}, maxQueryTime: 5 * time.Second})
	defer pool.Close()
	rows, err := pool.QueryContext(context.Background(), "SELECT 1")
	require.NoError(t, err)
	require.NoError(t, rows.Close())
	_, err = pool.ExecContext(context.Background(), "UPDATE t SET a = 1")
	require.NoError(t, err)
	require.Equal(t, []string{
		"SELECT /*+ MAX_EXECUTION_TIME(5000) */ /*vt+ QUERY_TIMEOUT_MS=5000 */ 1 /*app='core-service'*/",
		"UPDATE t SET a = 1 /*app='core-service'*/",
	}, rec.seen)
}
