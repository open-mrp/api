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

func TestPreview9_PaymentTermArchivedReadsInactive(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeCustomer, `{"id":"cus_1","object":"customer","defaults":{"payment_term":{"id":"pytm_1","object":"payment_term","status":"archived"}}}`)
	assert.Equal(t, "inactive", field(got, "defaults", "payment_term")["status"])

	active := downgrade(t, constants.ObjectTypePaymentTerm, `{"id":"pytm_2","object":"payment_term","status":"active"}`)
	assert.Equal(t, "active", active["status"])
}

func TestPreview9_ShippingTermShape(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeShippingTerm, `{"id":"shtm_1","object":"shipping_term","type":"carrier_rate","status":"active","flat_rate":null,
		"free_shipping":{"minimum_order_value":{"amount":"500.00","currency":"usd"},"service_level_scope":"selected","service_levels":null}}`)
	assert.Equal(t, "carrier_rate_freight", got["type"])
	assert.NotContains(t, got, "free_shipping")
	assert.Equal(t, "500.00", field(got, "minimum_order_value")["value"], "the threshold is the money preview.9 knew")
	assert.Contains(t, got, "free_shipping_service_levels")
	assert.Nil(t, got["free_shipping_service_levels"], "service levels stay null unless included")

	none := downgrade(t, constants.ObjectTypeShippingTerm, `{"id":"shtm_2","object":"shipping_term","type":"free","free_shipping":null}`)
	assert.Equal(t, "free_freight", none["type"])
	assert.Nil(t, none["minimum_order_value"])
	assert.Nil(t, none["free_shipping_service_levels"])

	flat := downgrade(t, constants.ObjectTypeSalesOrder, `{"id":"so_1","object":"sales_order","shipping_term":{"id":"shtm_3","object":"shipping_term","type":"flat_rate"}}`)
	assert.Equal(t, "flat_rate_freight", field(flat, "shipping_term")["type"])
	assert.NotContains(t, field(flat, "shipping_term"), "minimum_order_value", "a stub without free_shipping is left without it")
}

func TestPreview9_ShippingTermWrites(t *testing.T) {
	t.Parallel()

	created := upgradeBody(constants.ObjectTypeShippingTerm, http.MethodPost, shippingTermsRoute, map[string]any{
		"name": "Free", "type": "free_freight", "free_shipping_service_level_ids": []any{"sl_1"},
	})
	assert.Equal(t, map[string]any{"name": "Free", "type": "free", "free_shipping_service_level_ids": []any{"sl_1"}}, created,
		"the threshold fields pass through to the fields still accepted for preview.9")

	updated := upgradeBody(constants.ObjectTypeShippingTerm, http.MethodPatch, shippingTermRoute, map[string]any{"type": "flat_rate_freight"})
	assert.Equal(t, "flat_rate", updated["type"])
}

func TestPreview9_ShippingTermFreeShippingInclude(t *testing.T) {
	t.Parallel()

	RegisterRouteIncludes(shippingTermRoute, []string{"owner", "owner.account", "free_shipping.service_levels"})

	q := version.TransformQuery(apiversion.V1_0_Forge_Preview9, apiversion.V1_0_Forge, constants.ObjectTypeShippingTerm, shippingTermRoute,
		url.Values{"include[]": {"owner", "free_shipping_service_levels"}})
	assert.Equal(t, []string{"owner", "free_shipping.service_levels"}, q["include[]"])

	q = version.TransformQuery(apiversion.V1_0_Forge_Preview9, apiversion.V1_0_Forge, constants.ObjectTypeShippingTerm, shippingTermRoute,
		url.Values{"include": {"owner,free_shipping_service_levels"}})
	assert.Equal(t, []string{"owner,free_shipping.service_levels"}, q["include"])
}

func TestPreview9_UnitTypeAndBaseUnit(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeUnit, `{"object":"list","page_info":{"next_page_url":null,"previous_page_url":null},"data":[
		{"id":"gram","object":"unit","dimension":"mass"},{"id":"un_kg","object":"unit","dimension":"mass"}]}`)
	units := got["data"].([]any)
	gram, kg := units[0].(map[string]any), units[1].(map[string]any)
	assert.Equal(t, "mass", gram["type"])
	assert.NotContains(t, gram, "dimension")
	assert.Equal(t, true, gram["is_base_unit"])
	assert.Equal(t, false, kg["is_base_unit"])
}

func TestPreview9_UnitWrites(t *testing.T) {
	t.Parallel()

	created := upgradeBody(constants.ObjectTypeUnit, http.MethodPost, unitsRoute, map[string]any{"name": "Kilogram", "type": "mass", "is_base_unit": false})
	assert.Equal(t, map[string]any{"name": "Kilogram", "dimension": "mass"}, created)

	bulk := upgradeBody(constants.ObjectTypeAsyncJob, http.MethodPost, unitsBulkUpsertRoute, map[string]any{"units": []any{
		map[string]any{"name": "Kilogram", "type": "mass", "is_base_unit": false},
	}})
	assert.Equal(t, map[string]any{"name": "Kilogram", "dimension": "mass"}, bulk["units"].([]any)[0])

	q := version.TransformQuery(apiversion.V1_0_Forge_Preview9, apiversion.V1_0_Forge, constants.ObjectTypeUnit, unitsRoute, url.Values{"type": {"mass"}})
	assert.Equal(t, url.Values{"dimensions[]": {"mass"}}, q)
}
