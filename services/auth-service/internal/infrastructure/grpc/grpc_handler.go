package grpc

import (
	"context"
	"time"

	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/auth"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type gRPCHandler struct {
	pb.UnimplementedAuthServiceServer

	authSvc      domain.AuthSvc
	userSvc      domain.UserSvc
	tokenSvc     domain.TokenSvc
	passwordSvc  domain.PasswordSvc
	apiKeySvc    domain.APIKeySvc
	docAPIKeySvc domain.DocAPIKeySvc
}

func NewGRPCHandler(server *grpc.Server, authSvc domain.AuthSvc, userSvc domain.UserSvc, tokenSvc domain.TokenSvc, passwordSvc domain.PasswordSvc, apiKeySvc domain.APIKeySvc, docAPIKeySvc domain.DocAPIKeySvc) *gRPCHandler {
	handler := &gRPCHandler{
		authSvc:      authSvc,
		userSvc:      userSvc,
		tokenSvc:     tokenSvc,
		passwordSvc:  passwordSvc,
		apiKeySvc:    apiKeySvc,
		docAPIKeySvc: docAPIKeySvc,
	}

	pb.RegisterAuthServiceServer(server, handler)
	return handler
}

func (h *gRPCHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	result, apiErr := h.userSvc.Login(ctx, req.Identifier, req.Password)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.LoginResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		User:         result.User.ToProto(),
	}, nil
}

func (h *gRPCHandler) ValidateCredential(ctx context.Context, cred *pb.Credential) (*pb.Identity, error) {
	if cred == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	result, apiErr := h.authSvc.ValidateCredential(ctx, cred.Token, cred.TargetAccountId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return result.ToProto(), nil
}

func (h *gRPCHandler) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.LoginResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	result, apiErr := h.userSvc.Register(ctx, domain.RegisterInput{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.LoginResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		User:         result.User.ToProto(),
	}, nil
}

func (h *gRPCHandler) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	result, apiErr := h.tokenSvc.RefreshToken(ctx, req.RefreshToken)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.RefreshTokenResponse{
		AccessToken: result.AccessToken,
	}, nil
}

func (h *gRPCHandler) RequestPasswordReset(ctx context.Context, req *pb.RequestPasswordResetRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	apiErr := h.passwordSvc.RequestPasswordReset(ctx, req.Identifier, req.AccountSlug)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *gRPCHandler) ResetPassword(ctx context.Context, req *pb.ResetPasswordRequest) (*pb.LoginResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	result, apiErr := h.passwordSvc.ResetPassword(ctx, req.Token, req.Password)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.LoginResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		User:         result.User.ToProto(),
	}, nil
}

func (h *gRPCHandler) RevokeRefreshToken(ctx context.Context, req *pb.RevokeRefreshTokenRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	apiErr := h.tokenSvc.RevokeRefreshToken(ctx, req.RefreshToken)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *gRPCHandler) UpdatePassword(ctx context.Context, req *pb.UpdatePasswordRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok {
		return nil, contracts.NewMissingIdentityMetadataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	apiErr := h.passwordSvc.UpdatePassword(ctx, identity.Actor.ID, req.OldPassword, req.NewPassword)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *gRPCHandler) CreateAPIKey(ctx context.Context, req *pb.CreateAPIKeyRequest) (*pb.CreateAPIKeyResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		t := req.ExpiresAt.AsTime()
		if !t.IsZero() {
			expiresAt = &t
		}
	}

	result, apiErr := h.apiKeySvc.CreateAPIKey(ctx, domain.CreateAPIKeyInput{
		RoleID:    req.RoleId,
		Name:      req.Name,
		ExpiresAt: expiresAt,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateAPIKeyResponse{
		ApiKeySecret: result.APIKeySecret,
		ApiKey:       result.APIKey.ToProto(),
	}, nil
}

func (h *gRPCHandler) ListAPIKeys(ctx context.Context, req *pb.ListAPIKeysRequest) (*pb.ListAPIKeysResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	statuses := make([]constants.APIKeyStatus, len(req.Statuses))
	for i, s := range req.Statuses {
		statuses[i] = constants.APIKeyStatus(s)
	}

	result, apiErr := h.apiKeySvc.ListAPIKeys(ctx, req.Cursor, req.Limit, req.Query, statuses)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbKeys := make([]*pb.APIKeyInfo, len(result.APIKeys))
	for i, key := range result.APIKeys {
		pbKeys[i] = key.ToProto()
	}

	return &pb.ListAPIKeysResponse{
		ApiKeys:    pbKeys,
		HasMore:    result.HasMore,
		NextCursor: result.NextCursor,
	}, nil
}

func (h *gRPCHandler) RevokeAPIKey(ctx context.Context, req *pb.RevokeAPIKeyRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	apiErr := h.apiKeySvc.RevokeAPIKey(ctx, domain.RevokeAPIKeyInput{
		APIKeyID: req.ApiKeyId,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *gRPCHandler) RotateAPIKey(ctx context.Context, req *pb.RotateAPIKeyRequest) (*pb.RotateAPIKeyResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	input := domain.RotateAPIKeyInput{
		APIKeyID: req.ApiKeyId,
	}
	if req.ExpiresAt != nil {
		t := req.ExpiresAt.AsTime()
		if !t.IsZero() {
			input.ExpiresAt = &t
		}
	}

	result, apiErr := h.apiKeySvc.RotateAPIKey(ctx, input)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.RotateAPIKeyResponse{
		ApiKeySecret: result.APIKeySecret,
		ApiKey:       result.APIKey.ToProto(),
	}, nil
}

func (h *gRPCHandler) GetOrCreateDocAPIKey(ctx context.Context, req *pb.GetOrCreateDocAPIKeyRequest) (*pb.GetOrCreateDocAPIKeyResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	result, apiErr := h.docAPIKeySvc.GetOrCreateDocAPIKey(ctx)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetOrCreateDocAPIKeyResponse{
		ApiKeySecret: result.APIKeySecret,
		ApiKey:       result.APIKey.ToProto(),
	}, nil
}
