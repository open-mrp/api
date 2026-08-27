//go:build e2e

package api_test

import (
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Behavioural coverage for the pick write surface: update, the three line ops, pick-all, void and
// pack. These MUTATE state, so they run against the dedicated PICK-PB-001 rows seeded in
// 0014_e2e_extras.sql — a pick whose order has no shipment, which is what makes the void happy
// path reachable at all (every pick in the base seed is blocked by the shipped-items guard).
//
// The state-mutating tests share that pick and run serially; only the rejection cases that never
// touch it opt into t.Parallel.
const (
	pbPickID  = "pk_01seedpbpick000000"
	pbLine1ID = "pkln_01seedpb_ln1_000"
	pbLine2ID = "pkln_01seedpb_ln2_000"
	// Ordered quantities on the two order lines behind those pick lines.
	pbLine1Ordered = 10.0
	pbLine2Ordered = 4.0

	// Packing marks lines packed permanently, so it gets its own pick rather than poisoning the
	// one the reversible tests share.
	pbPackPickID = "pk_01seedpb2pick00000"
)

// Returns each pick line's current quantity, keyed by pick line id.
func readPickLineQuantities(t *testing.T, pickID string) map[string]float64 {
	t.Helper()

	status, body, err := apiClient.GetListRaw(picksPath+"/"+pickID, url.Values{"include": {"lines"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	out := map[string]float64{}
	lines := jsonObject(parseJSON(body), "lines")
	require.NotNil(t, lines, "lines present with ?include=lines")

	data, _ := lines["data"].([]any)
	for _, raw := range data {
		line, ok := raw.(map[string]any)
		require.True(t, ok, "line entry is an object")
		qty := jsonObject(line, "quantity")
		require.NotNil(t, qty, "pick line carries a quantity")
		v, err := strconv.ParseFloat(jsonField(qty, "value"), 64)
		require.NoError(t, err, "quantity value parses")
		out[jsonField(line, "id")] = v
	}
	return out
}

// Resets both dedicated lines to zero, so each test starts from a known state. Lines are voided one
// at a time rather than through the pick-level void action, which the shipments the pack tests leave
// behind would refuse.
func resetPBPick(t *testing.T) {
	t.Helper()
	for _, lineID := range []string{pbLine1ID, pbLine2ID} {
		status, body, err := apiClient.Put(picksPath+"/"+pbPickID+"/lines/"+lineID+"/actions/void", nil)
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
	}
}

// --- item 6: line ops -----------------------------------------------------

func TestPicks_PickLine_FillsRemainingExcludingItself(t *testing.T) {
	resetPBPick(t)

	// From zero, picking the line fills it to the full ordered quantity.
	status, body, err := apiClient.Put(picksPath+"/"+pbPickID+"/lines/"+pbLine1ID+"/actions/pick", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.InDelta(t, pbLine1Ordered, readPickLineQuantities(t, pbPickID)[pbLine1ID], 0.001)

	// Picking again is a no-op rather than a reset: remaining excludes the line's own quantity,
	// so a fully-picked line stays full. This is a deliberate divergence from Dashboard, which
	// subtracts the total including self and would leave it at zero.
	status, body, err = apiClient.Put(picksPath+"/"+pbPickID+"/lines/"+pbLine1ID+"/actions/pick", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.InDelta(t, pbLine1Ordered, readPickLineQuantities(t, pbPickID)[pbLine1ID], 0.001,
		"picking a full line keeps it full")
}

func TestPicks_UpdateAndVoidLine(t *testing.T) {
	resetPBPick(t)

	status, body, err := apiClient.Patch(picksPath+"/"+pbPickID+"/lines/"+pbLine2ID,
		map[string]any{"quantity_value": "3"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.InDelta(t, 3.0, readPickLineQuantities(t, pbPickID)[pbLine2ID], 0.001)

	status, body, err = apiClient.Put(picksPath+"/"+pbPickID+"/lines/"+pbLine2ID+"/actions/void", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.InDelta(t, 0.0, readPickLineQuantities(t, pbPickID)[pbLine2ID], 0.001, "void zeroes the line")
}

func TestPicks_LineOps_RejectLineFromAnotherPick(t *testing.T) {
	t.Parallel()

	// A pick line that belongs to a different pick must not be reachable through this one.
	status, body, err := apiClient.Put(picksPath+"/"+pbPickID+"/lines/"+SeedPickLineID+"/actions/pick", nil)
	require.NoError(t, err)
	assert.Less(t, status, 500, "belongs-to-pick mismatch must not 5xx: %s", string(body))
	assert.GreaterOrEqual(t, status, 400, "mismatched line should be rejected")
}

// --- item 7: pick-all -----------------------------------------------------

func TestPicks_PickAllLines_FillsEveryUnpackedLine(t *testing.T) {
	resetPBPick(t)

	status, body, err := apiClient.Put(picksPath+"/"+pbPickID+"/actions/pick", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	quantities := readPickLineQuantities(t, pbPickID)
	assert.InDelta(t, pbLine1Ordered, quantities[pbLine1ID], 0.001)
	assert.InDelta(t, pbLine2Ordered, quantities[pbLine2ID], 0.001)

	// Progress is a base scalar, so the action response itself reports it.
	picked := jsonObject(jsonObject(parseJSON(body), "totals"), "picked")
	assert.InDelta(t, 1.0, picked["completion"].(float64), 0.001,
		"a fully picked pick reports picked completion 1")
}

// --- item 8: void ---------------------------------------------------------

func TestPicks_Void_ZeroesLinesAndClearsFinishedAt(t *testing.T) {
	resetPBPick(t)

	status, body, err := apiClient.Put(picksPath+"/"+pbPickID+"/actions/pick", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	status, body, err = apiClient.Put(picksPath+"/"+pbPickID+"/actions/void", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	quantities := readPickLineQuantities(t, pbPickID)
	assert.InDelta(t, 0.0, quantities[pbLine1ID], 0.001)
	assert.InDelta(t, 0.0, quantities[pbLine2ID], 0.001)
	assert.Nil(t, parseJSON(body)["finished_at"], "void leaves the pick open")
}

func TestPicks_Void_RejectedWhenOrderHasShipments(t *testing.T) {
	t.Parallel()

	// The seeded pick's order carries SHP-003, so the guard fires. It must be a validation
	// error, never a 5xx.
	status, body, err := apiClient.Put(picksPath+"/"+SeedPickID+"/actions/void", nil)
	require.NoError(t, err)
	assert.Less(t, status, 500, "shipped-items guard must not 5xx: %s", string(body))
	require.Equal(t, 400, status, "expected a validation error, got %d: %s", status, string(body))
	assert.Contains(t, strings.ToLower(string(body)), "shipped",
		"error should name the shipped-items guard")
}

// --- item 9: pack ---------------------------------------------------------

func TestPicks_Pack_CreatesShipmentAndSynthesizesRemainingLine(t *testing.T) {
	// Pack the line only partially. Packing marks a line packed for good, so packing the full
	// quantity would leave nothing to pack and make this test single-use; a partial pack also
	// exercises the remaining-quantity line the flow synthesizes.
	before := len(readPickLineQuantities(t, pbPackPickID))

	unpacked := firstUnpackedPickLine(t, pbPackPickID)
	require.NotEmpty(t, unpacked, "the pack pick has an unpacked line to work with")

	status, body, err := apiClient.Patch(picksPath+"/"+pbPackPickID+"/lines/"+unpacked,
		map[string]any{"quantity_value": "1"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	job := acceptPackPick(t, pbPackPickID, 2)

	results := jobResults(job)
	require.Len(t, results, 1, "one pack, one result")
	assert.Equal(t, "created", results[0]["status"])
	shipmentID := jobResultResourceID(results[0])
	require.NotEmpty(t, shipmentID, "the completed job names the shipment it created")

	// The result names the shipment as well as identifying it, so a caller can print a label
	// without reading the shipment back.
	shipmentNumber := jsonField(jsonObject(results[0], "resource"), "name")
	require.NotEmpty(t, shipmentNumber, "the job result carries the shipment's number")
	assert.Equal(t, jsonField(readShipment(t, shipmentID), "number"), shipmentNumber,
		"the name on the result is the shipment's own number")
	// First shipment for an order takes the order number; later ones append a suffix.
	assert.True(t, strings.HasPrefix(shipmentNumber, "ORD-PB-002"),
		"shipment number derives from the order number, got %q", shipmentNumber)

	// Quantity is still outstanding, so the flow adds a fresh zero-quantity line for the rest
	// rather than reopening the packed one.
	assert.Greater(t, len(readPickLineQuantities(t, pbPackPickID)), before,
		"packing a partial quantity synthesizes a remaining-quantity line")
}

// Returns the id of the first pick line that has not been packed yet.
func firstUnpackedPickLine(t *testing.T, pickID string) string {
	t.Helper()

	status, body, err := apiClient.GetListRaw(picksPath+"/"+pickID, url.Values{"include": {"lines"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	lines := jsonObject(parseJSON(body), "lines")
	require.NotNil(t, lines, "lines present with ?include=lines")
	data, _ := lines["data"].([]any)
	for _, raw := range data {
		line, ok := raw.(map[string]any)
		if ok && line["packed_at"] == nil {
			return jsonField(line, "id")
		}
	}
	return ""
}

func TestPicks_Pack_RejectsWhenNothingToPack(t *testing.T) {
	resetPBPick(t)

	// All lines sit at zero after the reset, so there is nothing eligible. Packing is async, but
	// this is checked at accept — a caller learns now rather than from a job that failed later.
	status, body, err := apiClient.Post(picksPath+"/"+pbPickID+"/actions/pack",
		map[string]any{"shipment_case_count": 1}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Less(t, status, 500, "no-lines-to-pack must not 5xx: %s", string(body))
	assert.GreaterOrEqual(t, status, 400, "packing nothing is rejected synchronously, never accepted as a job")
}

func TestPicks_Pack_RejectsCaseCountBelowOne(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(picksPath+"/"+pbPickID+"/actions/pack",
		map[string]any{"shipment_case_count": 0}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Less(t, status, 500, "validation must not 5xx: %s", string(body))
	assert.Equal(t, 400, status, "shipment_case_count must be >= 1")
}

// --- pick line quantity unit ----------------------------------------------

// Returns the id of another countable unit, so a unit-swap test never "changes" to what is already set.
func otherQuantityUnitID(t *testing.T, currentID string) string {
	t.Helper()

	list, status, err := apiClient.GetList(unitsPath, url.Values{"limit": {"100"}})
	require.NoError(t, err)
	require.Equal(t, 200, status, "units list should return 200")

	for _, raw := range list.Data {
		var unit map[string]any
		require.NoError(t, json.Unmarshal(raw, &unit))
		id := jsonField(unit, "id")
		if id != "" && id != currentID && jsonField(unit, "type") == "quantity" {
			return id
		}
	}
	require.FailNow(t, "the account needs a second countable unit to switch to")
	return ""
}

// Packing creates each case's freight amount and weight quantities, and their unit ids are resolved
// from the unit table rather than written into the service. A wrong id strands the quantity: the
// case then drops out of the shipping_cases expansion, which inner-joins the unit, while case_count
// still counts it — so this asserts the units resolve AND that every counted case expands.
func TestPicks_Pack_CreatesCasesWithResolvedUnits(t *testing.T) {
	unpacked := firstUnpackedPickLine(t, pbPackPickID)
	require.NotEmpty(t, unpacked, "the pack pick has an unpacked line to work with")

	status, body, err := apiClient.Patch(picksPath+"/"+pbPackPickID+"/lines/"+unpacked,
		map[string]any{"quantity_value": "1"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	job := acceptPackPick(t, pbPackPickID, 2)

	results := jobResults(job)
	require.Len(t, results, 1)
	shipmentID := jobResultResourceID(results[0])
	require.NotEmpty(t, shipmentID, "the completed job names the shipment it created")

	// Both cases ride along as sub-resources of the shipment the pack produced, each named
	// with its own case number so a caller can label them without reading them back.
	var caseNames []string
	for _, raw := range jsonListData(results[0], "sub_resources") {
		if entry, ok := raw.(map[string]any); ok && jsonField(entry, "type") == "shipping_case" {
			caseNames = append(caseNames, jsonField(entry, "name"))
		}
	}
	require.Len(t, caseNames, 2, "the job reports both shipping cases it created")
	shipmentNumber := jsonField(jsonObject(results[0], "resource"), "name")
	assert.Equal(t, []string{shipmentNumber + "-1", shipmentNumber + "-2"}, caseNames,
		"cases are numbered sequentially from the shipment number")

	status, body, err = apiClient.GetListRaw(shipmentsPath+"/"+shipmentID, url.Values{"include": {"shipping_cases"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	shipment := parseJSON(body)

	cases := jsonObject(shipment, "shipping_cases")
	require.NotNil(t, cases)
	rows := jsonArray(cases, "data")
	require.Len(t, rows, 2, "both cases must expand; a stranded unit id would silently drop one")
	assert.Equal(t, float64(2), shipment["case_count"], "case_count must agree with what expands")

	row, ok := rows[0].(map[string]any)
	require.True(t, ok)
	caseID := jsonField(row, "id")

	status, body, err = apiClient.GetListRaw(shippingCasesPath+"/"+caseID,
		url.Values{"include": {"freight_weight.unit", "freight_amount.unit"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	got := parseJSON(body)

	weightUnit := jsonObject(jsonObject(got, "freight_weight"), "unit")
	require.NotNil(t, weightUnit, "the freight weight's unit must resolve")
	assert.Equal(t, "lb", jsonField(weightUnit, "abbreviation"),
		"freight is weighed in pounds — the mass base unit is grams, so this must not be resolved as a base unit")

	amountUnit := jsonObject(jsonObject(got, "freight_amount"), "unit")
	require.NotNil(t, amountUnit, "the freight amount's unit must resolve")
	assert.Equal(t, "$", jsonField(amountUnit, "abbreviation"))
}

// A short pick is normal and an over-pick is a real floor event, so the quantity is stored as
// given; capping it at the ordered amount would silently lose what was actually pulled.
func TestPicks_UpdateLine_AcceptsMoreThanOrdered(t *testing.T) {
	resetPBPick(t)

	over := pbLine2Ordered + 2
	status, body, err := apiClient.Patch(picksPath+"/"+pbPickID+"/lines/"+pbLine2ID,
		map[string]any{"quantity_value": "6"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.InDelta(t, over, readPickLineQuantities(t, pbPickID)[pbLine2ID], 0.001,
		"the picked quantity is not capped at the ordered quantity")

	resetPBPick(t)
}

// Over-picking is a real floor event — more came off the shelf than was ordered — so packing
// ships what was picked rather than capping it, and leaves nothing outstanding behind.
func TestPicks_Pack_CarriesAnOverPickedQuantity(t *testing.T) {
	t.Parallel()

	customerID := leadTimeCustomer(t, "e2e-pick-overpack", ptrInt(30), "")
	pickID := promisedOrderPick(t, customerID, 7)

	line := firstUnpackedPickLine(t, pickID)
	require.NotEmpty(t, line, "a freshly issued order's pick has an unpacked line")

	// The order asks for one; three came off the shelf.
	status, body, err := apiClient.Patch(picksPath+"/"+pickID+"/lines/"+line,
		map[string]any{"quantity_value": "3"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	linesBefore := len(readPickLineQuantities(t, pickID))

	job := acceptPackPick(t, pickID, 1)

	assert.Len(t, readPickLineQuantities(t, pickID), linesBefore,
		"an over-packed order line has nothing outstanding, so no remainder line is synthesized")

	status, body, err = apiClient.GetListRaw(picksPath+"/"+pickID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.NotNil(t, parseJSON(body)["finished_at"], "every line is packed, so the pick closes")

	results := jobResults(job)
	require.Len(t, results, 1)
	shipmentID := jobResultResourceID(results[0])
	require.NotEmpty(t, shipmentID)

	status, body, err = apiClient.GetListRaw(shipmentsPath+"/"+shipmentID, url.Values{"include": {"lines"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	shipmentLines, _ := jsonObject(parseJSON(body), "lines")["data"].([]any)
	require.Len(t, shipmentLines, 1)
	shipped, err := strconv.ParseFloat(
		jsonField(jsonObject(shipmentLines[0].(map[string]any), "quantity"), "value"), 64)
	require.NoError(t, err)
	assert.InDelta(t, 3.0, shipped, 0.001, "the shipment carries what was picked, not what was ordered")
}

// The pick line's unit is the sales order line's; a pick records how much was pulled, never in what.
// Offering the field back would let a pick relabel a quantity without converting it.
func TestPicks_UpdateLine_RejectsAQuantityUnit(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Patch(picksPath+"/"+pbPickID+"/lines/"+pbLine2ID,
		map[string]any{"quantity_unit_id": "un_01seedpair000000000"}, newIdempotencyKey())
	require.NoError(t, err)
	require.Equal(t, 400, status, "quantity_unit_id is not part of the request: %s", string(body))
}
