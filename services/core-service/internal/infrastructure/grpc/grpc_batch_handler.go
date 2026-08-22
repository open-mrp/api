package grpc

import (
	"context"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/contracts"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func batchQuantityToProto(q *domain.BatchQuantity) *pb.BatchQuantityInfo {
	if q == nil {
		return nil
	}
	return &pb.BatchQuantityInfo{
		Id:               q.ID,
		Measure:          q.Measure.String(),
		UnitId:           q.Unit.ID,
		UnitAbbreviation: q.Unit.Abbreviation,
		UnitType:         q.Unit.Type,
	}
}

func batchToProto(b *domain.Batch) *pb.BatchInfo {
	if b == nil {
		return nil
	}

	pbBatch := &pb.BatchInfo{
		Id:        b.ID,
		ItemId:    b.Item.ID,
		ItemSku:   b.Item.SKU,
		Quantity:  batchQuantityToProto(&b.Quantity),
		Seconds:   batchQuantityToProto(b.Seconds),
		Waste:     batchQuantityToProto(b.Waste),
		CreatedAt: timestamppb.New(b.CreatedAt),
		UpdatedAt: timestamppb.New(b.UpdatedAt),
	}

	if b.ScanningStation != nil {
		pbBatch.ScanningStationId = &b.ScanningStation.ID
		pbBatch.ScanningStationName = &b.ScanningStation.Name
	}

	if b.ProductionStep != nil {
		pbBatch.ProductionStepId = &b.ProductionStep.ID
		pbBatch.ProductionStepName = &b.ProductionStep.Name
	}

	if b.ProductionRun != nil {
		pbBatch.ProductionRunId = &b.ProductionRun.ID
		pbBatch.ProductionRunNumber = &b.ProductionRun.Number
	}

	if b.Machines != nil {
		pbBatch.Machines = make([]*pb.LightMachineInfo, len(b.Machines))
		for i, m := range b.Machines {
			pbBatch.Machines[i] = &pb.LightMachineInfo{
				Id:   m.ID,
				Name: m.Name,
			}
		}
	}

	if b.DepartmentID != nil {
		pbBatch.DepartmentId = b.DepartmentID
	}
	if b.DepartmentName != nil {
		pbBatch.DepartmentName = b.DepartmentName
	}

	if b.Lots != nil {
		pbBatch.Lots = make([]*pb.BatchLotInfo, len(b.Lots))
		for i, l := range b.Lots {
			pbBatch.Lots[i] = &pb.BatchLotInfo{
				LotNumber: l.LotNumber,
				Type:      l.Type,
			}
		}
	}

	pbBatch.InputBatchIds = b.InputBatchIDs
	pbBatch.OutputBatchIds = b.OutputBatchIDs

	if b.ClosedAt != nil {
		pbBatch.ClosedAt = timestamppb.New(*b.ClosedAt)
	}

	if b.ScannedAt != nil {
		pbBatch.ScannedAt = timestamppb.New(*b.ScannedAt)
	}

	return pbBatch
}

func baseBatchToProto(b *domain.BaseBatch) *pb.BaseBatchInfo {
	if b == nil {
		return nil
	}

	pbBatch := &pb.BaseBatchInfo{
		Id:        b.ID,
		ItemId:    b.Item.ID,
		ItemSku:   b.Item.SKU,
		Quantity:  batchQuantityToProto(&b.Quantity),
		Seconds:   batchQuantityToProto(b.Seconds),
		Waste:     batchQuantityToProto(b.Waste),
		CreatedAt: timestamppb.New(b.CreatedAt),
		UpdatedAt: timestamppb.New(b.UpdatedAt),
	}

	if b.ScanningStation != nil {
		pbBatch.ScanningStationId = &b.ScanningStation.ID
		pbBatch.ScanningStationName = &b.ScanningStation.Name
	}

	if b.ProductionStep != nil {
		pbBatch.ProductionStepId = &b.ProductionStep.ID
		pbBatch.ProductionStepName = &b.ProductionStep.Name
	}

	if b.ProductionRun != nil {
		pbBatch.ProductionRunId = &b.ProductionRun.ID
		pbBatch.ProductionRunNumber = &b.ProductionRun.Number
	}

	if b.DepartmentID != nil {
		pbBatch.DepartmentId = b.DepartmentID
	}
	if b.DepartmentName != nil {
		pbBatch.DepartmentName = b.DepartmentName
	}

	if b.ClosedAt != nil {
		pbBatch.ClosedAt = timestamppb.New(*b.ClosedAt)
	}

	if b.ScannedAt != nil {
		pbBatch.ScannedAt = timestamppb.New(*b.ScannedAt)
	}

	return pbBatch
}

