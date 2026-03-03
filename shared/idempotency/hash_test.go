package idempotency

import (
	"encoding/json"
	"testing"
)

func TestComputeHTTPScopeHash_SameInputsSameHash(t *testing.T) {
	actorID := "user_123"
	targetAccountID := "acct_456"
	method := "POST"
	normalizedRoute := "/api/v1/orders"
	idempotencyKey := "key-abc"

	hash1 := ComputeHTTPScopeHash(&actorID, &targetAccountID, method, normalizedRoute, idempotencyKey)
	hash2 := ComputeHTTPScopeHash(&actorID, &targetAccountID, method, normalizedRoute, idempotencyKey)

	if hash1 != hash2 {
		t.Errorf("Expected same hash for same inputs, got %s and %s", hash1, hash2)
	}
}

func TestComputeHTTPScopeHash_DifferentActorID(t *testing.T) {
	actorID1 := "user_123"
	actorID2 := "user_456"
	targetAccountID := "acct_456"
	method := "POST"
	normalizedRoute := "/api/v1/orders"
	idempotencyKey := "key-abc"

	hash1 := ComputeHTTPScopeHash(&actorID1, &targetAccountID, method, normalizedRoute, idempotencyKey)
	hash2 := ComputeHTTPScopeHash(&actorID2, &targetAccountID, method, normalizedRoute, idempotencyKey)

	if hash1 == hash2 {
		t.Errorf("Expected different hash for different actorIDs, got same hash: %s", hash1)
	}
}

func TestComputeHTTPScopeHash_DifferentTargetAccountID(t *testing.T) {
	actorID := "user_123"
	targetAccountID1 := "acct_456"
	targetAccountID2 := "acct_789"
	method := "POST"
	normalizedRoute := "/api/v1/orders"
	idempotencyKey := "key-abc"

	hash1 := ComputeHTTPScopeHash(&actorID, &targetAccountID1, method, normalizedRoute, idempotencyKey)
	hash2 := ComputeHTTPScopeHash(&actorID, &targetAccountID2, method, normalizedRoute, idempotencyKey)

	if hash1 == hash2 {
		t.Errorf("Expected different hash for different targetAccountIDs, got same hash: %s", hash1)
	}
}

func TestComputeHTTPScopeHash_DifferentMethod(t *testing.T) {
	actorID := "user_123"
	targetAccountID := "acct_456"
	normalizedRoute := "/api/v1/orders"
	idempotencyKey := "key-abc"

	hash1 := ComputeHTTPScopeHash(&actorID, &targetAccountID, "POST", normalizedRoute, idempotencyKey)
	hash2 := ComputeHTTPScopeHash(&actorID, &targetAccountID, "PATCH", normalizedRoute, idempotencyKey)

	if hash1 == hash2 {
		t.Errorf("Expected different hash for different methods, got same hash: %s", hash1)
	}
}

func TestComputeHTTPScopeHash_DifferentRoute(t *testing.T) {
	actorID := "user_123"
	targetAccountID := "acct_456"
	method := "POST"
	idempotencyKey := "key-abc"

	hash1 := ComputeHTTPScopeHash(&actorID, &targetAccountID, method, "/api/v1/orders", idempotencyKey)
	hash2 := ComputeHTTPScopeHash(&actorID, &targetAccountID, method, "/api/v1/users", idempotencyKey)

	if hash1 == hash2 {
		t.Errorf("Expected different hash for different routes, got same hash: %s", hash1)
	}
}

func TestComputeHTTPScopeHash_DifferentIdempotencyKey(t *testing.T) {
	actorID := "user_123"
	targetAccountID := "acct_456"
	method := "POST"
	normalizedRoute := "/api/v1/orders"

	hash1 := ComputeHTTPScopeHash(&actorID, &targetAccountID, method, normalizedRoute, "key-abc")
	hash2 := ComputeHTTPScopeHash(&actorID, &targetAccountID, method, normalizedRoute, "key-xyz")

	if hash1 == hash2 {
		t.Errorf("Expected different hash for different idempotency keys, got same hash: %s", hash1)
	}
}

