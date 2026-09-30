//go:build integration

package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
)

// legacyNewCustomersAggregate is the dashboard's new-customers query (customer.repo.ts findNewCustomersTable),
// unchanged but for reading every buyer of the account instead of a list: each buyer's first order and
// lifetime sales, computed from the order lines themselves.
const legacyNewCustomersAggregate = `SELECT
    so.buyer_account_id,
    CAST(SUM(
        ((q_in.value * (u_in.ratio_numerator / u_in.ratio_denominator)) + (u_in.offset_numerator / u_in.offset_denominator))
        *
        (((r_price.value * (u_price_num.ratio_numerator / u_price_num.ratio_denominator)) + (u_price_num.offset_numerator / u_price_num.offset_denominator))
         / NULLIF((u_price_den.ratio_numerator / u_price_den.ratio_denominator) + (u_price_den.offset_numerator / u_price_den.offset_denominator), 0))
    ) AS DECIMAL(65,30)),
    MIN(so.issued_at)
FROM invoice_line il
JOIN invoice i ON i.id = il.invoice_id
JOIN sales_order so ON so.id = i.sales_order_id
JOIN sales_order_line sol ON sol.id = il.sales_order_line_id
JOIN product fg ON fg.id = sol.product_id
JOIN product_line pl ON pl.id = fg.product_line_id
JOIN quantity q_in ON q_in.id = il.quantity_id
JOIN unit u_in ON u_in.id = q_in.unit_id
LEFT JOIN rate r_price ON r_price.id = sol.unit_price_id
LEFT JOIN unit u_price_num ON u_price_num.id = r_price.numerator_unit_id
LEFT JOIN unit u_price_den ON u_price_den.id = r_price.denominator_unit_id
WHERE so.owner_account_id = ? AND so.sales_order_type_code = 'sales_order'
  AND r_price.value > 0 AND LOWER(pl.name) NOT IN ('shipping', 'misc')
GROUP BY so.buyer_account_id`

type buyerSummary struct {
	total decimal.Decimal
	first time.Time
}

// refreshAllFacts recomputes every fact of the account as the refresher would, so each carries its order date and price flag.
func refreshAllFacts(t *testing.T, ctx context.Context, pool *sql.DB, repo domain.SalesFactRepo) {
	t.Helper()
	invoices := distinctIDs(t, pool, `SELECT id FROM invoice WHERE account_id = ?`, rollupTestAccountID)
	for start := 0; start < len(invoices); start += 200 {
		batch := invoices[start:min(start+200, len(invoices))]
		facts, apiErr := repo.ComputeFacts(ctx, batch)
		require.Nil(t, apiErr)
		require.Nil(t, repo.UpsertFacts(ctx, facts))
	}
}

func distinctIDs(t *testing.T, pool *sql.DB, query string, args ...any) []string {
	t.Helper()
	rows, err := pool.Query(query, args...)
	require.NoError(t, err)
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		require.NoError(t, rows.Scan(&id))
		out = append(out, id)
	}
	return out
}

func rebuildAllBuyerSummaries(t *testing.T, ctx context.Context, repo domain.SalesFactRepo) {
	t.Helper()
	cursor := domain.SalesBuyerKey{}
	for {
		next, apiErr := repo.NextBuyers(ctx, cursor, 50)
		require.Nil(t, apiErr)
		if len(next) == 0 {
			return
		}
		byAccount := map[string][]string{}
		for _, k := range next {
			byAccount[k.AccountID] = append(byAccount[k.AccountID], k.BuyerAccountID)
		}
		for account, buyers := range byAccount {
			require.Nil(t, repo.RebuildBuyerSummaries(ctx, account, buyers))
		}
		cursor = next[len(next)-1]
	}
}

