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

func ItemInventoryPresenter(ctx context.Context, resp *pb.GetItemInventoryResponse) *apiresource.ItemInventory {
	if resp == nil {
		return nil
	}
	meta := resourcekit.GetLoadMeta(ctx)
	stashItemInventoryQuantity(ctx, meta, "on_hand", resp.OnHand)
	stashItemInventoryQuantity(ctx, meta, "reserved", resp.Reserved)
	stashItemInventoryQuantity(ctx, meta, "available_to_promise", resp.AvailableToPromise)
	stashItemInventoryQuantity(ctx, meta, "short", resp.Short)

	return &apiresource.ItemInventory{
		Object: constants.ObjectTypeItemInventory,
	}
}

// The four inventory figures are netted out of the ledger at read time, so each is a computed
// quantity: there is no row behind it to carry an id, and the unit rides in `display_value`.
func stashItemInventoryQuantity(ctx context.Context, meta *resourcekit.LoadMeta, key string, q *pb.QuantityInfo) {
	if q == nil {
		return
	}
	quantity := &apiresource.ComputedQuantity{
		Object:       constants.ObjectTypeComputedQuantity,
		Value:        apiresource.NormalizeQuantityValue(q.Value, q.UnitType),
		DisplayValue: apiresource.FormatDisplayValue(q.Value, q.UnitAbbreviation, q.UnitType),
	}
	meta.Set(constants.ObjectTypeItemInventory, "singleton", key, quantity)
}

func ItemCostsPresenter(resp *pb.GetItemCostsResponse) *apiresource.ItemCosts {
	if resp == nil {
		return nil
	}
	return &apiresource.ItemCosts{
		Object:             constants.ObjectTypeItemCosts,
		DirectMaterialCost: resp.DirectMaterialCost,
		DirectLaborCost:    resp.DirectLaborCost,
		OverheadCost:       resp.OverheadCost,
		TotalCost:          resp.TotalCost,
		NumeratorUnit:      unitRef(resp.NumeratorUnitId),
		DenominatorUnit:    unitRef(resp.UnitId),
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

func BulkReconcileItemsPresenter(resp *pb.BulkReconcileItemsResponse) *apiresource.BulkReconcileItemsResponse {
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
			Object: constants.ObjectTypeReconciledItemResult,
			// The row identifies the item it reconciled; only the id and SKU are known here, so the rest of the item stays unset rather than being invented.
			Item:             &apiresource.Item{ID: r.ItemId, Object: constants.ObjectTypeItem, SKU: r.Sku},
			PreviousQuantity: reconciledQuantity(r.PreviousMeasure, r.UnitId, r.UnitAbbreviation),
			NewQuantity:      reconciledQuantity(r.NewMeasure, r.UnitId, r.UnitAbbreviation),
		}
	}

	skipped := make([]apiresource.SkippedItemResult, len(resp.SkippedItems))
	for i, s := range resp.SkippedItems {
		skipped[i] = apiresource.SkippedItemResult{Object: constants.ObjectTypeSkippedItemResult, SKU: s.Sku, Reason: s.Reason}
	}

	errors := make([]apiresource.ReconcileErrorResult, len(resp.Errors))
	for i, e := range resp.Errors {
		sku := e.Sku
		errors[i] = apiresource.ReconcileErrorResult{
			Object: constants.ObjectTypeReconcileErrorResult,
			Item: &apiresource.Entity{
				ID:     e.ItemId,
				Object: constants.ObjectTypeEntity,
				Type:   constants.ObjectTypeItem,
				Name:   &sku,
			},
			Error: e.Error,
		}
	}

	return &apiresource.BulkReconcileItemsResponse{
		Object:          constants.ObjectTypeBulkReconcileItemsResponse,
		ReconciledItems: apiresource.NewList(reconciled, apiresource.PageInfo{}),
		SkippedItems:    apiresource.NewList(skipped, apiresource.PageInfo{}),
		Errors:          apiresource.NewList(errors, apiresource.PageInfo{}),
	}
}

// unitRef names a unit by id. The unit itself is resolved by the include machinery where a caller asks for it; here it identifies which unit the figures are counted in.
func unitRef(id string) *apiresource.Unit {
	if id == "" {
		return nil
	}
	return &apiresource.Unit{ID: id, Object: constants.ObjectTypeUnit}
}

// reconciledQuantity presents a reconciled measure, which is computed against the ledger rather than stored, so it carries no id.
func reconciledQuantity(value, unitID, unitAbbreviation string) *apiresource.ComputedQuantity {
	q := &apiresource.ComputedQuantity{
		Object:       constants.ObjectTypeComputedQuantity,
		Value:        value,
		DisplayValue: apiresource.FormatDisplayValue(value, unitAbbreviation, ""),
	}
	if unitID != "" {
		q.Unit = &apiresource.Unit{ID: unitID, Object: constants.ObjectTypeUnit}
	}
	return q
}
