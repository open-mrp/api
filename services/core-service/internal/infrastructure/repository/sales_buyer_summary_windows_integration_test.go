//go:build integration

package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/stretchr/testify/require"
)

const windowsTestAccountID = "ac_buyer_windows"

// A buyer split into invoiced_at windows must summarize exactly as one query over all its facts does.
func TestBuyerSummaryReadInWindowsMatchesOneRead(t *testing.T) {
	pool := testDB(t)
	ctx := context.Background()
	repo := NewSalesFactRepo(sqlc.New(pool))

	var shipping, other string
	require.NoError(t, pool.QueryRow(`SELECT id FROM product_line WHERE LOWER(name) = 'shipping' LIMIT 1`).Scan(&shipping))
	require.NoError(t, pool.QueryRow(`SELECT id FROM product_line WHERE LOWER(name) NOT IN ('shipping', 'misc') LIMIT 1`).Scan(&other))

	clear := func() {
		for _, table := range []string{"sales_line_fact", "sales_fact_rollup", "sales_buyer_summary"} {
			_, _ = pool.Exec(`DELETE FROM `+table+` WHERE account_id = ?`, windowsTestAccountID)
		}
	}
	clear()
	t.Cleanup(clear)

	type fact struct {
		at, ordered time.Time
		line        string
		total       any
		priced      bool
	}
	jan := time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC)
	facts := []fact{
		{jan, jan.AddDate(0, 0, -3), other, "10.125", true},
		{jan.AddDate(0, 1, 0), jan.AddDate(0, 0, -9), other, "20", true},     // the earliest order, a month later
		{jan.AddDate(0, 1, 2), jan.AddDate(0, 0, -30), shipping, "99", true}, // excluded line
		{jan.AddDate(0, 2, 0), jan.AddDate(0, 2, -1), other, nil, true},      // priced but no total
		{jan.AddDate(0, 3, 0), jan.AddDate(0, 3, -1), other, "-5.5", false},  // unpriced
		{jan.AddDate(0, 4, 0), jan.AddDate(0, 4, -1), other, "0.25", true},
	}
	for i, f := range facts {
		_, err := pool.Exec(`INSERT INTO sales_line_fact (account_id, invoiced_at, invoice_line_id, invoice_id, sales_order_id, sales_order_type_code,
buyer_account_id, product_id, item_id, product_line_id, total_invoiced, refreshed_at, ordered_at, is_priced)
VALUES (?, ?, ?, ?, 'or_x', 'sales_order', 'ac_win_buyer', 'pd_x', 'it_x', ?, ?, NOW(3), ?, ?)`,
			windowsTestAccountID, f.at, "ivln_win_"+string(rune('a'+i)), "iv_win_"+string(rune('a'+i)), f.line, f.total, f.ordered, f.priced)
		require.NoError(t, err)
		require.Nil(t, repo.RebuildRollupDay(ctx, domain.SalesRollupDay{AccountID: windowsTestAccountID, Day: f.at}))
	}

	summary := func() (time.Time, string) {
		var first time.Time
		var total string
		require.NoError(t, pool.QueryRow(`SELECT first_ordered_at, total_invoiced FROM sales_buyer_summary WHERE account_id = ? AND buyer_account_id = 'ac_win_buyer'`,
			windowsTestAccountID).Scan(&first, &total))
		return first.UTC(), total
	}

	require.Nil(t, repo.RebuildBuyerSummaries(ctx, windowsTestAccountID, []string{"ac_win_buyer"}))
	wholeFirst, wholeTotal := summary()
	require.Equal(t, jan.AddDate(0, 0, -9), wholeFirst)
	require.Equal(t, "30.375000000000000000000000000000", wholeTotal)

	budget := salesBuyerSummaryLineBudget
	salesBuyerSummaryLineBudget = 1
	t.Cleanup(func() { salesBuyerSummaryLineBudget = budget })
	_, err := pool.Exec(`DELETE FROM sales_buyer_summary WHERE account_id = ?`, windowsTestAccountID)
	require.NoError(t, err)
	require.Nil(t, repo.RebuildBuyerSummaries(ctx, windowsTestAccountID, []string{"ac_win_buyer"}))
	first, total := summary()
	require.Equal(t, wholeFirst, first)
	require.Equal(t, wholeTotal, total)

	// A buyer left with no qualifying sale loses its summary.
	_, err = pool.Exec(`UPDATE sales_line_fact SET is_priced = 0 WHERE account_id = ?`, windowsTestAccountID)
	require.NoError(t, err)
	require.Nil(t, repo.RebuildBuyerSummaries(ctx, windowsTestAccountID, []string{"ac_win_buyer"}))
	err = pool.QueryRow(`SELECT 1 FROM sales_buyer_summary WHERE account_id = ?`, windowsTestAccountID).Scan(new(int))
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestBuyerSweepReadsSkipCurrentSummaries(t *testing.T) {
	pool := testDB(t)
	ctx := context.Background()
	repo := NewSalesFactRepo(sqlc.New(pool))
	t.Cleanup(func() {
		_, _ = pool.Exec(`DELETE FROM sales_line_fact WHERE account_id = ?`, windowsTestAccountID)
		_, _ = pool.Exec(`DELETE FROM sales_buyer_summary WHERE account_id = ?`, windowsTestAccountID)
	})
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	_, err := pool.Exec(`INSERT INTO sales_line_fact (account_id, invoiced_at, invoice_line_id, invoice_id, sales_order_id, sales_order_type_code,
buyer_account_id, product_id, item_id, product_line_id, refreshed_at) VALUES (?, ?, 'ivln_sweep', 'iv_sweep', 'or_x', 'sales_order', 'ac_sweep_buyer', 'pd_x', 'it_x', 'pl_x', ?)`,
		windowsTestAccountID, at, at)
	require.NoError(t, err)
	_, err = pool.Exec(`INSERT INTO sales_buyer_summary (account_id, buyer_account_id, first_ordered_at, total_invoiced, refreshed_at)
VALUES (?, 'ac_sweep_buyer', ?, 1, ?), (?, 'ac_sweep_zz_gone', ?, 1, ?)`, windowsTestAccountID, at, at.Add(time.Hour), windowsTestAccountID, at, at)
	require.NoError(t, err)

	latest, apiErr := repo.LatestBuyerFactRefreshes(ctx, windowsTestAccountID, []string{"ac_sweep_buyer", "ac_nobody"})
	require.Nil(t, apiErr)
	require.Equal(t, map[string]time.Time{"ac_sweep_buyer": at}, utcValues(latest))

	through := domain.SalesBuyerKey{AccountID: windowsTestAccountID, BuyerAccountID: "ac_sweep_buyer"}
	inRange, apiErr := repo.ListBuyerSummaryRefreshes(ctx, domain.SalesBuyerKey{AccountID: windowsTestAccountID}, &through)
	require.Nil(t, apiErr)
	require.Len(t, inRange, 1)
	rest, apiErr := repo.ListBuyerSummaryRefreshes(ctx, through, nil)
	require.Nil(t, apiErr)
	require.Contains(t, rest, domain.SalesBuyerKey{AccountID: windowsTestAccountID, BuyerAccountID: "ac_sweep_zz_gone"})

	require.Nil(t, repo.DeleteBuyerSummaries(ctx, []domain.SalesBuyerKey{{AccountID: windowsTestAccountID, BuyerAccountID: "ac_sweep_zz_gone"}}))
	var n int
	require.NoError(t, pool.QueryRow(`SELECT COUNT(*) FROM sales_buyer_summary WHERE account_id = ?`, windowsTestAccountID).Scan(&n))
	require.Equal(t, 1, n)
}

func utcValues(m map[string]time.Time) map[string]time.Time {
	out := make(map[string]time.Time, len(m))
	for k, v := range m {
		out[k] = v.UTC()
	}
	return out
}
