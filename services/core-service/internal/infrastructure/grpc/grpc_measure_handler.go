package grpc

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/core"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func RegisterMeasureService(server *grpc.Server, measureSvc domain.MeasureSvc) {
	handler.measureSvc = measureSvc
}

func (h *gRPCHandler) UpdateQuantity(ctx context.Context, req *pb.UpdateQuantityRequest) (*pb.UpdateQuantityResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateQuantityParams{
		QuantityID: req.Id,
		Value:      req.Value,
		UnitID:     req.UnitId,
		ObjectID:   req.ObjectId,
	}
	if req.ObjectType != nil {
		t := constants.ObjectType(*req.ObjectType)
		params.ObjectType = &t
	}

	quantity, apiErr := h.measureSvc.UpdateQuantity(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateQuantityResponse{
		Quantity: enrichedQuantityToProto(quantity),
	}, nil
}

func (h *gRPCHandler) UpdateRate(ctx context.Context, req *pb.UpdateRateRequest) (*pb.UpdateRateResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateRateParams{
		RateID:            req.Id,
		Value:             req.Value,
		NumeratorUnitID:   req.NumeratorUnitId,
		DenominatorUnitID: req.DenominatorUnitId,
		ObjectID:          req.ObjectId,
	}
	if req.ObjectType != nil {
		t := constants.ObjectType(*req.ObjectType)
		params.ObjectType = &t
	}

	rate, apiErr := h.measureSvc.UpdateRate(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateRateResponse{
		Rate: enrichedRateToProto(rate),
	}, nil
}

// enrichedQuantityToProto converts a domain Quantity (with unit details) to proto.
func enrichedQuantityToProto(q *domain.Quantity) *pb.QuantityInfo {
	if q == nil {
		return nil
	}
	return &pb.QuantityInfo{
		Id:               q.ID,
		Value:            q.Value,
		UnitId:           q.UnitID,
		UnitName:         q.UnitName,
		UnitAbbreviation: q.UnitAbbreviation,
		UnitType:         q.UnitType,
		CreatedAt:        timestamppb.New(q.CreatedAt),
		UpdatedAt:        timestamppb.New(q.UpdatedAt),
	}
}

// enrichedRateToProto converts a domain Rate (with unit details) to proto.
func enrichedRateToProto(r *domain.Rate) *pb.RateInfo {
	if r == nil {
		return nil
	}
	return &pb.RateInfo{
		Id:                          r.ID,
		Value:                       r.Value,
		NumeratorUnitId:             r.NumeratorUnitID,
		NumeratorUnitName:           r.NumeratorUnitName,
		NumeratorUnitAbbreviation:   r.NumeratorUnitAbbreviation,
		NumeratorUnitType:           r.NumeratorUnitType,
		DenominatorUnitId:           r.DenominatorUnitID,
		DenominatorUnitName:         r.DenominatorUnitName,
		DenominatorUnitAbbreviation: r.DenominatorUnitAbbreviation,
		DenominatorUnitType:         r.DenominatorUnitType,
		CreatedAt:                   timestamppb.New(r.CreatedAt),
		UpdatedAt:                   timestamppb.New(r.UpdatedAt),
	}
}
