//go:build e2e

package api_test

import (
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A receiving order's totals: what it was ordered for, and how far it has been put away.
//
// The seeded purchase order line is 4 units at 9.50, so every order here is ordered for 38.00 and
// each unit stocked moves completion by a quarter. Those numbers are what the assertions below are
// written against.

const (
	e2eOrderedAmount     = "38"
	e2eLineUnitPrice     = 9.50
	e2eLineOrderedUnits  = 4
	e2eCompletionEpsilon = 0.0001
)

// receivingOrderTotals reads an order's totals, which are expandable and absent unless asked for.
func receivingOrderTotals(t *testing.T, receivingOrderID string) map[string]any {
	t.Helper()

	status, body, err := apiClient.GetListRaw(
		receivingOrdersPath+"/"+receivingOrderID,
		url.Values{"include": {"totals"}},
	)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	totals := jsonObject(parseJSON(body), "totals")
	require.NotNil(t, totals, "totals must expand when asked for: %s", string(body))
	return totals
}

func stageCompletion(t *testing.T, totals map[string]any, stage string) float64 {
	t.Helper()

	stageTotal := jsonObject(totals, stage)
	require.NotNil(t, stageTotal, "totals carry a %s stage: %v", stage, totals)

	completion, ok := stageTotal["completion"].(float64)
	require.True(t, ok, "%s completion is a number: %v", stage, stageTotal)
	return completion
}

func stageAmount(t *testing.T, totals map[string]any, stage string) string {
	t.Helper()

	stageTotal := jsonObject(totals, stage)
	require.NotNil(t, stageTotal, "totals carry a %s stage: %v", stage, totals)
	return jsonField(stageTotal, "amount")
}

// setReceivingLineQuantity narrows a receiving line to the quantity that actually turned up, which
// is what an operator does before putting away a short delivery.
func setReceivingLineQuantity(t *testing.T, receivingOrderID, lineID, value string) {
	t.Helper()

	status, body, err := apiClient.Patch(
		receivingOrdersPath+"/"+receivingOrderID+"/lines/"+lineID,
		map[string]any{"quantity_value": value},
		newIdempotencyKey(),
	)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
}

// touchPurchaseOrderLine edits the purchase order line, which is what raises a second receiving line
// for whatever is still outstanding once the first one has been stocked.
func touchPurchaseOrderLine(t *testing.T, purchaseOrderID string) {
	t.Helper()

	status, body, err := apiClient.GetListRaw(
		purchaseOrdersPath+"/"+purchaseOrderID,
		url.Values{"include": {"lines"}},
	)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	lines := jsonListData(parseJSON(body), "lines")
	require.NotEmpty(t, lines, "a purchase order has at least one line: %s", string(body))
	line, ok := lines[0].(map[string]any)
	require.True(t, ok)

	patchStatus, patchBody, err := apiClient.Patch(
		purchaseOrdersPath+"/"+purchaseOrderID+"/lines/"+jsonField(line, "id"),
		map[string]any{"product_description": uniqueName("E2E remaining")},
		newIdempotencyKey(),
	)
	require.NoError(t, err)
	require.Less(t, patchStatus, 500, "updating a purchase order line must not 5xx: %s", string(patchBody))
	requireStatus(t, 200, patchStatus, patchBody)
}

// --- Totals ---

func TestReceivingOrders_TotalsReportTheOrderedValueBeforeAnythingIsStocked(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := issuedPurchaseOrderReceiving(t)

	totals := receivingOrderTotals(t, receivingOrderID)
	assertDecimalEqual(t, e2eOrderedAmount, jsonField(totals, "ordered"),
		"4 units at 9.50 is an order worth 38.00")
	assert.InDelta(t, 0, stageCompletion(t, totals, "stocked"), e2eCompletionEpsilon,
		"nothing is put away yet")
	assert.InDelta(t, 0, stageCompletion(t, totals, "rejected"), e2eCompletionEpsilon,
		"and nothing was refused")
}

func TestReceivingOrders_TotalsReachFullCompletionWhenEverythingIsStocked(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := issuedPurchaseOrderReceiving(t)
	lineID := jsonField(firstLine(t, receivingOrderID), "id")

	status, body := stockReceivingOrder(t, receivingOrderID, []map[string]any{{
		"receiving_order_line_id": lineID,
		"allocations":             []map[string]any{{"quantity": "4", "location_id": SeedLocationID}},
	}})
	requireStatus(t, 200, status, body)

	totals := receivingOrderTotals(t, receivingOrderID)
	assertDecimalEqual(t, e2eOrderedAmount, jsonField(totals, "ordered"),
		"stocking does not change what was ordered")
	assertDecimalEqual(t, e2eOrderedAmount, stageAmount(t, totals, "stocked"),
		"the whole order is put away, so the stocked amount is the ordered one")
	assert.InDelta(t, 1, stageCompletion(t, totals, "stocked"), e2eCompletionEpsilon,
		"an order put away in full is complete")
}

// The bug this pins: a purchase order line received in installments carries one receiving line per
// installment, and the ordered figure is a property of the purchase order line, not of the receiving
// lines raised against it. Summed once per receiving line, a line received in two goes reports twice
// what was ordered — and completion, which divides by it, reports half of the truth. An order that
// is fully received then reads as half done.
func TestReceivingOrders_TotalsCountTheOrderedValueOncePerPurchaseOrderLine(t *testing.T) {
	t.Parallel()

	purchaseOrderID, receivingOrderID := issuedPurchaseOrderReceiving(t)

	// One unit of four turns up, so the line is narrowed and put away.
	firstLineID := jsonField(firstLine(t, receivingOrderID), "id")
	setReceivingLineQuantity(t, receivingOrderID, firstLineID, "1")
	status, body := stockReceivingOrder(t, receivingOrderID, []map[string]any{{
		"receiving_order_line_id": firstLineID,
		"allocations":             []map[string]any{{"quantity": "1", "location_id": SeedLocationID}},
	}})
	requireStatus(t, 200, status, body)

	// Editing the purchase order line raises a second receiving line for the outstanding three.
	touchPurchaseOrderLine(t, purchaseOrderID)

	lines := receivingOrderLines(t, receivingOrderID)
	require.Len(t, lines, 2,
		"a partially stocked line gets a second receiving line for the remainder: %v", lines)

	totals := receivingOrderTotals(t, receivingOrderID)
	assertDecimalEqual(t, e2eOrderedAmount, jsonField(totals, "ordered"),
		"two receiving lines against one purchase order line is still one order for 38.00, not two")
	assert.InDelta(t, 0.25, stageCompletion(t, totals, "stocked"), e2eCompletionEpsilon,
		"one unit of four is a quarter of the order, whatever it took to receive it")

	// Putting the remainder away completes the order rather than leaving it at half.
	var remainingID string
	for _, raw := range lines {
		line, ok := raw.(map[string]any)
		require.True(t, ok)
		if line["stocked_at"] == nil {
			remainingID = jsonField(line, "id")
		}
	}
	require.NotEmpty(t, remainingID, "one of the two lines is still outstanding: %v", lines)

	secondStatus, secondBody := stockReceivingOrder(t, receivingOrderID, []map[string]any{{
		"receiving_order_line_id": remainingID,
		"allocations":             []map[string]any{{"quantity": "3", "location_id": SeedLocationID}},
	}})
	requireStatus(t, 200, secondStatus, secondBody)

	finalTotals := receivingOrderTotals(t, receivingOrderID)
	assertDecimalEqual(t, e2eOrderedAmount, jsonField(finalTotals, "ordered"),
		"and the ordered value never grew with the receipts")
	assert.InDelta(t, 1, stageCompletion(t, finalTotals, "stocked"), e2eCompletionEpsilon,
		"an order received in two installments is complete, not half complete")
}

// Completion divides amounts rather than quantities. A receiving order's lines can each count in a
// different unit, so summing their quantities would add pairs to metres; the agreed unit price is
// the only thing that makes the stages comparable.
func TestReceivingOrders_StockedCompletionIsTheStockedShareOfTheOrderedAmount(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := issuedPurchaseOrderReceiving(t)
	lineID := jsonField(firstLine(t, receivingOrderID), "id")

	setReceivingLineQuantity(t, receivingOrderID, lineID, "1")
	status, body := stockReceivingOrder(t, receivingOrderID, []map[string]any{{
		"receiving_order_line_id": lineID,
		"allocations":             []map[string]any{{"quantity": "1", "location_id": SeedLocationID}},
	}})
	requireStatus(t, 200, status, body)

	totals := receivingOrderTotals(t, receivingOrderID)

	stocked := stageAmount(t, totals, "stocked")
	assertDecimalEqual(t, strconv.FormatFloat(e2eLineUnitPrice, 'f', -1, 64), stocked,
		"one unit at 9.50 is 9.50 put away")
	assert.InDelta(t, 1.0/e2eLineOrderedUnits, stageCompletion(t, totals, "stocked"), e2eCompletionEpsilon,
		"9.50 of an order for 38.00 is a quarter complete")
}

// --- Expandable ---

func TestReceivingOrders_TotalsAreAbsentWithoutTheInclude(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := issuedPurchaseOrderReceiving(t)

	status, body, err := apiClient.GetListRaw(receivingOrdersPath+"/"+receivingOrderID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	assertNilField(t, parseJSON(body), "totals")
}

// --- Pagination ---

// Totals are attached to a page of orders in one aggregate after the rows are read, which is a step
// each of the three cursor branches has to take for itself. Paging backwards must not be the branch
// that forgets: an order carrying its totals forwards and null backwards is the same order reported
// two different ways depending on which direction the client walked.
//
// The order is one this test issues, because a seeded order with no lines has no totals in either
// direction and would pass the assertion without testing anything.
func TestReceivingOrders_TotalsSurviveBackwardPagination(t *testing.T) {
	t.Parallel()

	_, receivingOrderID := issuedPurchaseOrderReceiving(t)

	forward, forwardBody := listReceivingOrdersPage(t, url.Values{
		"limit":   {"25"},
		"include": {"totals"},
	})
	forwardOrder := findReceivingOrder(forward, receivingOrderID)
	require.NotNil(t, forwardOrder,
		"the order just issued is on the newest page: %s", string(forwardBody))
	require.NotNil(t, forwardOrder["totals"],
		"an order with a line has totals when they are asked for: %s", string(forwardBody))

	pageInfo := jsonObject(parseJSON(forwardBody), "page_info")
	require.NotNil(t, pageInfo, "a list response carries page info: %s", string(forwardBody))
	if jsonField(pageInfo, "next_cursor") == "" {
		t.Skip("only one page of receiving orders; there is no cursor to walk back from")
	}

	_, secondBody := listReceivingOrdersPage(t, url.Values{
		"limit":          {"25"},
		"include":        {"totals"},
		"starting_after": {jsonField(pageInfo, "next_cursor")},
	})
	secondPageInfo := jsonObject(parseJSON(secondBody), "page_info")
	require.NotNil(t, secondPageInfo)
	backCursor := jsonField(secondPageInfo, "previous_cursor")
	require.NotEmpty(t, backCursor, "the second page can be walked back from: %s", string(secondBody))

	backward, backwardBody := listReceivingOrdersPage(t, url.Values{
		"limit":         {"25"},
		"include":       {"totals"},
		"ending_before": {backCursor},
	})
	backwardOrder := findReceivingOrder(backward, receivingOrderID)
	require.NotNil(t, backwardOrder,
		"walking back returns the page the order is on: %s", string(backwardBody))

	assert.Equal(t, forwardOrder["totals"], backwardOrder["totals"],
		"the same order reports the same totals whichever direction it was paged: %s", string(backwardBody))
}

// findReceivingOrder picks one order out of a page by id, or nil when the page does not hold it.
func findReceivingOrder(page []any, receivingOrderID string) map[string]any {
	for _, raw := range page {
		order, ok := raw.(map[string]any)
		if ok && jsonField(order, "id") == receivingOrderID {
			return order
		}
	}
	return nil
}

func listReceivingOrdersPage(t *testing.T, query url.Values) ([]any, []byte) {
	t.Helper()

	status, body, err := apiClient.GetListRaw(receivingOrdersPath, query)
	require.NoError(t, err)
	require.Less(t, status, 500, "listing receiving orders must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)
	return jsonArray(parseJSON(body), "data"), body
}
