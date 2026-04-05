package grpc

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/core"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func carrierToProto(c *domain.Carrier) *pb.CarrierInfo {
	if c == nil {
		return nil
	}
	info := &pb.CarrierInfo{
		Id:                     c.ID,
		Name:                   c.Name,
		Code:                   c.Code,
		ShippoCarrierAccountId: c.ShippoCarrierAccountID,
		AccountNumber:          c.AccountNumber,
		IsPortalEnabled:        c.IsPortalEnabled,
		IsDefault:              c.AccountID == nil,
		CreatedAt:              timestamppb.New(c.CreatedAt),
		UpdatedAt:              timestamppb.New(c.UpdatedAt),
		AccountId:              c.AccountID,
	}
	if c.DeletedAt != nil {
		info.DeletedAt = timestamppb.New(*c.DeletedAt)
	}
	if c.ServiceLevels != nil {
		info.ServiceLevels = make([]*pb.ServiceLevelInfo, len(c.ServiceLevels))
		for i, sl := range c.ServiceLevels {
			info.ServiceLevels[i] = serviceLevelToProto(sl)
		}
	}
	return info
}

func (h *gRPCHandler) ListCarriers(ctx context.Context, req *pb.ListCarriersRequest) (*pb.ListCarriersResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListCarriersParams{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	result, apiErr := h.carrierSvc.ListCarriers(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbCarriers := make([]*pb.CarrierInfo, len(result.Carriers))
	for i, c := range result.Carriers {
		pbCarriers[i] = carrierToProto(c)
	}

	return &pb.ListCarriersResponse{
		Carriers: pbCarriers,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) GetCarrier(ctx context.Context, req *pb.GetCarrierRequest) (*pb.GetCarrierResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	carrier, apiErr := h.carrierSvc.GetCarrier(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetCarrierResponse{
		Carrier: carrierToProto(carrier),
	}, nil
}

func (h *gRPCHandler) CreateCarrier(ctx context.Context, req *pb.CreateCarrierRequest) (*pb.CreateCarrierResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.CreateCarrierParams{
		Name:            req.Name,
		Code:            req.Code,
		AccountNumber:   req.AccountNumber,
		IsPortalEnabled: req.IsPortalEnabled,
	}

	carrier, apiErr := h.carrierSvc.CreateCarrier(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateCarrierResponse{
		Carrier: carrierToProto(carrier),
	}, nil
}

func (h *gRPCHandler) UpdateCarrier(ctx context.Context, req *pb.UpdateCarrierRequest) (*pb.UpdateCarrierResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateCarrierParams{
		CarrierID:       req.Id,
		Name:            req.Name,
		IsPortalEnabled: req.IsPortalEnabled,
	}

	carrier, apiErr := h.carrierSvc.UpdateCarrier(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateCarrierResponse{
		Carrier: carrierToProto(carrier),
	}, nil
}

func (h *gRPCHandler) DeleteCarrier(ctx context.Context, req *pb.DeleteCarrierRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.carrierSvc.DeleteCarrier(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *gRPCHandler) InitiateCarrierOAuth(ctx context.Context, req *pb.InitiateCarrierOAuthRequest) (*pb.InitiateCarrierOAuthResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	oauthURL, apiErr := h.carrierSvc.InitiateOAuth(ctx, req.CarrierId, req.RedirectUri, req.State)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.InitiateCarrierOAuthResponse{
		OauthUrl: oauthURL,
	}, nil
}

func (h *gRPCHandler) GetCarrierOAuthStatus(ctx context.Context, req *pb.GetCarrierOAuthStatusRequest) (*pb.GetCarrierOAuthStatusResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	status, apiErr := h.carrierSvc.GetOAuthStatus(ctx, req.CarrierId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetCarrierOAuthStatusResponse{
		Status: status,
	}, nil
}

func (h *gRPCHandler) SyncServiceLevels(ctx context.Context, req *pb.SyncServiceLevelsRequest) (*pb.SyncServiceLevelsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	carrier, apiErr := h.carrierSvc.SyncOptions(ctx, req.CarrierId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.SyncServiceLevelsResponse{
		Carrier: carrierToProto(carrier),
	}, nil
}
