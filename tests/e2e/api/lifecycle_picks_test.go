//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Picks: the warehouse side of a sales order.
//
// One is created when a sales order is issued, with a line per order line, and it is where
// picked quantities are recorded before anything is packed. Picking is reversible until a
// line is packed; after that it is not, because packing is what produced a shipment.
//
// Void, the per-line actions, and the shipments listing were all endpoints the suite never
// called.

// issuedOrderPick issues a fresh sales order and returns its pick, which is the only way one is created.
func issuedOrderPick(t *testing.T) map[string]any {
	t.Helper()

	order := issueOrderForCustomer(t, SeedCustomerAccountID, nil)
	orderID := jsonField(order, "id")

	status, body, err := apiClient.GetListRaw(salesOrdersPath+"/"+orderID, url.Values{"include": {"related.pick"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	related := jsonObject(parseJSON(body), "related")
	require.NotNil(t, related, "issuing a sales order must create its pick: %s", string(body))
	pick := jsonObject(related, "pick")
	require.NotNil(t, pick, "issuing a sales order must create its pick: %s", string(body))
	require.NotEmpty(t, jsonField(pick, "id"))
	return pick
}

func pickLines(t *testing.T, pickID string) []any {
	t.Helper()
	status, body, err := apiClient.GetListRaw(picksPath+"/"+pickID, url.Values{"include": {"lines"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return jsonListData(parseJSON(body), "lines")
}

func firstPickLine(t *testing.T, pickID string) map[string]any {
	t.Helper()
	lines := pickLines(t, pickID)
	require.NotEmpty(t, lines, "a pick created from an issued order has a line per order line")
	line, ok := lines[0].(map[string]any)
	require.True(t, ok)
	return line
}

func TestPicks_PickAllLinesThenVoid(t *testing.T) {
	t.Parallel()

	pickID := jsonField(issuedOrderPick(t), "id")

	status, body, err := apiClient.Put(picksPath+"/"+pickID+"/actions/pick", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "pick-all must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	line := firstPickLine(t, pickID)
	ordered := jsonField(jsonObject(line, "ordered_quantity"), "value")
	picked := jsonField(jsonObject(line, "quantity"), "value")
	assert.Equal(t, ordered, picked, "picking all lines sets each line to its ordered quantity: %v", line)

	// Voiding undoes the picking work but keeps the pick, which is still the warehouse's record of the job.
	voidStatus, voidBody, err := apiClient.Put(picksPath+"/"+pickID+"/actions/void", nil)
	require.NoError(t, err)
	require.Less(t, voidStatus, 500, "void must not 5xx: %s", string(voidBody))
	requireStatus(t, 200, voidStatus, voidBody)
	assert.Empty(t, jsonField(parseJSON(voidBody), "finished_at"), "voiding clears the finished timestamp")

	after := firstPickLine(t, pickID)
	assertDecimalEquals(t, 0, jsonField(jsonObject(after, "quantity"), "value"), "picked quantity after void")
}

func TestPickLines_PickAndVoidASingleLine(t *testing.T) {
	t.Parallel()

	pickID := jsonField(issuedOrderPick(t), "id")
	line := firstPickLine(t, pickID)
	linePath := picksPath + "/" + pickID + "/lines/" + jsonField(line, "id")

	status, body, err := apiClient.Put(linePath+"/actions/pick", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "line pick must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	picked := parseJSON(body)
	assert.Equal(t,
		jsonField(jsonObject(picked, "ordered_quantity"), "value"),
		jsonField(jsonObject(picked, "quantity"), "value"),
		"picking a line fills it to the outstanding ordered quantity: %v", picked)

	voidStatus, voidBody, err := apiClient.Put(linePath+"/actions/void", nil)
	require.NoError(t, err)
	require.Less(t, voidStatus, 500, "line void must not 5xx: %s", string(voidBody))
	requireStatus(t, 200, voidStatus, voidBody)
	assertDecimalEquals(t, 0, jsonField(jsonObject(parseJSON(voidBody), "quantity"), "value"), "line quantity after void")
}

// Picking a line twice is not additive: the second call has nothing outstanding left to take, so a retry cannot pick more than was ordered.
func TestPickLines_PickingTwiceIsNotAdditive(t *testing.T) {
	t.Parallel()

	pickID := jsonField(issuedOrderPick(t), "id")
	line := firstPickLine(t, pickID)
	ordered := jsonField(jsonObject(line, "ordered_quantity"), "value")
	linePath := picksPath + "/" + pickID + "/lines/" + jsonField(line, "id")

	for range 2 {
		status, body, err := apiClient.Put(linePath+"/actions/pick", nil)
		require.NoError(t, err)
		require.Less(t, status, 500, "line pick must not 5xx: %s", string(body))
		requireStatus(t, 200, status, body)
	}

	after := firstPickLine(t, pickID)
	assert.Equal(t, ordered, jsonField(jsonObject(after, "quantity"), "value"),
		"a second pick must not take more than was ordered: %v", after)
}

// The pick's shipments come through its sales order, so a pick that has never been packed has none.
func TestPicks_ShipmentsIsEmptyBeforePacking(t *testing.T) {
	t.Parallel()

	pickID := jsonField(issuedOrderPick(t), "id")

	// The response is its own shape, not a paginated list, so this reads shipment_numbers
	// rather than a `data` array that would be absent either way.
	numbers, count := pickShipments(t, pickID, nil)
	assert.Empty(t, numbers, "a pick that has never been packed has shipped nothing")
	assert.Equal(t, 0, count)
}

// ──────────────────────────────────────────────
// Unknown IDs
// ──────────────────────────────────────────────

func TestPicks_ActionsOnUnknownPickAre404(t *testing.T) {
	t.Parallel()

	for _, action := range []string{"pick", "void"} {
		status, body, err := apiClient.Put(picksPath+"/pk_doesnotexist00000/actions/"+action, nil)
		require.NoError(t, err)
		require.Less(t, status, 500, "%s must 404 rather than 5xx: %s", action, string(body))
		assert.Equal(t, 404, status, "%s on an unknown pick must 404: %s", action, string(body))
	}

	getStatus, getBody, err := apiClient.GetListRaw(picksPath+"/pk_doesnotexist00000/shipments", nil)
	require.NoError(t, err)
	require.Less(t, getStatus, 500, "shipments must 404 rather than 5xx: %s", string(getBody))
	assert.Equal(t, 404, getStatus, "shipments for an unknown pick must 404: %s", string(getBody))
}

func TestPickLines_ActionsOnUnknownLineAre404(t *testing.T) {
	t.Parallel()

	pickID := jsonField(issuedOrderPick(t), "id")

	for _, action := range []string{"pick", "void"} {
		path := picksPath + "/" + pickID + "/lines/pkln_doesnotexist0/actions/" + action
		status, body, err := apiClient.Put(path, nil)
		require.NoError(t, err)
		require.Less(t, status, 500, "%s must 404 rather than 5xx: %s", action, string(body))
		assert.Equal(t, 404, status, "%s on an unknown pick line must 404: %s", action, string(body))
	}
}

// A pick line belongs to one pick. Addressing it through another must not resolve, or the path parameter is decoration.
func TestPickLines_LineFromAnotherPickIsNotAddressable(t *testing.T) {
	t.Parallel()

	firstPick := jsonField(issuedOrderPick(t), "id")
	secondPick := jsonField(issuedOrderPick(t), "id")

	foreignLineID := jsonField(firstPickLine(t, firstPick), "id")

	path := picksPath + "/" + secondPick + "/lines/" + foreignLineID + "/actions/pick"
	status, body, err := apiClient.Put(path, nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Equal(t, 404, status, "a line from another pick must not be addressable: %s", string(body))
}
