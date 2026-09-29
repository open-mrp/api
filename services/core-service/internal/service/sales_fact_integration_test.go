//go:build integration

package service

import (
	"context"
	"database/sql"
	"math/big"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/repository"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/lease"
	"github.com/open-mrp/api/shared/pagination"
)

// sales_line_fact and its reports are SQL end to end, so these run against the seeded dev database:
//
//	go test -tags integration ./services/core-service/internal/service/ -run SalesFact
//
// Assertions are invariants of whatever sales the database holds, plus edits to one seeded invoice that
// each test reverts.

const (
	seedAccountID      = "ac_01k0a5smf9ekb8rqg12555zjqa"
	seedInvoiceID      = "iv_01seedinvoice002000"
	salesFactTestLease = "sales-fact-refresher-test"
)

func salesFactDB(t *testing.T) (*sql.DB, domain.RepoFactory) {
	t.Helper()
	dsn := os.Getenv("SQL_PREPARE_TEST_DSN")
	if dsn == "" {
		dsn = "root:Testing123!@tcp(localhost:3306)/openmrp?parseTime=true"
	}
	pool, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}
	t.Cleanup(func() { _ = pool.Close() })
	if err := pool.Ping(); err != nil {
		t.Fatalf("ping mysql: %v", err)
	}
	pagination.Init([]byte("sales-fact-integration-test"))
	return pool, repository.NewRepoFactory(sqlc.New(pool))
}

func newTestRefresher(repos domain.RepoFactory, pool *sql.DB) *SalesFactRefresher {
	return NewSalesFactRefresher(&SalesFactRefresherConfig{
		Repos:           repos,
		Lease:           lease.New(repository.NewLeaseRepo(sqlc.New(pool))),
		ReconcileBudget: time.Minute,
	})
}

// forceReconcile runs a complete reconcile pass now, regardless of when the last one ran.
func forceReconcile(t *testing.T, ctx context.Context, pool *sql.DB, r *SalesFactRefresher) {
	t.Helper()
	if _, err := pool.ExecContext(ctx, `UPDATE sales_fact_sync SET cursor_created_at = NULL, cursor_invoice_id = NULL, pass_started_at = NOW(3) - INTERVAL 2 DAY WHERE name = 'reconcile'`); err != nil {
		t.Fatal(err)
	}
	if apiErr := r.reconcile(ctx); apiErr != nil {
		t.Fatalf("reconcile: %v", apiErr)
	}
}

