package authep

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/augno/api/services/api-gateway/internal/cookie"
	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	"github.com/augno/api/services/api-gateway/internal/middleware"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/auth"
	corepb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Identifier-based throttle for failed login attempts. This sits on top of the
// IP-based RateLimitMiddleware so that an attacker rotating source addresses
// cannot brute-force a single account. Limits are per gateway pod.
const (
	loginFailureLimit  = 10
	loginFailureWindow = 5 * time.Minute
)

const (
	MissingAuthApiKeyError = "This client is not authorized."
)

type AuthSvc interface {
	Login(ctx context.Context, req *LoginRequest) (*apiresource.User, *apierror.APIError)
	Register(ctx context.Context, req *RegisterRequest) (*apiresource.User, *apierror.APIError)
	MagicLogin(ctx context.Context, req *MagicLoginRequest) (*apiresource.User, *apierror.APIError)
	RefreshToken(ctx context.Context, req *RefreshTokenRequest) (*apiresource.EmptyResource, *apierror.APIError)
	RequestPasswordReset(ctx context.Context, req *RequestPasswordResetRequest) (*apiresource.EmptyResource, *apierror.APIError)
	ResetPassword(ctx context.Context, req *ResetPasswordRequest) (*apiresource.EmptyResource, *apierror.APIError)
	RevokeRefreshToken(ctx context.Context, req *RevokeRefreshTokenRequest) (*apiresource.EmptyResource, *apierror.APIError)
	UpdatePassword(ctx context.Context, req *UpdatePasswordRequest) (*apiresource.EmptyResource, *apierror.APIError)
	UpdateScannerPassword(ctx context.Context, req *UpdateScannerPasswordRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type AuthSvcConfig struct {
	AuthClient pb.AuthServiceClient
	CoreClient corepb.CoreServiceClient
}

type authSvcImpl struct {
	authClient          pb.AuthServiceClient
	coreClient          corepb.CoreServiceClient
	loginFailureLimiter *middleware.RateLimiter
}

var authSvcTracer = tracing.GetTracer("api-gateway.endpoints.auth.service")

func (c *AuthSvcConfig) validate() error {
	if c.AuthClient == nil {
		return fmt.Errorf("auth endpoint service: auth client is required")
	}
	if c.CoreClient == nil {
		return fmt.Errorf("auth endpoint service: core client is required")
	}
	return nil
}

func NewAuthSvc(config *AuthSvcConfig) AuthSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &authSvcImpl{
		authClient:          config.AuthClient,
		coreClient:          config.CoreClient,
		loginFailureLimiter: middleware.NewRateLimiter(loginFailureLimit, loginFailureWindow),
	}
}

// loginThrottleKey normalizes an identifier so that case and surrounding
// whitespace do not let an attacker fan their attempts across multiple
// rate-limit buckets for the same account.
func loginThrottleKey(identifier string) string {
	return "login:" + strings.ToLower(strings.TrimSpace(identifier))
}

func (m *authSvcImpl) Login(ctx context.Context, req *LoginRequest) (*apiresource.User, *apierror.APIError) {
	throttleKey := loginThrottleKey(req.Identifier)
	if allowed, _ := m.loginFailureLimiter.Check(throttleKey); !allowed {
		return nil, apierror.NewRateLimitExceededError("Too many failed login attempts for this account. Please try again later.")
	}

	resp, apiErr := grpcutil.CallRPC(ctx, authSvcTracer, "service.auth.login", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.LoginResponse, error) {
			return m.authClient.Login(ctx, &pb.LoginRequest{
				Identifier: req.Identifier,
				Password:   req.Password,
			}, opts...)
		})

	if apiErr != nil {
		m.loginFailureLimiter.RecordFailure(throttleKey)
		return nil, apiErr
	}

	appctx.AddCookies(ctx, cookie.MakeAuthCookies(ctx, resp.AccessToken, resp.RefreshToken))

	presented := userFromAuthProto(resp.User)
	return &presented, nil
}

