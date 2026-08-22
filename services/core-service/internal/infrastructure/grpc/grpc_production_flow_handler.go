package grpc

import (
	"context"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/contracts"
	pb "github.com/open-mrp/api/shared/proto/core"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func productionFlowStepToProto(step *domain.ProductionFlowStep) *pb.ProductionFlowStepInfo {
	if step == nil {
		return nil
	}

	info := &pb.ProductionFlowStepInfo{
		Id:             step.ID,
		Name:           step.Name,
		Notes:          step.Notes,
		InStepIds:      step.InStepIDs,
		OutStepIds:     step.OutStepIDs,
		LevelingFactor: step.LevelingFactor,
		Allowances:     step.Allowances,
		MachineIds:     step.MachineIDs,
		CreatedAt:      timestamppb.New(step.CreatedAt),
		UpdatedAt:      timestamppb.New(step.UpdatedAt),
	}

	if step.ScanningStationID != nil {
		info.ScanningStationId = step.ScanningStationID
	}
	if step.DepartmentID != nil {
		info.DepartmentId = step.DepartmentID
	}

	// Production info
	info.Production = &pb.ProductionFlowProductionInfo{
		Id:       step.Production.ID,
		ItemId:   step.Production.ProducedItem.ID,
		ItemSku:  step.Production.ProducedItem.SKU,
		Quantity: productionFlowQuantityToProto(step.Production.Quantity),
	}

	// Consumptions
	consumptions := make([]*pb.ProductionFlowConsumptionInfo, 0, len(step.Consumptions))
	for _, c := range step.Consumptions {
		ci := &pb.ProductionFlowConsumptionInfo{
			Id:            c.ID,
			ItemId:        c.ConsumedItem.ID,
			ItemSku:       c.ConsumedItem.SKU,
			Quantity:      productionFlowQuantityToProto(c.Quantity),
			WasteQuantity: productionFlowQuantityToProto(c.WasteQuantity),
			Instructions:  c.Instructions,
			CreatedAt:     timestamppb.New(c.CreatedAt),
			UpdatedAt:     timestamppb.New(c.UpdatedAt),
		}
		consumptions = append(consumptions, ci)
	}
	info.Consumptions = consumptions

	// Rates
	if step.LaborRate != nil {
		info.LaborRate = flowRateToProto(step.LaborRate)
	}
	if step.LaborTime != nil {
		info.LaborTime = flowRateToProto(step.LaborTime)
	}
	if step.OverheadRate != nil {
		info.OverheadRate = flowRateToProto(step.OverheadRate)
	}

	return info
}

func productionFlowQuantityToProto(q domain.BatchQuantity) *pb.QuantityInfo {
	return &pb.QuantityInfo{
		Id:               q.ID,
		Value:            q.Measure.String(),
		UnitId:           q.Unit.ID,
		UnitName:         q.Unit.Name,
		UnitAbbreviation: q.Unit.Abbreviation,
		UnitType:         q.Unit.Type,
		CreatedAt:        timestamppb.New(q.Unit.CreatedAt),
		UpdatedAt:        timestamppb.New(q.Unit.UpdatedAt),
		UnitDetail:       lightUnitToProto(&q.Unit),
	}
}

func flowRateToProto(r *domain.FlowRate) *pb.RateInfo {
	if r == nil {
		return nil
	}
	return &pb.RateInfo{
		Id:                r.ID,
		Value:             r.Value,
		NumeratorUnitId:   r.NumeratorUnitID,
		DenominatorUnitId: r.DenominatorUnitID,
	}
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
