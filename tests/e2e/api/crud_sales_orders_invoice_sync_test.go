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

// Invoice lines snapshot the order line's quantity (value + unit) into their own
// quantity row when the invoice is created. Updating a sales order line must push the
// new quantity into the invoice lines that mirror it — otherwise an issued invoice
// silently keeps billing the old quantity (and, before this fix, changing the unit on
// an order line left the invoice line showing the old unit).
//
// The sync follows legacy billing semantics (dashboard invoice.repo.ts): an invoice
// line billed either the full ordered quantity (order-created invoices, non-shipped
// items) or a partial shipped snapshot. Only lines still holding the order line's
// pre-update value follow the edit; partial snapshots must stay untouched.
//
// Uses the dedicated ORD-SYNC-001 / INV-SYNC-001 / INV-SYNC-002 seed rows
// (0014_e2e_extras.sql), which exist solely for these mutation tests. Seeded state:
// order line 25 pair @ $9/pair; INV-SYNC-001 line mirrors it (25 pair); INV-SYNC-002
// line is a partial snapshot (10 pair).

const (
	syncOrderID              = "or_01seedsyncorder0000"
	syncOrderLineID          = "orln_01seedsync_ln1_00"
	syncInvoiceID            = "iv_01seedsyncinvoice00"
	syncInvoiceLineID        = "ivln_01seedsync_ln1_00"
	syncPartialInvoiceID     = "iv_01seedsyncinvoice02"
	syncPartialInvoiceLineID = "ivln_01seedsync_ln2_00"
	pairUnitID               = "un_01seedpair000000000"
	// dozenUnitID ("un_01seeddozen00000000", abbreviation "dz") is declared in
	// crud_sales_orders_production_run_test.go and reused here for the unit flip.
)

// fetchInvoiceLineQuantity reads one invoice line's quantity off an invoice: its
// decimal value and its display value (which embeds the unit abbreviation, e.g. "25 pr").
func fetchInvoiceLineQuantity(t *testing.T, invoiceID, invoiceLineID string) (value float64, display string) {
	t.Helper()

	status, body, err := apiClient.GetListRaw(invoicesPath+"/"+invoiceID, url.Values{"include": {"lines"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	lines := jsonObject(parseJSON(body), "lines")
	require.NotNil(t, lines, "lines present with ?include=lines")
	for _, raw := range jsonArray(lines, "data") {
		line, ok := raw.(map[string]any)
		require.True(t, ok)
		if jsonField(line, "id") != invoiceLineID {
			continue
		}
		qty := jsonObject(line, "quantity")
		require.NotNil(t, qty, "invoice line has a quantity")
		v, err := strconv.ParseFloat(jsonField(qty, "value"), 64)
		require.NoError(t, err)
		return v, jsonField(qty, "display_value")
	}
	t.Fatalf("invoice line %s not found on %s", invoiceLineID, invoiceID)
	return 0, ""
}

// requirePartialLineUntouched asserts INV-SYNC-002's partial-snapshot line still holds
// its seeded 10 pair — the sync must never rewrite a partial billing snapshot.
func requirePartialLineUntouched(t *testing.T) {
	t.Helper()
	value, display := fetchInvoiceLineQuantity(t, syncPartialInvoiceID, syncPartialInvoiceLineID)
	assert.InDelta(t, 10, value, 0.0001, "partial invoice line value must not be synced")
	assert.True(t, strings.HasSuffix(display, " pr"),
		"partial invoice line unit must not be synced (display %q should end in \" pr\")", display)
}

// patchSyncOrderLineQuantity sets ORD-SYNC-001's line quantity and requires a 200.
func patchSyncOrderLineQuantity(t *testing.T, value, unitID string) {
	t.Helper()
	status, body, err := apiClient.Patch(salesOrdersPath+"/"+syncOrderID+"/lines/"+syncOrderLineID,
		map[string]any{"quantity": map[string]any{"value": value, "unit_id": unitID}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
}

func TestSalesOrderLineUpdate_SyncsInvoiceLineQuantity(t *testing.T) {
	// Deliberately not parallel: these steps mutate and assert on shared seeded rows
	// dedicated to this test, and must run in order.

	// Baseline sanity: the seeded mirror invoice line matches the order line (25 pair).
	value, display := fetchInvoiceLineQuantity(t, syncInvoiceID, syncInvoiceLineID)
	require.InDelta(t, 25, value, 0.0001, "seeded invoice line quantity")
	requirePartialLineUntouched(t)

	// 1. Change value and unit together: 25 pair → 30 dozen.
	patchSyncOrderLineQuantity(t, "30", dozenUnitID)
	value, display = fetchInvoiceLineQuantity(t, syncInvoiceID, syncInvoiceLineID)
	assert.InDelta(t, 30, value, 0.0001, "invoice line value follows the order line")
	assert.True(t, strings.HasSuffix(display, " dz"),
		"invoice line unit follows the order line (display %q should end in \" dz\")", display)
	requirePartialLineUntouched(t)

	// 2. Unit-only change (the originally-reported bug): 30 dozen → 30 pair.
	patchSyncOrderLineQuantity(t, "30", pairUnitID)
	value, display = fetchInvoiceLineQuantity(t, syncInvoiceID, syncInvoiceLineID)
	assert.InDelta(t, 30, value, 0.0001, "value untouched by a unit-only change")
	assert.True(t, strings.HasSuffix(display, " pr"),
		"invoice line unit follows a unit-only order line change (display %q should end in \" pr\")", display)
	requirePartialLineUntouched(t)

	// 3. Value-only change: 30 pair → 12 pair.
	patchSyncOrderLineQuantity(t, "12", pairUnitID)
	value, display = fetchInvoiceLineQuantity(t, syncInvoiceID, syncInvoiceLineID)
	assert.InDelta(t, 12, value, 0.0001, "invoice line value follows a value-only change")
	assert.True(t, strings.HasSuffix(display, " pr"), "unit unchanged on a value-only change (display %q)", display)
	requirePartialLineUntouched(t)
}
