//go:build integration

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/pagination"
	"github.com/stretchr/testify/require"
)

// The order-book reports read from the open-order snapshot must equal the same reports read from the
// source tables. Run against the seeded dev database:
//
//	go test -tags integration ./services/core-service/internal/infrastructure/repository/ -run OpenOrderSnapshot

const (
	snapshotFixture      = "snapfx"
	snapshotOtherAccount = "ac_snapfx_other"
)

// snapshotFixtures adds orders the seed lacks, removing them at cleanup: an issued order already completed,
// whose lines have no cost rate, a missing quantity row, or no product, and whose invoice lines include one
// with a missing quantity row; and a copy of it under another account, which no report of the seed
// account may see.
func snapshotFixtures(t *testing.T, ctx context.Context, pool *sql.DB) {
	t.Helper()
	removeSnapshotFixtures(t, ctx, pool)
	t.Cleanup(func() { removeSnapshotFixtures(t, context.Background(), pool) })

	var orderID, lineID, invoiceLineID string
	require.NoError(t, pool.QueryRowContext(ctx, `SELECT so.id, sol.id FROM sales_order so JOIN sales_order_line sol ON sol.sales_order_id = so.id
WHERE so.owner_account_id = ? AND so.sales_order_type_code = 'sales_order' AND so.sales_order_status_code = 'issued' AND sol.unit_cost_id IS NOT NULL
ORDER BY so.id, sol.id LIMIT 1`, rollupTestAccountID).Scan(&orderID, &lineID), "no issued order with a costed line; seed one")
	require.NoError(t, pool.QueryRowContext(ctx, `SELECT id FROM invoice_line ORDER BY id LIMIT 1`).Scan(&invoiceLineID), "no invoice line; seed one")

	conn, err := pool.Conn(ctx)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()
	exec := func(query string, args ...any) {
		t.Helper()
		_, err := conn.ExecContext(ctx, query, args...)
		require.NoError(t, err, query)
	}
	// copyRow inserts a copy of a row under a new id, with set applied to the copy.
	copyRow := func(table, srcID, newID, set string, args ...any) {
		t.Helper()
		exec("CREATE TEMPORARY TABLE snapshot_copy AS SELECT * FROM `"+table+"` WHERE id = ?", srcID)
		exec("UPDATE snapshot_copy SET id = ?"+set, append([]any{newID}, args...)...)
		exec("INSERT INTO `" + table + "` SELECT * FROM snapshot_copy")
		exec("DROP TEMPORARY TABLE snapshot_copy")
	}
	var quantityID, priceID, costID, invoiceQuantityID string
	require.NoError(t, conn.QueryRowContext(ctx, `SELECT quantity_id, unit_price_id, unit_cost_id FROM sales_order_line WHERE id = ?`, lineID).
		Scan(&quantityID, &priceID, &costID))
	require.NoError(t, conn.QueryRowContext(ctx, `SELECT quantity_id FROM invoice_line WHERE id = ?`, invoiceLineID).Scan(&invoiceQuantityID))

	for _, account := range []string{rollupTestAccountID, snapshotOtherAccount} {
		id := func(name string) string {
			return fmt.Sprintf("%s_%s_%s", snapshotFixture, account[len(account)-5:], name)
		}
		copyRow("sales_order", orderID, id("so"), ", owner_account_id = ?, number = ?, completed_at = '2026-01-02 03:04:05.678'", account, id("no"))
		copyRow("quantity", quantityID, id("q1"), ", value = 7.5")
		copyRow("quantity", quantityID, id("q3"), ", value = 0.125")
		copyRow("quantity", invoiceQuantityID, id("iq1"), ", value = 2")
		for _, r := range []string{"p1", "p2", "p3"} {
			copyRow("rate", priceID, id(r), "")
		}
		copyRow("rate", costID, id("c2"), "")
		copyRow("sales_order_line", lineID, id("l1"), ", sales_order_id = ?, quantity_id = ?, unit_price_id = ?, unit_cost_id = NULL",
			id("so"), id("q1"), id("p1"))
		copyRow("sales_order_line", lineID, id("l2"), ", sales_order_id = ?, quantity_id = ?, unit_price_id = ?, unit_cost_id = ?",
			id("so"), id("q_missing"), id("p2"), id("c2"))
		copyRow("sales_order_line", lineID, id("l3"), ", sales_order_id = ?, quantity_id = ?, unit_price_id = ?, unit_cost_id = NULL, product_id = NULL",
			id("so"), id("q3"), id("p3"))
		copyRow("invoice_line", invoiceLineID, id("il1"), ", sales_order_line_id = ?, quantity_id = ?", id("l1"), id("iq1"))
		copyRow("invoice_line", invoiceLineID, id("il2"), ", sales_order_line_id = ?, quantity_id = ?", id("l1"), id("iq_missing"))
	}
}

