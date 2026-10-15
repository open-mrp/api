//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A preview.9 caller still creates carriers by code, reads customer_portal_visibility and deleted_at, flags a
// service level is_default, and reads service_level_token and default_transit_days.
func TestVersionCompatCarriers_Preview9Shape(t *testing.T) {
	t.Parallel()
	preview := apiClient.WithAPIVersion(preview9Version)

	status, raw, err := preview.Post(carriersPath, map[string]any{
		"name":                       uniqueName("e2e-compat-carrier"),
		"code":                       "delivery",
		"customer_portal_visibility": "hidden",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	carrier := parseJSON(raw)
	carrierID := jsonField(carrier, "id")
	t.Cleanup(func() {
		_, _, _ = apiClient.Patch(carriersPath+"/"+carrierID, map[string]any{"default_service_level_id": nil}, newIdempotencyKey())
		_, _, _ = apiClient.Delete(carriersPath + "/" + carrierID)
	})
	assert.Equal(t, "delivery", jsonField(carrier, "code"))
	assert.Equal(t, "hidden", jsonField(carrier, "customer_portal_visibility"))
	assert.Contains(t, carrier, "deleted_at")
	assertNilField(t, carrier, "deleted_at")
	assert.NotContains(t, carrier, "type")
	assert.NotContains(t, carrier, "portal_visibility")
	assert.NotContains(t, carrier, "default_service_level")

	// forge sees the same carrier by its type.
	status, raw, err = apiClient.GetListRaw(carriersPath+"/"+carrierID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	assert.Equal(t, "local_delivery", jsonField(parseJSON(raw), "type"))

	levels := serviceLevelsPath(carrierID)
	status, raw, err = preview.Post(levels, map[string]any{
		"name":                       uniqueName("e2e-compat-sl"),
		"code":                       uniqueName("e2e-compat-sl-code"),
		"is_default":                 true,
		"default_transit_days":       3,
		"customer_portal_visibility": "hidden",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	first := parseJSON(raw)
	firstID := jsonField(first, "id")
	assert.Equal(t, "true", jsonField(first, "is_default"))
	assert.NotEmpty(t, jsonField(first, "service_level_token"))
	assert.Equal(t, "3", jsonField(first, "default_transit_days"))
	assert.Equal(t, "hidden", jsonField(first, "customer_portal_visibility"))
	assert.NotContains(t, first, "carrier")
	assert.NotContains(t, first, "code")

	// A second default takes over from the first.
	status, raw, err = preview.Post(levels, map[string]any{
		"name":       uniqueName("e2e-compat-sl2"),
		"code":       uniqueName("e2e-compat-sl2-code"),
		"is_default": false,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	secondID := jsonField(parseJSON(raw), "id")
	assert.Equal(t, "false", jsonField(parseJSON(raw), "is_default"))

	status, raw, err = preview.Patch(levels+"/"+secondID, map[string]any{"is_default": true, "default_transit_days": nil}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	assert.Equal(t, "true", jsonField(parseJSON(raw), "is_default"))
	assertNilField(t, parseJSON(raw), "default_transit_days")

	status, raw, err = preview.GetListRaw(levels+"/"+firstID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	assert.Equal(t, "false", jsonField(parseJSON(raw), "is_default"))

	// The carrier's embedded service levels and the list both read is_default.
	status, raw, err = preview.GetListRaw(carriersPath+"/"+carrierID, url.Values{"include": {"service_levels"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	embedded := map[string]string{}
	for _, item := range jsonObject(parseJSON(raw), "service_levels")["data"].([]any) {
		m := item.(map[string]any)
		embedded[jsonField(m, "id")] = jsonField(m, "is_default")
	}
	assert.Equal(t, map[string]string{firstID: "false", secondID: "true"}, embedded)

	list, _, err := preview.GetList(levels, nil)
	require.NoError(t, err)
	listed := map[string]string{}
	for _, row := range list.Data {
		m := parseJSON(row)
		listed[jsonField(m, "id")] = jsonField(m, "is_default")
		assert.NotContains(t, m, "carrier")
	}
	assert.Equal(t, map[string]string{firstID: "false", secondID: "true"}, listed)

	// forge reads the default on the carrier.
	status, raw, err = apiClient.GetListRaw(carriersPath+"/"+carrierID, url.Values{"include[]": {"default_service_level"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	assert.Equal(t, secondID, jsonField(jsonObject(parseJSON(raw), "default_service_level"), "id"))
}
