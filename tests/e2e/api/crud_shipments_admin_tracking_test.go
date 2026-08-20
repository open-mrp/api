//go:build e2e

package api_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Coverage for the admin tracking overrides on shipments and shipping cases, plus the carrier
// cascade a shipment update runs onto its cases. These MUTATE the SHP-SB-001 fixture (they ship it
// and re-route it), so they restore both the dispatch state and the carrier and never run parallel.

const adminUpdateTrackingAction = "/actions/admin-update-tracking"

// Returns the id of an account carrier other than the one given, so a re-routing test never
// "changes" the carrier to the value it already had.
func otherCarrierID(t *testing.T, currentID string) string {
	t.Helper()

	list, status, err := apiClient.GetList(carriersPath, nil)
	require.NoError(t, err)
	require.Equal(t, 200, status, "carriers list should return 200")

	for _, raw := range list.Data {
		var carrier map[string]any
		require.NoError(t, json.Unmarshal(raw, &carrier))
		if id := jsonField(carrier, "id"); id != "" && id != currentID {
			return id
		}
	}
	require.FailNow(t, "the account needs a second carrier to re-route to")
	return ""
}

// Reads the shipment's current carrier id, which the freight include carries.
func shipmentCarrierID(t *testing.T, shipmentID string) string {
	t.Helper()

	freight, ok := readShipment(t, shipmentID, "freight")["freight"].(map[string]any)
	require.True(t, ok, "freight present with ?include=freight")
	carrier, ok := freight["carrier"].(map[string]any)
	require.True(t, ok, "freight carries the carrier")
	return jsonField(carrier, "id")
}

// Returns the ids of every shipping case on the shipment.
func shipmentCaseIDs(t *testing.T, shipmentID string) []string {
	t.Helper()

	cases, ok := readShipment(t, shipmentID, "shipping_cases")["shipping_cases"].(map[string]any)
	require.True(t, ok, "shipping_cases present with ?include=shipping_cases")

	data, _ := cases["data"].([]any)
	out := make([]string, 0, len(data))
	for _, raw := range data {
		sc, ok := raw.(map[string]any)
		require.True(t, ok, "case entry is an object")
		out = append(out, jsonField(sc, "id"))
	}
	return out
}

