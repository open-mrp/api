package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/augno/api/shared/idempotency"
)

func TestIdempotencyScopeHash_SameTargetAccountSharesScope(t *testing.T) {
	t.Parallel()
	actorID := "user_123"
	targetAccountID := "acct_456"
	method := "POST"
	route := "/api/v1/orders"
	key := "idem-key-1"

	hash1 := idempotency.ComputeHTTPScopeHash(&actorID, &targetAccountID, method, route, key)
	hash2 := idempotency.ComputeHTTPScopeHash(&actorID, &targetAccountID, method, route, key)

	if hash1 != hash2 {
		t.Errorf("Expected same scope hash for same target account, got %s and %s", hash1, hash2)
	}
}

func TestIdempotencyScopeHash_DifferentTargetAccountDifferentScope(t *testing.T) {
	t.Parallel()
	actorID := "user_123"
	targetAccountID1 := "acct_456"
	targetAccountID2 := "acct_789"
	method := "POST"
	route := "/api/v1/orders"
	key := "idem-key-1"

	hash1 := idempotency.ComputeHTTPScopeHash(&actorID, &targetAccountID1, method, route, key)
	hash2 := idempotency.ComputeHTTPScopeHash(&actorID, &targetAccountID2, method, route, key)

	if hash1 == hash2 {
		t.Errorf("Expected different scope hash for different target accounts, got same hash: %s", hash1)
	}
}

func TestIdempotencyScopeHash_NilTargetAccountHandledCorrectly(t *testing.T) {
	t.Parallel()
	actorID := "user_123"
	targetAccountID := "acct_456"
	method := "POST"
	route := "/api/v1/orders"
	key := "idem-key-1"

	// With nil target account
	hashNil := idempotency.ComputeHTTPScopeHash(&actorID, nil, method, route, key)
	// With set target account
	hashSet := idempotency.ComputeHTTPScopeHash(&actorID, &targetAccountID, method, route, key)

	if hashNil == hashSet {
		t.Errorf("Expected different scope hash for nil vs set target account, got same hash: %s", hashNil)
	}
}

func TestIdempotencyScopeHash_NilIdentityHandledCorrectly(t *testing.T) {
	t.Parallel(
	// When identity is nil, both actorID and targetAccountID should be nil
	)

	method := "POST"
	route := "/api/v1/orders"
	key := "idem-key-1"

	hash1 := idempotency.ComputeHTTPScopeHash(nil, nil, method, route, key)
	hash2 := idempotency.ComputeHTTPScopeHash(nil, nil, method, route, key)

	if hash1 != hash2 {
		t.Errorf("Expected same scope hash for nil identity, got %s and %s", hash1, hash2)
	}
}

func TestIsTransientStatusCode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		code     int
		expected bool
	}{
		{200, false},
		{201, false},
		{204, false},
		{400, false},
		{401, false},
		{403, false},
		{404, false},
		{409, false},
		{422, false},
		{429, true},
		{500, true},
		{502, true},
		{503, true},
		{504, true},
		{501, false},
	}

	for _, tt := range tests {
		got := isTransientStatusCode(tt.code)
		if got != tt.expected {
			t.Errorf("isTransientStatusCode(%d) = %v, want %v", tt.code, got, tt.expected)
		}
	}
}

func TestReadAndRestoreBody_RejectsOversizedBody(t *testing.T) {
	t.Parallel()

	body := bytes.Repeat([]byte("a"), maxIdempotencyRequestBodySize+1)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/actions/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	got, _, apiErr := readAndRestoreBody(w, req)

	if apiErr == nil {
		t.Fatalf("expected APIError for oversized body, got nil")
	}
	if got != nil {
		t.Errorf("expected nil body bytes when over the limit, got %d bytes", len(got))
	}
	if !strings.Contains(apiErr.PublicMessage, "exceeds the maximum allowed size") {
		t.Errorf("unexpected public message: %q", apiErr.PublicMessage)
	}
}

func TestReadAndRestoreBody_AllowsBodyAtLimit(t *testing.T) {
	t.Parallel()

	body := bytes.Repeat([]byte("b"), maxIdempotencyRequestBodySize)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/actions/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	got, gotReq, apiErr := readAndRestoreBody(w, req)
	if apiErr != nil {
		t.Fatalf("unexpected APIError at the size limit: %v", apiErr)
	}
	if len(got) != maxIdempotencyRequestBodySize {
		t.Errorf("expected %d bytes returned, got %d", maxIdempotencyRequestBodySize, len(got))
	}

	restored, err := io.ReadAll(gotReq.Body)
	if err != nil {
		t.Fatalf("failed to read restored body: %v", err)
	}
	if len(restored) != maxIdempotencyRequestBodySize {
		t.Errorf("expected restored body of %d bytes, got %d", maxIdempotencyRequestBodySize, len(restored))
	}
}

func TestIdempotencyScopeHash_AttackScenarioPrevented(t *testing.T) {
	t.Parallel(
	// This test verifies that the security issue is fixed:
	// An attacker cannot replay a request intended for account A against account B
	)

	actorID := "attacker_123"
	targetAccountA := "acct_victim_A"
	targetAccountB := "acct_victim_B"
	method := "POST"
	route := "/api/v1/transfers"
	key := "same-idem-key"

	hashA := idempotency.ComputeHTTPScopeHash(&actorID, &targetAccountA, method, route, key)
	hashB := idempotency.ComputeHTTPScopeHash(&actorID, &targetAccountB, method, route, key)

	if hashA == hashB {
		t.Error("SECURITY ISSUE: Same idempotency key with different target accounts produces same hash. Attacker could replay requests across accounts!")
	}
}
