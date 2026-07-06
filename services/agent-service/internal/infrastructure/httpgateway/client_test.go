package httpgateway

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/augno/api/services/agent-service/internal/domain"
	"github.com/augno/api/services/auth-service/pkg/types"
)

func TestClientDo_SetsInternalHeadersAndForwardsRequest(t *testing.T) {
	var (
		gotMethod   string
		gotPath     string
		gotQuery    string
		gotToken    string
		gotIdentity string
		gotBody     string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotToken = r.Header.Get(internalServiceTokenHeader)
		gotIdentity = r.Header.Get(internalIdentityHeader)
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"so_1"}`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "secret-token")
	accountID := "acct_1"
	identity := &types.Identity{
		Type:   types.IdentityActorTypeAgent,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor:  &types.IdentityActor{RelationType: types.IdentityRelationTypeInternal, ID: "adef_1", AccountID: &accountID},
	}

	out, err := client.Do(context.Background(), domain.GatewayRequest{
		Method:   http.MethodPost,
		Path:     "/v1/sales/sales-orders",
		Query:    url.Values{"foo": {"bar"}},
		Body:     json.RawMessage(`{"buyer_account_id":"acct_2"}`),
		Identity: identity,
	})
	if err != nil {
		t.Fatal(err)
	}

	if out != `{"id":"so_1"}` {
		t.Errorf("result = %q", out)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q", gotMethod)
	}
	if gotPath != "/v1/sales/sales-orders" {
		t.Errorf("path = %q", gotPath)
	}
	if gotQuery != "foo=bar" {
		t.Errorf("query = %q", gotQuery)
	}
	if gotToken != "secret-token" {
		t.Errorf("service token header = %q, want secret-token", gotToken)
	}
	if gotBody != `{"buyer_account_id":"acct_2"}` {
		t.Errorf("body = %q", gotBody)
	}

	var sent types.Identity
	if err := json.Unmarshal([]byte(gotIdentity), &sent); err != nil {
		t.Fatalf("identity header not valid JSON: %v", err)
	}
	if sent.Type != types.IdentityActorTypeAgent {
		t.Errorf("forwarded identity type = %v, want agent", sent.Type)
	}
}

func TestClientDo_ForwardsIdempotencyKeyWhenSet(t *testing.T) {
	var gotKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get(idempotencyKeyHeader)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "tok")
	_, err := client.Do(context.Background(), domain.GatewayRequest{
		Method:         http.MethodPost,
		Path:           "/v1/x",
		Identity:       &types.Identity{Type: types.IdentityActorTypeAgent},
		IdempotencyKey: "run_1:toolu_1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotKey != "run_1:toolu_1" {
		t.Errorf("idempotency key header = %q, want run_1:toolu_1", gotKey)
	}
}

func TestClientDo_OmitsIdempotencyKeyWhenEmpty(t *testing.T) {
	var hadKey bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, hadKey = r.Header[idempotencyKeyHeader]
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "tok")
	_, err := client.Do(context.Background(), domain.GatewayRequest{
		Method:   http.MethodGet,
		Path:     "/v1/x",
		Identity: &types.Identity{Type: types.IdentityActorTypeAgent},
	})
	if err != nil {
		t.Fatal(err)
	}
	if hadKey {
		t.Error("idempotency key header should not be sent when GatewayRequest.IdempotencyKey is empty")
	}
}

func TestClientDo_ReturnsErrorForNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"bad"}`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "tok")
	out, err := client.Do(context.Background(), domain.GatewayRequest{
		Method:   http.MethodGet,
		Path:     "/v1/x",
		Identity: &types.Identity{Type: types.IdentityActorTypeAgent},
	})
	// A non-2xx must surface as a Go error so the runner marks the tool result is_error — otherwise a failed write reads as a success.
	if err == nil {
		t.Fatalf("non-2xx should return a Go error, got out=%q err=nil", out)
	}
	if want := "HTTP 400"; !strings.Contains(err.Error(), want) {
		t.Errorf("error %q should contain %q", err.Error(), want)
	}
	// The response body must ride along in the error so the model can self-correct.
	if want := `{"error":"bad"}`; !strings.Contains(err.Error(), want) {
		t.Errorf("error %q should contain the response body %q", err.Error(), want)
	}
}
