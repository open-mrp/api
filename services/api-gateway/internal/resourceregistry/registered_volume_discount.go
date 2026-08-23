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
		ObjectType: constants.ObjectTypeVolumeDiscount,
		Load:       resourceloaders.LoadVolumeDiscounts,
		Subs: []resourcekit.SubField{
			{Key: "customer_groups", Cardinality: resourcekit.CardinalityList, Populate: populateCustomerGroupsOnVolumeDiscount},
			{Key: "product_lines", Cardinality: resourcekit.CardinalityList, Populate: populateProductLinesOnVolumeDiscount},
			{Key: "categories", Cardinality: resourcekit.CardinalityList, Populate: populateCategoriesOnVolumeDiscount},
			// The discount's own category rows carry no properties, so this one hop
			// through the category loader is what makes the attribute pickers on the
			// discount page render at all.
			{
				Key:         "categories.properties",
				Target:      constants.ObjectTypeItemCategory,
				Cardinality: resourcekit.CardinalityList,
				ExtractIDs:  extractCategoryIDsFromVolumeDiscount,
				Populate:    populatePropertiesOnVolumeDiscountCategories,
			},
			{Key: "attributes", Cardinality: resourcekit.CardinalityList, Populate: populateAttributesOnVolumeDiscount},
			{Key: "acceptable_units", Cardinality: resourcekit.CardinalityList, Populate: populateAcceptableUnitsOnVolumeDiscount},
		},
	})
}

func populateCustomerGroupsOnVolumeDiscount(ctx context.Context, parent any, _ map[string]any) {
	vd := parent.(*apiresource.VolumeDiscount)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeVolumeDiscount, vd.ID, "customer_groups")
	if !ok {
		return
	}
	vd.CustomerGroups = v.(*apiresource.List[apiresource.AccountGroup])
}

func populateProductLinesOnVolumeDiscount(ctx context.Context, parent any, _ map[string]any) {
	vd := parent.(*apiresource.VolumeDiscount)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeVolumeDiscount, vd.ID, "product_lines")
	if !ok {
		return
	}
	vd.ProductLines = v.(*apiresource.List[apiresource.ProductLine])
}

func populateCategoriesOnVolumeDiscount(ctx context.Context, parent any, _ map[string]any) {
	vd := parent.(*apiresource.VolumeDiscount)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeVolumeDiscount, vd.ID, "categories")
	if !ok {
		return
	}
	vd.Categories = v.(*apiresource.List[apiresource.ItemCategory])
}

func extractCategoryIDsFromVolumeDiscount(_ context.Context, parent any) []string {
	vd := parent.(*apiresource.VolumeDiscount)
	if vd.Categories == nil {
		return nil
	}
	ids := make([]string, len(vd.Categories.Data))
	for i, cat := range vd.Categories.Data {
		ids[i] = cat.ID
	}
	return ids
}

func populatePropertiesOnVolumeDiscountCategories(ctx context.Context, parent any, loaded map[string]any) {
	vd := parent.(*apiresource.VolumeDiscount)
	if vd.Categories == nil {
		return
	}
	meta := resourcekit.GetLoadMeta(ctx)
	for i := range vd.Categories.Data {
		cat := &vd.Categories.Data[i]
		if _, ok := loaded[cat.ID]; !ok {
			continue
		}
		v, ok := meta.Get(constants.ObjectTypeItemCategory, cat.ID, "properties_list")
		if !ok || v == nil {
			continue
		}
		cat.Properties = v.(*apiresource.List[apiresource.Property])
	}
}

func populateAttributesOnVolumeDiscount(ctx context.Context, parent any, _ map[string]any) {
	vd := parent.(*apiresource.VolumeDiscount)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeVolumeDiscount, vd.ID, "attributes")
	if !ok {
		return
	}
	vd.Attributes = v.(*apiresource.List[apiresource.Attribute])
}

func populateAcceptableUnitsOnVolumeDiscount(ctx context.Context, parent any, _ map[string]any) {
	vd := parent.(*apiresource.VolumeDiscount)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeVolumeDiscount, vd.ID, "acceptable_units")
	if !ok {
		return
	}
	vd.AcceptableUnits = v.(*apiresource.List[apiresource.Unit])
}
