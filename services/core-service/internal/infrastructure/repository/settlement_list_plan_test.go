//go:build plans

package repository

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	dbpkg "github.com/open-mrp/api/shared/db"
	"github.com/open-mrp/api/shared/pagination"
)

// planFinRecentTransactions are the newest allocated transactions: together they appear in a few
// dozen of the newest settlements.
func planFinRecentTransactions() []string {
	var ids []string
	for i := planTxRows - 1; len(ids) < 100; i-- {
		if i%2000 != planFinUnallocated {
			ids = append(ids, planTxID(i))
		}
	}
	return ids
}

func planFinRecentInvoices() []string {
	var ids []string
	for n := planFinInvoices - 101; n < planFinInvoices-1; n++ {
		ids = append(ids, planFinInvoiceID(n))
	}
	return ids
}

func settlementPlanDims() []planDim[domain.ListSettlementsParams] {
	str := func(s string) *string { return &s }
	at := func(d time.Time) *time.Time { return &d }
	recent := planTxCreatedAt(planTxRows - 1)
	old := planTxCreatedAt(planTxRows / 4)
	type P = domain.ListSettlementsParams
	return []planDim[P]{
		{"search", []planValue[P]{
			{"one", func(p *P) { p.Query = str(planTxRareSearch) }},
			{"every", func(p *P) { p.Query = str(planTxDenseSearch) }},
		}},
		{"transaction", []planValue[P]{
			{"tx-recent100", func(p *P) { p.TransactionIDs = planFinRecentTransactions() }},
			{"tx-old", func(p *P) { p.TransactionIDs = []string{planTxID(100)} }},
			{"tx-unsettled", func(p *P) { p.TransactionIDs = []string{planTxID(planFinUnallocated)} }},
		}},
		{"invoice", []planValue[P]{
			{"inv-recent100", func(p *P) { p.InvoiceIDs = planFinRecentInvoices() }},
			{"inv-old", func(p *P) { p.InvoiceIDs = []string{planFinInvoiceID(100)} }},
			{"inv-unpaid", func(p *P) { p.InvoiceIDs = []string{planFinNoInvoice} }},
		}},
		{"created", []planValue[P]{
			{"last30d", func(p *P) { p.StartDate = at(recent.Add(-30 * 24 * time.Hour)) }},
			{"old30d", func(p *P) { p.StartDate, p.EndDate = at(old), at(old.Add(30*24*time.Hour)) }},
		}},
	}
}

// settlementFilterFloor is how many settlements the request's unordered filter matches, or 0 when it
// has none. A number search is answered from its FULLTEXT key, which reads every match whatever else
// the request filters by, so a search is held to its matches. Otherwise a transaction or invoice
// filter (a property of the settlement's allocations) is held to the settlements it names.
func settlementFilterFloor(t *testing.T, db *sql.DB, p domain.ListSettlementsParams) float64 {
	t.Helper()
	where, args := []string{"s.account_id = ?"}, []any{p.AccountID}
	switch {
	case dbpkg.AllWordsPrefixQuery(p.Query) != "":
		where, args = append(where, "MATCH(s.number) AGAINST(? IN BOOLEAN MODE)"), append(args, dbpkg.AllWordsPrefixQuery(p.Query))
	case len(p.TransactionIDs) > 0 || len(p.InvoiceIDs) > 0:
		sub := "s.id IN (SELECT settlement_id FROM transaction_allocation WHERE TRUE"
		if len(p.TransactionIDs) > 0 {
			sub += " AND transaction_id IN (" + placeholders(len(p.TransactionIDs)) + ")"
			args = append(args, stringArgs(p.TransactionIDs)...)
		}
		if len(p.InvoiceIDs) > 0 {
			sub += " AND invoice_id IN (" + placeholders(len(p.InvoiceIDs)) + ")"
			args = append(args, stringArgs(p.InvoiceIDs)...)
		}
		where = append(where, sub+")")
	default:
		return 0
	}
	var n float64
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM settlement s WHERE "+strings.Join(where, " AND "), args...).Scan(&n))
	return n
}

func TestSettlementList_ReadsAboutAPage(t *testing.T) {
	ensureFinanceCorpus(t)
	type P = domain.ListSettlementsParams
	mid := planTxCreatedAt(planTxRows / 2)
	cursor := func(dir pagination.Direction) func(*P) {
		return func(p *P) { p.Cursor = cursorAt(mid, dir) }
	}
	listPlanSuite[P]{
		table: "settlement", scopeColumn: "account_id",
		from: "FROM settlement s", alias: "s",
		cases: planCases(P{AccountID: planTxAccount, Limit: 25}, settlementPlanDims(), []planValue[P]{
			{"first", func(*P) {}},
			{"deep-next", cursor(pagination.DirectionForward)},
			{"deep-prev", cursor(pagination.DirectionBackward)},
		}),
		limit: func(p P) int32 { return p.Limit },
		list: func(ctx context.Context, q *sqlc.Queries, p P) error {
			if _, apiErr := NewSettlementRepo(q).List(ctx, p); apiErr != nil {
				return apiErr
			}
			return nil
		},
		floor: settlementFilterFloor,
	}.run(t)
}
