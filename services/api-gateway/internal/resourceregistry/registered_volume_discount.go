package resourceregistry

import (
	"context"

	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
)

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeVolumeDiscount,
		Load:       resourceloaders.LoadVolumeDiscounts,
		Subs: []resourcekit.SubField{
			{Key: "customer_groups", Cardinality: resourcekit.CardinalityList, Populate: populateCustomerGroupsOnVolumeDiscount},
			{Key: "product_lines", Cardinality: resourcekit.CardinalityList, Populate: populateProductLinesOnVolumeDiscount},
			{Key: "categories", Cardinality: resourcekit.CardinalityList, Populate: populateCategoriesOnVolumeDiscount},
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
