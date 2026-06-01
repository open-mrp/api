package grpc

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/patch"
	pb "github.com/augno/api/shared/proto/core"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func partToProto(p *domain.Part) *pb.PartInfo {
	if p == nil {
		return nil
	}

	return &pb.PartInfo{
		Id:        p.ID,
		ItemId:    p.ItemID,
		Item:      itemToProto(p.Item),
		CreatedAt: timestamppb.New(p.CreatedAt),
		UpdatedAt: timestamppb.New(p.UpdatedAt),
	}
}

func (h *gRPCHandler) CreatePart(ctx context.Context, req *pb.CreatePartRequest) (*pb.CreatePartResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.CreatePartParams{
		SKU:          req.Sku,
		Description:  req.Description,
		Notes:        req.Notes,
		CategoryID:   req.CategoryId,
		UnitPrice:    protoToCreateRateInput(req.UnitPrice),
		UnitCost:     protoToCreateRateInput(req.UnitCost),
		AttributeIDs: req.AttributeIds,
		Includes:     req.Includes,
	}

	part, apiErr := h.partSvc.CreatePart(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreatePartResponse{
		Part: partToProto(part),
	}, nil
}

func (h *gRPCHandler) GetPart(ctx context.Context, req *pb.GetPartRequest) (*pb.GetPartResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	part, apiErr := h.partSvc.GetPart(ctx, domain.GetPartParams{
		PartID:   req.Id,
		Includes: req.Includes,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetPartResponse{
		Part: partToProto(part),
	}, nil
}

func (h *gRPCHandler) ListParts(ctx context.Context, req *pb.ListPartsRequest) (*pb.ListPartsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListPartsParams{
		Cursor:       req.Cursor,
		Limit:        req.Limit,
		Query:        req.Query,
		CategoryIDs:  req.CategoryIds,
		AttributeIDs: req.AttributeIds,
		Includes:     req.Includes,
	}

	if req.StartDate != nil {
		t := req.StartDate.AsTime()
		params.StartDate = &t
	}
	if req.EndDate != nil {
		t := req.EndDate.AsTime()
		params.EndDate = &t
	}

	result, apiErr := h.partSvc.ListParts(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbParts := make([]*pb.PartInfo, len(result.Parts))
	for i, part := range result.Parts {
		pbParts[i] = partToProto(part)
	}

	return &pb.ListPartsResponse{
		Parts: pbParts,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) ExportParts(ctx context.Context, req *pb.ExportPartsRequest) (*pb.ExportPartsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ExportPartsParams{
		Query:        req.Query,
		CategoryIDs:  req.CategoryIds,
		AttributeIDs: req.AttributeIds,
	}
	if req.StartDate != nil {
		t := req.StartDate.AsTime()
		params.StartDate = &t
	}
	if req.EndDate != nil {
		t := req.EndDate.AsTime()
		params.EndDate = &t
	}

	parts, apiErr := h.partSvc.ExportParts(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbParts := make([]*pb.PartInfo, len(parts))
	for i, part := range parts {
		pbParts[i] = partToProto(part)
	}

	return &pb.ExportPartsResponse{Parts: pbParts}, nil
}

func (h *gRPCHandler) UpdatePart(ctx context.Context, req *pb.UpdatePartRequest) (*pb.UpdatePartResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdatePartParams{
		PartID:      req.Id,
		SKU:         req.Sku,
		Description: patch.StringFieldFromProto(req.Description),
		Notes:       patch.StringFieldFromProto(req.Notes),
		Includes:    req.Includes,
	}

	part, apiErr := h.partSvc.UpdatePart(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdatePartResponse{
		Part: partToProto(part),
	}, nil
}

func (h *gRPCHandler) DeletePart(ctx context.Context, req *pb.DeletePartRequest) (*pb.DeletePartResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	part, apiErr := h.partSvc.DeletePart(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.DeletePartResponse{
		Part: partToProto(part),
	}, nil
}

func (h *gRPCHandler) BatchGetPartsByIDs(ctx context.Context, req *pb.BatchGetPartsByIDsRequest) (*pb.BatchGetPartsByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	parts, apiErr := h.partSvc.BatchGetPartsByIDs(ctx, req.Ids)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbParts := make([]*pb.PartInfo, len(parts))
	for i, p := range parts {
		pbParts[i] = partToProto(p)
	}

	return &pb.BatchGetPartsByIDsResponse{
		Parts: pbParts,
	}, nil
}
