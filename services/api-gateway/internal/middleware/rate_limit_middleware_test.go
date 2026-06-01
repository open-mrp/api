package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiterCheckDoesNotRecord(t *testing.T) {
	t.Parallel()
	rl := NewRateLimiter(3, time.Minute)

	for i := range 10 {
		allowed, retryAfter := rl.Check("user@example.com")
		if !allowed {
			t.Fatalf("Check should never flip on its own (iteration %d); retryAfter=%d", i, retryAfter)
		}
	}
}

func TestRateLimiterRecordFailureBlocksAtLimit(t *testing.T) {
	t.Parallel()
	rl := NewRateLimiter(3, time.Minute)

	// Up to limit-1 failures: still allowed. (Matches IsAllowed semantics:
	// the bucket fills until the limit is reached, and the *next* attempt
	// is the one that gets blocked.)
	for i := range 2 {
		rl.RecordFailure("user@example.com")
		if allowed, _ := rl.Check("user@example.com"); !allowed {
			t.Fatalf("Check should still allow after %d failures (limit=3)", i+1)
		}
	}

	rl.RecordFailure("user@example.com")
	allowed, retryAfter := rl.Check("user@example.com")
	if allowed {
		t.Fatalf("Check should block once limit is reached")
	}
	if retryAfter <= 0 {
		t.Fatalf("retryAfter should be positive when blocked, got %d", retryAfter)
	}
}

func TestRateLimiterIsolatesKeys(t *testing.T) {
	t.Parallel()
	rl := NewRateLimiter(2, time.Minute)

	for range 5 {
		rl.RecordFailure("alice@example.com")
	}

	if allowed, _ := rl.Check("alice@example.com"); allowed {
		t.Fatal("alice should be blocked after 5 failures")
	}
	if allowed, _ := rl.Check("bob@example.com"); !allowed {
		t.Fatal("bob should not be blocked by alice's failures")
	}
}

// TestRateLimitMiddleware_VaryingXFFIsRateLimited is the regression test for
// the X-Forwarded-For trust bug: when trustedProxyHops=1 (single edge proxy,
// e.g. AWS ALB), an attacker rotating the leftmost XFF value must NOT escape
// the rate limit, because the IP key is taken from the rightmost trusted
// entry which the proxy itself wrote.
func TestRateLimitMiddleware_VaryingXFFIsRateLimited(t *testing.T) {
	t.Parallel()
	mw := RateLimitMiddlewareWithConfig(5, time.Minute, 1)
	handler := mw(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	allowed, blocked := 0, 0
	for i := range 20 {
		req := httptest.NewRequest("POST", "/v1/auth/actions/login", nil)
		// Simulate the gateway being reached through an ALB that always
		// appends the same TCP source IP (the real attacker) to the right.
		req.RemoteAddr = "10.0.0.1:443"
		req.Header.Set("X-Forwarded-For", fakeAttackerXFF(i))
		rr := httptest.NewRecorder()
		handler(rr, req)

		switch rr.Code {
		case http.StatusOK:
			allowed++
		case http.StatusTooManyRequests:
			blocked++
		default:
			t.Fatalf("unexpected status %d on iteration %d", rr.Code, i)
		}
	}

	if allowed > 5 {
		t.Fatalf("expected at most 5 requests allowed, got %d", allowed)
	}
	if blocked == 0 {
		t.Fatalf("expected some requests blocked, got 0 (allowed=%d)", allowed)
	}
}

// TestRateLimitMiddleware_NoTrustedProxyIgnoresXFF ensures that when no
// trusted proxy is configured, the limiter keys on RemoteAddr only and the
// attacker cannot spoof a fresh bucket via XFF.
func TestRateLimitMiddleware_NoTrustedProxyIgnoresXFF(t *testing.T) {
	t.Parallel()
	mw := RateLimitMiddlewareWithConfig(3, time.Minute, 0)
	handler := mw(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	blocked := 0
	for i := range 10 {
		req := httptest.NewRequest("POST", "/", nil)
		req.RemoteAddr = "203.0.113.10:5555"
		req.Header.Set("X-Forwarded-For", fakeAttackerXFF(i))
		rr := httptest.NewRecorder()
		handler(rr, req)
		if rr.Code == http.StatusTooManyRequests {
			blocked++
		}
	}

	if blocked == 0 {
		t.Fatal("expected XFF to be ignored and RemoteAddr-keyed throttle to fire, but no requests were blocked")
	}
}

func fakeAttackerXFF(i int) string {
	octet := i % 256
	return "198.51.100." + itoa(octet) + ", 203.0.113.99"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [3]byte
	idx := len(buf)
	for n > 0 {
		idx--
		buf[idx] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[idx:])
}
