//go:build plans

package repository

import (
	"context"
	"database/sql"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/pagination"
)

const planInvoicePageLimit = 25

// salesInvoicePlanPages are the windows an invoice page is read in, each on its first page and a page
// from the middle of the window in both directions.
func salesInvoicePlanPages() []planValue[salesPlanRequest] {
	type window struct {
		label      string
		start, end time.Time
	}
	windows := []window{
		{"7d", time.Date(2026, 9, 23, 5, 0, 0, 0, time.UTC), time.Date(2026, 9, 30, 4, 59, 59, 999e6, time.UTC)},
		{"1y", time.Date(2025, 10, 1, 5, 0, 0, 0, time.UTC), time.Date(2026, 10, 1, 4, 59, 59, 999e6, time.UTC)},
		{"all-time", time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 1, 4, 59, 59, 999e6, time.UTC)},
	}
	var out []planValue[salesPlanRequest]
	for _, w := range windows {
		mid := w.start.Add(w.end.Sub(w.start) / 2)
		if mid.Before(planAnaOrigin) {
			mid = planAnaInvoicedAt(planAnaInvoices / 2)
		}
		for _, page := range []struct {
			label  string
			cursor *pagination.Direction
		}{{"first", nil}, {"deep-next", ptr(pagination.DirectionForward)}, {"deep-prev", ptr(pagination.DirectionBackward)}} {
			out = append(out, planValue[salesPlanRequest]{w.label + "/" + page.label, func(p *salesPlanRequest) {
				p.StartsAt, p.EndsAt = w.start, w.end
				p.cursor = nil
				if page.cursor != nil {
					c := pagination.EncodeStringCursor(pagination.StringCursor{OccurredAt: mid, ID: "iv_planana_~", Direction: *page.cursor})
					p.cursor = &c
				}
			}})
		}
	}
	return out
}

func ptr[T any](v T) *T { return &v }

func salesInvoicePlanCases() []planCase[salesPlanRequest] {
	return planCases(salesPlanRequest{SalesReportFilter: domain.SalesReportFilter{AccountID: planAnaAccount}}, salesPlanFilterDims(), salesInvoicePlanPages())
}

func (p salesPlanRequest) invoicePage(ctx context.Context, q *sqlc.Queries) (*domain.SalesInvoicePage, error) {
	page, apiErr := p.repo(q).GetInvoicePage(ctx, domain.AnalyzeSalesInvoicesParams{SalesReportFilter: p.SalesReportFilter, Limit: planInvoicePageLimit, Cursor: p.cursor})
	if apiErr != nil {
		return nil, apiErr
	}
	return page, nil
}

// salesInvoicePageFloor is an invoice page's scope. It is a list, so it should read about a page: a
// page of invoices, the lines that chose them, and the page's lines again for their totals.
//
// The lines that choose a page are those of the narrowest single filter between the cursor and the
// page's last invoice: every other filter is residual, so a walk reads its filter's lines until the
// page fills (to the window's end when it never does, which no key avoids). A filter of several values
// is walked a page per value, since any one of them might hold the whole page; that is the most it
// needs, but never more than its every match.
func salesInvoicePageFloor(t *testing.T, db *sql.DB, p salesPlanRequest) map[string]float64 {
	t.Helper()
	s := salesScopeOf(t, db, p)
	q, apiErr := (&salesReportRepoImpl{queries: sqlc.New(db)}).newSalesFactQuery(context.Background(), p.SalesReportFilter, nil)
	require.Nil(t, apiErr)
	values := 1
	for i, f := range q.filters {
		if i == 0 || len(f.ids) < values {
			values = len(f.ids)
		}
	}
	start, end, backward := p.StartsAt, p.EndsAt, false
	if p.cursor != nil {
		c, err := pagination.DecodeStringCursor(*p.cursor)
		require.NoError(t, err)
		if backward = c.Direction == pagination.DirectionBackward; backward {
			start = c.OccurredAt
		} else {
			end = c.OccurredAt
		}
	}
	matches := s.facts(t, db, []rawRange{{from: start, to: end, toInclusive: true}})
	probeFrom, probeTo := start, end
	page, apiErr := p.repo(sqlc.New(db)).GetInvoicePage(context.Background(), domain.AnalyzeSalesInvoicesParams{SalesReportFilter: p.SalesReportFilter, Limit: planInvoicePageLimit, Cursor: p.cursor})
	require.Nil(t, apiErr)
	if len(page.Invoices) == planInvoicePageLimit {
		// Pages read newest first, so a forward page ends at its oldest invoice and a backward one (from
		// a cursor older than it) at its newest.
		if backward {
			end = page.Invoices[0].InvoicedAt
		} else {
			start = page.Invoices[len(page.Invoices)-1].InvoicedAt
		}
	}
	spanned := s.facts(t, db, []rawRange{{from: start, to: end, toInclusive: true}})
	var pageLines float64
	if len(page.Invoices) > 0 {
		ids := make([]any, len(page.Invoices))
		for i, inv := range page.Invoices {
			ids[i] = inv.InvoiceID
		}
		require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM sales_line_fact WHERE invoice_id IN ("+placeholders(len(ids))+")", ids...).Scan(&pageLines))
	}
	// Choosing which filter to walk counts each one's matches, up to a page per value.
	var probes float64
	if len(q.filters) > 1 {
		for _, f := range q.filters {
			var n float64
			args := append(append([]any{planAnaAccount}, stringsToAny(f.ids)...), probeFrom, probeTo)
			require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM sales_line_fact WHERE account_id = ? AND "+strings.TrimPrefix(f.column, "f.")+
				" IN ("+placeholders(len(f.ids))+") AND invoiced_at >= ? AND invoiced_at <= ?", args...).Scan(&n))
			probes += math.Min(n, float64(len(f.ids)*(planInvoicePageLimit+1)))
		}
	}
	return map[string]float64{
		"i": planInvoicePageLimit + 1,
		"f": math.Min(matches, math.Max(spanned, float64(values)*planRowBudget(planInvoicePageLimit))) + pageLines + probes,
	}
}

