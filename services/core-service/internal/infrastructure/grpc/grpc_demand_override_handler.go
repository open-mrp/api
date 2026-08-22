package grpc

import (
	"context"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
	pb "github.com/open-mrp/api/shared/proto/core"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type demandOverrideGRPCHandler struct {
	pb.UnimplementedCoreDemandOverrideServiceServer

	demandOverrideSvc domain.DemandOverrideSvc
}

func demandOverrideTypeToProto(t *domain.DemandOverrideType) *pb.DemandOverrideTypeInfo {
	return &pb.DemandOverrideTypeInfo{
		Id:        t.ID,
		Code:      t.Code,
		Name:      t.Name,
		CreatedAt: timestamppb.New(t.CreatedAt),
		UpdatedAt: timestamppb.New(t.UpdatedAt),
	}
}

func demandOverrideToProto(o *domain.DemandOverride) *pb.DemandOverrideInfo {
	info := &pb.DemandOverrideInfo{
		Id:               o.ID,
		ScopeCode:        o.ScopeCode,
		ScopeRefId:       o.ScopeRefID,
		PeriodStartDate:  timestamppb.New(o.PeriodStartDate),
		PeriodEndDate:    timestamppb.New(o.PeriodEndDate),
		OverrideTypeCode: o.OverrideTypeCode,
		Value:            o.Value,
		CreatedById:      o.CreatedByID,
		EffectiveFrom:    timestamppb.New(o.EffectiveFrom),
		IsActive:         o.IsActive,
		CreatedAt:        timestamppb.New(o.CreatedAt),
		UpdatedAt:        timestamppb.New(o.UpdatedAt),
	}
	info.UnitId = o.UnitID
	info.ReasonCode = o.ReasonCode
	info.Note = o.Note
	info.ScopeName = o.ScopeName
	info.ScopeHandle = o.ScopeHandle
	if o.ExpiresAt != nil {
		info.ExpiresAt = timestamppb.New(*o.ExpiresAt)
	}
	return info
}

func (h *demandOverrideGRPCHandler) ListDemandOverrideTypes(ctx context.Context, req *pb.ListDemandOverrideTypesRequest) (*pb.ListDemandOverrideTypesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	types, apiErr := h.demandOverrideSvc.ListDemandOverrideTypes(ctx)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	out := make([]*pb.DemandOverrideTypeInfo, len(types))
	for i, t := range types {
		out[i] = demandOverrideTypeToProto(t)
	}

	return &pb.ListDemandOverrideTypesResponse{Types: out}, nil
}

func (h *demandOverrideGRPCHandler) ListDemandOverrides(ctx context.Context, req *pb.ListDemandOverridesRequest) (*pb.ListDemandOverridesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListDemandOverridesParams{
		Cursor:            req.Cursor,
		Limit:             req.Limit,
		ScopeCodes:        req.ScopeCodes,
		ScopeRefIDs:       req.ScopeRefIds,
		OverrideTypeCodes: req.OverrideTypeCodes,
		IsActive:          req.IsActive,
		Query:             req.Query,
	}

	if req.PeriodStart != nil {
		parsed, err := time.Parse(time.RFC3339, *req.PeriodStart)
		if err != nil {
			return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewValidationErrorWithParam("Invalid date format for starts_at.", "starts_at"))
		}
		params.PeriodStart = &parsed
	}
	if req.PeriodEnd != nil {
		parsed, err := time.Parse(time.RFC3339, *req.PeriodEnd)
		if err != nil {
			return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewValidationErrorWithParam("Invalid date format for ends_at.", "ends_at"))
		}
		params.PeriodEnd = &parsed
	}

	result, apiErr := h.demandOverrideSvc.ListDemandOverrides(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	overrides := make([]*pb.DemandOverrideInfo, len(result.Overrides))
	for i, o := range result.Overrides {
		overrides[i] = demandOverrideToProto(o)
	}

	return &pb.ListDemandOverridesResponse{
		Overrides: overrides,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *demandOverrideGRPCHandler) GetDemandOverride(ctx context.Context, req *pb.GetDemandOverrideRequest) (*pb.GetDemandOverrideResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	override, apiErr := h.demandOverrideSvc.GetDemandOverride(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetDemandOverrideResponse{Override: demandOverrideToProto(override)}, nil
}

func (h *demandOverrideGRPCHandler) CreateDemandOverride(ctx context.Context, req *pb.CreateDemandOverrideRequest) (*pb.CreateDemandOverrideResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	if req.PeriodStartDate == nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewValidationErrorWithParam("period_starts_at is required.", "period_starts_at"))
	}
	if req.PeriodEndDate == nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewValidationErrorWithParam("period_ends_at is required.", "period_ends_at"))
	}

	params := domain.CreateDemandOverrideParams{
		ScopeCode:        req.ScopeCode,
		ScopeRefID:       req.ScopeRefId,
		PeriodStartDate:  req.PeriodStartDate.AsTime(),
		PeriodEndDate:    req.PeriodEndDate.AsTime(),
		OverrideTypeCode: req.OverrideTypeCode,
		Value:            req.Value,
		UnitID:           req.UnitId,
		ReasonCode:       req.ReasonCode,
		Note:             req.Note,
		EffectiveFrom:    downtimeTimePtr(req.EffectiveFrom),
		ExpiresAt:        downtimeTimePtr(req.ExpiresAt),
		IsActive:         req.IsActive,
	}

	override, apiErr := h.demandOverrideSvc.CreateDemandOverride(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateDemandOverrideResponse{Override: demandOverrideToProto(override)}, nil
}

func (h *demandOverrideGRPCHandler) UpdateDemandOverride(ctx context.Context, req *pb.UpdateDemandOverrideRequest) (*pb.UpdateDemandOverrideResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateDemandOverrideParams{
		OverrideID:       req.Id,
		PeriodStartDate:  downtimeTimePtr(req.PeriodStartDate),
		PeriodEndDate:    downtimeTimePtr(req.PeriodEndDate),
		OverrideTypeCode: req.OverrideTypeCode,
		Value:            req.Value,
		UnitID:           field.StringClearableFromProto(req.UnitId),
		ReasonCode:       field.StringClearableFromProto(req.ReasonCode),
		Note:             field.StringClearableFromProto(req.Note),
		ExpiresAt:        field.TimestampClearableFromProto(req.ExpiresAt),
		IsActive:         req.IsActive,
	}

	override, apiErr := h.demandOverrideSvc.UpdateDemandOverride(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateDemandOverrideResponse{Override: demandOverrideToProto(override)}, nil
}

func (h *demandOverrideGRPCHandler) DeleteDemandOverride(ctx context.Context, req *pb.DeleteDemandOverrideRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	if apiErr := h.demandOverrideSvc.DeleteDemandOverride(ctx, req.Id); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *demandOverrideGRPCHandler) BatchGetDemandOverridesByIDs(ctx context.Context, req *pb.BatchGetDemandOverridesByIDsRequest) (*pb.BatchGetDemandOverridesByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	overrides, apiErr := h.demandOverrideSvc.BatchGetDemandOverridesByIDs(ctx, req.Ids)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	out := make([]*pb.DemandOverrideInfo, len(overrides))
	for i, o := range overrides {
		out[i] = demandOverrideToProto(o)
	}

	return &pb.BatchGetDemandOverridesByIDsResponse{Overrides: out}, nil
}
