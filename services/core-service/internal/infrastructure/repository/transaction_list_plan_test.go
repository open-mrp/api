//go:build plans

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/pagination"
)

// The transaction corpus is one merchant shaped like the largest production tenant: tens of thousands
// of transactions, nearly all allocated payments by check, a few dozen still open, a handful of
// customers holding most of the volume, and long tails of rare types, methods, and customers. The
// skew is the point. A uniform corpus lets every index look equally good; this one gives each filter
// a dense value and a rare one, which is where a plan that cannot stop at the page shows itself.
//
// Row widths follow production too (ID lengths, short numbers, a responsible user on most rows, funds
// received on nearly all): the optimizer prices an ordered index walk against a scan-and-sort by how
// many pages the rows fill, and a narrow corpus tips it toward the walk production does not take.
const (
	planTxAccount   = "ac_plantx"
	planTxRows      = 30_000
	planTxCustomers = 1_000
	planTxUsers     = 50
	planTxSpan      = 4 * 365 * 24 * time.Hour

	// planTxCorpusVersion is bumped whenever the corpus's shape changes, so a stale one is rebuilt.
	planTxCorpusVersion = "Plan Test Merchant v2"

	// planTxRareSearch matches one transaction number; planTxDenseSearch matches over a third of them.
	planTxRareSearch  = "12345"
	planTxDenseSearch = "1"
)

func planTxID(i int) string            { return fmt.Sprintf("tx_plantx_%016d", i) }
func planTxNumber(i int) string        { return fmt.Sprintf("%d", 1000+i) }
func planTxCustomerID(c int) string    { return fmt.Sprintf("%s_c%018d", planTxAccount, c) }
func planTxAccountUserID(u int) string { return fmt.Sprintf("acus_plantx_%014d", u) }

var planTxOrigin = time.Date(2022, 9, 1, 0, 0, 0, 0, time.UTC)

func planTxCreatedAt(i int) time.Time {
	return planTxOrigin.Add(time.Duration(i) * (planTxSpan / planTxRows))
}

// planTxCustomer gives half the volume to 20 large customers and spreads the rest over the tail.
// The last customer is rare: it holds only the rows planTxRareCustomerRows names.
func planTxCustomer(i int) int {
	if planTxRareCustomerRows[i] {
		return planTxCustomers - 1
	}
	if i%10 < 5 {
		return (i / 10) % 20
	}
	return 20 + (i*7919)%(planTxCustomers-21)
}

var planTxRareCustomerRows = map[int]bool{100: true, 15_000: true, 29_990: true}

func planTxType(i int) (typeCode string, method, adjustment *string) {
	str := func(s string) *string { return &s }
	switch {
	case i%1000 == 999:
		return "rebate", nil, nil
	case i%100 >= 94 && i%100 <= 96:
		return "adjustment", nil, str([]string{"discount", "fee", "write_off"}[i%100-94])
	case i%100 == 97 || i%100 == 98:
		return "credit_memo", nil, nil
	case i%997 == 0:
		return "payment", str("gift_card"), nil
	case i%5 == 0:
		return "payment", str("ach"), nil
	default:
		return "payment", str("check"), nil
	}
}

var planTxCorpusOnce sync.Once

