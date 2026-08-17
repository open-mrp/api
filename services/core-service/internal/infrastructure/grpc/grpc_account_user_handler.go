package grpc

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/field"
	pb "github.com/augno/api/shared/proto/core"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (h *gRPCHandler) ListAccountUsers(ctx context.Context, req *pb.ListAccountUsersRequest) (*pb.ListAccountUsersResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	result, apiErr := h.accountUserSvc.ListAccountUsers(ctx, domain.ListAccountUsersParams{
		Cursor:               req.Cursor,
		Limit:                req.Limit,
		Query:                req.Query,
		RoleType:             req.RoleType,
		IsCommissionEligible: req.IsCommissionEligible,
		IncludeRemoved:       req.IncludeRemoved,
		Includes:             req.Includes,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	items := make([]*pb.AccountUserDetail, len(result.Items))
	for i, item := range result.Items {
		items[i] = accountUserDetailToProto(item)
	}

	return &pb.ListAccountUsersResponse{
		AccountUsers: items,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
		TotalCount: result.TotalCount,
	}, nil
}

func (h *gRPCHandler) GetAccountUser(ctx context.Context, req *pb.GetAccountUserRequest) (*pb.GetAccountUserResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	detail, apiErr := h.accountUserSvc.GetAccountUser(ctx, req.AccountUserId, req.Includes)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetAccountUserResponse{
		AccountUser: accountUserDetailToProto(detail),
	}, nil
}

func (h *gRPCHandler) CreateAccountUser(ctx context.Context, req *pb.CreateAccountUserRequest) (*pb.CreateAccountUserResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	detail, apiErr := h.accountUserSvc.CreateAccountUser(ctx, domain.CreateAccountUserParams{
		Name:                    req.Name,
		Email:                   req.Email,
		Username:                req.Username,
		Password:                req.Password,
		RoleID:                  req.RoleId,
		DepartmentID:            req.DepartmentId,
		IsCommissionEligible:    req.IsCommissionEligible,
		NotificationPreferences: notificationPrefsToDomain(req.NotificationPreferences),
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateAccountUserResponse{
		AccountUser: accountUserDetailToProto(detail),
	}, nil
}

func (h *gRPCHandler) UpdateAccountUser(ctx context.Context, req *pb.UpdateAccountUserRequest) (*pb.UpdateAccountUserResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	// A nil slice ("no change") is distinguished from an explicit empty list, but proto3 repeated fields collapse both to an empty slice in the generated code. We treat any non-nil slice with at least one item as an update; callers that want "no change" simply omit the field.
	var prefs []domain.NotificationPreferenceItem
	if len(req.NotificationPreferences) > 0 {
		prefs = notificationPrefsToDomain(req.NotificationPreferences)
	}

	detail, apiErr := h.accountUserSvc.UpdateAccountUser(ctx, domain.UpdateAccountUserParams{
		AccountUserID:           req.AccountUserId,
		Name:                    req.Name,
		Email:                   req.Email,
		Username:                req.Username,
		RoleID:                  field.StringClearableFromProto(req.RoleId),
		DepartmentID:            field.StringClearableFromProto(req.DepartmentId),
		IsCommissionEligible:    field.SomePtr(req.IsCommissionEligible),
		NotificationPreferences: prefs,
	}, req.Includes)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateAccountUserResponse{
		AccountUser: accountUserDetailToProto(detail),
	}, nil
}

func (h *gRPCHandler) UpdateAccountUserStatus(ctx context.Context, req *pb.UpdateAccountUserStatusRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.accountUserSvc.UpdateAccountUserStatus(ctx, req.AccountUserId, constants.AccountUserStatus(req.StatusCode))
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *gRPCHandler) UpdateAccountUserPassword(ctx context.Context, req *pb.UpdateAccountUserPasswordRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	apiErr := h.accountUserSvc.UpdateAccountUserPassword(ctx, req.AccountUserId, req.RequesterPassword, req.NewPassword)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func notificationPrefsToDomain(in []*pb.NotificationPreferenceItem) []domain.NotificationPreferenceItem {
	out := make([]domain.NotificationPreferenceItem, len(in))
	for i, p := range in {
		out[i] = domain.NotificationPreferenceItem{
			NotificationTypeCode: p.NotificationTypeCode,
			Enabled:              p.Enabled,
		}
	}
	return out
}

func accountUserDetailToProto(d *domain.AccountUserDetail) *pb.AccountUserDetail {
	if d == nil {
		return nil
	}

	proto := &pb.AccountUserDetail{
		Id:                   d.ID,
		UserId:               d.UserID,
		Name:                 d.Name,
		Email:                d.Email,
		Username:             d.Username,
		ImageUrl:             d.ImageURL,
		EmailVerified:        d.EmailVerified,
		RoleId:               d.RoleID,
		RoleName:             d.RoleName,
		RoleTypeCode:         d.RoleType,
		DepartmentId:         d.DepartmentID,
		DepartmentName:       d.DepartmentName,
		StatusCode:           string(d.StatusCode),
		IsCommissionEligible: d.IsCommissionEligible,
		CreatedAt:            timestamppb.New(d.CreatedAt),
		UpdatedAt:            timestamppb.New(d.UpdatedAt),
	}

	if d.LastUsedAt != nil {
		proto.LastUsedAt = timestamppb.New(*d.LastUsedAt)
	}
	if d.DepartmentCreatedAt != nil {
		proto.DepartmentCreatedAt = timestamppb.New(*d.DepartmentCreatedAt)
	}
	if d.DepartmentUpdatedAt != nil {
		proto.DepartmentUpdatedAt = timestamppb.New(*d.DepartmentUpdatedAt)
	}

	return proto
}

func (h *gRPCHandler) BatchGetAccountUsersByIDs(ctx context.Context, req *pb.BatchGetAccountUsersByIDsRequest) (*pb.BatchGetAccountUsersByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	users, apiErr := h.accountUserSvc.BatchGetAccountUsersByIDs(ctx, req.Ids)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbUsers := make([]*pb.AccountUserDetail, len(users))
	for i, u := range users {
		pbUsers[i] = accountUserDetailToProto(u)
	}

	return &pb.BatchGetAccountUsersByIDsResponse{
		AccountUsers: pbUsers,
	}, nil
}
