//go:build integration

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/pagination"
	"github.com/stretchr/testify/require"
)

// Reports read through the rollups must equal the same reports read from sales_line_fact alone. Run against
// the seeded dev database:
//
//	go test -tags integration ./services/core-service/internal/infrastructure/repository/ -run SalesRollups

const rollupTestAccountID = "ac_01k0a5smf9ekb8rqg12555zjqa"

// rebuildAllRollups rebuilds every day, as a complete sweep pass does.
func rebuildAllRollups(t *testing.T, ctx context.Context, repo domain.SalesFactRepo) {
	t.Helper()
	cursor := domain.SalesRollupDay{Day: salesFactSweepFloor}
	for {
		next, apiErr := repo.NextRollupDay(ctx, cursor)
		require.Nil(t, apiErr)
		if next == nil {
			break
		}
		require.Nil(t, repo.RebuildRollupDay(ctx, *next))
		cursor = domain.SalesRollupDay{AccountID: next.AccountID, Day: next.Day.AddDate(0, 0, 1)}
	}
}

func distinctFactValues(t *testing.T, pool *sql.DB, column string) []string {
	t.Helper()
	rows, err := pool.Query(fmt.Sprintf(`SELECT DISTINCT %s FROM sales_line_fact WHERE account_id = ? AND %[1]s IS NOT NULL ORDER BY %[1]s`, column), rollupTestAccountID)
	require.NoError(t, err)
	defer rows.Close()
	var out []string
	for rows.Next() {
		var v string
		require.NoError(t, rows.Scan(&v))
		out = append(out, v)
	}
	require.NotEmpty(t, out, "no %s in the seed account's facts; seed it first", column)
	return out
}

func TestSalesRollupsMatchFacts(t *testing.T) {
	ctx := context.Background()
	pool := testDB(t)
	pagination.Init([]byte("sales-rollup-integration-test"))
	queries := sqlc.New(pool)
	rebuildAllRollups(t, ctx, NewSalesFactRepo(queries))

	rollups := &salesReportRepoImpl{queries: queries, mode: rollupAuto}
	facts := &salesReportRepoImpl{queries: queries, mode: rollupNever}
	salesRollupsReady.Store(true)
	t.Cleanup(func() { salesRollupsReady.Store(false) })

	productLines := distinctFactValues(t, pool, "product_line_id")
	reps := distinctFactValues(t, pool, "sales_rep_id")
	buyers := distinctFactValues(t, pool, "buyer_account_id")
	items := distinctFactValues(t, pool, "item_id")
	// Two product lines one invoice spans: summing their rows would count that invoice twice.
	var sharedA, sharedB string
	require.NoError(t, pool.QueryRow(`SELECT a.product_line_id, b.product_line_id FROM sales_line_fact a
JOIN sales_line_fact b ON b.invoice_id = a.invoice_id AND b.product_line_id > a.product_line_id
WHERE a.account_id = ? LIMIT 1`, rollupTestAccountID).Scan(&sharedA, &sharedB), "no invoice spans two product lines; seed one")

	est := time.FixedZone("est", -5*3600)
	windows := []struct {
		name       string
		start, end time.Time
		tzMinutes  int32
	}{
		{"everything", time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC).Add(-time.Millisecond), 0},
		{"a local year", time.Date(2025, 1, 1, 0, 0, 0, 0, est), time.Date(2026, 1, 1, 0, 0, 0, 0, est).Add(-time.Millisecond), -300},
		{"a local month", time.Date(2025, 3, 1, 0, 0, 0, 0, est), time.Date(2025, 4, 1, 0, 0, 0, 0, est).Add(-time.Millisecond), -300},
		{"ragged edges", time.Date(2024, 2, 17, 13, 47, 12, 0, time.UTC), time.Date(2025, 8, 3, 9, 5, 0, 0, time.UTC), 0},
		{"a half-hour offset", time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC).Add(-330 * time.Minute), time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC).Add(-330*time.Minute - time.Millisecond), 330},
		// No whole bucket: the union holds only raw rows, and must still name its columns.
		{"inside one hour", time.Date(2025, 3, 12, 14, 5, 0, 0, time.UTC), time.Date(2025, 3, 12, 14, 55, 0, 0, time.UTC), 0},
		{"a ragged day", time.Date(2025, 3, 12, 3, 5, 0, 0, time.UTC), time.Date(2025, 3, 12, 22, 55, 0, 0, time.UTC), 0},
	}
	filters := []struct {
		name string
		set  func(*domain.SalesReportFilter)
	}{
		{"no filter", func(*domain.SalesReportFilter) {}},
		{"one product line", func(f *domain.SalesReportFilter) { f.ProductLineIDs = productLines[:1] }},
		{"two product lines", func(f *domain.SalesReportFilter) { f.ProductLineIDs = productLines[:2] }},
		{"two product lines one invoice spans", func(f *domain.SalesReportFilter) { f.ProductLineIDs = []string{sharedA, sharedB} }},
		{"shared product lines and a rep", func(f *domain.SalesReportFilter) {
			f.ProductLineIDs, f.SalesRepIDs = []string{sharedA, sharedB}, reps[:1]
		}},
		{"sales reps", func(f *domain.SalesReportFilter) { f.SalesRepIDs = reps[:2] }},
		{"a rep and a product line", func(f *domain.SalesReportFilter) { f.SalesRepIDs, f.ProductLineIDs = reps[:1], productLines[1:2] }},
		{"customers", func(f *domain.SalesReportFilter) { f.CustomerIDs = buyers[:3] }},
		{"items", func(f *domain.SalesReportFilter) { f.ItemIDs = items[:2] }},
	}
	groupings := []constants.SalesBreakdownGroupBy{
		constants.SalesBreakdownGroupByCustomer, constants.SalesBreakdownGroupByProduct, constants.SalesBreakdownGroupByProductLine,
		constants.SalesBreakdownGroupByCustomerGroup, constants.SalesBreakdownGroupBySalesRep, constants.SalesBreakdownGroupByDiscount,
	}

	sawSales := false
	for _, w := range windows {
		for _, fl := range filters {
			t.Run(w.name+"/"+fl.name, func(t *testing.T) {
				filter := domain.SalesReportFilter{AccountID: rollupTestAccountID, StartsAt: w.start, EndsAt: w.end}
				cmpStart, cmpEnd := w.start.AddDate(-1, 0, 0), w.end.AddDate(-1, 0, 0)
				filter.ComparisonStartsAt, filter.ComparisonEndsAt = &cmpStart, &cmpEnd
				fl.set(&filter)

				summaryParams := domain.AnalyzeSalesSummaryParams{SalesReportFilter: filter, TZOffsetMinutes: w.tzMinutes}
				want, apiErr := facts.GetSummary(ctx, summaryParams, true)
				require.Nil(t, apiErr)
				got, apiErr := rollups.GetSummary(ctx, summaryParams, true)
				require.Nil(t, apiErr)
				require.Equal(t, want, got, "summary")
				if want.Current.LineCount > 0 {
					sawSales = true
				}

				for _, groupBy := range groupings {
					params := domain.AnalyzeSalesBreakdownParams{SalesReportFilter: filter, GroupBy: groupBy, Limit: 1000}
					want, apiErr := facts.GetBreakdown(ctx, params, true)
					require.Nil(t, apiErr)
					got, apiErr := rollups.GetBreakdown(ctx, params, true)
					require.Nil(t, apiErr)
					require.Equal(t, want, got, "%s breakdown", groupBy)
				}
			})
		}
	}
	require.True(t, sawSales, "no window held any sales; the comparison proved nothing")
}

