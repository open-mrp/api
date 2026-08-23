package resourceregistry

import (
	"context"
	"testing"

	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
)

// A volume discount's own category rows carry no properties, so `categories.properties`
// takes a second hop through the item-category loader. These tests lock the extract /
// populate wiring that grafts the loaded properties back onto the discount's categories —
// without them the discount page has nothing to hang its attribute pickers on.

func discountWithCategories(ids ...string) *apiresource.VolumeDiscount {
	cats := make([]apiresource.ItemCategory, len(ids))
	for i, id := range ids {
		cats[i] = apiresource.ItemCategory{ID: id, Object: constants.ObjectTypeItemCategory}
	}
	return &apiresource.VolumeDiscount{
		ID:         "quds_1",
		Categories: apiresource.NewList(cats, apiresource.PageInfo{}),
	}
}

func TestExtractCategoryIDsFromVolumeDiscount(t *testing.T) {
	ctx := resourcekit.WithLoadMeta(context.Background())

	ids := extractCategoryIDsFromVolumeDiscount(ctx, discountWithCategories("itcg_1", "itcg_2"))
	if len(ids) != 2 || ids[0] != "itcg_1" || ids[1] != "itcg_2" {
		t.Fatalf("extract = %v, want [itcg_1 itcg_2]", ids)
	}

	if ids := extractCategoryIDsFromVolumeDiscount(ctx, &apiresource.VolumeDiscount{ID: "quds_1"}); ids != nil {
		t.Errorf("extract with no categories = %v, want nil", ids)
	}
}

func TestPopulatePropertiesOnVolumeDiscountCategories(t *testing.T) {
	ctx := resourcekit.WithLoadMeta(context.Background())
	props := apiresource.NewList([]apiresource.Property{
		{ID: "prop_1", Object: constants.ObjectTypeProperty, Name: "Color"},
	}, apiresource.PageInfo{})
	resourcekit.GetLoadMeta(ctx).
		Set(constants.ObjectTypeItemCategory, "itcg_1", "properties_list", props)

	vd := discountWithCategories("itcg_1", "itcg_2")
	populatePropertiesOnVolumeDiscountCategories(ctx, vd, map[string]any{
		"itcg_1": &apiresource.ItemCategory{ID: "itcg_1"},
		"itcg_2": &apiresource.ItemCategory{ID: "itcg_2"},
	})

	got := vd.Categories.Data[0].Properties
	if got == nil || len(got.Data) != 1 || got.Data[0].ID != "prop_1" {
		t.Fatalf("categories[0].properties = %v, want the loaded property list", got)
	}
	// itcg_2 loaded but has no properties of its own — the field stays unexpanded
	// rather than becoming an empty list, matching every other expandable field.
	if vd.Categories.Data[1].Properties != nil {
		t.Errorf("categories[1].properties = %v, want nil", vd.Categories.Data[1].Properties)
	}
}

func TestPopulatePropertiesOnVolumeDiscountCategories_SkipsUnloaded(t *testing.T) {
	ctx := resourcekit.WithLoadMeta(context.Background())
	resourcekit.GetLoadMeta(ctx).Set(constants.ObjectTypeItemCategory, "itcg_1", "properties_list",
		apiresource.NewList([]apiresource.Property{{ID: "prop_1"}}, apiresource.PageInfo{}))

	vd := discountWithCategories("itcg_1")
	populatePropertiesOnVolumeDiscountCategories(ctx, vd, map[string]any{})

	if vd.Categories.Data[0].Properties != nil {
		t.Errorf("properties = %v, want nil when the category was not loaded", vd.Categories.Data[0].Properties)
	}
}
