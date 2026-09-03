//go:build e2e

package api_test

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Stocking a receiving order, and the line edit that precedes it.
//
// Stocking is the only route by which purchased stock enters inventory, and it is the request
// that writes receipts, records a delivery, and splits a short-received line. The lifecycle file
// covers receive and void; this covers the put-away itself and the paths around it — rejected
// quantities, lots, split allocations, short receipts, and the no-op.
//
// Update Receiving Order Line lives here because it is what an operator does first: correcting the
// quantity that actually turned up before anything is put away.

// firstLine returns a receiving order's first line, which for a single-line purchase order is the
// only one.
func firstLine(t *testing.T, receivingOrderID string) map[string]any {
	t.Helper()

	lines := receivingOrderLines(t, receivingOrderID)
	require.NotEmpty(t, lines, "an issued purchase order has at least one receiving line")

	line, ok := lines[0].(map[string]any)
	require.True(t, ok)
	return line
}

func lineQuantityValue(t *testing.T, line map[string]any) string {
	t.Helper()

	quantity := jsonObject(line, "quantity")
	require.NotNil(t, quantity, "a receiving line always carries a quantity: %v", line)
	return jsonField(quantity, "value")
}

func stockReceivingOrder(t *testing.T, receivingOrderID string, lineItems []map[string]any) (int, []byte) {
	t.Helper()

	status, body, err := apiClient.Post(
		receivingOrdersPath+"/"+receivingOrderID+"/actions/stock",
		map[string]any{"line_items": lineItems},
		newIdempotencyKey(),
	)
	require.NoError(t, err)
	require.Less(t, status, 500, "stocking must not 5xx: %s", string(body))
	return status, body
}

