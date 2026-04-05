package grpc

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/core"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func userRecordToProto(u *domain.UserRecord) *pb.UserInfo {
	if u == nil {
		return nil
	}

	info := &pb.UserInfo{
		Id:        u.ID,
		CreatedAt: timestamppb.New(u.CreatedAt),
		UpdatedAt: timestamppb.New(u.UpdatedAt),
	}

	if u.Email != nil {
		info.Email = u.Email
	}
	if u.Name != nil {
		info.Name = u.Name
	}
	if u.Username != nil {
		info.Username = u.Username
	}
	if u.ImageURL != nil {
		info.ImageUrl = u.ImageURL
	}
	if u.EmailVerified != nil {
		info.EmailVerifiedAt = timestamppb.New(*u.EmailVerified)
	}

	return info
}

func (h *gRPCHandler) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	user, apiErr := h.userSvc.GetUser(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetUserResponse{
		User: userRecordToProto(user),
	}, nil
}

func (h *gRPCHandler) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateUserParams{
		Name:     req.Name,
		ImageURL: req.ImageUrl,
	}
	if req.EmailVerifiedAt != nil {
		t := req.EmailVerifiedAt.AsTime()
		params.EmailVerified = &t
	}

	user, apiErr := h.userSvc.UpdateUser(ctx, req.Id, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateUserResponse{
		User: userRecordToProto(user),
	}, nil
}

func (h *gRPCHandler) UploadUserPhoto(ctx context.Context, req *pb.UploadUserPhotoRequest) (*pb.UploadUserPhotoResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.userSvc.UploadUserPhoto(ctx, req.Id, req.File, req.ContentType)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UploadUserPhotoResponse{
		Success: true,
	}, nil
}

func (h *gRPCHandler) GetUserPhotoURL(ctx context.Context, req *pb.GetUserPhotoURLRequest) (*pb.GetUserPhotoURLResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	url, apiErr := h.userSvc.GetUserPhotoURL(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetUserPhotoURLResponse{
		Url: url,
	}, nil
}