// Reads a shipping case's carrier id, which is expandable and so must be asked for.
func shippingCaseCarrierID(t *testing.T, caseID string) string {
	t.Helper()

	status, body, err := apiClient.GetListRaw(shippingCasesPath+"/"+caseID+"?include=carrier", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	carrier, ok := parseJSON(body)["carrier"].(map[string]any)
	require.True(t, ok, "carrier present with ?include=carrier")
	return jsonField(carrier, "id")
}

// Puts SHP-SB-001 back on the carrier it started on, after the dispatch has been undone.
func restoreShipmentCarrier(t *testing.T, carrierID string) {
	t.Helper()

	if shipmentCarrierID(t, sbShipmentID) == carrierID {
		return
	}
	status, body, err := apiClient.Patch(shipmentsPath+"/"+sbShipmentID,
		map[string]any{"carrier_id": carrierID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
}

// --- item 1: admin shipment tracking override ------------------------------

func TestShipmentsAdmin_TrackingOverrideReroutesAShippedShipment(t *testing.T) {
	originalCarrier := shipmentCarrierID(t, sbShipmentID)
	defer func() {
		restoreShipmentSB(t)
		restoreShipmentCarrier(t, originalCarrier)
	}()

	status, body, err := apiClient.Post(shipmentsPath+"/"+sbShipmentID+"/actions/ship",
		map[string]any{"email_customer": false}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	target := otherCarrierID(t, originalCarrier)

	// The ordinary update refuses to re-route a shipment that has already left.
	status, body, err = apiClient.Patch(shipmentsPath+"/"+sbShipmentID,
		map[string]any{"carrier_id": target}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 409, status, body)

	// The admin override is the sanctioned way through that guard.
	status, body, err = apiClient.Post(shipmentsPath+"/"+sbShipmentID+adminUpdateTrackingAction+"?include=freight",
		map[string]any{"carrier_id": target, "master_tracking_number": "1Z-ADMIN-FIX"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Equal(t, "1Z-ADMIN-FIX", jsonField(got, "master_tracking_number"))
	freight, ok := got["freight"].(map[string]any)
	require.True(t, ok, "the corrected routing rides in the response with ?include=freight")
	carrier, ok := freight["carrier"].(map[string]any)
	require.True(t, ok, "freight carries the carrier")
	assert.Equal(t, target, jsonField(carrier, "id"), "the override re-routes the shipment")

	// The correction sticks, and the cases follow the shipment onto the new carrier.
	assert.Equal(t, target, shipmentCarrierID(t, sbShipmentID))
	for _, caseID := range shipmentCaseIDs(t, sbShipmentID) {
		assert.Equal(t, target, shippingCaseCarrierID(t, caseID),
			"case %s must follow the shipment, or its tracking link points at the wrong carrier", caseID)
	}
}

func TestShipmentsAdmin_TrackingOverrideRejectsAnUnshippedShipment(t *testing.T) {
	require.Nil(t, readShipment(t, sbShipmentID)["shipped_at"], "fixture must start packed")

	status, body, err := apiClient.Post(shipmentsPath+"/"+sbShipmentID+adminUpdateTrackingAction,
		map[string]any{"master_tracking_number": "1Z-TOO-EARLY"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	assert.Contains(t, string(body), "has not been shipped yet",
		"the override is only for a dispatch that already happened")
}

// --- item 2: admin shipping-case tracking override -------------------------

func TestShippingCasesAdmin_TrackingOverrideOnAShippedCase(t *testing.T) {
	defer restoreShipmentSB(t)

	caseIDs := shipmentCaseIDs(t, sbShipmentID)
	require.NotEmpty(t, caseIDs, "fixture must be cased")
	caseID := caseIDs[0]

	status, body, err := apiClient.Post(shipmentsPath+"/"+sbShipmentID+"/actions/ship",
		map[string]any{"email_customer": false}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	status, body, err = apiClient.Post(shippingCasesPath+"/"+caseID+adminUpdateTrackingAction,
		map[string]any{"tracking_number": "1Z-CASE-FIX"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, "1Z-CASE-FIX", jsonField(parseJSON(body), "tracking_number"))

	status, body, err = apiClient.GetListRaw(shippingCasesPath+"/"+caseID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, "1Z-CASE-FIX", jsonField(parseJSON(body), "tracking_number"), "the correction is persisted")
}

func TestShippingCasesAdmin_TrackingOverrideRejectsAnUnshippedCase(t *testing.T) {
	require.Nil(t, readShipment(t, sbShipmentID)["shipped_at"], "fixture must start packed")

	caseIDs := shipmentCaseIDs(t, sbShipmentID)
	require.NotEmpty(t, caseIDs, "fixture must be cased")

	status, body, err := apiClient.Post(shippingCasesPath+"/"+caseIDs[0]+adminUpdateTrackingAction,
		map[string]any{"tracking_number": "1Z-TOO-EARLY"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	assert.Contains(t, string(body), "has not been shipped yet",
		"the override is only for a case that already went out")
}

// --- item 4: a carrier change cascades to the shipment's cases -------------

func TestShipmentsBehavioral_CarrierChangeCascadesToShippingCases(t *testing.T) {
	originalCarrier := shipmentCarrierID(t, sbShipmentID)
	defer restoreShipmentCarrier(t, originalCarrier)

	caseIDs := shipmentCaseIDs(t, sbShipmentID)
	require.NotEmpty(t, caseIDs, "fixture must be cased")

	target := otherCarrierID(t, originalCarrier)

	status, body, err := apiClient.Patch(shipmentsPath+"/"+sbShipmentID,
		map[string]any{"carrier_id": target}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	// Each case builds its own tracking deep-link from its carrier's code, so a case left on the
	// old carrier would link somewhere the parcel never went.
	for _, caseID := range caseIDs {
		assert.Equal(t, target, shippingCaseCarrierID(t, caseID), "case %s must move with the shipment", caseID)
	}
}
