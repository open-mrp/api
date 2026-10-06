package resourceregistry

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeItem,
		Load:       resourceloaders.LoadItems,
		Subs: []resourcekit.SubField{
			{
				// Read with the item, so no loader and no item_categories:read; the ref lets nested includes resolve.
				Key:         "category",
				Target:      constants.ObjectTypeItemCategory,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractRefs: extractCategoryRefFromItem,
				Populate:    populateCategoryOnItem,
			},
			{Key: "unit_value", Populate: populateUnitValueOnItem},
			{Key: "unit_cost", Populate: populateUnitCostOnItem},
			{Key: "burn_rate", Populate: populateBurnRateOnItem},
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
	if v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeItem, item.ID, "category"); ok && v != nil {
		item.Category = v.(*apiresource.ItemCategory)
	}
}

func populateUnitValueOnItem(ctx context.Context, parent any, _ map[string]any) {
	item := parent.(*apiresource.Item)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeItem, item.ID, "unit_value")
	if !ok || v == nil {
		return
	}
	item.UnitValue = v.(*apiresource.Rate)
}

func populateUnitCostOnItem(ctx context.Context, parent any, _ map[string]any) {
	item := parent.(*apiresource.Item)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeItem, item.ID, "unit_cost")
	if !ok || v == nil {
		return
	}
	item.UnitCost = v.(*apiresource.Rate)
}

func populateBurnRateOnItem(ctx context.Context, parent any, _ map[string]any) {
	item := parent.(*apiresource.Item)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeItem, item.ID, "burn_rate")
	if !ok || v == nil {
		return
	}
	item.BurnRate = v.(*apiresource.Rate)
}

func populateAttributesOnItem(ctx context.Context, parent any, _ map[string]any) {
	item := parent.(*apiresource.Item)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeItem, item.ID, "attributes_list")
	if !ok || v == nil {
		return
	}
	item.Attributes = v.(*apiresource.List[apiresource.Attribute])
}