func (m *authSvcImpl) Register(ctx context.Context, req *RegisterRequest) (*apiresource.User, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, authSvcTracer, "service.auth.register", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.LoginResponse, error) {
			return m.authClient.Register(ctx, &pb.RegisterRequest{
				Email:       req.Email,
				Password:    req.Password,
				Name:        req.Name,
				AccountSlug: req.AccountSlug,
			}, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	appctx.AddCookies(ctx, cookie.MakeAuthCookies(ctx, resp.AccessToken, resp.RefreshToken))

	presented := userFromAuthProto(resp.User)
	return &presented, nil
}

func (m *authSvcImpl) MagicLogin(ctx context.Context, req *MagicLoginRequest) (*apiresource.User, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, authSvcTracer, "service.auth.magic_login", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.LoginResponse, error) {
			return m.authClient.MagicLogin(ctx, &pb.MagicLoginRequest{
				Token: req.Token,
			}, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	appctx.AddCookies(ctx, cookie.MakeAuthCookies(ctx, resp.AccessToken, resp.RefreshToken))

	presented := userFromAuthProto(resp.User)
	return &presented, nil
}

func (m *authSvcImpl) RefreshToken(ctx context.Context, req *RefreshTokenRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, authSvcTracer, "service.auth.refresh_token", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.RefreshTokenResponse, error) {
			return m.authClient.RefreshToken(ctx, &pb.RefreshTokenRequest{
				RefreshToken: req.RefreshToken,
			}, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	appctx.AddCookies(ctx, []*http.Cookie{cookie.MakeAccessTokenCookie(ctx, resp.AccessToken)})

	return &apiresource.EmptyResource{}, nil
}

func (m *authSvcImpl) RequestPasswordReset(ctx context.Context, req *RequestPasswordResetRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	_, apiErr := grpcutil.CallRPC(ctx, authSvcTracer, "service.auth.request_password_reset", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.authClient.RequestPasswordReset(ctx, &pb.RequestPasswordResetRequest{
				Identifier:  req.Identifier,
				AccountSlug: req.AccountSlug,
			}, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func (m *authSvcImpl) ResetPassword(ctx context.Context, req *ResetPasswordRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, authSvcTracer, "service.auth.reset_password", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.LoginResponse, error) {
			return m.authClient.ResetPassword(ctx, &pb.ResetPasswordRequest{
				Token:    req.Token,
				Password: req.Password,
			}, opts...)
		}, grpcutil.WithTimeout(grpcutil.PasswordOperationTimeout))

	if apiErr != nil {
		return nil, apiErr
	}

	appctx.AddCookies(ctx, cookie.MakeAuthCookies(ctx, resp.AccessToken, resp.RefreshToken))

	return &apiresource.EmptyResource{}, nil
}

func (m *authSvcImpl) RevokeRefreshToken(ctx context.Context, req *RevokeRefreshTokenRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	_, apiErr := grpcutil.CallRPC(ctx, authSvcTracer, "service.auth.revoke_refresh_token", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.authClient.RevokeRefreshToken(ctx, &pb.RevokeRefreshTokenRequest{
				RefreshToken: req.RefreshToken,
			}, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	appctx.AddCookies(ctx, cookie.MakeClearAuthCookies(ctx))

	return &apiresource.EmptyResource{}, nil
}

func (m *authSvcImpl) UpdatePassword(ctx context.Context, req *UpdatePasswordRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	_, apiErr := grpcutil.CallRPC(ctx, authSvcTracer, "service.auth.update_password", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.authClient.UpdatePassword(ctx, &pb.UpdatePasswordRequest{
				OldPassword: req.OldPassword,
				NewPassword: req.NewPassword,
			}, opts...)
		}, grpcutil.WithTimeout(grpcutil.PasswordOperationTimeout))

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func (m *authSvcImpl) UpdateScannerPassword(ctx context.Context, req *UpdateScannerPasswordRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	_, apiErr := grpcutil.CallRPC(ctx, authSvcTracer, "service.auth.update_scanner_password", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.UpdateAccountUserPassword(ctx, &corepb.UpdateAccountUserPasswordRequest{
				AccountUserId:     req.AccountUserID,
				RequesterPassword: req.RequesterPassword,
				NewPassword:       req.NewPassword,
			}, opts...)
		}, grpcutil.WithTimeout(grpcutil.PasswordOperationTimeout))

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func userFromAuthProto(user *pb.User) apiresource.User {
	if user == nil {
		return apiresource.User{}
	}

	return apiresource.User{
		ID:              user.Id,
		Object:          constants.ObjectTypeUser,
		Email:           user.Email,
		Name:            user.Name,
		Username:        user.Username,
		ImageUrl:        user.ImageUrl,
		EmailVerifiedAt: grpcutil.TimestampToTimePtr(user.EmailVerified),
		CreatedAt:       grpcutil.TimestampToTime(user.CreatedAt),
		UpdatedAt:       grpcutil.TimestampToTime(user.UpdatedAt),
	}
}
