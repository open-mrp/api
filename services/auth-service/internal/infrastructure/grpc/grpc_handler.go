package grpc

import (
	"context"

	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/auth"

	grpcidentity "github.com/augno/api/services/auth-service/pkg/grpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

type gRPCHandler struct {
	pb.UnimplementedAuthServiceServer

	authSvc domain.AuthSvc
}

func NewGRPCHandler(server *grpc.Server, authSvc domain.AuthSvc) *gRPCHandler {
	handler := &gRPCHandler{
		authSvc: authSvc,
	}

	pb.RegisterAuthServiceServer(server, handler)
	return handler
}

func (h *gRPCHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	result, apiErr := h.authSvc.Login(ctx, req.Identifier, req.Password)
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

	result, apiErr := h.authSvc.Register(ctx, req.Email, req.Password, req.Name)
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

	result, apiErr := h.authSvc.RefreshToken(ctx, req.RefreshToken)
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

	apiErr := h.authSvc.RequestPasswordReset(ctx, req.Identifier, req.AccountSlug)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *gRPCHandler) ResetPassword(ctx context.Context, req *pb.ResetPasswordRequest) (*pb.LoginResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	result, apiErr := h.authSvc.ResetPassword(ctx, req.Token, req.Password)
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

	apiErr := h.authSvc.RevokeRefreshToken(ctx, req.RefreshToken)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *gRPCHandler) UpdatePassword(ctx context.Context, req *pb.UpdatePasswordRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, contracts.NewMissingIdentityMetadataError()
	}

	identity, apiErr := grpcidentity.GetIdentityFromMetadata(md)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	apiErr = h.authSvc.UpdatePassword(ctx, identity.Actor.ID, req.OldPassword, req.NewPassword)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}