func removeSnapshotFixtures(t *testing.T, ctx context.Context, pool *sql.DB) {
	t.Helper()
	for _, table := range []string{"invoice_line", "sales_order_line", "rate", "quantity", "sales_order", "open_order_invoiced", "open_order_line", "open_order"} {
		_, err := pool.ExecContext(ctx, "DELETE FROM `"+table+"` WHERE id LIKE ?", snapshotFixture+"\\_%")
		require.NoError(t, err)
	}
}

// syncSnapshot runs a complete snapshot pass, as the refresher's does.
func syncSnapshot(t *testing.T, ctx context.Context, repo domain.OpenOrderSnapshotRepo) {
	t.Helper()
	after := ""
	for {
		ids, apiErr := repo.ListIssuedOrderIDsAfter(ctx, after, 50)
		require.Nil(t, apiErr)
		_, _, apiErr = repo.RefreshOrders(ctx, ids)
		require.Nil(t, apiErr)
		if len(ids) < 50 {
			break
		}
		after = ids[len(ids)-1]
	}
	held, apiErr := repo.ListSnapshotOrderIDsAfter(ctx, "", 10_000)
	require.Nil(t, apiErr)
	stale, apiErr := repo.FilterNotIssued(ctx, held)
	require.Nil(t, apiErr)
	_, _, apiErr = repo.RefreshOrders(ctx, stale)
	require.Nil(t, apiErr)
}

func distinctOrderValues(t *testing.T, pool *sql.DB, query string) []string {
	t.Helper()
	ids, err := selectStrings(context.Background(), pool, query, rollupTestAccountID)
	require.NoError(t, err)
	require.NotEmpty(t, ids, "seed has none for %q", query)
	return ids
}

// orderBookFilters are filter combinations over the seed account's issued orders.
func orderBookFilters(t *testing.T, pool *sql.DB) map[string]domain.OpenOrderFilter {
	t.Helper()
	buyers := distinctOrderValues(t, pool, `SELECT DISTINCT buyer_account_id FROM sales_order WHERE owner_account_id = ? AND sales_order_status_code = 'issued' ORDER BY 1`)
	reps := distinctOrderValues(t, pool, `SELECT DISTINCT sales_rep_id FROM sales_order WHERE owner_account_id = ? AND sales_order_status_code = 'issued' AND sales_rep_id IS NOT NULL ORDER BY 1`)
	lines := distinctOrderValues(t, pool, `SELECT DISTINCT p.product_line_id FROM sales_order so JOIN sales_order_line sol ON sol.sales_order_id = so.id
JOIN product p ON p.id = sol.product_id WHERE so.owner_account_id = ? AND so.sales_order_status_code = 'issued' ORDER BY 1`)
	items := distinctOrderValues(t, pool, `SELECT DISTINCT p.item_id FROM sales_order so JOIN sales_order_line sol ON sol.sales_order_id = so.id
JOIN product p ON p.id = sol.product_id WHERE so.owner_account_id = ? AND so.sales_order_status_code = 'issued' ORDER BY 1`)
	groups, _ := selectStrings(context.Background(), pool, `SELECT DISTINCT account_group_id FROM account_relation WHERE owner_account_id = ? AND account_group_id IS NOT NULL`, rollupTestAccountID)

	filters := map[string]domain.OpenOrderFilter{
		"all":                  {},
		"one buyer":            {CustomerIDs: buyers[:1]},
		"two buyers":           {CustomerIDs: buyers[:min(2, len(buyers))]},
		"unknown buyer":        {CustomerIDs: []string{"ac_nobody"}},
		"rep":                  {SalesRepIDs: reps[:1]},
		"product line":         {ProductLineIDs: lines[:1]},
		"product lines":        {ProductLineIDs: lines},
		"item":                 {ItemIDs: items[:1]},
		"buyer, rep and line":  {CustomerIDs: buyers, SalesRepIDs: reps, ProductLineIDs: lines[:1]},
		"item and two buyers":  {ItemIDs: items, CustomerIDs: buyers[:min(2, len(buyers))]},
		"line no order counts": {ProductLineIDs: []string{"pl_nothing"}},
	}
	if len(groups) > 0 {
		filters["customer group"] = domain.OpenOrderFilter{CustomerGroupIDs: groups[:1]}
	}
	for name, f := range filters {
		f.AccountID = rollupTestAccountID
		filters[name] = f
	}
	return filters
}