// TestSalesInvoicePage_ReadsItsScope holds every filter pair on every page of the invoice list to
// reading about a page of invoices and lines (aggregatePlanSuite, with a list's floor).
func TestSalesInvoicePage_ReadsItsScope(t *testing.T) {
	ensureAnalyticsInvoices(t)
	aggregatePlanSuite[salesPlanRequest]{
		tables: []aggregateTable{
			{table: "sales_line_fact", scopeColumn: "account_id", from: "FROM sales_line_fact f", alias: "f"},
			{table: "invoice", scopeColumn: "account_id", from: "FROM invoice i", alias: "i"},
		},
		cases: salesInvoicePlanCases(),
		report: func(ctx context.Context, q *sqlc.Queries, p salesPlanRequest) error {
			_, err := p.invoicePage(ctx, q)
			return err
		},
		floor: salesInvoicePageFloor,
	}.run(t)
}

// TestSalesInvoicePage_MatchesTheInvoiceWalk pins the page chosen from a filter's lines to the page the
// invoice walk chooses for the same filter, which is how every page was read before.
func TestSalesInvoicePage_MatchesTheInvoiceWalk(t *testing.T) {
	ensureAnalyticsInvoices(t)
	db := planDB(t)
	ctx := context.Background()
	repo := &salesReportRepoImpl{queries: sqlc.New(db)}
	ids := func(query string, args []any) []string {
		rows, err := db.Query(query, args...)
		require.NoError(t, err)
		defer func() { _ = rows.Close() }()
		var out []string
		for rows.Next() {
			var id, number, buyer, name string
			var at time.Time
			require.NoError(t, rows.Scan(&id, &number, &at, &buyer, &name))
			out = append(out, strings.Join([]string{id, number, at.Format(time.RFC3339Nano), buyer, name}, "|"))
		}
		require.NoError(t, rows.Err())
		return out
	}
	for _, c := range salesInvoicePlanCases() {
		t.Run(c.name, func(t *testing.T) {
			params := domain.AnalyzeSalesInvoicesParams{SalesReportFilter: c.params.SalesReportFilter, Limit: planInvoicePageLimit, Cursor: c.params.cursor}
			cursor, apiErr := keysetCursor(params.Cursor)
			require.Nil(t, apiErr)
			q, apiErr := repo.newSalesFactQuery(ctx, params.SalesReportFilter, nil)
			require.Nil(t, apiErr)
			if q.empty {
				return
			}
			walk := *q
			walk.filters = nil
			want := ids(invoicePageQuery(&walk, 0, params, cursor))
			// Whichever filter the page walks, it is the same page.
			for pin := range q.filters {
				require.Equal(t, want, ids(invoicePageQuery(q, pin, params, cursor)), "walking filter %d", pin)
			}
		})
	}
}
