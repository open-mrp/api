package itemep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func lightRatePresenter(r *pb.RateInfo) *apiresource.Rate {
	if r == nil {
		return nil
	}
	return &apiresource.Rate{
		ID:     r.Id,
		Object: constants.ObjectTypeRate,
		Value:  r.Value,
		NumeratorUnit: &apiresource.Unit{
			ID:     r.NumeratorUnitId,
			Object: constants.ObjectTypeUnit,
		},
		DenominatorUnit: &apiresource.Unit{
			ID:     r.DenominatorUnitId,
			Object: constants.ObjectTypeUnit,
		},
		DisplayValue: "",
	}
}

func lightItemCategoryPresenter(c *pb.ItemCategoryInfo) *apiresource.ItemCategory {
	if c == nil {
		return nil
	}
	return &apiresource.ItemCategory{
		ID:     c.Id,
		Object: constants.ObjectTypeItemCategory,
		Name:   c.Name,
		Owner:  apiresource.NewOwner(c.AccountId),
	}
}

func lightAttributePresenter(a *pb.ItemAttributeInfo) *apiresource.Attribute {
	if a == nil {
		return nil
	}
	var colorCode constants.Color
	if a.ColorCode != nil {
		colorCode = constants.Color(*a.ColorCode)
	}
	return &apiresource.Attribute{
		ID:        a.Id,
		Object:    constants.ObjectTypeAttribute,
		Value:     a.Value,
		ColorCode: colorCode,
	}
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
		IsDirty:      i.IsDirty,
		CreatedAt:    grpcutil.TimestampToTime(i.CreatedAt),
		UpdatedAt:    grpcutil.TimestampToTime(i.UpdatedAt),
	}

	if i.Attributes != nil {
		attrs := make([]apiresource.Attribute, len(i.Attributes))
		for j, a := range i.Attributes {
			if p := lightAttributePresenter(a); p != nil {
				attrs[j] = *p
			}
		}
		item.Attributes = apiresource.NewList(attrs, apiresource.PageInfo{})
	}

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

func quantityInfoPresenter(q *pb.QuantityInfo) *apiresource.QuantityInfo {
	if q == nil {
		return nil
	}
	return &apiresource.QuantityInfo{
		Value: apiresource.NormalizeQuantityValue(q.Value, q.UnitType),
		Unit: &apiresource.Unit{
			ID:     q.UnitId,
			Object: constants.ObjectTypeUnit,
		},
	}
}

func ItemInventoryPresenter(resp *pb.GetItemInventoryResponse) *apiresource.ItemInventory {
	if resp == nil {
		return nil
	}
	return &apiresource.ItemInventory{
		Object:             constants.ObjectTypeItem,
		OnHand:             quantityInfoPresenter(resp.OnHand),
		Reserved:           quantityInfoPresenter(resp.Reserved),
		AvailableToPromise: quantityInfoPresenter(resp.AvailableToPromise),
		Short:              quantityInfoPresenter(resp.Short),
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
			Object: constants.ObjectTypeBulkReconcileItemsResponse,
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
		ReconciledItems: reconciled,
		SkippedItems:    skipped,
		Errors:          errors,
	}
}
