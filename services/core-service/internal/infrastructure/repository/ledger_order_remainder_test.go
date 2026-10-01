//go:build ledger

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/services/core-service/internal/ledgerlock"
)

// Reopening a closed order reserves what it has not yet shipped, and only that. The remainder is
// measured against every issue the order holds: shipped issues are open or closed, and an order closed
// before close released anything still carries its balance as a reserved issue.
func TestGetUnreservedRemainders(t *testing.T) {
	f := newFixture(t)
	base := time.Now().UTC().Add(-time.Hour).Truncate(time.Millisecond)
	repo := NewSalesOrderRepo(sqlc.New(f.db))
	saleProduct := f.insertProduct(t, f.itemID, "sale")

	remainders := func(orderID string) []domain.SalesOrderItemRemainder {
		t.Helper()
		got, apiErr := repo.GetUnreservedRemainders(context.Background(), f.accountID, orderID)
		require.Nil(t, apiErr)
		return got
	}
	requireRemainder := func(got []domain.SalesOrderItemRemainder, unitID, want string) {
		t.Helper()
		require.Len(t, got, 1)
		require.Equal(t, f.itemID, got[0].ItemID)
		require.Equal(t, unitID, got[0].UnitID)
		value, err := decimal.NewFromString(got[0].RemainingValue)
		require.NoError(t, err)
		require.True(t, value.Equal(decimal.RequireFromString(want)), "remaining %s, want %s", value, want)
	}

	t.Run("short-shipped balance", func(t *testing.T) {
		orderID := f.nextID("or")
		f.insertOrderLine(t, orderID, saleProduct, f.itemID, 1, "1900", f.pair)
		f.insertOrderIssue(t, orderID, "closed", "1570", f.pair, base)

		requireRemainder(remainders(orderID), f.pair, "330")
	})

	t.Run("balance still reserved by an old close", func(t *testing.T) {
		orderID := f.nextID("or")
		f.insertOrderLine(t, orderID, saleProduct, f.itemID, 1, "1900", f.pair)
		f.insertOrderIssue(t, orderID, "closed", "1570", f.pair, base)
		f.insertOrderIssue(t, orderID, "reserved", "330", f.pair, base)

		require.Empty(t, remainders(orderID), "the surviving reservation already covers the balance")
	})

	t.Run("lines in mixed units, expressed in the first line's unit", func(t *testing.T) {
		orderID := f.nextID("or")
		f.insertOrderLine(t, orderID, saleProduct, f.itemID, 1, "10", f.pair)
		f.insertOrderLine(t, orderID, saleProduct, f.itemID, 2, "10", f.each)
		// 30 each ordered, 4 each shipped: 26 each is 13 pair.
		f.insertOrderIssue(t, orderID, "open", "4", f.each, base)

		requireRemainder(remainders(orderID), f.pair, "13")
	})

	t.Run("over-shipped line", func(t *testing.T) {
		orderID := f.nextID("or")
		f.insertOrderLine(t, orderID, saleProduct, f.itemID, 1, "10", f.pair)
		f.insertOrderIssue(t, orderID, "closed", "12", f.pair, base)

		require.Empty(t, remainders(orderID))
	})

	t.Run("non-sale lines reserve nothing", func(t *testing.T) {
		orderID := f.nextID("or")
		otherItem := f.itemID + "_svc"
		f.insertOrderLine(t, orderID, f.insertProduct(t, otherItem, "service"), otherItem, 1, "5", f.each)

		require.Empty(t, remainders(orderID))
	})
}

func (f *fixture) insertProduct(t *testing.T, itemID, typeCode string) string {
	t.Helper()
	productID := f.nextID("pd")
	_, err := f.db.Exec(
		`INSERT INTO product (id, item_id, product_type_code, created_at, updated_at) VALUES (?, ?, ?, NOW(3), NOW(3))`,
		productID, itemID, typeCode)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = f.db.Exec(`DELETE FROM product WHERE id = ?`, productID) })
	return productID
}

func (f *fixture) insertOrderLine(t *testing.T, orderID, productID, itemID string, lineNumber int, value, unitID string) {
	t.Helper()
	qID, lineID, priceID := f.nextID("qy"), f.nextID("orln"), f.nextID("rt")
	_, err := f.db.Exec(
		`INSERT INTO quantity (id, value, unit_id, created_at, updated_at) VALUES (?, ?, ?, NOW(3), NOW(3))`,
		qID, value, unitID)
	require.NoError(t, err)
	// unit_price_id is unique but has no foreign key, and the query never reads it.
	_, err = f.db.Exec(
		`INSERT INTO sales_order_line (id, product_sku, line_item_number, product_id, sales_order_id, quantity_id, unit_price_id, item_id, created_at, updated_at)
		 VALUES (?, 'LEDGERTEST', ?, ?, ?, ?, ?, ?, NOW(3), NOW(3))`,
		lineID, lineNumber, productID, orderID, qID, priceID, itemID)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = f.db.Exec(`DELETE FROM sales_order_line WHERE id = ?`, lineID)
		_, _ = f.db.Exec(`DELETE FROM quantity WHERE id = ?`, qID)
	})
}

