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
			// No item names a manufacturing method yet, so the include always reads null.
			{Key: "default_manufacturing_method", Populate: func(context.Context, any, map[string]any) {}},
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
