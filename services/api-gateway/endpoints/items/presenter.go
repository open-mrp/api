package itemep

import (
	"context"

	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/id"
	pb "github.com/augno/api/shared/proto/core"
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

func stashItemInventoryQuantity(ctx context.Context, meta *resourcekit.LoadMeta, key string, q *pb.QuantityInfo) {
	if q == nil {
		return
	}
	quantity := buildQuantityFromProto(ctx, q)
	meta.Set(constants.ObjectTypeItemInventory, "singleton", key, quantity)
}

func buildQuantityFromProto(ctx context.Context, q *pb.QuantityInfo) *apiresource.Quantity {
	qid, _ := id.GenID(id.QuantityIDPrefix, nil)

	meta := resourcekit.GetLoadMeta(ctx)
	// unit is an expandable reference on the quantity: stash the FK id so
	// LoadUnits fetches the real Unit on ?include=...unit. Never fabricate.
	if q.UnitId != "" {
		meta.Set(constants.ObjectTypeQuantity, qid, "unit_id", q.UnitId)
	}

	return &apiresource.Quantity{
		ID:     qid,
		Object: constants.ObjectTypeQuantity,
		Value:  apiresource.NormalizeQuantityValue(q.Value, q.UnitType),
		DisplayValue: apiresource.FormatDisplayValue(
			q.Value,
			q.UnitAbbreviation,
			q.UnitType,
		),
		// Unit left nil: populated with real data via LoadUnits on ?include=...unit.
	}
}

func ItemCostsPresenter(resp *pb.GetItemCostsResponse) *apiresource.ItemCosts {
	if resp == nil {
		return nil
	}
	return &apiresource.ItemCosts{
		Object:             constants.ObjectTypeItem,
		DirectMaterialCost: resp.DirectMaterialCost,
		DirectLaborCost:    resp.DirectLaborCost,
		OverheadCost:       resp.OverheadCost,
		TotalCost:          resp.TotalCost,
		Unit: &apiresource.Unit{
			ID:     resp.UnitId,
			Object: constants.ObjectTypeUnit,
		},
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
		Object:    constants.ObjectTypeItem,
		TrendType: resp.TrendType,
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
			ItemID: r.ItemId, SKU: r.Sku,
			PreviousQuantity: r.PreviousQuantity, NewQuantity: r.NewQuantity,
		}
	}

	skipped := make([]apiresource.SkippedItemResult, len(resp.SkippedItems))
	for i, s := range resp.SkippedItems {
		skipped[i] = apiresource.SkippedItemResult{SKU: s.Sku, Reason: s.Reason}
	}

	errors := make([]apiresource.ReconcileErrorResult, len(resp.Errors))
	for i, e := range resp.Errors {
		errors[i] = apiresource.ReconcileErrorResult{SKU: e.Sku, Error: e.Error}
	}

	return &apiresource.BulkReconcileItemsResponse{
		Object:          constants.ObjectTypeBulkReconcileItemsResponse,
		ReconciledItems: apiresource.NewList(reconciled, apiresource.PageInfo{}),
		SkippedItems:    apiresource.NewList(skipped, apiresource.PageInfo{}),
		Errors:          apiresource.NewList(errors, apiresource.PageInfo{}),
	}
}