func TestComputeHTTPScopeHash_NilActorID(t *testing.T) {
	targetAccountID := "acct_456"
	method := "POST"
	normalizedRoute := "/api/v1/orders"
	idempotencyKey := "key-abc"

	hash1 := ComputeHTTPScopeHash(nil, &targetAccountID, method, normalizedRoute, idempotencyKey)
	hash2 := ComputeHTTPScopeHash(nil, &targetAccountID, method, normalizedRoute, idempotencyKey)

	if hash1 != hash2 {
		t.Errorf("Expected same hash for nil actorID, got %s and %s", hash1, hash2)
	}
}

func TestComputeHTTPScopeHash_NilTargetAccountID(t *testing.T) {
	actorID := "user_123"
	method := "POST"
	normalizedRoute := "/api/v1/orders"
	idempotencyKey := "key-abc"

	hash1 := ComputeHTTPScopeHash(&actorID, nil, method, normalizedRoute, idempotencyKey)
	hash2 := ComputeHTTPScopeHash(&actorID, nil, method, normalizedRoute, idempotencyKey)

	if hash1 != hash2 {
		t.Errorf("Expected same hash for nil targetAccountID, got %s and %s", hash1, hash2)
	}
}

func TestComputeHTTPScopeHash_NilVsEmptyString(t *testing.T) {
	emptyString := ""
	targetAccountID := "acct_456"
	method := "POST"
	normalizedRoute := "/api/v1/orders"
	idempotencyKey := "key-abc"

	// nil actorID should be treated same as empty string
	hashNil := ComputeHTTPScopeHash(nil, &targetAccountID, method, normalizedRoute, idempotencyKey)
	hashEmpty := ComputeHTTPScopeHash(&emptyString, &targetAccountID, method, normalizedRoute, idempotencyKey)

	if hashNil != hashEmpty {
		t.Errorf("Expected same hash for nil and empty actorID, got %s and %s", hashNil, hashEmpty)
	}
}

func TestComputeHTTPScopeHash_NilVsEmptyStringTargetAccount(t *testing.T) {
	actorID := "user_123"
	emptyString := ""
	method := "POST"
	normalizedRoute := "/api/v1/orders"
	idempotencyKey := "key-abc"

	// nil targetAccountID should be treated same as empty string
	hashNil := ComputeHTTPScopeHash(&actorID, nil, method, normalizedRoute, idempotencyKey)
	hashEmpty := ComputeHTTPScopeHash(&actorID, &emptyString, method, normalizedRoute, idempotencyKey)

	if hashNil != hashEmpty {
		t.Errorf("Expected same hash for nil and empty targetAccountID, got %s and %s", hashNil, hashEmpty)
	}
}

func TestComputeHTTPScopeHash_BothNullableFieldsNil(t *testing.T) {
	method := "POST"
	normalizedRoute := "/api/v1/orders"
	idempotencyKey := "key-abc"

	hash1 := ComputeHTTPScopeHash(nil, nil, method, normalizedRoute, idempotencyKey)
	hash2 := ComputeHTTPScopeHash(nil, nil, method, normalizedRoute, idempotencyKey)

	if hash1 != hash2 {
		t.Errorf("Expected same hash when both nullable fields are nil, got %s and %s", hash1, hash2)
	}
}

func TestComputeHTTPScopeHash_NilActorVsNilTargetAccount(t *testing.T) {
	actorID := "user_123"
	targetAccountID := "acct_456"
	method := "POST"
	normalizedRoute := "/api/v1/orders"
	idempotencyKey := "key-abc"

	// Hash with nil actorID but set targetAccountID
	hash1 := ComputeHTTPScopeHash(nil, &targetAccountID, method, normalizedRoute, idempotencyKey)
	// Hash with set actorID but nil targetAccountID
	hash2 := ComputeHTTPScopeHash(&actorID, nil, method, normalizedRoute, idempotencyKey)

	if hash1 == hash2 {
		t.Errorf("Expected different hash for different combinations of nil fields, got same hash: %s", hash1)
	}
}

