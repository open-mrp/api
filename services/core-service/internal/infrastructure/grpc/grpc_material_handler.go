package grpc

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/core"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func protoToQuantityInput(q *pb.QuantityInput) *domain.QuantityInput {
	if q == nil {
		return nil
	}
	return &domain.QuantityInput{
		Value:  q.Value,
		UnitID: q.UnitId,
	}
}

// protoToCreateRateInput converts an optional CreateRateInput proto into the domain CreateRateParams shape. Returns nil when the proto is nil so callers can fall back to default-rate behavior in service layer.
func protoToCreateRateInput(r *pb.CreateRateInput) *domain.CreateRateParams {
	if r == nil {
		return nil
	}
	return &domain.CreateRateParams{
		Value:             r.Value,
		NumeratorUnitID:   r.NumeratorUnitId,
		DenominatorUnitID: r.DenominatorUnitId,
	}
}

func materialToProto(m *domain.Material) *pb.MaterialInfo {
	if m == nil {
		return nil
	}
	return &pb.MaterialInfo{
		Id:         m.ID,
		ItemId:     m.ItemID,
		Item:       itemToProto(m.Item),
		OrderPoint: quantityToProto(m.OrderPoint),
		LeadTime:   quantityToProto(m.LeadTime),
		CreatedAt:  timestamppb.New(m.CreatedAt),
		UpdatedAt:  timestamppb.New(m.UpdatedAt),
	}
}

func supplierMaterialToProto(sm *domain.SupplierMaterial) *pb.SupplierMaterialInfo {
	if sm == nil {
		return nil
	}
	return &pb.SupplierMaterialInfo{
		Id:                  sm.ID,
		MaterialId:          sm.MaterialID,
		SupplierAccountId:   sm.SupplierAccountID,
		SupplierPartNumber:  sm.SupplierPartNumber,
		SupplierDescription: sm.SupplierDescription,
		IsActive:            sm.IsActive,
		OwnerAccountId:      sm.OwnerAccountID,
		CreatedAt:           timestamppb.New(sm.CreatedAt),
		UpdatedAt:           timestamppb.New(sm.UpdatedAt),
		Material:            materialToProto(sm.Material),
	}
}

// Material handlers

