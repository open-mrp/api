package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// blockedIPForTest must match an entry in blockedIPs. Tests assert against
// the production block list directly so a change there is caught here.
const blockedIPForTest = "49.43.184.36"

func TestIPBlockMiddleware_BlocksRemoteAddrWhenNoTrustedProxy(t *testing.T) {
	t.Parallel()
	mw := IPBlockMiddleware(0)
	called := false
	handler := mw(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = blockedIPForTest + ":1234"
	rr := httptest.NewRecorder()
	handler(rr, req)

	if called {
		t.Fatal("blocked IP should have been rejected before reaching the handler")
	}
	if rr.Code == http.StatusOK {
		t.Fatalf("expected non-200, got %d", rr.Code)
	}
}

// TestIPBlockMiddleware_SpoofedXFFCannotBypass is the regression test for the
// XFF trust fix as it applies to IP blocking: when an attacker connects from
// the blocked IP and tries to mask themselves with a benign X-Forwarded-For,
// they must still be blocked because RemoteAddr is what gets keyed when no
// trusted proxy is configured.
func TestIPBlockMiddleware_SpoofedXFFCannotBypass(t *testing.T) {
	t.Parallel()
	mw := IPBlockMiddleware(0)
	called := false
	handler := mw(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = blockedIPForTest + ":1234"
	req.Header.Set("X-Forwarded-For", "8.8.8.8") // attacker's attempted disguise
	rr := httptest.NewRecorder()
	handler(rr, req)

	if called {
		t.Fatal("spoofed XFF must not let a blocked IP through")
	}
}

// TestIPBlockMiddleware_BlocksRightmostXFFEntryBehindProxy confirms that
// when sitting behind one trusted proxy (e.g. ALB), the block list keys on
// the rightmost XFF entry — the one the ALB itself wrote — rather than the
// attacker-supplied leftmost value.
func TestIPBlockMiddleware_BlocksRightmostXFFEntryBehindProxy(t *testing.T) {
	t.Parallel()
	mw := IPBlockMiddleware(1)
	called := false
	handler := mw(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:443" // ALB pod IP
	// Attacker injects a benign leftmost value but ALB appends their real IP.
	req.Header.Set("X-Forwarded-For", "8.8.8.8, "+blockedIPForTest)
	rr := httptest.NewRecorder()
	handler(rr, req)

	if called {
		t.Fatal("blocked IP (rightmost XFF, written by trusted proxy) should have been rejected")
	}
}

// TestIPBlockMiddleware_AllowsUnblockedIPs sanity-checks the happy path so
// the middleware doesn't become a deny-all if blockedIPs gets misconfigured.
func TestIPBlockMiddleware_AllowsUnblockedIPs(t *testing.T) {
	t.Parallel()
	mw := IPBlockMiddleware(0)
	called := false
	handler := mw(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "203.0.113.10:5555"
	rr := httptest.NewRecorder()
	handler(rr, req)

	if !called {
		t.Fatal("unblocked IP should reach the handler")
	}
}

func TestIPBlockMiddleware_AssertBlockedIPStillBlocked(t *testing.T) {
	t.Parallel()
	if _, ok := blockedIPs[blockedIPForTest]; !ok {
		t.Fatalf("test constant %q is no longer in blockedIPs; update either the test or the block list", blockedIPForTest)
	}
}