func ensureTransactionCorpus(t *testing.T) {
	t.Helper()
	db := planDB(t)
	planTxCorpusOnce.Do(func() {
		var have int
		require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM `transaction` WHERE account_id = ?", planTxAccount).Scan(&have))
		var version string
		_ = db.QueryRow("SELECT name FROM account WHERE id = ?", planTxAccount).Scan(&version)
		if have >= planTxRows && version == planTxCorpusVersion {
			return
		}
		t.Logf("seeding the transaction plan corpus (%d transactions); it is kept for later runs", planTxRows)

		exec := func(query string, args ...any) {
			_, err := db.Exec(query, args...)
			require.NoError(t, err)
		}
		exec("DELETE FROM `transaction` WHERE account_id = ?", planTxAccount)
		exec("DELETE FROM quantity WHERE id LIKE 'qy\\_plantx\\_%'")
		exec("DELETE FROM account_relation WHERE owner_account_id = ?", planTxAccount)
		exec("DELETE FROM account WHERE id LIKE 'ac\\_plantx\\_%'")
		exec(`INSERT IGNORE INTO unit (id, name, abbreviation, unit_dimension_code, account_id,
		                               ratio_numerator, ratio_denominator, is_base_unit, created_at, updated_at)
		      VALUES ('un_plantx', 'plantest', 'un_plantx', 'quantity', NULL, 1, 1, 0, NOW(3), NOW(3))`)
		exec(`INSERT INTO account (id, name, account_type_code, onboarding_status_code) VALUES (?, ?, 'company', 'active')
		      ON DUPLICATE KEY UPDATE name = VALUES(name)`, planTxAccount, planTxCorpusVersion)
		for u := range planTxUsers {
			userID := fmt.Sprintf("us_plantx_%016d", u)
			exec("INSERT IGNORE INTO `user` (id) VALUES (?)", userID)
			exec(`INSERT IGNORE INTO account_user (id, user_id, account_id) VALUES (?, ?, ?)`, planTxAccountUserID(u), userID, planTxAccount)
		}
		for _, g := range []string{"big", "small"} {
			exec(`INSERT IGNORE INTO account_group (id, owner_account_id, name, commission_status_code, freight_status_code, account_group_type_code)
			      VALUES (?, ?, ?, 'commission_applied', 'billed_freight', 'type_group')`, "ag_plantx_"+g, planTxAccount, "Plan "+g)
		}

		var accVals, relVals []string
		var accArgs, relArgs []any
		for c := range planTxCustomers {
			id := planTxCustomerID(c)
			var group any
			switch {
			case c < 300:
				group = "ag_plantx_big"
			case c == planTxCustomers-1:
				group = "ag_plantx_small"
			}
			accVals = append(accVals, "(?, ?, 'company', 'unclaimed')")
			accArgs = append(accArgs, id, fmt.Sprintf("Plan Customer %04d", c))
			relVals = append(relVals, "(?, ?, ?, 'customer', ?, 'normal', ?)")
			relArgs = append(relArgs, fmt.Sprintf("ar_plantx_%04d", c), planTxAccount, id, fmt.Sprintf("C%04d", c), group)
		}
		exec(`INSERT IGNORE INTO account (id, name, account_type_code, onboarding_status_code) VALUES `+strings.Join(accVals, ","), accArgs...)
		exec(`INSERT IGNORE INTO account_relation (id, owner_account_id, counterparty_account_id, account_relation_role_code, external_number, priority_code, account_group_id)
		      VALUES `+strings.Join(relVals, ","), relArgs...)

		const batch = 1_000
		for start := 0; start < planTxRows; start += batch {
			var qVals, tVals []string
			var qArgs, tArgs []any
			for i := start; i < start+batch; i++ {
				createdAt := planTxCreatedAt(i)
				typeCode, method, adjustment := planTxType(i)
				var responsible, note any
				if i%15 != 0 {
					responsible = planTxAccountUserID(i % planTxUsers)
				}
				if i%10 == 3 {
					note = "check " + planTxNumber(i)
				}
				qID := fmt.Sprintf("qy_plantx_%016d", i)
				qVals = append(qVals, "(?, '100', 'un_plantx', ?, ?)")
				qArgs = append(qArgs, qID, createdAt, createdAt)
				tVals = append(tVals, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
				tArgs = append(tArgs,
					planTxID(i), planTxNumber(i), planTxAccount, planTxCustomerID(planTxCustomer(i)), qID,
					typeCode, method, adjustment, i%1000 != 500, responsible, note,
					createdAt.Add(24*time.Hour), createdAt, createdAt)
			}
			exec(`INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES `+strings.Join(qVals, ","), qArgs...)
			exec("INSERT IGNORE INTO `transaction` (id, number, account_id, customer_account_id, amount_id, transaction_type_code, "+
				"transaction_method_code, adjustment_type_code, is_fully_allocated, responsible_user_id, note, funds_received_at, created_at, updated_at) VALUES "+
				strings.Join(tVals, ","), tArgs...)
		}
		exec("ANALYZE TABLE `transaction`, account_relation, account_user")
	})
}

// planValue is one setting of one list filter.
type planValue struct {
	label string
	apply func(*domain.ListTransactionsParams)
}

type planDim struct {
	name   string
	values []planValue
}

func transactionPlanDims() []planDim {
	str := func(s string) *string { return &s }
	at := func(d time.Time) *time.Time { return &d }
	recent := planTxCreatedAt(planTxRows - 1)
	old := planTxCreatedAt(planTxRows / 4)
	return []planDim{
		{"status", []planValue{
			{"allocated", func(p *domain.ListTransactionsParams) { p.Status = str("allocated") }},
			{"unallocated", func(p *domain.ListTransactionsParams) { p.Status = str("unallocated") }},
		}},
		{"type", []planValue{
			{"payment", func(p *domain.ListTransactionsParams) { p.TypeCodes = []string{"payment"} }},
			{"rebate", func(p *domain.ListTransactionsParams) { p.TypeCodes = []string{"rebate"} }},
			{"payment+rebate", func(p *domain.ListTransactionsParams) { p.TypeCodes = []string{"payment", "rebate"} }},
		}},
		{"method", []planValue{
			{"check", func(p *domain.ListTransactionsParams) { p.MethodCodes = []string{"check"} }},
			{"gift_card", func(p *domain.ListTransactionsParams) { p.MethodCodes = []string{"gift_card"} }},
		}},
		{"adjustment", []planValue{
			{"write_off", func(p *domain.ListTransactionsParams) { p.AdjustmentTypeCodes = []string{"write_off"} }},
			{"refund(none)", func(p *domain.ListTransactionsParams) { p.AdjustmentTypeCodes = []string{"refund"} }},
		}},
		{"customer", []planValue{
			{"large", func(p *domain.ListTransactionsParams) { p.CustomerIDs = []string{planTxCustomerID(0)} }},
			{"rare", func(p *domain.ListTransactionsParams) {
				p.CustomerIDs = []string{planTxCustomerID(planTxCustomers - 1)}
			}},
		}},
		{"group", []planValue{
			{"big", func(p *domain.ListTransactionsParams) { p.CustomerGroupIDs = []string{"ag_plantx_big"} }},
			{"small", func(p *domain.ListTransactionsParams) { p.CustomerGroupIDs = []string{"ag_plantx_small"} }},
		}},
		{"search", []planValue{
			{"one", func(p *domain.ListTransactionsParams) { p.Query = str(planTxRareSearch) }},
			{"every", func(p *domain.ListTransactionsParams) { p.Query = str(planTxDenseSearch) }},
		}},
		{"funds", []planValue{
			{"last30d", func(p *domain.ListTransactionsParams) { p.StartDate = at(recent.Add(-30 * 24 * time.Hour)) }},
			{"old30d", func(p *domain.ListTransactionsParams) {
				p.StartDate, p.EndDate = at(old), at(old.Add(30*24*time.Hour))
			}},
		}},
	}
}

// planCase is one list request: a filter combination on one page.
type planCase struct {
	name   string
	params domain.ListTransactionsParams
}

// transactionPlanCases is every filter value alone and every pair of values across two filters, each
// on the first page and on a page deep in the account in both directions. Pairs are enough: the
// guarantee is that some index pins the most selective filter and every other filter is residual.
func transactionPlanCases() []planCase {
	dims := transactionPlanDims()
	var combos [][]planValue
	combos = append(combos, nil)
	for i, d := range dims {
		for _, v := range d.values {
			combos = append(combos, []planValue{v})
			for _, d2 := range dims[i+1:] {
				for _, v2 := range d2.values {
					combos = append(combos, []planValue{v, v2})
				}
			}
		}
	}

	mid := planTxCreatedAt(planTxRows / 2)
	pages := []struct {
		label  string
		cursor *string
	}{
		{"first", nil},
		{"deep-next", cursorAt(mid, pagination.DirectionForward)},
		{"deep-prev", cursorAt(mid, pagination.DirectionBackward)},
	}

	var cases []planCase
	for _, combo := range combos {
		labels := []string{}
		for _, v := range combo {
			labels = append(labels, v.label)
		}
		if len(labels) == 0 {
			labels = []string{"unfiltered"}
		}
		for _, page := range pages {
			p := domain.ListTransactionsParams{AccountID: planTxAccount, Limit: 25, Cursor: page.cursor}
			for _, v := range combo {
				v.apply(&p)
			}
			cases = append(cases, planCase{name: strings.Join(labels, ",") + "/" + page.label, params: p})
		}
	}
	return cases
}

func cursorAt(at time.Time, dir pagination.Direction) *string {
	c := pagination.EncodeStringCursor(pagination.StringCursor{OccurredAt: at, ID: "tx_plantx_~", Direction: dir})
	return &c
}

// transactionPlanGap is a request shape the list is known to serve badly. Each is logged rather than
// failed, so the suite holds every other shape to the bar today; fixing one means deleting its entry.
type transactionPlanGap struct {
	reason  string
	matches func(domain.ListTransactionsParams) bool
}

var transactionPlanGaps = []transactionPlanGap{
	{
		reason: "a FULLTEXT match yields rows in relevance order, so a search matching most numbers is read whole and sorted",
		matches: func(p domain.ListTransactionsParams) bool {
			return p.Query != nil && *p.Query == planTxDenseSearch
		},
	},
	{
		reason: "TODO: the customer group filters the joined account_relation, so the planner drives from it through " +
			"transaction_customer_account_id_idx and reads every transaction of the group; resolve the group to " +
			"customer IDs first, as ListByCustomer does",
		matches: func(p domain.ListTransactionsParams) bool { return len(p.CustomerGroupIDs) > 0 },
	},
	{
		reason: "TODO: the funds-received range filters a column the list does not sort by, so the planner walks " +
			"created_at past every transaction outside the window; transaction_open_credits_idx ranges the window " +
			"and sorting it reads only the window",
		matches: func(p domain.ListTransactionsParams) bool { return p.StartDate != nil || p.EndDate != nil },
	},
	{
		reason: "TODO: the type, method, adjustment, and customer composites are descending, which InnoDB cannot scan " +
			"backward, so the previous page sorts the filter's whole range; rebuild them ascending",
		matches: func(p domain.ListTransactionsParams) bool {
			cur, _ := decodeTransactionCursor(p.Cursor)
			return cur != nil && cur.Direction == pagination.DirectionBackward &&
				(len(p.TypeCodes) > 0 || len(p.MethodCodes) > 0 || len(p.AdjustmentTypeCodes) > 0 || len(p.CustomerIDs) > 0)
		},
	},
}

func transactionPlanGapFor(p domain.ListTransactionsParams) *transactionPlanGap {
	for i := range transactionPlanGaps {
		if transactionPlanGaps[i].matches(p) {
			return &transactionPlanGaps[i]
		}
	}
	return nil
}

// planRowBudget is the most of the transaction table one page may read: the page, plus room for
// residual filters that reject some rows the driving index yields.
func planRowBudget(limit int32) float64 { return float64(10 * (limit + 1)) }

// TestTransactionList_ReadsAboutAPage measures every filter combination ListTransactions accepts
// against the corpus. Each request is replayed with every account index forced, and two things are
// asserted separately:
//   - the plan it got reads about what the best of those indexes reads: a miss is the planner (or a
//     hint) choosing badly, fixed in the query;
//   - the best index reads about a page: a miss is an index that does not exist, fixed in a migration.
//
// A request that returns less than it asked for is exempt from the second: whichever index drives it,
// finding there is no next row means reading that index's range to its end, which no index avoids.
func TestTransactionList_ReadsAboutAPage(t *testing.T) {
	ensureTransactionCorpus(t)
	db := planDB(t)
	edb := &explainingDB{db: db}
	repo := &transactionRepoImpl{queries: sqlc.New(edb)}
	indexes := accountIndexes(t, db, "transaction")

	for _, mode := range planStatsModes {
		t.Run("stats="+mode, func(t *testing.T) {
			usePlanStats(t, db, "transaction", mode)
			t.Cleanup(func() { usePlanStats(t, db, "transaction", "analyzed") })
			for _, tc := range transactionPlanCases() {
				t.Run(tc.name, func(t *testing.T) { checkTransactionPlan(t, db, edb, repo, indexes, tc) })
			}
		})
	}
}

func checkTransactionPlan(t *testing.T, db *sql.DB, edb *explainingDB, repo *transactionRepoImpl, indexes []string, tc planCase) {
	t.Helper()

	edb.statements = nil
	_, apiErr := repo.List(context.Background(), tc.params)
	require.Nil(t, apiErr)
	require.Len(t, edb.statements, 1)
	stmt := edb.statements[0]

	got := tableAccess(stmt.plan, "t")
	page := float64(tc.params.Limit + 1)
	if got.rows <= planRowBudget(tc.params.Limit) {
		return
	}
	best, bestIndex := bestForcedAccess(t, db, stmt, "FROM `transaction` t", "t", indexes)

	var problem string
	switch {
	case got.rows > 2*best.rows+page:
		problem = fmt.Sprintf("read %.0f transactions via %v; forcing %s reads %.0f", got.rows, got.indexes, bestIndex, best.rows)
	case best.rows > planRowBudget(tc.params.Limit) && planReturned(stmt.plan) >= page:
		problem = fmt.Sprintf("no index serves this: the best, %s, reads %.0f transactions to return a page of %d", bestIndex, best.rows, tc.params.Limit)
	default:
		return
	}
	if gap := transactionPlanGapFor(tc.params); gap != nil {
		t.Logf("known gap: %s (%s)", problem, gap.reason)
		return
	}
	t.Errorf("%s\n%s", problem, stmt.plan)
}
