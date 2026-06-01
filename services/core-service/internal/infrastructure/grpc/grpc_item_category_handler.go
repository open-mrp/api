package grpc

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/core"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func itemCategoryFullToProto(ic *domain.ItemCategoryFull) *pb.ItemCategoryInfo {
	if ic == nil {
		return nil
	}

	info := &pb.ItemCategoryInfo{
		Id:                   ic.ID,
		Name:                 ic.Name,
		ItemCategoryTypeCode: ic.ItemCategoryTypeCode,
		UnitGroupId:          ic.UnitGroupID,
		Notes:                ic.Notes,
		CreatedAt:            timestamppb.New(ic.CreatedAt),
		UpdatedAt:            timestamppb.New(ic.UpdatedAt),
	}

	if ic.AccountID != nil {
		info.AccountId = ic.AccountID
	}

	if ic.Properties != nil {
		info.Properties = make([]*pb.ItemCategoryPropertyInfo, len(ic.Properties))
		for i, p := range ic.Properties {
			info.Properties[i] = &pb.ItemCategoryPropertyInfo{
				Id:        p.ID,
				Name:      p.Name,
				CreatedAt: timestamppb.New(p.CreatedAt),
				UpdatedAt: timestamppb.New(p.UpdatedAt),
			}
		}
	}

	if ic.UnitGroup != nil {
		ugInfo := &pb.ItemCategoryUnitGroupInfo{
			Id:         ic.UnitGroup.ID,
			Name:       ic.UnitGroup.Name,
			BaseUnitId: ic.UnitGroup.BaseUnitID,
			Type:       ic.UnitGroup.Type,
			CreatedAt:  timestamppb.New(ic.UnitGroup.CreatedAt),
			UpdatedAt:  timestamppb.New(ic.UnitGroup.UpdatedAt),
			BaseUnit:   lightUnitToProto(ic.UnitGroup.BaseUnit),
		}
		if len(ic.UnitGroup.AssociatedUnits) > 0 {
			ugInfo.AssociatedUnits = make([]*pb.ItemCategoryUnitGroupUnitInfo, len(ic.UnitGroup.AssociatedUnits))
			for i, u := range ic.UnitGroup.AssociatedUnits {
				ugInfo.AssociatedUnits[i] = itemCategoryUnitGroupUnitToProto(u)
			}
		}
		info.UnitGroup = ugInfo
	}

	return info
}

func (h *gRPCHandler) ListItemCategories(ctx context.Context, req *pb.ListItemCategoriesRequest) (*pb.ListItemCategoriesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListItemCategoriesParams{
		Cursor:   req.Cursor,
		Limit:    req.Limit,
		Query:    req.Query,
		Type:     req.Type,
		Includes: req.Includes,
	}

	result, apiErr := h.itemCategorySvc.ListItemCategories(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbItems := make([]*pb.ItemCategoryInfo, len(result.ItemCategories))
	for i, ic := range result.ItemCategories {
		pbItems[i] = itemCategoryFullToProto(ic)
	}

	return &pb.ListItemCategoriesResponse{
		ItemCategories: pbItems,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) GetItemCategory(ctx context.Context, req *pb.GetItemCategoryRequest) (*pb.GetItemCategoryResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	itemCategory, apiErr := h.itemCategorySvc.GetItemCategory(ctx, domain.GetItemCategoryParams{
		ItemCategoryID: req.Id,
		Includes:       req.Includes,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetItemCategoryResponse{
		ItemCategory: itemCategoryFullToProto(itemCategory),
	}, nil
}

func (h *gRPCHandler) CreateItemCategory(ctx context.Context, req *pb.CreateItemCategoryRequest) (*pb.CreateItemCategoryResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.CreateItemCategoryParams{
		Name:                 req.Name,
		ItemCategoryTypeCode: req.Type,
		UnitGroupID:          req.UnitGroupId,
		Includes:             req.Includes,
	}

	itemCategory, apiErr := h.itemCategorySvc.CreateItemCategory(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateItemCategoryResponse{
		ItemCategory: itemCategoryFullToProto(itemCategory),
	}, nil
}

func (h *gRPCHandler) UpdateItemCategory(ctx context.Context, req *pb.UpdateItemCategoryRequest) (*pb.UpdateItemCategoryResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateItemCategoryParams{
		ItemCategoryID: req.Id,
		Name:           req.Name,
		Notes:          req.Notes,
		Includes:       req.Includes,
	}

	itemCategory, apiErr := h.itemCategorySvc.UpdateItemCategory(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateItemCategoryResponse{
		ItemCategory: itemCategoryFullToProto(itemCategory),
	}, nil
}

func (h *gRPCHandler) DeleteItemCategory(ctx context.Context, req *pb.DeleteItemCategoryRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.itemCategorySvc.DeleteItemCategory(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *gRPCHandler) AddItemCategoryProperty(ctx context.Context, req *pb.AddItemCategoryPropertyRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.AddItemCategoryPropertyParams{
		ItemCategoryID: req.Id,
		PropertyID:     req.PropertyId,
	}

	if apiErr := h.itemCategorySvc.AddItemCategoryProperty(ctx, params); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *gRPCHandler) RemoveItemCategoryProperty(ctx context.Context, req *pb.RemoveItemCategoryPropertyRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.RemoveItemCategoryPropertyParams{
		ItemCategoryID: req.Id,
		PropertyID:     req.PropertyId,
	}

	if apiErr := h.itemCategorySvc.RemoveItemCategoryProperty(ctx, params); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *gRPCHandler) BatchGetItemCategoriesByIDs(ctx context.Context, req *pb.BatchGetItemCategoriesByIDsRequest) (*pb.BatchGetItemCategoriesByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	categories, apiErr := h.itemCategorySvc.BatchGetItemCategoriesByIDs(ctx, req.Ids)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbItems := make([]*pb.ItemCategoryInfo, len(categories))
	for i, ic := range categories {
		pbItems[i] = itemCategoryFullToProto(ic)
	}

	return &pb.BatchGetItemCategoriesByIDsResponse{
		ItemCategories: pbItems,
	}, nil
}

func (h *gRPCHandler) ChangeItemCategoryUnitGroup(ctx context.Context, req *pb.ChangeItemCategoryUnitGroupRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ChangeItemCategoryUnitGroupParams{
		ItemCategoryID: req.Id,
		UnitGroupID:    req.UnitGroupId,
	}

	if apiErr := h.itemCategorySvc.ChangeItemCategoryUnitGroup(ctx, params); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}
