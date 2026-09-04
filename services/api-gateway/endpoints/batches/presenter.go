package batchep

import (
	"context"

	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/ptrutil"
)

// BatchQuantityPresenter converts a proto BatchQuantityInfo to an apiresource.Quantity.
func BatchQuantityPresenter(meta *resourcekit.LoadMeta, q *pb.BatchQuantityInfo) *apiresource.Quantity {
	if q == nil {
		return nil
	}

	// Unit left nil: the batch query carries only the unit's id, abbreviation and dimension, so the
	// id is stashed and the real unit is resolved on `?include=quantity.unit`; never fabricated.
	if meta != nil {
		meta.Set(constants.ObjectTypeQuantity, q.Id, "unit_id", q.UnitId)
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
	}
}

// BatchPresenter converts a proto BatchInfo to an apiresource.Batch.
func BatchPresenter(meta *resourcekit.LoadMeta, b *pb.BatchInfo) apiresource.Batch {
	if b == nil {
		return apiresource.Batch{}
	}

	item := batchRef(b.ItemId, constants.ObjectTypeItem, b.ItemSku)

	scanningStation := batchRef(ptrutil.Deref(b.ScanningStationId), constants.ObjectTypeScanningStation, ptrutil.Deref(b.ScanningStationName))

	productionStep := batchRef(ptrutil.Deref(b.ProductionStepId), constants.ObjectTypeProductionStep, ptrutil.Deref(b.ProductionStepName))

	var productionRun *apiresource.ProductionRunReference
	if b.ProductionRunId != nil && *b.ProductionRunId != "" {
		productionRun = &apiresource.ProductionRunReference{
			ID:     *b.ProductionRunId,
			Object: constants.ObjectTypeProductionRun,
			Number: ptrutil.Deref(b.ProductionRunNumber),
		}
	}

	machines := make([]apiresource.Entity, 0, len(b.Machines))
	for _, m := range b.Machines {
		if ref := batchRef(m.Id, constants.ObjectTypeMachine, m.Name); ref != nil {
			machines = append(machines, *ref)
		}
	}

	department := batchRef(ptrutil.Deref(b.DepartmentId), constants.ObjectTypeDepartment, ptrutil.Deref(b.DepartmentName))

	lots := make([]apiresource.BatchLot, len(b.Lots))
	for i, l := range b.Lots {
		lots[i] = apiresource.BatchLot{
			Object:    constants.ObjectTypeBatchLot,
			LotNumber: l.LotNumber,
			Type:      constants.BatchLotType(l.Type),
		}
	}

	return apiresource.Batch{
		ID:              b.Id,
		Object:          constants.ObjectTypeBatch,
		Item:            item,
		Quantity:        BatchQuantityPresenter(meta, b.Quantity),
		Seconds:         BatchQuantityPresenter(meta, b.Seconds),
		Waste:           BatchQuantityPresenter(meta, b.Waste),
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
func BaseBatchPresenter(meta *resourcekit.LoadMeta, b *pb.BaseBatchInfo) apiresource.Batch {
	if b == nil {
		return apiresource.Batch{}
	}

	item := batchRef(b.ItemId, constants.ObjectTypeItem, b.ItemSku)

	scanningStation := batchRef(ptrutil.Deref(b.ScanningStationId), constants.ObjectTypeScanningStation, ptrutil.Deref(b.ScanningStationName))

	productionStep := batchRef(ptrutil.Deref(b.ProductionStepId), constants.ObjectTypeProductionStep, ptrutil.Deref(b.ProductionStepName))

	var productionRun *apiresource.ProductionRunReference
	if b.ProductionRunId != nil && *b.ProductionRunId != "" {
		productionRun = &apiresource.ProductionRunReference{
			ID:     *b.ProductionRunId,
			Object: constants.ObjectTypeProductionRun,
			Number: ptrutil.Deref(b.ProductionRunNumber),
		}
	}

	department := batchRef(ptrutil.Deref(b.DepartmentId), constants.ObjectTypeDepartment, ptrutil.Deref(b.DepartmentName))

	return apiresource.Batch{
		ID:              b.Id,
		Object:          constants.ObjectTypeBatch,
		Item:            item,
		Quantity:        BatchQuantityPresenter(meta, b.Quantity),
		Seconds:         BatchQuantityPresenter(meta, b.Seconds),
		Waste:           BatchQuantityPresenter(meta, b.Waste),
		ScanningStation: scanningStation,
		Department:      department,
		ProductionStep:  productionStep,
		ProductionRun:   productionRun,
		Machines:        apiresource.NewList([]apiresource.Entity{}, apiresource.PageInfo{}),
		ClosedAt:        grpcutil.TimestampToTimePtr(b.ClosedAt),
		ScannedAt:       grpcutil.TimestampToTimePtr(b.ScannedAt),
		CreatedAt:       grpcutil.TimestampToTime(b.CreatedAt),
		UpdatedAt:       grpcutil.TimestampToTime(b.UpdatedAt),
	}
}

// BatchFlowNodePresenter converts a proto BatchFlowNodeInfo to an apiresource.BatchFlowNode.
func BatchFlowNodePresenter(meta *resourcekit.LoadMeta, n *pb.BatchFlowNodeInfo) apiresource.BatchFlowNode {
	if n == nil {
		return apiresource.BatchFlowNode{}
	}

	return apiresource.BatchFlowNode{
		Object:        constants.ObjectTypeBatchFlowNode,
		Batch:         BatchPresenter(meta, n.Batch),
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

	// References, not records: the summary groups open batches and knows only which item and station
	// they sit at, so it names them rather than shipping catalog objects with every other field blank.
	item := openBatchSummaryItem(s.Item)
	scanningStation := openBatchSummaryStation(s.ScanningStation)

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
		ID:     s.Id,
		Object: constants.ObjectTypeScanningProductionStepInfo,
		Name:   s.Name,
		Type:   scanningStepType(s.IsMultiPart),
	}
}

// BatchListPresenter converts a proto ListBatchesByScanningStationResponse to a paginated list.
func BatchListPresenter(ctx context.Context, resp *pb.ListBatchesByScanningStationResponse) *apiresource.List[apiresource.Batch] {
	meta := resourcekit.GetLoadMeta(ctx)
	if resp == nil {
		return apiresource.NewList[apiresource.Batch](nil, apiresource.PageInfo{})
	}

	batches := make([]apiresource.Batch, len(resp.Batches))
	for i, b := range resp.Batches {
		batches[i] = BatchPresenter(meta, b)
	}

	return apiresource.NewList(batches, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo))
}

// scanningStepType names what the backend reports as a multi-part flag, so the client reads a value rather than inferring one from a boolean.
func scanningStepType(multiPart bool) constants.ScanningStepType {
	if multiPart {
		return constants.ScanningStepTypeMultiPart
	}
	return constants.ScanningStepTypeSingle
}

func openBatchSummaryItem(i *pb.OpenBatchSummaryItemProto) *apiresource.Entity {
	if i == nil {
		return nil
	}
	sku := i.Sku
	return apiresource.NewEntity(i.Id, constants.ObjectTypeItem, &sku, nil)
}

func openBatchSummaryStation(st *pb.OpenBatchSummaryScanningStationProto) *apiresource.Entity {
	if st == nil {
		return nil
	}
	return apiresource.NewEntity(st.Id, constants.ObjectTypeScanningStation, nil, nil)
}

// batchRef names one of the records a batch points at. The batch query carries an id and a label
// for each and nothing more, so they are named rather than shipped as records with every other
// required field blank.
func batchRef(id string, entityType constants.ObjectType, label string) *apiresource.Entity {
	if id == "" {
		return nil
	}
	var name *string
	if label != "" {
		name = &label
	}
	return apiresource.NewEntity(id, entityType, name, nil)
}
