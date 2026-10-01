//go:build plans

package repository

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	dbpkg "github.com/open-mrp/api/shared/db"
	"github.com/open-mrp/api/shared/pagination"
)

// ListAccountTransactions always names a customer, so the customer is the base of every case rather
// than one of its filters: a large one, the rare one, and each family (planFinFamilies).
func accountTransactionPlanDims() []planDim[domain.ListAccountTransactionsParams] {
	str := func(s string) *string { return &s }
	type P = domain.ListAccountTransactionsParams
	return []planDim[P]{
		{"status", []planValue[P]{
			{"allocated", func(p *P) { p.Status = str("allocated") }},
			{"unallocated", func(p *P) { p.Status = str("unallocated") }},
		}},
		{"type", []planValue[P]{
			{"payment", func(p *P) { p.Type = str("payment") }},
			{"rebate", func(p *P) { p.Type = str("rebate") }},
		}},
		{"search", []planValue[P]{
			{"one", func(p *P) { p.Query = str(planTxRareSearch) }},
			{"every", func(p *P) { p.Query = str(planTxDenseSearch) }},
		}},
	}
}

// accountTransactionFloor is how many transactions the request's unordered filter matches, or 0 when it
// has none. A number search is answered from its FULLTEXT key, which reads every match. A customer with
// children lists several customers, and no key yields several customers' transactions in list order,
// so the best plan reads the family's matching transactions and sorts them.
func accountTransactionFloor(t *testing.T, db *sql.DB, p domain.ListAccountTransactionsParams) float64 {
	t.Helper()
	where, args := []string{"account_id = ?"}, []any{p.AccountID}
	if term := dbpkg.AllWordsPrefixQuery(p.Query); term != "" {
		where, args = append(where, "MATCH(number) AGAINST(? IN BOOLEAN MODE)"), append(args, term)
	} else {
		var children int
		require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM account_relation child
			JOIN account_relation parent ON parent.id = child.parent_account_relation_id
			WHERE parent.owner_account_id = ? AND parent.counterparty_account_id = ?`, p.AccountID, p.CustomerAccountID).Scan(&children))
		if children == 0 {
			return 0
		}
		where = append(where, `customer_account_id IN (SELECT ? UNION SELECT child.counterparty_account_id FROM account_relation child
			JOIN account_relation parent ON parent.id = child.parent_account_relation_id
			WHERE parent.owner_account_id = ? AND parent.counterparty_account_id = ?)`)
		args = append(args, p.CustomerAccountID, p.AccountID, p.CustomerAccountID)
		if p.Status != nil {
			where = append(where, map[string]string{
				"allocated":   "is_fully_allocated = 1",
				"unallocated": "is_fully_allocated = 0 AND funds_received_at IS NOT NULL",
			}[*p.Status])
		}
		if p.Type != nil {
			where, args = append(where, "transaction_type_code = ?"), append(args, *p.Type)
		}
	}
	var n float64
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM `transaction` WHERE "+strings.Join(where, " AND "), args...).Scan(&n))
	return n
}

func TestAccountTransactionList_ReadsAboutAPage(t *testing.T) {
	ensureFinanceCorpus(t)
	type P = domain.ListAccountTransactionsParams
	mid := planTxCreatedAt(planTxRows / 2)
	cursor := func(dir pagination.Direction) func(*P) {
		return func(p *P) { p.Cursor = cursorAt(mid, dir) }
	}
	var cases []planCase[P]
	for _, customer := range []struct {
		label string
		id    string
	}{
		{"large", planTxCustomerID(0)},
		{"rare", planTxCustomerID(planTxCustomers - 1)},
		{"family", planTxCustomerID(planFinParent)},
		{"small-family", planTxCustomerID(planFinSmallParent)},
	} {
		for _, c := range planCases(P{AccountID: planTxAccount, CustomerAccountID: customer.id, Limit: 25, WithAllocations: true},
			accountTransactionPlanDims(), []planValue[P]{
				{"first", func(*P) {}},
				{"deep-next", cursor(pagination.DirectionForward)},
				{"deep-prev", cursor(pagination.DirectionBackward)},
			}) {
			c.name = customer.label + "," + c.name
			cases = append(cases, c)
		}
	}
	listPlanSuite[P]{
		table: "transaction", scopeColumn: "account_id",
		from: "FROM `transaction` t", alias: "t",
		cases: cases,
		limit: func(p P) int32 { return p.Limit },
		list: func(ctx context.Context, q *sqlc.Queries, p P) error {
			if _, apiErr := NewTransactionRepo(q).ListByCustomer(ctx, p); apiErr != nil {
				return apiErr
			}
			return nil
		},
		floor: accountTransactionFloor,
	}.run(t)
}

// TestTransactionGet_ReadsOneRow holds a transaction read (with its allocations, as the allocations
// include loads them) to its own row, for old, new, and open transactions.
func TestTransactionGet_ReadsOneRow(t *testing.T) {
	ensureFinanceCorpus(t)
	type P struct {
		id    string
		limit int32
	}
	var cases []planCase[P]
	for _, i := range []int{0, planTxRows / 2, planTxRows - 1, planFinUnallocated} {
		cases = append(cases, planCase[P]{name: planTxID(i), params: P{id: planTxID(i), limit: 1}})
	}
	listPlanSuite[P]{
		table: "transaction", scopeColumn: "account_id",
		from: "FROM `transaction` t", alias: "t",
		cases: cases,
		limit: func(p P) int32 { return p.limit },
		list: func(ctx context.Context, q *sqlc.Queries, p P) error {
			repo := NewTransactionRepo(q)
			if _, apiErr := repo.Get(ctx, planTxAccount, p.id); apiErr != nil {
				return apiErr
			}
			if _, apiErr := repo.GetAllocations(ctx, p.id); apiErr != nil {
				return apiErr
			}
			return nil
		},
	}.run(t)
}