// deliveryForReceivingOrder finds the delivery a stocking run recorded, via the receiving order's
// related deliveries.
func deliveryForReceivingOrder(t *testing.T, receivingOrderID string) map[string]any {
	t.Helper()

	status, body, err := apiClient.GetListRaw(
		receivingOrdersPath+"/"+receivingOrderID,
		url.Values{"include": {"related", "related.deliveries"}},
	)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	related := jsonObject(parseJSON(body), "related")
	require.NotNil(t, related, "related must expand when asked for: %s", string(body))

	deliveries := jsonListData(related, "deliveries")
	require.Len(t, deliveries, 1, "one stocking run records exactly one delivery: %s", string(body))

	record, ok := deliveries[0].(map[string]any)
	require.True(t, ok)

	deliveryID := jsonField(record, "id")
	require.NotEmpty(t, deliveryID)

	getStatus, getBody, err := apiClient.GetListRaw(deliveriesPath+"/"+deliveryID, url.Values{
		"include": {"lines", "lines.item", "lines.unit_cost", "lines.location", "lines.lot"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	return parseJSON(getBody)
}

// --- Update line ---

// The endpoint takes the measure alone, as `quantity_value`.
func TestReceivingOrderLines_UpdateSetsTheReceivedQuantity(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := issuedPurchaseOrderReceiving(t)
	line := firstLine(t, receivingOrderID)
	lineID := jsonField(line, "id")

	assertDecimalEqual(t, "4", lineQuantityValue(t, line), "the line starts at the full ordered quantity")

	status, body, err := apiClient.Patch(
		receivingOrdersPath+"/"+receivingOrderID+"/lines/"+lineID,
		map[string]any{"quantity_value": "2"},
		newIdempotencyKey(),
	)
	require.NoError(t, err)
	require.Less(t, status, 500, "updating a line must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	assertDecimalEqual(t, "2", lineQuantityValue(t, parseJSON(body)), "the response reports the new quantity")
	assertDecimalEqual(t, "2", lineQuantityValue(t, firstLine(t, receivingOrderID)), "and it was persisted")
}

// The endpoint documents an omitted quantity_value as "returned unchanged", but an empty body is
// refused before it gets that far. The stronger behaviour is the useful one — it is what turns a
// client sending the wrong field name into an error rather than a silent no-op — so it is pinned
// here against the doc comment drifting back.
func TestReceivingOrderLines_UpdateWithAnEmptyBodyIsRejected(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := issuedPurchaseOrderReceiving(t)
	lineID := jsonField(firstLine(t, receivingOrderID), "id")

	status, body, err := apiClient.Patch(
		receivingOrdersPath+"/"+receivingOrderID+"/lines/"+lineID,
		map[string]any{},
		newIdempotencyKey(),
	)
	require.NoError(t, err)
	require.Less(t, status, 500, "an empty body is a client error: %s", string(body))
	assert.Equal(t, 400, status, "a PATCH must name at least one field: %s", string(body))

	assertDecimalEqual(t, "4", lineQuantityValue(t, firstLine(t, receivingOrderID)),
		"and the line is untouched")
}

func TestReceivingOrderLines_UpdateRejectsAnUnknownBodyField(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := issuedPurchaseOrderReceiving(t)
	lineID := jsonField(firstLine(t, receivingOrderID), "id")
	path := receivingOrdersPath + "/" + receivingOrderID + "/lines/" + lineID

	status, body, err := apiClient.Patch(path, map[string]any{
		bogusE2EJSONField: "x",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assertJSONUnknownFieldRejected(t, "PATCH", path, status, body)
}

// A nested quantity object is the shape a caller reaches for by habit, and the field the endpoint
// actually takes is a bare decimal string. Sending the object must fail rather than be ignored:
// silently dropping it leaves the operator looking at the quantity they thought they had changed.
func TestReceivingOrderLines_UpdateRejectsANestedQuantityObject(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := issuedPurchaseOrderReceiving(t)
	lineID := jsonField(firstLine(t, receivingOrderID), "id")

	status, body, err := apiClient.Patch(
		receivingOrdersPath+"/"+receivingOrderID+"/lines/"+lineID,
		map[string]any{"quantity": map[string]any{"value": "2", "unit_id": SeedUnitID}},
		newIdempotencyKey(),
	)
	require.NoError(t, err)
	require.Less(t, status, 500, "a wrongly shaped body is a client error: %s", string(body))
	assert.Equal(t, 400, status,
		"the endpoint takes quantity_value, and must reject a nested quantity rather than ignore it: %s", string(body))

	assertDecimalEqual(t, "4", lineQuantityValue(t, firstLine(t, receivingOrderID)),
		"and the line is untouched either way")
}

func TestReceivingOrderLines_UpdateOnUnknownLineIs404(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := issuedPurchaseOrderReceiving(t)

	status, body, err := apiClient.Patch(
		receivingOrdersPath+"/"+receivingOrderID+"/lines/rcln_doesnotexist00",
		map[string]any{"quantity_value": "1"},
		newIdempotencyKey(),
	)
	require.NoError(t, err)
	require.Less(t, status, 500, "an unknown line must 404 rather than 5xx: %s", string(body))
	assert.Equal(t, 404, status, "an unknown line is a 404: %s", string(body))
}

// --- Stocking ---

func TestReceivingOrders_StockPutsTheQuantityAway(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := issuedPurchaseOrderReceiving(t)
	lineID := jsonField(firstLine(t, receivingOrderID), "id")

	status, body := stockReceivingOrder(t, receivingOrderID, []map[string]any{{
		"receiving_order_line_id": lineID,
		"allocations":             []map[string]any{{"quantity": "4", "location_id": SeedLocationID}},
	}})
	requireStatus(t, 200, status, body)

	assert.NotEmpty(t, jsonField(firstLine(t, receivingOrderID), "stocked_at"),
		"a stocked line records when it was put away")
}

func TestReceivingOrders_StockCompletesTheOrderWhenEveryLineIsStocked(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := issuedPurchaseOrderReceiving(t)
	lineID := jsonField(firstLine(t, receivingOrderID), "id")

	status, body := stockReceivingOrder(t, receivingOrderID, []map[string]any{{
		"receiving_order_line_id": lineID,
		"allocations":             []map[string]any{{"quantity": "4", "location_id": SeedLocationID}},
	}})
	requireStatus(t, 200, status, body)

	assert.NotEmpty(t, jsonField(parseJSON(body), "completed_at"),
		"stocking the last outstanding line completes the order")
}

// Each allocation becomes its own inventory receipt, so a line split across two locations produces
// two delivery lines rather than one summed row.
func TestReceivingOrders_StockSplitsAcrossLocationsIntoSeparateDeliveryLines(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := issuedPurchaseOrderReceiving(t)
	lineID := jsonField(firstLine(t, receivingOrderID), "id")

	status, body := stockReceivingOrder(t, receivingOrderID, []map[string]any{{
		"receiving_order_line_id": lineID,
		"allocations": []map[string]any{
			{"quantity": "1", "location_id": SeedLocationID},
			{"quantity": "3", "location_id": SeedLocationID},
		},
	}})
	requireStatus(t, 200, status, body)

	lines := jsonListData(deliveryForReceivingOrder(t, receivingOrderID), "lines")
	assert.Len(t, lines, 2, "one delivery line per allocation: %v", lines)
}

// An allocation may be recorded without naming a location, which is the path an account that does
// not use storage locations takes.
func TestReceivingOrders_StockWithoutALocationIsAccepted(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := issuedPurchaseOrderReceiving(t)
	lineID := jsonField(firstLine(t, receivingOrderID), "id")

	status, body := stockReceivingOrder(t, receivingOrderID, []map[string]any{{
		"receiving_order_line_id": lineID,
		"allocations":             []map[string]any{{"quantity": "4"}},
	}})
	requireStatus(t, 200, status, body)

	deliveryLines := jsonListData(deliveryForReceivingOrder(t, receivingOrderID), "lines")
	require.Len(t, deliveryLines, 1)
	line, ok := deliveryLines[0].(map[string]any)
	require.True(t, ok)
	assertNilField(t, line, "location")
}

// A lot number creates the lot on first use and applies to every allocation on the line.
func TestReceivingOrders_StockUnderALotRecordsItOnEveryDeliveryLine(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := issuedPurchaseOrderReceiving(t)
	lineID := jsonField(firstLine(t, receivingOrderID), "id")
	lotNumber := uniqueName("E2E-LOT")

	status, body := stockReceivingOrder(t, receivingOrderID, []map[string]any{{
		"receiving_order_line_id": lineID,
		"lot_number":              lotNumber,
		"allocations": []map[string]any{
			{"quantity": "2", "location_id": SeedLocationID},
			{"quantity": "2", "location_id": SeedLocationID},
		},
	}})
	requireStatus(t, 200, status, body)

	deliveryLines := jsonListData(deliveryForReceivingOrder(t, receivingOrderID), "lines")
	require.Len(t, deliveryLines, 2)

	for i, raw := range deliveryLines {
		line, ok := raw.(map[string]any)
		require.True(t, ok)

		lot := jsonObject(line, "lot")
		require.NotNil(t, lot, "delivery line %d must carry the lot: %v", i, line)
		assert.Equal(t, lotNumber, jsonField(lot, "lot_number"))
	}
}

// A refused quantity is recorded on the delivery and on the line, but never enters inventory. It
// produces its own delivery line, marked rejected rather than accepted.
func TestReceivingOrders_StockRecordsARejectedQuantityWithoutStockingIt(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := issuedPurchaseOrderReceiving(t)
	lineID := jsonField(firstLine(t, receivingOrderID), "id")

	status, body := stockReceivingOrder(t, receivingOrderID, []map[string]any{{
		"receiving_order_line_id": lineID,
		"rejected_quantity":       "1",
		"allocations":             []map[string]any{{"quantity": "3", "location_id": SeedLocationID}},
	}})
	requireStatus(t, 200, status, body)

	rejected := jsonObject(firstLine(t, receivingOrderID), "rejected_quantity")
	require.NotNil(t, rejected, "the refused quantity is recorded on the line")
	assertDecimalEqual(t, "1", jsonField(rejected, "value"))

	deliveryLines := jsonListData(deliveryForReceivingOrder(t, receivingOrderID), "lines")
	require.Len(t, deliveryLines, 2, "one line per allocation plus one for the refusal: %v", deliveryLines)

	var accepted, refused int
	for _, raw := range deliveryLines {
		line, ok := raw.(map[string]any)
		require.True(t, ok)

		if _, isRejected := line["rejected_at"]; isRejected && line["rejected_at"] != nil {
			refused++
			continue
		}
		accepted++
	}
	assert.Equal(t, 1, refused, "exactly one delivery line records the refusal")
	assert.Equal(t, 1, accepted, "exactly one delivery line records the accepted allocation")
}

// A line stocked short of its ordered quantity leaves a remainder still expected, so the order is
// not silently closed on a partial delivery.
func TestReceivingOrders_StockingShortCreatesARemainderLine(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := issuedPurchaseOrderReceiving(t)
	lineID := jsonField(firstLine(t, receivingOrderID), "id")

	patchStatus, patchBody, err := apiClient.Patch(
		receivingOrdersPath+"/"+receivingOrderID+"/lines/"+lineID,
		map[string]any{"quantity_value": "1"},
		newIdempotencyKey(),
	)
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	status, body := stockReceivingOrder(t, receivingOrderID, []map[string]any{{
		"receiving_order_line_id": lineID,
		"allocations":             []map[string]any{{"quantity": "1", "location_id": SeedLocationID}},
	}})
	requireStatus(t, 200, status, body)

	lines := receivingOrderLines(t, receivingOrderID)
	assert.Len(t, lines, 2, "the outstanding 3 becomes a new unstocked line: %v", lines)

	var unstocked int
	for _, raw := range lines {
		line, ok := raw.(map[string]any)
		require.True(t, ok)
		if line["stocked_at"] == nil {
			unstocked++
		}
	}
	assert.Equal(t, 1, unstocked, "exactly one line is still expected")
}

// Stocking an order with nothing left to put away is a no-op rather than a second delivery.
func TestReceivingOrders_StockingAnAlreadyStockedOrderRecordsNothingNew(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := issuedPurchaseOrderReceiving(t)
	lineID := jsonField(firstLine(t, receivingOrderID), "id")

	status, body := stockReceivingOrder(t, receivingOrderID, []map[string]any{{
		"receiving_order_line_id": lineID,
		"allocations":             []map[string]any{{"quantity": "4", "location_id": SeedLocationID}},
	}})
	requireStatus(t, 200, status, body)

	secondStatus, secondBody := stockReceivingOrder(t, receivingOrderID, nil)
	requireStatus(t, 200, secondStatus, secondBody)

	// Still exactly one delivery: deliveryForReceivingOrder requires it.
	deliveryForReceivingOrder(t, receivingOrderID)
}

// A line omitted from line_items is still marked stocked, but contributes nothing to inventory and
// no delivery line. That is a sharp edge worth pinning: silence means "nothing arrived", not
// "leave it alone".
func TestReceivingOrders_StockMarksOmittedLinesStockedWithoutStockingThem(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := issuedPurchaseOrderReceiving(t)

	status, body := stockReceivingOrder(t, receivingOrderID, []map[string]any{})
	requireStatus(t, 200, status, body)

	assert.NotEmpty(t, jsonField(firstLine(t, receivingOrderID), "stocked_at"),
		"an omitted line is still marked stocked")
}

// --- Stocking validation ---

func TestReceivingOrders_StockIgnoresAnAllocationForAnotherOrdersLine(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := issuedPurchaseOrderReceiving(t)
	_, otherOrderID := issuedPurchaseOrderReceiving(t)
	foreignLineID := jsonField(firstLine(t, otherOrderID), "id")

	status, body := stockReceivingOrder(t, receivingOrderID, []map[string]any{{
		"receiving_order_line_id": foreignLineID,
		"allocations":             []map[string]any{{"quantity": "1", "location_id": SeedLocationID}},
	}})
	requireStatus(t, 200, status, body)

	assertDecimalEqual(t, "4", lineQuantityValue(t, firstLine(t, otherOrderID)),
		"the other order's line was not touched through this one")
}

// A line id the order does not own is ignored rather than refused, and the order's own unstocked
// lines are still marked stocked. Worth pinning: a client that sends a stale line id gets a 200 and
// an order it did not mean to close.
func TestReceivingOrders_StockIgnoresAnUnknownLine(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := issuedPurchaseOrderReceiving(t)

	status, body := stockReceivingOrder(t, receivingOrderID, []map[string]any{{
		"receiving_order_line_id": "rcln_doesnotexist00",
		"allocations":             []map[string]any{{"quantity": "1"}},
	}})
	requireStatus(t, 200, status, body)

	assert.NotEmpty(t, jsonField(parseJSON(body), "completed_at"),
		"the order's own lines were still swept as stocked: %s", string(body))
}

func TestReceivingOrders_StockTreatsAMalformedQuantityAsNothingToPutAway(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := issuedPurchaseOrderReceiving(t)
	lineID := jsonField(firstLine(t, receivingOrderID), "id")

	for _, quantity := range []string{"not-a-number", ""} {
		t.Run(fmt.Sprintf("quantity=%q", quantity), func(t *testing.T) {
			status, body := stockReceivingOrder(t, receivingOrderID, []map[string]any{{
				"receiving_order_line_id": lineID,
				"allocations":             []map[string]any{{"quantity": quantity}},
			}})
			require.Less(t, status, 500, "a malformed quantity must not 5xx: %s", string(body))
			assert.Equal(t, 200, status,
				"an unparseable allocation quantity is currently treated as nothing to put away: %s", string(body))
		})
	}
}

func TestReceivingOrders_StockOnUnknownOrderIs404(t *testing.T) {
	t.Parallel()

	status, body := stockReceivingOrder(t, "rcor_doesnotexist00", nil)
	assert.Equal(t, 404, status, "an unknown receiving order is a 404: %s", string(body))
}

// --- List status filter ---

// The API's word for a finished receiving order is `completed`. The dashboard calls the same state
// "closed", and sending that word instead has to fail loudly rather than being read as a default.
func TestReceivingOrders_ListRejectsAnUnknownStatus(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"closed", "bogus_e2e_status"} {
		t.Run(value, func(t *testing.T) {
			t.Parallel()

			status, body, err := apiClient.GetListRaw(receivingOrdersPath, url.Values{"status": {value}})
			require.NoError(t, err)
			require.Less(t, status, 500, "an unknown status is a client error: %s", string(body))
			require.Equal(t, 400, status, "status only accepts open, completed and all: %s", string(body))
			requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
		})
	}
}

func TestReceivingOrders_ListAcceptsTheDocumentedStatuses(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"open", "completed", "all"} {
		t.Run(value, func(t *testing.T) {
			t.Parallel()

			status, body, err := apiClient.GetListRaw(receivingOrdersPath, url.Values{"status": {value}, "limit": {"1"}})
			require.NoError(t, err)
			require.Less(t, status, 500, "status=%s must not 5xx: %s", value, string(body))
			assert.Equal(t, 200, status, "status=%s is a documented value: %s", value, string(body))
		})
	}
}

// Completed orders are hidden when status is omitted, so the default page is not "everything".
func TestReceivingOrders_ListHidesCompletedOrdersByDefault(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(receivingOrdersPath, url.Values{"limit": {"20"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)

	for _, raw := range list.Data {
		assert.Nil(t, parseJSON(raw)["completed_at"],
			"the default page shows open orders only: %s", string(raw))
	}
}

// --- Totals ---

// Stocking progress is reported off totals rather than counted by the caller, so the figure has to
// move as lines are put away.
func TestReceivingOrders_TotalsReportStockingCompletion(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := issuedPurchaseOrderReceiving(t)

	status, body, err := apiClient.GetListRaw(receivingOrdersPath+"/"+receivingOrderID, url.Values{"include": {"totals"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	totals := jsonObject(parseJSON(body), "totals")
	require.NotNil(t, totals, "totals must expand when asked for: %s", string(body))
	assertObjectField(t, totals, "receiving_order_totals")
	assert.NotEmpty(t, jsonField(totals, "ordered"), "the ordered value is the baseline")

	stocked := jsonObject(totals, "stocked")
	require.NotNil(t, stocked, "totals always name the stocked stage: %v", totals)
	assertDecimalEqual(t, "0", jsonField(stocked, "completion"), "nothing is stocked yet")

	lineID := jsonField(firstLine(t, receivingOrderID), "id")
	stockStatus, stockBody := stockReceivingOrder(t, receivingOrderID, []map[string]any{{
		"receiving_order_line_id": lineID,
		"allocations":             []map[string]any{{"quantity": "4", "location_id": SeedLocationID}},
	}})
	requireStatus(t, 200, stockStatus, stockBody)

	afterStatus, afterBody, err := apiClient.GetListRaw(receivingOrdersPath+"/"+receivingOrderID, url.Values{"include": {"totals"}})
	require.NoError(t, err)
	requireStatus(t, 200, afterStatus, afterBody)

	afterStocked := jsonObject(jsonObject(parseJSON(afterBody), "totals"), "stocked")
	require.NotNil(t, afterStocked)
	assertDecimalEqual(t, "1", jsonField(afterStocked, "completion"),
		"a fully stocked order reports completion 1")
}

func TestReceivingOrders_TotalsAreNullWithoutInclude(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := issuedPurchaseOrderReceiving(t)

	status, body, err := apiClient.GetListRaw(receivingOrdersPath+"/"+receivingOrderID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	assertNilField(t, parseJSON(body), "totals")
}

// The deliveries booked against an order are what its receiving history is made of. They were
// carried on the retrieve but dropped from the list mapping, so both are checked.
func TestReceivingOrders_ListRelatedNamesItsDeliveries(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(receivingOrdersPath, url.Values{
		"include": {"related", "related.deliveries"},
		"status":  {"all"},
		"limit":   {"10"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	require.NotEmpty(t, list.Data)

	var withDeliveries int
	for _, raw := range list.Data {
		related := jsonObject(parseJSON(raw), "related")
		require.NotNil(t, related, "related must expand on the list: %s", string(raw))
		if refs := jsonListData(related, "deliveries"); len(refs) > 0 {
			withDeliveries++
			ref, ok := refs[0].(map[string]any)
			require.True(t, ok)
			assert.NotEmpty(t, jsonField(ref, "id"))
			assert.NotEmpty(t, jsonField(ref, "number"))
		}
	}
	assert.Positive(t, withDeliveries, "a seeded receiving order has deliveries booked against it")
}

// A freshly issued order has nothing booked against it yet, so the list has to be present and
// empty rather than absent.
func TestReceivingOrders_RelatedDeliveriesIsEmptyBeforeStocking(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := issuedPurchaseOrderReceiving(t)

	status, body, err := apiClient.GetListRaw(receivingOrdersPath+"/"+receivingOrderID, url.Values{
		"include": {"related", "related.deliveries"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	related := jsonObject(parseJSON(body), "related")
	require.NotNil(t, related)
	assert.Empty(t, jsonListData(related, "deliveries"),
		"nothing has been stocked yet: %s", string(body))
}