func (h *gRPCHandler) ListMaterials(ctx context.Context, req *pb.ListMaterialsRequest) (*pb.ListMaterialsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListMaterialsParams{
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

	result, apiErr := h.materialSvc.ListMaterials(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbMaterials := make([]*pb.MaterialInfo, len(result.Materials))
	for i, m := range result.Materials {
		pbMaterials[i] = materialToProto(m)
	}

	return &pb.ListMaterialsResponse{
		Materials: pbMaterials,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) ExportMaterials(ctx context.Context, req *pb.ExportMaterialsRequest) (*pb.ExportMaterialsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ExportMaterialsParams{
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

	materials, apiErr := h.materialSvc.ExportMaterials(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbMaterials := make([]*pb.MaterialInfo, len(materials))
	for i, m := range materials {
		pbMaterials[i] = materialToProto(m)
	}

	return &pb.ExportMaterialsResponse{Materials: pbMaterials}, nil
}

func (h *gRPCHandler) GetMaterial(ctx context.Context, req *pb.GetMaterialRequest) (*pb.GetMaterialResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	material, apiErr := h.materialSvc.GetMaterial(ctx, domain.GetMaterialParams{
		MaterialID: req.Id,
		Includes:   req.Includes,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetMaterialResponse{
		Material: materialToProto(material),
	}, nil
}

func (h *gRPCHandler) CreateMaterial(ctx context.Context, req *pb.CreateMaterialRequest) (*pb.CreateMaterialResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.CreateMaterialParams{
		SKU:          req.Sku,
		CategoryID:   req.CategoryId,
		OrderPoint:   protoToQuantityInput(req.OrderPoint),
		LeadTime:     protoToQuantityInput(req.LeadTime),
		UnitPrice:    protoToCreateRateInput(req.UnitPrice),
		UnitCost:     protoToCreateRateInput(req.UnitCost),
		AttributeIDs: req.AttributeIds,
		Includes:     req.Includes,
	}

	if req.Description != nil {
		params.Description = req.Description
	}
	if req.Notes != nil {
		params.Notes = req.Notes
	}

	material, apiErr := h.materialSvc.CreateMaterial(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateMaterialResponse{
		Material: materialToProto(material),
	}, nil
}

func (h *gRPCHandler) UpdateMaterial(ctx context.Context, req *pb.UpdateMaterialRequest) (*pb.UpdateMaterialResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateMaterialParams{
		MaterialID:        req.Id,
		SKU:               req.Sku,
		Description:       req.Description,
		UpdateDescription: req.UpdateDescription,
		Notes:             req.Notes,
		UpdateNotes:       req.UpdateNotes,
		OrderPoint:        protoToQuantityInput(req.OrderPoint),
		LeadTime:          protoToQuantityInput(req.LeadTime),
		UnitCost:          protoToCreateRateInput(req.UnitCost),
		Includes:          req.Includes,
	}

	material, apiErr := h.materialSvc.UpdateMaterial(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateMaterialResponse{
		Material: materialToProto(material),
	}, nil
}

func (h *gRPCHandler) DeleteMaterial(ctx context.Context, req *pb.DeleteMaterialRequest) (*pb.DeleteMaterialResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	material, apiErr := h.materialSvc.DeleteMaterial(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.DeleteMaterialResponse{
		Material: materialToProto(material),
	}, nil
}

func (h *gRPCHandler) BatchGetMaterialsByIDs(ctx context.Context, req *pb.BatchGetMaterialsByIDsRequest) (*pb.BatchGetMaterialsByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	materials, apiErr := h.materialSvc.BatchGetMaterialsByIDs(ctx, req.Ids)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbMaterials := make([]*pb.MaterialInfo, len(materials))
	for i, m := range materials {
		pbMaterials[i] = materialToProto(m)
	}

	return &pb.BatchGetMaterialsByIDsResponse{
		Materials: pbMaterials,
	}, nil
}

// Supplier material handlers

func (h *gRPCHandler) ListSupplierMaterials(ctx context.Context, req *pb.ListSupplierMaterialsRequest) (*pb.ListSupplierMaterialsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListSupplierMaterialsParams{
		SupplierAccountID: req.SupplierAccountId,
		Cursor:            req.Cursor,
		Limit:             req.Limit,
		Query:             req.Query,
	}

	result, apiErr := h.supplierMaterialSvc.ListSupplierMaterials(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbSupplierMaterials := make([]*pb.SupplierMaterialInfo, len(result.SupplierMaterials))
	for i, sm := range result.SupplierMaterials {
		pbSupplierMaterials[i] = supplierMaterialToProto(sm)
	}

	return &pb.ListSupplierMaterialsResponse{
		SupplierMaterials: pbSupplierMaterials,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) GetSupplierMaterial(ctx context.Context, req *pb.GetSupplierMaterialRequest) (*pb.GetSupplierMaterialResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	sm, apiErr := h.supplierMaterialSvc.GetSupplierMaterial(ctx, req.SupplierAccountId, req.MaterialId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetSupplierMaterialResponse{
		SupplierMaterial: supplierMaterialToProto(sm),
	}, nil
}

func (h *gRPCHandler) CreateSupplierMaterial(ctx context.Context, req *pb.CreateSupplierMaterialRequest) (*pb.CreateSupplierMaterialResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.CreateSupplierMaterialParams{
		MaterialID:         req.MaterialId,
		SupplierAccountID:  req.SupplierAccountId,
		SupplierPartNumber: req.SupplierPartNumber,
		IsActive:           req.IsActive,
	}

	if req.SupplierDescription != nil {
		params.SupplierDescription = req.SupplierDescription
	}

	sm, apiErr := h.supplierMaterialSvc.CreateSupplierMaterial(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateSupplierMaterialResponse{
		SupplierMaterial: supplierMaterialToProto(sm),
	}, nil
}

func (h *gRPCHandler) UpdateSupplierMaterial(ctx context.Context, req *pb.UpdateSupplierMaterialRequest) (*pb.UpdateSupplierMaterialResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateSupplierMaterialParams{
		SupplierAccountID:   req.SupplierAccountId,
		MaterialID:          req.MaterialId,
		SupplierPartNumber:  req.SupplierPartNumber,
		SupplierDescription: req.SupplierDescription,
		UpdateDescription:   req.UpdateDescription,
		IsActive:            req.IsActive,
	}

	sm, apiErr := h.supplierMaterialSvc.UpdateSupplierMaterial(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateSupplierMaterialResponse{
		SupplierMaterial: supplierMaterialToProto(sm),
	}, nil
}

func (h *gRPCHandler) DeleteSupplierMaterial(ctx context.Context, req *pb.DeleteSupplierMaterialRequest) (*pb.DeleteSupplierMaterialResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.DeleteSupplierMaterialParams{
		SupplierAccountID: req.SupplierAccountId,
		MaterialID:        req.MaterialId,
	}

	sm, apiErr := h.supplierMaterialSvc.DeleteSupplierMaterial(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.DeleteSupplierMaterialResponse{
		SupplierMaterial: supplierMaterialToProto(sm),
	}, nil
}
