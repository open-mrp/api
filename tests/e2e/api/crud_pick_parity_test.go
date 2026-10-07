//go:build e2e

package api_test

import (
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Behavioural parity for the pick writes the dashboard drives: the fill math behind Finish picking
// and Pick all, the guards on packed lines and finished picks, and packing a pick at most once.
// Every test issues its own order, so they run in parallel and leave the shared seed alone.

// Fills one line through its pick action (the dashboard's Finish picking) and returns its quantity.
func finishPickLine(t *testing.T, pickID, lineID string) float64 {
	t.Helper()
	status, body, err := apiClient.Put(picksPath+"/"+pickID+"/lines/"+lineID+"/actions/pick", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return pickLineQuantity(t, pickID, lineID)
}

func pickLineQuantity(t *testing.T, pickID, lineID string) float64 {
	t.Helper()
	quantity, ok := readPickLineQuantities(t, pickID)[lineID]
	require.True(t, ok, "pick %s has line %s", pickID, lineID)
	return quantity
}

func sumPickLineQuantities(t *testing.T, pickID string) float64 {
	t.Helper()
	var total float64
	for _, quantity := range readPickLineQuantities(t, pickID) {
		total += quantity
	}
	return total
}

// Issues a ten-unit order, picks and packs six, and returns the pick and its open remainder line —
// the shape where the order line has more than one pick line.
func partlyPackedPick(t *testing.T) (pickID, remainderID string) {
	t.Helper()
	pickID = pickForOrderBody(t, orderBodyForQuantity(t, "10"))
	setPickedQuantity(t, pickID, firstUnpackedPickLine(t, pickID), "6")
	packPick(t, pickID)

	remainderID = firstUnpackedPickLine(t, pickID)
	require.NotEmpty(t, remainderID, "packing 6 of 10 opens a remainder line")
	require.InDelta(t, 0.0, pickLineQuantity(t, pickID, remainderID), 0.001, "the remainder line starts at zero")
	return pickID, remainderID
}

// Issues a five-unit order, picks and packs all of it, and returns the pick and its packed line.
func fullyPackedPick(t *testing.T) (pickID, lineID string) {
	t.Helper()
	pickID = pickForOrderBody(t, orderBodyForQuantity(t, "5"))
	lineID = firstUnpackedPickLine(t, pickID)
	require.NotEmpty(t, lineID)
	pickAllLines(t, pickID)
	packPick(t, pickID)
	require.Empty(t, firstUnpackedPickLine(t, pickID), "a fully picked line leaves no remainder")
	return pickID, lineID
}

// --- Finish picking a line ---

func TestPickParity_FinishLine_FillsOnlyWhatOtherLinesLeave(t *testing.T) {
	t.Parallel()
	pickID, remainder := partlyPackedPick(t)

	assert.InDelta(t, 4.0, finishPickLine(t, pickID, remainder), 0.001,
		"10 ordered less the 6 already packed")
	assert.InDelta(t, 4.0, finishPickLine(t, pickID, remainder), 0.001,
		"a second click changes nothing rather than adding the outstanding quantity again")

	setPickedQuantity(t, pickID, remainder, "3")
	assert.InDelta(t, 4.0, finishPickLine(t, pickID, remainder), 0.001,
		"a short pick is topped up to what is outstanding, not by it")

	assert.InDelta(t, 10.0, sumPickLineQuantities(t, pickID), 0.001,
		"the order line's pick lines hold exactly what was ordered")
}

func TestPickParity_FinishLine_KeepsAnOverPick(t *testing.T) {
	t.Parallel()
	pickID := pickForOrderBody(t, orderBodyForQuantity(t, "10"))
	line := firstUnpackedPickLine(t, pickID)

	setPickedQuantity(t, pickID, line, "12")
	assert.InDelta(t, 12.0, finishPickLine(t, pickID, line), 0.001,
		"an over-pick is a real floor event and is never lowered to the ordered quantity")
}

// --- Pick all ---

func TestPickParity_PickAll_FillsTheRemainderOnce(t *testing.T) {
	t.Parallel()
	pickID, remainder := partlyPackedPick(t)

	pickAllLines(t, pickID)
	assert.InDelta(t, 4.0, pickLineQuantity(t, pickID, remainder), 0.001, "10 ordered less the 6 already packed")

	pickAllLines(t, pickID)
	assert.InDelta(t, 4.0, pickLineQuantity(t, pickID, remainder), 0.001, "picking all again changes nothing")
	assert.InDelta(t, 10.0, sumPickLineQuantities(t, pickID), 0.001)
}

func TestPickParity_PickAll_FillsAPartialPickToOrdered(t *testing.T) {
	t.Parallel()
	pickID := pickForOrderBody(t, orderBodyForQuantity(t, "10"))
	line := firstUnpackedPickLine(t, pickID)

	setPickedQuantity(t, pickID, line, "3")
	pickAllLines(t, pickID)
	assert.InDelta(t, 10.0, pickLineQuantity(t, pickID, line), 0.001, "the line fills to the ordered quantity, not 3 short of it")
}

func TestPickParity_PickAll_KeepsAnOverPick(t *testing.T) {
	t.Parallel()
	pickID := pickForOrderBody(t, orderBodyForQuantity(t, "10"))
	line := firstUnpackedPickLine(t, pickID)

	setPickedQuantity(t, pickID, line, "12")
	pickAllLines(t, pickID)
	assert.InDelta(t, 12.0, pickLineQuantity(t, pickID, line), 0.001, "pick all never lowers a line")
	pickAllLines(t, pickID)
	assert.InDelta(t, 12.0, pickLineQuantity(t, pickID, line), 0.001)
}

// Each order line fills on its own: one over-picked and one short in the same pick.
func TestPickParity_PickAll_FillsEachOrderLineIndependently(t *testing.T) {
	t.Parallel()
	body := minimalSalesOrderCreateBody(t, SeedCustomerAccountID)
	body["lines"] = []map[string]any{
		{"product_id": SeedProductID, "quantity": map[string]any{"value": "10", "unit_id": SeedUnitID}},
		{"product_id": SeedProductID, "quantity": map[string]any{"value": "5", "unit_id": SeedUnitID}},
	}
	pickID := pickForOrderBody(t, body)
	tenLine := pickLineIDByOrdered(t, pickID, 10)
	fiveLine := pickLineIDByOrdered(t, pickID, 5)

	setPickedQuantity(t, pickID, tenLine, "12")
	setPickedQuantity(t, pickID, fiveLine, "2")
	pickAllLines(t, pickID)

	quantities := readPickLineQuantities(t, pickID)
	assert.InDelta(t, 12.0, quantities[tenLine], 0.001, "the over-picked line is kept")
	assert.InDelta(t, 5.0, quantities[fiveLine], 0.001, "the short line fills to its own order line")
}

// --- Packed lines ---

// A packed line's quantity is what its shipment carries, so neither action may change it.
func TestPickParity_VoidLine_RefusesAPackedLine(t *testing.T) {
	t.Parallel()
	pickID, line := fullyPackedPick(t)

	status, body, err := apiClient.Put(picksPath+"/"+pickID+"/lines/"+line+"/actions/void", nil)
	require.NoError(t, err)
	require.Equal(t, 400, status, "voiding a packed line is a validation error: %s", string(body))
	assert.Contains(t, strings.ToLower(string(body)), "packed")
	assert.InDelta(t, 5.0, pickLineQuantity(t, pickID, line), 0.001, "the packed quantity is untouched")
}

func TestPickParity_UpdateLine_RefusesAPackedLine(t *testing.T) {
	t.Parallel()
	pickID, line := fullyPackedPick(t)

	status, body, err := apiClient.Patch(picksPath+"/"+pickID+"/lines/"+line,
		map[string]any{"quantity_value": "2"}, newIdempotencyKey())
	require.NoError(t, err)
	require.Equal(t, 400, status, "editing a packed line is a validation error: %s", string(body))
	assert.Contains(t, strings.ToLower(string(body)), "packed")
	assert.InDelta(t, 5.0, pickLineQuantity(t, pickID, line), 0.001, "the packed quantity still matches its shipment line")
}

// --- Pack ---

func TestPickParity_Pack_RefusesAFinishedPick(t *testing.T) {
	t.Parallel()
	pickID := pickForOrderBody(t, orderBodyForQuantity(t, "5"))
	pickAllLines(t, pickID)

	// The API only finishes a pick once its lines are packed, so the state is set directly: the guard,
	// not an empty line set, has to be what refuses this pack.
	_, err := authDB(t).Exec("UPDATE pick SET finished_at = NOW(3) WHERE id = ?", pickID)
	require.NoError(t, err)

	status, body, err := apiClient.Post(picksPath+"/"+pickID+"/actions/pack",
		map[string]any{"shipment_case_count": 1}, newIdempotencyKey())
	require.NoError(t, err)
	require.Equal(t, 400, status, "packing a finished pick is refused at accept: %s", string(body))
	assert.Contains(t, strings.ToLower(string(body)), "finished")
	assert.Empty(t, pickShipmentNumbers(t, pickID), "no shipment was raised")
}

// Two packs of one pick (two tabs, or two core pods taking the jobs) must ship the lines once.
func TestPickParity_Pack_ConcurrentPacksMakeOneShipment(t *testing.T) {
	t.Parallel()
	pickID := pickForOrderBody(t, orderBodyForQuantity(t, "5"))
	pickAllLines(t, pickID)

	type accepted struct {
		status int
		body   []byte
		err    error
	}
	results := make([]accepted, 2)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			status, body, err := apiClient.Post(picksPath+"/"+pickID+"/actions/pack",
				map[string]any{"shipment_case_count": 1}, newIdempotencyKey())
			results[i] = accepted{status: status, body: body, err: err}
		}(i)
	}
	close(start)
	wg.Wait()

	completed, refused := 0, 0
	for _, r := range results {
		require.NoError(t, r.err)
		if r.status == 400 {
			// The other pack had already packed every line by the time this one was accepted.
			refused++
			continue
		}
		requireStatus(t, 202, r.status, r.body)
		job := pollJobUntilTerminal(t, jsonField(parseJSON(r.body), "id"))
		switch jsonField(job, "status") {
		case "completed":
			completed++
		case "failed":
			refused++
		default:
			t.Fatalf("pack job ended %q: %v", jsonField(job, "status"), job)
		}
	}
	assert.Equal(t, 1, completed, "exactly one pack ships the lines")
	assert.Equal(t, 1, refused, "the other finds nothing left to pack")
	assert.Len(t, pickShipmentNumbers(t, pickID), 1, "the pick's order has one shipment, not two")
	assert.True(t, pickIsFinished(t, pickID), "the one pack finishes the pick")
}