func allInvoiceIDs(t *testing.T, ctx context.Context, pool *sql.DB) []string {
	t.Helper()
	rows, err := pool.QueryContext(ctx, `SELECT id FROM invoice`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	return ids
}

func TestSalesFactReconcileMatchesSource(t *testing.T) {
	ctx := context.Background()
	pool, repos := salesFactDB(t)
	r := newTestRefresher(repos, pool)

	// An orphan: a fact whose invoice does not exist. The pass must remove it.
	if _, err := pool.ExecContext(ctx, `INSERT INTO sales_line_fact (account_id, invoiced_at, invoice_line_id, invoice_id, sales_order_id, sales_order_type_code, buyer_account_id, product_id, item_id, product_line_id, total_invoiced, refreshed_at)
VALUES (?, NOW(3), 'ivln_test_orphan', 'iv_test_missing', 'or_x', 'sales_order', 'ac_x', 'pd_x', 'it_x', 'pl_x', 1, NOW(3))`, seedAccountID); err != nil {
		t.Fatal(err)
	}
	forceReconcile(t, ctx, pool, r)

	sync, apiErr := repos.NewSalesFactRepo().GetSync(ctx)
	if apiErr != nil || sync.Cursor != nil || sync.LastCompletedAt == nil {
		t.Fatalf("sync after pass = %+v, %v; want a completed pass", sync, apiErr)
	}
	ids := allInvoiceIDs(t, ctx, pool)
	computed, apiErr := repos.NewSalesFactRepo().ComputeFacts(ctx, ids)
	if apiErr != nil {
		t.Fatal(apiErr)
	}
	stored, apiErr := repos.NewSalesFactRepo().GetFacts(ctx, append(ids, "iv_test_missing"))
	if apiErr != nil {
		t.Fatal(apiErr)
	}
	if len(computed) == 0 {
		t.Fatal("no invoiced sales in the database; seed it first")
	}
	upserts, deletes := diffSalesFacts(computed, stored)
	if len(upserts) != 0 || len(deletes) != 0 {
		t.Fatalf("after a pass, %d facts differ from source and %d are stale", len(upserts), len(deletes))
	}
}

func TestSalesFactDirtyMarkRefreshesEditedInvoice(t *testing.T) {
	ctx := context.Background()
	pool, repos := salesFactDB(t)
	r := newTestRefresher(repos, pool)
	forceReconcile(t, ctx, pool, r)

	var quantityID string
	if err := pool.QueryRowContext(ctx, `SELECT quantity_id FROM invoice_line WHERE invoice_id = ? ORDER BY id LIMIT 1`, seedInvoiceID).Scan(&quantityID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.ExecContext(ctx, `UPDATE quantity SET value = value + 1 WHERE id = ?`, quantityID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.ExecContext(ctx, `UPDATE quantity SET value = value - 1 WHERE id = ?`, quantityID)
		_ = r.drainDirtyAfterMark(ctx, repos)
	})

	if apiErr := repos.NewSalesFactRepo().MarkDirty(ctx, domain.SalesFactScopeInvoice, seedInvoiceID, seedAccountID); apiErr != nil {
		t.Fatal(apiErr)
	}
	if apiErr := r.drainDirty(ctx); apiErr != nil {
		t.Fatal(apiErr)
	}
	computed, _ := repos.NewSalesFactRepo().ComputeFacts(ctx, []string{seedInvoiceID})
	stored, _ := repos.NewSalesFactRepo().GetFacts(ctx, []string{seedInvoiceID})
	if upserts, deletes := diffSalesFacts(computed, stored); len(upserts)+len(deletes) != 0 {
		t.Fatalf("edited invoice not refreshed: %d changed, %d stale", len(upserts), len(deletes))
	}
	marks, _ := repos.NewSalesFactRepo().ListDirty(ctx, 1000)
	for _, m := range marks {
		if m.ScopeID == seedInvoiceID {
			t.Fatal("dirty mark was not cleared after its refresh")
		}
	}
}

// drainDirtyAfterMark restores the seeded invoice's facts once its quantity is reverted.
func (s *SalesFactRefresher) drainDirtyAfterMark(ctx context.Context, repos domain.RepoFactory) error {
	if apiErr := repos.NewSalesFactRepo().MarkDirty(ctx, domain.SalesFactScopeInvoice, seedInvoiceID, seedAccountID); apiErr != nil {
		return apiErr
	}
	if apiErr := s.drainDirty(ctx); apiErr != nil {
		return apiErr
	}
	return nil
}

func ratOf(t *testing.T, s string) *big.Rat {
	t.Helper()
	r, ok := new(big.Rat).SetString(s)
	if !ok {
		t.Fatalf("not a decimal: %q", s)
	}
	return r
}

func TestSalesReportsAreConsistent(t *testing.T) {
	ctx := context.Background()
	pool, repos := salesFactDB(t)
	forceReconcile(t, ctx, pool, newTestRefresher(repos, pool))
	reports := repos.NewSalesReportRepo()

	window := domain.SalesReportFilter{
		AccountID: seedAccountID,
		StartsAt:  time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
		EndsAt:    time.Now().UTC().Add(time.Hour),
	}
	cmpStart, cmpEnd := window.StartsAt.AddDate(-1, 0, 0), window.StartsAt
	withComparison := window
	withComparison.ComparisonStartsAt, withComparison.ComparisonEndsAt = &cmpStart, &cmpEnd

	summary, apiErr := reports.GetSummary(ctx, domain.AnalyzeSalesSummaryParams{SalesReportFilter: window, TZOffsetMinutes: -300}, true)
	if apiErr != nil {
		t.Fatalf("summary: %v", apiErr)
	}
	if summary.Current.LineCount == 0 {
		t.Fatal("no sales for the seed account; seed it first")
	}

	// Daily totals add up to the period.
	daySum, dayLines := new(big.Rat), int64(0)
	for _, d := range summary.Daily {
		daySum.Add(daySum, ratOf(t, d.Invoiced))
		dayLines += d.LineCount
	}
	if daySum.Cmp(ratOf(t, summary.Current.Invoiced)) != 0 || dayLines != summary.Current.LineCount {
		t.Errorf("daily totals %s over %d lines; period says %s over %d", daySum.FloatString(6), dayLines, summary.Current.Invoiced, summary.Current.LineCount)
	}

	// Every dimension reads without error; product and customer slices partition the period, so they add up to it too.
	for _, groupBy := range []constants.SalesBreakdownGroupBy{
		constants.SalesBreakdownGroupByCustomer, constants.SalesBreakdownGroupByProduct, constants.SalesBreakdownGroupByProductLine,
		constants.SalesBreakdownGroupByCustomerGroup, constants.SalesBreakdownGroupBySalesRep, constants.SalesBreakdownGroupByDiscount,
	} {
		sum, lines := new(big.Rat), int64(0)
		var cursor *string
		for page := 0; ; page++ {
			b, apiErr := reports.GetBreakdown(ctx, domain.AnalyzeSalesBreakdownParams{SalesReportFilter: withComparison, GroupBy: groupBy, Limit: 2, Cursor: cursor}, true)
			if apiErr != nil {
				t.Fatalf("%s breakdown: %v", groupBy, apiErr)
			}
			for _, g := range b.Groups {
				if g.Comparison == nil || g.Totals.Cost == nil {
					t.Fatalf("%s group %q is missing comparison or cost", groupBy, g.Key)
				}
				sum.Add(sum, ratOf(t, g.Totals.Invoiced))
				lines += g.Totals.LineCount
			}
			if !b.PageInfo.HasNextPage {
				break
			}
			cursor = b.PageInfo.NextCursor
		}
		partitions := groupBy == constants.SalesBreakdownGroupByProduct
		if partitions && (sum.Cmp(ratOf(t, summary.Current.Invoiced)) != 0 || lines != summary.Current.LineCount) {
			t.Errorf("%s groups total %s over %d lines; period is %s over %d", groupBy, sum.FloatString(6), lines, summary.Current.Invoiced, summary.Current.LineCount)
		}
	}

	// Cost is withheld when the caller may not see it.
	hidden, apiErr := reports.GetBreakdown(ctx, domain.AnalyzeSalesBreakdownParams{SalesReportFilter: window, GroupBy: constants.SalesBreakdownGroupByCustomer, Limit: 5}, false)
	if apiErr != nil {
		t.Fatal(apiErr)
	}
	for _, g := range hidden.Groups {
		if g.Totals.Cost != nil {
			t.Fatalf("cost returned to a caller who may not see it")
		}
	}

	// Walking the lines forward visits each once; walking back from the end revisits them in reverse.
	var forward []string
	var cursor *string
	var last domain.SalesLinePage
	for {
		page, apiErr := reports.GetLinePage(ctx, domain.ListSalesLinesParams{SalesReportFilter: window, HasWindow: true, Limit: 3, Cursor: cursor})
		if apiErr != nil {
			t.Fatalf("lines: %v", apiErr)
		}
		for _, l := range page.Lines {
			forward = append(forward, l.ID)
		}
		last = *page
		if !page.PageInfo.HasNextPage {
			break
		}
		cursor = page.PageInfo.NextCursor
	}
	if int64(len(forward)) != summary.Current.LineCount {
		t.Fatalf("paged %d lines; period has %d", len(forward), summary.Current.LineCount)
	}
	seen := map[string]bool{}
	for _, id := range forward {
		if seen[id] {
			t.Fatalf("line %s listed twice", id)
		}
		seen[id] = true
	}
	var backward []string
	cursor = last.PageInfo.PrevCursor
	backward = append(backward, idsOf(last.Lines)...)
	for cursor != nil {
		page, apiErr := reports.GetLinePage(ctx, domain.ListSalesLinesParams{SalesReportFilter: window, HasWindow: true, Limit: 3, Cursor: cursor})
		if apiErr != nil {
			t.Fatalf("lines backward: %v", apiErr)
		}
		backward = append(idsOf(page.Lines), backward...)
		if !page.PageInfo.HasPrevPage {
			break
		}
		cursor = page.PageInfo.PrevCursor
	}
	if len(backward) != len(forward) {
		t.Fatalf("walking back visited %d lines; forward visited %d", len(backward), len(forward))
	}
	for i := range forward {
		if forward[i] != backward[i] {
			t.Fatalf("walking back diverges at %d: %s vs %s", i, backward[i], forward[i])
		}
	}

	// Invoices page without error and never repeat.
	invSeen := map[string]bool{}
	cursor = nil
	for {
		page, apiErr := reports.GetInvoicePage(ctx, domain.AnalyzeSalesInvoicesParams{SalesReportFilter: window, Limit: 2, Cursor: cursor})
		if apiErr != nil {
			t.Fatalf("invoices: %v", apiErr)
		}
		for _, inv := range page.Invoices {
			if invSeen[inv.InvoiceID] {
				t.Fatalf("invoice %s listed twice", inv.InvoiceID)
			}
			invSeen[inv.InvoiceID] = true
		}
		if !page.PageInfo.HasNextPage {
			break
		}
		cursor = page.PageInfo.NextCursor
	}
	if int64(len(invSeen)) != summary.Current.InvoiceCount {
		t.Errorf("listed %d invoices; summary counts %d", len(invSeen), summary.Current.InvoiceCount)
	}

	// A filter that matches no buyer returns nothing rather than everything.
	none := window
	none.CustomerIDs = []string{"ac_does_not_exist"}
	empty, apiErr := reports.GetSummary(ctx, domain.AnalyzeSalesSummaryParams{SalesReportFilter: none}, true)
	if apiErr != nil || empty.Current.LineCount != 0 {
		t.Fatalf("unknown customer: %+v, %v; want no sales", empty, apiErr)
	}
}

func idsOf(lines []domain.SalesEntry) []string {
	ids := make([]string, len(lines))
	for i, l := range lines {
		ids[i] = l.ID
	}
	return ids
}
