//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Expandable Fields ---

// receive_calendar is null until included, then the calendar the address names.
func TestAddresses_ReceiveCalendarInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(addressesPath+"/"+SeedReceivingDockAddressID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	got := parseJSON(body)
	assert.Contains(t, got, "receive_calendar")
	assertNilField(t, got, "receive_calendar")
	assert.NotContains(t, got, "receive_calendar_id")

	status, body, err = apiClient.GetListRaw(addressesPath+"/"+SeedReceivingDockAddressID, url.Values{"include[]": {"receive_calendar"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	calendar := jsonObject(parseJSON(body), "receive_calendar")
	require.NotNil(t, calendar, "the seeded dock names the receiving calendar: %s", body)
	assert.Equal(t, "occd_01e2erecvcal00000", jsonField(calendar, "id"))
	assertObjectField(t, calendar, "operating_calendar")
	assert.Equal(t, "receive", jsonField(calendar, "kind"))
}

// --- Shape ---

// Every address field is top-level, under the same names on create and in the response.
func TestAddresses_FlatShape(t *testing.T) {
	t.Parallel()

	created := createAndCleanup(t, addressesPath, map[string]any{
		"name":        uniqueName("e2e-addr-flat"),
		"type":        "standard",
		"line_1":      "1 Flat St",
		"line_2":      "Unit 2",
		"city":        "Columbus",
		"state":       "OH",
		"postal_code": "43204",
		"country":     "US",
		"phone":       "555-0100",
		"email":       "dock@example.com",
	})
	for _, key := range []string{"id", "object", "name", "type", "line_1", "line_2", "city", "state", "postal_code", "country", "phone", "email", "receive_calendar", "created_at", "updated_at"} {
		assert.Contains(t, created, key)
	}
	for _, gone := range []string{"geolocation", "street_line_1", "street_line_2", "locality", "receive_calendar_id"} {
		assert.NotContains(t, created, gone)
	}
	assert.Equal(t, "1 Flat St", jsonField(created, "line_1"))
	assert.Equal(t, "Unit 2", jsonField(created, "line_2"))
	assert.Equal(t, "Columbus", jsonField(created, "city"))
	assert.Equal(t, "dock@example.com", jsonField(created, "email"))
}

// --- Validation ---

// country is an uppercase ISO 3166-1 alpha-2 code, on create and update.
func TestAddresses_CountryMustBeAnISOCode(t *testing.T) {
	t.Parallel()

	for _, country := range []string{"us", "XX", "USA"} {
		status, body, err := apiClient.Post(addressesPath, map[string]any{"name": uniqueName("e2e-addr-iso"), "country": country}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 422, status, body)
		assert.Equal(t, "country", jsonField(jsonObject(parseJSON(body), "error"), "param"), country)
	}

	created := createAndCleanup(t, addressesPath, map[string]any{"name": uniqueName("e2e-addr-iso-ok"), "country": "CA"})
	status, body, err := apiClient.Patch(addressesPath+"/"+jsonField(created, "id"), map[string]any{"country": "ca"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, body)
}

// --- List ---

// types[] narrows to the listed types; naming both is no narrowing.
func TestAddresses_ListTypesFilter(t *testing.T) {
	t.Parallel()

	standard := jsonField(createAndCleanup(t, addressesPath, map[string]any{"name": uniqueName("e2e-addr-types-std"), "country": "US"}), "id")
	dropShip := jsonField(createAndCleanup(t, addressesPath, map[string]any{"name": uniqueName("e2e-addr-types-ds"), "type": "drop_ship", "country": "US"}), "id")

	both := listFindByField(t, addressesPath, url.Values{"types[]": {"standard", "drop_ship"}}, "id", standard)
	assert.NotNil(t, both, "both types listed returns standard addresses")
	both = listFindByField(t, addressesPath, url.Values{"types[]": {"standard", "drop_ship"}}, "id", dropShip)
	assert.NotNil(t, both, "both types listed returns drop-ship addresses")

	assert.Nil(t, listFindByField(t, addressesPath, url.Values{"types[]": {"drop_ship"}}, "id", standard))
}

// --- Delete ---

// An address a customer defaults to can't be deleted; an unused one is deleted and its stub returned.
func TestAddresses_DeleteBlockedWhileACustomerDefault(t *testing.T) {
	t.Parallel()

	customerID := dashCustomersNew(t, "e2e-addr-in-use", nil)
	_, customer := dashCustomersRead(t, apiClient, customerID, "bill_to_address")
	billTo := jsonObject(customer, "bill_to_address")
	require.NotNil(t, billTo, "the customer has a default billing address: %v", customer)

	inCustomer := apiClient.WithAccountID(customerID)
	status, body, err := inCustomer.Delete(addressesPath + "/" + jsonField(billTo, "id"))
	require.NoError(t, err)
	requireStatus(t, 409, status, body)
	apiErr := jsonObject(parseJSON(body), "error")
	assert.Equal(t, "resource_in_use", jsonField(apiErr, "code"))
	assert.Contains(t, jsonField(apiErr, "message"), "customer or supplier defaults")

	status, body, err = inCustomer.Post(addressesPath, map[string]any{"name": uniqueName("e2e-addr-spare"), "country": "US"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	spareID := jsonField(parseJSON(body), "id")
	status, body, err = inCustomer.Delete(addressesPath + "/" + spareID)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	stub := parseJSON(body)
	assert.Equal(t, spareID, jsonField(stub, "id"))
	assertObjectField(t, stub, "address")
	assert.Equal(t, "true", jsonField(stub, "deleted"))

	status, body, err = inCustomer.GetListRaw(addressesPath+"/"+spareID, nil)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
}

// The old /v1/sales/addresses routes are gone for forge callers.
func TestAddresses_SalesRouteIsGone(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw("/v1/sales/addresses", nil)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
}