func (f *fixture) insertOrderIssue(t *testing.T, orderID, status, value, unitID string, createdAt time.Time) {
	t.Helper()
	qID, iID := f.nextID("qy"), f.nextID("ivis")
	_, err := f.db.Exec(
		`INSERT INTO quantity (id, value, unit_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		qID, value, unitID, createdAt, createdAt)
	require.NoError(t, err)
	_, err = f.db.Exec(
		`INSERT INTO inventory_issue (id, account_id, item_id, status_code, quantity_id, order_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		iID, f.accountID, f.itemID, status, qID, orderID, createdAt, createdAt)
	require.NoError(t, err)
}

// A line edit on an issued order releases only what the order has reserved beyond what it still has
// to ship: ordered on its sale lines, less what has shipped. An item no line carries is all excess.
func TestGetExcessReservedItemIDs(t *testing.T) {
	f := newFixture(t)
	base := time.Now().UTC().Add(-time.Hour).Truncate(time.Millisecond)
	repo := NewSalesOrderRepo(sqlc.New(f.db))
	saleProduct := f.insertProduct(t, f.itemID, "sale")

	excess := func(orderID string) []string {
		t.Helper()
		got, apiErr := repo.GetExcessReservedItemIDs(context.Background(), f.accountID, orderID)
		require.Nil(t, apiErr)
		return got
	}

	t.Run("reservation matches the line", func(t *testing.T) {
		orderID := f.nextID("or")
		f.insertOrderLine(t, orderID, saleProduct, f.itemID, 1, "10", f.pair)
		f.insertOrderIssue(t, orderID, "reserved", "20", f.each, base)

		require.Empty(t, excess(orderID), "20 each is the 10 pair the line orders")
	})

	t.Run("line cut below its reservation", func(t *testing.T) {
		orderID := f.nextID("or")
		f.insertOrderLine(t, orderID, saleProduct, f.itemID, 1, "8", f.pair)
		f.insertOrderIssue(t, orderID, "reserved", "10", f.pair, base)

		require.Equal(t, []string{f.itemID}, excess(orderID))
	})

	t.Run("partly shipped, balance reserved", func(t *testing.T) {
		orderID := f.nextID("or")
		f.insertOrderLine(t, orderID, saleProduct, f.itemID, 1, "10", f.pair)
		f.insertOrderIssue(t, orderID, "closed", "6", f.pair, base)
		f.insertOrderIssue(t, orderID, "reserved", "4", f.pair, base)

		require.Empty(t, excess(orderID))
	})

	t.Run("line removed", func(t *testing.T) {
		orderID := f.nextID("or")
		f.insertOrderIssue(t, orderID, "reserved", "1", f.pair, base)

		require.Equal(t, []string{f.itemID}, excess(orderID))
	})
}

// Releasing some of an order's items leaves its other reservations exactly as they were.
func TestReleaseReservedIssuesForOrderItems_LeavesOtherItemsAlone(t *testing.T) {
	f := newFixture(t)
	base := time.Now().UTC().Add(-time.Hour).Truncate(time.Millisecond)
	orderID := f.nextID("or")

	released := f.insertReservedIssueForOrder(t, orderID, "4", f.each, base)

	otherItem := f.itemID + "_other"
	otherQty, otherIssue := f.nextID("qy"), f.nextID("ivis")
	_, err := f.db.Exec(`INSERT INTO quantity (id, value, unit_id, created_at, updated_at) VALUES (?, '5', ?, NOW(3), NOW(3))`, otherQty, f.each)
	require.NoError(t, err)
	_, err = f.db.Exec(`INSERT INTO inventory_issue (id, account_id, item_id, status_code, quantity_id, order_id, created_at, updated_at)
		VALUES (?, ?, ?, 'reserved', ?, ?, NOW(3), NOW(3))`, otherIssue, f.accountID, otherItem, otherQty, orderID)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = f.db.Exec(`DELETE FROM inventory_issue WHERE id = ?`, otherIssue)
		_, _ = f.db.Exec(`DELETE FROM quantity WHERE id = ?`, otherQty)
		_, _ = f.db.Exec(`DELETE FROM inventory_item_lock WHERE item_id = ?`, otherItem)
	})

	a := f.actor(t, "line-edit")
	scope, apiErr := ledgerlock.Acquire(context.Background(), a.repo, []string{f.itemID})
	require.Nil(t, apiErr)
	items, apiErr := a.repo.ReleaseReservedIssuesForOrderItems(context.Background(), scope, f.accountID, orderID, []string{f.itemID})
	require.Nil(t, apiErr)
	a.commit(t)

	require.Equal(t, []string{f.itemID}, items)
	var remaining []string
	rows, err := f.db.Query(`SELECT id FROM inventory_issue WHERE order_id = ? AND status_code = 'reserved' ORDER BY id`, orderID)
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var issueID string
		require.NoError(t, rows.Scan(&issueID))
		remaining = append(remaining, issueID)
	}
	require.Equal(t, []string{otherIssue}, remaining, "issue %s was released; the other item's reservation must survive", released)

	var qtyLeft int
	require.NoError(t, f.db.QueryRow(`SELECT COUNT(*) FROM quantity q JOIN inventory_issue ii ON ii.quantity_id = q.id WHERE ii.id = ?`, released).Scan(&qtyLeft))
	require.Zero(t, qtyLeft)
}
