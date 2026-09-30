package db

import (
	"context"
	"strconv"
	"strings"
	"time"
)

// queryTimeoutMargin is how much sooner than its caller's deadline a statement is stopped, so the
// database gives up before the caller's own timeout fires rather than just after it.
const queryTimeoutMargin = 250 * time.Millisecond

// withQueryTimeout bounds a SELECT on the database side. Cancelling a context only stops the client
// waiting: the query itself runs on in vttablet or mysqld, so a report the caller abandoned at 10s
// could scan for minutes and pile up with its retries. The limit is maxTime, or the time left
// before ctx's deadline less queryTimeoutMargin when that is sooner. Both directives are set:
// vtgate honours QUERY_TIMEOUT_MS and kills the query on the tablet; plain MySQL (local, e2e)
// honours MAX_EXECUTION_TIME. Other statements pass through, so a write is never cut short.
func withQueryTimeout(ctx context.Context, query string, maxTime time.Duration) string {
	if maxTime <= 0 {
		return query
	}
	at := selectKeywordEnd(query)
	if at < 0 {
		return query
	}
	limit := maxTime
	if deadline, ok := ctx.Deadline(); ok {
		if left := time.Until(deadline) - queryTimeoutMargin; left < limit {
			limit = left
		}
	}
	ms := limit.Milliseconds()
	if ms < 1 {
		ms = 1
	}
	n := strconv.FormatInt(ms, 10)
	return query[:at] + " /*+ MAX_EXECUTION_TIME(" + n + ") */ /*vt+ QUERY_TIMEOUT_MS=" + n + " */" + query[at:]
}

// selectKeywordEnd returns the index just past a statement's leading SELECT keyword, or -1 when
// the statement is not a SELECT. Leading whitespace and comments (sqlc's "-- name:" header, block
// comments) are skipped.
func selectKeywordEnd(q string) int {
	i := 0
	for i < len(q) {
		switch {
		case q[i] == ' ' || q[i] == '\t' || q[i] == '\n' || q[i] == '\r':
			i++
		case strings.HasPrefix(q[i:], "--") || q[i] == '#':
			nl := strings.IndexByte(q[i:], '\n')
			if nl < 0 {
				return -1
			}
			i += nl + 1
		case strings.HasPrefix(q[i:], "/*"):
			end := strings.Index(q[i+2:], "*/")
			if end < 0 {
				return -1
			}
			i += 2 + end + 2
		default:
			const kw = "SELECT"
			if len(q)-i < len(kw) || !strings.EqualFold(q[i:i+len(kw)], kw) {
				return -1
			}
			end := i + len(kw)
			if end < len(q) {
				c := q[end]
				if c != ' ' && c != '\t' && c != '\n' && c != '\r' && c != '(' && c != '/' {
					return -1 // e.g. SELECTED
				}
			}
			return end
		}
	}
	return -1
}
