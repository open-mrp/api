//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A customer's default service level is adopted by its orders together with its default carrier, so it must be one of that carrier's. SeedServiceLevelID is SeedCarrierID's; the transit levels are SeedTransitCarrierID's.

func customerDefaultRouting(t *testing.T, customerID string) (carrierID, serviceLevelID string) {
	t.Helper()
	status, got := dashCustomersRead(t, apiClient, customerID, "freight_preferences,freight_preferences.carrier,freight_preferences.service_level")
	require.Equal(t, 200, status)
	freight := jsonObject(got, "freight_preferences")
	require.NotNil(t, freight)
	return jsonField(jsonObject(freight, "carrier"), "id"), jsonField(jsonObject(freight, "service_level"), "id")
}

func assertDefaultServiceLevelOffCarrier(t *testing.T, status int, body []byte, msg string) {
	t.Helper()
	if assert.Equal(t, 400, status, "%s: %s", msg, string(body)) {
		assertErrorParam(t, requireErrorResponse(t, body, "validation_failed", "invalid_request_error"), "default_service_level_id")
	}
}

func TestCustomerServiceLevelOnCarrier_Create(t *testing.T) {
	t.Parallel()

	body := validCustomerBody(uniqueName("e2e-cust-sl-off"))
	body["default_service_level_id"] = SeedTransitGroundServiceLevelID
	status, resp, err := apiClient.Post(customersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	if status == 201 {
		id := jsonField(parseJSON(resp), "id")
		t.Cleanup(func() { _, _, _ = apiClient.Delete(customersPath + "/" + id) })
	}
	assertDefaultServiceLevelOffCarrier(t, status, resp, "another carrier's service level")

	customerID := dashCustomersNew(t, "e2e-cust-sl-on", map[string]any{
		"default_carrier_id":       SeedTransitCarrierID,
		"default_service_level_id": SeedTransitGroundServiceLevelID,
	})
	carrier, serviceLevel := customerDefaultRouting(t, customerID)
	assert.Equal(t, SeedTransitCarrierID, carrier)
	assert.Equal(t, SeedTransitGroundServiceLevelID, serviceLevel)
}

// The pair is checked as the customer will hold it, so a default carrier or service level sent alone is checked against the other one already held.
func TestCustomerServiceLevelOnCarrier_Update(t *testing.T) {
	t.Parallel()
	customerID := dashCustomersNew(t, "e2e-cust-sl-upd", map[string]any{"default_service_level_id": SeedServiceLevelID})
	path := customersPath + "/" + customerID

	for _, tc := range []struct {
		name string
		body map[string]any
	}{
		{"carrier alone", map[string]any{"default_carrier_id": SeedTransitCarrierID}},
		{"service level alone", map[string]any{"default_service_level_id": SeedTransitGroundServiceLevelID}},
		{"both", map[string]any{"default_carrier_id": SeedSystemCarrierID, "default_service_level_id": SeedTransitGroundServiceLevelID}},
	} {
		status, body, err := apiClient.Patch(path, tc.body, newIdempotencyKey())
		require.NoError(t, err)
		assertDefaultServiceLevelOffCarrier(t, status, body, tc.name)
		carrier, serviceLevel := customerDefaultRouting(t, customerID)
		assert.Equal(t, SeedCarrierID, carrier, "%s: the customer keeps its default carrier", tc.name)
		assert.Equal(t, SeedServiceLevelID, serviceLevel, "%s: the customer keeps its default service level", tc.name)
	}

	patch := func(fields map[string]any) {
		t.Helper()
		status, body, err := apiClient.Patch(path, fields, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
	}
	patch(map[string]any{"default_carrier_id": SeedTransitCarrierID, "default_service_level_id": nil})
	carrier, serviceLevel := customerDefaultRouting(t, customerID)
	assert.Equal(t, SeedTransitCarrierID, carrier, "a new carrier with the service level cleared")
	assert.Empty(t, serviceLevel)

	patch(map[string]any{"default_service_level_id": SeedTransitGroundServiceLevelID})
	_, serviceLevel = customerDefaultRouting(t, customerID)
	assert.Equal(t, SeedTransitGroundServiceLevelID, serviceLevel, "a service level of the carrier the customer holds")
}