// --- Void ---

func TestPickParity_Void_RefusedOnceShipped(t *testing.T) {
	t.Parallel()
	pickID, line := fullyPackedPick(t)

	status, body, err := apiClient.Put(picksPath+"/"+pickID+"/actions/void", nil)
	require.NoError(t, err)
	require.Equal(t, 400, status, "voiding a pick whose order has a shipment is refused: %s", string(body))
	assert.Contains(t, strings.ToLower(string(body)), "shipped")
	assert.InDelta(t, 5.0, pickLineQuantity(t, pickID, line), 0.001)
	assert.True(t, pickIsFinished(t, pickID), "the refused void leaves the pick finished")
}

// --- Order close and reopen ---

// Reopening a closed order undoes the close on its pick, and only that: a line a shipment carries
// stays packed, so its goods cannot be picked, edited or shipped a second time.
func TestPickParity_ReopenOrder_KeepsShippedLinesPacked(t *testing.T) {
	t.Parallel()
	pickID, remainder := partlyPackedPick(t)
	orderID := jsonField(jsonObject(jsonObject(retrievePick(t, pickID, "related.sales_order"), "related"), "sales_order"), "id")
	require.NotEmpty(t, orderID)

	var shippedLine string
	for _, line := range pickLineObjects(t, pickID) {
		if line["packed_at"] != nil {
			shippedLine = jsonField(line, "id")
		}
	}
	require.NotEmpty(t, shippedLine, "the six picked are packed onto the first shipment")

	status, body := salesOrderAction(t, orderID, "close", false)
	requireStatus(t, 200, status, body)
	status, body = salesOrderAction(t, orderID, "open", false)
	requireStatus(t, 200, status, body)

	packed := map[string]bool{}
	for _, line := range pickLineObjects(t, pickID) {
		packed[jsonField(line, "id")] = line["packed_at"] != nil
	}
	assert.True(t, packed[shippedLine], "the shipped line stays packed")
	assert.False(t, packed[remainder], "the remainder the close packed is open again")
	assert.False(t, pickIsFinished(t, pickID), "the reopened pick has work left")

	status, body, err := apiClient.Put(picksPath+"/"+pickID+"/lines/"+shippedLine+"/actions/void", nil)
	require.NoError(t, err)
	assert.Equal(t, 400, status, "the shipped line cannot be voided: %s", string(body))

	// Finishing the order ships exactly the four still outstanding.
	pickAllLines(t, pickID)
	assert.InDelta(t, 6.0, pickLineQuantity(t, pickID, shippedLine), 0.001)
	assert.InDelta(t, 4.0, pickLineQuantity(t, pickID, remainder), 0.001)
	packPick(t, pickID)

	shipments := jsonArray(jsonObject(jsonObject(retrievePick(t, pickID, "related.shipments"), "related"), "shipments"), "data")
	require.Len(t, shipments, 2)
	var shipped float64
	for _, raw := range shipments {
		for _, line := range jsonListData(readShipment(t, jsonField(raw.(map[string]any), "id"), "lines"), "lines") {
			value, err := strconv.ParseFloat(jsonField(jsonObject(line.(map[string]any), "quantity"), "value"), 64)
			require.NoError(t, err)
			shipped += value
		}
	}
	assert.InDelta(t, 10.0, shipped, 0.001, "the two shipments carry the ten ordered, nothing twice")
}
