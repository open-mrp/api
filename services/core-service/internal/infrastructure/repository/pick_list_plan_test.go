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
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/pagination"
)

func pickPlanDims() []planDim[domain.ListPicksParams] {
	str := func(s string) *string { return &s }
	day := func(t time.Time) *string { return str(t.Format(time.DateOnly)) }
	recent := planFulCreatedAt(planFulOrders - 1)
	old := planFulCreatedAt(planFulOrders / 4)
	var tail []string
	for c := 100; c < 160; c++ {
		tail = append(tail, planFulCustomerID(c))
	}
	return []planDim[domain.ListPicksParams]{
		{"sort", []planValue[domain.ListPicksParams]{
			{"created", func(p *domain.ListPicksParams) { p.Sort = constants.PickSortCreatedAt }},
		}},
		{"status", []planValue[domain.ListPicksParams]{
			{"open", func(p *domain.ListPicksParams) { p.Status = str("open") }},
			{"closed", func(p *domain.ListPicksParams) { p.Status = str("closed") }},
		}},
		{"customer", []planValue[domain.ListPicksParams]{
			{"large", func(p *domain.ListPicksParams) { p.CustomerIDs = []string{planFulCustomerID(0)} }},
			{"rare", func(p *domain.ListPicksParams) { p.CustomerIDs = []string{planFulCustomerID(planFulRareCustomer)} }},
			{"three", func(p *domain.ListPicksParams) {
				p.CustomerIDs = []string{planFulCustomerID(1), planFulCustomerID(300), planFulCustomerID(planFulRareCustomer)}
			}},
			{"sixty", func(p *domain.ListPicksParams) { p.CustomerIDs = tail }},
		}},
		{"group", []planValue[domain.ListPicksParams]{
			{"big", func(p *domain.ListPicksParams) { p.CustomerGroupIDs = []string{"ag_planful_big"} }},
			{"once", func(p *domain.ListPicksParams) { p.CustomerGroupIDs = []string{"ag_planful_once"} }},
			{"small", func(p *domain.ListPicksParams) { p.CustomerGroupIDs = []string{"ag_planful_small"} }},
		}},
		{"product_line", []planValue[domain.ListPicksParams]{
			{"dense", func(p *domain.ListPicksParams) { p.ProductLineIDs = []string{planFulLineDense} }},
			{"rare", func(p *domain.ListPicksParams) { p.ProductLineIDs = []string{planFulLineRare} }},
			{"zero", func(p *domain.ListPicksParams) { p.ProductLineIDs = []string{planFulLineZero} }},
		}},
		{"search", []planValue[domain.ListPicksParams]{
			{"prefix", func(p *domain.ListPicksParams) { p.Query = str("2") }},
			{"prefix(none)", func(p *domain.ListPicksParams) { p.Query = str("9") }},
			{"phrase(one)", func(p *domain.ListPicksParams) { p.Query = str("12345") }},
			{"phrase(every)", func(p *domain.ListPicksParams) { p.Query = str("Customer") }},
		}},
		{"created", []planValue[domain.ListPicksParams]{
			{"last30d", func(p *domain.ListPicksParams) { p.StartDate = day(recent.Add(-30 * 24 * time.Hour)) }},
			{"old30d", func(p *domain.ListPicksParams) { p.StartDate, p.EndDate = day(old), day(old.Add(30*24*time.Hour)) }},
		}},
	}
}

// pickPlanCases is every filter pair on the first page and on a page deep in the account in both
// directions. The cursor sits mid-corpus under either sort: ship-by dates trail creation by weeks.
func pickPlanCases() []planCase[domain.ListPicksParams] {
	mid := planFulCreatedAt(planFulOrders / 2)
	cursor := func(dir pagination.Direction) func(*domain.ListPicksParams) {
		return func(p *domain.ListPicksParams) {
			c := pagination.EncodeStringCursor(pagination.StringCursor{OccurredAt: mid, ID: "pk_planful_~", Direction: dir})
			p.Cursor = &c
		}
	}
	return planCases(
		domain.ListPicksParams{AccountID: planFulAccount, Limit: 25},
		qualifiedPlanDims(pickPlanDims()),
		[]planValue[domain.ListPicksParams]{
			{"first", func(*domain.ListPicksParams) {}},
			{"deep-next", cursor(pagination.DirectionForward)},
			{"deep-prev", cursor(pagination.DirectionBackward)},
		},
	)
}

