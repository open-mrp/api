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
	dbpkg "github.com/open-mrp/api/shared/db"
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

func transactionPlanDims() []planDim[domain.ListTransactionsParams] {
	str := func(s string) *string { return &s }
	at := func(d time.Time) *time.Time { return &d }
	recent := planTxCreatedAt(planTxRows - 1)
	old := planTxCreatedAt(planTxRows / 4)
	return []planDim[domain.ListTransactionsParams]{
		{"status", []planValue[domain.ListTransactionsParams]{
			{"allocated", func(p *domain.ListTransactionsParams) { p.Status = str("allocated") }},
			{"unallocated", func(p *domain.ListTransactionsParams) { p.Status = str("unallocated") }},
		}},
		{"type", []planValue[domain.ListTransactionsParams]{
			{"payment", func(p *domain.ListTransactionsParams) { p.TypeCodes = []string{"payment"} }},
			{"rebate", func(p *domain.ListTransactionsParams) { p.TypeCodes = []string{"rebate"} }},
			{"payment+rebate", func(p *domain.ListTransactionsParams) { p.TypeCodes = []string{"payment", "rebate"} }},
			{"rebate+credit_memo", func(p *domain.ListTransactionsParams) { p.TypeCodes = []string{"rebate", "credit_memo"} }},
		}},
		{"method", []planValue[domain.ListTransactionsParams]{
			{"check", func(p *domain.ListTransactionsParams) { p.MethodCodes = []string{"check"} }},
			{"gift_card", func(p *domain.ListTransactionsParams) { p.MethodCodes = []string{"gift_card"} }},
		}},
		{"adjustment", []planValue[domain.ListTransactionsParams]{
			{"write_off", func(p *domain.ListTransactionsParams) { p.AdjustmentTypeCodes = []string{"write_off"} }},
			{"refund(none)", func(p *domain.ListTransactionsParams) { p.AdjustmentTypeCodes = []string{"refund"} }},
		}},
		{"customer", []planValue[domain.ListTransactionsParams]{
			{"large", func(p *domain.ListTransactionsParams) { p.CustomerIDs = []string{planTxCustomerID(0)} }},
			{"rare", func(p *domain.ListTransactionsParams) {
				p.CustomerIDs = []string{planTxCustomerID(planTxCustomers - 1)}
			}},
			{"rare+tail", func(p *domain.ListTransactionsParams) {
				p.CustomerIDs = []string{planTxCustomerID(planTxCustomers - 1), planTxCustomerID(500)}
			}},
		}},
		{"group", []planValue[domain.ListTransactionsParams]{
			{"big", func(p *domain.ListTransactionsParams) { p.CustomerGroupIDs = []string{"ag_plantx_big"} }},
			{"small", func(p *domain.ListTransactionsParams) { p.CustomerGroupIDs = []string{"ag_plantx_small"} }},
		}},
		{"search", []planValue[domain.ListTransactionsParams]{
			{"one", func(p *domain.ListTransactionsParams) { p.Query = str(planTxRareSearch) }},
			{"every", func(p *domain.ListTransactionsParams) { p.Query = str(planTxDenseSearch) }},
		}},
		{"funds", []planValue[domain.ListTransactionsParams]{
			{"last30d", func(p *domain.ListTransactionsParams) { p.StartDate = at(recent.Add(-30 * 24 * time.Hour)) }},
			{"old30d", func(p *domain.ListTransactionsParams) {
				p.StartDate, p.EndDate = at(old), at(old.Add(30*24*time.Hour))
			}},
		}},
	}
}

// transactionPlanCases is every filter pair on the first page and on a page deep in the account in
// both directions.
func transactionPlanCases() []planCase[domain.ListTransactionsParams] {
	mid := planTxCreatedAt(planTxRows / 2)
	cursor := func(dir pagination.Direction) func(*domain.ListTransactionsParams) {
		return func(p *domain.ListTransactionsParams) { p.Cursor = cursorAt(mid, dir) }
	}
	return planCases(
		domain.ListTransactionsParams{AccountID: planTxAccount, Limit: 25},
		transactionPlanDims(),
		[]planValue[domain.ListTransactionsParams]{
			{"first", func(*domain.ListTransactionsParams) {}},
			{"deep-next", cursor(pagination.DirectionForward)},
			{"deep-prev", cursor(pagination.DirectionBackward)},
		},
	)
}

func cursorAt(at time.Time, dir pagination.Direction) *string {
	c := pagination.EncodeStringCursor(pagination.StringCursor{OccurredAt: at, ID: "tx_plantx_~", Direction: dir})
	return &c
}

// transactionRangeFloor is how many transactions a request's unordered filter matches, or 0 when it
// has none. A number search (FULLTEXT, answered in relevance order), a funds-received range (a column
// the list does not sort by), and a multi-valued filter (no key yields several values in list order)
// cannot stop at a page; the best any plan can do is read only the rows the filter matches, so that,
// not a page, is the bar such a request is held to. Beside a multi-valued filter, that is the
// narrowest active filter's matches.
func transactionRangeFloor(t *testing.T, db *sql.DB, p domain.ListTransactionsParams) float64 {
	t.Helper()
	where, args := []string{"account_id = ?"}, []any{p.AccountID}
	switch {
	case dbpkg.AllWordsPrefixQuery(p.Query) != "":
		where, args = append(where, "MATCH(number) AGAINST(? IN BOOLEAN MODE)"), append(args, dbpkg.AllWordsPrefixQuery(p.Query))
	case p.StartDate != nil || p.EndDate != nil:
		if p.Status != nil {
			where, args = append(where, "is_fully_allocated = ?"), append(args, *p.Status == "allocated")
		}
		if p.StartDate != nil {
			where, args = append(where, "funds_received_at >= ?"), append(args, *p.StartDate)
		}
		if p.EndDate != nil {
			where, args = append(where, "funds_received_at <= ?"), append(args, *p.EndDate)
		}
	default:
		return transactionMultiValueFloor(t, db, p)
	}
	var n float64
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM `transaction` WHERE "+strings.Join(where, " AND "), args...).Scan(&n))
	return n
}