// --- ComputeServiceScopeHash ---

func TestComputeServiceScopeHash_SameInputsSameHash(t *testing.T) {
	actorID := "user_123"
	accountID := "acct_456"
	hash1 := ComputeServiceScopeHash(&actorID, &accountID, "order-service", "CreateOrder", "key-1")
	hash2 := ComputeServiceScopeHash(&actorID, &accountID, "order-service", "CreateOrder", "key-1")
	if hash1 != hash2 {
		t.Errorf("expected same hash, got %s and %s", hash1, hash2)
	}
}

func TestComputeServiceScopeHash_DifferentService(t *testing.T) {
	actorID := "user_123"
	accountID := "acct_456"
	hash1 := ComputeServiceScopeHash(&actorID, &accountID, "order-service", "Create", "key-1")
	hash2 := ComputeServiceScopeHash(&actorID, &accountID, "auth-service", "Create", "key-1")
	if hash1 == hash2 {
		t.Errorf("expected different hashes for different services, got same: %s", hash1)
	}
}

func TestComputeServiceScopeHash_DifferentHandler(t *testing.T) {
	actorID := "user_123"
	accountID := "acct_456"
	hash1 := ComputeServiceScopeHash(&actorID, &accountID, "order-service", "Create", "key-1")
	hash2 := ComputeServiceScopeHash(&actorID, &accountID, "order-service", "Delete", "key-1")
	if hash1 == hash2 {
		t.Errorf("expected different hashes for different handlers, got same: %s", hash1)
	}
}

func TestComputeServiceScopeHash_DifferentKey(t *testing.T) {
	actorID := "user_123"
	accountID := "acct_456"
	hash1 := ComputeServiceScopeHash(&actorID, &accountID, "order-service", "Create", "key-1")
	hash2 := ComputeServiceScopeHash(&actorID, &accountID, "order-service", "Create", "key-2")
	if hash1 == hash2 {
		t.Errorf("expected different hashes for different keys, got same: %s", hash1)
	}
}

func TestComputeServiceScopeHash_DifferentAccount(t *testing.T) {
	actorID := "user_123"
	accountA := "acct_A"
	accountB := "acct_B"
	hash1 := ComputeServiceScopeHash(&actorID, &accountA, "order-service", "Create", "key-1")
	hash2 := ComputeServiceScopeHash(&actorID, &accountB, "order-service", "Create", "key-1")
	if hash1 == hash2 {
		t.Errorf("expected different hashes for different target accounts, got same: %s", hash1)
	}
}

func TestComputeServiceScopeHash_NilActorID(t *testing.T) {
	hash1 := ComputeServiceScopeHash(nil, nil, "svc", "handler", "key")
	hash2 := ComputeServiceScopeHash(nil, nil, "svc", "handler", "key")
	if hash1 != hash2 {
		t.Errorf("expected same hash for nil actorID, got %s and %s", hash1, hash2)
	}
}

func TestComputeServiceScopeHash_NilVsEmptyActorID(t *testing.T) {
	empty := ""
	hashNil := ComputeServiceScopeHash(nil, nil, "svc", "handler", "key")
	hashEmpty := ComputeServiceScopeHash(&empty, nil, "svc", "handler", "key")
	if hashNil != hashEmpty {
		t.Errorf("expected same hash for nil and empty actorID, got %s and %s", hashNil, hashEmpty)
	}
}

func TestComputeServiceScopeHash_NilVsEmptyTargetAccountID(t *testing.T) {
	actorID := "user_123"
	empty := ""
	hashNil := ComputeServiceScopeHash(&actorID, nil, "svc", "handler", "key")
	hashEmpty := ComputeServiceScopeHash(&actorID, &empty, "svc", "handler", "key")
	if hashNil != hashEmpty {
		t.Errorf("expected same hash for nil and empty targetAccountID, got %s and %s", hashNil, hashEmpty)
	}
}

// --- ComputeRequestBodyHash ---

