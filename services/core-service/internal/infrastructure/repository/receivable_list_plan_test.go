//go:build plans

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/pagination"
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
		// Read from the smaller of the account's unpaid invoices and the customer's orders, as
		// ListCustomerInvoices is.
		floor: func(t *testing.T, db *sql.DB, p P) float64 {
			return customerInvoiceFloor(t, db, domain.ListCustomerInvoicesParams{AccountID: p.AccountID, CustomerAccountID: p.CustomerAccountID})
		},
	}.run(t)
}

// TestReceivableByCustomerList_MatchesEveryUnpaidInvoice pins ListReceivablesByCustomer, whichever set it
// reads, to the customer's unpaid invoices and their balances, in list order across pages both ways.
func TestReceivableByCustomerList_MatchesEveryUnpaidInvoice(t *testing.T) {
	ensureSalesCorpus(t)
	db := planDB(t)
	repo := NewReceivableRepo(sqlc.New(db))
	str := func(s string) *string { return &s }
	recent, old := receivableCutoffs()
	type listCase struct {
		customer int
		cutoff   *time.Time
		query    *string
	}
	cases := map[string]listCase{
		"large":        {customer: 0},
		"mid":          {customer: planSalesMidCustomer},
		"rare":         {customer: planSalesRareCustomer},
		"parent":       {customer: planSalesParentCustomer},
		"asof recent":  {customer: 0, cutoff: &recent},
		"asof old":     {customer: 2, cutoff: &old},
		"search dense": {customer: 0, query: str(planSalesDenseSearch)},
		"search name":  {customer: planSalesMidCustomer, query: str(planSalesCustomerName(planSalesMidCustomer))},
		"search asof":  {customer: 0, cutoff: &recent, query: str(planSalesDenseSearch)},
	}
	const limit = 5
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			customerID := planSalesCustomerID(c.customer)
			where := "inv.account_id = ? AND so.buyer_account_id = ? AND inv.is_paid_in_full = false"
			args := []any{buildAllocationCutoffParam(c.cutoff), planSalesAccount, customerID}
			having := ""
			if c.cutoff != nil {
				where += " AND inv.created_at < ?"
				args = append(args, *c.cutoff)
				having = " HAVING remaining_balance > 0"
			}
			if c.query != nil {
				like := buildReceivableSearchParam(c.query).String
				where += " AND (inv.number LIKE ? OR buyer.name LIKE ?)"
				args = append(args, like, like)
			}
			rows, err := db.QueryContext(ctx, `SELECT inv.id, inv.created_at, `+receivableBalanceSQL+` AS remaining_balance
FROM invoice inv JOIN sales_order so ON so.id = inv.sales_order_id
JOIN account_relation ar ON ar.owner_account_id = inv.account_id AND ar.counterparty_account_id = so.buyer_account_id
    AND ar.account_relation_role_code = 'customer'
JOIN account buyer ON buyer.id = so.buyer_account_id
WHERE `+where+having+" ORDER BY inv.created_at DESC, inv.id DESC", args...)
			require.NoError(t, err)
			var want []string
			var createdAt []time.Time
			for rows.Next() {
				var id string
				var at time.Time
				var balance float64
				require.NoError(t, rows.Scan(&id, &at, &balance))
				want = append(want, fmt.Sprintf("%s %.2f", id, balance))
				createdAt = append(createdAt, at)
			}
			require.NoError(t, rows.Close())
			require.NotEmpty(t, want, "a case must match what it names")

			params := domain.ListReceivablesByCustomerParams{AccountID: planSalesAccount, CustomerAccountID: customerID,
				CutoffDate: c.cutoff, Query: c.query, Limit: limit}
			var got []string
			for {
				res, apiErr := repo.ListByCustomer(ctx, params)
				require.Nil(t, apiErr)
				for _, e := range res.Items {
					got = append(got, e.InvoiceID+" "+e.RemainingBalance)
				}
				if res.PageString == nil {
					break
				}
				params.Cursor = res.PageString
			}
			require.Equal(t, want, got)

			// Back from the second page's first row is the first page.
			if len(want) > limit {
				back := pagination.EncodeStringCursor(pagination.StringCursor{OccurredAt: createdAt[limit],
					ID: strings.Fields(want[limit])[0], Direction: pagination.DirectionBackward})
				params.Cursor = &back
				res, apiErr := repo.ListByCustomer(ctx, params)
				require.Nil(t, apiErr)
				var first []string
				for _, e := range res.Items {
					first = append(first, e.InvoiceID+" "+e.RemainingBalance)
				}
				require.Equal(t, want[:limit], first)
			}

			if c.query == nil {
				all, apiErr := repo.ListAllByCustomer(ctx, planSalesAccount, customerID, c.cutoff)
				require.Nil(t, apiErr)
				require.Len(t, all, len(want))
			}
		})
	}
}
