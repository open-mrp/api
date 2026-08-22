package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	"github.com/open-mrp/api/services/api-gateway/internal/header"
	pb "github.com/open-mrp/api/shared/proto/auth"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// fakeAuthClient stubs pb.AuthServiceClient. Embedding the interface gives nil
// implementations for the methods we don't exercise.
type fakeAuthClient struct {
	pb.AuthServiceClient
	called   bool
	gotToken string
}

func (f *fakeAuthClient) ValidateCredential(_ context.Context, in *pb.Credential, _ ...grpc.CallOption) (*pb.Identity, error) {
	f.called = true
	f.gotToken = in.GetToken()
	// No real credential was supplied, so authentication fails.
	return nil, status.Error(codes.Unauthenticated, "invalid credentials")
}

// TestPublicAuthMiddlewareIgnoresInternalHeaders is the regression guard for the
// trust boundary: a request to the PUBLIC listener that forges the internal
// identity + service-token headers must NOT be authenticated as that identity.
// The public AuthMiddleware never reads the internal headers; it still demands a
// real credential validated by auth-service. So the forged request is rejected
// and the internal headers are never treated as a token.
func TestPublicAuthMiddlewareIgnoresInternalHeaders(t *testing.T) {
	fake := &fakeAuthClient{}
	mw := AuthMiddleware(&AuthMiddlewareConfig{
		AuthClient: &grpcclient.AuthServiceClient{Client: fake},
	})

	called := false
	handler := mw(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/sales/sales-orders", nil)
	// Forge exactly the headers the internal listener would trust. The public
	// path must give them no meaning.
	req.Header.Set(header.InternalIdentityHeader, `{"Type":"agent","Target":{"AccountID":"acct_victim"},"Actor":{"RelationType":"internal","ID":"adef_x","AccountID":"acct_victim"}}`)
	req.Header.Set(header.InternalServiceTokenHeader, "anything")

	rr := httptest.NewRecorder()
	handler(rr, req)

	if called {
		t.Error("downstream handler ran — forged internal headers must not authenticate on the public listener")
	}
	if rr.Code == http.StatusOK {
		t.Errorf("status = %d, want a non-200 auth rejection", rr.Code)
	}
	if !fake.called {
		t.Error("AuthMiddleware did not call ValidateCredential — it must still require a real credential")
	}
	if fake.gotToken != "" {
		t.Errorf("ValidateCredential token = %q, want empty — the internal headers must not be used as a credential", fake.gotToken)
	}
}