func TestComputeRequestBodyHash_SameBodySameHash(t *testing.T) {
	body := []byte(`{"name":"test","value":1}`)
	hash1 := ComputeRequestBodyHash(body, nil)
	hash2 := ComputeRequestBodyHash(body, nil)
	if hash1 != hash2 {
		t.Errorf("expected same hash, got %s and %s", hash1, hash2)
	}
}

func TestComputeRequestBodyHash_KeyOrderInsensitive(t *testing.T) {
	body1 := []byte(`{"name":"test","value":1}`)
	body2 := []byte(`{"value":1,"name":"test"}`)
	hash1 := ComputeRequestBodyHash(body1, nil)
	hash2 := ComputeRequestBodyHash(body2, nil)
	if hash1 != hash2 {
		t.Errorf("expected same hash regardless of JSON key order, got %s and %s", hash1, hash2)
	}
}

func TestComputeRequestBodyHash_DifferentBodyDifferentHash(t *testing.T) {
	body1 := []byte(`{"name":"a"}`)
	body2 := []byte(`{"name":"b"}`)
	hash1 := ComputeRequestBodyHash(body1, nil)
	hash2 := ComputeRequestBodyHash(body2, nil)
	if hash1 == hash2 {
		t.Errorf("expected different hashes for different bodies, got same: %s", hash1)
	}
}

func TestComputeRequestBodyHash_WithParams(t *testing.T) {
	body := []byte(`{"name":"test"}`)
	params := map[string]string{"page": "1", "limit": "10"}
	hash := ComputeRequestBodyHash(body, params)
	if hash == "" {
		t.Error("expected non-empty hash")
	}

	// Same params, same hash
	hash2 := ComputeRequestBodyHash(body, params)
	if hash != hash2 {
		t.Errorf("expected same hash for same params, got %s and %s", hash, hash2)
	}
}

func TestComputeRequestBodyHash_ParamOrderInsensitive(t *testing.T) {
	body := []byte(`{}`)
	params1 := map[string]string{"b": "2", "a": "1"}
	params2 := map[string]string{"a": "1", "b": "2"}
	hash1 := ComputeRequestBodyHash(body, params1)
	hash2 := ComputeRequestBodyHash(body, params2)
	if hash1 != hash2 {
		t.Errorf("expected same hash regardless of param order, got %s and %s", hash1, hash2)
	}
}

func TestComputeRequestBodyHash_DifferentParamsDifferentHash(t *testing.T) {
	body := []byte(`{}`)
	hash1 := ComputeRequestBodyHash(body, map[string]string{"page": "1"})
	hash2 := ComputeRequestBodyHash(body, map[string]string{"page": "2"})
	if hash1 == hash2 {
		t.Errorf("expected different hashes for different params, got same: %s", hash1)
	}
}

func TestComputeRequestBodyHash_NilVsEmptyParams(t *testing.T) {
	body := []byte(`{"name":"test"}`)
	hashNil := ComputeRequestBodyHash(body, nil)
	hashEmpty := ComputeRequestBodyHash(body, map[string]string{})
	if hashNil != hashEmpty {
		t.Errorf("expected same hash for nil and empty params, got %s and %s", hashNil, hashEmpty)
	}
}

func TestComputeRequestBodyHash_EmptyBody(t *testing.T) {
	hash := ComputeRequestBodyHash([]byte{}, nil)
	if hash == "" {
		t.Error("expected non-empty hash for empty body")
	}
}

func TestComputeRequestBodyHash_InvalidJSON_FallsBackToRawBody(t *testing.T) {
	body := []byte(`not valid json`)
	hash1 := ComputeRequestBodyHash(body, nil)
	hash2 := ComputeRequestBodyHash(body, nil)
	if hash1 != hash2 {
		t.Errorf("expected deterministic hash for invalid JSON, got %s and %s", hash1, hash2)
	}
}

func TestComputeRequestBodyHash_NestedJSON(t *testing.T) {
	body1 := []byte(`{"outer":{"b":2,"a":1}}`)
	body2 := []byte(`{"outer":{"a":1,"b":2}}`)
	hash1 := ComputeRequestBodyHash(body1, nil)
	hash2 := ComputeRequestBodyHash(body2, nil)
	if hash1 != hash2 {
		t.Errorf("expected same hash for nested JSON with reordered keys, got %s and %s", hash1, hash2)
	}
}

