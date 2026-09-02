//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Behavioural coverage for the shipment write surface: ship, void, update, delete and line CRUD.
// These MUTATE state, so they run against the dedicated SHP-SB rows seeded in 0014_e2e_extras.sql
// rather than the shared SHP-001/2/3 the include tests read.
//
// SHP-SB-001 is cased and weighed, so it is the only fixture that can actually ship. Ship and void
// are inverses, so the tests that use it restore it; they are deliberately NOT parallel.
const (
	sbShipmentID    = "sh_01seedsbship00000"
	sbOrderID       = "or_01seedsborder00000"
	sbPickID        = "pk_01seedsbpick000000"
	sbOrderLine1ID  = "orln_01seedsb_ln1_000"
	sbOrderLine2ID  = "orln_01seedsb_ln2_000"
	sbShipmentLine1 = "shln_01seedsb_ln1_000"
	// Ordered on line 1, and how much of it SHP-SB-001 already ships.
	sbLine1Ordered = 10.0
	sbLine1Shipped = 6.0

	// Delete cascades and cannot be undone, so it gets its own shipment.
	sbDeleteShipmentID = "sh_01seedsb2ship0000"
	sbDeletePickLineID = "pkln_01seedsb2_ln1_00"

	// The unit the SB fixtures' quantities are denominated in.
	sbQuantityUnitID = "un_01seedpair000000000"
)

func readShipment(t *testing.T, id string, includes ...string) map[string]any {
	t.Helper()
	params := url.Values{}
	for _, inc := range includes {
		params.Add("include", inc)
	}
	status, body, err := apiClient.GetListRaw(shipmentsPath+"/"+id, params)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return parseJSON(body)
}

