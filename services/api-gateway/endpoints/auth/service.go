package authep

import (
	"context"
	"net/http"

	"github.com/augno/api/services/api-gateway/internal/cookie"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/appctx"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/auth"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	AuthServiceName        = "Authentication service"
	MissingAuthApiKeyError = "This client is not authorized."
)

type AuthSvc interface {
	Login(ctx context.Context, req *LoginRequest) (*apiresource.User, *apierror.APIError)
	Register(ctx context.Context, req *RegisterRequest) (*apiresource.User, *apierror.APIError)
	RefreshToken(ctx context.Context, req *RefreshTokenRequest) (*apiresource.EmptyResource, *apierror.APIError)
	RequestPasswordReset(ctx context.Context, req *RequestPasswordResetRequest) (*apiresource.EmptyResource, *apierror.APIError)
	ResetPassword(ctx context.Context, req *ResetPasswordRequest) (*apiresource.EmptyResource, *apierror.APIError)
	RevokeRefreshToken(ctx context.Context, req *RevokeRefreshTokenRequest) (*apiresource.EmptyResource, *apierror.APIError)
	UpdatePassword(ctx context.Context, req *UpdatePasswordRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type AuthSvcConfig struct {
	AuthClient pb.AuthServiceClient
}

type authSvcImpl struct {
	authClient pb.AuthServiceClient
}

var authSvcTracer = tracing.GetTracer("api-gateway.endpoints.auth.service")

func NewAuthSvc(config AuthSvcConfig) AuthSvc {
	return &authSvcImpl{
		authClient: config.AuthClient,
	}
}

func (m *authSvcImpl) Login(ctx context.Context, req *LoginRequest) (*apiresource.User, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, authSvcTracer, "service.auth.login", AuthServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.LoginResponse, error) {
			return m.authClient.Login(ctx, &pb.LoginRequest{
				Identifier: req.Identifier,
				Password:   req.Password,
			}, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	appctx.AddCookies(ctx, cookie.MakeAuthCookies(ctx, resp.AccessToken, resp.RefreshToken))

	presented := UserPresenter(resp.User)
	return &presented, nil
}

func (m *authSvcImpl) Register(ctx context.Context, req *RegisterRequest) (*apiresource.User, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, authSvcTracer, "service.auth.register", AuthServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.LoginResponse, error) {
			return m.authClient.Register(ctx, &pb.RegisterRequest{
				Email:    req.Email,
				Password: req.Password,
				Name:     req.Name,
			}, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	appctx.AddCookies(ctx, cookie.MakeAuthCookies(ctx, resp.AccessToken, resp.RefreshToken))

	presented := UserPresenter(resp.User)
	return &presented, nil
}

func (m *authSvcImpl) RefreshToken(ctx context.Context, req *RefreshTokenRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, authSvcTracer, "service.auth.refresh_token", AuthServiceName,
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
	_, apiErr := grpcutil.CallRPC(ctx, authSvcTracer, "service.auth.request_password_reset", AuthServiceName,
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
	resp, apiErr := grpcutil.CallRPC(ctx, authSvcTracer, "service.auth.reset_password", AuthServiceName,
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
	_, apiErr := grpcutil.CallRPC(ctx, authSvcTracer, "service.auth.revoke_refresh_token", AuthServiceName,
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
	_, apiErr := grpcutil.CallRPC(ctx, authSvcTracer, "service.auth.update_password", AuthServiceName,
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
