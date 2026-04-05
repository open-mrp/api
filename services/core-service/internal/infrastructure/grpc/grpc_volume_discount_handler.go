package grpc

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/core"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func volumeDiscountToProto(d *domain.VolumeDiscount) *pb.VolumeDiscountInfo {
	if d == nil {
		return nil
	}

	tiers := make([]*pb.VolumeDiscountTierInfo, len(d.Tiers))
	for i, t := range d.Tiers {
		tiers[i] = &pb.VolumeDiscountTierInfo{
			Id:                 t.ID,
			Name:               t.Name,
			DiscountPercentage: t.DiscountPercentage,
			Threshold:          t.Threshold,
			ParentTierId:       t.ParentTierID,
			CreatedAt:          timestamppb.New(t.CreatedAt),
			UpdatedAt:          timestamppb.New(t.UpdatedAt),
		}
	}

	customerGroups := make([]*pb.VolumeDiscountCustomerGroupInfo, len(d.CustomerGroups))
	for i, cg := range d.CustomerGroups {
		customerGroups[i] = &pb.VolumeDiscountCustomerGroupInfo{
			Id:             cg.ID,
			AccountGroupId: cg.AccountGroupID,
			Name:           cg.Name,
		}
	}

	productLines := make([]*pb.VolumeDiscountProductLineInfo, len(d.ProductLines))
	for i, pl := range d.ProductLines {
		productLines[i] = &pb.VolumeDiscountProductLineInfo{
			Id:   pl.ID,
			Name: pl.Name,
		}
	}

	categories := make([]*pb.VolumeDiscountCategoryInfo, len(d.Categories))
	for i, cat := range d.Categories {
		categories[i] = &pb.VolumeDiscountCategoryInfo{
			Id:   cat.ID,
			Name: cat.Name,
		}
	}

	attributes := make([]*pb.VolumeDiscountAttributeInfo, len(d.Attributes))
	for i, attr := range d.Attributes {
		attributes[i] = &pb.VolumeDiscountAttributeInfo{
			Id:   attr.ID,
			Name: attr.Name,
		}
	}

	units := make([]*pb.VolumeDiscountUnitInfo, len(d.AcceptableUnits))
	for i, u := range d.AcceptableUnits {
		units[i] = &pb.VolumeDiscountUnitInfo{
			Id:           u.ID,
			Name:         u.Name,
			Abbreviation: u.Abbreviation,
		}
	}

	return &pb.VolumeDiscountInfo{
		Id:              d.ID,
		Name:            d.Name,
		Tiers:           tiers,
		CustomerGroups:  customerGroups,
		ProductLines:    productLines,
		Categories:      categories,
		Attributes:      attributes,
		AcceptableUnits: units,
		CreatedAt:       timestamppb.New(d.CreatedAt),
		UpdatedAt:       timestamppb.New(d.UpdatedAt),
	}
}

func (h *salesGRPCHandler) ListVolumeDiscounts(ctx context.Context, req *pb.ListVolumeDiscountsRequest) (*pb.ListVolumeDiscountsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListVolumeDiscountsParams{
		Cursor:            req.Cursor,
		Limit:             req.Limit,
		Query:             req.Query,
		CustomerAccountID: req.CustomerAccountId,
	}

	result, apiErr := h.volumeDiscountSvc.ListVolumeDiscounts(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	discounts := make([]*pb.VolumeDiscountInfo, len(result.VolumeDiscounts))
	for i, d := range result.VolumeDiscounts {
		discounts[i] = volumeDiscountToProto(d)
	}

	return &pb.ListVolumeDiscountsResponse{
		VolumeDiscounts: discounts,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *salesGRPCHandler) GetVolumeDiscount(ctx context.Context, req *pb.GetVolumeDiscountRequest) (*pb.GetVolumeDiscountResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.GetVolumeDiscountParams{
		VolumeDiscountID:  req.Id,
		CustomerAccountID: req.CustomerAccountId,
	}

	discount, apiErr := h.volumeDiscountSvc.GetVolumeDiscount(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetVolumeDiscountResponse{
		VolumeDiscount: volumeDiscountToProto(discount),
	}, nil
}

func (h *salesGRPCHandler) CreateVolumeDiscount(ctx context.Context, req *pb.CreateVolumeDiscountRequest) (*pb.CreateVolumeDiscountResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	tiers := make([]domain.CreateVolumeDiscountTierParams, len(req.Tiers))
	for i, t := range req.Tiers {
		tiers[i] = domain.CreateVolumeDiscountTierParams{
			Name:               t.Name,
			DiscountPercentage: t.DiscountPercentage,
			Threshold:          t.Threshold,
			ParentTierID:       t.ParentTierId,
		}
	}

	customerGroups := make([]domain.CreateVolumeDiscountCustomerGroupParams, len(req.CustomerGroupIds))
	for i, cgID := range req.CustomerGroupIds {
		customerGroups[i] = domain.CreateVolumeDiscountCustomerGroupParams{
			AccountGroupID: cgID,
		}
	}

	params := domain.CreateVolumeDiscountParams{
		Name:           req.Name,
		Tiers:          tiers,
		CustomerGroups: customerGroups,
		ProductLineIDs: req.ProductLineIds,
		CategoryIDs:    req.CategoryIds,
		AttributeIDs:   req.AttributeIds,
		UnitIDs:        req.UnitIds,
	}

	discount, apiErr := h.volumeDiscountSvc.CreateVolumeDiscount(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateVolumeDiscountResponse{
		VolumeDiscount: volumeDiscountToProto(discount),
	}, nil
}

func (h *salesGRPCHandler) UpdateVolumeDiscount(ctx context.Context, req *pb.UpdateVolumeDiscountRequest) (*pb.UpdateVolumeDiscountResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	tiers := make([]domain.UpdateVolumeDiscountTierParams, len(req.Tiers))
	for i, t := range req.Tiers {
		tiers[i] = domain.UpdateVolumeDiscountTierParams{
			ID:                 t.Id,
			Name:               t.Name,
			DiscountPercentage: t.DiscountPercentage,
			Threshold:          t.Threshold,
			ParentTierID:       t.ParentTierId,
		}
	}

	customerGroups := make([]domain.UpdateVolumeDiscountCustomerGroupParams, len(req.CustomerGroupIds))
	for i, cgID := range req.CustomerGroupIds {
		customerGroups[i] = domain.UpdateVolumeDiscountCustomerGroupParams{
			AccountGroupID: cgID,
		}
	}

	params := domain.UpdateVolumeDiscountParams{
		VolumeDiscountID:  req.Id,
		Name:              req.Name,
		Tiers:             tiers,
		CustomerGroups:    customerGroups,
		ProductLineIDs:    req.ProductLineIds,
		CategoryIDs:       req.CategoryIds,
		AttributeIDs:      req.AttributeIds,
		UnitIDs:           req.UnitIds,
		HasTiers:          req.HasTiers,
		HasCustomerGroups: req.HasCustomerGroups,
		HasProductLines:   req.HasProductLines,
		HasCategories:     req.HasCategories,
		HasAttributes:     req.HasAttributes,
		HasUnits:          req.HasUnits,
	}

	discount, apiErr := h.volumeDiscountSvc.UpdateVolumeDiscount(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateVolumeDiscountResponse{
		VolumeDiscount: volumeDiscountToProto(discount),
	}, nil
}

func (h *salesGRPCHandler) DeleteVolumeDiscount(ctx context.Context, req *pb.DeleteVolumeDiscountRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.volumeDiscountSvc.DeleteVolumeDiscount(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}
