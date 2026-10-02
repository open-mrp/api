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
	"github.com/open-mrp/api/shared/pagination"
)

// The open credits are the transaction corpus's few dozen not fully allocated, funded transactions: the
// finance corpus applies half of them in part and leaves the rest untouched. Every filter is measured
// against that set, which the open-credits key reads in list order.
func openCreditPlanDims() []planDim[domain.ListOpenCreditsParams] {
	str := func(s string) *string { return &s }
	at := func(d time.Time) *time.Time { return &d }
	recent := planTxCreatedAt(planTxRows - 1)
	old := planTxCreatedAt(planTxRows / 4)
	type P = domain.ListOpenCreditsParams
	return []planDim[P]{
		{"customer", []planValue[P]{
			{"large", func(p *P) { p.CustomerIDs = []string{planTxCustomerID(0)} }},
			{"rare", func(p *P) { p.CustomerIDs = []string{planTxCustomerID(planTxCustomers - 1)} }},
		}},
		{"search", []planValue[P]{
			{"one", func(p *P) { p.SearchQuery = str(planTxNumber(planFinUnallocated + 1000)) }},
			{"every", func(p *P) { p.SearchQuery = str("plan") }},
		}},
		{"funds", []planValue[P]{
			{"last90d", func(p *P) { p.StartDate = at(recent.Add(-90 * 24 * time.Hour)) }},
			{"old90d", func(p *P) { p.StartDate, p.EndDate = at(old), at(old.Add(90*24*time.Hour)) }},
		}},
	}
}

// openCreditFloor is how many open credits a search matches, or 0 without one: a contains match on the
// number, note, or customer name is no key's prefix, so the best plan reads every open credit to find
// them. The open-credits key bounds that to the account's open credits, not its transactions.
func openCreditFloor(t *testing.T, db *sql.DB, p domain.ListOpenCreditsParams) float64 {
	t.Helper()
	if p.SearchQuery == nil || strings.TrimSpace(*p.SearchQuery) == "" {
		return 0
	}
	var n float64
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM `transaction` WHERE account_id = ? AND is_fully_allocated = 0 AND funds_received_at IS NOT NULL",
		p.AccountID).Scan(&n))
	return n
}

func TestOpenCreditList_ReadsAboutAPage(t *testing.T) {
	ensureFinanceCorpus(t)
	type P = domain.ListOpenCreditsParams
	mid := planTxCreatedAt(planTxRows / 2)
	cursor := func(dir pagination.Direction) func(*P) {
		return func(p *P) { p.Cursor = cursorAt(mid, dir) }
	}
	listPlanSuite[P]{
		table: "transaction", scopeColumn: "account_id",
		from: "FROM `transaction` t", alias: "t",
		cases: planCases(P{AccountID: planTxAccount, Limit: 10}, openCreditPlanDims(), []planValue[P]{
			{"first", func(*P) {}},
			{"deep-next", cursor(pagination.DirectionForward)},
			{"deep-prev", cursor(pagination.DirectionBackward)},
		}),
		limit: func(p P) int32 { return p.Limit },
		list: func(ctx context.Context, q *sqlc.Queries, p P) error {
			if _, apiErr := NewTransactionAllocationRepo(q).ListOpenCredits(ctx, p); apiErr != nil {
				return apiErr
			}
			return nil
		},
		floor: openCreditFloor,
	}.run(t)
}
