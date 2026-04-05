package grpc

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/core"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func customerProductLineAccessToProto(a *domain.CustomerProductLineAccess) *pb.CustomerProductLineAccessInfo {
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

	return &pb.CustomerProductLineAccessInfo{
		CustomerId:     a.CustomerID,
		CustomerName:   a.CustomerName,
		CustomerNumber: a.CustomerNumber,
		ProductLines:   productLines,
		CreatedAt:      timestamppb.New(a.CreatedAt),
		UpdatedAt:      timestamppb.New(a.UpdatedAt),
	}
}

func (h *gRPCHandler) ListCustomerProductLineAccess(ctx context.Context, req *pb.ListCustomerProductLineAccessRequest) (*pb.ListCustomerProductLineAccessResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListCustomerProductLineAccessParams{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	result, apiErr := h.customerProductLineAccessSvc.ListCustomerProductLineAccess(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbItems := make([]*pb.CustomerProductLineAccessInfo, len(result.Items))
	for i, item := range result.Items {
		pbItems[i] = customerProductLineAccessToProto(item)
	}

	return &pb.ListCustomerProductLineAccessResponse{
		Items: pbItems,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) GetCustomerProductLineAccess(ctx context.Context, req *pb.GetCustomerProductLineAccessRequest) (*pb.GetCustomerProductLineAccessResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	access, apiErr := h.customerProductLineAccessSvc.GetCustomerProductLineAccess(ctx, req.CustomerId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetCustomerProductLineAccessResponse{
		Item: customerProductLineAccessToProto(access),
	}, nil
}

func (h *gRPCHandler) CreateCustomerProductLineAccess(ctx context.Context, req *pb.CreateCustomerProductLineAccessRequest) (*pb.CreateCustomerProductLineAccessResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.CreateCustomerProductLineAccessParams{
		CustomerID:     req.CustomerId,
		ProductLineIDs: req.ProductLineIds,
	}

	access, apiErr := h.customerProductLineAccessSvc.CreateCustomerProductLineAccess(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateCustomerProductLineAccessResponse{
		Item: customerProductLineAccessToProto(access),
	}, nil
}

func (h *gRPCHandler) UpdateCustomerProductLineAccess(ctx context.Context, req *pb.UpdateCustomerProductLineAccessRequest) (*pb.UpdateCustomerProductLineAccessResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateCustomerProductLineAccessParams{
		CustomerID:     req.CustomerId,
		ProductLineIDs: req.ProductLineIds,
	}

	access, apiErr := h.customerProductLineAccessSvc.UpdateCustomerProductLineAccess(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateCustomerProductLineAccessResponse{
		Item: customerProductLineAccessToProto(access),
	}, nil
}

func (h *gRPCHandler) DeleteCustomerProductLineAccess(ctx context.Context, req *pb.DeleteCustomerProductLineAccessRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.customerProductLineAccessSvc.DeleteCustomerProductLineAccess(ctx, req.CustomerId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}
