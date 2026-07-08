package batchep

import (
	"context"

	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/ptrutil"
)

// BatchQuantityPresenter converts a proto BatchQuantityInfo to an apiresource.Quantity.
func BatchQuantityPresenter(q *pb.BatchQuantityInfo) *apiresource.Quantity {
	if q == nil {
		return nil
	}

	var unit *apiresource.Unit
	if q.UnitId != "" {
		unit = &apiresource.Unit{
			ID:           q.UnitId,
			Object:       constants.ObjectTypeUnit,
			Abbreviation: q.UnitAbbreviation,
			Type:         constants.UnitType(q.UnitType),
		}
	}

	norm := apiresource.NormalizeQuantityValue(q.Measure, q.UnitType)
	displayValue := ""
	if q.UnitAbbreviation != "" {
		displayValue = apiresource.FormatDisplayValue(norm, q.UnitAbbreviation, q.UnitType)
	}

	return &apiresource.Quantity{
		ID:           q.Id,
		Object:       constants.ObjectTypeQuantity,
		Value:        norm,
		DisplayValue: displayValue,
		Unit:         unit,
	}
}

// BatchPresenter converts a proto BatchInfo to an apiresource.Batch.
func BatchPresenter(b *pb.BatchInfo) apiresource.Batch {
	if b == nil {
		return apiresource.Batch{}
	}

	var item *apiresource.Item
	if b.ItemId != "" {
		item = &apiresource.Item{
			ID:     b.ItemId,
			Object: constants.ObjectTypeItem,
			SKU:    b.ItemSku,
		}
	}

	var scanningStation *apiresource.ScanningStation
	if b.ScanningStationId != nil && *b.ScanningStationId != "" {
		scanningStation = &apiresource.ScanningStation{
			ID:     *b.ScanningStationId,
			Object: constants.ObjectTypeScanningStation,
			Name:   ptrutil.Deref(b.ScanningStationName),
		}
	}

	var productionStep *apiresource.ProductionStep
	if b.ProductionStepId != nil && *b.ProductionStepId != "" {
		productionStep = &apiresource.ProductionStep{
			ID:     *b.ProductionStepId,
			Object: constants.ObjectTypeProductionStep,
			Name:   ptrutil.Deref(b.ProductionStepName),
		}
	}

	var productionRun *apiresource.ProductionRun
	if b.ProductionRunId != nil && *b.ProductionRunId != "" {
		productionRun = &apiresource.ProductionRun{
			ID:     *b.ProductionRunId,
			Object: constants.ObjectTypeProductionRun,
			Number: ptrutil.Deref(b.ProductionRunNumber),
		}
	}

	machines := make([]apiresource.Machine, len(b.Machines))
	for i, m := range b.Machines {
		machines[i] = apiresource.Machine{
			ID:     m.Id,
			Object: constants.ObjectTypeMachine,
			Name:   m.Name,
		}
	}

	var department *apiresource.Department
	if b.DepartmentId != nil && *b.DepartmentId != "" {
		department = &apiresource.Department{
			ID:     *b.DepartmentId,
			Object: constants.ObjectTypeDepartment,
			Name:   ptrutil.Deref(b.DepartmentName),
		}
	}

	lots := make([]apiresource.BatchLot, len(b.Lots))
	for i, l := range b.Lots {
		lots[i] = apiresource.BatchLot{
			Object:    constants.ObjectTypeBatchLot,
			LotNumber: l.LotNumber,
			Type:      l.Type,
		}
	}

	return apiresource.Batch{
		ID:              b.Id,
		Object:          constants.ObjectTypeBatch,
		Item:            item,
		Quantity:        BatchQuantityPresenter(b.Quantity),
		Seconds:         BatchQuantityPresenter(b.Seconds),
		Waste:           BatchQuantityPresenter(b.Waste),
		ScanningStation: scanningStation,
		Department:      department,
		ProductionStep:  productionStep,
		ProductionRun:   productionRun,
		Machines:        apiresource.NewList(machines, apiresource.PageInfo{}),
		Lots:            apiresource.NewList(lots, apiresource.PageInfo{}),
		InputBatches:    batchReferenceList(b.InputBatchIds),
		OutputBatches:   batchReferenceList(b.OutputBatchIds),
		ClosedAt:        grpcutil.TimestampToTimePtr(b.ClosedAt),
		ScannedAt:       grpcutil.TimestampToTimePtr(b.ScannedAt),
		CreatedAt:       grpcutil.TimestampToTime(b.CreatedAt),
		UpdatedAt:       grpcutil.TimestampToTime(b.UpdatedAt),
	}
}

