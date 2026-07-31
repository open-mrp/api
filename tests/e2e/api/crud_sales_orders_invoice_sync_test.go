//go:build e2e

package api_test

import (
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Pick, shipment, and invoice lines each snapshot the order line's quantity (value +
// unit) into their own quantity row when created. Updating a sales order line must keep
// them in sync:
//   - Unit always follows on all three — a unit edit is a data correction, and the
//     pack/invoiced rollups sum these values without unit conversion.
//   - Value follows for shipment and invoice lines only while it still mirrors the
//     pre-update ordered quantity; partial snapshots keep the amount that actually
//     shipped/billed (legacy billing semantics, dashboard invoice.repo.ts).
//   - Pick line values are picking progress and are handled by pick reconciliation,
//     never by the sync.
//
// Uses the dedicated ORD-SYNC-001 seed rows (0014_e2e_extras.sql), which exist solely
// for these mutation tests. Seeded state: order line 25 pair @ $9/pair; INV-SYNC-001
// line mirrors it (25 pair); INV-SYNC-002 line is a partial snapshot (10 pair);
// PICK-SYNC-001 has one packed line (25 pair picked); SHP-SYNC-001 has one line
// mirroring the ordered quantity (25 pair).

const (
	syncOrderID              = "or_01seedsyncorder0000"
	syncOrderLineID          = "orln_01seedsync_ln1_00"
	syncInvoiceID            = "iv_01seedsyncinvoice00"
	syncInvoiceLineID        = "ivln_01seedsync_ln1_00"
	syncPartialInvoiceID     = "iv_01seedsyncinvoice02"
	syncPartialInvoiceLineID = "ivln_01seedsync_ln2_00"
	syncPickID               = "pk_01seedsyncpick00000"
	syncPickLineID           = "pkln_01seedsync_ln1_00"
	syncShipmentID           = "sh_01seedsyncship00000"
	syncShipmentLineID       = "shln_01seedsync_ln1_00"
	pairUnitID               = "un_01seedpair000000000"
	// dozenUnitID ("un_01seeddozen00000000", abbreviation "dz") is declared in
	// crud_sales_orders_production_run_test.go and reused here for the unit flip.
)

// fetchSyncLineQuantity reads one line's quantity off a resource detail response
// (`GET {path}?include=lines`): its decimal value and its display value (which embeds
// the unit abbreviation, e.g. "25 pr"). Works for invoice, pick, and shipment lines —
// all three expose `lines.data[].quantity`.
func fetchSyncLineQuantity(t *testing.T, path, lineID string) (value float64, display string) {
	t.Helper()

	status, body, err := apiClient.GetListRaw(path, url.Values{"include": {"lines"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	lines := jsonObject(parseJSON(body), "lines")
	require.NotNil(t, lines, "lines present with ?include=lines on %s", path)
	for _, raw := range jsonArray(lines, "data") {
		line, ok := raw.(map[string]any)
		require.True(t, ok)
		if jsonField(line, "id") != lineID {
			continue
		}
		qty := jsonObject(line, "quantity")
		require.NotNil(t, qty, "line has a quantity")
		v, err := strconv.ParseFloat(jsonField(qty, "value"), 64)
		require.NoError(t, err)
		return v, jsonField(qty, "display_value")
	}
	t.Fatalf("line %s not found on %s", lineID, path)
	return 0, ""
}

// requireSyncState asserts the full downstream state for the sync order line: the
// mirror invoice line and shipment line values, the partial invoice line (value fixed
// at its seeded 10), the packed pick line (value fixed at its seeded 25), and — since
// units always follow — the same unit abbreviation suffix on all four.
func requireSyncState(t *testing.T, wantMirrorValue float64, wantUnitSuffix, step string) {
	t.Helper()

	value, display := fetchSyncLineQuantity(t, invoicesPath+"/"+syncInvoiceID, syncInvoiceLineID)
	assert.InDelta(t, wantMirrorValue, value, 0.0001, "%s: mirror invoice line value follows the order line", step)
	assert.True(t, strings.HasSuffix(display, wantUnitSuffix), "%s: mirror invoice line unit (display %q, want suffix %q)", step, display, wantUnitSuffix)

	value, display = fetchSyncLineQuantity(t, invoicesPath+"/"+syncPartialInvoiceID, syncPartialInvoiceLineID)
	assert.InDelta(t, 10, value, 0.0001, "%s: partial invoice line keeps its billed amount", step)
	assert.True(t, strings.HasSuffix(display, wantUnitSuffix), "%s: partial invoice line unit still follows (display %q, want suffix %q)", step, display, wantUnitSuffix)

	value, display = fetchSyncLineQuantity(t, shipmentsPath+"/"+syncShipmentID, syncShipmentLineID)
	assert.InDelta(t, wantMirrorValue, value, 0.0001, "%s: mirror shipment line value follows the order line", step)
	assert.True(t, strings.HasSuffix(display, wantUnitSuffix), "%s: shipment line unit (display %q, want suffix %q)", step, display, wantUnitSuffix)

	value, display = fetchSyncLineQuantity(t, picksPath+"/"+syncPickID, syncPickLineID)
	assert.InDelta(t, 25, value, 0.0001, "%s: pick line keeps its picked amount (progress, not a mirror)", step)
	assert.True(t, strings.HasSuffix(display, wantUnitSuffix), "%s: pick line unit relabeled (display %q, want suffix %q)", step, display, wantUnitSuffix)
}

// patchSyncOrderLineQuantity sets ORD-SYNC-001's line quantity and requires a 200.
func patchSyncOrderLineQuantity(t *testing.T, value, unitID string) {
	t.Helper()
	status, body, err := apiClient.Patch(salesOrdersPath+"/"+syncOrderID+"/lines/"+syncOrderLineID,
		map[string]any{"quantity": map[string]any{"value": value, "unit_id": unitID}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
}

func TestSalesOrderLineUpdate_SyncsFulfillmentLineQuantities(t *testing.T) {
	// Deliberately not parallel: these steps mutate and assert on shared seeded rows
	// dedicated to this test, and must run in order. Note the order has a pick, so
	// quantity edits also exercise pick reconciliation (an increase opens a remainder
	// pick line; a decrease back below the packed amount removes it) — the seeded
	// packed line asserted here is untouched by that churn.

	// Baseline sanity: everything mirrors the seeded order line (25 pair).
	requireSyncState(t, 25, " pr", "baseline")

	// 1. Change value and unit together: 25 pair → 30 dozen.
	patchSyncOrderLineQuantity(t, "30", dozenUnitID)
	requireSyncState(t, 30, " dz", "value+unit change")

	// 2. Unit-only change (the originally-reported bug): 30 dozen → 30 pair.
	patchSyncOrderLineQuantity(t, "30", pairUnitID)
	requireSyncState(t, 30, " pr", "unit-only change")

	// 3. Value-only change: 30 pair → 12 pair.
	patchSyncOrderLineQuantity(t, "12", pairUnitID)
	requireSyncState(t, 12, " pr", "value-only change")
}
