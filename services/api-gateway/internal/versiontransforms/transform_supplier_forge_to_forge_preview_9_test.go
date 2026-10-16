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

const forgeSupplierJSON = `{
	"id":"ac_s1","object":"supplier","status":"archived","name":"Acme Supplies","number":"SUP-1",
	"instructions":"Confirm by email",
	"contact":{"email":"a@acme.test","phone":null,"url":null},
	"carrier_billing":{"type":"sender","account_number":null},
	"defaults":{
		"carrier":null,"service_level":null,"payment_term":{"id":"pt_1","object":"payment_term"},"shipping_term":null,
		"bill_to_address":{"id":"ad_1","object":"address","name":"HQ"},
		"ship_to_address":null
	},
	"material_count":3,
	"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}`

func TestPreview9_SupplierShape(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeSupplier, forgeSupplierJSON)
	assert.Equal(t, "Confirm by email", got["note"])
	assert.Equal(t, "ad_1", got["bill_to_address"].(map[string]any)["id"])
	assert.Contains(t, got, "ship_to_address")
	assert.Nil(t, got["ship_to_address"])
	assert.Equal(t, float64(3), got["material_count"])
	for _, gone := range []string{"status", "instructions", "contact", "carrier_billing", "defaults"} {
		assert.NotContains(t, got, gone)
	}

	nested := downgrade(t, constants.ObjectTypePurchaseOrder, `{"id":"po_1","object":"purchase_order","supplier":{"id":"ac_s1","object":"supplier","name":"A","number":"1","defaults":{}}}`)
	supplier := nested["supplier"].(map[string]any)
	assert.Nil(t, supplier["note"])
	assert.Nil(t, supplier["material_count"])
	assert.NotContains(t, supplier, "defaults")
}

func TestPreview9_SupplierWrites(t *testing.T) {
	t.Parallel()

	created := upgradeBody(constants.ObjectTypeSupplier, http.MethodPost, suppliersRoute, map[string]any{"name": "A", "number": "1", "note": "n"})
	assert.Equal(t, map[string]any{"name": "A", "number": "1", "instructions": "n"}, created)

	updated := upgradeBody(constants.ObjectTypeSupplier, http.MethodPatch, supplierRoute, map[string]any{
		"note": "n", "update_note": true, "bill_to_address_id": "ad_2",
	})
	assert.Equal(t, map[string]any{"instructions": "n", "defaults": map[string]any{"bill_to_address_id": "ad_2"}}, updated)

	cleared := upgradeBody(constants.ObjectTypeSupplier, http.MethodPatch, supplierRoute, map[string]any{"update_note": true})
	assert.Equal(t, map[string]any{"instructions": nil}, cleared, "an applied note without a value clears the instructions")

	ignored := upgradeBody(constants.ObjectTypeSupplier, http.MethodPatch, supplierRoute, map[string]any{"note": "n", "update_note": false, "name": "B"})
	assert.Equal(t, map[string]any{"name": "B"}, ignored, "a note not applied changes nothing")
}

func TestPreview9_SupplierListFiltersAndIncludes(t *testing.T) {
	RegisterRouteIncludes(suppliersRoute, []string{"defaults.bill_to_address", "defaults.ship_to_address"})

	q := version.TransformQuery(apiversion.V1_0_Forge_Preview9, apiversion.V1_0_Forge, constants.ObjectTypeSupplier, suppliersRoute, url.Values{
		"starts_at": {"2026-01-01T00:00:00Z"},
		"ends_at":   {"2026-02-01T00:00:00Z"},
		"include[]": {"bill_to_address", "ship_to_address"},
	})
	assert.Equal(t, []string{"2026-01-01T00:00:00Z"}, q["created_after"])
	assert.Equal(t, []string{"2026-02-01T00:00:00Z"}, q["created_before"])
	assert.NotContains(t, q, "starts_at")
	assert.Equal(t, []string{"defaults.bill_to_address", "defaults.ship_to_address"}, q["include[]"])
}