func TestComputeRequestBodyHash_WhitespaceInsensitive(t *testing.T) {
	compact := []byte(`{"a":1,"b":2}`)
	pretty := []byte(`{  "a" : 1 ,  "b" : 2  }`)
	hash1 := ComputeRequestBodyHash(compact, nil)
	hash2 := ComputeRequestBodyHash(pretty, nil)
	if hash1 != hash2 {
		t.Errorf("expected same hash regardless of whitespace, got %s and %s", hash1, hash2)
	}
}

// --- CanonicalizeJSON ---

func TestCanonicalizeJSON_EmptyInput(t *testing.T) {
	result, err := CanonicalizeJSON([]byte{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result, got %q", result)
	}
}

func TestCanonicalizeJSON_NilInput(t *testing.T) {
	result, err := CanonicalizeJSON(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result, got %q", result)
	}
}

func TestCanonicalizeJSON_InvalidJSON(t *testing.T) {
	_, err := CanonicalizeJSON([]byte(`{invalid`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestCanonicalizeJSON_KeyOrderNormalized(t *testing.T) {
	input1 := []byte(`{"b":2,"a":1}`)
	input2 := []byte(`{"a":1,"b":2}`)

	result1, err := CanonicalizeJSON(input1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result2, err := CanonicalizeJSON(input2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(result1) != string(result2) {
		t.Errorf("expected same canonical output, got %q and %q", result1, result2)
	}
}

func TestCanonicalizeJSON_PreservesValues(t *testing.T) {
	input := []byte(`{"name":"test","count":42,"active":true,"tags":["a","b"]}`)
	result, err := CanonicalizeJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("canonical output is not valid JSON: %v", err)
	}

	if parsed["name"] != "test" {
		t.Errorf("expected name=%q, got %v", "test", parsed["name"])
	}
	if parsed["count"] != float64(42) {
		t.Errorf("expected count=42, got %v", parsed["count"])
	}
	if parsed["active"] != true {
		t.Errorf("expected active=true, got %v", parsed["active"])
	}
}

func TestCanonicalizeJSON_NestedObjects(t *testing.T) {
	input1 := []byte(`{"outer":{"c":3,"a":1,"b":2}}`)
	input2 := []byte(`{"outer":{"a":1,"b":2,"c":3}}`)

	result1, _ := CanonicalizeJSON(input1)
	result2, _ := CanonicalizeJSON(input2)

	if string(result1) != string(result2) {
		t.Errorf("expected same canonical output for nested objects, got %q and %q", result1, result2)
	}
}

func TestCanonicalizeJSON_Arrays(t *testing.T) {
	// Array order should be preserved (arrays are ordered).
	input := []byte(`{"items":[3,1,2]}`)
	result, err := CanonicalizeJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	items := parsed["items"].([]any)
	if len(items) != 3 || items[0] != float64(3) || items[1] != float64(1) || items[2] != float64(2) {
		t.Errorf("expected array order preserved [3,1,2], got %v", items)
	}
}

func TestCanonicalizeJSON_ScalarValues(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"string", `"hello"`},
		{"number", `42`},
		{"boolean", `true`},
		{"null", `null`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CanonicalizeJSON([]byte(tt.input))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result) == 0 {
				t.Error("expected non-empty result")
			}
		})
	}
}

// --- Hash output format ---

func TestHashOutputFormat_SHA256Hex(t *testing.T) {
	actorID := "user_123"
	hash := ComputeHTTPScopeHash(&actorID, nil, "POST", "/api/v1/orders", "key-1")

	// SHA256 produces 32 bytes = 64 hex characters
	if len(hash) != 64 {
		t.Errorf("expected 64-character hex string, got length %d: %s", len(hash), hash)
	}

	for _, c := range hash {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("expected lowercase hex character, got %c in %s", c, hash)
			break
		}
	}
}