func batchFlowNodeToProto(n *domain.BatchFlowNode) *pb.BatchFlowNodeInfo {
	if n == nil {
		return nil
	}
	return &pb.BatchFlowNodeInfo{
		Batch:          batchToProto(&n.Batch),
		InputBatchIds:  n.InputBatchIDs,
		OutputBatchIds: n.OutputBatchIDs,
	}
}

func batchQuantityFromProto(q *pb.BatchQuantityInfo) domain.BatchQuantity {
	measure, _ := decimal.NewFromString(q.GetMeasure())
	return domain.BatchQuantity{
		ID:      q.GetId(),
		Measure: measure,
		Unit: domain.LightUnit{
			ID:           q.GetUnitId(),
			Abbreviation: q.GetUnitAbbreviation(),
			Type:         q.GetUnitType(),
		},
	}
}

func batchQuantityPtrFromProto(q *pb.BatchQuantityInfo) *domain.BatchQuantity {
	if q == nil {
		return nil
	}
	bq := batchQuantityFromProto(q)
	return &bq
}

func (h *gRPCHandler) GetBatchFlow(ctx context.Context, req *pb.GetBatchFlowRequest) (*pb.GetBatchFlowResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	nodes, apiErr := h.batchSvc.GetBatchFlow(ctx, req.BatchId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbNodes := make([]*pb.BatchFlowNodeInfo, len(nodes))
	for i, n := range nodes {
		pbNodes[i] = batchFlowNodeToProto(&n)
	}

	return &pb.GetBatchFlowResponse{
		Nodes: pbNodes,
	}, nil
}

func (h *gRPCHandler) ListBatchesByScanningStation(ctx context.Context, req *pb.ListBatchesByScanningStationRequest) (*pb.ListBatchesByScanningStationResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListBatchesByScanningStationParams{
		ScanningStationID: req.ScanningStationId,
		Cursor:            req.Cursor,
		Limit:             req.Limit,
		Query:             req.Query,
	}

	result, apiErr := h.batchSvc.ListBatchesByScanningStation(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbBatches := make([]*pb.BatchInfo, len(result.Batches))
	for i, b := range result.Batches {
		pbBatches[i] = batchToProto(b)
	}

	return &pb.ListBatchesByScanningStationResponse{
		Batches: pbBatches,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) GetBatchPossibleNextSteps(ctx context.Context, req *pb.GetBatchPossibleNextStepsRequest) (*pb.GetBatchPossibleNextStepsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	steps, apiErr := h.batchSvc.GetPossibleNextSteps(ctx, req.ScanningStationId, req.BatchId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbSteps := make([]*pb.ScanningProductionStepInfoProto, len(steps))
	for i, s := range steps {
		pbSteps[i] = &pb.ScanningProductionStepInfoProto{
			Id:          s.ID,
			Name:        s.Name,
			IsMultiPart: s.IsMultiPart,
		}
	}

	return &pb.GetBatchPossibleNextStepsResponse{
		Steps: pbSteps,
	}, nil
}

func (h *gRPCHandler) AnalyzeOpenBatches(ctx context.Context, req *pb.AnalyzeOpenBatchesRequest) (*pb.AnalyzeOpenBatchesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	summaries, apiErr := h.batchSvc.AnalyzeOpenBatches(ctx, req.ItemIds, req.ProductLineIds)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbSummaries := make([]*pb.OpenBatchSummaryInfo, len(summaries))
	for i, s := range summaries {
		pbSummaries[i] = &pb.OpenBatchSummaryInfo{
			DepartmentName: s.DepartmentName,
			Item: &pb.OpenBatchSummaryItemProto{
				Id:  s.ItemID,
				Sku: s.ItemName,
			},
			ScanningStation: &pb.OpenBatchSummaryScanningStationProto{
				Id: s.ScanningStationID,
			},
			Count: s.Count.String(),
			Unit:  s.Unit,
		}
	}

	return &pb.AnalyzeOpenBatchesResponse{
		Summaries: pbSummaries,
	}, nil
}

func (h *gRPCHandler) InitializeBatch(ctx context.Context, req *pb.InitializeBatchRequest) (*pb.InitializeBatchResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	batch, apiErr := h.batchSvc.InitializeBatch(ctx, req.BatchId, req.ScanningStationId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.InitializeBatchResponse{
		Batch: baseBatchToProto(batch),
	}, nil
}

func (h *gRPCHandler) MoveBatches(ctx context.Context, req *pb.MoveBatchesRequest) (*pb.MoveBatchesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.MoveBatchesParams{
		BatchIDs:          req.BatchIds,
		ProductionStepID:  req.ProductionStepId,
		ScanningStationID: req.ScanningStationId,
	}

	batch, apiErr := h.batchSvc.MoveBatches(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.MoveBatchesResponse{
		Batch: baseBatchToProto(batch),
	}, nil
}

func (h *gRPCHandler) MergeBatches(ctx context.Context, req *pb.MergeBatchesRequest) (*pb.MergeBatchesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.MergeBatchesParams{
		BatchIDs:          req.BatchIds,
		ScanningStationID: req.ScanningStationId,
		ProductionStepID:  req.ProductionStepId,
	}

	batch, apiErr := h.batchSvc.MergeBatches(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.MergeBatchesResponse{
		Batch: baseBatchToProto(batch),
	}, nil
}

func (h *gRPCHandler) SplitBatch(ctx context.Context, req *pb.SplitBatchRequest) (*pb.SplitBatchResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.SplitBatchParams{
		BatchIDs:          req.BatchIds,
		ScanningStationID: req.ScanningStationId,
		ProductionStepID:  req.ProductionStepId,
		Firsts:            batchQuantityFromProto(req.Firsts),
		Seconds:           batchQuantityPtrFromProto(req.Seconds),
		Waste:             batchQuantityPtrFromProto(req.Waste),
		CloseBatch:        req.CloseBatch,
	}

	batch, apiErr := h.batchSvc.SplitBatch(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.SplitBatchResponse{
		Batch: baseBatchToProto(batch),
	}, nil
}

func (h *gRPCHandler) GetRemainingQuantityToSplit(ctx context.Context, req *pb.GetRemainingQuantityToSplitRequest) (*pb.GetRemainingQuantityToSplitResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	quantity, apiErr := h.batchSvc.GetRemainingQuantityToSplit(ctx, req.BatchIds, req.ProductionStepId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetRemainingQuantityToSplitResponse{
		Quantity: batchQuantityToProto(quantity),
	}, nil
}

func (h *gRPCHandler) GetScanningStationConsumption(ctx context.Context, req *pb.GetScanningStationConsumptionRequest) (*pb.GetScanningStationConsumptionResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.GetConsumptionParams{
		ScanningStationID: req.ScanningStationId,
		BatchIDs:          req.BatchIds,
		ProductionStepID:  req.ProductionStepId,
		SplitQuantity:     batchQuantityPtrFromProto(req.SplitQuantity),
	}

	consumptions, apiErr := h.batchSvc.GetScanningStationConsumption(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbConsumptions := make([]*pb.ScanningConsumptionInfo, len(consumptions))
	for i, c := range consumptions {
		pbConsumptions[i] = &pb.ScanningConsumptionInfo{
			Sku:              c.SKU,
			DemandMeasure:    c.DemandMeasure,
			DemandUnit:       c.DemandUnit,
			InventoryMeasure: c.InventoryMeasure,
			InventoryUnit:    c.InventoryUnit,
			Instructions:     c.Instructions,
		}
	}

	return &pb.GetScanningStationConsumptionResponse{
		Consumptions: pbConsumptions,
	}, nil
}

func (h *gRPCHandler) CloseBatch(ctx context.Context, req *pb.CloseBatchRequest) (*pb.CloseBatchResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	batch, apiErr := h.batchSvc.CloseBatch(ctx, req.BatchId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CloseBatchResponse{
		Batch: baseBatchToProto(batch),
	}, nil
}

func (h *gRPCHandler) DeleteBatch(ctx context.Context, req *pb.DeleteBatchRequest) (*pb.DeleteBatchResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	batch, apiErr := h.batchSvc.DeleteBatch(ctx, req.BatchId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.DeleteBatchResponse{
		Batch: baseBatchToProto(batch),
	}, nil
}

func (h *gRPCHandler) DeleteManyBatches(ctx context.Context, req *pb.DeleteManyBatchesRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.batchSvc.DeleteManyBatches(ctx, req.BatchIds)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}
