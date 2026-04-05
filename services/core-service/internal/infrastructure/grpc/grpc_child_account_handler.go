package grpc

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/core"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func childAccountToProto(ca *domain.ChildAccount) *pb.ChildAccountProto {
	p := &pb.ChildAccountProto{
		RelationId:     ca.RelationID,
		AccountId:      ca.AccountID,
		AccountName:    ca.AccountName,
		ExternalNumber: ca.ExternalNumber,
		Email:          ca.Email,
		CreatedAt:      timestamppb.New(ca.CreatedAt),
		UpdatedAt:      timestamppb.New(ca.UpdatedAt),
	}
	return p
}

func (h *gRPCHandler) ListChildAccounts(ctx context.Context, req *pb.ListChildAccountsRequest) (*pb.ListChildAccountsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	result, apiErr := h.childAccountSvc.ListChildAccounts(ctx, req.Cursor, req.Limit, req.Query)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbItems := make([]*pb.ChildAccountProto, len(result.Items))
	for i, item := range result.Items {
		pbItems[i] = childAccountToProto(item)
	}

	return &pb.ListChildAccountsResponse{
		Items: pbItems,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) AddChildAccount(ctx context.Context, req *pb.AddChildAccountRequest) (*pb.AddChildAccountResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	result, apiErr := h.childAccountSvc.AddChildAccount(ctx, req.ChildAccountId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.AddChildAccountResponse{
		ChildAccount: childAccountToProto(result),
	}, nil
}

func (h *gRPCHandler) RemoveChildAccount(ctx context.Context, req *pb.RemoveChildAccountRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	apiErr := h.childAccountSvc.RemoveChildAccount(ctx, req.ChildAccountId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}
