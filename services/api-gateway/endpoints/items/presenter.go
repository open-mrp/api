package itemep

import (
	"strconv"

	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
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
	if len(c.Properties) > 0 {
		props := make([]apiresource.Property, 0, len(c.Properties))
		for _, p := range c.Properties {
			if p == nil {
				continue
			}
			props = append(props, apiresource.Property{
				ID:        p.Id,
				Object:    constants.ObjectTypeProperty,
				Name:      p.Name,
				CreatedAt: grpcutil.TimestampToTime(p.CreatedAt),
				UpdatedAt: grpcutil.TimestampToTime(p.UpdatedAt),
			})
		}
		ic.Properties = apiresource.NewList(props, apiresource.PageInfo{})
	}
	if c.UnitGroup != nil {
		ug := &apiresource.UnitGroup{
			ID:        c.UnitGroup.Id,
			Object:    constants.ObjectTypeUnitGroup,
			Name:      c.UnitGroup.Name,
			Type:      constants.UnitType(c.UnitGroup.Type),
			CreatedAt: grpcutil.TimestampToTime(c.UnitGroup.CreatedAt),
			UpdatedAt: grpcutil.TimestampToTime(c.UnitGroup.UpdatedAt),
		}
		if c.UnitGroup.BaseUnit != nil {
			u := c.UnitGroup.BaseUnit
			mapped := apiresource.Unit{
				ID:                u.Id,
				Object:            constants.ObjectTypeUnit,
				Name:              u.Name,
				Abbreviation:      u.Abbreviation,
				Type:              constants.UnitType(u.Type),
				RatioNumerator:    db.TrimDecimal(u.RatioNumerator),
				RatioDenominator:  db.TrimDecimal(u.RatioDenominator),
				OffsetNumerator:   db.TrimDecimal(u.OffsetNumerator),
				OffsetDenominator: db.TrimDecimal(u.OffsetDenominator),
				IsBaseUnit:        u.IsBaseUnit,
				CreatedAt:         grpcutil.TimestampToTime(u.CreatedAt),
				UpdatedAt:         grpcutil.TimestampToTime(u.UpdatedAt),
			}
			ug.BaseUnit = &mapped
		}
		if len(c.UnitGroup.AssociatedUnits) > 0 {
			units := make([]apiresource.UnitGroupUnit, 0, len(c.UnitGroup.AssociatedUnits))
			for _, au := range c.UnitGroup.AssociatedUnits {
				if au == nil {
					continue
				}
				discountPct, _ := strconv.ParseFloat(au.DiscountPercentage, 64)
				discountFixed, _ := strconv.ParseFloat(au.DiscountFixed, 64)
				visibility := constants.CustomerPortalVisibilityHidden
				if au.IsVisible {
					visibility = constants.CustomerPortalVisibilityVisible
				}
				ugu := apiresource.UnitGroupUnit{
					ID:                       au.Id,
					Object:                   constants.ObjectTypeUnitGroupUnit,
					DiscountPercentage:       discountPct,
					DiscountFixed:            discountFixed,
					CustomerPortalVisibility: visibility,
					CreatedAt:                grpcutil.TimestampToTime(au.CreatedAt),
					UpdatedAt:                grpcutil.TimestampToTime(au.UpdatedAt),
				}
				if au.Unit != nil {
					u := au.Unit
					mapped := apiresource.Unit{
						ID:                u.Id,
						Object:            constants.ObjectTypeUnit,
						Name:              u.Name,
						Abbreviation:      u.Abbreviation,
						Type:              constants.UnitType(u.Type),
						RatioNumerator:    db.TrimDecimal(u.RatioNumerator),
						RatioDenominator:  db.TrimDecimal(u.RatioDenominator),
						OffsetNumerator:   db.TrimDecimal(u.OffsetNumerator),
						OffsetDenominator: db.TrimDecimal(u.OffsetDenominator),
						IsBaseUnit:        u.IsBaseUnit,
						CreatedAt:         grpcutil.TimestampToTime(u.CreatedAt),
						UpdatedAt:         grpcutil.TimestampToTime(u.UpdatedAt),
					}
					ugu.Unit = &mapped
				}
				units = append(units, ugu)
			}
			ug.AssociatedUnits = apiresource.NewList(units, apiresource.PageInfo{})
		}
		ic.UnitGroup = ug
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

func ItemListPresenter(resp *pb.ListItemsResponse) *apiresource.List[apiresource.Item] {
	if resp == nil {
		return apiresource.NewList[apiresource.Item](nil, apiresource.PageInfo{})
	}

	items := make([]apiresource.Item, len(resp.Items))
	for i, item := range resp.Items {
		items[i] = ItemPresenter(item)
	}

	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(resp.PageInfo))
}

func ItemInventoryPresenter(resp *pb.GetItemInventoryResponse) *apiresource.ItemInventory {
	if resp == nil {
		return nil
	}
	return &apiresource.ItemInventory{
		Object:             constants.ObjectTypeItemInventory,
		OnHand:             inventoryQuantityFromProto(resp.OnHand),
		Reserved:           inventoryQuantityFromProto(resp.Reserved),
		AvailableToPromise: inventoryQuantityFromProto(resp.AvailableToPromise),
		Short:              inventoryQuantityFromProto(resp.Short),
	}
}

func inventoryQuantityFromProto(q *pb.QuantityInfo) *apiresource.Quantity {
	if q == nil {
		return nil
	}
	qid, _ := id.GenID(id.QuantityIDPrefix, nil)
	return &apiresource.Quantity{
		ID:     qid,
		Object: constants.ObjectTypeQuantity,
		Value:  apiresource.NormalizeQuantityValue(q.Value, q.UnitType),
		DisplayValue: apiresource.FormatDisplayValue(
			q.Value,
			q.UnitAbbreviation,
			q.UnitType,
		),
		Unit: &apiresource.Unit{
			ID:           q.UnitId,
			Object:       constants.ObjectTypeUnit,
			Name:         q.UnitAbbreviation,
			Abbreviation: q.UnitAbbreviation,
			Type:         constants.UnitType(q.UnitType),
		},
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

func ExportItemsPresenter(resp *pb.ExportItemsResponse) *apiresource.ExportItemsResponse {
	if resp == nil {
		return nil
	}

	items := make([]*apiresource.ExportItem, len(resp.Items))
	for i, item := range resp.Items {
		items[i] = &apiresource.ExportItem{
			ID:             item.Id,
			Object:         constants.ObjectTypeItem,
			SKU:            item.Sku,
			Description:    item.Description,
			Notes:          item.Notes,
			ItemTypeCode:   constants.ItemTypeCode(item.ItemTypeCode),
			CategoryName:   item.CategoryName,
			OnHandQuantity: item.OnHandQuantity,
			OnHandUnit: &apiresource.Unit{
				ID:     item.OnHandUnitId,
				Object: constants.ObjectTypeUnit,
			},
			CreatedAt: grpcutil.TimestampToTime(item.CreatedAt),
			UpdatedAt: grpcutil.TimestampToTime(item.UpdatedAt),
		}
	}

	return &apiresource.ExportItemsResponse{
		Object: constants.ObjectTypeList,
		Items:  items,
		Count:  resp.Count,
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