// TestTransactionList_ReadsAboutAPage holds every filter combination ListTransactions accepts to
// reading about a page of transactions (listPlanSuite).
func TestTransactionList_ReadsAboutAPage(t *testing.T) {
	ensureTransactionCorpus(t)
	listPlanSuite[domain.ListTransactionsParams]{
		table: "transaction", scopeColumn: "account_id",
		from: "FROM `transaction` t", alias: "t",
		cases: transactionPlanCases(),
		limit: func(p domain.ListTransactionsParams) int32 { return p.Limit },
		list: func(ctx context.Context, q *sqlc.Queries, p domain.ListTransactionsParams) error {
			if _, apiErr := NewTransactionRepo(q).List(ctx, p); apiErr != nil {
				return apiErr
			}
			return nil
		},
		floor: transactionRangeFloor,
	}.run(t)
}

func transactionMultiValueFloor(t *testing.T, db *sql.DB, p domain.ListTransactionsParams) float64 {
	t.Helper()
	customerIDs := p.CustomerIDs
	if len(p.CustomerGroupIDs) > 0 {
		rows, err := db.Query("SELECT counterparty_account_id FROM account_relation WHERE owner_account_id = ? AND account_relation_role_code = 'customer' AND account_group_id IN ("+
			placeholders(len(p.CustomerGroupIDs))+")", append([]any{p.AccountID}, stringArgs(p.CustomerGroupIDs)...)...)
		require.NoError(t, err)
		var group []string
		for rows.Next() {
			var id string
			require.NoError(t, rows.Scan(&id))
			group = append(group, id)
		}
		require.NoError(t, rows.Close())
		if len(customerIDs) > 0 {
			group = intersectStrings(customerIDs, group)
		}
		customerIDs = group
	}
	filters := []struct {
		column string
		values []string
	}{
		{"transaction_type_code", p.TypeCodes},
		{"transaction_method_code", p.MethodCodes},
		{"adjustment_type_code", p.AdjustmentTypeCodes},
		{"customer_account_id", customerIDs},
	}
	if p.Status != nil && (*p.Status == "allocated" || *p.Status == "unallocated") {
		allocated := "0"
		if *p.Status == "allocated" {
			allocated = "1"
		}
		filters = append(filters, struct {
			column string
			values []string
		}{"is_fully_allocated", []string{allocated}})
	}
	multiValued, floor := false, -1.0
	for _, f := range filters {
		if len(f.values) == 0 {
			continue
		}
		multiValued = multiValued || len(f.values) > 1
		args := append([]any{p.AccountID}, stringArgs(f.values)...)
		var n float64
		require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM `transaction` WHERE account_id = ? AND "+f.column+
			" IN ("+placeholders(len(f.values))+")", args...).Scan(&n))
		if floor < 0 || n < floor {
			floor = n
		}
	}
	if !multiValued {
		return 0
	}
	return floor
}

// TestTransactionList_ResolvesTheResponsibleUser pins the responsible user a page shows to the account
// user the column names, whether it holds an account_user id or (on legacy rows) a user id.
func TestTransactionList_ResolvesTheResponsibleUser(t *testing.T) {
	ensureTransactionCorpus(t)
	db := planDB(t)
	ctx := context.Background()

	legacy, dangling := planTxID(planTxRows-2), planTxID(planTxRows-3)
	var legacyWas, danglingWas sql.NullString
	require.NoError(t, db.QueryRow("SELECT responsible_user_id FROM `transaction` WHERE id = ?", legacy).Scan(&legacyWas))
	require.NoError(t, db.QueryRow("SELECT responsible_user_id FROM `transaction` WHERE id = ?", dangling).Scan(&danglingWas))
	_, err := db.Exec("UPDATE `transaction` SET responsible_user_id = 'us_plantx_0000000000000007' WHERE id = ?", legacy)
	require.NoError(t, err)
	_, err = db.Exec("UPDATE `transaction` SET responsible_user_id = 'us_plantx_unknown' WHERE id = ?", dangling)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = db.Exec("UPDATE `transaction` SET responsible_user_id = ? WHERE id = ?", legacyWas, legacy)
		_, _ = db.Exec("UPDATE `transaction` SET responsible_user_id = ? WHERE id = ?", danglingWas, dangling)
	})

	repo := NewTransactionRepo(sqlc.New(db))
	str := func(s sql.NullString) string { return s.String }
	deref := func(s *string) string {
		if s == nil {
			return ""
		}
		return *s
	}
	for _, id := range []string{legacy, dangling, planTxID(planTxRows - 1), planTxID(planTxRows - 15)} {
		tx, apiErr := repo.Get(ctx, planTxAccount, id)
		require.Nil(t, apiErr)
		var want, status sql.NullString
		require.NoError(t, db.QueryRow("SELECT COALESCE(au.id, t.responsible_user_id), au.status_code FROM `transaction` t"+
			" LEFT JOIN account_user au ON au.account_id = t.account_id AND (au.id = t.responsible_user_id OR au.user_id = t.responsible_user_id)"+
			" WHERE t.id = ?", id).Scan(&want, &status))
		require.Equal(t, str(want), deref(tx.ResponsibleUserID), id)
		require.Equal(t, str(status), deref(tx.ResponsibleUserStatusCode), id)
	}
}
