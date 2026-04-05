package grpc

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/core"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (h *gRPCHandler) ListAccountUsers(ctx context.Context, req *pb.ListAccountUsersRequest) (*pb.ListAccountUsersResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	result, apiErr := h.accountUserSvc.ListAccountUsers(ctx, domain.ListAccountUsersParams{
		Cursor:         req.Cursor,
		Limit:          req.Limit,
		Query:          req.Query,
		RoleType:       req.RoleType,
		IncludeRemoved: req.IncludeRemoved,
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

	detail, apiErr := h.accountUserSvc.GetAccountUser(ctx, req.AccountUserId)
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
		Name:                          req.Name,
		Email:                         req.Email,
		Username:                      req.Username,
		Password:                      req.Password,
		RoleID:                        req.RoleId,
		DepartmentID:                  req.DepartmentId,
		IsSalesRep:                    req.IsSalesRep != nil && *req.IsSalesRep,
		ReceivesOrderAcknowledgements: req.ReceivesOrderAcknowledgements,
		ReceivesInvoiceNotifications:  req.ReceivesInvoiceNotifications,
		ReceivesPurchaseOrderSubmissionNotifications: req.ReceivesPurchaseOrderSubmissionNotifications,
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

	detail, apiErr := h.accountUserSvc.UpdateAccountUser(ctx, domain.UpdateAccountUserParams{
		AccountUserID: req.AccountUserId,
		Name:          req.Name,
		Email:         req.Email,
		Username:      req.Username,
		RoleID:        req.RoleId,
		DepartmentID:  req.DepartmentId,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateAccountUserResponse{
		AccountUser: accountUserDetailToProto(detail),
	}, nil
}

func (h *gRPCHandler) DeleteAccountUser(ctx context.Context, req *pb.DeleteAccountUserRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.accountUserSvc.DeleteAccountUser(ctx, req.AccountUserId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *gRPCHandler) LockAccountUser(ctx context.Context, req *pb.LockAccountUserRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.accountUserSvc.LockAccountUser(ctx, req.AccountUserId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *gRPCHandler) UnlockAccountUser(ctx context.Context, req *pb.UnlockAccountUserRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.accountUserSvc.UnlockAccountUser(ctx, req.AccountUserId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *gRPCHandler) RestoreAccountUser(ctx context.Context, req *pb.RestoreAccountUserRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.accountUserSvc.RestoreAccountUser(ctx, req.AccountUserId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *gRPCHandler) UpdateAccountUserPassword(ctx context.Context, req *pb.UpdateAccountUserPasswordRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.accountUserSvc.UpdateAccountUserPassword(ctx, req.AccountUserId, req.RequesterPassword, req.NewPassword)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *gRPCHandler) UpdateNotificationPreferences(ctx context.Context, req *pb.UpdateNotificationPreferencesRequest) (*pb.UpdateNotificationPreferencesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	prefs := make([]domain.UpdateNotificationPreferenceItem, len(req.Preferences))
	for i, p := range req.Preferences {
		prefs[i] = domain.UpdateNotificationPreferenceItem{
			NotificationTypeCode: p.NotificationTypeCode,
			Enabled:              p.Enabled,
		}
	}

	detail, apiErr := h.accountUserSvc.UpdateNotificationPreferences(ctx, domain.UpdateNotificationPreferencesParams{
		AccountUserID: req.AccountUserId,
		Preferences:   prefs,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateNotificationPreferencesResponse{
		AccountUser: accountUserDetailToProto(detail),
	}, nil
}

func accountUserDetailToProto(d *domain.AccountUserDetail) *pb.AccountUserDetail {
	if d == nil {
		return nil
	}

	proto := &pb.AccountUserDetail{
		Id:             d.ID,
		UserId:         d.UserID,
		Name:           d.Name,
		Email:          d.Email,
		Username:       d.Username,
		ImageUrl:       d.ImageURL,
		EmailVerified:  d.EmailVerified,
		RoleId:         d.RoleID,
		RoleName:       d.RoleName,
		RoleTypeCode:   d.RoleTypeCode,
		DepartmentId:   d.DepartmentID,
		DepartmentName: d.DepartmentName,
		StatusCode:     string(d.StatusCode),
		CreatedAt:      timestamppb.New(d.CreatedAt),
		UpdatedAt:      timestamppb.New(d.UpdatedAt),
	}

	if d.LastUsedAt != nil {
		proto.LastUsedAt = timestamppb.New(*d.LastUsedAt)
	}

	return proto
}
