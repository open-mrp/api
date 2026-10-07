//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A service level is one of its carrier's, so a record cannot ship on one carrier at another carrier's service level. SeedServiceLevelID is SeedCarrierID's; the transit levels are SeedTransitCarrierID's.

func assertServiceLevelOffCarrier(t *testing.T, status int, body []byte, msg string) {
	t.Helper()
	if assert.Equal(t, 400, status, "%s: %s", msg, string(body)) {
		assertErrorParam(t, requireErrorResponse(t, body, "", "invalid_request_error"), "service_level_id")
	}
}

func TestServiceLevelOnCarrier_SalesOrderCreate(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomer(t)

	body := minimalSalesOrderCreateBody(t, customerID)
	body["service_level_id"] = SeedTransitGroundServiceLevelID
	status, resp, err := apiClient.Post(salesOrdersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	if status == 201 {
		deleteOrder(t, jsonField(parseJSON(resp), "id"))
	}
	assertServiceLevelOffCarrier(t, status, resp, "another carrier's service level")

	body = minimalSalesOrderCreateBody(t, customerID)
	body["carrier_id"] = SeedTransitCarrierID
	body["service_level_id"] = SeedTransitGroundServiceLevelID
	status, resp, err = apiClient.Post(salesOrdersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, resp)
	orderID := jsonField(parseJSON(resp), "id")
	deleteOrder(t, orderID)
	carrier, serviceLevel, _, _ := orderRefIDs(t, orderID)
	assert.Equal(t, SeedTransitCarrierID, carrier)
	assert.Equal(t, SeedTransitGroundServiceLevelID, serviceLevel)
}

// The pair is checked as the order will hold it, so a carrier or a service level sent alone is checked against the other one already on the order.
func TestServiceLevelOnCarrier_SalesOrderUpdate(t *testing.T) {
	t.Parallel()
	status, resp, err := apiClient.Post(salesOrdersPath, minimalSalesOrderCreateBody(t, setupOrderCustomer(t)), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, resp)
	orderID := jsonField(parseJSON(resp), "id")
	deleteOrder(t, orderID)

	for _, tc := range []struct {
		name string
		body map[string]any
	}{
		{"carrier alone", map[string]any{"carrier_id": SeedTransitCarrierID}},
		{"service level alone", map[string]any{"service_level_id": SeedTransitGroundServiceLevelID}},
		{"both", map[string]any{"carrier_id": SeedSystemCarrierID, "service_level_id": SeedTransitGroundServiceLevelID}},
	} {
		status, body, err := apiClient.Patch(salesOrdersPath+"/"+orderID, tc.body, newIdempotencyKey())
		require.NoError(t, err)
		assertServiceLevelOffCarrier(t, status, body, tc.name)
		carrier, serviceLevel, _, _ := orderRefIDs(t, orderID)
		assert.Equal(t, SeedCarrierID, carrier, "%s: the order keeps its carrier", tc.name)
		assert.Equal(t, SeedServiceLevelID, serviceLevel, "%s: the order keeps its service level", tc.name)
	}

	patch := func(fields map[string]any) {
		t.Helper()
		status, body, err := apiClient.Patch(salesOrdersPath+"/"+orderID, fields, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
	}
	patch(map[string]any{"carrier_id": SeedTransitCarrierID, "service_level_id": nil})
	carrier, serviceLevel, _, _ := orderRefIDs(t, orderID)
	assert.Equal(t, SeedTransitCarrierID, carrier, "a new carrier with the service level cleared")
	assert.Empty(t, serviceLevel)

	patch(map[string]any{"service_level_id": SeedTransitGroundServiceLevelID})
	_, serviceLevel, _, _ = orderRefIDs(t, orderID)
	assert.Equal(t, SeedTransitGroundServiceLevelID, serviceLevel, "a service level of the carrier the order holds")
}

func TestServiceLevelOnCarrier_PurchaseOrderCreate(t *testing.T) {
	t.Parallel()

	body := validPurchaseOrderBody()
	body["carrier_id"] = SeedCarrierID
	body["service_level_id"] = SeedTransitGroundServiceLevelID
	status, resp, err := apiClient.Post(purchaseOrdersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	if status == 201 {
		id := jsonField(parseJSON(resp), "id")
		t.Cleanup(func() { _, _, _ = apiClient.Delete(purchaseOrdersPath + "/" + id) })
	}
	assertServiceLevelOffCarrier(t, status, resp, "another carrier's service level")

	created := createPurchaseOrder(t, func(body map[string]any) {
		body["carrier_id"] = SeedTransitCarrierID
		body["service_level_id"] = SeedTransitGroundServiceLevelID
	})
	status, resp, err = apiClient.GetListRaw(purchaseOrdersPath+"/"+jsonField(created, "id"), url.Values{"include": {"freight"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, resp)
	freight := jsonObject(parseJSON(resp), "freight")
	assert.Equal(t, SeedTransitCarrierID, jsonField(jsonObject(freight, "carrier"), "id"))
	assert.Equal(t, SeedTransitGroundServiceLevelID, jsonField(jsonObject(freight, "service_level"), "id"))
}

func TestServiceLevelOnCarrier_ShipmentUpdate(t *testing.T) {
	t.Parallel()
	_, _, shipmentID := dashShipmentsPacked(t)
	path := shipmentsPath + "/" + shipmentID

	for _, tc := range []struct {
		name string
		body map[string]any
	}{
		{"carrier alone", map[string]any{"carrier_id": SeedTransitCarrierID}},
		{"service level alone", map[string]any{"service_level_id": SeedTransitGroundServiceLevelID}},
		{"both", map[string]any{"carrier_id": SeedSystemCarrierID, "service_level_id": SeedTransitGroundServiceLevelID}},
	} {
		status, body, err := apiClient.Patch(path, tc.body, newIdempotencyKey())
		require.NoError(t, err)
		assertServiceLevelOffCarrier(t, status, body, tc.name)
		carrier, serviceLevel := dashShipmentsFreightIDs(t, shipmentID)
		assert.Equal(t, SeedCarrierID, carrier, "%s: the shipment keeps its carrier", tc.name)
		assert.Equal(t, SeedServiceLevelID, serviceLevel, "%s: the shipment keeps its service level", tc.name)
	}

	status, body, err := apiClient.Patch(path, map[string]any{
		"carrier_id": SeedTransitCarrierID, "service_level_id": SeedTransitGroundServiceLevelID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	carrier, serviceLevel := dashShipmentsFreightIDs(t, shipmentID)
	assert.Equal(t, SeedTransitCarrierID, carrier)
	assert.Equal(t, SeedTransitGroundServiceLevelID, serviceLevel)
}

func TestServiceLevelOnCarrier_ShipmentAdminTracking(t *testing.T) {
	t.Parallel()
	_, _, shipmentID := dashShipmentsPacked(t)
	status, body := shipParity(t, shipmentID)
	requireStatus(t, 200, status, body)
	path := shipmentsPath + "/" + shipmentID + adminUpdateTrackingAction

	status, body, err := apiClient.Post(path, map[string]any{"carrier_id": SeedTransitCarrierID}, newIdempotencyKey())
	require.NoError(t, err)
	assertServiceLevelOffCarrier(t, status, body, "carrier alone")
	carrier, serviceLevel := dashShipmentsFreightIDs(t, shipmentID)
	assert.Equal(t, SeedCarrierID, carrier, "the shipment keeps its carrier")
	assert.Equal(t, SeedServiceLevelID, serviceLevel, "the shipment keeps its service level")

	status, body, err = apiClient.Post(path, map[string]any{"carrier_id": SeedTransitCarrierID, "service_level_id": nil}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	carrier, serviceLevel = dashShipmentsFreightIDs(t, shipmentID)
	assert.Equal(t, SeedTransitCarrierID, carrier)
	assert.Empty(t, serviceLevel)
}