func TestOpenOrderSnapshotMatchesTheSourceTables(t *testing.T) {
	ctx := context.Background()
	pool := testDB(t)
	pagination.Init([]byte("open-order-snapshot-integration-test"))
	queries := sqlc.New(pool)
	snapshotFixtures(t, ctx, pool)
	syncSnapshot(t, ctx, NewOpenOrderSnapshotRepo(queries))

	live := &analyticsRepoImpl{queries: queries, orderBook: orderBookLive}
	snapshot := &analyticsRepoImpl{queries: queries}
	openOrderSnapshotReady.Store(true)
	t.Cleanup(func() { openOrderSnapshotReady.Store(false) })

	for name, f := range orderBookFilters(t, pool) {
		t.Run(name, func(t *testing.T) {
			scope, empty, apiErr := snapshot.newOrderBookScope(ctx, f, true)
			require.Nil(t, apiErr)
			require.True(t, empty || scope.snapshot, "the snapshot read the live tables")
			requireSameOrderBook(t, ctx, live, snapshot, f)
		})
	}
}

// requireSameOrderBook asserts every order-book report reads the same through both repos.
func requireSameOrderBook(t *testing.T, ctx context.Context, live, snapshot *analyticsRepoImpl, f domain.OpenOrderFilter) {
	t.Helper()
	wantSummary, apiErr := live.GetOpenOrdersSummary(ctx, f)
	require.Nil(t, apiErr)
	gotSummary, apiErr := snapshot.GetOpenOrdersSummary(ctx, f)
	require.Nil(t, apiErr)
	require.Equal(t, wantSummary, gotSummary, "summary")

	for _, openOnly := range []bool{true, false} {
		want := rawOrderEntries(t, ctx, live, f, openOnly)
		got := rawOrderEntries(t, ctx, snapshot, f, openOnly)
		require.Equal(t, want, got, "order entries (open only: %v)", openOnly)
	}
	wantEntries, apiErr := live.GetOpenOrderLineEntries(ctx, f, 3)
	require.Nil(t, apiErr)
	gotEntries, apiErr := snapshot.GetOpenOrderLineEntries(ctx, f, 3)
	require.Nil(t, apiErr)
	require.Equal(t, wantEntries, gotEntries, "open line entries")
	wantAll, apiErr := live.GetOrderEntries(ctx, domain.AnalyzeOrdersParams{AccountID: f.AccountID, CustomerIDs: f.CustomerIDs,
		CustomerGroupIDs: f.CustomerGroupIDs, SalesRepIDs: f.SalesRepIDs, ProductLineIDs: f.ProductLineIDs})
	require.Nil(t, apiErr)
	gotAll, apiErr := snapshot.GetOrderEntries(ctx, domain.AnalyzeOrdersParams{AccountID: f.AccountID, CustomerIDs: f.CustomerIDs,
		CustomerGroupIDs: f.CustomerGroupIDs, SalesRepIDs: f.SalesRepIDs, ProductLineIDs: f.ProductLineIDs})
	require.Nil(t, apiErr)
	require.Equal(t, wantAll, gotAll, "order entries")

	// Pages forward to the end, then back from there.
	var cursor *string
	for page := 0; ; page++ {
		params := domain.AnalyzeOpenOrderProductsParams{OpenOrderFilter: f, Limit: 2, Cursor: cursor}
		want, apiErr := live.GetOpenOrderProducts(ctx, params)
		require.Nil(t, apiErr)
		got, apiErr := snapshot.GetOpenOrderProducts(ctx, params)
		require.Nil(t, apiErr)
		require.Equal(t, want, got, "products page %d", page)
		if !want.PageInfo.HasNextPage {
			if want.PageInfo.PrevCursor != nil {
				params.Cursor = want.PageInfo.PrevCursor
				wantPrev, apiErr := live.GetOpenOrderProducts(ctx, params)
				require.Nil(t, apiErr)
				gotPrev, apiErr := snapshot.GetOpenOrderProducts(ctx, params)
				require.Nil(t, apiErr)
				require.Equal(t, wantPrev, gotPrev, "products previous page")
			}
			break
		}
		cursor = want.PageInfo.NextCursor
	}
	cursor = nil
	for page := 0; ; page++ {
		params := domain.ListOpenOrdersParams{OpenOrderFilter: f, Limit: 3, Cursor: cursor}
		want, apiErr := live.ListOpenOrders(ctx, params)
		require.Nil(t, apiErr)
		got, apiErr := snapshot.ListOpenOrders(ctx, params)
		require.Nil(t, apiErr)
		require.Equal(t, want, got, "orders page %d", page)
		if !want.PageInfo.HasNextPage {
			if want.PageInfo.PrevCursor != nil {
				params.Cursor = want.PageInfo.PrevCursor
				wantPrev, apiErr := live.ListOpenOrders(ctx, params)
				require.Nil(t, apiErr)
				gotPrev, apiErr := snapshot.ListOpenOrders(ctx, params)
				require.Nil(t, apiErr)
				require.Equal(t, wantPrev, gotPrev, "orders previous page")
			}
			break
		}
		cursor = want.PageInfo.NextCursor
	}
}

