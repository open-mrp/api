package grpc

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/core"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func consumptionToProto(c *domain.Consumption) *pb.ConsumptionInfo {
	if c == nil {
		return nil
	}
	info := &pb.ConsumptionInfo{
		Id:            c.ID,
		Quantity:      quantityToProto(&c.Quantity),
		WasteQuantity: quantityToProto(&c.WasteQuantity),
		ItemId:        c.ItemID,
		ItemSku:       c.ItemSKU,
		ItemTypeCode:  c.ItemTypeCode,
		Instructions:  c.Instructions,
		CreatedAt:     timestamppb.New(c.CreatedAt),
		UpdatedAt:     timestamppb.New(c.UpdatedAt),
	}
	if c.ItemDescription != nil {
		info.ItemDescription = c.ItemDescription
	}
	return info
}

func (h *gRPCHandler) GetConsumption(ctx context.Context, req *pb.GetConsumptionRequest) (*pb.GetConsumptionResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	consumption, apiErr := h.consumptionSvc.GetConsumption(ctx, req.ProductionStepId, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetConsumptionResponse{
		Consumption: consumptionToProto(consumption),
	}, nil
}

func (h *gRPCHandler) CreateConsumption(ctx context.Context, req *pb.CreateConsumptionRequest) (*pb.CreateConsumptionResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	consumption, apiErr := h.consumptionSvc.CreateConsumption(ctx, domain.CreateConsumptionParams{
		ProductionStepID:    req.ProductionStepId,
		ItemID:              req.ItemId,
		QuantityValue:       req.QuantityValue,
		QuantityUnitID:      req.QuantityUnitId,
		WasteQuantityValue:  req.WasteQuantityValue,
		WasteQuantityUnitID: req.WasteQuantityUnitId,
		Instructions:        req.Instructions,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateConsumptionResponse{
		Consumption: consumptionToProto(consumption),
	}, nil
}

func (h *gRPCHandler) UpdateConsumption(ctx context.Context, req *pb.UpdateConsumptionRequest) (*pb.UpdateConsumptionResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	consumption, apiErr := h.consumptionSvc.UpdateConsumption(ctx, domain.UpdateConsumptionParams{
		ProductionStepID:    req.ProductionStepId,
		ConsumptionID:       req.Id,
		ItemID:              req.ItemId,
		QuantityValue:       req.QuantityValue,
		QuantityUnitID:      req.QuantityUnitId,
		WasteQuantityValue:  req.WasteQuantityValue,
		WasteQuantityUnitID: req.WasteQuantityUnitId,
		Instructions:        req.Instructions,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateConsumptionResponse{
		Consumption: consumptionToProto(consumption),
	}, nil
}

func (h *gRPCHandler) DeleteConsumption(ctx context.Context, req *pb.DeleteConsumptionRequest) (*pb.DeleteConsumptionResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	consumption, apiErr := h.consumptionSvc.DeleteConsumption(ctx, domain.DeleteConsumptionParams{
		ProductionStepID: req.ProductionStepId,
		ConsumptionID:    req.Id,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.DeleteConsumptionResponse{
		Consumption: consumptionToProto(consumption),
	}, nil
}
