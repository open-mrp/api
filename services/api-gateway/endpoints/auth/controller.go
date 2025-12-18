package authep

import (
	"context"

	"github.com/augno/api/services/api-gateway/internal/cookie"
	"github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/auth"
	"github.com/augno/api/shared/tracing"
)

const (
	AuthServiceName        = "Authentication service"
	MissingAuthApiKeyError = "This client is not authorized."
)

type AuthCtrl interface {
	Login(ctx context.Context, req *LoginRequest) (*apiresource.User, *contracts.APIError)
	Register(ctx context.Context, req *RegisterRequest) (*apiresource.User, *contracts.APIError)
	RefreshToken(ctx context.Context, req *RefreshTokenRequest) (*apiresource.EmptyResource, *contracts.APIError)
	RequestPasswordReset(ctx context.Context, req *RequestPasswordResetRequest) (*apiresource.EmptyResource, *contracts.APIError)
	ResetPassword(ctx context.Context, req *ResetPasswordRequest) (*apiresource.EmptyResource, *contracts.APIError)
	RevokeRefreshToken(ctx context.Context, req *RevokeRefreshTokenRequest) (*apiresource.EmptyResource, *contracts.APIError)
	UpdatePassword(ctx context.Context, req *UpdatePasswordRequest) (*apiresource.EmptyResource, *contracts.APIError)
}

type AuthCtrlConfig struct {
	AuthClient pb.AuthServiceClient
}

type authCtrlImpl struct {
	authClient pb.AuthServiceClient
}

var authCtrlTracer = tracing.GetTracer("api-gateway.endpoints.auth.controller")

func NewAuthCtrl(config AuthCtrlConfig) AuthCtrl {
	return &authCtrlImpl{
		authClient: config.AuthClient,
	}
}

func (m *authCtrlImpl) Login(ctx context.Context, req *LoginRequest) (*apiresource.User, *contracts.APIError) {
	ctx, span := authCtrlTracer.Start(ctx, "controller.auth.login")
	defer span.End()

	rpcCtx, cancel := grpc.PrepareRPCCtx(ctx)
	defer cancel()

	resp, err := m.authClient.Login(rpcCtx, &pb.LoginRequest{
		Identifier: req.Identifier,
		Password:   req.Password,
	})

	if apiErr := contracts.ConvertGRPCError(ctx, err, AuthServiceName); apiErr != nil {
		return nil, apiErr
	}

	cookie.SetAuthCookiesFromContext(ctx, resp.AccessToken, resp.RefreshToken)

	presented := UserPresenter(resp.User)
	return &presented, nil
}

func (m *authCtrlImpl) Register(ctx context.Context, req *RegisterRequest) (*apiresource.User, *contracts.APIError) {
	ctx, span := authCtrlTracer.Start(ctx, "controller.auth.register")
	defer span.End()

	rpcCtx, cancel := grpc.PrepareRPCCtx(ctx)
	defer cancel()

	resp, err := m.authClient.Register(rpcCtx, &pb.RegisterRequest{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
	})

	if apiErr := contracts.ConvertGRPCError(ctx, err, AuthServiceName); apiErr != nil {
		return nil, apiErr
	}

	cookie.SetAuthCookiesFromContext(ctx, resp.AccessToken, resp.RefreshToken)

	presented := UserPresenter(resp.User)
	return &presented, nil
}

func (m *authCtrlImpl) RefreshToken(ctx context.Context, req *RefreshTokenRequest) (*apiresource.EmptyResource, *contracts.APIError) {
	ctx, span := authCtrlTracer.Start(ctx, "controller.auth.refresh_token")
	defer span.End()

	rpcCtx, cancel := grpc.PrepareRPCCtx(ctx)
	defer cancel()

	resp, err := m.authClient.RefreshToken(rpcCtx, &pb.RefreshTokenRequest{
		RefreshToken: req.RefreshToken,
	})

	if apiErr := contracts.ConvertGRPCError(ctx, err, AuthServiceName); apiErr != nil {
		return nil, apiErr
	}

	cookie.SetAccessTokenCookieFromContext(ctx, resp.AccessToken)

	return &apiresource.EmptyResource{}, nil
}

func (m *authCtrlImpl) RequestPasswordReset(ctx context.Context, req *RequestPasswordResetRequest) (*apiresource.EmptyResource, *contracts.APIError) {
	ctx, span := authCtrlTracer.Start(ctx, "controller.auth.request_password_reset")
	defer span.End()

	rpcCtx, cancel := grpc.PrepareRPCCtx(ctx)
	defer cancel()

	_, err := m.authClient.RequestPasswordReset(rpcCtx, &pb.RequestPasswordResetRequest{
		Identifier:  req.Identifier,
		AccountSlug: req.AccountSlug,
	})

	if apiErr := contracts.ConvertGRPCError(ctx, err, AuthServiceName); apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func (m *authCtrlImpl) ResetPassword(ctx context.Context, req *ResetPasswordRequest) (*apiresource.EmptyResource, *contracts.APIError) {
	ctx, span := authCtrlTracer.Start(ctx, "controller.auth.reset_password")
	defer span.End()

	// Use longer timeout for password operations (hashing can be slow)
	rpcCtx, cancel := grpc.PrepareRPCCtxWithTimeout(ctx, grpc.PasswordOperationTimeout)
	defer cancel()

	resp, err := m.authClient.ResetPassword(rpcCtx, &pb.ResetPasswordRequest{
		Token:    req.Token,
		Password: req.Password,
	})

	if apiErr := contracts.ConvertGRPCError(ctx, err, AuthServiceName); apiErr != nil {
		return nil, apiErr
	}

	cookie.SetAuthCookiesFromContext(ctx, resp.AccessToken, resp.RefreshToken)

	return &apiresource.EmptyResource{}, nil
}

func (m *authCtrlImpl) RevokeRefreshToken(ctx context.Context, req *RevokeRefreshTokenRequest) (*apiresource.EmptyResource, *contracts.APIError) {
	ctx, span := authCtrlTracer.Start(ctx, "controller.auth.revoke_refresh_token")
	defer span.End()

	rpcCtx, cancel := grpc.PrepareRPCCtx(ctx)
	defer cancel()

	_, err := m.authClient.RevokeRefreshToken(rpcCtx, &pb.RevokeRefreshTokenRequest{
		RefreshToken: req.RefreshToken,
	})

	if apiErr := contracts.ConvertGRPCError(ctx, err, AuthServiceName); apiErr != nil {
		return nil, apiErr
	}

	cookie.ClearRefreshTokenCookieFromContext(ctx)

	return &apiresource.EmptyResource{}, nil
}

func (m *authCtrlImpl) UpdatePassword(ctx context.Context, req *UpdatePasswordRequest) (*apiresource.EmptyResource, *contracts.APIError) {
	ctx, span := authCtrlTracer.Start(ctx, "controller.auth.update_password")
	defer span.End()

	// Use longer timeout for password operations (hashing can be slow)
	rpcCtx, cancel := grpc.PrepareRPCCtxWithTimeout(ctx, grpc.PasswordOperationTimeout)
	defer cancel()

	_, err := m.authClient.UpdatePassword(rpcCtx, &pb.UpdatePasswordRequest{
		OldPassword: req.OldPassword,
		NewPassword: req.NewPassword,
	})

	if apiErr := contracts.ConvertGRPCError(ctx, err, AuthServiceName); apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}
