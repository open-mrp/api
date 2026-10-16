package resourceregistry

import (
	"context"

	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeItem,
		Load:       resourceloaders.LoadItems,
		Subs: []include.SubField{
			{
				// Read with the item, so no loader and no item_categories:read; the ref lets nested includes resolve.
				Key:         "category",
				Target:      constants.ObjectTypeItemCategory,
				Cardinality: include.CardinalityOnePtr,
				ExtractRefs: extractCategoryRefFromItem,
				Populate:    populateCategoryOnItem,
			},
			{
				// Read with the item; the refs let `attributes.property` resolve.
				Key:         "attributes",
				Target:      constants.ObjectTypeAttribute,
				ExtractRefs: extractAttributeRefsFromItem,
				Populate:    populateAttributesOnItem,
			},
			{
				Key:         "sales.product_line",
				Target:      constants.ObjectTypeProductLine,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractSalesProductLineIDFromItem,
				Populate:    populateSalesProductLineOnItem,
			},
			{
				Key:         "stocking_unit",
				Target:      constants.ObjectTypeUnit,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  itemUnitIDExtractor(resourceloaders.ItemStockingUnitMetaKey),
				Populate: itemUnitPopulator(resourceloaders.ItemStockingUnitMetaKey, func(item *apiresource.Item, u *apiresource.Unit) {
					item.StockingUnit = u
				}),
			},
			{
				Key:         "default_order_unit",
				Target:      constants.ObjectTypeUnit,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  itemUnitIDExtractor(resourceloaders.ItemDefaultOrderUnitMetaKey),
				Populate: itemUnitPopulator(resourceloaders.ItemDefaultOrderUnitMetaKey, func(item *apiresource.Item, u *apiresource.Unit) {
					item.DefaultOrderUnit = u
				}),
			},
			{
				// Read with the item; each unit loads with the response's other value-shape units.
				Key:      "order_units",
				Populate: populateOrderUnitsOnItem,
			},
			{
				// The method is looked up by item: the loader resolves the item's default method in the same batch.
				Key:         "default_manufacturing_method",
				Target:      constants.ObjectTypeManufacturingMethod,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs: func(_ context.Context, parent any) []string {
					return []string{resourceloaders.DefaultMethodOfItemKey(parent.(*apiresource.Item).ID)}
				},
				Populate: func(_ context.Context, parent any, loaded map[string]any) {
					item := parent.(*apiresource.Item)
					if v, ok := loaded[resourceloaders.DefaultMethodOfItemKey(item.ID)]; ok {
						item.DefaultManufacturingMethod = v.(*apiresource.ManufacturingMethod)
					}
				},
			},
		},
	})
}

func extractCategoryRefFromItem(_ context.Context, parent any) []any {
	item := parent.(*apiresource.Item)
	if item.Category == nil {
		return nil
	}
	return []any{item.Category}
}

func populateCategoryOnItem(ctx context.Context, parent any, _ map[string]any) {
	item := parent.(*apiresource.Item)
	if v, ok := include.GetLoadMeta(ctx).Get(constants.ObjectTypeItem, item.ID, "category"); ok && v != nil {
		item.Category = v.(*apiresource.ItemCategory)
	}
}

func populateAttributesOnItem(ctx context.Context, parent any, _ map[string]any) {
	item := parent.(*apiresource.Item)
	v, ok := include.GetLoadMeta(ctx).
		Get(constants.ObjectTypeItem, item.ID, "attributes_list")
	if !ok || v == nil {
		return
	}
	item.Attributes = v.(*apiresource.List[apiresource.Attribute])
}

func extractAttributeRefsFromItem(_ context.Context, parent any) []any {
	item := parent.(*apiresource.Item)
	if item.Attributes == nil {
		return nil
	}
	refs := make([]any, len(item.Attributes.Data))
	for i := range item.Attributes.Data {
		refs[i] = &item.Attributes.Data[i]
	}
	return refs
}

func salesProductLineID(ctx context.Context, item *apiresource.Item) string {
	if item.Sales == nil {
		return ""
	}
	id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeItem, item.ID, resourceloaders.ItemSalesProductLineMetaKey)
	return id
}

func extractSalesProductLineIDFromItem(ctx context.Context, parent any) []string {
	if id := salesProductLineID(ctx, parent.(*apiresource.Item)); id != "" {
		return []string{id}
	}
	return nil
}

func populateSalesProductLineOnItem(ctx context.Context, parent any, loaded map[string]any) {
	item := parent.(*apiresource.Item)
	id := salesProductLineID(ctx, item)
	if v, ok := loaded[id]; ok && id != "" {
		item.Sales.ProductLine = v.(*apiresource.ProductLine)
	}
}

func itemUnitIDExtractor(metaKey string) func(context.Context, any) []string {
	return func(ctx context.Context, parent any) []string {
		item := parent.(*apiresource.Item)
		if id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeItem, item.ID, metaKey); id != "" {
			return []string{id}
		}
		return nil
	}
}

func itemUnitPopulator(metaKey string, set func(*apiresource.Item, *apiresource.Unit)) func(context.Context, any, map[string]any) {
	return func(ctx context.Context, parent any, loaded map[string]any) {
		item := parent.(*apiresource.Item)
		id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeItem, item.ID, metaKey)
		if v, ok := loaded[id]; ok && id != "" {
			set(item, v.(*apiresource.Unit))
		}
	}
}

func populateOrderUnitsOnItem(ctx context.Context, parent any, _ map[string]any) {
	item := parent.(*apiresource.Item)
	if v, ok := include.GetLoadMeta(ctx).Get(constants.ObjectTypeItem, item.ID, resourceloaders.ItemOrderUnitsMetaKey); ok && v != nil {
		item.OrderUnits = v.(*apiresource.List[apiresource.ItemOrderUnit])
	}
}
