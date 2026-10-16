package itemep

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	"github.com/open-mrp/apikit/include"

	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
)

func lightItemCategoryPresenter(c *pb.ItemCategoryInfo) *apiresource.ItemCategory {
	if c == nil {
		return nil
	}
	ic := &apiresource.ItemCategory{
		ID:        c.Id,
		Object:    constants.ObjectTypeItemCategory,
		Name:      c.Name,
		Type:      constants.ItemCategoryTypeFromCode(c.ItemCategoryTypeCode),
		Owner:     apiresource.NewOwner(c.AccountId),
		CreatedAt: grpcutil.TimestampToTime(c.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(c.UpdatedAt),
	}
	return ic
}

func lightAttributePresenter(a *pb.ItemAttributeInfo) *apiresource.Attribute {
	if a == nil {
		return nil
	}
	var colorCode constants.Color
	if a.ColorCode != nil {
		colorCode = constants.Color(*a.ColorCode)
	}
	attr := &apiresource.Attribute{
		ID:        a.Id,
		Object:    constants.ObjectTypeAttribute,
		Value:     a.Value,
		ColorCode: colorCode,
		SortOrder: a.SortOrder,
		Status:    constants.ArchiveStatus(a.Status),
		CreatedAt: grpcutil.TimestampToTime(a.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(a.UpdatedAt),
	}
	return attr
}

// ItemPresenter presents an item embedded whole in an older-version response, with its category and attributes; an
// attribute's property is left to its include. preview9 is as for resourceloaders.ItemFromProto.
func ItemPresenter(i *pb.ItemInfo, preview9 bool) apiresource.Item {
	if i == nil {
		return apiresource.Item{}
	}

	item := *resourceloaders.ItemFromProto(i, preview9)
	item.Category = lightItemCategoryPresenter(i.Category)

	attrs := make([]apiresource.Attribute, len(i.Attributes))
	for j, a := range i.Attributes {
		if p := lightAttributePresenter(a); p != nil {
			attrs[j] = *p
		}
	}
	item.Attributes = apiresource.NewList(attrs, apiresource.PageInfo{})

	return item
}

func ItemInventoryPresenter(ctx context.Context, resp *pb.GetItemInventoryResponse, units map[string]*apiresource.Unit) *apiresource.ItemInventory {
	if resp == nil {
		return nil
	}
	meta := include.GetLoadMeta(ctx)
	stashItemInventoryQuantity(meta, "on_hand", resp.OnHand, units)
	stashItemInventoryQuantity(meta, "reserved", resp.Reserved, units)
	stashItemInventoryQuantity(meta, "available_to_promise", resp.AvailableToPromise, units)
	stashItemInventoryQuantity(meta, "short", resp.Short, units)

	return &apiresource.ItemInventory{
		Object: constants.ObjectTypeItemInventory,
	}
}

// ItemInventoryUnitIDs names the units the four inventory figures are counted in, so the caller can resolve them before presenting.
func ItemInventoryUnitIDs(resp *pb.GetItemInventoryResponse) []string {
	if resp == nil {
		return nil
	}
	ids := make([]string, 0, 4)
	for _, q := range []*pb.QuantityInfo{resp.OnHand, resp.Reserved, resp.AvailableToPromise, resp.Short} {
		if q != nil {
			ids = append(ids, q.UnitId)
		}
	}
	return ids
}

// The four inventory figures are netted out of the ledger at read time; each arrives with its unit.
func stashItemInventoryQuantity(meta *include.LoadMeta, key string, q *pb.QuantityInfo, units map[string]*apiresource.Unit) {
	if q == nil {
		return
	}
	quantity := &apiresource.Quantity{
		Value: apiresource.NormalizeQuantityValue(q.Value, q.UnitType),
		Unit:  units[q.UnitId],
	}
	meta.Set(constants.ObjectTypeItemInventory, "singleton", key, quantity)
}

// ItemCostsPresenter renders a cost breakdown against units the caller has already resolved. Costs read as currency per item unit, so both units are part of the figure rather than a sub-resource to ask for separately.
func ItemCostsPresenter(resp *pb.GetItemCostsResponse, units map[string]*apiresource.Unit) *apiresource.ItemCosts {
	if resp == nil {
		return nil
	}
	return &apiresource.ItemCosts{
		Object:             constants.ObjectTypeItemCosts,
		DirectMaterialCost: resp.DirectMaterialCost,
		DirectLaborCost:    resp.DirectLaborCost,
		OverheadCost:       resp.OverheadCost,
		TotalCost:          resp.TotalCost,
		NumeratorUnit:      units[resp.NumeratorUnitId],
		DenominatorUnit:    units[resp.UnitId],
	}
}

func ItemTrendsPresenter(resp *pb.GetItemTrendsResponse, units map[string]*apiresource.Unit) *apiresource.ItemTrends {
	if resp == nil {
		return nil
	}

	points := make([]apiresource.ItemTrendPoint, len(resp.Points))
	for i, p := range resp.Points {
		points[i] = apiresource.ItemTrendPoint{
			Object:     constants.ObjectTypeItemTrendPoint,
			OccurredAt: grpcutil.TimestampToTime(p.Date),
			Value:      p.Value,
		}
	}

	return &apiresource.ItemTrends{
		Object:    constants.ObjectTypeItemTrends,
		TrendType: constants.ItemTrendType(resp.TrendType),
		Unit:      units[resp.UnitId],
		Points:    apiresource.NewList(points, apiresource.PageInfo{}),
	}
}

// BulkReconcileUnitIDs names the units the reconciled measures are counted in, so the caller can resolve them before presenting.
func BulkReconcileUnitIDs(resp *pb.BulkReconcileItemsResponse) []string {
	if resp == nil {
		return nil
	}
	ids := make([]string, 0, len(resp.ReconciledItems))
	for _, r := range resp.ReconciledItems {
		ids = append(ids, r.UnitId)
	}
	return ids
}

func BulkReconcileItemsPresenter(resp *pb.BulkReconcileItemsResponse, units map[string]*apiresource.Unit) *apiresource.BulkReconcileItemsResponse {
	if resp == nil {
		return &apiresource.BulkReconcileItemsResponse{
			Object:          constants.ObjectTypeBulkReconcileItemsResponse,
			ReconciledItems: apiresource.NewList([]apiresource.ReconciledItemResult{}, apiresource.PageInfo{}),
			SkippedItems:    apiresource.NewList([]apiresource.SkippedItemResult{}, apiresource.PageInfo{}),
			Errors:          apiresource.NewList([]apiresource.ReconcileErrorResult{}, apiresource.PageInfo{}),
		}
	}

	reconciled := make([]apiresource.ReconciledItemResult, len(resp.ReconciledItems))
	for i, r := range resp.ReconciledItems {
		reconciled[i] = apiresource.ReconciledItemResult{
			Object:           constants.ObjectTypeReconciledItemResult,
			Item:             itemEntity(r.ItemId, r.Sku),
			PreviousQuantity: reconciledQuantity(r.PreviousMeasure, units[r.UnitId]),
			NewQuantity:      reconciledQuantity(r.NewMeasure, units[r.UnitId]),
		}
	}

	skipped := make([]apiresource.SkippedItemResult, len(resp.SkippedItems))
	for i, s := range resp.SkippedItems {
		skipped[i] = apiresource.SkippedItemResult{Object: constants.ObjectTypeSkippedItemResult, SKU: s.Sku, Reason: s.Reason}
	}

	errors := make([]apiresource.ReconcileErrorResult, len(resp.Errors))
	for i, e := range resp.Errors {
		errors[i] = apiresource.ReconcileErrorResult{
			Object: constants.ObjectTypeReconcileErrorResult,
			Item:   itemEntity(e.ItemId, e.Sku),
			Error:  e.Error,
		}
	}

	return &apiresource.BulkReconcileItemsResponse{
		Object:          constants.ObjectTypeBulkReconcileItemsResponse,
		ReconciledItems: apiresource.NewList(reconciled, apiresource.PageInfo{}),
		SkippedItems:    apiresource.NewList(skipped, apiresource.PageInfo{}),
		Errors:          apiresource.NewList(errors, apiresource.PageInfo{}),
	}
}

// itemEntity names the item a reconcile row acted on. Every list in the response identifies its item the same way, and an id with a SKU is all a reconcile run knows about one.
func itemEntity(id, sku string) *apiresource.Entity {
	name := sku
	return apiresource.NewEntity(id, constants.ObjectTypeItem, &name, nil)
}

// reconciledQuantity presents a reconciled measure, computed against the ledger, with the unit it is counted in.
func reconciledQuantity(value string, unit *apiresource.Unit) *apiresource.Quantity {
	return &apiresource.Quantity{Value: value, Unit: unit}
}
