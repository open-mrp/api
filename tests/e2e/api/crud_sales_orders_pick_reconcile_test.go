//go:build e2e

package api_test

import (
	"net/http"
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Pin the legacy Dashboard pick-line reconciliation semantics on a sales-order line
// quantity change. In the legacy Express API, changing an order line's ordered quantity
// (or adding a line) to an order that already has a pick would create a fresh pick line
// for any quantity not yet covered by existing pick lines — so a line that was fully
// picked/packed and then bumped up gets a new open pick line for the delta, rather than
// silently under-fulfilling. These tests exercise the real issue -> pick -> pack ->
// re-quantify flow end to end and, per the project rule, fail loudly on any 5xx.
//
// The reconciliation lives in core-service's createPickLineForRemainingQuantity, invoked
// from both CreateSalesOrderLine and UpdateSalesOrderLine. Its guards, mirrored from
// legacy order-line.repo.ts / pick-line.repo.ts:
//   - remaining = ordered - SUM(all pick lines for the order line); skip when <= 0.
//   - skip when an unpacked ("open") pick line already exists (one open line per order line).
//   - a fresh pick line is seeded at 0 picked (the delta is what remains to pick, not
//     what has been picked).
//   - on UPDATE the reconciliation is gated to sale-type product lines; freight/credit
//     ("system") lines are never picked. (On CREATE the legacy code is ungated; we do not
//     assert the create-of-a-system-line case here because it is not a real workflow.)

// orderSaleLineID returns the id of the order's sale product line (the line whose product
// is SeedProductID). The synthesized shipping line is skipped.
func orderSaleLineID(t *testing.T, orderID string) string {
	t.Helper()
	return orderLineIDMatching(t, orderID, func(product map[string]any) bool {
		return jsonField(product, "id") == SeedProductID
	})
}

// orderSystemLineID returns the id of the order's synthesized shipping/credit line (the
// line whose product is NOT SeedProductID).
func orderSystemLineID(t *testing.T, orderID string) string {
	t.Helper()
	return orderLineIDMatching(t, orderID, func(product map[string]any) bool {
		return jsonField(product, "id") != SeedProductID
	})
}

func orderLineIDMatching(t *testing.T, orderID string, match func(product map[string]any) bool) string {
	t.Helper()
	got := getSalesOrder(t, orderID, url.Values{"include": {"lines", "lines.product"}})
	lines := jsonObject(got, "lines")
	require.NotNil(t, lines, "lines present with ?include=lines")
	for _, raw := range jsonArray(lines, "data") {
		line, ok := raw.(map[string]any)
		require.True(t, ok)
		product := jsonObject(line, "product")
		if product != nil && match(product) {
			return jsonField(line, "id")
		}
	}
	t.Fatalf("no matching order line found on %s", orderID)
	return ""
}

// patchSaleLineQuantity sets the sale line's ordered quantity and requires a 200.
func patchSaleLineQuantity(t *testing.T, orderID, lineID, value string) {
	t.Helper()
	status, body, err := apiClient.Patch(salesOrdersPath+"/"+orderID+"/lines/"+lineID,
		map[string]any{"quantity": map[string]any{"value": value, "unit_id": SeedUnitID}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
}

// orderPickID issues the order (creating a pick) is done separately; this reads the pick id
// off the issued order.
func orderPickID(t *testing.T, orderID string) string {
	t.Helper()
	withPick := getSalesOrder(t, orderID, url.Values{"include": {"related.pick"}})
	related := jsonObject(withPick, "related")
	require.NotNil(t, related, "related present with ?include=related.pick")
	pick := jsonObject(related, "pick")
	require.NotNil(t, pick, "issuing an order creates a pick")
	pickID := jsonField(pick, "id")
	require.NotEmpty(t, pickID)
	return pickID
}

// pickLineRow is a decoded pick line: its picked quantity, its originating order-line
// ordered quantity, and whether it has been packed.
type pickLineRow struct {
	picked  float64
	ordered float64
	packed  bool
}

// fetchPickLines reads all pick lines for a pick.
func fetchPickLines(t *testing.T, pickID string) []pickLineRow {
	t.Helper()
	status, body, err := apiClient.GetListRaw("/v1/operations/picks/"+pickID, url.Values{"include": {"lines"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	lines := jsonObject(parseJSON(body), "lines")
	require.NotNil(t, lines, "pick lines present with ?include=lines")

	var rows []pickLineRow
	for _, raw := range jsonArray(lines, "data") {
		line, ok := raw.(map[string]any)
		require.True(t, ok)

		row := pickLineRow{packed: line["packed_at"] != nil}
		if qty := jsonObject(line, "quantity"); qty != nil {
			row.picked, _ = strconv.ParseFloat(jsonField(qty, "value"), 64)
		}
		if ordered := jsonObject(line, "ordered_quantity"); ordered != nil {
			row.ordered, _ = strconv.ParseFloat(jsonField(ordered, "value"), 64)
		}
		rows = append(rows, row)
	}
	return rows
}

// pickAllLines picks every unpacked line up to its ordered quantity.
func pickAllLines(t *testing.T, pickID string) {
	t.Helper()
	status, body, err := apiClient.Do(http.MethodPut, "/v1/operations/picks/"+pickID+"/actions/pick",
		map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
}

// Packs the pick into a shipment with a single case, waiting for the job so callers can read
// the writes it made.
func packPick(t *testing.T, pickID string) {
	t.Helper()
	acceptPackPick(t, pickID, 1)
}

// Posts a pack, requires the 202, follows the job to completion and returns it.
func acceptPackPick(t *testing.T, pickID string, caseCount int) map[string]any {
	t.Helper()

	status, body, err := apiClient.Post("/v1/operations/picks/"+pickID+"/actions/pack",
		map[string]any{"shipment_case_count": caseCount}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 202, status, body)

	accepted := parseJSON(body)
	require.Equal(t, "job", jsonField(accepted, "object"), "202 returns the canonical job resource")
	jobID := jsonField(accepted, "id")
	require.NotEmpty(t, jobID, "the 202 must name the job to poll")

	job := pollJobUntilTerminal(t, jobID)
	require.Equal(t, "completed", jsonField(job, "status"), "the pack job should complete: %v", job)
	return job
}

// firstPickLineID returns the id of the pick's single (or first) pick line.
func firstPickLineID(t *testing.T, pickID string) string {
	t.Helper()
	status, body, err := apiClient.GetListRaw("/v1/operations/picks/"+pickID, url.Values{"include": {"lines"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	lines := jsonObject(parseJSON(body), "lines")
	require.NotNil(t, lines)
	data := jsonArray(lines, "data")
	require.NotEmpty(t, data, "the pick has at least one line")
	line, ok := data[0].(map[string]any)
	require.True(t, ok)
	id := jsonField(line, "id")
	require.NotEmpty(t, id)
	return id
}

// setPickedQuantity sets a pick line's picked quantity (a partial pick) and requires a 200.
func setPickedQuantity(t *testing.T, pickID, pickLineID, value string) {
	t.Helper()
	status, body, err := apiClient.Patch("/v1/operations/picks/"+pickID+"/lines/"+pickLineID,
		map[string]any{"quantity_value": value}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
}

// pickLineObjects returns the raw pick line JSON objects for a pick.
func pickLineObjects(t *testing.T, pickID string) []map[string]any {
	t.Helper()
	status, body, err := apiClient.GetListRaw("/v1/operations/picks/"+pickID, url.Values{"include": {"lines"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	lines := jsonObject(parseJSON(body), "lines")
	require.NotNil(t, lines)
	var out []map[string]any
	for _, raw := range jsonArray(lines, "data") {
		if obj, ok := raw.(map[string]any); ok {
			out = append(out, obj)
		}
	}
	return out
}

// pickLineIDByOrdered returns the id of the pick line whose ordered_quantity matches want.
func pickLineIDByOrdered(t *testing.T, pickID string, want float64) string {
	t.Helper()
	for _, line := range pickLineObjects(t, pickID) {
		if ordered := jsonObject(line, "ordered_quantity"); ordered != nil {
			v, _ := strconv.ParseFloat(jsonField(ordered, "value"), 64)
			if v == want {
				return jsonField(line, "id")
			}
		}
	}
	t.Fatalf("no pick line with ordered_quantity %v on pick %s", want, pickID)
	return ""
}

// pickIsFinished reports whether the pick's finished_at is set (the pick reads as closed).
func pickIsFinished(t *testing.T, pickID string) bool {
	t.Helper()
	status, body, err := apiClient.GetListRaw("/v1/operations/picks/"+pickID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return parseJSON(body)["finished_at"] != nil
}

// countPacked splits pick lines into packed and unpacked buckets.
func countPacked(rows []pickLineRow) (packed, unpacked int) {
	for _, r := range rows {
		if r.packed {
			packed++
		} else {
			unpacked++
		}
	}
	return packed, unpacked
}

// TestSalesOrder_LineQuantityIncrease_AfterFullPickPack_CreatesRemainderPickLine is the
// primary scenario: a line ordered 10, fully picked and packed, then bumped to 20. The
// original pick line stays packed at 10 and a NEW open pick line is created for the
// remaining 10 (seeded at 0 picked). This is the legacy "recognize the first line was
// fulfilled, open a fresh pick line for the added quantity" behavior.
func TestSalesOrder_LineQuantityIncrease_AfterFullPickPack_CreatesRemainderPickLine(t *testing.T) {
	t.Parallel()
	orderID := createLifecycleOrder(t)
	lineID := orderSaleLineID(t, orderID)

	// Order 10, then commit for fulfillment.
	patchSaleLineQuantity(t, orderID, lineID, "10")
	status, body := salesOrderAction(t, orderID, "issue", false)
	requireStatus(t, 200, status, body)

	pickID := orderPickID(t, orderID)

	// Pick all 10, then pack — after which exactly one packed pick line at 10 exists and
	// nothing remains to pick (so packing itself does not open a remainder line).
	pickAllLines(t, pickID)
	packPick(t, pickID)

	packedRows := fetchPickLines(t, pickID)
	require.Len(t, packedRows, 1, "after a full pick+pack there is exactly one (packed) pick line")
	assert.True(t, packedRows[0].packed, "the line is packed")
	assert.Equal(t, 10.0, packedRows[0].picked, "the packed line carries the full picked quantity")

	// Bump the order line 10 -> 20. Reconciliation must open a fresh pick line for the added 10.
	patchSaleLineQuantity(t, orderID, lineID, "20")

	rows := fetchPickLines(t, pickID)
	require.Len(t, rows, 2, "increasing a fully-packed line opens a new pick line for the delta")

	packed, unpacked := countPacked(rows)
	assert.Equal(t, 1, packed, "the original fulfilled pick line stays packed")
	assert.Equal(t, 1, unpacked, "a single new open pick line covers the added quantity")

	for _, r := range rows {
		if r.packed {
			assert.Equal(t, 10.0, r.picked, "the packed line's picked quantity is unchanged")
		} else {
			assert.Equal(t, 0.0, r.picked, "the new remainder line is seeded at 0 picked, not the delta")
			assert.Equal(t, 20.0, r.ordered, "the new line's ordered quantity reflects the current order line")
		}
	}
}

// TestSalesOrder_LineQuantityIncrease_Twice_DoesNotDuplicateRemainderPickLine pins the
// "one open pick line per order line" invariant: once an open pick line exists for the
// added quantity, bumping the order line again must NOT open a second one — the existing
// open line is the remaining-to-pick bucket.
func TestSalesOrder_LineQuantityIncrease_Twice_DoesNotDuplicateRemainderPickLine(t *testing.T) {
	t.Parallel()
	orderID := createLifecycleOrder(t)
	lineID := orderSaleLineID(t, orderID)

	patchSaleLineQuantity(t, orderID, lineID, "10")
	status, body := salesOrderAction(t, orderID, "issue", false)
	requireStatus(t, 200, status, body)

	pickID := orderPickID(t, orderID)
	pickAllLines(t, pickID)
	packPick(t, pickID)

	// First bump opens one remainder line (2 total).
	patchSaleLineQuantity(t, orderID, lineID, "20")
	require.Len(t, fetchPickLines(t, pickID), 2, "first increase opens a remainder pick line")

	// Second bump: an open line already exists, so no new line is opened.
	patchSaleLineQuantity(t, orderID, lineID, "30")
	rows := fetchPickLines(t, pickID)
	require.Len(t, rows, 2, "a second increase must not duplicate the open remainder pick line")

	packed, unpacked := countPacked(rows)
	assert.Equal(t, 1, packed, "still one packed line")
	assert.Equal(t, 1, unpacked, "still one open line")
}

// TestSalesOrder_AddLineToPickedOrder_CreatesPickLine covers the create path: adding a new
// sale line to an order that already has a pick opens a pick line for the new line.
func TestSalesOrder_AddLineToPickedOrder_CreatesPickLine(t *testing.T) {
	t.Parallel()
	orderID := createLifecycleOrder(t)

	status, body := salesOrderAction(t, orderID, "issue", false)
	requireStatus(t, 200, status, body)
	pickID := orderPickID(t, orderID)

	before := fetchPickLines(t, pickID)

	// Add a second sale line (distinct quantity 5 so we can identify its pick line).
	lineBody := map[string]any{
		"product_id":  SeedProductID,
		"product_sku": "E2E-RECON-ADD",
		"quantity":    map[string]any{"value": "5", "unit_id": SeedUnitID},
		"unit_price": map[string]any{
			"value":               "9.00",
			"numerator_unit_id":   e2eCurrencyUnitID,
			"denominator_unit_id": SeedUnitID,
		},
	}
	aStatus, aBody, err := apiClient.Post(salesOrdersPath+"/"+orderID+"/lines", lineBody, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, aStatus, aBody)

	after := fetchPickLines(t, pickID)
	require.Len(t, after, len(before)+1, "adding a sale line to a picked order opens a pick line for it")

	var foundNew bool
	for _, r := range after {
		if r.ordered == 5.0 {
			foundNew = true
			assert.Equal(t, 0.0, r.picked, "the new line's pick line starts at 0 picked")
			assert.False(t, r.packed, "the new line's pick line is open")
		}
	}
	assert.True(t, foundNew, "a pick line exists for the newly added order line (ordered 5)")
}

// TestSalesOrder_UpdateSystemLineOnPickedOrder_DoesNotCreatePickLine pins the sale-type gate
// on UPDATE: the synthesized shipping line is never picked, so editing it on a picked order
// must not open a spurious pick line.
func TestSalesOrder_UpdateSystemLineOnPickedOrder_DoesNotCreatePickLine(t *testing.T) {
	t.Parallel()
	orderID := createLifecycleOrder(t)
	systemLineID := orderSystemLineID(t, orderID)

	status, body := salesOrderAction(t, orderID, "issue", false)
	requireStatus(t, 200, status, body)
	pickID := orderPickID(t, orderID)

	before := fetchPickLines(t, pickID)

	// Edit the shipping line (a non-sale, "system" line). The reconciliation is gated to
	// sale-type lines, so this must not add a pick line.
	desc := "e2e system-line edit"
	uStatus, uBody, err := apiClient.Patch(salesOrdersPath+"/"+orderID+"/lines/"+systemLineID,
		map[string]any{"product_description": desc}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, uStatus, uBody)

	after := fetchPickLines(t, pickID)
	assert.Len(t, after, len(before), "editing a system (shipping) line must not open a pick line")
}

// TestSalesOrder_LineQuantityDecrease_AfterFullPickPack_LeavesPickLinesUntouched pins that a
// decrease is not reconciled: remaining clamps to 0, so no new pick line is opened and the
// already-packed line is left exactly as it was (matching legacy, which never shrinks or
// voids pick lines on a quantity drop).
func TestSalesOrder_LineQuantityDecrease_AfterFullPickPack_LeavesPickLinesUntouched(t *testing.T) {
	t.Parallel()
	orderID := createLifecycleOrder(t)
	lineID := orderSaleLineID(t, orderID)

	patchSaleLineQuantity(t, orderID, lineID, "10")
	status, body := salesOrderAction(t, orderID, "issue", false)
	requireStatus(t, 200, status, body)

	pickID := orderPickID(t, orderID)
	pickAllLines(t, pickID)
	packPick(t, pickID)
	require.Len(t, fetchPickLines(t, pickID), 1)

	// Drop the order line 10 -> 5. Nothing is opened and the packed line is unchanged.
	patchSaleLineQuantity(t, orderID, lineID, "5")

	rows := fetchPickLines(t, pickID)
	require.Len(t, rows, 1, "decreasing quantity does not open a pick line")
	assert.True(t, rows[0].packed, "the packed line remains packed")
	assert.Equal(t, 10.0, rows[0].picked, "the packed line's picked quantity is not shrunk")
}

// TestPick_PartialPickThenPack_OpensRemainderPickLine exercises the same reconciliation via
// the pack path: a line ordered 10 but only 6 picked, then packed, leaves the 6 packed and
// opens a fresh open pick line for the outstanding 4. This shares CalculateRemainingForOrderLine
// with the line-update path, so it guards the same decimal-decoding fix at the pack call site.
func TestPick_PartialPickThenPack_OpensRemainderPickLine(t *testing.T) {
	t.Parallel()
	orderID := createLifecycleOrder(t)
	lineID := orderSaleLineID(t, orderID)

	patchSaleLineQuantity(t, orderID, lineID, "10")
	status, body := salesOrderAction(t, orderID, "issue", false)
	requireStatus(t, 200, status, body)

	pickID := orderPickID(t, orderID)

	// Pick only 6 of the 10, then pack.
	setPickedQuantity(t, pickID, firstPickLineID(t, pickID), "6")
	packPick(t, pickID)

	rows := fetchPickLines(t, pickID)
	require.Len(t, rows, 2, "packing a partially-picked line opens a remainder pick line for the outstanding quantity")

	packed, unpacked := countPacked(rows)
	assert.Equal(t, 1, packed, "the picked portion is packed")
	assert.Equal(t, 1, unpacked, "one open remainder line covers the rest")

	for _, r := range rows {
		if r.packed {
			assert.Equal(t, 6.0, r.picked, "the packed line carries the 6 that were picked")
		} else {
			assert.Equal(t, 0.0, r.picked, "the remainder line is seeded at 0 picked")
			assert.Equal(t, 10.0, r.ordered, "the remainder line points at the same order line (ordered 10)")
		}
	}

	// The pick still has outstanding quantity to pick, so it must NOT read as finished.
	assert.False(t, pickIsFinished(t, pickID), "a pick with an open remainder line is not finished")
}

// TestSalesOrder_LineQuantityIncrease_ReopensFinishedPick pins that bumping a line on a
// fully-packed (finished) pick reopens it — otherwise the new remainder pick line is
// frozen behind a closed pick in the UI.
func TestSalesOrder_LineQuantityIncrease_ReopensFinishedPick(t *testing.T) {
	t.Parallel()
	orderID := createLifecycleOrder(t)
	lineID := orderSaleLineID(t, orderID)

	patchSaleLineQuantity(t, orderID, lineID, "10")
	status, body := salesOrderAction(t, orderID, "issue", false)
	requireStatus(t, 200, status, body)

	pickID := orderPickID(t, orderID)
	pickAllLines(t, pickID)
	packPick(t, pickID)

	// A full pick+pack finishes the pick.
	require.True(t, pickIsFinished(t, pickID), "a fully-packed pick is finished")

	// Bumping the quantity adds outstanding work, so the pick must reopen.
	patchSaleLineQuantity(t, orderID, lineID, "20")
	assert.False(t, pickIsFinished(t, pickID), "adding quantity reopens the finished pick")
	require.Len(t, fetchPickLines(t, pickID), 2, "the added quantity opens a remainder pick line")
}

// TestSalesOrder_LineQuantityDecrease_DeletesSurplusOpenPickLine pins that dropping a line
// back to its already-packed quantity removes the now-unneeded open remainder pick line
// (and re-finishes the pick). Ordered 10 packed, bumped to 15 (opening a remainder), then
// dropped back to 10: the remainder line is deleted.
func TestSalesOrder_LineQuantityDecrease_DeletesSurplusOpenPickLine(t *testing.T) {
	t.Parallel()
	orderID := createLifecycleOrder(t)
	lineID := orderSaleLineID(t, orderID)

	patchSaleLineQuantity(t, orderID, lineID, "10")
	status, body := salesOrderAction(t, orderID, "issue", false)
	requireStatus(t, 200, status, body)

	pickID := orderPickID(t, orderID)
	pickAllLines(t, pickID)
	packPick(t, pickID)

	// Bump to 15 → a remainder line opens and the pick reopens.
	patchSaleLineQuantity(t, orderID, lineID, "15")
	require.Len(t, fetchPickLines(t, pickID), 2, "the increase opens a remainder pick line")
	require.False(t, pickIsFinished(t, pickID), "the pick reopens while there is outstanding quantity")

	// Drop back to 10 → the packed 10 now covers the order, so the surplus open line is deleted.
	patchSaleLineQuantity(t, orderID, lineID, "10")
	rows := fetchPickLines(t, pickID)
	require.Len(t, rows, 1, "dropping back to the packed quantity deletes the surplus open pick line")
	assert.True(t, rows[0].packed, "the remaining line is the original packed line")
	assert.Equal(t, 10.0, rows[0].picked, "the packed line is left untouched")
	assert.True(t, pickIsFinished(t, pickID), "with nothing left to pick, the pick finishes again")
}

// TestSalesOrder_DeleteLine_FinishesPickWhenRemainingLinesPacked pins that deleting the
// last open line from a pick finishes it when every remaining line is packed.
func TestSalesOrder_DeleteLine_FinishesPickWhenRemainingLinesPacked(t *testing.T) {
	t.Parallel()
	orderID := createLifecycleOrder(t)
	lineA := orderSaleLineID(t, orderID)
	patchSaleLineQuantity(t, orderID, lineA, "10")

	// Add a second sale line (B, ordered 5) that will stay unpicked.
	lineBody := map[string]any{
		"product_id":  SeedProductID,
		"product_sku": "E2E-RECON-B",
		"quantity":    map[string]any{"value": "5", "unit_id": SeedUnitID},
		"unit_price": map[string]any{
			"value":               "7.00",
			"numerator_unit_id":   e2eCurrencyUnitID,
			"denominator_unit_id": SeedUnitID,
		},
	}
	aStatus, aBody, err := apiClient.Post(salesOrdersPath+"/"+orderID+"/lines", lineBody, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, aStatus, aBody)
	lineB := jsonField(parseJSON(aBody), "id")

	status, body := salesOrderAction(t, orderID, "issue", false)
	requireStatus(t, 200, status, body)
	pickID := orderPickID(t, orderID)

	// Pick + pack only line A (ordered 10); line B stays open at 0.
	setPickedQuantity(t, pickID, pickLineIDByOrdered(t, pickID, 10), "10")
	packPick(t, pickID)
	require.False(t, pickIsFinished(t, pickID), "line B is still open, so the pick is not finished")
	require.Len(t, fetchPickLines(t, pickID), 2)

	// Delete line B. Its open pick line goes away, leaving only the packed line A → finish.
	dStatus, dBody, err := apiClient.Delete(salesOrdersPath + "/" + orderID + "/lines/" + lineB)
	require.NoError(t, err)
	require.GreaterOrEqual(t, dStatus, 200, "delete line B: %s", string(dBody))
	require.Less(t, dStatus, 300, "delete line B: %s", string(dBody))

	rows := fetchPickLines(t, pickID)
	require.Len(t, rows, 1, "only line A's packed pick line remains")
	assert.True(t, rows[0].packed)
	assert.True(t, pickIsFinished(t, pickID), "with every remaining line packed, the pick finishes")
}

// TestSalesOrder_DeleteLine_EmptyPickUnissuesOrder pins that deleting the last line that
// leaves the pick empty deletes the pick and reverts the order to estimate.
func TestSalesOrder_DeleteLine_EmptyPickUnissuesOrder(t *testing.T) {
	t.Parallel()
	orderID := createLifecycleOrder(t)
	saleLine := orderSaleLineID(t, orderID)

	status, body := salesOrderAction(t, orderID, "issue", false)
	requireStatus(t, 200, status, body)
	assert.Equal(t, "issued", jsonField(getSalesOrder(t, orderID, nil), "status"))
	pickID := orderPickID(t, orderID)

	// Deleting the only sale line empties the pick.
	dStatus, dBody, err := apiClient.Delete(salesOrdersPath + "/" + orderID + "/lines/" + saleLine)
	require.NoError(t, err)
	require.GreaterOrEqual(t, dStatus, 200, "delete sale line: %s", string(dBody))
	require.Less(t, dStatus, 300, "delete sale line: %s", string(dBody))

	// The order reverts to estimate and the now-empty pick is deleted.
	after := getSalesOrder(t, orderID, url.Values{"include": {"related.pick"}})
	assert.Equal(t, "estimate", jsonField(after, "status"), "emptying the pick reverts the order to estimate")
	if related := jsonObject(after, "related"); related != nil {
		assert.Nil(t, related["pick"], "the empty pick is deleted")
	}

	pStatus, _, err := apiClient.GetListRaw("/v1/operations/picks/"+pickID, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, pStatus, "the deleted pick is gone")
}