// BaseBatchPresenter converts a proto BaseBatchInfo to an apiresource.Batch.
// BaseBatchInfo is used for mutation responses and does not include machines.
func BaseBatchPresenter(b *pb.BaseBatchInfo) apiresource.Batch {
	if b == nil {
		return apiresource.Batch{}
	}

	var item *apiresource.Item
	if b.ItemId != "" {
		item = &apiresource.Item{
			ID:     b.ItemId,
			Object: constants.ObjectTypeItem,
			SKU:    b.ItemSku,
		}
	}

	var scanningStation *apiresource.ScanningStation
	if b.ScanningStationId != nil && *b.ScanningStationId != "" {
		scanningStation = &apiresource.ScanningStation{
			ID:     *b.ScanningStationId,
			Object: constants.ObjectTypeScanningStation,
			Name:   ptrutil.Deref(b.ScanningStationName),
		}
	}

	var productionStep *apiresource.ProductionStep
	if b.ProductionStepId != nil && *b.ProductionStepId != "" {
		productionStep = &apiresource.ProductionStep{
			ID:     *b.ProductionStepId,
			Object: constants.ObjectTypeProductionStep,
			Name:   ptrutil.Deref(b.ProductionStepName),
		}
	}

	var productionRun *apiresource.ProductionRun
	if b.ProductionRunId != nil && *b.ProductionRunId != "" {
		productionRun = &apiresource.ProductionRun{
			ID:     *b.ProductionRunId,
			Object: constants.ObjectTypeProductionRun,
			Number: ptrutil.Deref(b.ProductionRunNumber),
		}
	}

	var department *apiresource.Department
	if b.DepartmentId != nil && *b.DepartmentId != "" {
		department = &apiresource.Department{
			ID:     *b.DepartmentId,
			Object: constants.ObjectTypeDepartment,
			Name:   ptrutil.Deref(b.DepartmentName),
		}
	}

	return apiresource.Batch{
		ID:              b.Id,
		Object:          constants.ObjectTypeBatch,
		Item:            item,
		Quantity:        BatchQuantityPresenter(b.Quantity),
		Seconds:         BatchQuantityPresenter(b.Seconds),
		Waste:           BatchQuantityPresenter(b.Waste),
		ScanningStation: scanningStation,
		Department:      department,
		ProductionStep:  productionStep,
		ProductionRun:   productionRun,
		Machines:        apiresource.NewList([]apiresource.Machine{}, apiresource.PageInfo{}),
		ClosedAt:        grpcutil.TimestampToTimePtr(b.ClosedAt),
		ScannedAt:       grpcutil.TimestampToTimePtr(b.ScannedAt),
		CreatedAt:       grpcutil.TimestampToTime(b.CreatedAt),
		UpdatedAt:       grpcutil.TimestampToTime(b.UpdatedAt),
	}
}

// BatchFlowNodePresenter converts a proto BatchFlowNodeInfo to an apiresource.BatchFlowNode.
func BatchFlowNodePresenter(n *pb.BatchFlowNodeInfo) apiresource.BatchFlowNode {
	if n == nil {
		return apiresource.BatchFlowNode{}
	}

	return apiresource.BatchFlowNode{
		Object:        constants.ObjectTypeBatchFlowNode,
		Batch:         BatchPresenter(n.Batch),
		InputBatches:  batchReferenceList(n.InputBatchIds),
		OutputBatches: batchReferenceList(n.OutputBatchIds),
	}
}

// batchReferenceList converts a slice of batch IDs to an embedded list of
// minimal batch references.
func batchReferenceList(ids []string) *apiresource.List[apiresource.BatchReference] {
	refs := make([]apiresource.BatchReference, len(ids))
	for i, id := range ids {
		refs[i] = apiresource.BatchReference{ID: id, Object: constants.ObjectTypeBatch}
	}
	return apiresource.NewList(refs, apiresource.PageInfo{})
}

// ScanningConsumptionPresenter converts a proto ScanningConsumptionInfo to an apiresource.ScanningConsumption.
func ScanningConsumptionPresenter(c *pb.ScanningConsumptionInfo) apiresource.ScanningConsumption {
	if c == nil {
		return apiresource.ScanningConsumption{}
	}

	return apiresource.ScanningConsumption{
		SKU:              c.Sku,
		Object:           constants.ObjectTypeScanningConsumption,
		DemandMeasure:    c.DemandMeasure,
		DemandUnit:       c.DemandUnit,
		InventoryMeasure: c.InventoryMeasure,
		InventoryUnit:    c.InventoryUnit,
		Instructions:     c.Instructions,
	}
}

// OpenBatchSummaryPresenter converts a proto OpenBatchSummaryInfo to an apiresource.OpenBatchSummary.
func OpenBatchSummaryPresenter(s *pb.OpenBatchSummaryInfo) apiresource.OpenBatchSummary {
	if s == nil {
		return apiresource.OpenBatchSummary{}
	}

	var item *apiresource.Item
	if s.Item != nil {
		item = &apiresource.Item{
			ID:     s.Item.Id,
			Object: constants.ObjectTypeItem,
			SKU:    s.Item.Sku,
		}
	}

	var scanningStation *apiresource.ScanningStation
	if s.ScanningStation != nil {
		scanningStation = &apiresource.ScanningStation{
			ID:     s.ScanningStation.Id,
			Object: constants.ObjectTypeScanningStation,
		}
	}

	return apiresource.OpenBatchSummary{
		Object:          constants.ObjectTypeOpenBatchSummary,
		DepartmentName:  s.DepartmentName,
		Item:            item,
		ScanningStation: scanningStation,
		Count:           s.Count,
		Unit:            s.Unit,
	}
}

// ScanningProductionStepInfoPresenter converts a proto ScanningProductionStepInfoProto to an apiresource.ScanningProductionStepInfo.
func ScanningProductionStepInfoPresenter(s *pb.ScanningProductionStepInfoProto) apiresource.ScanningProductionStepInfo {
	if s == nil {
		return apiresource.ScanningProductionStepInfo{}
	}

	return apiresource.ScanningProductionStepInfo{
		ID:          s.Id,
		Object:      constants.ObjectTypeScanningProductionStepInfo,
		Name:        s.Name,
		IsMultiPart: s.IsMultiPart,
	}
}

// BatchListPresenter converts a proto ListBatchesByScanningStationResponse to a paginated list.
func BatchListPresenter(ctx context.Context, resp *pb.ListBatchesByScanningStationResponse) *apiresource.List[apiresource.Batch] {
	if resp == nil {
		return apiresource.NewList[apiresource.Batch](nil, apiresource.PageInfo{})
	}

	batches := make([]apiresource.Batch, len(resp.Batches))
	for i, b := range resp.Batches {
		batches[i] = BatchPresenter(b)
	}

	return apiresource.NewList(batches, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo))
}
