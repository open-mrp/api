package grpc

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/core"

	"google.golang.org/protobuf/types/known/emptypb"
)

func productionFlowStepToProto(step *domain.ProductionFlowStep) *pb.ProductionFlowStepInfo {
	if step == nil {
		return nil
	}

	info := &pb.ProductionFlowStepInfo{
		Id:             step.ID,
		Name:           step.Name,
		InStepIds:      step.InStepIDs,
		OutStepIds:     step.OutStepIDs,
		LevelingFactor: step.LevelingFactor,
		Allowances:     step.Allowances,
	}

	if step.ScanningStationID != nil {
		info.ScanningStationId = step.ScanningStationID
	}

	// Production info
	info.Production = &pb.ProductionFlowProductionInfo{
		Id:      step.Production.ID,
		ItemId:  step.Production.ProducedItem.ID,
		ItemSku: step.Production.ProducedItem.SKU,
		Quantity: &pb.QuantityInfo{
			Id:               step.Production.Quantity.ID,
			Value:            step.Production.Quantity.Measure.String(),
			UnitId:           step.Production.Quantity.Unit.ID,
			UnitAbbreviation: step.Production.Quantity.Unit.Abbreviation,
			UnitType:         step.Production.Quantity.Unit.Type,
		},
	}

	// Consumptions
	consumptions := make([]*pb.ProductionFlowConsumptionInfo, 0, len(step.Consumptions))
	for _, c := range step.Consumptions {
		ci := &pb.ProductionFlowConsumptionInfo{
			Id:      c.ID,
			ItemId:  c.ConsumedItem.ID,
			ItemSku: c.ConsumedItem.SKU,
			Quantity: &pb.QuantityInfo{
				Id:               c.Quantity.ID,
				Value:            c.Quantity.Measure.String(),
				UnitId:           c.Quantity.Unit.ID,
				UnitAbbreviation: c.Quantity.Unit.Abbreviation,
				UnitType:         c.Quantity.Unit.Type,
			},
			WasteQuantity: &pb.QuantityInfo{
				Id:               c.WasteQuantity.ID,
				Value:            c.WasteQuantity.Measure.String(),
				UnitId:           c.WasteQuantity.Unit.ID,
				UnitAbbreviation: c.WasteQuantity.Unit.Abbreviation,
				UnitType:         c.WasteQuantity.Unit.Type,
			},
			Instructions: c.Instructions,
		}
		consumptions = append(consumptions, ci)
	}
	info.Consumptions = consumptions

	// Rates
	if step.LaborRate != nil {
		info.LaborRate = &pb.RateInfo{
			Id:                step.LaborRate.ID,
			Value:             step.LaborRate.Value,
			NumeratorUnitId:   step.LaborRate.NumeratorUnitID,
			DenominatorUnitId: step.LaborRate.DenominatorUnitID,
		}
	}
	if step.LaborTime != nil {
		info.LaborTime = &pb.RateInfo{
			Id:                step.LaborTime.ID,
			Value:             step.LaborTime.Value,
			NumeratorUnitId:   step.LaborTime.NumeratorUnitID,
			DenominatorUnitId: step.LaborTime.DenominatorUnitID,
		}
	}
	if step.OverheadRate != nil {
		info.OverheadRate = &pb.RateInfo{
			Id:                step.OverheadRate.ID,
			Value:             step.OverheadRate.Value,
			NumeratorUnitId:   step.OverheadRate.NumeratorUnitID,
			DenominatorUnitId: step.OverheadRate.DenominatorUnitID,
		}
	}

	return info
}

func (h *gRPCHandler) GetProductionFlow(ctx context.Context, req *pb.GetProductionFlowRequest) (*pb.GetProductionFlowResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	steps, apiErr := h.productionFlowSvc.GetProductionFlow(ctx, req.ItemId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	protoSteps := make([]*pb.ProductionFlowStepInfo, 0, len(steps))
	for _, step := range steps {
		protoSteps = append(protoSteps, productionFlowStepToProto(step))
	}

	return &pb.GetProductionFlowResponse{
		Steps: protoSteps,
	}, nil
}

func (h *gRPCHandler) ConnectProductionSteps(ctx context.Context, req *pb.ConnectProductionStepsRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	apiErr := h.productionFlowSvc.ConnectSteps(ctx, req.SourceProductionStepId, req.TargetProductionStepId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}
