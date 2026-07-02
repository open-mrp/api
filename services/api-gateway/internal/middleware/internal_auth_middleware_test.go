package middleware

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/augno/api/services/api-gateway/internal/header"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/appctx"
)

const testServiceToken = "test-internal-secret"

func agentIdentityJSON(t *testing.T) string {
	t.Helper()
	accountID := "acct_123"
	id := &types.Identity{
		Type:   types.IdentityActorTypeAgent,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "adef_123",
			AccountID:    &accountID,
		},
	}
	b, err := json.Marshal(id)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func userIdentityJSON(t *testing.T) string {
	t.Helper()
	accountID := "acct_123"
	id := &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "user_123",
			AccountID:    &accountID,
		},
	}
	b, err := json.Marshal(id)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestInternalAuthMiddleware(t *testing.T) {
	mw := InternalAuthMiddleware(&InternalAuthMiddlewareConfig{ServiceToken: testServiceToken})

	tests := []struct {
		name       string
		token      string
		identity   string
		wantCalled bool
		wantStatus int
	}{
		{
			name:       "valid token and agent identity passes",
			token:      testServiceToken,
			identity:   agentIdentityJSON(t),
			wantCalled: true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing token is rejected",
			token:      "",
			identity:   agentIdentityJSON(t),
			wantCalled: false,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "wrong token is rejected",
			token:      "nope",
			identity:   agentIdentityJSON(t),
			wantCalled: false,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "missing identity is rejected",
			token:      testServiceToken,
			identity:   "",
			wantCalled: false,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "malformed identity is rejected",
			token:      testServiceToken,
			identity:   "{not json",
			wantCalled: false,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "non-agent identity is rejected",
			token:      testServiceToken,
			identity:   userIdentityJSON(t),
			wantCalled: false,
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			var gotIdentity *types.Identity
			handler := mw(func(w http.ResponseWriter, r *http.Request) {
				called = true
				gotIdentity, _ = appctx.GetIdentityFromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/v1/sales/sales-orders", nil)
			if tc.token != "" {
				req.Header.Set(header.InternalServiceTokenHeader, tc.token)
			}
			if tc.identity != "" {
				req.Header.Set(header.InternalIdentityHeader, tc.identity)
			}
			rr := httptest.NewRecorder()
			handler(rr, req)

			if called != tc.wantCalled {
				t.Errorf("next called = %v, want %v", called, tc.wantCalled)
			}
			if rr.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tc.wantStatus)
			}
			if tc.wantCalled {
				if gotIdentity == nil {
					t.Fatal("expected identity in context")
				}
				if gotIdentity.Type != types.IdentityActorTypeAgent {
					t.Errorf("identity type = %v, want agent", gotIdentity.Type)
				}
			}
		})
	}
}

func TestInternalAuthMiddleware_SkipsHealthz(t *testing.T) {
	mw := InternalAuthMiddleware(&InternalAuthMiddlewareConfig{ServiceToken: testServiceToken})
	called := false
	handler := mw(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)

	if !called {
		t.Error("/healthz should bypass internal auth")
	}
}

// TestInternalAuthMiddleware_PopulatesRequestLog asserts that an agent request
// flowing through the production middleware order (logging is OUTER, internal-auth
// is INNER) produces a fully-attributed, saved request log. This is the invariant
// that makes agent traffic on the internal listener show up in the request-log
// list: the account-scoped list query matches on (account_id OR target_account_id),
// so both must be populated from the agent identity before the log is saved.
func TestInternalAuthMiddleware_PopulatesRequestLog(t *testing.T) {
	logger := log.New(io.Discard, "", 0)
	saver := &stubSaver{}

	internalAuth := InternalAuthMiddleware(&InternalAuthMiddlewareConfig{ServiceToken: testServiceToken})
	// Compose exactly as the router does: logging wraps internal-auth, so the
	// request log created by logging is in context when internal-auth populates it,
	// and logging's deferred Save sees the populated struct.
	chain := LoggingMiddleware(logger, internalAuth(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), saver, nil, 0)

	req := httptest.NewRequest(http.MethodGet, "/v1/sales/sales-orders", nil)
	req.Header.Set(header.InternalServiceTokenHeader, testServiceToken)
	req.Header.Set(header.InternalIdentityHeader, agentIdentityJSON(t))
	rr := httptest.NewRecorder()

	chain.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	rl := saver.savedRL
	if rl == nil {
		t.Fatal("expected request log to be saved for an agent request")
	}
	if rl.AccountID == nil || *rl.AccountID != "acct_123" {
		t.Errorf("AccountID = %v, want acct_123", rl.AccountID)
	}
	if rl.TargetAccountID == nil || *rl.TargetAccountID != "acct_123" {
		t.Errorf("TargetAccountID = %v, want acct_123 (else the account-scoped list query hides the log)", rl.TargetAccountID)
	}
	if rl.ActorID == nil || *rl.ActorID != "adef_123" {
		t.Errorf("ActorID = %v, want adef_123", rl.ActorID)
	}
	if rl.ActorType == nil || *rl.ActorType != string(types.IdentityRelationTypeInternal) {
		t.Errorf("ActorType = %v, want %s", rl.ActorType, types.IdentityRelationTypeInternal)
	}
	if rl.IdentityType == nil || *rl.IdentityType != string(types.IdentityActorTypeAgent) {
		t.Errorf("IdentityType = %v, want %s", rl.IdentityType, types.IdentityActorTypeAgent)
	}
}

func TestInternalAuthMiddleware_PanicsWithoutToken(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected panic when service token is empty")
		}
	}()
	InternalAuthMiddleware(&InternalAuthMiddlewareConfig{ServiceToken: ""})
}
