package middleware

import (
	"testing"

	"github.com/augno/api/shared/idempotency"
)

func TestIdempotencyScopeHash_SameTargetAccountSharesScope(t *testing.T) {
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
	// When identity is nil, both actorID and targetAccountID should be nil
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

func TestIdempotencyScopeHash_AttackScenarioPrevented(t *testing.T) {
	// This test verifies that the security issue is fixed:
	// An attacker cannot replay a request intended for account A against account B
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
