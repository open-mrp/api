//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Receiving orders: the inbound half of a purchase order.
//
// One is created when a purchase order is issued, with a line per order line, and it is the
// only route by which purchased stock enters inventory. Receiving records what arrived;
// stocking puts it away. Voiding reverses receiving progress but never deletes the order,
// because the order is the record that goods were expected at all.
//
// Nothing here had e2e coverage: receive, void, and both line actions were among the
// endpoints never called by the suite.

const receivingOrdersPath = "/v1/operations/receiving-orders"

// issuedPurchaseOrderReceiving issues a purchase order and returns its receiving order ID,
// which is the only way one comes into existence. The order is left issued; the caller
// unissues it if it needs the purchase order deleted.
func issuedPurchaseOrderReceiving(t *testing.T) (purchaseOrderID, receivingOrderID string) {
	t.Helper()

	purchaseOrderID = jsonField(createPurchaseOrder(t, nil), "id")

	status, body := changePurchaseOrderStatus(t, purchaseOrderID, "issue")
	requireStatus(t, 200, status, body)

	getStatus, getBody, err := apiClient.GetListRaw(purchaseOrdersPath+"/"+purchaseOrderID, url.Values{"include": {"receiving_order"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	receiving := jsonObject(parseJSON(getBody), "receiving_order")
	require.NotNil(t, receiving, "issuing a purchase order must create its receiving order: %s", string(getBody))
	receivingOrderID = jsonField(receiving, "id")
	require.NotEmpty(t, receivingOrderID)

	t.Cleanup(func() {
		_, _, _ = apiClient.Put(purchaseOrdersPath+"/"+purchaseOrderID+"/actions/change-status", map[string]any{"status_change": "unissue"})
	})
	return purchaseOrderID, receivingOrderID
}

func receivingOrderLines(t *testing.T, receivingOrderID string) []any {
	t.Helper()
	status, body, err := apiClient.GetListRaw(receivingOrdersPath+"/"+receivingOrderID, url.Values{"include": {"lines"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return jsonListData(parseJSON(body), "lines")
}

func TestReceivingOrders_IssuingAPurchaseOrderCreatesOneLinePerOrderLine(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := issuedPurchaseOrderReceiving(t)

	lines := receivingOrderLines(t, receivingOrderID)
	assert.Len(t, lines, 1, "one receiving line per purchase order line")
}

func TestReceivingOrders_ReceiveRecordsTheOutstandingQuantity(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := issuedPurchaseOrderReceiving(t)

	status, body, err := apiClient.Put(receivingOrdersPath+"/"+receivingOrderID+"/actions/receive", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "receive must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	lines := receivingOrderLines(t, receivingOrderID)
	require.Len(t, lines, 1)
	line, ok := lines[0].(map[string]any)
	require.True(t, ok)
	// The purchase order line ordered 4, and nothing has been stocked, so the whole 4 is outstanding.
	assertDecimalEquals(t, 4, jsonField(jsonObject(line, "quantity"), "value"), "received quantity")
}

// Receiving is not stocking: the quantity is recorded, but nothing enters inventory until the order is stocked.
func TestReceivingOrders_ReceiveDoesNotStock(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := issuedPurchaseOrderReceiving(t)

	status, body, err := apiClient.Put(receivingOrdersPath+"/"+receivingOrderID+"/actions/receive", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	lines := receivingOrderLines(t, receivingOrderID)
	require.Len(t, lines, 1)
	line, ok := lines[0].(map[string]any)
	require.True(t, ok)
	assert.Empty(t, jsonField(line, "stocked_at"), "receiving alone must not stamp a line as stocked: %v", line)
}

func TestReceivingOrders_VoidResetsReceivingProgress(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := issuedPurchaseOrderReceiving(t)

	receiveStatus, receiveBody, err := apiClient.Put(receivingOrdersPath+"/"+receivingOrderID+"/actions/receive", nil)
	require.NoError(t, err)
	requireStatus(t, 200, receiveStatus, receiveBody)

	status, body, err := apiClient.Put(receivingOrdersPath+"/"+receivingOrderID+"/actions/void", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "void must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	lines := receivingOrderLines(t, receivingOrderID)
	require.Len(t, lines, 1, "voiding keeps one line per purchase order line")
	line, ok := lines[0].(map[string]any)
	require.True(t, ok)
	assertDecimalEquals(t, 0, jsonField(jsonObject(line, "quantity"), "value"), "received quantity after void")

	// The order itself survives: it is the record that goods were expected.
	getStatus, getBody, err := apiClient.GetListRaw(receivingOrdersPath+"/"+receivingOrderID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
}

func TestReceivingOrderLines_ReceiveAndVoidASingleLine(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := issuedPurchaseOrderReceiving(t)

	lines := receivingOrderLines(t, receivingOrderID)
	require.Len(t, lines, 1)
	line, ok := lines[0].(map[string]any)
	require.True(t, ok)
	lineID := jsonField(line, "id")
	require.NotEmpty(t, lineID)

	linePath := receivingOrdersPath + "/" + receivingOrderID + "/lines/" + lineID

	status, body, err := apiClient.Put(linePath+"/actions/receive", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "line receive must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)
	assertDecimalEquals(t, 4, jsonField(jsonObject(parseJSON(body), "quantity"), "value"), "line received quantity")

	voidStatus, voidBody, err := apiClient.Put(linePath+"/actions/void", nil)
	require.NoError(t, err)
	require.Less(t, voidStatus, 500, "line void must not 5xx: %s", string(voidBody))
	requireStatus(t, 200, voidStatus, voidBody)
	assertDecimalEquals(t, 0, jsonField(jsonObject(parseJSON(voidBody), "quantity"), "value"), "line received quantity after void")
}

// Receiving a line twice is not additive: the second call sees nothing outstanding and leaves the line alone, so a retry cannot book stock that never arrived.
func TestReceivingOrderLines_ReceivingTwiceIsNotAdditive(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := issuedPurchaseOrderReceiving(t)

	lines := receivingOrderLines(t, receivingOrderID)
	require.Len(t, lines, 1)
	line, ok := lines[0].(map[string]any)
	require.True(t, ok)
	linePath := receivingOrdersPath + "/" + receivingOrderID + "/lines/" + jsonField(line, "id")

	for range 2 {
		status, body, err := apiClient.Put(linePath+"/actions/receive", nil)
		require.NoError(t, err)
		require.Less(t, status, 500, "line receive must not 5xx: %s", string(body))
		requireStatus(t, 200, status, body)
	}

	after := receivingOrderLines(t, receivingOrderID)
	require.Len(t, after, 1)
	line2, ok := after[0].(map[string]any)
	require.True(t, ok)
	assertDecimalEquals(t, 4, jsonField(jsonObject(line2, "quantity"), "value"),
		"a second receive must not double the recorded quantity")
}

// ──────────────────────────────────────────────
// Unknown IDs
// ──────────────────────────────────────────────

func TestReceivingOrders_ActionsOnUnknownOrderAre404(t *testing.T) {
	t.Parallel()

	// Stocking is a POST, unlike the two PUT actions beside it, so it is exercised separately below.
	for _, action := range []string{"receive", "void"} {
		status, body, err := apiClient.Put(receivingOrdersPath+"/rcor_doesnotexist00/actions/"+action, nil)
		require.NoError(t, err)
		require.Less(t, status, 500, "%s must 404 rather than 5xx: %s", action, string(body))
		assert.Equal(t, 404, status, "%s on an unknown receiving order must 404: %s", action, string(body))
	}

	stockStatus, stockBody, err := apiClient.Post(receivingOrdersPath+"/rcor_doesnotexist00/actions/stock", nil, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, stockStatus, 500, "stock must 404 rather than 5xx: %s", string(stockBody))
	assert.Equal(t, 404, stockStatus, "stock on an unknown receiving order must 404: %s", string(stockBody))
}

func TestReceivingOrderLines_ActionsOnUnknownLineAre404(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := issuedPurchaseOrderReceiving(t)

	for _, action := range []string{"receive", "void"} {
		path := receivingOrdersPath + "/" + receivingOrderID + "/lines/rcln_doesnotexist00/actions/" + action
		status, body, err := apiClient.Put(path, nil)
		require.NoError(t, err)
		require.Less(t, status, 500, "%s must 404 rather than 5xx: %s", action, string(body))
		assert.Equal(t, 404, status, "%s on an unknown line must 404: %s", action, string(body))
	}
}

// A line belongs to exactly one receiving order, so addressing it through another order must not resolve — otherwise the path parameter is decoration and one account's order could act on another's line.
func TestReceivingOrderLines_LineFromAnotherOrderIsNotAddressable(t *testing.T) {
	t.Parallel()

	_, firstReceiving := issuedPurchaseOrderReceiving(t)
	_, secondReceiving := issuedPurchaseOrderReceiving(t)

	lines := receivingOrderLines(t, firstReceiving)
	require.Len(t, lines, 1)
	line, ok := lines[0].(map[string]any)
	require.True(t, ok)
	foreignLineID := jsonField(line, "id")

	path := receivingOrdersPath + "/" + secondReceiving + "/lines/" + foreignLineID + "/actions/receive"
	status, body, err := apiClient.Put(path, nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Equal(t, 404, status, "a line from another order must not be addressable: %s", string(body))
}