// pickBuyerSet is the customers a request may list, or nil for any: what pickRepoImpl.buyerFilter
// resolves, read here independently of it.
func pickBuyerSet(t *testing.T, db *sql.DB, p domain.ListPicksParams) []string {
	t.Helper()
	if len(p.CustomerGroupIDs) == 0 {
		return p.CustomerIDs
	}
	rows, err := db.Query(`SELECT DISTINCT counterparty_account_id FROM account_relation
		WHERE owner_account_id = ? AND account_group_id IN (`+placeholders(len(p.CustomerGroupIDs))+`)`,
		append([]any{p.AccountID}, stringArgs(p.CustomerGroupIDs)...)...)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	wanted := map[string]bool{}
	for _, id := range p.CustomerIDs {
		wanted[id] = true
	}
	set := []string{}
	for rows.Next() {
		var id string
		require.NoError(t, rows.Scan(&id))
		if len(p.CustomerIDs) == 0 || wanted[id] {
			set = append(set, id)
		}
	}
	require.NoError(t, rows.Err())
	return set
}

// pickUnorderedFloor is how many picks the request's narrowest unordered filter matches, or 0 when it
// has none. No key yields these in list order: a set of customers (ranges of the buyer key), product
// lines (a child table), a number search (a FULLTEXT match, or a prefix range on the number key), and
// a creation window under the ship-by sort. The best any plan can do is read one filter's matches.
func pickUnorderedFloor(t *testing.T, db *sql.DB, p domain.ListPicksParams) float64 {
	t.Helper()
	var floors []float64
	count := func(where string, args ...any) {
		var n float64
		require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM pick p WHERE p.account_id = ? AND "+where,
			append([]any{p.AccountID}, args...)...).Scan(&n))
		floors = append(floors, n)
	}
	if buyers := pickBuyerSet(t, db, p); len(buyers) > 1 {
		count("p.buyer_account_id IN ("+placeholders(len(buyers))+")", stringArgs(buyers)...)
	}
	if len(p.ProductLineIDs) > 0 {
		count("p.id IN ("+pickProductLineMatches(len(p.ProductLineIDs))+")", stringArgs(p.ProductLineIDs)...)
	}
	switch search := newPickSearch(p.Query); {
	case search.NumberPrefix != "":
		count("p.number LIKE ?", search.NumberPrefix)
	case search.Phrase != "":
		count("p.id IN (SELECT id FROM ("+pickPhraseMatches+") matched)",
			p.AccountID, search.Phrase, p.AccountID, search.Phrase, p.AccountID, search.Phrase, p.AccountID, search.Phrase)
	}
	if p.Sort != constants.PickSortCreatedAt && (p.StartDate != nil || p.EndDate != nil) {
		where, args := []string{"TRUE"}, []any{}
		if start := parseDateFilter(p.StartDate); start.Valid {
			where, args = append(where, "p.created_at >= ?"), append(args, start.Time)
		}
		if end := parseEndDateFilter(p.EndDate); end.Valid {
			where, args = append(where, "p.created_at <= ?"), append(args, end.Time)
		}
		count(strings.Join(where, " AND "), args...)
	}
	if len(floors) == 0 {
		return 0
	}
	floor := floors[0]
	for _, f := range floors[1:] {
		floor = min(floor, f)
	}
	// A floor of zero means no unordered filter; a filter that matches nothing reads nothing.
	return max(floor, 1)
}

// TestPickList_ReadsAboutAPage holds every filter combination ListPicks accepts to reading about a
// page of picks (listPlanSuite).
func TestPickList_ReadsAboutAPage(t *testing.T) {
	ensureFulfillmentCorpus(t)
	db := planDB(t)
	listPlanSuite[domain.ListPicksParams]{
		table: "pick", scopeColumn: "account_id",
		from: "FROM pick p", alias: "p",
		statement: func(query string) bool {
			return strings.HasPrefix(query, "SELECT STRAIGHT_JOIN p.id") || strings.HasPrefix(query, "SELECT id FROM (")
		},
		cases: pickPlanCases(),
		limit: func(p domain.ListPicksParams) int32 { return p.Limit },
		list: func(ctx context.Context, q *sqlc.Queries, p domain.ListPicksParams) error {
			if _, apiErr := NewPickRepo(q).List(ctx, p); apiErr != nil {
				return apiErr
			}
			return nil
		},
		floor: pickUnorderedFloor,
		empty: func(p domain.ListPicksParams) bool {
			buyers := pickBuyerSet(t, db, p)
			return buyers != nil && len(buyers) == 0
		},
	}.run(t)
}
