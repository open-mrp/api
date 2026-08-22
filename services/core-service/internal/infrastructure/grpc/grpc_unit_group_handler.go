package grpc

import (
	"context"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/field"
	pb "github.com/open-mrp/api/shared/proto/core"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func unitGroupToProto(ug *domain.UnitGroupFull) *pb.UnitGroupInfo {
	var conversions []*pb.UnitGroupUnitInfo
	for _, c := range ug.UnitConversions {
		conversions = append(conversions, unitGroupUnitToProto(c))
	}

	return &pb.UnitGroupInfo{
		Id:    ug.ID,
		Name:  ug.Name,
		Notes: ug.Notes,
		Type:  ug.Type,
		BaseUnit: unitToProto(&domain.Unit{
			ID:                ug.BaseUnit.ID,
			Name:              ug.BaseUnit.Name,
			Abbreviation:      ug.BaseUnit.Abbreviation,
			UnitDimensionCode: ug.BaseUnit.Type,
			RatioNumerator:    ug.BaseUnit.RatioNumerator,
			RatioDenominator:  ug.BaseUnit.RatioDenominator,
			OffsetNumerator:   ug.BaseUnit.OffsetNumerator,
			OffsetDenominator: ug.BaseUnit.OffsetDenominator,
			IsBaseUnit:        ug.BaseUnit.IsBaseUnit,
			AccountID:         ug.BaseUnit.AccountID,
			CreatedAt:         ug.BaseUnit.CreatedAt,
			UpdatedAt:         ug.BaseUnit.UpdatedAt,
		}),
		UnitConversions: conversions,
		IsInternal:      ug.AccountID != nil,
		CreatedAt:       timestamppb.New(ug.CreatedAt),
		UpdatedAt:       timestamppb.New(ug.UpdatedAt),
		AccountId:       ug.AccountID,
	}
}

func unitGroupUnitToProto(u *domain.UnitGroupUnit) *pb.UnitGroupUnitInfo {
	return &pb.UnitGroupUnitInfo{
		Id:                 u.ID,
		UnitId:             u.UnitID,
		UnitGroupId:        u.UnitGroupID,
		DiscountPercentage: u.DiscountPercentage,
		DiscountFixed:      u.DiscountFixed,
		IsVisible:          u.IsVisible,
		Unit: unitToProto(&domain.Unit{
			ID:                u.Unit.ID,
			Name:              u.Unit.Name,
			Abbreviation:      u.Unit.Abbreviation,
			UnitDimensionCode: u.Unit.Type,
			RatioNumerator:    u.Unit.RatioNumerator,
			RatioDenominator:  u.Unit.RatioDenominator,
			OffsetNumerator:   u.Unit.OffsetNumerator,
			OffsetDenominator: u.Unit.OffsetDenominator,
			IsBaseUnit:        u.Unit.IsBaseUnit,
			AccountID:         u.Unit.AccountID,
			CreatedAt:         u.Unit.CreatedAt,
			UpdatedAt:         u.Unit.UpdatedAt,
		}),
		CreatedAt: timestamppb.New(u.CreatedAt),
		UpdatedAt: timestamppb.New(u.UpdatedAt),
	}
}

func (h *gRPCHandler) ExportUnitGroups(ctx context.Context, req *pb.ExportUnitGroupsRequest) (*pb.ExportUnitGroupsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	job, apiErr := h.unitGroupSvc.ExportUnitGroups(ctx, domain.ExportUnitGroupsParams{Query: req.Query})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.ExportUnitGroupsResponse{Job: jobToProto(job)}, nil
}

func (h *gRPCHandler) ListUnitGroups(ctx context.Context, req *pb.ListUnitGroupsRequest) (*pb.ListUnitGroupsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListUnitGroupsParams{
		Cursor:   req.Cursor,
		Limit:    req.Limit,
		Query:    req.Query,
		Type:     req.Type,
		Includes: req.Includes,
	}

	result, apiErr := h.unitGroupSvc.ListUnitGroups(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbUnitGroups := make([]*pb.UnitGroupInfo, len(result.UnitGroups))
	for i, ug := range result.UnitGroups {
		pbUnitGroups[i] = unitGroupToProto(ug)
	}

	return &pb.ListUnitGroupsResponse{
		UnitGroups: pbUnitGroups,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) GetUnitGroup(ctx context.Context, req *pb.GetUnitGroupRequest) (*pb.GetUnitGroupResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	unitGroup, apiErr := h.unitGroupSvc.GetUnitGroup(ctx, domain.GetUnitGroupParams{
		UnitGroupID: req.Id,
		Includes:    req.Includes,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetUnitGroupResponse{
		UnitGroup: unitGroupToProto(unitGroup),
	}, nil
}

func (h *gRPCHandler) CreateUnitGroup(ctx context.Context, req *pb.CreateUnitGroupRequest) (*pb.CreateUnitGroupResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	conversions := make([]domain.CreateUnitGroupUnitParams, len(req.UnitConversions))
	for i, uc := range req.UnitConversions {
		conversions[i] = domain.CreateUnitGroupUnitParams{
			UnitID:             uc.UnitId,
			DiscountPercentage: uc.DiscountPercentage,
			DiscountFixed:      uc.DiscountFixed,
			IsVisible:          uc.IsVisible,
		}
	}

	params := domain.CreateUnitGroupParams{
		Name:            req.Name,
		Notes:           req.Notes,
		Type:            req.Type,
		BaseUnitID:      req.BaseUnitId,
		UnitConversions: conversions,
		Includes:        req.Includes,
	}

	unitGroup, apiErr := h.unitGroupSvc.CreateUnitGroup(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateUnitGroupResponse{
		UnitGroup: unitGroupToProto(unitGroup),
	}, nil
}

func (h *gRPCHandler) UpdateUnitGroup(ctx context.Context, req *pb.UpdateUnitGroupRequest) (*pb.UpdateUnitGroupResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateUnitGroupParams{
		UnitGroupID: req.Id,
		Name:        req.Name,
		Notes:       field.StringClearableFromProto(req.Notes),
		BaseUnitID:  req.BaseUnitId,
		Includes:    req.Includes,
	}

	if req.UpdateUnitConversions {
		conversions := make([]domain.CreateUnitGroupUnitParams, len(req.UnitConversions))
		for i, uc := range req.UnitConversions {
			conversions[i] = domain.CreateUnitGroupUnitParams{
				UnitID:             uc.UnitId,
				DiscountPercentage: uc.DiscountPercentage,
				DiscountFixed:      uc.DiscountFixed,
				IsVisible:          uc.IsVisible,
			}
		}
		params.UnitConversions = &conversions
	}

	unitGroup, apiErr := h.unitGroupSvc.UpdateUnitGroup(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateUnitGroupResponse{
		UnitGroup: unitGroupToProto(unitGroup),
	}, nil
}

func (h *gRPCHandler) DeleteUnitGroup(ctx context.Context, req *pb.DeleteUnitGroupRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.unitGroupSvc.DeleteUnitGroup(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *gRPCHandler) UpsertUnitGroupUnit(ctx context.Context, req *pb.UpsertUnitGroupUnitRequest) (*pb.UpsertUnitGroupUnitResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpsertUnitGroupUnitParams{
		UnitGroupID:        req.UnitGroupId,
		UnitGroupUnitID:    req.UnitGroupUnitId,
		UnitID:             req.UnitId,
		DiscountPercentage: req.DiscountPercentage,
		DiscountFixed:      req.DiscountFixed,
		IsVisibleProvided:  req.IsVisible != nil,
		Includes:           req.Includes,
	}
	if req.IsVisible != nil {
		params.IsVisible = *req.IsVisible
	}

	result, apiErr := h.unitGroupSvc.UpsertUnitGroupUnit(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpsertUnitGroupUnitResponse{
		UnitGroupUnit: unitGroupUnitToProto(result),
	}, nil
}

func (h *gRPCHandler) DeleteUnitGroupUnit(ctx context.Context, req *pb.DeleteUnitGroupUnitRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.DeleteUnitGroupUnitParams{
		UnitGroupID:     req.UnitGroupId,
		UnitGroupUnitID: req.UnitGroupUnitId,
	}

	apiErr := h.unitGroupSvc.DeleteUnitGroupUnit(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *gRPCHandler) ListUnitGroupUnits(ctx context.Context, req *pb.ListUnitGroupUnitsRequest) (*pb.ListUnitGroupUnitsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	units, apiErr := h.unitGroupSvc.ListUnitGroupUnits(ctx, req.UnitGroupId, req.Includes)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbUnits := make([]*pb.UnitGroupUnitInfo, len(units))
	for i, u := range units {
		pbUnits[i] = unitGroupUnitToProto(u)
	}

	return &pb.ListUnitGroupUnitsResponse{
		UnitGroupUnits: pbUnits,
	}, nil
}

func (h *gRPCHandler) GetUnitGroupUnit(ctx context.Context, req *pb.GetUnitGroupUnitRequest) (*pb.GetUnitGroupUnitResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.GetUnitGroupUnitParams{
		UnitGroupID:     req.UnitGroupId,
		UnitGroupUnitID: req.UnitGroupUnitId,
		Includes:        req.Includes,
	}

	result, apiErr := h.unitGroupSvc.GetUnitGroupUnit(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetUnitGroupUnitResponse{
		UnitGroupUnit: unitGroupUnitToProto(result),
	}, nil
}

func (h *gRPCHandler) BatchGetUnitGroupsByIDs(ctx context.Context, req *pb.BatchGetUnitGroupsByIDsRequest) (*pb.BatchGetUnitGroupsByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	unitGroups, apiErr := h.unitGroupSvc.BatchGetUnitGroupsByIDs(ctx, req.Ids)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	pbUnitGroups := make([]*pb.UnitGroupInfo, len(unitGroups))
	for i, ug := range unitGroups {
		pbUnitGroups[i] = unitGroupToProto(ug)
	}
	return &pb.BatchGetUnitGroupsByIDsResponse{UnitGroups: pbUnitGroups}, nil
}

func (h *gRPCHandler) BatchGetUnitGroupUnitsByIDs(ctx context.Context, req *pb.BatchGetUnitGroupUnitsByIDsRequest) (*pb.BatchGetUnitGroupUnitsByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	units, apiErr := h.unitGroupSvc.BatchGetUnitGroupUnitsByIDs(ctx, req.Ids)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	pbUnits := make([]*pb.UnitGroupUnitInfo, len(units))
	for i, u := range units {
		pbUnits[i] = unitGroupUnitToProto(u)
	}
	return &pb.BatchGetUnitGroupUnitsByIDsResponse{UnitGroupUnits: pbUnits}, nil
}

func (h *gRPCHandler) BulkUpsertUnitGroups(ctx context.Context, req *pb.BulkUpsertUnitGroupsRequest) (*pb.BulkUpsertUnitGroupsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	unitGroups := make([]domain.UpsertUnitGroupParams, 0, len(req.UnitGroups))
	for _, ug := range req.UnitGroups {
		conversions := make([]domain.UpsertUnitConversionParams, 0, len(ug.UnitConversions))
		for _, c := range ug.UnitConversions {
			conversions = append(conversions, domain.UpsertUnitConversionParams{
				Unit:               unitIdentifierFromProto(c.Unit),
				DiscountPercentage: c.DiscountPercentage,
			})
		}
		unitGroups = append(unitGroups, domain.UpsertUnitGroupParams{
			Name:            ug.Name,
			Notes:           ug.Notes,
			Type:            ug.Type,
			BaseUnit:        unitIdentifierFromProto(ug.BaseUnit),
			UnitConversions: conversions,
		})
	}

	job, apiErr := h.unitGroupSvc.BulkUpsertUnitGroups(ctx, domain.BulkUpsertUnitGroupsParams{UnitGroups: unitGroups})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.BulkUpsertUnitGroupsResponse{Job: jobToProto(job)}, nil
}
