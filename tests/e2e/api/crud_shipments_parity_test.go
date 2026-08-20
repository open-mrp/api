//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Response-parity gate for the fields the dashboard renders. One assertion per WIP section-3 gap
// that the v2 collapse filled, so a projection or presenter change that drops one fails here
// rather than as a blank cell on the shipping page. Read-only.

// The full include set the shipping page requests, minus shipping_cases, which only the detail
// RPC expands.
var shipmentPageIncludes = []string{
	"related.sales_order", "customer", "freight", "shipping_address", "shipped_by",
	"related.invoice", "related.pick", "lines", "lines.sales_order_line", "lines.sales_order_line.product",
}

func TestShipmentsParity_ListRowCarriesEveryColumnTheTableRenders(t *testing.T) {
	t.Parallel()

	params := url.Values{"limit": {"25"}}
	for _, inc := range shipmentPageIncludes {
		params.Add("include", inc)
	}
	status, body, err := apiClient.GetListRaw(shipmentsPath, params)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	rows := parseJSON(body)["data"].([]any)
	require.NotEmpty(t, rows)

	var sawCases bool
	for _, r := range rows {
		row := r.(map[string]any)

		// Base scalars — computed server-side, so the row never has to expand shipping_cases.
		assert.Contains(t, []any{"low", "normal", "high"}, row["priority"], "priority must be a code")
		assert.NotNil(t, row["case_count"], "case_count must be present on a list row")
		assert.NotNil(t, row["is_ready_to_ship"], "is_ready_to_ship must be present on a list row")
		assert.Contains(t, []any{"packed", "shipped"}, row["status"])

		if row["case_count"].(float64) > 0 {
			sawCases = true
		}

		// Relations the row renders: customer cell, carrier pill, order link.
		require.NotNil(t, row["customer"], "the customer cell needs the customer include")
		assert.NotEmpty(t, jsonField(row["customer"].(map[string]any), "name"))
		require.NotNil(t, row["freight"], "the carrier pill reads freight.carrier")
		assert.NotNil(t, row["freight"].(map[string]any)["carrier"])
		related := jsonObject(row, "related")
		require.NotNil(t, related, "the row links to its order")
		assert.NotEmpty(t, jsonField(jsonObject(related, "sales_order"), "id"))
		require.NotNil(t, row["shipping_address"], "the customer cell renders the ship-to address")
	}
	assert.True(t, sawCases, "fixture should include at least one cased shipment so case_count is exercised")
}

func TestShipmentsParity_IsReadyToShipTracksCaseWeights(t *testing.T) {
	t.Parallel()

	// SHP-SB-001 is cased with non-zero freight weights; SHP-003 has no cases at all.
	ready := readShipment(t, sbShipmentID)
	assert.Equal(t, true, ready["is_ready_to_ship"], "cased and weighed shipment is ready")
	assert.Greater(t, ready["case_count"].(float64), 0.0)

	uncased := readShipment(t, "sh_01k0a87w33emw8pmkz1mf86cg2") // SHP-003, no cases
	assert.Equal(t, false, uncased["is_ready_to_ship"], "a shipment with no cases is never ready")
	assert.Equal(t, 0.0, uncased["case_count"])
}

func TestShipmentsParity_DetailCarriesLineSkuAndDescription(t *testing.T) {
	t.Parallel()

	shipment := readShipment(t, SeedShipmentID, shipmentPageIncludes...)
	lines := shipment["lines"].(map[string]any)["data"].([]any)
	require.NotEmpty(t, lines, "the lines card needs lines")

	for _, l := range lines {
		line := l.(map[string]any)
		require.NotNil(t, line["sales_order_line"], "the lines card reads orderLine.product.sku")

		orderLine := line["sales_order_line"].(map[string]any)
		assert.NotEmpty(t, jsonField(orderLine, "id"), "the order line resolves to a real row")
		assert.Greater(t, orderLine["line_item_number"].(float64), 0.0, "line_item_number backs the card's line ordering")
	}
}

func TestShipmentsParity_DetailCarriesTheHeaderRelations(t *testing.T) {
	t.Parallel()

	shipment := readShipment(t, SeedShipmentID, append(shipmentPageIncludes, "shipping_cases")...)

	// The detail header links to the order and the pick, and shows the shipping address.
	related := jsonObject(shipment, "related")
	require.NotNil(t, related)
	assert.NotEmpty(t, jsonField(jsonObject(related, "sales_order"), "number"))
	require.NotNil(t, jsonObject(related, "pick"), "the header has a See Pick button")
	require.NotNil(t, shipment["shipping_address"])

	// Carrier code backs the tracking-number URL the header builds.
	require.NotNil(t, shipment["freight"])
	carrier := shipment["freight"].(map[string]any)["carrier"].(map[string]any)
	assert.NotEmpty(t, jsonField(carrier, "id"))

	// The cases card renders freight weight and amount per case.
	cases := shipment["shipping_cases"].(map[string]any)["data"].([]any)
	require.NotEmpty(t, cases)
	for _, c := range cases {
		sc := c.(map[string]any)
		assert.NotEmpty(t, jsonField(sc, "number"))
		assert.NotNil(t, sc["freight_weight"], "the cases card edits freight weight")
		assert.NotNil(t, sc["freight_amount"], "the cases card edits freight amount")
	}
}

func TestShipmentsParity_ListAndDetailAgreeOnTheSameShipment(t *testing.T) {
	t.Parallel()

	// The collapse's whole point: one resource, one shape. A field that differs between the two
	// means the list projection has drifted from the detail again.
	params := url.Values{"limit": {"25"}}
	for _, inc := range shipmentPageIncludes {
		params.Add("include", inc)
	}
	status, body, err := apiClient.GetListRaw(shipmentsPath, params)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	var row map[string]any
	for _, r := range parseJSON(body)["data"].([]any) {
		if jsonField(r.(map[string]any), "id") == SeedShipmentID {
			row = r.(map[string]any)
			break
		}
	}
	require.NotNil(t, row, "seed shipment must appear in the unfiltered list")

	detail := readShipment(t, SeedShipmentID, shipmentPageIncludes...)
	for _, field := range []string{
		"id", "object", "number", "status", "priority", "case_count", "is_ready_to_ship",
		"note", "bill_of_lading", "master_tracking_number", "shipped_at", "created_at",
	} {
		assert.Equal(t, detail[field], row[field], "list and detail disagree on %q", field)
	}
}
