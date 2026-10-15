package versiontransforms

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-mrp/api/shared/apiversion"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/version"
)

func upgradeBody(ot constants.ObjectType, method, route string, body map[string]any) map[string]any {
	return version.TransformEndpointRequest(apiversion.V1_0_Forge_Preview9, apiversion.V1_0_Forge, ot, method, route, body)
}

func TestPreview9_ItemCategoryTypeAndNotes(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeItem, `{"id":"itm_1","object":"item","category":{"id":"ic_1","object":"item_category","type":"product"}}`)
	category := got["category"].(map[string]any)
	assert.Equal(t, "product_category", category["type"])
	assert.Contains(t, category, "notes")
	assert.Nil(t, category["notes"])

	list := downgrade(t, constants.ObjectTypeItemCategory, `{"object":"list","page_info":{"next_page_url":null,"previous_page_url":null},"data":[{"id":"ic_2","object":"item_category","type":"material"}]}`)
	assert.Equal(t, "material_category", list["data"].([]any)[0].(map[string]any)["type"])
}

func TestPreview9_ItemCategoryWrites(t *testing.T) {
	t.Parallel()

	created := upgradeBody(constants.ObjectTypeItemCategory, http.MethodPost, itemCategoriesRoute, map[string]any{"name": "Socks", "type": "product_category", "unit_group_id": "ug_1"})
	assert.Equal(t, map[string]any{"name": "Socks", "type": "product", "unit_group_id": "ug_1"}, created)

	updated := upgradeBody(constants.ObjectTypeItemCategory, http.MethodPatch, itemCategoryRoute, map[string]any{"name": "Socks", "notes": "kept elsewhere"})
	assert.Equal(t, map[string]any{"name": "Socks"}, updated)

	bulk := upgradeBody(constants.ObjectTypeAsyncJob, http.MethodPost, itemCategoriesBulkUpsertRoute, map[string]any{"item_categories": []any{
		map[string]any{"name": "Socks", "type": "material_category", "notes": nil},
	}})
	assert.Equal(t, map[string]any{"name": "Socks", "type": "material"}, bulk["item_categories"].([]any)[0])
}

func TestPreview9_ItemCategoryTypeFilter(t *testing.T) {
	t.Parallel()

	q := version.TransformQuery(apiversion.V1_0_Forge_Preview9, apiversion.V1_0_Forge, constants.ObjectTypeItemCategory, itemCategoriesRoute, url.Values{"type": {"material_category"}})
	assert.Equal(t, url.Values{"types[]": {"material"}}, q)

	other := version.TransformQuery(apiversion.V1_0_Forge_Preview9, apiversion.V1_0_Forge, constants.ObjectTypeItem, "/v1/catalog/items", url.Values{"type": {"material"}})
	assert.Equal(t, "material", other.Get("type"), "other routes keep their own type filter")
}

func TestPreview9_AttributeColorOnCreate(t *testing.T) {
	t.Parallel()

	created := upgradeBody(constants.ObjectTypeAttribute, http.MethodPost, propertyAttributesRoute, map[string]any{"value": "Large"})
	assert.Contains(t, preview9AttributeColors, created["color"])

	chosen := upgradeBody(constants.ObjectTypeAttribute, http.MethodPost, propertyAttributesRoute, map[string]any{"value": "Large", "color": "default"})
	assert.Equal(t, "default", chosen["color"])

	updated := upgradeBody(constants.ObjectTypeAttribute, http.MethodPatch, "/v1/catalog/properties/{property_id}/attributes/{id}", map[string]any{"value": "Large"})
	assert.NotContains(t, updated, "color", "an update without a color leaves it as it is")
}

func TestPreview9_ItemAttributesCarryTheirProperty(t *testing.T) {
	t.Parallel()

	const route = "/v1/test/cat1/items/{id}"
	RegisterRouteIncludes(route, []string{"attributes", "attributes.property", "category"})

	q := version.TransformQuery(apiversion.V1_0_Forge_Preview9, apiversion.V1_0_Forge, constants.ObjectTypeItem, route, url.Values{"include[]": {"attributes", "category"}})
	assert.ElementsMatch(t, []string{"attributes", "category", "attributes.property"}, q["include[]"])

	q = version.TransformQuery(apiversion.V1_0_Forge_Preview9, apiversion.V1_0_Forge, constants.ObjectTypeItem, "/v1/test/cat1/other", url.Values{"include": {"attributes"}})
	assert.Equal(t, []string{"attributes"}, q["include"], "a route that doesn't offer the property is left alone")
}
