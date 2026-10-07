package middleware

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	"github.com/open-mrp/api/services/api-gateway/internal/header"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/appctx"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/idempotency"
	pb "github.com/open-mrp/api/shared/proto/platform"
	"google.golang.org/grpc"
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
	if apiErr.Code != apierror.ErrorCodeRequestTooLarge {
		t.Errorf("an oversized body is a 413 request_too_large, got %q: %q", apiErr.Code, apiErr.PublicMessage)
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

// A response larger than any buffer cap must reach the client, and the cache, whole.
func TestResponseRecorder_KeepsLargeBodiesWhole(t *testing.T) {
	t.Parallel()

	body := []byte(`{"data":"` + strings.Repeat("x", 200*1024) + `"}`)
	rec := newResponseRecorder(httptest.NewRecorder())
	rec.WriteHeader(http.StatusCreated)
	for i := 0; i < len(body); i += 4096 {
		end := min(i+4096, len(body))
		if _, err := rec.Write(body[i:end]); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	if !bytes.Equal(rec.body.Bytes(), body) {
		t.Fatalf("recorded %d bytes, want %d", rec.body.Len(), len(body))
	}

	out := httptest.NewRecorder()
	rec.flush(out)
	if out.Code != http.StatusCreated {
		t.Errorf("status %d, want 201", out.Code)
	}
	if !bytes.Equal(out.Body.Bytes(), body) {
		t.Errorf("client got %d bytes, want %d", out.Body.Len(), len(body))
	}
}

// fakeIdempotencyStore keeps the platform service's contract: one row per scope hash, a differing request hash is a mismatch.
type fakeIdempotencyStore struct {
	pb.IdempotencyServiceClient

	mu   sync.Mutex
	rows map[string]*fakeIdempotencyRow
}

type fakeIdempotencyRow struct {
	id              string
	requestBodyHash string
	responseCode    *int32
	responseBody    []byte
}

func (f *fakeIdempotencyStore) ProcessIdempotencyKey(_ context.Context, in *pb.ProcessIdempotencyKeyRequest, _ ...grpc.CallOption) (*pb.ProcessIdempotencyKeyResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	row, ok := f.rows[in.ScopeHash]
	if !ok {
		row = &fakeIdempotencyRow{id: in.ScopeHash, requestBodyHash: in.RequestBodyHash}
		f.rows[in.ScopeHash] = row
		return &pb.ProcessIdempotencyKeyResponse{Result: pb.ProcessIdempotencyKeyResult_PROCESS_RESULT_NEW, IdempotencyKeyId: row.id}, nil
	}
	if row.requestBodyHash != in.RequestBodyHash {
		return &pb.ProcessIdempotencyKeyResponse{Result: pb.ProcessIdempotencyKeyResult_PROCESS_RESULT_HASH_MISMATCH}, nil
	}
	return &pb.ProcessIdempotencyKeyResponse{
		Result:           pb.ProcessIdempotencyKeyResult_PROCESS_RESULT_REPLAY,
		IdempotencyKeyId: row.id,
		ResponseCode:     row.responseCode,
		ResponseBody:     row.responseBody,
	}, nil
}

func (f *fakeIdempotencyStore) SetIdempotencyKeyResponse(_ context.Context, in *pb.SetIdempotencyKeyResponseRequest, _ ...grpc.CallOption) (*pb.SetIdempotencyKeyResponseResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, row := range f.rows {
		if row.id == in.IdempotencyKeyId {
			code := in.ResponseCode
			row.responseCode = &code
			row.responseBody = in.ResponseBody
		}
	}
	return &pb.SetIdempotencyKeyResponseResponse{Success: true}, nil
}

// patchItemInventory sends a request the way the router hands it on: route pattern and path parameters already on the context.
func patchItemInventory(handler http.HandlerFunc, itemID, key string) *httptest.ResponseRecorder {
	actorAccount := "ac_seller"
	req := httptest.NewRequest(http.MethodPatch, "/v1/catalog/items/"+itemID+"/inventory", strings.NewReader(`{"quantity":"1"}`))
	req.Header.Set(header.IdempotencyKeyHeader, key)
	ctx := appctx.WithIdentity(req.Context(), &types.Identity{
		Type:   types.IdentityActorTypeAPIKey,
		Target: &types.IdentityTarget{AccountID: actorAccount},
		Actor:  &types.IdentityActor{ID: "ak_actor", AccountID: &actorAccount, RelationType: types.IdentityRelationTypeInternal},
	})
	ctx = appctx.WithRequestLog(ctx, &appctx.RequestLog{ID: "req_" + itemID})
	ctx = appctx.WithRoutePattern(ctx, "/v1/catalog/items/{id}/inventory")
	ctx = appctx.WithPathParams(ctx, map[string]string{"id": itemID})

	w := httptest.NewRecorder()
	handler(w, req.WithContext(ctx))
	return w
}

// A key belongs to one request: reused on another item it is refused, not answered with the first item's result.
func TestIdempotencyMiddleware_KeyReusedOnAnotherResourceIsNotReplayed(t *testing.T) {
	t.Parallel()

	store := &fakeIdempotencyStore{rows: map[string]*fakeIdempotencyRow{}}
	adjusted := []string{}
	handler := IdempotencyMiddleware(&IdempotencyMiddlewareConfig{
		PlatformClient: &grpcclient.PlatformServiceClient{Client: store},
	})(func(w http.ResponseWriter, r *http.Request) {
		params, _ := appctx.GetPathParams(r.Context())
		adjusted = append(adjusted, params["id"])
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"item":"` + params["id"] + `"}`))
	})

	first := patchItemInventory(handler, "itm_a", "key-1")
	if first.Code != http.StatusOK {
		t.Fatalf("first request: status %d: %s", first.Code, first.Body.String())
	}

	retry := patchItemInventory(handler, "itm_a", "key-1")
	if retry.Code != http.StatusOK || retry.Header().Get(header.IdempotentReplayedHeader) != "true" {
		t.Fatalf("a retry of the same request replays: status %d, replayed %q", retry.Code, retry.Header().Get(header.IdempotentReplayedHeader))
	}
	if retry.Body.String() != first.Body.String() {
		t.Errorf("replayed body %s, want %s", retry.Body.String(), first.Body.String())
	}

	other := patchItemInventory(handler, "itm_b", "key-1")
	if other.Header().Get(header.IdempotentReplayedHeader) == "true" {
		t.Fatalf("the other item's request was answered with a replay: %s", other.Body.String())
	}
	if other.Code != http.StatusBadRequest || !strings.Contains(other.Body.String(), `"idempotency_error"`) {
		t.Errorf("key reused on another item: status %d: %s, want 400 idempotency_error", other.Code, other.Body.String())
	}

	if len(adjusted) != 1 || adjusted[0] != "itm_a" {
		t.Errorf("handler ran for %v, want only itm_a", adjusted)
	}
}
