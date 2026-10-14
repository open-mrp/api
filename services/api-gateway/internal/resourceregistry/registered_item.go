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
			valueShapeSub(constants.ObjectTypeItem, "unit_value", func(p any) string { return p.(*apiresource.Item).ID }, func(p, v any) {
				p.(*apiresource.Item).UnitValue = v.(*apiresource.UnitPrice)
			}),
			valueShapeSub(constants.ObjectTypeItem, "unit_cost", func(p any) string { return p.(*apiresource.Item).ID }, func(p, v any) {
				p.(*apiresource.Item).UnitCost = v.(*apiresource.UnitPrice)
			}),
			valueShapeSub(constants.ObjectTypeItem, "burn_rate", func(p any) string { return p.(*apiresource.Item).ID }, func(p, v any) {
				p.(*apiresource.Item).BurnRate = v.(*apiresource.Rate)
			}),
			{Key: "attributes", Populate: populateAttributesOnItem},
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