func TestBuyerSummariesMatchTheLegacyNewCustomersReport(t *testing.T) {
	ctx := context.Background()
	pool := testDB(t)
	repo := NewSalesFactRepo(sqlc.New(pool))
	refreshAllFacts(t, ctx, pool, repo)
	rebuildAllBuyerSummaries(t, ctx, repo)

	want := map[string]buyerSummary{}
	rows, err := pool.Query(legacyNewCustomersAggregate, rollupTestAccountID)
	require.NoError(t, err)
	for rows.Next() {
		var (
			buyer string
			total sql.NullString
			first sql.NullTime
		)
		require.NoError(t, rows.Scan(&buyer, &total, &first))
		if !first.Valid {
			continue // the legacy report drops a customer with no dated order
		}
		s := buyerSummary{first: first.Time}
		if total.Valid {
			s.total = decimal.RequireFromString(total.String)
		}
		want[buyer] = s
	}
	require.NoError(t, rows.Close())
	require.NotEmpty(t, want, "the seed account has no qualifying sales; seed it first")

	got := map[string]buyerSummary{}
	rows, err = pool.Query(`SELECT buyer_account_id, CAST(total_invoiced AS CHAR), first_ordered_at FROM sales_buyer_summary WHERE account_id = ?`, rollupTestAccountID)
	require.NoError(t, err)
	for rows.Next() {
		var (
			buyer, total string
			first        time.Time
		)
		require.NoError(t, rows.Scan(&buyer, &total, &first))
		got[buyer] = buyerSummary{total: decimal.RequireFromString(total), first: first}
	}
	require.NoError(t, rows.Close())

	require.Len(t, got, len(want), "the same customers have ordered")
	for buyer, w := range want {
		g, ok := got[buyer]
		require.True(t, ok, "buyer %s has no summary", buyer)
		require.True(t, w.first.Equal(g.first), "buyer %s first order: legacy %s, summary %s", buyer, w.first, g.first)
		// Facts store each line rounded to 10 places; the legacy query summed them unrounded.
		require.True(t, w.total.Sub(g.total).Abs().LessThan(decimal.New(1, -6)), "buyer %s lifetime sales: legacy %s, summary %s", buyer, w.total, g.total)
	}
}

// A summary follows its facts: a line priced at zero or in an excluded product line never counts, and a
// buyer left with no qualifying line loses its summary.
func TestBuyerSummaryFollowsItsFacts(t *testing.T) {
	ctx := context.Background()
	pool := testDB(t)
	repo := NewSalesFactRepo(sqlc.New(pool))
	// An account of its own: the rollup comparison reads the seed account, and fake facts there would skew it.
	const account, buyer = "ac_summary_test_owner", "ac_summary_test"
	shipping := distinctIDs(t, pool, `SELECT id FROM product_line WHERE LOWER(name) IN ('shipping', 'misc') LIMIT 1`)
	regular := distinctIDs(t, pool, `SELECT id FROM product_line WHERE LOWER(name) NOT IN ('shipping', 'misc') LIMIT 1`)
	require.NotEmpty(t, regular)
	t.Cleanup(func() {
		_, _ = pool.Exec(`DELETE FROM sales_line_fact WHERE buyer_account_id = ?`, buyer)
		_, _ = pool.Exec(`DELETE FROM sales_buyer_summary WHERE buyer_account_id = ?`, buyer)
	})

	first := time.Date(2025, 2, 3, 10, 0, 0, 0, time.UTC)
	insert := func(line, pl string, total string, priced bool, orderedAt time.Time) {
		_, err := pool.Exec(`INSERT INTO sales_line_fact (account_id, invoiced_at, invoice_line_id, invoice_id, sales_order_id, sales_order_type_code, buyer_account_id, product_id, item_id, product_line_id, total_invoiced, ordered_at, is_priced, refreshed_at)
VALUES (?, ?, ?, 'iv_summary_test', 'or_summary_test', 'sales_order', ?, 'pd_x', 'it_x', ?, ?, ?, ?, NOW(3))`, account, orderedAt.Add(24*time.Hour), line, buyer, pl, total, orderedAt, priced)
		require.NoError(t, err)
	}
	summary := func() *buyerSummary {
		require.Nil(t, repo.RebuildBuyerSummaries(ctx, account, []string{buyer}))
		var total string
		var at time.Time
		err := pool.QueryRow(`SELECT CAST(total_invoiced AS CHAR), first_ordered_at FROM sales_buyer_summary WHERE account_id = ? AND buyer_account_id = ?`, account, buyer).Scan(&total, &at)
		if err == sql.ErrNoRows {
			return nil
		}
		require.NoError(t, err)
		return &buyerSummary{total: decimal.RequireFromString(total), first: at}
	}

	insert("ivln_sum_zero", regular[0], "0", false, first.Add(-48*time.Hour))
	require.Nil(t, summary(), "a line priced at zero is not a sale")

	insert("ivln_sum_a", regular[0], "12.5", true, first)
	insert("ivln_sum_b", regular[0], "7.25", true, first.Add(72*time.Hour))
	if len(shipping) > 0 {
		insert("ivln_sum_ship", shipping[0], "99", true, first.Add(-24*time.Hour))
	}
	s := summary()
	require.NotNil(t, s)
	require.True(t, decimal.RequireFromString("19.75").Equal(s.total), "shipping and zero-priced lines excluded, got %s", s.total)
	require.True(t, first.Equal(s.first), "the earliest qualifying order, got %s", s.first)

	_, err := pool.Exec(`DELETE FROM sales_line_fact WHERE invoice_line_id IN ('ivln_sum_a', 'ivln_sum_b')`)
	require.NoError(t, err)
	require.Nil(t, summary(), "no qualifying line left")
}

