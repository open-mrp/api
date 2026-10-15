package versiontransforms

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-mrp/api/shared/apiversion"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/version"
)

func TestPreview9_CarrierShape(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeCarrier, `{"id":"cr_1","object":"carrier","type":"local_delivery","scac":"ABCD","status":"active","portal_visibility":"visible",
		"default_service_level":{"id":"crop_2","object":"service_level","code":"express","portal_visibility":"hidden","estimated_transit_days":2},
		"service_levels":{"object":"list","page_info":{"next_page_url":null,"previous_page_url":null},"data":[
			{"id":"crop_1","object":"service_level","code":"ground","portal_visibility":"visible","estimated_transit_days":5},
			{"id":"crop_2","object":"service_level","code":"express","portal_visibility":"hidden","estimated_transit_days":2}]}}`)
	assert.Equal(t, "delivery", got["code"])
	assert.NotContains(t, got, "type")
	assert.Equal(t, "visible", got["customer_portal_visibility"])
	assert.NotContains(t, got, "portal_visibility")
	assert.Contains(t, got, "deleted_at")
	assert.Nil(t, got["deleted_at"])
	assert.NotContains(t, got, "default_service_level")

	levels := got["service_levels"].(map[string]any)["data"].([]any)
	ground, express := levels[0].(map[string]any), levels[1].(map[string]any)
	assert.Equal(t, false, ground["is_default"])
	assert.Equal(t, true, express["is_default"])
	assert.Equal(t, "ground", ground["service_level_token"])
	assert.NotContains(t, ground, "code")
	assert.Equal(t, float64(5), ground["default_transit_days"])
	assert.Equal(t, "visible", ground["customer_portal_visibility"])
}

func TestPreview9_CarrierTypeOtherHasNoCode(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeCarrier, `{"id":"cr_1","object":"carrier","type":"other","default_service_level":null,"service_levels":null}`)
	assert.Contains(t, got, "code")
	assert.Nil(t, got["code"])

	fedex := downgrade(t, constants.ObjectTypeCarrier, `{"id":"cr_2","object":"carrier","type":"fedex"}`)
	assert.Equal(t, "fedex", fedex["code"])
}

func TestPreview9_ServiceLevelIsDefaultFromItsCarrier(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeServiceLevel, `{"id":"crop_1","object":"service_level","code":"ground","status":"active",
		"carrier":{"id":"cr_1","object":"carrier","type":"ups","default_service_level":{"id":"crop_1","object":"service_level","code":"ground"}}}`)
	assert.Equal(t, true, got["is_default"])
	assert.NotContains(t, got, "carrier")
	assert.Equal(t, "ground", got["service_level_token"])

	list := downgrade(t, constants.ObjectTypeServiceLevel, `{"object":"list","page_info":{"next_page_url":null,"previous_page_url":null},"data":[
		{"id":"crop_2","object":"service_level","code":"air","carrier":{"id":"cr_1","object":"carrier","type":"ups","default_service_level":{"id":"crop_1","object":"service_level"}}},
		{"id":"crop_3","object":"service_level","code":"sea"}]}`)
	data := list["data"].([]any)
	assert.Equal(t, false, data[0].(map[string]any)["is_default"])
	assert.Equal(t, false, data[1].(map[string]any)["is_default"], "no carrier to read a default from")
}

func TestPreview9_CarrierNestedInACustomer(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeCustomer, `{"id":"cus_1","object":"customer","freight_preferences":{
		"carrier":{"id":"cr_1","object":"carrier","type":"ltl","portal_visibility":"visible","default_service_level":null,"service_levels":null},
		"service_level":{"id":"crop_1","object":"service_level","code":"ground","portal_visibility":"hidden"}}}`)
	fp := got["freight_preferences"].(map[string]any)
	assert.Equal(t, "ltl", fp["carrier"].(map[string]any)["code"])
	sl := fp["service_level"].(map[string]any)
	assert.Equal(t, "ground", sl["service_level_token"])
	assert.Equal(t, "hidden", sl["customer_portal_visibility"])
}

func TestPreview9_CarrierForcedIncludes(t *testing.T) {
	t.Parallel()

	assert.Contains(t, version.ForcedIncludes(apiversion.V1_0_Forge, apiversion.V1_0_Forge_Preview9, constants.ObjectTypeCarrier), "default_service_level")
	forced := version.ForcedIncludes(apiversion.V1_0_Forge, apiversion.V1_0_Forge_Preview9, constants.ObjectTypeServiceLevel)
	assert.Contains(t, forced, "carrier")
	assert.Contains(t, forced, "carrier.default_service_level")
}

func TestPreview9_CarrierWrites(t *testing.T) {
	t.Parallel()

	for code, want := range map[string]string{"ltl1": "ltl", "delivery": "local_delivery", "freight_collect": "other", "fedex": "fedex", "will_call": "will_call"} {
		created := upgradeBody(constants.ObjectTypeCarrier, http.MethodPost, carriersRoute, map[string]any{"name": "C", "code": code, "customer_portal_visibility": "hidden"})
		assert.Equal(t, map[string]any{"name": "C", "type": want, "portal_visibility": "hidden"}, created, code)
	}
	noCode := upgradeBody(constants.ObjectTypeCarrier, http.MethodPost, carriersRoute, map[string]any{"name": "C"})
	assert.Equal(t, map[string]any{"name": "C", "type": "other"}, noCode)

	updated := upgradeBody(constants.ObjectTypeCarrier, http.MethodPatch, carrierRoute, map[string]any{"customer_portal_visibility": "visible"})
	assert.Equal(t, map[string]any{"portal_visibility": "visible"}, updated)

	level := upgradeBody(constants.ObjectTypeServiceLevel, http.MethodPost, serviceLevelsRoute, map[string]any{"name": "G", "code": "g", "is_default": true, "default_transit_days": float64(3), "customer_portal_visibility": "visible"})
	assert.Equal(t, map[string]any{"name": "G", "code": "g", "is_default": true, "estimated_transit_days": float64(3), "portal_visibility": "visible"}, level)

	levelPatch := upgradeBody(constants.ObjectTypeServiceLevel, http.MethodPatch, serviceLevelRoute, map[string]any{"default_transit_days": nil})
	assert.Equal(t, map[string]any{"estimated_transit_days": nil}, levelPatch)
}