func TestSalesRollupsFollowFactChanges(t *testing.T) {
	ctx := context.Background()
	pool := testDB(t)
	queries := sqlc.New(pool)
	repo := NewSalesFactRepo(queries)
	rebuildAllRollups(t, ctx, repo)

	day := time.Date(2025, 5, 14, 0, 0, 0, 0, time.UTC)
	var before int64
	require.NoError(t, pool.QueryRow(`SELECT COALESCE(SUM(line_count), 0) FROM sales_fact_rollup WHERE account_id = ? AND dimension = 'total' AND product_line_key = '' AND grain = 'month' AND bucket_start = ?`,
		rollupTestAccountID, truncateUTCMonth(day)).Scan(&before))

	// A fact added to a day shows up in that day's and month's buckets once the day is rebuilt, and leaves when removed.
	_, err := pool.Exec(`INSERT INTO sales_line_fact (account_id, invoiced_at, invoice_line_id, invoice_id, sales_order_id, sales_order_type_code, buyer_account_id, product_id, item_id, product_line_id, total_invoiced, refreshed_at)
VALUES (?, ?, 'ivln_rollup_test', 'iv_rollup_test', 'or_x', 'sales_order', 'ac_x', 'pd_x', 'it_x', 'pl_x', 12.5, NOW(3))`, rollupTestAccountID, day.Add(10*time.Hour))
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(`DELETE FROM sales_line_fact WHERE invoice_line_id = 'ivln_rollup_test'`)
		_ = repo.RebuildRollupDay(ctx, domain.SalesRollupDay{AccountID: rollupTestAccountID, Day: day})
	})

	monthLines := func() int64 {
		require.Nil(t, repo.RebuildRollupDay(ctx, domain.SalesRollupDay{AccountID: rollupTestAccountID, Day: day}))
		var n int64
		require.NoError(t, pool.QueryRow(`SELECT COALESCE(SUM(line_count), 0) FROM sales_fact_rollup WHERE account_id = ? AND dimension = 'total' AND product_line_key = '' AND grain = 'month' AND bucket_start = ?`,
			rollupTestAccountID, truncateUTCMonth(day)).Scan(&n))
		return n
	}
	require.Equal(t, before+1, monthLines())

	_, err = pool.Exec(`DELETE FROM sales_line_fact WHERE invoice_line_id = 'ivln_rollup_test'`)
	require.NoError(t, err)
	require.Equal(t, before, monthLines())
}
