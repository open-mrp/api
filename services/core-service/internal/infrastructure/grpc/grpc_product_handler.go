package grpc

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/core"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func productFullToProto(p *domain.ProductFull) *pb.ProductFullInfo {
	if p == nil {
		return nil
	}

	info := &pb.ProductFullInfo{
		Id:              p.ID,
		ProductTypeCode: p.ProductTypeCode,
		IsPortalReady:   p.IsPortalReady,
		ItemId:          p.ItemID,
		CreatedAt:       timestamppb.New(p.CreatedAt),
		UpdatedAt:       timestamppb.New(p.UpdatedAt),
		Item:            itemToProto(p.Item),
	}

	if p.ProductLineID != nil {
		info.ProductLineId = p.ProductLineID
	}

	if p.ProductLine != nil {
		info.ProductLine = productLineFullToProto(p.ProductLine)
	}

	if p.ProductType != nil {
		info.ProductType = productTypeToProto(p.ProductType)
	}

	return info
}

func (h *gRPCHandler) ListProductsFull(ctx context.Context, req *pb.ListProductsFullRequest) (*pb.ListProductsFullResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListProductsFullParams{
		Cursor:         req.Cursor,
		Limit:          req.Limit,
		Query:          req.Query,
		CustomerIDs:    req.CustomerIds,
		ProductLineIDs: req.ProductLineIds,
		CategoryIDs:    req.CategoryIds,
		AttributeIDs:   req.AttributeIds,
		IsPortalReady:  req.IsPortalReady,
		Includes:       req.Includes,
	}

	if req.StartDate != nil {
		t := req.StartDate.AsTime()
		params.StartDate = &t
	}
	if req.EndDate != nil {
		t := req.EndDate.AsTime()
		params.EndDate = &t
	}

	result, apiErr := h.productSvc.ListProductsFull(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbProducts := make([]*pb.ProductFullInfo, len(result.Products))
	for i, p := range result.Products {
		pbProducts[i] = productFullToProto(p)
	}

	return &pb.ListProductsFullResponse{
		Products: pbProducts,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.GetProductResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.GetProductFullParams{
		ProductID: req.Id,
		Includes:  req.Includes,
	}

	product, apiErr := h.productSvc.GetProduct(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetProductResponse{
		Product: productFullToProto(product),
	}, nil
}

func (h *gRPCHandler) CreateProduct(ctx context.Context, req *pb.CreateProductRequest) (*pb.CreateProductResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.CreateProductParams{
		SKU:             req.Sku,
		ProductTypeCode: req.ProductTypeCode,
		ProductLineID:   req.ProductLineId,
		CategoryID:      req.CategoryId,
		IsPortalReady:   req.IsPortalReady,
		Includes:        req.Includes,
	}

	if req.Description != nil {
		params.Description = req.Description
	}
	if req.Notes != nil {
		params.Notes = req.Notes
	}
	params.UnitPrice = protoToCreateRateInput(req.UnitPrice)
	params.UnitCost = protoToCreateRateInput(req.UnitCost)
	params.BurnRate = protoToCreateRateInput(req.BurnRate)
	if len(req.AttributeIds) > 0 {
		params.AttributeIDs = req.AttributeIds
	}

	product, apiErr := h.productSvc.CreateProduct(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateProductResponse{
		Product: productFullToProto(product),
	}, nil
}

func (h *gRPCHandler) UpdateProduct(ctx context.Context, req *pb.UpdateProductRequest) (*pb.UpdateProductResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateProductParams{
		ProductID:         req.Id,
		SKU:               req.Sku,
		Description:       req.Description,
		UpdateDescription: req.UpdateDescription,
		Notes:             req.Notes,
		UpdateNotes:       req.UpdateNotes,
		IsPortalReady:     req.IsPortalReady,
		UnitPrice:         protoToCreateRateInput(req.UnitPrice),
		Includes:          req.Includes,
	}

	product, apiErr := h.productSvc.UpdateProduct(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateProductResponse{
		Product: productFullToProto(product),
	}, nil
}

func (h *gRPCHandler) DeleteProduct(ctx context.Context, req *pb.DeleteProductRequest) (*pb.DeleteProductResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.DeleteProductParams{
		ProductID: req.Id,
	}

	product, apiErr := h.productSvc.DeleteProduct(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.DeleteProductResponse{
		Product: productFullToProto(product),
	}, nil
}

func (h *gRPCHandler) ChangeProductProductLine(ctx context.Context, req *pb.ChangeProductProductLineRequest) (*pb.ChangeProductProductLineResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.ChangeProductProductLineParams{
		ProductID:     req.Id,
		ProductLineID: req.ProductLineId,
		Includes:      req.Includes,
	}

	product, apiErr := h.productSvc.ChangeProductProductLine(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.ChangeProductProductLineResponse{
		Product: productFullToProto(product),
	}, nil
}

func (h *gRPCHandler) ValidateProducts(ctx context.Context, req *pb.ValidateProductsRequest) (*pb.ValidateProductsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ValidateProductsParams{
		ProductsMap: req.ProductsMap,
	}

	result, apiErr := h.productSvc.ValidateProducts(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbProducts := make(map[string]*pb.ProductFullInfo, len(result.Products))
	for k, p := range result.Products {
		pbProducts[k] = productFullToProto(p)
	}

	return &pb.ValidateProductsResponse{
		Products: pbProducts,
	}, nil
}
