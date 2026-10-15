//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const preview9AddressesPath = "/v1/sales/addresses"

// A preview.9 caller still uses /v1/sales/addresses with the geolocation shape and receive_calendar_id, and filters
// by type.
func TestVersionCompatAddresses_Preview9Shape(t *testing.T) {
	t.Parallel()
	preview := apiClient.WithAPIVersion(preview9Version)

	status, raw, err := preview.Post(preview9AddressesPath, map[string]any{
		"name":                uniqueName("e2e-compat-addr"),
		"type":                "drop_ship",
		"street_line_1":       "1 Compat Way",
		"street_line_2":       "Dock 3",
		"locality":            "Columbus",
		"state":               "OH",
		"postal_code":         "43204",
		"country":             "us",
		"receive_calendar_id": "occd_01e2erecvcal00000",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	created := parseJSON(raw)
	id := jsonField(created, "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(addressesPath + "/" + id) })

	geo := jsonObject(created, "geolocation")
	require.NotNil(t, geo, "preview.9 keeps the nested geolocation: %s", raw)
	assertObjectField(t, geo, "geolocation")
	assert.Equal(t, "1 Compat Way", jsonField(geo, "street_line_1"))
	assert.Equal(t, "Dock 3", jsonField(geo, "street_line_2"))
	assert.Equal(t, "Columbus", jsonField(geo, "locality"))
	assert.Equal(t, "US", jsonField(geo, "country"), "a lowercase country is stored uppercase")
	assert.Equal(t, "occd_01e2erecvcal00000", jsonField(created, "receive_calendar_id"))
	for _, gone := range []string{"line_1", "city", "country", "receive_calendar"} {
		assert.NotContains(t, created, gone)
	}

	status, raw, err = preview.Patch(preview9AddressesPath+"/"+id, map[string]any{"street_line_2": nil, "locality": "Dublin"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	geo = jsonObject(parseJSON(raw), "geolocation")
	assertNilField(t, geo, "street_line_2")
	assert.Equal(t, "Dublin", jsonField(geo, "locality"))

	// forge reads the same address flat.
	status, raw, err = apiClient.GetListRaw(addressesPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	forge := parseJSON(raw)
	assert.Equal(t, "Dublin", jsonField(forge, "city"))
	assertNilField(t, forge, "line_2")

	list, _, err := preview.GetList(preview9AddressesPath, url.Values{"type": {"standard"}, "limit": {"100"}})
	require.NoError(t, err)
	for _, item := range list.Data {
		assert.NotEqual(t, id, DataItemField(item, "id"), "a type=standard filter leaves out the drop-ship address")
	}

	status, raw, err = preview.Delete(preview9AddressesPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
}

// A preview.9 caller validates with PUT and address_line_1 and reads address_line_1 back.
func TestVersionCompatAddresses_Preview9Validate(t *testing.T) {
	t.Parallel()

	status, raw, err := apiClient.WithAPIVersion(preview9Version).Put(addressValidatePath, map[string]any{
		"address_line_1": "1600 Amphitheatre Parkway",
		"city":           "Mountain View",
		"state":          "CA",
		"postal_code":    "94043",
		"country":        "US",
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	got := parseJSON(raw)
	assertObjectField(t, got, "validated_address")
	if components := jsonObject(got, "components"); components != nil {
		assert.Contains(t, components, "address_line_1")
		assert.NotContains(t, components, "line_1")
	}
}
