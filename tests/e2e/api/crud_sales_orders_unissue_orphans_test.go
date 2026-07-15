//go:build e2e

package api_test

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Orphan coverage for the issue -> unissue reversal. Issuing a sales order creates a pick,
// pick lines (each with its own quantity row), and — for every sale line backed by an item —
// a reserved inventory_issue with its own quantity row. Unissuing must delete ALL of them:
// none of these child rows are reachable through the API once the order is back to estimate,
// so any that survive are silent orphans that accumulate on every issue/unissue cycle.
//
// The reserved-issue quantity rows are the subtle case: inventory_issue.quantity_id is the
// only reference to them, so deleting the issues without first deleting those quantities
// leaves the quantity table growing unbounded. These assertions read straight from the e2e
// DB (via authDB) keyed by order id, exactly like the production-run reservation tests.

// pickCountForOrder counts the pick rows for an order (issuing creates one).
func pickCountForOrder(t *testing.T, orderID string) int {
	t.Helper()
	var n int
	err := authDB(t).QueryRow(
		`SELECT COUNT(*) FROM pick WHERE sales_order_id = ?`, orderID,
	).Scan(&n)
	require.NoError(t, err, "counting picks for order %s", orderID)
	return n
}

// pickLineCountForOrder counts pick lines for an order via the sales-order-line join, so it
// still detects orphaned pick lines even after the parent pick row is gone.
func pickLineCountForOrder(t *testing.T, orderID string) int {
	t.Helper()
	var n int
	err := authDB(t).QueryRow(
		`SELECT COUNT(*) FROM pick_line pl
		   JOIN sales_order_line sol ON sol.id = pl.sales_order_line_id
		  WHERE sol.sales_order_id = ?`,
		orderID,
	).Scan(&n)
	require.NoError(t, err, "counting pick lines for order %s", orderID)
	return n
}

// reservedIssueCountForOrder counts the reserved inventory issues for an order.
func reservedIssueCountForOrder(t *testing.T, orderID string) int {
	t.Helper()
	var n int
	err := authDB(t).QueryRow(
		`SELECT COUNT(*) FROM inventory_issue WHERE order_id = ? AND status_code = 'reserved'`,
		orderID,
	).Scan(&n)
	require.NoError(t, err, "counting reserved issues for order %s", orderID)
	return n
}

// issueChildQuantityIDs returns every quantity row id that issuing the order created: the
// pick-line quantities and the reserved-issue quantities. Captured while the order is issued
// so the test can later prove each id is gone (not orphaned) after unissue.
func issueChildQuantityIDs(t *testing.T, orderID string) []string {
	t.Helper()
	rows, err := authDB(t).Query(
		`SELECT pl.quantity_id
		   FROM pick_line pl
		   JOIN pick pk ON pk.id = pl.pick_id
		  WHERE pk.sales_order_id = ?
		 UNION ALL
		 SELECT ii.quantity_id
		   FROM inventory_issue ii
		  WHERE ii.order_id = ? AND ii.status_code = 'reserved'`,
		orderID, orderID,
	)
	require.NoError(t, err, "reading child quantity ids for order %s", orderID)
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id sql.NullString
		require.NoError(t, rows.Scan(&id))
		if id.Valid {
			ids = append(ids, id.String)
		}
	}
	require.NoError(t, rows.Err())
	return ids
}

// existingQuantityCount counts how many of the given quantity ids still exist.
func existingQuantityCount(t *testing.T, ids []string) int {
	t.Helper()
	if len(ids) == 0 {
		return 0
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	var n int
	err := authDB(t).QueryRow(
		`SELECT COUNT(*) FROM quantity WHERE id IN (`+placeholders+`)`, args...,
	).Scan(&n)
	require.NoError(t, err, "counting surviving quantity rows")
	return n
}

func TestSalesOrder_Unissue_LeavesNoOrphanedRecords(t *testing.T) {
	t.Parallel()
	orderID := createLifecycleOrder(t)

	// Issue: creates a pick, pick lines, and a reserved inventory issue for the seed
	// product's item — each with its own quantity row.
	status, body := salesOrderAction(t, orderID, "issue", false)
	requireStatus(t, 200, status, body)

	// Sanity-check the pre-condition: the child records the test is about to verify get
	// cleaned up actually exist. If issuing stopped creating them the test would pass
	// vacuously and never guard the orphan.
	require.Positive(t, pickCountForOrder(t, orderID), "issuing creates a pick")
	require.Positive(t, pickLineCountForOrder(t, orderID), "issuing creates pick lines")
	require.Positive(t, reservedIssueCountForOrder(t, orderID),
		"the seed product is item-backed, so issuing reserves inventory")

	childQuantityIDs := issueChildQuantityIDs(t, orderID)
	require.NotEmpty(t, childQuantityIDs, "issuing creates pick-line and reserved-issue quantities")

	// Unissue: must reverse everything issuing created.
	status, body = salesOrderAction(t, orderID, "unissue", false)
	requireStatus(t, 200, status, body)
	assert.Equal(t, "estimate", jsonField(parseJSON(body), "status"))

	// No pick, pick lines, or reserved issues survive.
	assert.Zero(t, pickCountForOrder(t, orderID), "unissue deletes the pick")
	assert.Zero(t, pickLineCountForOrder(t, orderID), "unissue deletes the pick lines")
	assert.Zero(t, reservedIssueCountForOrder(t, orderID), "unissue releases the reserved inventory issues")

	// The subtle case: every quantity row the issue created is gone. A reserved issue's
	// quantity is referenced only by inventory_issue.quantity_id, so deleting the issue
	// without first deleting its quantity would leave it orphaned here.
	assert.Zero(t, existingQuantityCount(t, childQuantityIDs),
		"unissue deletes the pick-line and reserved-issue quantity rows (no orphans)")
}

func TestSalesOrder_IssueUnissueCycle_DoesNotAccumulateOrphans(t *testing.T) {
	t.Parallel()
	orderID := createLifecycleOrder(t)

	// Cycling issue/unissue must be net-zero: each unissue removes exactly what its issue
	// created, and re-issuing (which pre-clears stale reservations) must not leave the
	// previous cycle's quantities behind either.
	var everCreated []string
	for i := 0; i < 3; i++ {
		status, body := salesOrderAction(t, orderID, "issue", false)
		requireStatus(t, 200, status, body)
		everCreated = append(everCreated, issueChildQuantityIDs(t, orderID)...)

		status, body = salesOrderAction(t, orderID, "unissue", false)
		requireStatus(t, 200, status, body)
	}

	require.NotEmpty(t, everCreated, "each issue creates child quantity rows")
	assert.Zero(t, pickCountForOrder(t, orderID), "no pick survives the cycle")
	assert.Zero(t, reservedIssueCountForOrder(t, orderID), "no reserved issue survives the cycle")
	assert.Zero(t, existingQuantityCount(t, everCreated),
		"no quantity row from any issue/unissue cycle is orphaned")
}
