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

// pairs is a quantity in the seeded pair unit, the unit the seeded purchase order lines are ordered
// and received in.
func pairs(value string) map[string]any {
	return map[string]any{"value": value, "unit_id": SeedUnitID}
}

// dozens is a quantity in the seeded dozen, which belongs to the seeded product's unit group
// alongside the pair: a dozen is twelve each, so six pairs. It is a unit a line may be counted or put
// away in without being the one it was ordered in.
func dozens(value string) map[string]any {
	return map[string]any{"value": value, "unit_id": seedDozenUnitID}
}

func lineQuantityUnitID(t *testing.T, line map[string]any) string {
	t.Helper()

	unit := jsonObject(jsonObject(line, "quantity"), "unit")
	require.NotNil(t, unit, "the line's quantity unit must expand: %v", line)
	return jsonField(unit, "id")
}

func patchReceivingOrderLine(t *testing.T, receivingOrderID, lineID string, body map[string]any) (int, []byte) {
	t.Helper()

	status, respBody, err := apiClient.Patch(
		receivingOrdersPath+"/"+receivingOrderID+"/lines/"+lineID,
		body,
		newIdempotencyKey(),
	)
	require.NoError(t, err)
	require.Less(t, status, 500, "updating a line must not 5xx: %s", string(respBody))
	return status, respBody
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

// The endpoint takes a quantity with its unit, as every other quantity in the API does.
func TestReceivingOrderLines_UpdateSetsTheReceivedQuantity(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := receivedPurchaseOrderReceiving(t)
	line := firstLine(t, receivingOrderID)
	lineID := jsonField(line, "id")

	assertDecimalEqual(t, "4", lineQuantityValue(t, line), "receiving records the full outstanding quantity on the line")

	status, body, err := apiClient.Patch(
		receivingOrdersPath+"/"+receivingOrderID+"/lines/"+lineID,
		map[string]any{"quantity": pairs("2")},
		newIdempotencyKey(),
	)
	require.NoError(t, err)
	require.Less(t, status, 500, "updating a line must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	assertDecimalEqual(t, "2", lineQuantityValue(t, parseJSON(body)), "the response reports the new quantity")
	assertDecimalEqual(t, "2", lineQuantityValue(t, firstLine(t, receivingOrderID)), "and it was persisted")
}

// The endpoint documents an omitted quantity as "returned unchanged", but an empty body is
// refused before it gets that far. The stronger behaviour is the useful one — it is what turns a
// client sending the wrong field name into an error rather than a silent no-op — so it is pinned
// here against the doc comment drifting back.
func TestReceivingOrderLines_UpdateWithAnEmptyBodyIsRejected(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := receivedPurchaseOrderReceiving(t)
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

	_, receivingOrderID := receivedPurchaseOrderReceiving(t)
	lineID := jsonField(firstLine(t, receivingOrderID), "id")
	path := receivingOrdersPath + "/" + receivingOrderID + "/lines/" + lineID

	status, body, err := apiClient.Patch(path, map[string]any{
		bogusE2EJSONField: "x",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assertJSONUnknownFieldRejected(t, "PATCH", path, status, body)
}

// The bare `quantity_value` the endpoint took before the unit was carried must now fail rather than
// be ignored: silently dropping it leaves the operator looking at the quantity they thought they had
// changed.
func TestReceivingOrderLines_UpdateRejectsTheRetiredQuantityValue(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := receivedPurchaseOrderReceiving(t)
	lineID := jsonField(firstLine(t, receivingOrderID), "id")

	status, body := patchReceivingOrderLine(t, receivingOrderID, lineID, map[string]any{"quantity_value": "2"})
	assert.Equal(t, 400, status, "quantity_value is no longer a field of this endpoint: %s", string(body))

	assertDecimalEqual(t, "4", lineQuantityValue(t, firstLine(t, receivingOrderID)),
		"and the line is untouched either way")
}

// The dashboard's receiving screen lets an operator count a line in any unit of the item's group,
// so the line takes the unit along with the value.
func TestReceivingOrderLines_UpdateRecordsTheUnitItWasCountedIn(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := receivedPurchaseOrderReceiving(t)
	lineID := jsonField(firstLine(t, receivingOrderID), "id")

	status, body := patchReceivingOrderLine(t, receivingOrderID, lineID, map[string]any{"quantity": dozens("1")})
	requireStatus(t, 200, status, body)

	got, getBody, err := apiClient.GetListRaw(receivingOrdersPath+"/"+receivingOrderID, url.Values{"include": {"lines", "lines.quantity", "lines.quantity.unit"}})
	require.NoError(t, err)
	requireStatus(t, 200, got, getBody)
	line, ok := jsonListData(parseJSON(getBody), "lines")[0].(map[string]any)
	require.True(t, ok)
	assertDecimalEqual(t, "1", lineQuantityValue(t, line))
	assert.Equal(t, seedDozenUnitID, lineQuantityUnitID(t, line), "the line is now counted in dozens")
}

func TestReceivingOrderLines_UpdateRejectsAUnitOutsideTheItemsGroup(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := receivedPurchaseOrderReceiving(t)
	lineID := jsonField(firstLine(t, receivingOrderID), "id")

	status, body := patchReceivingOrderLine(t, receivingOrderID, lineID, map[string]any{
		"quantity": map[string]any{"value": "2", "unit_id": SeedMaterialUnitID},
	})
	assert.Equal(t, 400, status, "pounds are not a unit socks are counted in: %s", string(body))
	assertDecimalEqual(t, "4", lineQuantityValue(t, firstLine(t, receivingOrderID)), "and the line is untouched")
}

func TestReceivingOrderLines_UpdateRejectsANegativeQuantity(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := receivedPurchaseOrderReceiving(t)
	lineID := jsonField(firstLine(t, receivingOrderID), "id")

	status, body := patchReceivingOrderLine(t, receivingOrderID, lineID, map[string]any{"quantity": pairs("-1")})
	assert.Equal(t, 400, status, "a received quantity cannot be negative: %s", string(body))
}

// A stocked line's quantity is what went into inventory. Editing or voiding it afterwards would
// leave the receiving order disagreeing with the stock it booked, so both are refused.
func TestReceivingOrderLines_AStockedLineCannotBeChanged(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := receivedPurchaseOrderReceiving(t)
	lineID := jsonField(firstLine(t, receivingOrderID), "id")

	patchStatus, patchBody := patchReceivingOrderLine(t, receivingOrderID, lineID, map[string]any{"quantity": pairs("1")})
	requireStatus(t, 200, patchStatus, patchBody)
	stockStatus, stockBody := stockReceivingOrder(t, receivingOrderID, []map[string]any{{
		"receiving_order_line_id": lineID,
		"allocations":             []map[string]any{{"quantity": pairs("1"), "location_id": SeedLocationID}},
	}})
	requireStatus(t, 200, stockStatus, stockBody)

	status, body := patchReceivingOrderLine(t, receivingOrderID, lineID, map[string]any{"quantity": pairs("3")})
	assert.Equal(t, 400, status, "a stocked line cannot be edited: %s", string(body))

	voidStatus, voidBody, err := apiClient.Put(receivingOrdersPath+"/"+receivingOrderID+"/lines/"+lineID+"/actions/void", nil)
	require.NoError(t, err)
	assert.Equal(t, 400, voidStatus, "a stocked line cannot be voided: %s", string(voidBody))

	receiveStatus, receiveBody, err := apiClient.Put(receivingOrdersPath+"/"+receivingOrderID+"/lines/"+lineID+"/actions/receive", nil)
	require.NoError(t, err)
	assert.Equal(t, 400, receiveStatus, "a stocked line cannot be received again: %s", string(receiveBody))
}

func TestReceivingOrderLines_UpdateOnUnknownLineIs404(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := receivedPurchaseOrderReceiving(t)

	status, body, err := apiClient.Patch(
		receivingOrdersPath+"/"+receivingOrderID+"/lines/rcln_doesnotexist00",
		map[string]any{"quantity": pairs("1")},
		newIdempotencyKey(),
	)
	require.NoError(t, err)
	require.Less(t, status, 500, "an unknown line must 404 rather than 5xx: %s", string(body))
	assert.Equal(t, 404, status, "an unknown line is a 404: %s", string(body))
}

// --- Stocking ---

func TestReceivingOrders_StockPutsTheQuantityAway(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := receivedPurchaseOrderReceiving(t)
	lineID := jsonField(firstLine(t, receivingOrderID), "id")

	status, body := stockReceivingOrder(t, receivingOrderID, []map[string]any{{
		"receiving_order_line_id": lineID,
		"allocations":             []map[string]any{{"quantity": pairs("4"), "location_id": SeedLocationID}},
	}})
	requireStatus(t, 200, status, body)

	assert.NotEmpty(t, jsonField(firstLine(t, receivingOrderID), "stocked_at"),
		"a stocked line records when it was put away")
}

func TestReceivingOrders_StockCompletesTheOrderWhenEveryLineIsStocked(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := receivedPurchaseOrderReceiving(t)
	lineID := jsonField(firstLine(t, receivingOrderID), "id")

	status, body := stockReceivingOrder(t, receivingOrderID, []map[string]any{{
		"receiving_order_line_id": lineID,
		"allocations":             []map[string]any{{"quantity": pairs("4"), "location_id": SeedLocationID}},
	}})
	requireStatus(t, 200, status, body)

	assert.NotEmpty(t, jsonField(parseJSON(body), "completed_at"),
		"stocking the last outstanding line completes the order")
}

// Each allocation becomes its own inventory receipt, so a line split across two locations produces
// two delivery lines rather than one summed row.
func TestReceivingOrders_StockSplitsAcrossLocationsIntoSeparateDeliveryLines(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := receivedPurchaseOrderReceiving(t)
	lineID := jsonField(firstLine(t, receivingOrderID), "id")

	status, body := stockReceivingOrder(t, receivingOrderID, []map[string]any{{
		"receiving_order_line_id": lineID,
		"allocations": []map[string]any{
			{"quantity": pairs("1"), "location_id": SeedLocationID},
			{"quantity": pairs("3"), "location_id": SeedLocationID},
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

	_, receivingOrderID := receivedPurchaseOrderReceiving(t)
	lineID := jsonField(firstLine(t, receivingOrderID), "id")

	status, body := stockReceivingOrder(t, receivingOrderID, []map[string]any{{
		"receiving_order_line_id": lineID,
		"allocations":             []map[string]any{{"quantity": pairs("4")}},
	}})
	requireStatus(t, 200, status, body)

	deliveryLines := jsonListData(deliveryForReceivingOrder(t, receivingOrderID), "lines")
	require.Len(t, deliveryLines, 1)
	line, ok := deliveryLines[0].(map[string]any)
	require.True(t, ok)
	assertNilField(t, line, "location")
}

// A line orders a product, or names the item it restocks. Either way the stock lands on an item: the
// product's, or the one named. Each case orders a fresh one, so its on-hand and change log hold only
// this stocking.
func TestReceivingOrders_StockingRestocksTheLinesItem(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		line   func(t *testing.T) (line map[string]any, itemID string)
		onHand string
	}{
		"product": {
			line: func(t *testing.T) (map[string]any, string) {
				productID, itemID := newProductItemIDs(t, "e2e-ro-stock-product")
				line := purchaseOrderLineBody("E2E-RO-STOCK-PRODUCT")
				line["product_id"] = productID
				return line, itemID
			},
			onHand: "4",
		},
		"item": {
			line: func(t *testing.T) (map[string]any, string) {
				_, itemID := newReconcilableItem(t)
				line := materialLineBody("E2E-RO-STOCK-ITEM")
				line["item_id"] = itemID
				return line, itemID
			},
			onHand: "40",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			line, itemID := tc.line(t)
			quantity, ok := line["quantity"].(map[string]any)
			require.True(t, ok)
			_, receivingOrderID := receivedPurchaseOrderReceivingOf(t, func(b map[string]any) {
				b["lines"] = []map[string]any{line}
			})

			status, body := stockReceivingOrder(t, receivingOrderID, []map[string]any{{
				"receiving_order_line_id": jsonField(firstLine(t, receivingOrderID), "id"),
				"allocations":             []map[string]any{{"quantity": quantity, "location_id": SeedLocationID}},
			}})
			requireStatus(t, 200, status, body)

			// Creating the item logged its opening level as a user action; stocking is the system's.
			logs, status, err := apiClient.GetList(inventoryChangeLogsPath, url.Values{
				"item_ids": {itemID}, "action_types": {"system_action"}, "include": {"item"},
			})
			require.NoError(t, err)
			require.Equal(t, 200, status)
			require.Len(t, logs.Data, 1, "stocking logs one change, on the line's item")
			log := parseJSON(logs.Data[0])
			assert.Equal(t, itemID, jsonField(jsonObject(log, "item"), "id"))
			assertDecimalEqual(t, fmt.Sprint(quantity["value"]), jsonField(jsonObject(log, "quantity"), "value"))

			assertDecimalEqual(t, tc.onHand, readInventory(t, itemID).onHand.String(), "the put-away stock is on hand")
		})
	}
}

// A line that orders a product names no item of its own. Its receiving line and the delivery line it
// is stocked onto show the product's item, and each list's item filter finds them by it. The product
// is a fresh one, so the filtered lists hold only this order's documents.
func TestReceivingOrders_AProductOrderedLineShowsAndFiltersByTheProductsItem(t *testing.T) {
	t.Parallel()

	productID, itemID := newProductItemIDs(t, "e2e-ro-product-item")
	_, otherItemID := newProductItemIDs(t, "e2e-ro-product-other")
	line := purchaseOrderLineBody("E2E-RO-PRODUCT-ITEM")
	line["product_id"] = productID
	quantity, ok := line["quantity"].(map[string]any)
	require.True(t, ok)
	_, receivingOrderID := receivedPurchaseOrderReceivingOf(t, func(b map[string]any) {
		b["lines"] = []map[string]any{line}
	})
	itemOf := func(row any) string {
		t.Helper()
		m, ok := row.(map[string]any)
		require.True(t, ok)
		item := jsonObject(m, "item")
		require.NotNil(t, item, "the line's item must expand: %v", m)
		return jsonField(item, "id")
	}

	status, body, err := apiClient.GetListRaw(receivingOrdersPath+"/"+receivingOrderID, url.Values{"include": {"lines", "lines.item"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	lines := jsonListData(parseJSON(body), "lines")
	require.Len(t, lines, 1)
	assert.Equal(t, itemID, itemOf(lines[0]), "the receiving order's line receives the product's item")

	lineID := jsonField(lines[0].(map[string]any), "id")
	status, body, err = apiClient.Patch(receivingOrdersPath+"/"+receivingOrderID+"/lines/"+lineID+"?include=item",
		map[string]any{"quantity": quantity}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, itemID, itemOf(parseJSON(body)), "and so does the line read on its own")

	assert.Equal(t, []string{receivingOrderID}, listIDs(t, receivingOrdersPath, url.Values{"item_ids": {itemID}}),
		"the receiving order list's item filter finds the order by the product's item")
	assert.Empty(t, listIDs(t, receivingOrdersPath, url.Values{"item_ids": {otherItemID}}))

	status, body = stockReceivingOrder(t, receivingOrderID, []map[string]any{{
		"receiving_order_line_id": lineID,
		"allocations":             []map[string]any{{"quantity": quantity, "location_id": SeedLocationID}},
	}})
	requireStatus(t, 200, status, body)

	delivery := deliveryForReceivingOrder(t, receivingOrderID)
	deliveryLines := jsonListData(delivery, "lines")
	require.Len(t, deliveryLines, 1)
	assert.Equal(t, itemID, itemOf(deliveryLines[0]), "the delivery line received the product's item")

	assert.Equal(t, []string{jsonField(delivery, "id")}, listIDs(t, deliveriesPath, url.Values{"item_ids": {itemID}}),
		"the delivery list's item filter finds the delivery by the product's item")
	assert.Empty(t, listIDs(t, deliveriesPath, url.Values{"item_ids": {otherItemID}}))
}

// A lot number creates the lot on first use and applies to every allocation on the line.
func TestReceivingOrders_StockUnderALotRecordsItOnEveryDeliveryLine(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := receivedPurchaseOrderReceiving(t)
	lineID := jsonField(firstLine(t, receivingOrderID), "id")
	lotNumber := uniqueName("E2E-LOT")

	status, body := stockReceivingOrder(t, receivingOrderID, []map[string]any{{
		"receiving_order_line_id": lineID,
		"lot_number":              lotNumber,
		"allocations": []map[string]any{
			{"quantity": pairs("2"), "location_id": SeedLocationID},
			{"quantity": pairs("2"), "location_id": SeedLocationID},
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

	_, receivingOrderID := receivedPurchaseOrderReceiving(t)
	lineID := jsonField(firstLine(t, receivingOrderID), "id")

	status, body := stockReceivingOrder(t, receivingOrderID, []map[string]any{{
		"receiving_order_line_id": lineID,
		"rejected_quantity":       pairs("1"),
		"allocations":             []map[string]any{{"quantity": pairs("3"), "location_id": SeedLocationID}},
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
// not silently closed on a partial delivery. The new line opens at zero, as a line does when the
// order is issued: nothing is stocked against it until someone counts what arrives.
func TestReceivingOrders_StockingShortOpensARemainderLineAtZero(t *testing.T) {
	t.Parallel()

	purchaseOrderID, receivingOrderID := receivedPurchaseOrderReceiving(t)
	lineID := jsonField(firstLine(t, receivingOrderID), "id")

	patchStatus, patchBody := patchReceivingOrderLine(t, receivingOrderID, lineID, map[string]any{"quantity": pairs("1")})
	requireStatus(t, 200, patchStatus, patchBody)

	status, body := stockReceivingOrder(t, receivingOrderID, []map[string]any{{
		"receiving_order_line_id": lineID,
		"allocations":             []map[string]any{{"quantity": pairs("1"), "location_id": SeedLocationID}},
	}})
	requireStatus(t, 200, status, body)

	lines := receivingOrderLines(t, receivingOrderID)
	require.Len(t, lines, 2, "the outstanding 3 is expected on a new line: %v", lines)

	var remainder map[string]any
	for _, raw := range lines {
		line, ok := raw.(map[string]any)
		require.True(t, ok)
		if line["stocked_at"] == nil {
			require.Nil(t, remainder, "exactly one line is still expected")
			remainder = line
		}
	}
	require.NotNil(t, remainder)
	assertDecimalEqual(t, "0", lineQuantityValue(t, remainder), "the remainder line opens at zero, not at the outstanding 3")

	// Stocking again before anything is counted puts nothing away: the order stays open and the
	// purchase order unfulfilled. A pre-filled remainder would have been booked here sight unseen.
	againStatus, againBody := stockReceivingOrder(t, receivingOrderID, []map[string]any{})
	requireStatus(t, 200, againStatus, againBody)
	assertNilField(t, parseJSON(againBody), "completed_at")

	poStatus, poBody, err := apiClient.GetListRaw(purchaseOrdersPath+"/"+purchaseOrderID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, poStatus, poBody)
	assert.NotEqual(t, "fulfilled", jsonField(parseJSON(poBody), "status"), "nothing more arrived, so the purchase order is not fulfilled")
}

// Receiving the remainder finishes it at what is still outstanding: the ordered 4 less the 1 already
// stocked.
func TestReceivingOrders_ReceivingTheRemainderRecordsOnlyWhatIsOutstanding(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := receivedPurchaseOrderReceiving(t)
	lineID := jsonField(firstLine(t, receivingOrderID), "id")

	patchStatus, patchBody := patchReceivingOrderLine(t, receivingOrderID, lineID, map[string]any{"quantity": pairs("1")})
	requireStatus(t, 200, patchStatus, patchBody)
	stockStatus, stockBody := stockReceivingOrder(t, receivingOrderID, []map[string]any{{
		"receiving_order_line_id": lineID,
		"allocations":             []map[string]any{{"quantity": pairs("1"), "location_id": SeedLocationID}},
	}})
	requireStatus(t, 200, stockStatus, stockBody)

	status, body, err := apiClient.Put(receivingOrdersPath+"/"+receivingOrderID+"/actions/receive", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	for _, raw := range receivingOrderLines(t, receivingOrderID) {
		line, ok := raw.(map[string]any)
		require.True(t, ok)
		if line["stocked_at"] == nil {
			assertDecimalEqual(t, "3", lineQuantityValue(t, line), "the remainder takes the outstanding 3")
		}
	}
}

// Finishing a partly counted line tops it up to the order, rather than setting it to what was still
// outstanding beside it (Express turned a line counted at 1 of 4 into 3).
func TestReceivingOrderLines_ReceiveFinishesAPartlyCountedLine(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := issuedPurchaseOrderReceiving(t)
	lineID := jsonField(firstLine(t, receivingOrderID), "id")

	patchStatus, patchBody := patchReceivingOrderLine(t, receivingOrderID, lineID, map[string]any{"quantity": pairs("1")})
	requireStatus(t, 200, patchStatus, patchBody)

	status, body, err := apiClient.Put(receivingOrdersPath+"/"+receivingOrderID+"/lines/"+lineID+"/actions/receive", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	assertDecimalEqual(t, "4", lineQuantityValue(t, firstLine(t, receivingOrderID)), "the line now holds the whole order")
}

// Stocking an order with nothing left to put away is a no-op rather than a second delivery.
func TestReceivingOrders_StockingAnAlreadyStockedOrderRecordsNothingNew(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := receivedPurchaseOrderReceiving(t)
	lineID := jsonField(firstLine(t, receivingOrderID), "id")

	status, body := stockReceivingOrder(t, receivingOrderID, []map[string]any{{
		"receiving_order_line_id": lineID,
		"allocations":             []map[string]any{{"quantity": pairs("4"), "location_id": SeedLocationID}},
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

	_, receivingOrderID := receivedPurchaseOrderReceiving(t)

	status, body := stockReceivingOrder(t, receivingOrderID, []map[string]any{})
	requireStatus(t, 200, status, body)

	assert.NotEmpty(t, jsonField(firstLine(t, receivingOrderID), "stocked_at"),
		"an omitted line is still marked stocked")
}

// --- Stocking validation ---

func TestReceivingOrders_StockRefusesAnAllocationForAnotherOrdersLine(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := receivedPurchaseOrderReceiving(t)
	_, otherOrderID := receivedPurchaseOrderReceiving(t)
	foreignLineID := jsonField(firstLine(t, otherOrderID), "id")

	status, body := stockReceivingOrder(t, receivingOrderID, []map[string]any{{
		"receiving_order_line_id": foreignLineID,
		"allocations":             []map[string]any{{"quantity": pairs("1"), "location_id": SeedLocationID}},
	}})
	assert.Equal(t, 400, status, "another order's line cannot be stocked through this one: %s", string(body))

	assert.Empty(t, jsonField(firstLine(t, receivingOrderID), "stocked_at"), "and nothing was stocked")
	assertDecimalEqual(t, "4", lineQuantityValue(t, firstLine(t, otherOrderID)),
		"the other order's line was not touched through this one")
}

// A stale line id used to be ignored while the order's own lines were swept as stocked, closing an
// order the client did not mean to close. It is refused now, and nothing is stocked.
func TestReceivingOrders_StockRefusesAnUnknownLine(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := receivedPurchaseOrderReceiving(t)

	status, body := stockReceivingOrder(t, receivingOrderID, []map[string]any{{
		"receiving_order_line_id": "rcln_doesnotexist00",
		"allocations":             []map[string]any{{"quantity": pairs("1")}},
	}})
	assert.Equal(t, 400, status, "an unknown line is refused: %s", string(body))
	assert.Empty(t, jsonField(firstLine(t, receivingOrderID), "stocked_at"), "and nothing was stocked")
}

func TestReceivingOrders_StockRefusesAMalformedQuantity(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := receivedPurchaseOrderReceiving(t)
	lineID := jsonField(firstLine(t, receivingOrderID), "id")

	for _, quantity := range []string{"not-a-number", ""} {
		t.Run(fmt.Sprintf("quantity=%q", quantity), func(t *testing.T) {
			status, body := stockReceivingOrder(t, receivingOrderID, []map[string]any{{
				"receiving_order_line_id": lineID,
				"allocations":             []map[string]any{{"quantity": pairs(quantity)}},
			}})
			assert.Equal(t, 400, status, "an unparseable quantity is refused rather than stocked as zero: %s", string(body))
		})
	}
	assert.Empty(t, jsonField(firstLine(t, receivingOrderID), "stocked_at"), "and nothing was stocked")
}

func TestReceivingOrders_StockRefusesMoreThanWasReceived(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := receivedPurchaseOrderReceiving(t)
	lineID := jsonField(firstLine(t, receivingOrderID), "id")

	status, body := stockReceivingOrder(t, receivingOrderID, []map[string]any{{
		"receiving_order_line_id": lineID,
		"rejected_quantity":       pairs("1"),
		"allocations":             []map[string]any{{"quantity": pairs("4"), "location_id": SeedLocationID}},
	}})
	assert.Equal(t, 400, status, "4 put away and 1 refused is more than the 4 received: %s", string(body))
}

func TestReceivingOrders_StockRefusesALineTwice(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := receivedPurchaseOrderReceiving(t)
	lineID := jsonField(firstLine(t, receivingOrderID), "id")

	item := map[string]any{
		"receiving_order_line_id": lineID,
		"allocations":             []map[string]any{{"quantity": pairs("2"), "location_id": SeedLocationID}},
	}
	status, body := stockReceivingOrder(t, receivingOrderID, []map[string]any{item, item})
	assert.Equal(t, 400, status, "a line is stocked once per request: %s", string(body))
}

// The stocking dialog offers every unit of the item's group. Here the delivery was counted as a
// dozen and is put away in pairs: the check against what was received converts first, so 6 pairs
// is the dozen and 7 is more, and the receipt is recorded in the pairs it was put away in.
func TestReceivingOrders_StockInAnotherUnitOfTheGroup(t *testing.T) {
	t.Parallel()

	countedInDozens := func(t *testing.T) (receivingOrderID, lineID string) {
		t.Helper()
		_, receivingOrderID = issuedPurchaseOrderReceiving(t)
		lineID = jsonField(firstLine(t, receivingOrderID), "id")
		status, body := patchReceivingOrderLine(t, receivingOrderID, lineID, map[string]any{"quantity": dozens("1")})
		requireStatus(t, 200, status, body)
		return receivingOrderID, lineID
	}

	overID, overLineID := countedInDozens(t)
	overStatus, overBody := stockReceivingOrder(t, overID, []map[string]any{{
		"receiving_order_line_id": overLineID,
		"allocations":             []map[string]any{{"quantity": pairs("7"), "location_id": SeedLocationID}},
	}})
	assert.Equal(t, 400, overStatus, "7 pairs is more than the dozen received: %s", string(overBody))

	receivingOrderID, lineID := countedInDozens(t)
	status, body := stockReceivingOrder(t, receivingOrderID, []map[string]any{{
		"receiving_order_line_id": lineID,
		"allocations":             []map[string]any{{"quantity": pairs("6"), "location_id": SeedLocationID}},
	}})
	requireStatus(t, 200, status, body)
	assert.NotEmpty(t, jsonField(parseJSON(body), "completed_at"), "a dozen covers the 4 pairs ordered, so the order is complete")

	deliveryLines := jsonListData(deliveryForReceivingOrder(t, receivingOrderID), "lines")
	require.Len(t, deliveryLines, 1)
	deliveryLine, ok := deliveryLines[0].(map[string]any)
	require.True(t, ok)
	quantity := jsonObject(deliveryLine, "quantity")
	require.NotNil(t, quantity, "a delivery line carries its quantity: %v", deliveryLine)
	assertDecimalEqual(t, "6", jsonField(quantity, "value"), "recorded as put away, in pairs")
}

// --- Action responses ---

// The receiving screen refreshes itself from what an action returns, so every action accepts the
// retrieve endpoint's includes and fills them.
func TestReceivingOrders_ActionsReturnTheRequestedIncludes(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := issuedPurchaseOrderReceiving(t)
	include := url.Values{"include": {"lines", "lines.order_line", "lines.quantity", "lines.quantity.unit", "supplier", "totals"}}

	status, body, err := apiClient.Put(receivingOrdersPath+"/"+receivingOrderID+"/actions/receive?"+include.Encode(), nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	order := parseJSON(body)
	require.NotNil(t, jsonObject(order, "supplier"), "the supplier expands on an action: %s", string(body))
	require.NotNil(t, jsonObject(order, "totals"), "and so do the totals: %s", string(body))
	lines := jsonListData(order, "lines")
	require.Len(t, lines, 1, "and the lines: %s", string(body))
	line, ok := lines[0].(map[string]any)
	require.True(t, ok)
	require.NotNil(t, jsonObject(line, "order_line"), "each line's order line expands: %v", line)
	assert.Equal(t, SeedUnitID, lineQuantityUnitID(t, line))

	lineID := jsonField(line, "id")
	lineStatus, lineBody, err := apiClient.Put(
		receivingOrdersPath+"/"+receivingOrderID+"/lines/"+lineID+"/actions/void?"+url.Values{"include": {"order_line", "quantity.unit"}}.Encode(), nil)
	require.NoError(t, err)
	requireStatus(t, 200, lineStatus, lineBody)
	require.NotNil(t, jsonObject(parseJSON(lineBody), "order_line"), "a line action fills its includes too: %s", string(lineBody))
}

// --- List search ---

// The dashboard searches receiving orders by supplier as well as by number. A supplier that is also
// a customer of the account must not appear twice: the relation is joined in its supplier role only.
func TestReceivingOrders_ListSearchFindsAnOrderBySupplierName(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := issuedPurchaseOrderReceiving(t)

	status, body, err := apiClient.GetListRaw(receivingOrdersPath+"/"+receivingOrderID, url.Values{"include": {"supplier"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	supplierName := jsonField(jsonObject(parseJSON(body), "supplier"), "name")
	require.NotEmpty(t, supplierName)

	seen := map[string]int{}
	found := false
	listStatus, listBody, err := apiClient.GetListRaw(receivingOrdersPath, url.Values{"q": {supplierName}, "limit": {"100"}})
	for range 50 {
		require.NoError(t, err)
		requireStatus(t, 200, listStatus, listBody)
		for _, raw := range jsonArray(parseJSON(listBody), "data") {
			order, ok := raw.(map[string]any)
			require.True(t, ok)
			id := jsonField(order, "id")
			seen[id]++
			found = found || id == receivingOrderID
		}
		info := receivingOrdersPageInfo(t, listBody)
		if found || !info.HasNextPage {
			break
		}
		listStatus, listBody, err = apiClient.GetListRawFromPageURL(info.NextPageURL)
	}
	assert.True(t, found, "searching for the supplier's name finds its open receiving order")
	for id, n := range seen {
		assert.Equal(t, 1, n, "receiving order %s is listed once, not once per relation", id)
	}
}

// A purchase order line can name a product instead of an item. What is stocked is that product's
// item, so the order line and the delivery line both carry it.
func TestReceivingOrders_StockingAProductLineRecordsTheProductsItem(t *testing.T) {
	t.Parallel()

	purchaseOrderID, receivingOrderID := receivedPurchaseOrderReceiving(t)
	lineID := jsonField(firstLine(t, receivingOrderID), "id")

	status, body := stockReceivingOrder(t, receivingOrderID, []map[string]any{{
		"receiving_order_line_id": lineID,
		"allocations":             []map[string]any{{"quantity": pairs("4"), "location_id": SeedLocationID}},
	}})
	requireStatus(t, 200, status, body)

	lines := jsonListData(deliveryForReceivingOrder(t, receivingOrderID), "lines")
	require.NotEmpty(t, lines)
	for _, raw := range lines {
		line, ok := raw.(map[string]any)
		require.True(t, ok)
		item := jsonObject(line, "item")
		require.NotNil(t, item, "a delivery line bought as a product carries the product's item: %v", line)
		assert.Equal(t, SeedItemID, jsonField(item, "id"))
	}

	getStatus, getBody, err := apiClient.GetListRaw(purchaseOrdersPath+"/"+purchaseOrderID, url.Values{"include": {"lines", "lines.item"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	orderLines := jsonListData(parseJSON(getBody), "lines")
	require.NotEmpty(t, orderLines)
	orderLine, ok := orderLines[0].(map[string]any)
	require.True(t, ok)
	orderItem := jsonObject(orderLine, "item")
	require.NotNil(t, orderItem, "a purchase order line bought as a product carries the product's item")
	assert.Equal(t, SeedItemID, jsonField(orderItem, "id"))
}
