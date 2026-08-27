//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Shipments: what packing a pick produces, and the two ways it can be undone.
//
// Shipping is a one-way move — void is the only route back, and it has to unwind everything
// shipping stamped: timestamps, tracking, case labels, freight, the invoice, and the order's
// fulfilment state. Deleting goes further and reopens the pick so the goods can be repacked.
//
// Delete, void, and line delete were all endpoints the suite never called.

// packedShipment runs a sales order through issue, pick, and pack, and returns the shipment that falls out.
func packedShipment(t *testing.T) map[string]any {
	t.Helper()

	pick := issuedOrderPick(t)
	pickID := jsonField(pick, "id")

	pickAllLines(t, pickID)
	packPick(t, pickID)

	numbers := pickShipmentNumbers(t, pickID)
	require.NotEmpty(t, numbers, "packing a pick must produce a shipment")

	listStatus, listBody, err := apiClient.GetListRaw(shipmentsPath, url.Values{"q": {numbers[0]}})
	require.NoError(t, err)
	requireStatus(t, 200, listStatus, listBody)

	data := jsonArray(parseJSON(listBody), "data")
	require.NotEmpty(t, data, "the packed shipment should be findable by its number: %s", string(listBody))
	shipment, ok := data[0].(map[string]any)
	require.True(t, ok)
	require.NotEmpty(t, jsonField(shipment, "id"))
	return shipment
}

func TestShipments_PackingAPickProducesAPackedShipment(t *testing.T) {
	t.Parallel()

	shipment := packedShipment(t)
	assert.Equal(t, "packed", jsonField(shipment, "status"), "a freshly packed shipment is packed, not shipped: %v", shipment)
}

