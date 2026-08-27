//go:build e2e

package api_test

import (
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Closes the coverage gaps across /v1/operations/picks that the behavioural, filter, sort and
// lifecycle suites leave open: the shipments listing beyond "it is empty", the rejection paths on
// every write endpoint, and the request-shape guards (unknown include, unknown query parameter,
// unknown JSON field) each route is supposed to enforce.

// Creates and issues an order from a caller-built body and returns its pick's id. Tests that need
// a field *absent* from the create body build their own rather than going through
// issueOrderForCustomer, which can only add keys to the minimal body, never drop one.
func pickForOrderBody(t *testing.T, body map[string]any) string {
	t.Helper()

	status, respBody, err := apiClient.Post(salesOrdersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, respBody)
	orderID := jsonField(parseJSON(respBody), "id")
	deleteOrder(t, orderID)

	issueStatus, issueBody, err := apiClient.Put(salesOrdersPath+"/"+orderID+"/actions/issue", nil)
	require.NoError(t, err)
	requireStatus(t, 200, issueStatus, issueBody)

	status, respBody, err = apiClient.GetListRaw(salesOrdersPath+"/"+orderID, url.Values{"include": {"related.pick"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, respBody)
	pickID := jsonField(jsonObject(jsonObject(parseJSON(respBody), "related"), "pick"), "id")
	require.NotEmpty(t, pickID, "issuing an order creates its pick")
	return pickID
}

// Issues an order for a single line of the given quantity and returns its pick's id, so a test can
// pack it in more than one round.
func pickForQuantity(t *testing.T, customerID, quantity string) string {
	t.Helper()

	body := minimalSalesOrderCreateBody(t, customerID)
	body["lines"] = []map[string]any{
		{
			"product_id": SeedProductID,
			"quantity":   map[string]any{"value": quantity, "unit_id": SeedUnitID},
		},
	}
	return pickForOrderBody(t, body)
}

// Reads the pick's related shipments and returns their numbers, oldest first.
func pickShipmentNumbers(t *testing.T, pickID string) []string {
	t.Helper()

	status, body, err := apiClient.GetListRaw(picksPath+"/"+pickID, url.Values{"include": {"related.shipments"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	// A pick whose order has shipped nothing collapses `related` to null rather than reporting an
	// empty list, so an absent relation reads as no shipments rather than a failure.
	numbers := make([]string, 0)
	shipments := jsonObject(jsonObject(parseJSON(body), "related"), "shipments")
	for _, raw := range jsonArray(shipments, "data") {
		shipment, ok := raw.(map[string]any)
		require.True(t, ok, "related.shipments.data holds objects: %s", string(body))
		numbers = append(numbers, jsonField(shipment, "number"))
	}
	return numbers
}

// ──────────────────────────────────────────────
// Get Pick Shipments
// ──────────────────────────────────────────────

// Every partial pack adds another shipment to the pick's order, and related.shipments is the
// picker's record of what has left the building so far.
func TestCovOperationsPicks_RelatedShipmentsListsEveryPackOldestFirst(t *testing.T) {
	t.Parallel()

	customerID := leadTimeCustomer(t, "e2e-pick-ships", nil, "")
	pickID := pickForQuantity(t, customerID, "4")

	first := packPartOfPick(t, pickID, "2")
	second := packPartOfPick(t, pickID, "2")
	require.NotEqual(t, first, second, "two packs produce two distinct shipments")

	assert.Equal(t, []string{first, second}, pickShipmentNumbers(t, pickID), "shipments come back oldest first")
}

// Reads the pick's shipment numbers off the seeded pick, whose order carries exactly one.
func TestCovOperationsPicks_RelatedShipmentsOnASeededPick(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{"SHP-003"}, pickShipmentNumbers(t, SeedPickID), "PICK-001's order carries SHP-003")
}

// Sets one unpacked line to the given quantity, packs the pick, and returns the shipment's number.
func packPartOfPick(t *testing.T, pickID, quantity string) string {
	t.Helper()

	line := firstUnpackedPickLine(t, pickID)
	require.NotEmpty(t, line, "the pick has an unpacked line left to work")

	status, body, err := apiClient.Patch(picksPath+"/"+pickID+"/lines/"+line,
		map[string]any{"quantity_value": quantity}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	results := jobResults(acceptPackPick(t, pickID, 1))
	require.Len(t, results, 1, "one pack, one result")
	number := jsonField(jsonObject(results[0], "resource"), "name")
	require.NotEmpty(t, number, "the completed job names the shipment it created")
	return number
}

// ──────────────────────────────────────────────
// Retrieve / List — request shape
// ──────────────────────────────────────────────

func TestCovOperationsPicks_RetrieveUnknownPickIs404(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(picksPath+"/pk_doesnotexist00000", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "an unknown pick must not 5xx: %s", string(body))
	assert.Equal(t, 404, status, "retrieving an unknown pick must 404: %s", string(body))
}

func TestCovOperationsPicks_RejectAnUnknownInclude(t *testing.T) {
	t.Parallel()

	for _, path := range []string{picksPath, picksPath + "/" + SeedPickID} {
		status, body, err := apiClient.GetListRaw(path, url.Values{"include": {"bogus_e2e_include"}})
		require.NoError(t, err)
		assert.Equal(t, 400, status, "GET %s with an unknown include should 400: %s", path, string(body))
	}
}

func TestCovOperationsPicks_ListRejectsAnUnknownQueryParam(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(picksPath, url.Values{bogusE2EQueryParam: {"1"}})
	require.NoError(t, err)
	assertUnknownQueryParamRejected(t, picksPath, status, body)
}

func TestCovOperationsPicks_ListRejectsAnUnknownStatus(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(picksPath, url.Values{"status": {"bogus_e2e_status"}})
	require.NoError(t, err)
	require.Equal(t, 400, status, "status only accepts the documented values: %s", string(body))
	requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
}

func TestCovOperationsPicks_ListRejectsAnOutOfRangeLimit(t *testing.T) {
	t.Parallel()

	for _, limit := range []string{"0", "-1", "abc"} {
		status, body, err := apiClient.GetListRaw(picksPath, url.Values{"limit": {limit}})
		require.NoError(t, err)
		assert.Equal(t, 400, status, "limit=%s should 400: %s", limit, string(body))
	}
}

// ──────────────────────────────────────────────
// Pick line writes
// ──────────────────────────────────────────────

func TestCovOperationsPicks_UpdateLineOnUnknownPickOrLineIs404(t *testing.T) {
	t.Parallel()

	for name, path := range map[string]string{
		"unknown pick": picksPath + "/pk_doesnotexist00000/lines/" + pbLine2ID,
		"unknown line": picksPath + "/" + pbPickID + "/lines/pkln_doesnotexist0",
	} {
		status, body, err := apiClient.Patch(path, map[string]any{"quantity_value": "1"}, newIdempotencyKey())
		require.NoError(t, err)
		require.Less(t, status, 500, "%s must not 5xx: %s", name, string(body))
		assert.Equal(t, 404, status, "%s must 404: %s", name, string(body))
	}
}

// A quantity that is not a number has to be turned away at the edge. It reaches a DECIMAL column
// otherwise, and the driver's rejection surfaces as a 500 for what is a caller mistake.
func TestCovOperationsPicks_UpdateLineRejectsANonNumericQuantity(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"abc", "1.2.3", "ten"} {
		status, body, err := apiClient.Patch(picksPath+"/"+pbPickID+"/lines/"+pbLine2ID,
			map[string]any{"quantity_value": value}, newIdempotencyKey())
		require.NoError(t, err)
		require.Less(t, status, 500, "quantity_value=%q must not 5xx: %s", value, string(body))
		assert.Equal(t, 400, status, "quantity_value=%q is not a decimal: %s", value, string(body))
	}
}

func TestCovOperationsPicks_UpdateLineRejectsAnUnknownField(t *testing.T) {
	t.Parallel()

	path := picksPath + "/" + pbPickID + "/lines/" + pbLine2ID
	status, body, err := apiClient.Patch(path, map[string]any{bogusE2EJSONField: "x"}, newIdempotencyKey())
	require.NoError(t, err)
	assertJSONUnknownFieldRejected(t, "PATCH", path, status, body)
}

// Packing is what makes picking work permanent: the line can no longer be undone, and picking it
// again has nothing left to take.
func TestCovOperationsPicks_PackedLineIsPickedOutAndCannotBeVoided(t *testing.T) {
	t.Parallel()

	customerID := leadTimeCustomer(t, "e2e-pick-packed-line", nil, "")
	pickID := pickForQuantity(t, customerID, "2")

	line := firstUnpackedPickLine(t, pickID)
	require.NotEmpty(t, line)
	packPartOfPick(t, pickID, "2")

	quantities := readPickLineQuantities(t, pickID)
	require.Contains(t, quantities, line, "the packed line stays on the pick")
	packed := quantities[line]

	status, body, err := apiClient.Put(picksPath+"/"+pickID+"/lines/"+line+"/actions/void", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "the packed-line guard must not 5xx: %s", string(body))
	require.Equal(t, 400, status, "voiding a packed line is rejected: %s", string(body))
	requireErrorResponse(t, body, "validation_failed", "invalid_request_error")

	// Picking it is not an error, just a no-op — nothing is outstanding on the order line.
	status, body, err = apiClient.Put(picksPath+"/"+pickID+"/lines/"+line+"/actions/pick", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.InDelta(t, packed, readPickLineQuantities(t, pickID)[line], 0.001,
		"picking a packed line leaves its quantity alone")
}

// ──────────────────────────────────────────────
// Pack Pick
// ──────────────────────────────────────────────

func TestCovOperationsPicks_PackUnknownPickIs404(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(picksPath+"/pk_doesnotexist00000/actions/pack",
		map[string]any{"shipment_case_count": 1}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "an unknown pick must not 5xx: %s", string(body))
	assert.Equal(t, 404, status, "packing an unknown pick must 404: %s", string(body))
}

func TestCovOperationsPicks_PackRequiresACaseCount(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(picksPath+"/"+pbPickID+"/actions/pack",
		map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	require.Equal(t, 400, status, "shipment_case_count is required: %s", string(body))
	errObj := requireErrorResponse(t, body, "missing_field", "invalid_request_error")
	assert.Equal(t, "shipment_case_count", errObj["param"])
}

// A pack that closes out every line finishes the pick and leaves nothing to pack a second time.
func TestCovOperationsPicks_PackingEverythingClosesThePick(t *testing.T) {
	t.Parallel()

	customerID := leadTimeCustomer(t, "e2e-pick-pack-close", nil, "")
	pickID := pickForQuantity(t, customerID, "3")

	status, body, err := apiClient.Put(picksPath+"/"+pickID+"/actions/pick", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	acceptPackPick(t, pickID, 1)

	pick := retrievePick(t, pickID)
	assert.NotNil(t, pick["finished_at"], "every line is packed, so the pick closes")
	completion, ok := jsonObject(jsonObject(pick, "totals"), "packed")["completion"].(float64)
	require.True(t, ok)
	assert.InDelta(t, 1.0, completion, 0.001, "a fully packed pick reports packed completion 1")

	// Nothing is left with a picked quantity, so a second pack has nothing eligible.
	status, body, err = apiClient.Post(picksPath+"/"+pickID+"/actions/pack",
		map[string]any{"shipment_case_count": 1}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "re-packing a closed pick must not 5xx: %s", string(body))
	assert.Equal(t, 400, status, "a fully packed pick has nothing left to pack: %s", string(body))
}

// ──────────────────────────────────────────────
// Void Pick
// ──────────────────────────────────────────────

// Voiding is the undo for a pick that was worked wrongly, so it has to survive being run on a pick
// that was already reset — the picker cannot tell from the screen whether the first call landed.
func TestCovOperationsPicks_VoidIsRepeatable(t *testing.T) {
	t.Parallel()

	customerID := leadTimeCustomer(t, "e2e-pick-void-twice", nil, "")
	pickID := pickForQuantity(t, customerID, "2")

	status, body, err := apiClient.Put(picksPath+"/"+pickID+"/actions/pick", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	for range 2 {
		status, body, err = apiClient.Put(picksPath+"/"+pickID+"/actions/void", nil)
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
	}

	for lineID, quantity := range readPickLineQuantities(t, pickID) {
		assert.InDelta(t, 0.0, quantity, 0.001, "line %s is back to zero", lineID)
	}
	assert.Nil(t, parseJSON(body)["finished_at"], "void leaves the pick open")
}

// The picked quantity is the pick's own record, so a void must not reach through to the order it
// fulfills — the order still wants what it always wanted.
func TestCovOperationsPicks_VoidLeavesTheOrderAlone(t *testing.T) {
	t.Parallel()

	customerID := leadTimeCustomer(t, "e2e-pick-void-order", nil, "")
	pickID := pickForQuantity(t, customerID, "5")

	orderID := jsonField(jsonObject(jsonObject(retrievePick(t, pickID, "related.sales_order"), "related"), "sales_order"), "id")
	require.NotEmpty(t, orderID, "the pick names the order it fulfills")

	before := orderedQuantityOnPick(t, pickID)

	for _, action := range []string{"pick", "void"} {
		status, body, err := apiClient.Put(picksPath+"/"+pickID+"/actions/"+action, nil)
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
	}

	assert.InDelta(t, before, orderedQuantityOnPick(t, pickID), 0.001,
		"the ordered quantity is the order's, and voiding the pick does not touch it")
}

// Sums the ordered quantity across the pick's lines.
func orderedQuantityOnPick(t *testing.T, pickID string) float64 {
	t.Helper()

	var total float64
	for _, raw := range pickLines(t, pickID) {
		line, ok := raw.(map[string]any)
		require.True(t, ok)
		ordered := jsonObject(line, "ordered_quantity")
		require.NotNil(t, ordered, "ordered_quantity is a base scalar")
		value, err := strconv.ParseFloat(jsonField(ordered, "value"), 64)
		require.NoError(t, err)
		total += value
	}
	return total
}
