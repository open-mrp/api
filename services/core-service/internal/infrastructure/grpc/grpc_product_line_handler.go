package grpc

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/core"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func productLineFullToProto(pl *domain.ProductLineFull) *pb.ProductLineInfo {
	if pl == nil {
		return nil
	}

	info := &pb.ProductLineInfo{
		Id:               pl.ID,
		Name:             pl.Name,
		CommissionPolicy: string(pl.CommissionPolicy),
		FreightPolicy:    string(pl.FreightPolicy),
		UnitGroupId:      pl.UnitGroupID,
		CreatedAt:        timestamppb.New(pl.CreatedAt),
		UpdatedAt:        timestamppb.New(pl.UpdatedAt),
	}

	if pl.Description != nil {
		info.Description = pl.Description
	}

	if pl.Notes != nil {
		info.Notes = pl.Notes
	}

	if pl.AccountID != nil {
		info.AccountId = pl.AccountID
	}

	if pl.UnitGroup != nil {
		info.UnitGroup = &pb.ItemCategoryUnitGroupInfo{
			Id:         pl.UnitGroup.ID,
			Name:       pl.UnitGroup.Name,
			BaseUnitId: pl.UnitGroup.BaseUnitID,
			Type:       pl.UnitGroup.Type,
			CreatedAt:  timestamppb.New(pl.UnitGroup.CreatedAt),
			UpdatedAt:  timestamppb.New(pl.UnitGroup.UpdatedAt),
		}
	}

	return info
}

func (h *gRPCHandler) ListProductLines(ctx context.Context, req *pb.ListProductLinesRequest) (*pb.ListProductLinesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListProductLinesParams{
		Cursor:   req.Cursor,
		Limit:    req.Limit,
		Query:    req.Query,
		Includes: req.Includes,
	}

	result, apiErr := h.productLineSvc.ListProductLines(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbItems := make([]*pb.ProductLineInfo, len(result.ProductLines))
	for i, pl := range result.ProductLines {
		pbItems[i] = productLineFullToProto(pl)
	}

	return &pb.ListProductLinesResponse{
		ProductLines: pbItems,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) GetProductLine(ctx context.Context, req *pb.GetProductLineRequest) (*pb.GetProductLineResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	productLine, apiErr := h.productLineSvc.GetProductLine(ctx, domain.GetProductLineParams{
		ProductLineID: req.Id,
		Includes:      req.Includes,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetProductLineResponse{
		ProductLine: productLineFullToProto(productLine),
	}, nil
}

func (h *gRPCHandler) CreateProductLine(ctx context.Context, req *pb.CreateProductLineRequest) (*pb.CreateProductLineResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.CreateProductLineParams{
		Name:             req.Name,
		UnitGroupID:      req.UnitGroupId,
		CommissionPolicy: constants.CommissionPolicy(req.CommissionPolicy),
		FreightPolicy:    constants.FreightPolicy(req.FreightPolicy),
		Includes:         req.Includes,
	}

	productLine, apiErr := h.productLineSvc.CreateProductLine(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateProductLineResponse{
		ProductLine: productLineFullToProto(productLine),
	}, nil
}

func (h *gRPCHandler) UpdateProductLine(ctx context.Context, req *pb.UpdateProductLineRequest) (*pb.UpdateProductLineResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateProductLineParams{
		ProductLineID: req.Id,
		Name:          req.Name,
		UnitGroupID:   req.UnitGroupId,
		Includes:      req.Includes,
	}
	if req.CommissionPolicy != nil {
		cp := constants.CommissionPolicy(*req.CommissionPolicy)
		params.CommissionPolicy = &cp
	}
	if req.FreightPolicy != nil {
		fp := constants.FreightPolicy(*req.FreightPolicy)
		params.FreightPolicy = &fp
	}

	productLine, apiErr := h.productLineSvc.UpdateProductLine(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateProductLineResponse{
		ProductLine: productLineFullToProto(productLine),
	}, nil
}

func (h *gRPCHandler) DeleteProductLine(ctx context.Context, req *pb.DeleteProductLineRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.productLineSvc.DeleteProductLine(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}