func TestShipments_ShipThenVoidReturnsItToPacked(t *testing.T) {
	t.Parallel()

	shipmentID := jsonField(packedShipment(t), "id")

	status, body, err := apiClient.Post(shipmentsPath+"/"+shipmentID+"/actions/ship", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "ship must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	shipped := parseJSON(body)
	assert.Equal(t, "shipped", jsonField(shipped, "status"))
	assert.NotEmpty(t, jsonField(shipped, "shipped_at"), "shipping stamps the dispatch time")

	// Void is the only way back, and it has to clear what shipping stamped.
	voidStatus, voidBody, err := apiClient.Post(shipmentsPath+"/"+shipmentID+"/actions/void", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, voidStatus, 500, "void must not 5xx: %s", string(voidBody))
	requireStatus(t, 200, voidStatus, voidBody)

	voided := parseJSON(voidBody)
	assert.Equal(t, "packed", jsonField(voided, "status"))
	assert.Empty(t, jsonField(voided, "shipped_at"), "voiding clears the dispatch time: %v", voided)
}

// Shipping is one-way: a second call must be refused rather than re-stamping the dispatch.
func TestShipments_ShippingTwiceIsRefused(t *testing.T) {
	t.Parallel()

	shipmentID := jsonField(packedShipment(t), "id")

	first, firstBody, err := apiClient.Post(shipmentsPath+"/"+shipmentID+"/actions/ship", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, first, firstBody)

	second, secondBody, err := apiClient.Post(shipmentsPath+"/"+shipmentID+"/actions/ship", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, second, 500, "a repeat ship must not 5xx: %s", string(secondBody))
	assert.Equal(t, 409, second, "shipping an already-shipped shipment must conflict: %s", string(secondBody))
}

// Void only applies to a shipment that has actually shipped; there is nothing to unwind otherwise.
func TestShipments_VoidingAPackedShipmentIsRefused(t *testing.T) {
	t.Parallel()

	shipmentID := jsonField(packedShipment(t), "id")

	status, body, err := apiClient.Post(shipmentsPath+"/"+shipmentID+"/actions/void", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Equal(t, 409, status, "voiding a shipment that never shipped must conflict: %s", string(body))
}

// Deleting goes further than voiding: it reopens the pick so the goods can be packed again.
func TestShipments_DeleteReopensThePick(t *testing.T) {
	t.Parallel()

	pick := issuedOrderPick(t)
	pickID := jsonField(pick, "id")
	pickAllLines(t, pickID)
	packPick(t, pickID)

	before := firstPickLine(t, pickID)
	require.NotEmpty(t, jsonField(before, "packed_at"), "packing stamps the pick line")

	numbers := pickShipmentNumbers(t, pickID)
	require.NotEmpty(t, numbers)

	listStatus, listBody, err := apiClient.GetListRaw(shipmentsPath, url.Values{"q": {numbers[0]}})
	require.NoError(t, err)
	requireStatus(t, 200, listStatus, listBody)
	data := jsonArray(parseJSON(listBody), "data")
	require.NotEmpty(t, data)
	shipment, ok := data[0].(map[string]any)
	require.True(t, ok)

	status, body, err := apiClient.Delete(shipmentsPath + "/" + jsonField(shipment, "id"))
	require.NoError(t, err)
	require.Less(t, status, 500, "delete must not 5xx: %s", string(body))
	assert.Contains(t, []int{200, 204}, status, "delete should succeed: %s", string(body))

	after := firstPickLine(t, pickID)
	assert.Empty(t, jsonField(after, "packed_at"), "deleting the shipment must unpack the pick line: %v", after)
}

// Removing one line is the narrower operation: the pick keeps its packed state.
func TestShipmentLines_DeleteLeavesThePickPacked(t *testing.T) {
	t.Parallel()

	pick := issuedOrderPick(t)
	pickID := jsonField(pick, "id")
	pickAllLines(t, pickID)
	packPick(t, pickID)

	numbers := pickShipmentNumbers(t, pickID)
	require.NotEmpty(t, numbers)

	listStatus, listBody, err := apiClient.GetListRaw(shipmentsPath, url.Values{"q": {numbers[0]}})
	require.NoError(t, err)
	requireStatus(t, 200, listStatus, listBody)
	data := jsonArray(parseJSON(listBody), "data")
	require.NotEmpty(t, data)
	shipment, ok := data[0].(map[string]any)
	require.True(t, ok)
	shipmentID := jsonField(shipment, "id")

	linesStatus, linesBody, err := apiClient.GetListRaw(shipmentsPath+"/"+shipmentID+"/lines", nil)
	require.NoError(t, err)
	requireStatus(t, 200, linesStatus, linesBody)
	lines := jsonArray(parseJSON(linesBody), "data")
	require.NotEmpty(t, lines, "a packed shipment has lines: %s", string(linesBody))
	line, ok := lines[0].(map[string]any)
	require.True(t, ok)

	status, body, err := apiClient.Delete(shipmentsPath + "/" + shipmentID + "/lines/" + jsonField(line, "id"))
	require.NoError(t, err)
	require.Less(t, status, 500, "line delete must not 5xx: %s", string(body))
	assert.Contains(t, []int{200, 204}, status, "line delete should succeed: %s", string(body))

	after := firstPickLine(t, pickID)
	assert.NotEmpty(t, jsonField(after, "packed_at"), "removing one shipment line must leave the pick packed: %v", after)
}

// ──────────────────────────────────────────────
// Unknown IDs
// ──────────────────────────────────────────────

func TestShipments_ActionsOnUnknownShipmentAre404(t *testing.T) {
	t.Parallel()

	for _, action := range []string{"ship", "void"} {
		status, body, err := apiClient.Post(shipmentsPath+"/shp_doesnotexist000/actions/"+action, map[string]any{}, newIdempotencyKey())
		require.NoError(t, err)
		require.Less(t, status, 500, "%s must 404 rather than 5xx: %s", action, string(body))
		assert.Equal(t, 404, status, "%s on an unknown shipment must 404: %s", action, string(body))
	}

	delStatus, delBody, err := apiClient.Delete(shipmentsPath + "/shp_doesnotexist000")
	require.NoError(t, err)
	require.Less(t, delStatus, 500, "delete must 404 rather than 5xx: %s", string(delBody))
	assert.Equal(t, 404, delStatus, "deleting an unknown shipment must 404: %s", string(delBody))
}

func TestShipmentLines_DeleteUnknownLineIs404(t *testing.T) {
	t.Parallel()

	shipmentID := jsonField(packedShipment(t), "id")

	status, body, err := apiClient.Delete(shipmentsPath + "/" + shipmentID + "/lines/shln_doesnotexist")
	require.NoError(t, err)
	require.Less(t, status, 500, "must 404 rather than 5xx: %s", string(body))
	assert.Equal(t, 404, status, "deleting an unknown shipment line must 404: %s", string(body))
}
