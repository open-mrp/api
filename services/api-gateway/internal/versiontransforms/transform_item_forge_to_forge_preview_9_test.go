package versiontransforms

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/shared/apiversion"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/version"
)

const forgeItemJSON = `{"id":"itm_1","object":"item","type":"service","status":"archived","sku":"SVC-1","description":null,
	"category":null,"attributes":null,
	"unit_cost":{"amount":"2.00","currency":"usd","per_unit":{"id":"each","object":"unit","abbreviation":"ea","dimension":"quantity"}},
	"burn_rate":null,"supply":"make",
	"sales":{"unit_price":{"amount":"5.00","currency":"usd","per_unit":{"id":"each","object":"unit","abbreviation":"ea","dimension":"quantity"}},
		"product_line":null,"portal_visibility":"hidden","lead_time":null},
	"material":null,"default_manufacturing_method":null,"stocking_unit":null,"default_order_unit":null,
	"order_units":{"object":"list","data":[{"unit":{"id":"dozen","object":"unit"},"portal_visibility":"visible","discount":null,"quantity":null}]},
	"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}`

func TestPreview9_ItemShape(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeItem, forgeItemJSON)
	assert.Equal(t, "product", got["type"], "a service was a product")
	for _, k := range []string{"status", "supply", "sales", "material", "default_manufacturing_method", "stocking_unit", "default_order_unit", "order_units"} {
		assert.NotContains(t, got, k)
	}
	assert.Contains(t, got, "notes")
	assert.Nil(t, got["notes"])
	unitValue, ok := got["unit_value"].(map[string]any)
	require.True(t, ok, "the sales price is the unit value")
	assert.Equal(t, "rate", unitValue["object"])
	assert.Equal(t, "5.00", unitValue["value"])
	assert.Equal(t, "rate", got["unit_cost"].(map[string]any)["object"])
}

func TestPreview9_ItemNotSoldHasNoUnitValue(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeItem, `{"id":"itm_2","object":"item","type":"material","sku":"M-1","sales":null,"material":{"order_point":null,"lead_time":null}}`)
	assert.Equal(t, "material", got["type"])
	assert.Contains(t, got, "unit_value")
	assert.Nil(t, got["unit_value"])
	assert.NotContains(t, got, "material")
}

func TestPreview9_ItemInsideAListAndAnotherResource(t *testing.T) {
	t.Parallel()

	list := downgrade(t, constants.ObjectTypeItem, `{"object":"list","page_info":{"next_page_url":null,"previous_page_url":null},"data":[`+forgeItemJSON+`]}`)
	first := list["data"].([]any)[0].(map[string]any)
	assert.Equal(t, "product", first["type"])
	assert.NotContains(t, first, "sales")

	material := downgrade(t, constants.ObjectTypeMaterial, `{"id":"ml_1","object":"material","item":`+forgeItemJSON+`,"order_point":null,"lead_time":null}`)
	item := material["item"].(map[string]any)
	assert.Contains(t, item, "unit_value")
	assert.NotContains(t, item, "status")
	assert.Equal(t, "material", material["object"], "the material itself is untouched")
}

// An item reference that isn't a whole item, such as a reconcile row's {id, object, name}, gains nothing.
func TestPreview9_ItemStubUntouched(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeItem, `{"id":"itm_3","object":"item","name":"SKU-3"}`)
	assert.Equal(t, map[string]any{"id": "itm_3", "object": "item", "name": "SKU-3"}, got)
}

// preview.9 included an item's value to get the item; the item still comes, carrying its values.
func TestPreview9_ItemValueIncludeKeepsTheItem(t *testing.T) {
	t.Parallel()

	const route = "/v1/test/itm1/materials/{id}"
	RegisterRouteIncludes(route, []string{"item", "item.category"})
	q := version.TransformQuery(apiversion.V1_0_Forge_Preview9, apiversion.V1_0_Forge, constants.ObjectTypeMaterial, route,
		url.Values{"include[]": {"item.unit_value", "item.burn_rate.numerator_unit", "item.category"}})
	assert.ElementsMatch(t, []string{"item", "item.category"}, q["include[]"])
}

func TestPreview9_ItemListFilters(t *testing.T) {
	t.Parallel()

	q := version.TransformQuery(apiversion.V1_0_Forge_Preview9, apiversion.V1_0_Forge, constants.ObjectTypeItem, itemsRoute,
		url.Values{"starts_at": {"2026-01-01T00:00:00Z"}, "ends_at": {"2026-02-01T00:00:00Z"}, "supplier_id": {"ac_sup"}})
	assert.Equal(t, "2026-01-01T00:00:00Z", q.Get("created_after"))
	assert.Equal(t, "2026-02-01T00:00:00Z", q.Get("created_before"))
	assert.Equal(t, []string{"ac_sup"}, q["supplier_ids[]"])
	assert.Equal(t, []string{"active"}, q["statuses[]"], "archived items are the ones preview.9 hid as deleted")
	for _, k := range []string{"starts_at", "ends_at", "supplier_id"} {
		assert.False(t, q.Has(k))
	}

	other := version.TransformQuery(apiversion.V1_0_Forge_Preview9, apiversion.V1_0_Forge, constants.ObjectTypeItem, "/v1/catalog/items/{id}", url.Values{})
	assert.False(t, other.Has("statuses[]"), "only the list is filtered")
}
