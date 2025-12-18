package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apicontext "github.com/augno/api/services/api-gateway/internal/context"
	"github.com/augno/api/services/api-gateway/internal/domain"
	pb "github.com/augno/api/shared/proto/auth"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type stubAuthClient struct {
	identity *pb.Identity
	err      error
}

func (s *stubAuthClient) ValidateCredential(ctx context.Context, in *pb.Credential, opts ...grpc.CallOption) (*pb.Identity, error) {
	return s.identity, s.err
}

// Unused methods for the AuthServiceClient interface
func (s *stubAuthClient) Login(ctx context.Context, in *pb.LoginRequest, opts ...grpc.CallOption) (*pb.LoginResponse, error) {
	return nil, nil
}
func (s *stubAuthClient) Register(ctx context.Context, in *pb.RegisterRequest, opts ...grpc.CallOption) (*pb.LoginResponse, error) {
	return nil, nil
}
func (s *stubAuthClient) RefreshToken(ctx context.Context, in *pb.RefreshTokenRequest, opts ...grpc.CallOption) (*pb.RefreshTokenResponse, error) {
	return nil, nil
}
func (s *stubAuthClient) RequestPasswordReset(ctx context.Context, in *pb.RequestPasswordResetRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}
func (s *stubAuthClient) ResetPassword(ctx context.Context, in *pb.ResetPasswordRequest, opts ...grpc.CallOption) (*pb.LoginResponse, error) {
	return nil, nil
}
func (s *stubAuthClient) RevokeRefreshToken(ctx context.Context, in *pb.RevokeRefreshTokenRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}
func (s *stubAuthClient) UpdatePassword(ctx context.Context, in *pb.UpdatePasswordRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func TestAuthMiddlewareSetsActorAndIdentityTypes(t *testing.T) {
	targetAccount := "acct-1"
	stubClient := &stubAuthClient{
		identity: &pb.Identity{
			Type:            pb.IdentityType_IDENTITY_TYPE_API_KEY,
			TargetAccountId: &targetAccount,
			Actor:           &pb.IdentityActor{Type: pb.IdentityActorType_IDENTITY_ACTOR_TYPE_CUSTOMER, Id: "actor-1"},
			AccountMode:     pb.AccountMode_ACCOUNT_MODE_PRODUCTION,
		},
	}

	rl := &domain.RequestLog{}
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer token")
	ctx := context.WithValue(req.Context(), apicontext.RequestLogKey, rl)
	req = req.WithContext(ctx)

	config := AuthMiddlewareConfig{
		AuthClient: &grpcclient.AuthServiceClient{Client: stubClient},
	}

	rec := httptest.NewRecorder()
	handler := AuthMiddleware(config)(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler(rec, req)

	if rl.ActorType != "customer" {
		t.Fatalf("expected actor_type customer, got %q", rl.ActorType)
	}
	if rl.IdentityType != "api_key" {
		t.Fatalf("expected identity_type api_key, got %q", rl.IdentityType)
	}
	if rl.ActorID != "actor-1" {
		t.Fatalf("expected actor_id actor-1, got %q", rl.ActorID)
	}
	if rl.AccountID != targetAccount {
		t.Fatalf("expected account_id %s, got %q", targetAccount, rl.AccountID)
	}
}
