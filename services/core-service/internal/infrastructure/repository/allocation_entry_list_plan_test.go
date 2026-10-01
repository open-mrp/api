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
	dbpkg "github.com/open-mrp/api/shared/db"
	"github.com/open-mrp/api/shared/pagination"
)

func allocationEntryPlanDims() []planDim[domain.ListAllocationEntriesParams] {
	str := func(s string) *string { return &s }
	at := func(d time.Time) *time.Time { return &d }
	recent := planTxCreatedAt(planTxRows - 1)
	old := planTxCreatedAt(planTxRows / 4)
	type P = domain.ListAllocationEntriesParams
	return []planDim[P]{
		{"type", []planValue[P]{
			{"payment", func(p *P) { p.TransactionType = str("payment") }},
			{"rebate", func(p *P) { p.TransactionType = str("rebate") }},
		}},
		{"search", []planValue[P]{
			{"tx-number", func(p *P) { p.Query = str(planTxRareSearch) }},
			{"invoice-number", func(p *P) { p.Query = str(planFinInvoiceNumber(100)) }},
			{"tail-customer", func(p *P) { p.Query = str("Customer 0500") }},
			{"large-customer", func(p *P) { p.Query = str("Customer 0005") }},
			{"every", func(p *P) { p.Query = str("Plan") }},
			{"none", func(p *P) { p.Query = str("zzz") }},
		}},
		{"created", []planValue[P]{
			{"last30d", func(p *P) { p.StartDate = at(recent.Add(-30 * 24 * time.Hour)) }},
			{"old30d", func(p *P) { p.StartDate, p.EndDate = at(old), at(old.Add(30*24*time.Hour)) }},
		}},
	}
}

// allocationEntryFloor is how many allocations a search matches, or 0 without one. A search matches a
// customer by part of its name, which no key orders, so the best plan reads the search's matches (or,
// for a common one, walks the list until a page of them turns up).
func allocationEntryFloor(t *testing.T, db *sql.DB, p domain.ListAllocationEntriesParams) float64 {
	t.Helper()
	if p.Query == nil {
		return 0
	}
	var n float64
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM transaction_allocation ta
		JOIN invoice inv ON inv.id = ta.invoice_id
		JOIN `+"`transaction`"+` t ON t.id = ta.transaction_id
		JOIN account cust_acct ON cust_acct.id = t.customer_account_id
		LEFT JOIN account_relation ar ON ar.counterparty_account_id = t.customer_account_id
			AND ar.owner_account_id = t.account_id AND ar.account_relation_role_code = 'customer'
		WHERE t.account_id = ? AND (inv.number = ? OR t.number = ? OR ar.external_number = ? OR cust_acct.name LIKE CONCAT('%', ?, '%'))`,
		p.AccountID, *p.Query, *p.Query, *p.Query, dbpkg.EscapeLike(*p.Query)).Scan(&n))
	return n
}

func TestAllocationEntryList_ReadsAboutAPage(t *testing.T) {
	ensureFinanceCorpus(t)
	type P = domain.ListAllocationEntriesParams
	mid := planTxCreatedAt(planTxRows / 2)
	cursor := func(dir pagination.Direction) func(*P) {
		return func(p *P) { p.Cursor = cursorAt(mid, dir) }
	}
	var cases []planCase[P]
	for _, tenant := range []struct{ label, account string }{{"large", planTxAccount}, {"small", planFinSmallAccount}} {
		for _, c := range planCases(P{AccountID: tenant.account, Limit: 25}, allocationEntryPlanDims(), []planValue[P]{
			{"first", func(*P) {}},
			{"deep-next", cursor(pagination.DirectionForward)},
			{"deep-prev", cursor(pagination.DirectionBackward)},
		}) {
			c.name = tenant.label + "," + c.name
			cases = append(cases, c)
		}
	}
	listPlanSuite[P]{
		table: "transaction_allocation", scopeColumn: "account_id",
		from: "FROM transaction_allocation ta", alias: "ta",
		cases: cases,
		limit: func(p P) int32 { return p.Limit },
		list: func(ctx context.Context, q *sqlc.Queries, p P) error {
			if _, apiErr := NewTransactionAllocationRepo(q).ListEntries(ctx, p); apiErr != nil {
				return apiErr
			}
			return nil
		},
		floor: allocationEntryFloor,
	}.run(t)
}
