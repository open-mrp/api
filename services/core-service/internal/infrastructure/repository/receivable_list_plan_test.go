//go:build plans

package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
)

// receivableCutoffs are an as-of date after every unpaid invoice and one before all but the oldest.
func receivableCutoffs() (recent, old time.Time) {
	return planSalesInvoicedAt(planSalesOrders - 1), planSalesInvoicedAt(planSalesOrders / 2)
}

// receivableAsOfFloor is how many unpaid invoices an as-of request reads before it can drop the
// settled ones, or 0 for a plain listing. A balance is computed per invoice, so no key can skip the
// invoices that were settled by the cutoff; reading only the account's unpaid ones before it is the bar.
func receivableAsOfFloor(t *testing.T, db *sql.DB, accountID string, customerID *string, cutoff *time.Time) float64 {
	t.Helper()
	if cutoff == nil {
		return 0
	}
	query := `SELECT COUNT(*) FROM invoice inv JOIN sales_order so ON so.id = inv.sales_order_id
		WHERE inv.account_id = ? AND inv.is_paid_in_full = false AND inv.created_at < ?`
	args := []any{accountID, *cutoff}
	if customerID != nil {
		query += " AND so.buyer_account_id = ?"
		args = append(args, *customerID)
	}
	var n float64
	require.NoError(t, db.QueryRow(query, args...).Scan(&n))
	return n
}

func receivablePlanDims[P any](setCutoff func(*P, *time.Time), setQuery func(*P, *string)) []planDim[P] {
	str := func(s string) *string { return &s }
	recent, old := receivableCutoffs()
	return []planDim[P]{
		{"cutoff", []planValue[P]{
			{"asof-recent", func(p *P) { setCutoff(p, &recent) }},
			{"asof-old", func(p *P) { setCutoff(p, &old) }},
		}},
		{"search", []planValue[P]{
			{"search-one", func(p *P) { setQuery(p, str(planSalesRareSearch)) }},
			{"search-every", func(p *P) { setQuery(p, str(planSalesDenseSearch)) }},
		}},
	}
}

// TestReceivableList_ReadsAboutAPage holds ListReceivables (the account's unpaid invoices, optionally
// as of a date) to reading about a page of invoices.
func TestReceivableList_ReadsAboutAPage(t *testing.T) {
	ensureSalesCorpus(t)
	type P = domain.ListReceivablesParams
	listPlanSuite[P]{
		table: "invoice", scopeColumn: "account_id",
		from: "FROM invoice inv", alias: "inv",
		cases: planCases(P{AccountID: planSalesAccount, Limit: 25},
			receivablePlanDims(func(p *P, c *time.Time) { p.CutoffDate = c }, func(p *P, q *string) { p.Query = q }),
			invoicePlanPages(func(p *P, c *string) { p.Cursor = c })),
		limit: func(p P) int32 { return p.Limit },
		list: func(ctx context.Context, q *sqlc.Queries, p P) error {
			if _, apiErr := NewReceivableRepo(q).List(ctx, p); apiErr != nil {
				return apiErr
			}
			return nil
		},
		floor: func(t *testing.T, db *sql.DB, p P) float64 {
			return receivableAsOfFloor(t, db, p.AccountID, nil, p.CutoffDate)
		},
	}.run(t)
}

// TestReceivableByCustomerList_ReadsAboutAPage holds ListReceivablesByCustomer to reading about a page
// of invoices.
func TestReceivableByCustomerList_ReadsAboutAPage(t *testing.T) {
	ensureSalesCorpus(t)
	type P = domain.ListReceivablesByCustomerParams
	dims := append([]planDim[P]{
		{"customer", []planValue[P]{
			{"mid", func(p *P) { p.CustomerAccountID = planSalesCustomerID(planSalesMidCustomer) }},
			{"rare", func(p *P) { p.CustomerAccountID = planSalesCustomerID(planSalesRareCustomer) }},
		}},
	}, receivablePlanDims(func(p *P, c *time.Time) { p.CutoffDate = c }, func(p *P, q *string) { p.Query = q })...)
	listPlanSuite[P]{
		table: "invoice", scopeColumn: "account_id",
		from: "FROM invoice inv", alias: "inv",
		cases: planCases(P{AccountID: planSalesAccount, CustomerAccountID: planSalesCustomerID(0), Limit: 25}, dims,
			invoicePlanPages(func(p *P, c *string) { p.Cursor = c })),
		limit: func(p P) int32 { return p.Limit },
		list: func(ctx context.Context, q *sqlc.Queries, p P) error {
			if _, apiErr := NewReceivableRepo(q).ListByCustomer(ctx, p); apiErr != nil {
				return apiErr
			}
			return nil
		},
		floor: func(t *testing.T, db *sql.DB, p P) float64 {
			return receivableAsOfFloor(t, db, p.AccountID, &p.CustomerAccountID, p.CutoffDate)
		},
	}.run(t)
}
