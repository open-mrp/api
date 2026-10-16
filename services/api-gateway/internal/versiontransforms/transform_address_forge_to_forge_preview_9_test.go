package versiontransforms

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-mrp/api/shared/apiversion"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/version"
)

const forgeAddressJSON = `{"id":"ad_1","object":"address","name":"Dock","type":"standard","line_1":"1 Elm","line_2":null,
	"city":"Columbus","state":"OH","postal_code":"43204","country":"US","phone":null,"email":null,
	"receive_calendar":{"id":"occd_1","object":"operating_calendar","name":"Dock days"},"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}`

func TestPreview9_AddressGetsItsGeolocationBack(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeAddress, forgeAddressJSON)
	assert.Equal(t, map[string]any{
		"id": nil, "object": "geolocation", "street_line_1": "1 Elm", "street_line_2": nil,
		"locality": "Columbus", "state": "OH", "postal_code": "43204", "country": "US",
	}, got["geolocation"])
	for _, key := range []string{"line_1", "line_2", "city", "state", "postal_code", "country", "receive_calendar"} {
		assert.NotContains(t, got, key)
	}
	assert.Equal(t, "occd_1", got["receive_calendar_id"])
	assert.Equal(t, "Dock", got["name"])
}

func TestPreview9_AddressWithoutACalendar(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeAddress, `{"id":"ad_1","object":"address","country":"CA","receive_calendar":null}`)
	assert.Contains(t, got, "receive_calendar_id")
	assert.Nil(t, got["receive_calendar_id"])
	assert.Equal(t, "CA", got["geolocation"].(map[string]any)["country"])
}

func TestPreview9_AddressInAListAndNested(t *testing.T) {
	t.Parallel()

	list := downgrade(t, constants.ObjectTypeAddress, `{"object":"list","page_info":{"next_page_url":null,"previous_page_url":null},"data":[`+forgeAddressJSON+`]}`)
	row := list["data"].([]any)[0].(map[string]any)
	assert.Equal(t, "1 Elm", row["geolocation"].(map[string]any)["street_line_1"])

	customer := downgrade(t, constants.ObjectTypeCustomer, `{"id":"cus_1","object":"customer","bill_to_address":`+forgeAddressJSON+`,"ship_to_address":null}`)
	bill := customer["bill_to_address"].(map[string]any)
	assert.Equal(t, "Columbus", bill["geolocation"].(map[string]any)["locality"])
	assert.Nil(t, customer["ship_to_address"])
}

func TestPreview9_AddressSnapshotsAreLeftAlone(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeSalesOrder, `{"id":"or_1","object":"sales_order","ship_to":{"name":"Dock","line_1":"1 Elm","country":"US","address":null}}`)
	assert.Equal(t, "1 Elm", got["ship_to"].(map[string]any)["line_1"], "a snapshot has no object and belongs to its document's transformer")
}

func TestPreview9_AddressComponentsLineNames(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeValidatedAddress, `{"object":"validated_address","status":"valid","components":{"object":"address_components","line_1":"1 Elm","line_2":"Suite 4","city":"Columbus"}}`)
	c := got["components"].(map[string]any)
	assert.Equal(t, "1 Elm", c["address_line_1"])
	assert.Equal(t, "Suite 4", c["address_line_2"])
	assert.NotContains(t, c, "line_1")
}

func TestPreview9_AddressForcedIncludes(t *testing.T) {
	t.Parallel()

	assert.Contains(t, version.ForcedIncludes(apiversion.V1_0_Forge, apiversion.V1_0_Forge_Preview9, constants.ObjectTypeAddress), "receive_calendar")
	assert.Contains(t, version.ForcedIncludes(apiversion.V1_0_Forge, apiversion.V1_0_Forge_Preview9, constants.ObjectTypeInvoice), "billing_address.receive_calendar")
}

