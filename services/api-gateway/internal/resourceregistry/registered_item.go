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
				Key:         "category",
				Target:      constants.ObjectTypeItemCategory,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractCategoryIDFromItem,
				Populate:    populateCategoryOnItem,
			},
			{Key: "unit_value", Populate: populateUnitValueOnItem},
			{Key: "unit_cost", Populate: populateUnitCostOnItem},
			{Key: "burn_rate", Populate: populateBurnRateOnItem},
			{Key: "attributes", Populate: populateAttributesOnItem},
		},
	})
}

func extractCategoryIDFromItem(ctx context.Context, parent any) []string {
	item := parent.(*apiresource.Item)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeItem, item.ID, "item_category_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateCategoryOnItem(ctx context.Context, parent any, loaded map[string]any) {
	item := parent.(*apiresource.Item)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeItem, item.ID, "item_category_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
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
