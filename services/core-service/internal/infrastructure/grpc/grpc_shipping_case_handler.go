package grpc

import (
	"context"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/contracts"
	pb "github.com/open-mrp/api/shared/proto/core"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type shippingCaseGRPCHandler struct {
	pb.UnimplementedCoreShippingCaseServiceServer

	shippingCaseSvc domain.ShippingCaseSvc
}

func shippingCaseToProto(sc *domain.ShippingCase) *pb.ShippingCaseInfo {
	if sc == nil {
		return nil
	}

	info := &pb.ShippingCaseInfo{
		Id:                                 sc.ID,
		Number:                             sc.Number,
		ShipmentId:                         sc.ShipmentID,
		ShipmentNumber:                     &sc.ShipmentNumber,
		ShipmentStatusCode:                 &sc.ShipmentStatusCode,
		ShipmentStatusName:                 &sc.ShipmentStatusName,
		ShipmentCreatedAt:                  timestamppb.New(sc.ShipmentCreatedAt),
		ShipmentUpdatedAt:                  timestamppb.New(sc.ShipmentUpdatedAt),
		CarrierId:                          sc.CarrierID,
		CarrierName:                        sc.CarrierName,
		CarrierIsPortalEnabled:             &sc.CarrierIsPortalEnabled,
		CarrierCreatedAt:                   timestamppb.New(sc.CarrierCreatedAt),
		CarrierUpdatedAt:                   timestamppb.New(sc.CarrierUpdatedAt),
		CreatedAt:                          timestamppb.New(sc.CreatedAt),
		UpdatedAt:                          timestamppb.New(sc.UpdatedAt),
		FreightAmountId:                    sc.FreightAmountID,
		FreightAmountValue:                 sc.FreightAmountValue,
		FreightAmountUnitId:                sc.FreightAmountUnitID,
		FreightAmountUnitName:              sc.FreightAmountUnitName,
		FreightAmountUnitAbbreviation:      sc.FreightAmountUnitAbbreviation,
		FreightAmountUnitType:              sc.FreightAmountUnitType,
		FreightAmountUnitRatioNumerator:    sc.FreightAmountUnitRatioNumerator,
		FreightAmountUnitRatioDenominator:  sc.FreightAmountUnitRatioDenominator,
		FreightAmountUnitOffsetNumerator:   sc.FreightAmountUnitOffsetNumerator,
		FreightAmountUnitOffsetDenominator: sc.FreightAmountUnitOffsetDenominator,
		FreightAmountUnitCreatedAt:         timestamppb.New(sc.FreightAmountUnitCreatedAt),
		FreightAmountUnitUpdatedAt:         timestamppb.New(sc.FreightAmountUnitUpdatedAt),
		FreightWeightId:                    sc.FreightWeightID,
		FreightWeightValue:                 sc.FreightWeightValue,
		FreightWeightUnitId:                sc.FreightWeightUnitID,
		FreightWeightUnitName:              sc.FreightWeightUnitName,
		FreightWeightUnitAbbreviation:      sc.FreightWeightUnitAbbreviation,
		FreightWeightUnitType:              sc.FreightWeightUnitType,
		FreightWeightUnitRatioNumerator:    sc.FreightWeightUnitRatioNumerator,
		FreightWeightUnitRatioDenominator:  sc.FreightWeightUnitRatioDenominator,
		FreightWeightUnitOffsetNumerator:   sc.FreightWeightUnitOffsetNumerator,
		FreightWeightUnitOffsetDenominator: sc.FreightWeightUnitOffsetDenominator,
		FreightWeightUnitCreatedAt:         timestamppb.New(sc.FreightWeightUnitCreatedAt),
		FreightWeightUnitUpdatedAt:         timestamppb.New(sc.FreightWeightUnitUpdatedAt),
	}

	if sc.SSCC != nil {
		info.Sscc = sc.SSCC
	}
	if sc.TrackingNumber != nil {
		info.TrackingNumber = sc.TrackingNumber
	}
	if sc.ShippedAt != nil {
		info.ShippedAt = timestamppb.New(*sc.ShippedAt)
	}

	return info
}

func (h *shippingCaseGRPCHandler) GetShippingCase(ctx context.Context, req *pb.GetShippingCaseRequest) (*pb.GetShippingCaseResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	sc, apiErr := h.shippingCaseSvc.GetShippingCase(ctx, "", req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetShippingCaseResponse{
		ShippingCase: shippingCaseToProto(sc),
	}, nil
}

func (h *shippingCaseGRPCHandler) UpdateShippingCase(ctx context.Context, req *pb.UpdateShippingCaseRequest) (*pb.UpdateShippingCaseResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateShippingCaseParams{
		ShippingCaseID:      req.Id,
		TrackingNumber:      req.TrackingNumber,
		FreightAmountValue:  req.FreightAmountValue,
		FreightAmountUnitID: req.FreightAmountUnitId,
		FreightWeightValue:  req.FreightWeightValue,
		FreightWeightUnitID: req.FreightWeightUnitId,
	}

	sc, apiErr := h.shippingCaseSvc.UpdateShippingCase(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateShippingCaseResponse{
		ShippingCase: shippingCaseToProto(sc),
	}, nil
}

// Corrects the tracking number of a shipping case that has already shipped.
func (h *shippingCaseGRPCHandler) AdminUpdateShippingCaseTracking(ctx context.Context, req *pb.AdminUpdateShippingCaseTrackingRequest) (*pb.AdminUpdateShippingCaseTrackingResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	sc, apiErr := h.shippingCaseSvc.AdminUpdateShippingCaseTracking(ctx, domain.AdminUpdateShippingCaseTrackingParams{
		ShippingCaseID: req.Id,
		TrackingNumber: req.TrackingNumber,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.AdminUpdateShippingCaseTrackingResponse{
		ShippingCase: shippingCaseToProto(sc),
	}, nil
}

func (h *shippingCaseGRPCHandler) DeleteShippingCase(ctx context.Context, req *pb.DeleteShippingCaseRequest) (*pb.DeleteShippingCaseResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.shippingCaseSvc.DeleteShippingCase(ctx, "", req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.DeleteShippingCaseResponse{}, nil
}

func (h *shippingCaseGRPCHandler) GetShippingCaseLabel(ctx context.Context, req *pb.GetShippingCaseLabelRequest) (*pb.GetShippingCaseLabelResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	url, apiErr := h.shippingCaseSvc.GetShippingCaseLabel(ctx, "", req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetShippingCaseLabelResponse{
		Url: url,
	}, nil
}