func TestPreview9_AddressRequestBodies(t *testing.T) {
	t.Parallel()

	create := upgradeBody(constants.ObjectTypeAddress, http.MethodPost, addressesRoute, map[string]any{
		"name": "Dock", "street_line_1": "1 Elm", "street_line_2": "Suite 4", "locality": "Columbus", "state": "OH", "country": "us"})
	assert.Equal(t, map[string]any{"name": "Dock", "line_1": "1 Elm", "line_2": "Suite 4", "city": "Columbus", "state": "OH", "country": "US"}, create)

	patch := upgradeBody(constants.ObjectTypeAddress, http.MethodPatch, addressRoute, map[string]any{"street_line_2": nil})
	assert.Contains(t, patch, "line_2")
	assert.Nil(t, patch["line_2"], "a cleared line stays cleared")

	customer := upgradeBody(constants.ObjectTypeCustomer, http.MethodPost, "/v1/sales/customers", map[string]any{
		"name":            "Acme",
		"bill_to_address": map[string]any{"name": "HQ", "locality": "Columbus", "country": "US"},
		"ship_to_address": map[string]any{"name": "Dock", "street_line_1": "1 Elm", "country": "US"}})
	assert.Equal(t, "Columbus", customer["bill_to_address"].(map[string]any)["city"])
	assert.Equal(t, "1 Elm", customer["ship_to_address"].(map[string]any)["line_1"])

	order := upgradeBody(constants.ObjectTypeSalesOrder, http.MethodPatch, "/v1/sales/sales-orders/{id}", map[string]any{
		"billing_address": map[string]any{"id": "ad_1", "street_line_1": "2 Oak"}})
	// An order keeps its own copy of the address, so the saved one it named isn't edited.
	assert.Equal(t, map[string]any{"line_1": "2 Oak"}, order["bill_to"])

	rate := upgradeBody("", http.MethodPost, "/v1/operations/shipments/actions/estimate-rate", map[string]any{
		"to_address": map[string]any{"name": "Dock", "locality": "Columbus", "country": "US"}})
	assert.Equal(t, "Columbus", rate["to_address"].(map[string]any)["city"])

	validate := upgradeBody(constants.ObjectTypeValidatedAddress, http.MethodPost, validateAddressRoute, map[string]any{
		"address_line_1": "1 Elm", "address_line_2": "Suite 4", "city": "Columbus"})
	assert.Equal(t, map[string]any{"line_1": "1 Elm", "line_2": "Suite 4", "city": "Columbus"}, validate)

	other := upgradeBody(constants.ObjectTypeCarrier, http.MethodPost, "/v1/operations/carriers", map[string]any{"locality": "x"})
	assert.Equal(t, "x", other["locality"], "bodies of other endpoints are left alone")
}

func TestPreview9_AddressTypeFilter(t *testing.T) {
	t.Parallel()

	got := version.TransformQuery(apiversion.V1_0_Forge_Preview9, apiversion.V1_0_Forge, constants.ObjectTypeAddress, addressesRoute, url.Values{"type": {"drop_ship"}, "q": {"dock"}})
	assert.Equal(t, []string{"drop_ship"}, got["types[]"])
	assert.NotContains(t, got, "type")
	assert.Equal(t, "dock", got.Get("q"))
}

func TestPreview9_AddressRoutes(t *testing.T) {
	t.Parallel()

	var method, path string
	h := RoutePreview9Addresses(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { method, path = r.Method, r.URL.Path }))
	serve := func(m, p string, v version.APIVersion) {
		req := httptest.NewRequest(m, p, nil)
		req.Header.Set(apiversion.Header, v.String())
		h.ServeHTTP(httptest.NewRecorder(), req)
	}

	serve(http.MethodGet, "/v1/sales/addresses", apiversion.V1_0_Forge_Preview9)
	assert.Equal(t, "/v1/core/addresses", path)
	serve(http.MethodPatch, "/v1/sales/addresses/ad_1", apiversion.V1_0_Forge_Preview9)
	assert.Equal(t, "/v1/core/addresses/ad_1", path)
	serve(http.MethodPut, "/v1/core/addresses/actions/validate", apiversion.V1_0_Forge_Preview9)
	assert.Equal(t, http.MethodPost, method)
	serve(http.MethodGet, "/v1/sales/addresses-other", apiversion.V1_0_Forge_Preview9)
	assert.Equal(t, "/v1/sales/addresses-other", path, "only the addresses path itself moves")

	serve(http.MethodGet, "/v1/sales/addresses", apiversion.V1_0_Forge)
	assert.Equal(t, "/v1/sales/addresses", path, "forge callers are not rerouted")
	serve(http.MethodPut, "/v1/core/addresses/actions/validate", apiversion.V1_0_Forge)
	assert.Equal(t, http.MethodPut, method)
}
