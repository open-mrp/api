package grpc

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/core"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func serviceLevelToProto(o *domain.ServiceLevel) *pb.ServiceLevelInfo {
	if o == nil {
		return nil
	}
	return &pb.ServiceLevelInfo{
		Id:                o.ID,
		Name:              o.Name,
		Code:              o.Code,
		ServiceLevelToken: o.ServiceLevelToken,
		IsPortalEnabled:   o.IsPortalEnabled,
		IsDefault:         o.IsDefault,
		CarrierId:         o.CarrierID,
		CreatedAt:         timestamppb.New(o.CreatedAt),
		UpdatedAt:         timestamppb.New(o.UpdatedAt),
		AccountId:         o.AccountID,
	}
}

func (h *gRPCHandler) ListServiceLevels(ctx context.Context, req *pb.ListServiceLevelsRequest) (*pb.ListServiceLevelsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListServiceLevelsParams{
		CarrierID: req.CarrierId,
		Cursor:    req.Cursor,
		Limit:     req.Limit,
		Query:     req.Query,
	}

	result, apiErr := h.serviceLevelSvc.ListServiceLevels(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbLevels := make([]*pb.ServiceLevelInfo, len(result.ServiceLevels))
	for i, o := range result.ServiceLevels {
		pbLevels[i] = serviceLevelToProto(o)
	}

	return &pb.ListServiceLevelsResponse{
		ServiceLevels: pbLevels,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) GetServiceLevel(ctx context.Context, req *pb.GetServiceLevelRequest) (*pb.GetServiceLevelResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	serviceLevel, apiErr := h.serviceLevelSvc.GetServiceLevel(ctx, req.CarrierId, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetServiceLevelResponse{
		ServiceLevel: serviceLevelToProto(serviceLevel),
	}, nil
}

func (h *gRPCHandler) CreateServiceLevel(ctx context.Context, req *pb.CreateServiceLevelRequest) (*pb.CreateServiceLevelResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.CreateServiceLevelParams{
		CarrierID:         req.CarrierId,
		Name:              req.Name,
		Code:              req.Code,
		ServiceLevelToken: req.ServiceLevelToken,
		IsPortalEnabled:   req.GetIsPortalEnabled(),
		IsDefault:         req.GetIsDefault(),
	}

	serviceLevel, apiErr := h.serviceLevelSvc.CreateServiceLevel(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateServiceLevelResponse{
		ServiceLevel: serviceLevelToProto(serviceLevel),
	}, nil
}

func (h *gRPCHandler) UpdateServiceLevel(ctx context.Context, req *pb.UpdateServiceLevelRequest) (*pb.UpdateServiceLevelResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateServiceLevelParams{
		ServiceLevelID:  req.Id,
		CarrierID:       req.CarrierId,
		Name:            req.Name,
		Code:            req.Code,
		IsPortalEnabled: req.IsPortalEnabled,
		IsDefault:       req.IsDefault,
	}

	serviceLevel, apiErr := h.serviceLevelSvc.UpdateServiceLevel(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateServiceLevelResponse{
		ServiceLevel: serviceLevelToProto(serviceLevel),
	}, nil
}

func (h *gRPCHandler) DeleteServiceLevel(ctx context.Context, req *pb.DeleteServiceLevelRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.serviceLevelSvc.DeleteServiceLevel(ctx, req.CarrierId, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}