// Returns SHP-SB-001 to packed so the next test in this file starts from the seeded state.
func restoreShipmentSB(t *testing.T) {
	t.Helper()
	if readShipment(t, sbShipmentID)["shipped_at"] == nil {
		return
	}
	status, body, err := apiClient.Post(shipmentsPath+"/"+sbShipmentID+"/actions/void", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
}

func TestShipmentsBehavioral_ShipMarksShippedAndAssignsSSCC(t *testing.T) {
	defer restoreShipmentSB(t)

	before := readShipment(t, sbShipmentID, "shipping_cases")
	require.Equal(t, "packed", before["status"], "fixture must start packed")
	require.Equal(t, true, before["is_ready_to_ship"], "fixture must be shippable: cased and weighed")

	status, body, err := apiClient.Post(shipmentsPath+"/"+sbShipmentID+"/actions/ship",
		map[string]any{"email_customer": false}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	shipped := parseJSON(body)
	assert.Equal(t, "shipped", shipped["status"])
	assert.NotNil(t, shipped["shipped_at"], "shipped_at must be stamped")
	assert.Equal(t, false, shipped["is_ready_to_ship"], "a shipped shipment is no longer ready to ship")

	// Every case gets an SSCC and a shipped stamp.
	after := readShipment(t, sbShipmentID, "shipping_cases")
	cases := after["shipping_cases"].(map[string]any)["data"].([]any)
	require.Len(t, cases, 2)
	for _, c := range cases {
		sc := c.(map[string]any)
		assert.NotEmpty(t, sc["sscc"], "ship must assign an SSCC to every case")
		assert.NotNil(t, sc["shipped_at"], "ship must stamp every case")
	}
}

func TestShipmentsBehavioral_ShipTwiceConflicts(t *testing.T) {
	defer restoreShipmentSB(t)

	status, body, err := apiClient.Post(shipmentsPath+"/"+sbShipmentID+"/actions/ship",
		map[string]any{"email_customer": false}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	status, body, err = apiClient.Post(shipmentsPath+"/"+sbShipmentID+"/actions/ship",
		map[string]any{"email_customer": false}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 409, status, body)
}

func TestShipmentsBehavioral_VoidReturnsToPackedAndClearsCases(t *testing.T) {
	defer restoreShipmentSB(t)

	status, body, err := apiClient.Post(shipmentsPath+"/"+sbShipmentID+"/actions/ship",
		map[string]any{"email_customer": false}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	status, body, err = apiClient.Post(shipmentsPath+"/"+sbShipmentID+"/actions/void", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	voided := parseJSON(body)
	assert.Equal(t, "packed", voided["status"], "void returns the shipment to packed")
	assert.Nil(t, voided["shipped_at"], "void clears shipped_at")

	after := readShipment(t, sbShipmentID, "shipping_cases")
	cases := after["shipping_cases"].(map[string]any)["data"].([]any)
	require.Len(t, cases, 2)
	for _, c := range cases {
		sc := c.(map[string]any)
		assert.Nil(t, sc["shipped_at"], "void clears each case's shipped stamp")
		assert.Nil(t, sc["tracking_number"], "void clears per-case tracking")
		// SSCCs are deliberately kept — they identify the physical case, not the dispatch.
		assert.NotEmpty(t, sc["sscc"], "void keeps the SSCC")
	}
}

func TestShipmentsBehavioral_VoidUnshippedConflicts(t *testing.T) {
	require.Nil(t, readShipment(t, sbShipmentID)["shipped_at"], "fixture must start packed")

	status, body, err := apiClient.Post(shipmentsPath+"/"+sbShipmentID+"/actions/void", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 409, status, body)
}

func TestShipmentsBehavioral_UpdateLeavesUntouchedFieldsAlone(t *testing.T) {
	before := readShipment(t, sbShipmentID, "freight")
	freightBefore := before["freight"].(map[string]any)

	status, body, err := apiClient.Patch(shipmentsPath+"/"+sbShipmentID,
		map[string]any{"note": "behavioral note"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	after := readShipment(t, sbShipmentID, "freight")
	assert.Equal(t, "behavioral note", after["note"])
	assert.Equal(t, before["number"], after["number"], "number must be untouched")
	assert.Equal(t, before["master_tracking_number"], after["master_tracking_number"])

	// A PATCH that omits service_level_id must not null it (the service backfills from existing).
	freightAfter := after["freight"].(map[string]any)
	assert.Equal(t, freightBefore["service_level"], freightAfter["service_level"],
		"omitting service_level_id must not clear the service level")
}

// Returns the shipment's current service level id, or "" when it carries none.
func shipmentServiceLevelID(t *testing.T, shipmentID string) string {
	t.Helper()

	freight := jsonObject(readShipment(t, shipmentID, "freight"), "freight")
	require.NotNil(t, freight, "freight must be expanded")
	sl := jsonObject(freight, "service_level")
	if sl == nil {
		return ""
	}
	return jsonField(sl, "id")
}

// Pins the three states of the shipment PATCH's service_level_id: omitted keeps, a value sets, and an
// explicit null clears. The omitted case is the one that regresses — the column is assigned outright.
func TestShipmentsBehavioral_UpdateServiceLevelIsThreeState(t *testing.T) {
	// Not parallel: mutates SHP-SB-001's routing and restores it.
	original := shipmentServiceLevelID(t, sbShipmentID)
	require.NotEmpty(t, original, "fixture must start with a service level for the clear to mean anything")

	patch := func(body map[string]any) {
		t.Helper()
		status, resp, err := apiClient.Patch(shipmentsPath+"/"+sbShipmentID, body, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, status, resp)
	}
	t.Cleanup(func() { patch(map[string]any{"service_level_id": original}) })

	// Omitted: an unrelated field must not take the service level down with it.
	patch(map[string]any{"note": "three-state check"})
	assert.Equal(t, original, shipmentServiceLevelID(t, sbShipmentID),
		"a PATCH that omits service_level_id must leave it alone")

	// Explicit null: clears.
	patch(map[string]any{"service_level_id": nil})
	assert.Empty(t, shipmentServiceLevelID(t, sbShipmentID),
		"an explicit null must clear the service level")

	// Value: sets it back.
	patch(map[string]any{"service_level_id": original})
	assert.Equal(t, original, shipmentServiceLevelID(t, sbShipmentID),
		"sending an id must set the service level")
}

func TestShipmentsBehavioral_ShippedShipmentCannotBeRerouted(t *testing.T) {
	defer restoreShipmentSB(t)

	status, body, err := apiClient.Post(shipmentsPath+"/"+sbShipmentID+"/actions/ship",
		map[string]any{"email_customer": false}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	// Correcting the tracking number after dispatch is legitimate.
	status, body, err = apiClient.Patch(shipmentsPath+"/"+sbShipmentID,
		map[string]any{"master_tracking_number": "1Z-CORRECTED"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	// Re-routing it is not: the label was bought against the original carrier.
	status, body, err = apiClient.Patch(shipmentsPath+"/"+sbShipmentID,
		map[string]any{"carrier_id": "will_call"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 409, status, body)
}

func TestShipmentsBehavioral_LineRejectsForeignOrderLine(t *testing.T) {
	t.Parallel()

	// SeedSalesOrderLineID belongs to ORD-001, not to SHP-SB-001's order.
	status, body, err := apiClient.Post(shipmentsPath+"/"+sbShipmentID+"/lines", map[string]any{
		"sales_order_line_id": SeedSalesOrderLineID,
		"quantity_value":      "1",
		"quantity_unit_id":    sbQuantityUnitID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	assert.Contains(t, string(body), "does not belong to this shipment's sales order")
}

func TestShipmentsBehavioral_LineRejectsQuantityOverRemaining(t *testing.T) {
	t.Parallel()

	// Line 1 is ordered 10 and SHP-SB-001 already ships 6, so 5 more overshoots by 1.
	status, body, err := apiClient.Post(shipmentsPath+"/"+sbShipmentID+"/lines", map[string]any{
		"sales_order_line_id": sbOrderLine1ID,
		"quantity_value":      "5",
		"quantity_unit_id":    sbQuantityUnitID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	assert.Contains(t, string(body), "remaining to ship")
}

func TestShipmentsBehavioral_LineCreateUpdateDeleteRoundTrip(t *testing.T) {
	// Not parallel: it adds a line to the shared fixture and removes it again.
	status, body, err := apiClient.Post(shipmentsPath+"/"+sbShipmentID+"/lines", map[string]any{
		"sales_order_line_id": sbOrderLine2ID,
		"quantity_value":      "1",
		"quantity_unit_id":    sbQuantityUnitID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	created := parseJSON(body)
	lineID := jsonField(created, "id")
	require.NotEmpty(t, lineID)

	defer func() {
		status, body, err := apiClient.Delete(shipmentsPath + "/" + sbShipmentID + "/lines/" + lineID)
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
	}()

	// Line 2 is ordered 4 and nothing else ships it, so the new line may be raised to the full 4.
	status, body, err = apiClient.Patch(shipmentsPath+"/"+sbShipmentID+"/lines/"+lineID,
		map[string]any{"quantity_value": "4"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	// Excluding its own row is what allows that; one more unit is over.
	status, body, err = apiClient.Patch(shipmentsPath+"/"+sbShipmentID+"/lines/"+lineID,
		map[string]any{"quantity_value": "5"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}

func TestShipmentsBehavioral_DeleteCascadesAndUnpacksPick(t *testing.T) {
	// Destructive and unrepeatable, so it owns SHP-SB-002.
	before := readShipment(t, sbDeleteShipmentID, "lines", "shipping_cases")
	require.NotEmpty(t, before["lines"].(map[string]any)["data"])
	require.NotEmpty(t, before["shipping_cases"].(map[string]any)["data"])

	status, body, err := apiClient.Delete(shipmentsPath + "/" + sbDeleteShipmentID)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	status, body, err = apiClient.GetListRaw(shipmentsPath+"/"+sbDeleteShipmentID, nil)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)

	// Deleting a shipment unpacks the pick it was packed from, so the goods can be re-packed.
	status, body, err = apiClient.GetListRaw(picksPath+"/"+"pk_01seedsb2pick00000", url.Values{"include": {"lines"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	lines := parseJSON(body)["lines"].(map[string]any)["data"].([]any)
	require.Len(t, lines, 1)
	assert.Nil(t, lines[0].(map[string]any)["packed_at"], "delete must clear the pick line's packed stamp")

	// The whole order line comes back, not just the packed stamp: the goods are unshipped again.
	assert.InDelta(t, 5.0, readPickLineQuantities(t, "pk_01seedsb2pick00000")[sbDeletePickLineID], 0.001,
		"delete must restore the pick line to the order line's full unshipped quantity")
}

// Proves a deleted partial shipment folds its backorder line into the reopened one, rather than
// leaving both behind to double-count the outstanding goods. SHP-SB-004 packs 6 of 10.
func TestShipmentsBehavioral_DeleteFoldsBackorderLineIntoTheReopenedOne(t *testing.T) {
	// Destructive and unrepeatable, so it owns SHP-SB-004.
	const (
		sb4ShipmentID  = "sh_01seedsb4ship0000"
		sb4PickID      = "pk_01seedsb4pick00000"
		sb4PackedLine  = "pkln_01seedsb4_ln1_00"
		sb4BackorderLn = "pkln_01seedsb4_ln2_00"
	)

	before := readPickLineQuantities(t, sb4PickID)
	require.Len(t, before, 2, "fixture must start with a packed line and a backorder line")
	require.InDelta(t, 6.0, before[sb4PackedLine], 0.001)
	require.InDelta(t, 4.0, before[sb4BackorderLn], 0.001)

	status, body, err := apiClient.Delete(shipmentsPath + "/" + sb4ShipmentID)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	after := readPickLineQuantities(t, sb4PickID)
	require.Len(t, after, 1, "the backorder line must be deleted, leaving one open line per order line")
	assert.NotContains(t, after, sb4BackorderLn, "the backorder line is the one that goes")
	assert.InDelta(t, 10.0, after[sb4PackedLine], 0.001,
		"the reopened line absorbs the backorder quantity, restoring the full ordered 10")

	status, body, err = apiClient.GetListRaw(picksPath+"/"+sb4PickID, url.Values{"include": {"lines"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	remaining := parseJSON(body)["lines"].(map[string]any)["data"].([]any)
	require.Len(t, remaining, 1)
	assert.Nil(t, remaining[0].(map[string]any)["packed_at"], "the surviving line must be open again")
}

// --- invoice-on-ship (item 1) ---

const (
	// SHP-SB-003 ships its order in full (one sale line ordered 5, shipped 5) plus a freight line,
	// so shipping it creates the invoice AND marks the order fulfilled.
	sbFullShipmentID = "sh_01seedsb3ship0000"
	sbFullOrderID    = "or_01seedsb3order0000"
)

// Reads the sales order's status straight from the API.
func readOrderStatus(t *testing.T, orderID string) string {
	t.Helper()
	status, body, err := apiClient.GetListRaw(salesOrdersPath+"/"+orderID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return jsonField(parseJSON(body), "status")
}

func TestShipmentsBehavioral_ShipCreatesInvoiceForShippedGoods(t *testing.T) {
	defer restoreShipmentSB(t)

	status, body, err := apiClient.Post(shipmentsPath+"/"+sbShipmentID+"/actions/ship",
		map[string]any{"email_customer": false}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	// The invoice is linked to the shipment and numbered after it.
	shipment := readShipment(t, sbShipmentID, "related.invoice")
	invoice := jsonObject(jsonObject(shipment, "related"), "invoice")
	require.NotNil(t, invoice, "ship must create and link an invoice")
	assert.Equal(t, "SHP-SB-001", jsonField(invoice, "number"), "invoice number is the shipment number")

	// A partial shipment does not fulfill the order.
	assert.Equal(t, "issued", readOrderStatus(t, sbOrderID), "a partial shipment must not fulfill the order")
}

func TestShipmentsBehavioral_ShippingWholeOrderFulfillsBillsFreightAndEmails(t *testing.T) {
	// Not restored: this fixture is single-use (shipping it fulfills the order). Runs alone.
	require.Equal(t, "issued", readOrderStatus(t, sbFullOrderID), "fixture must start issued")

	status, body, err := apiClient.Post(shipmentsPath+"/"+sbFullShipmentID+"/actions/ship",
		map[string]any{"email_customer": true}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	shipment := readShipment(t, sbFullShipmentID, "related.invoice")
	linked := jsonObject(jsonObject(shipment, "related"), "invoice")
	require.NotNil(t, linked)
	invoiceID := jsonField(linked, "id")

	// The invoice bills the shipped sale line AND the non-shipping freight line.
	status, body, err = apiClient.GetListRaw(invoicesPath+"/"+invoiceID, url.Values{"include": {"lines"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	invoice := parseJSON(body)
	lines := invoice["lines"].(map[string]any)["data"].([]any)
	assert.Len(t, lines, 2, "invoice must bill the shipped good plus the non-shipping freight line")

	// email_customer:true with an invoice contact on the order emails the invoice (has_been_sent).
	assert.Equal(t, true, invoice["has_been_sent"], "email_customer must email the invoice")

	// Every sale line is now invoiced, so the order is fulfilled.
	assert.Equal(t, "fulfilled", readOrderStatus(t, sbFullOrderID),
		"shipping the whole order must mark it fulfilled")
}

func TestShipmentsBehavioral_ShipWithoutEmailDoesNotSendInvoice(t *testing.T) {
	defer restoreShipmentSB(t)

	status, body, err := apiClient.Post(shipmentsPath+"/"+sbShipmentID+"/actions/ship",
		map[string]any{"email_customer": false}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	invoiceID := jsonField(jsonObject(jsonObject(readShipment(t, sbShipmentID, "related.invoice"), "related"), "invoice"), "id")
	status, body, err = apiClient.GetListRaw(invoicesPath+"/"+invoiceID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, false, parseJSON(body)["has_been_sent"],
		"email_customer:false must not email the invoice")
}

func TestShipmentsBehavioral_VoidDeletesTheInvoiceAndReopensTheOrder(t *testing.T) {
	defer restoreShipmentSB(t)

	status, body, err := apiClient.Post(shipmentsPath+"/"+sbShipmentID+"/actions/ship",
		map[string]any{"email_customer": false}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	invoiceID := jsonField(jsonObject(jsonObject(readShipment(t, sbShipmentID, "related.invoice"), "related"), "invoice"), "id")

	status, body, err = apiClient.Post(shipmentsPath+"/"+sbShipmentID+"/actions/void", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	// The invoice ship created is gone, and the shipment no longer points at one.
	status, getBody, err := apiClient.GetListRaw(invoicesPath+"/"+invoiceID, nil)
	require.NoError(t, err)
	requireStatus(t, 404, status, getBody)
	assert.Nil(t, readShipment(t, sbShipmentID, "related.invoice")["invoice"], "void unlinks the invoice")
}