// rawOrderEntries reads the order entries query's rows as the database returns them, so values are
// compared exactly rather than after their conversion to float.
func rawOrderEntries(t *testing.T, ctx context.Context, r *analyticsRepoImpl, f domain.OpenOrderFilter, openOnly bool) [][]string {
	t.Helper()
	scope, empty, apiErr := r.newOrderBookScope(ctx, f, openOnly)
	require.Nil(t, apiErr)
	if empty {
		return nil
	}
	query, args := orderEntriesQuery(scope, 0)
	rows, err := r.queries.DB().QueryContext(ctx, query, args...)
	require.NoError(t, err)
	defer rows.Close()
	cols, err := rows.Columns()
	require.NoError(t, err)
	var out [][]string
	for rows.Next() {
		values := make([]sql.NullString, len(cols))
		dest := make([]any, len(cols))
		for i := range values {
			dest[i] = &values[i]
		}
		require.NoError(t, rows.Scan(dest...))
		row := make([]string, len(cols))
		for i, v := range values {
			row[i] = fmt.Sprintf("%v:%s", v.Valid, v.String)
		}
		out = append(out, row)
	}
	require.NoError(t, rows.Err())
	return out
}

// TestOpenOrderSnapshotFollowsEdits edits each source an open line is copied from, refreshes the orders
// the edit's dirty mark resolves to, and holds the snapshot to the live reports after each.
func TestOpenOrderSnapshotFollowsEdits(t *testing.T) {
	ctx := context.Background()
	pool := testDB(t)
	pagination.Init([]byte("open-order-snapshot-integration-test"))
	queries := sqlc.New(pool)
	snapshotFixtures(t, ctx, pool)
	repo := NewOpenOrderSnapshotRepo(queries)
	syncSnapshot(t, ctx, repo)

	live := &analyticsRepoImpl{queries: queries, orderBook: orderBookLive}
	snapshot := &analyticsRepoImpl{queries: queries}
	openOrderSnapshotReady.Store(true)
	t.Cleanup(func() { openOrderSnapshotReady.Store(false) })
	f := domain.OpenOrderFilter{AccountID: rollupTestAccountID}
	order := snapshotFixture + "_" + rollupTestAccountID[len(rollupTestAccountID)-5:] + "_"

	exec := func(query string, args ...any) {
		t.Helper()
		_, err := pool.ExecContext(ctx, query, args...)
		require.NoError(t, err, query)
	}
	// refresh resolves the mark an edit would write and refreshes what it resolves to.
	refresh := func(scope domain.SalesFactScope, id string) {
		t.Helper()
		orders, apiErr := repo.ResolveOrders(ctx, scope, []string{id})
		require.Nil(t, apiErr)
		require.Contains(t, orders, order+"so", "a %s mark did not resolve to its order", scope)
		_, _, apiErr = repo.RefreshOrders(ctx, orders)
		require.Nil(t, apiErr)
		requireSameOrderBook(t, ctx, live, snapshot, f)
	}

	exec(`UPDATE quantity SET value = value + 3 WHERE id = ?`, order+"q1")
	refresh(domain.SalesFactScopeQuantity, order+"q1")

	exec(`UPDATE rate SET value = value * 2 WHERE id = ?`, order+"p1")
	refresh(domain.SalesFactScopeRate, order+"p1")

	exec(`UPDATE rate SET value = value + 1 WHERE id = ?`, order+"c2")
	refresh(domain.SalesFactScopeRate, order+"c2")

	exec(`UPDATE quantity SET value = 4 WHERE id = ?`, order+"iq1")
	refresh(domain.SalesFactScopeQuantity, order+"iq1")

	var invoiceID string
	require.NoError(t, pool.QueryRowContext(ctx, `SELECT invoice_id FROM invoice_line WHERE id = ?`, order+"il1").Scan(&invoiceID))
	exec(`DELETE FROM invoice_line WHERE id = ?`, order+"il1")
	refresh(domain.SalesFactScopeInvoice, invoiceID)

	exec(`UPDATE sales_order_line SET unit_cost_id = NULL WHERE id = ?`, order+"l2")
	exec(`UPDATE sales_order_line SET unit_cost_id = ? WHERE id = ?`, order+"c2", order+"l3")
	refresh(domain.SalesFactScopeSalesOrderLine, order+"l3")

	exec(`UPDATE sales_order SET completed_at = NULL WHERE id = ?`, order+"so")
	refresh(domain.SalesFactScopeSalesOrder, order+"so")

	// An order no longer issued leaves the snapshot, and comes back when it is issued again.
	exec(`UPDATE sales_order SET sales_order_status_code = 'fulfilled' WHERE id = ?`, order+"so")
	refresh(domain.SalesFactScopeSalesOrder, order+"so")
	var held int
	require.NoError(t, pool.QueryRowContext(ctx, `SELECT (SELECT COUNT(*) FROM open_order WHERE id = ?) + (SELECT COUNT(*) FROM open_order_line WHERE sales_order_id = ?)
    + (SELECT COUNT(*) FROM open_order_invoiced WHERE sales_order_id = ?)`, order+"so", order+"so", order+"so").Scan(&held))
	require.Zero(t, held, "a fulfilled order's rows stayed in the snapshot")
	exec(`UPDATE sales_order SET sales_order_status_code = 'issued' WHERE id = ?`, order+"so")
	refresh(domain.SalesFactScopeSalesOrder, order+"so")

	// A refresh that changes nothing writes nothing.
	accounts, changed, apiErr := repo.RefreshOrders(ctx, []string{order + "so"})
	require.Nil(t, apiErr)
	require.Zero(t, changed)
	require.Empty(t, accounts, "and names no account whose reports changed")
}
