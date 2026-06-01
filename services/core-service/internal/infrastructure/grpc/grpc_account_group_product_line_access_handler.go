package grpc

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/core"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func accountGroupProductLineAccessToProto(a *domain.AccountGroupProductLineAccess) *pb.AccountGroupProductLineAccessInfo {
	if a == nil {
		return nil
	}

	productLines := make([]*pb.ProductLineAccessInfo, len(a.ProductLines))
	for i, pl := range a.ProductLines {
		productLines[i] = &pb.ProductLineAccessInfo{
			Id:   pl.ID,
			Name: pl.Name,
		}
	}

	return &pb.AccountGroupProductLineAccessInfo{
		AccountGroupId:   a.AccountGroupID,
		AccountGroupName: a.AccountGroupName,
		ProductLines:     productLines,
		CreatedAt:        timestamppb.New(a.CreatedAt),
		UpdatedAt:        timestamppb.New(a.UpdatedAt),
	}
}

func (h *gRPCHandler) ListAccountGroupProductLineAccess(ctx context.Context, req *pb.ListAccountGroupProductLineAccessRequest) (*pb.ListAccountGroupProductLineAccessResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListAccountGroupProductLineAccessParams{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	result, apiErr := h.accountGroupProductLineAccessSvc.ListAccountGroupProductLineAccess(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbItems := make([]*pb.AccountGroupProductLineAccessInfo, len(result.Items))
	for i, item := range result.Items {
		pbItems[i] = accountGroupProductLineAccessToProto(item)
	}

	return &pb.ListAccountGroupProductLineAccessResponse{
		Items: pbItems,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) BatchGetAccountGroupProductLineAccessByIDs(ctx context.Context, req *pb.BatchGetAccountGroupProductLineAccessByIDsRequest) (*pb.BatchGetAccountGroupProductLineAccessByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	items, apiErr := h.accountGroupProductLineAccessSvc.BatchGetAccountGroupProductLineAccessByIDs(ctx, req.Ids)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	pbItems := make([]*pb.AccountGroupProductLineAccessInfo, len(items))
	for i, it := range items {
		pbItems[i] = accountGroupProductLineAccessToProto(it)
	}
	return &pb.BatchGetAccountGroupProductLineAccessByIDsResponse{Items: pbItems}, nil
}

func (h *gRPCHandler) GetAccountGroupProductLineAccess(ctx context.Context, req *pb.GetAccountGroupProductLineAccessRequest) (*pb.GetAccountGroupProductLineAccessResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	access, apiErr := h.accountGroupProductLineAccessSvc.GetAccountGroupProductLineAccess(ctx, req.AccountGroupId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetAccountGroupProductLineAccessResponse{
		Item: accountGroupProductLineAccessToProto(access),
	}, nil
}

func (h *gRPCHandler) CreateAccountGroupProductLineAccess(ctx context.Context, req *pb.CreateAccountGroupProductLineAccessRequest) (*pb.CreateAccountGroupProductLineAccessResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.CreateAccountGroupProductLineAccessParams{
		AccountGroupID: req.AccountGroupId,
		ProductLineIDs: req.ProductLineIds,
	}

	access, apiErr := h.accountGroupProductLineAccessSvc.CreateAccountGroupProductLineAccess(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateAccountGroupProductLineAccessResponse{
		Item: accountGroupProductLineAccessToProto(access),
	}, nil
}

func (h *gRPCHandler) UpdateAccountGroupProductLineAccess(ctx context.Context, req *pb.UpdateAccountGroupProductLineAccessRequest) (*pb.UpdateAccountGroupProductLineAccessResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateAccountGroupProductLineAccessParams{
		AccountGroupID: req.AccountGroupId,
		ProductLineIDs: req.ProductLineIds,
	}

	access, apiErr := h.accountGroupProductLineAccessSvc.UpdateAccountGroupProductLineAccess(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateAccountGroupProductLineAccessResponse{
		Item: accountGroupProductLineAccessToProto(access),
	}, nil
}

func (h *gRPCHandler) DeleteAccountGroupProductLineAccess(ctx context.Context, req *pb.DeleteAccountGroupProductLineAccessRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.accountGroupProductLineAccessSvc.DeleteAccountGroupProductLineAccess(ctx, req.AccountGroupId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}
