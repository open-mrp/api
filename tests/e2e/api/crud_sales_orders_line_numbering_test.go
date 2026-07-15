//go:build e2e

package api_test

import (
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// lineRoles splits an order's line item numbers into the sale lines (product SeedProductID)
// and the single freight/system line (any other product).
type lineRoles struct {
	sale    []int
	saleIDs []string
	freight int
}

func orderLineRoles(t *testing.T, orderID string) lineRoles {
	t.Helper()
	got := getSalesOrder(t, orderID, url.Values{"include": {"lines", "lines.product"}})
	lines := jsonObject(got, "lines")
	require.NotNil(t, lines, "lines present with ?include=lines")

	var roles lineRoles
	for _, raw := range jsonArray(lines, "data") {
		line, ok := raw.(map[string]any)
		require.True(t, ok)
		num, _ := strconv.Atoi(jsonField(line, "line_item_number"))
		product := jsonObject(line, "product")
		if product != nil && jsonField(product, "id") == SeedProductID {
			roles.sale = append(roles.sale, num)
			roles.saleIDs = append(roles.saleIDs, jsonField(line, "id"))
		} else {
			roles.freight = num
		}
	}
	return roles
}

func addSaleLine(t *testing.T, orderID, sku string) {
	t.Helper()
	status, body, err := apiClient.Post(salesOrdersPath+"/"+orderID+"/lines", map[string]any{
		"product_id":  SeedProductID,
		"product_sku": sku,
		"quantity":    map[string]any{"value": "1", "unit_id": SeedUnitID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
}

func deleteSaleLine(t *testing.T, orderID, lineID string) {
	t.Helper()
	status, body, err := apiClient.Delete(salesOrdersPath + "/" + orderID + "/lines/" + lineID)
	require.NoError(t, err)
	require.GreaterOrEqual(t, status, 200, "delete line: %s", string(body))
	require.Less(t, status, 300, "delete line: %s", string(body))
}

// TestSalesOrder_LineItemNumbers_CompactOnDeleteAndReadd pins that deleting product lines
// compacts line_item_numbers (freight/credit stay at the bottom) so later additions are
// numbered contiguously from 1 — instead of leaving a hole that pushes new lines above the
// freight line. Repro: an order with 2 sale lines + 1 freight line, delete both sale lines
// (freight must renumber to 1), then add 2 new lines (must be 1, 2 with freight at 3).
func TestSalesOrder_LineItemNumbers_CompactOnDeleteAndReadd(t *testing.T) {
	t.Parallel()
	// createLifecycleOrder makes one sale line (1) + a synthesized freight line (2).
	orderID := createLifecycleOrder(t)
	addSaleLine(t, orderID, "E2E-RESEQ-A") // → sale 1, sale 2, freight 3

	start := orderLineRoles(t, orderID)
	require.Len(t, start.sale, 2, "two sale lines")
	assert.ElementsMatch(t, []int{1, 2}, start.sale, "sale lines are numbered 1 and 2")
	assert.Equal(t, 3, start.freight, "freight is at the bottom, numbered 3")

	// Delete both sale lines.
	for _, id := range start.saleIDs {
		deleteSaleLine(t, orderID, id)
	}

	// Freight is the only line left and must have compacted to 1 (the bug left it at 3).
	afterDelete := orderLineRoles(t, orderID)
	require.Empty(t, afterDelete.sale, "no sale lines remain")
	assert.Equal(t, 1, afterDelete.freight, "freight compacts to 1 after the product lines are deleted")

	// Add two new sale lines — they must be numbered 1 and 2, with freight back at 3.
	addSaleLine(t, orderID, "E2E-RESEQ-C")
	addSaleLine(t, orderID, "E2E-RESEQ-D")

	final := orderLineRoles(t, orderID)
	assert.ElementsMatch(t, []int{1, 2}, final.sale, "re-added lines are numbered 1 and 2, not 3 and 4")
	assert.Equal(t, 3, final.freight, "freight stays at the bottom, numbered 3")
}
