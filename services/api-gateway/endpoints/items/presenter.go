package itemep

import (
	"context"

	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
)

func lightRatePresenter(r *pb.RateInfo) *apiresource.Rate {
	if r == nil {
		return nil
	}
	normalizedValue := apiresource.NormalizeRateValue(r.Value)
	return &apiresource.Rate{
		ID:     r.Id,
		Object: constants.ObjectTypeRate,
		Value:  normalizedValue,
		NumeratorUnit: &apiresource.Unit{
			ID:                r.NumeratorUnitId,
			Object:            constants.ObjectTypeUnit,
			Name:              r.NumeratorUnitName,
			Abbreviation:      r.NumeratorUnitAbbreviation,
			Type:              constants.UnitType(r.NumeratorUnitType),
			RatioNumerator:    r.NumeratorUnitRatioNumerator,
			RatioDenominator:  r.NumeratorUnitRatioDenominator,
			OffsetNumerator:   r.NumeratorUnitOffsetNumerator,
			OffsetDenominator: r.NumeratorUnitOffsetDenominator,
			CreatedAt:         grpcutil.TimestampToTime(r.NumeratorUnitCreatedAt),
			UpdatedAt:         grpcutil.TimestampToTime(r.NumeratorUnitUpdatedAt),
		},
		DenominatorUnit: &apiresource.Unit{
			ID:                r.DenominatorUnitId,
			Object:            constants.ObjectTypeUnit,
			Name:              r.DenominatorUnitName,
			Abbreviation:      r.DenominatorUnitAbbreviation,
			Type:              constants.UnitType(r.DenominatorUnitType),
			RatioNumerator:    r.DenominatorUnitRatioNumerator,
			RatioDenominator:  r.DenominatorUnitRatioDenominator,
			OffsetNumerator:   r.DenominatorUnitOffsetNumerator,
			OffsetDenominator: r.DenominatorUnitOffsetDenominator,
			CreatedAt:         grpcutil.TimestampToTime(r.DenominatorUnitCreatedAt),
			UpdatedAt:         grpcutil.TimestampToTime(r.DenominatorUnitUpdatedAt),
		},
		DisplayValue: apiresource.FormatRateDisplayValue(
			normalizedValue,
			r.NumeratorUnitAbbreviation,
			r.NumeratorUnitType,
			r.DenominatorUnitAbbreviation,
		),
		CreatedAt: grpcutil.TimestampToTime(r.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(r.UpdatedAt),
	}
}

func lightItemCategoryPresenter(c *pb.ItemCategoryInfo) *apiresource.ItemCategory {
	if c == nil {
		return nil
	}
	ic := &apiresource.ItemCategory{
		ID:        c.Id,
		Object:    constants.ObjectTypeItemCategory,
		Name:      c.Name,
		Type:      constants.ItemCategoryType(c.ItemCategoryTypeCode),
		Notes:     c.Notes,
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
		CreatedAt: grpcutil.TimestampToTime(a.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(a.UpdatedAt),
	}
	if a.PropertyId != "" {
		attr.Property = &apiresource.Property{
			ID:     a.PropertyId,
			Object: constants.ObjectTypeProperty,
		}
	}
	return attr
}

func ItemPresenter(i *pb.ItemInfo) apiresource.Item {
	if i == nil {
		return apiresource.Item{}
	}

	item := apiresource.Item{
		ID:           i.Id,
		Object:       constants.ObjectTypeItem,
		SKU:          i.Sku,
		Description:  i.Description,
		Notes:        i.Notes,
		ItemTypeCode: constants.ItemTypeCode(i.ItemTypeCode),
		Category:     lightItemCategoryPresenter(i.Category),
		UnitValue:    lightRatePresenter(i.UnitValue),
		UnitCost:     lightRatePresenter(i.UnitCost),
		BurnRate:     lightRatePresenter(i.BurnRate),
		CreatedAt:    grpcutil.TimestampToTime(i.CreatedAt),
		UpdatedAt:    grpcutil.TimestampToTime(i.UpdatedAt),
	}

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
	meta := resourcekit.GetLoadMeta(ctx)
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

// The four inventory figures are netted out of the ledger at read time, so each is a computed
// quantity: there is no row behind it to carry an id, but it still arrives with its unit.
func stashItemInventoryQuantity(meta *resourcekit.LoadMeta, key string, q *pb.QuantityInfo, units map[string]*apiresource.Unit) {
	if q == nil {
		return
	}
	quantity := &apiresource.ComputedQuantity{
		Object:       constants.ObjectTypeComputedQuantity,
		Value:        apiresource.NormalizeQuantityValue(q.Value, q.UnitType),
		DisplayValue: apiresource.FormatDisplayValue(q.Value, q.UnitAbbreviation, q.UnitType),
		Unit:         units[q.UnitId],
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

func ItemTrendsPresenter(resp *pb.GetItemTrendsResponse) *apiresource.ItemTrends {
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
			PreviousQuantity: reconciledQuantity(r.PreviousMeasure, r.UnitAbbreviation, units[r.UnitId]),
			NewQuantity:      reconciledQuantity(r.NewMeasure, r.UnitAbbreviation, units[r.UnitId]),
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

// reconciledQuantity presents a reconciled measure, which is computed against the ledger rather than stored, so it carries no id — but it does carry the unit it is counted in.
func reconciledQuantity(value, unitAbbreviation string, unit *apiresource.Unit) *apiresource.ComputedQuantity {
	return &apiresource.ComputedQuantity{
		Object:       constants.ObjectTypeComputedQuantity,
		Value:        value,
		DisplayValue: apiresource.FormatDisplayValue(value, unitAbbreviation, ""),
		Unit:         unit,
	}
}