func TestBuyerMarksShareTheFactQueueWithoutReachingTheFactDrain(t *testing.T) {
	ctx := context.Background()
	pool := testDB(t)
	repo := NewSalesFactRepo(sqlc.New(pool))
	t.Cleanup(func() {
		_, _ = pool.Exec(`DELETE FROM sales_fact_dirty WHERE account_id IN ('ac_mark_one', 'ac_mark_two')`)
	})

	// One customer buying from two accounts is two marks.
	one := domain.SalesBuyerKey{AccountID: "ac_mark_one", BuyerAccountID: "ac_shared_buyer"}
	two := domain.SalesBuyerKey{AccountID: "ac_mark_two", BuyerAccountID: "ac_shared_buyer"}
	require.Nil(t, repo.MarkBuyers(ctx, []domain.SalesBuyerKey{one, two, one}))

	marks, apiErr := repo.ListBuyerDirty(ctx, 10_000)
	require.Nil(t, apiErr)
	var mine []domain.SalesBuyerDirtyMark
	for _, m := range marks {
		if m.Buyer == one || m.Buyer == two {
			mine = append(mine, m)
		}
	}
	require.Len(t, mine, 2)

	factMarks, apiErr := repo.ListDirty(ctx, 10_000)
	require.Nil(t, apiErr)
	for _, m := range factMarks {
		require.NotEqual(t, domain.SalesFactScopeBuyerSummary, m.ScopeType, "the fact drain would try to resolve a buyer summary mark to invoices")
	}

	// A re-mark while the rebuild ran survives clearing the mark read before it.
	stale := mine[0]
	time.Sleep(5 * time.Millisecond)
	require.Nil(t, repo.MarkBuyers(ctx, []domain.SalesBuyerKey{stale.Buyer}))
	require.Nil(t, repo.ClearBuyerDirty(ctx, stale))
	marks, apiErr = repo.ListBuyerDirty(ctx, 10_000)
	require.Nil(t, apiErr)
	found := false
	for _, m := range marks {
		if m.Buyer == stale.Buyer {
			found = true
			require.Nil(t, repo.ClearBuyerDirty(ctx, m))
		}
	}
	require.True(t, found, "the re-mark was lost")
}

func TestTheBuyerSweepRowLeavesTheRollupSweepAlone(t *testing.T) {
	ctx := context.Background()
	pool := testDB(t)
	repo := NewSalesFactRepo(sqlc.New(pool))
	rollupBefore, apiErr := repo.GetRollupSync(ctx)
	require.Nil(t, apiErr)
	buyerBefore, apiErr := repo.GetBuyerSummarySync(ctx)
	require.Nil(t, apiErr)
	t.Cleanup(func() {
		_ = repo.SaveBuyerSummarySync(ctx, *buyerBefore)
		_ = repo.SaveRollupSync(ctx, *rollupBefore)
	})

	at := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	cursor := domain.SalesBuyerKey{AccountID: "ac_1", BuyerAccountID: "ac_2"}
	want := domain.SalesBuyerSummarySync{Cursor: &cursor, FactsSince: &at, PassStartedAt: &at}
	require.Nil(t, repo.SaveBuyerSummarySync(ctx, want))
	got, apiErr := repo.GetBuyerSummarySync(ctx)
	require.Nil(t, apiErr)
	require.Equal(t, cursor, *got.Cursor)
	require.True(t, at.Equal(*got.FactsSince))
	require.Nil(t, got.LastCompletedAt)

	rollupAfter, apiErr := repo.GetRollupSync(ctx)
	require.Nil(t, apiErr)
	require.Equal(t, rollupBefore, rollupAfter)
}
