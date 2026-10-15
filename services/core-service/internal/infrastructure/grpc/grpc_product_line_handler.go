package grpc

import (
	"context"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/fieldpb"
	pb "github.com/open-mrp/api/shared/proto/core"

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
		Supply:           pl.Supply.StringPtr(),
		Status:           string(pl.Status),
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
		ugInfo := &pb.ItemCategoryUnitGroupInfo{
			Id:         pl.UnitGroup.ID,
			Name:       pl.UnitGroup.Name,
			BaseUnitId: pl.UnitGroup.BaseUnitID,
			Type:       pl.UnitGroup.Type,
			CreatedAt:  timestamppb.New(pl.UnitGroup.CreatedAt),
			UpdatedAt:  timestamppb.New(pl.UnitGroup.UpdatedAt),
		}
		if pl.UnitGroup.BaseUnit != nil {
			ugInfo.BaseUnit = lightUnitToProto(pl.UnitGroup.BaseUnit)
		}
		if len(pl.UnitGroup.AssociatedUnits) > 0 {
			ugInfo.AssociatedUnits = make([]*pb.ItemCategoryUnitGroupUnitInfo, len(pl.UnitGroup.AssociatedUnits))
			for i, u := range pl.UnitGroup.AssociatedUnits {
				ugInfo.AssociatedUnits[i] = itemCategoryUnitGroupUnitToProto(u)
			}
		}
		info.UnitGroup = ugInfo
	}

	return info
}

func (h *gRPCHandler) ExportProductLines(ctx context.Context, req *pb.ExportProductLinesRequest) (*pb.ExportProductLinesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	job, apiErr := h.productLineSvc.ExportProductLines(ctx, domain.ExportProductLinesParams{Query: req.Query})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.ExportProductLinesResponse{Job: jobToProto(job)}, nil
}

func (h *gRPCHandler) ListProductLines(ctx context.Context, req *pb.ListProductLinesRequest) (*pb.ListProductLinesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListProductLinesParams{
		Cursor:   req.Cursor,
		Limit:    req.Limit,
		Query:    req.Query,
		IsActive: constants.IsActiveFilter(archiveStatuses(req.Statuses)),
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
		Description:      req.Description,
		UnitGroupID:      req.UnitGroupId,
		CommissionPolicy: constants.CommissionPolicy(req.CommissionPolicy),
		FreightPolicy:    constants.FreightPolicy(req.FreightPolicy),
		Status:           constants.ArchiveStatus(req.GetStatus()),
		Includes:         req.Includes,
	}
	if req.Supply != nil {
		supply := constants.Supply(*req.Supply)
		params.Supply = &supply
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
		Description:   fieldpb.StringClearableFromProto(req.Description),
		UnitGroupID:   req.UnitGroupId,
		Supply:        fieldpb.EnumClearableFromProto[constants.Supply](req.Supply),
		Includes:      req.Includes,
	}
	if req.Status != nil {
		status := constants.ArchiveStatus(*req.Status)
		params.Status = &status
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

func (h *gRPCHandler) BatchGetProductLinesByIDs(ctx context.Context, req *pb.BatchGetProductLinesByIDsRequest) (*pb.BatchGetProductLinesByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	productLines, apiErr := h.productLineSvc.BatchGetProductLinesByIDs(ctx, req.Ids)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	pbProductLines := make([]*pb.ProductLineInfo, len(productLines))
	for i, pl := range productLines {
		pbProductLines[i] = productLineFullToProto(pl)
	}
	return &pb.BatchGetProductLinesByIDsResponse{ProductLines: pbProductLines}, nil
}

func (h *gRPCHandler) BulkUpsertProductLines(ctx context.Context, req *pb.BulkUpsertProductLinesRequest) (*pb.BulkUpsertProductLinesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	inputs := make([]domain.UpsertProductLineParams, len(req.ProductLines))
	for i, pl := range req.ProductLines {
		inputs[i] = domain.UpsertProductLineParams{
			Name:             pl.Name,
			UnitGroup:        objectIdentifierFromProto(pl.UnitGroup),
			CommissionPolicy: constants.CommissionPolicy(pl.CommissionPolicy),
			FreightPolicy:    constants.FreightPolicy(pl.FreightPolicy),
		}
	}

	job, apiErr := h.productLineSvc.BulkUpsertProductLines(ctx, domain.BulkUpsertProductLinesParams{
		ProductLines: inputs,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.BulkUpsertProductLinesResponse{Job: jobToProto(job)}, nil
}

func archiveStatuses(values []string) []constants.ArchiveStatus {
	out := make([]constants.ArchiveStatus, len(values))
	for i, v := range values {
		out[i] = constants.ArchiveStatus(v)
	}
	return out
}
