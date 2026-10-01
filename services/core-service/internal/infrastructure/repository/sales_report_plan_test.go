//go:build plans

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
)

// The sales corpus is one merchant shaped like the largest production tenant's last four years of
// invoicing: tens of thousands of invoices of a few lines each, one buyer taking a tenth of the lines
// and a long tail of hundreds more, one item a sixth, one product line over a quarter, sales reps on
// two fifths of recent lines and almost none before, and a handful of discounted orders. Each filter
// has a dense value and a rare one (a buyer, item, product line and rep with three invoices each), and
// a customer has two child accounts and a group, so every way a report resolves its filters is
// exercised. Its rollups are built the way the refresher's sweep builds them.
const (
	planAnaAccount  = "ac_planana"
	planAnaInvoices = 28_000
	planAnaBuyers   = 600
	planAnaItems    = 500
	planAnaLines    = 26
	planAnaReps     = 10
	planAnaSpanDays = 4 * 365

	// planAnaCorpusVersion is bumped whenever the corpus's shape changes, so a stale one is rebuilt.
	planAnaCorpusVersion = "Plan Analytics Merchant v1"

	planAnaRareBuyer = planAnaBuyers - 1
	planAnaRareItem  = planAnaItems - 1
	planAnaRareLine  = planAnaLines - 1
	planAnaRareRep   = planAnaReps - 1
)

var (
	planAnaOrigin = time.Date(2022, 10, 1, 0, 0, 0, 0, time.UTC)
	// planAnaRepsFrom is when the merchant started assigning sales reps to most orders.
	planAnaRepsFrom = time.Date(2024, 10, 1, 0, 0, 0, 0, time.UTC)

	planAnaRareBuyerInvoices = map[int]bool{1_000: true, 15_000: true, 27_000: true}
	planAnaRareItemInvoices  = map[int]bool{5_000: true, 18_000: true, 27_900: true}
	planAnaRareRepInvoices   = map[int]bool{20_000: true, 26_000: true, 27_500: true}
)

func planAnaBuyerID(b int) string { return fmt.Sprintf("%s_b%05d", planAnaAccount, b) }
func planAnaItemID(i int) string  { return fmt.Sprintf("it_planana_%05d", i) }
func planAnaLineID(l int) string  { return fmt.Sprintf("pl_planana_%02d", l) }
func planAnaRepID(r int) string   { return fmt.Sprintf("acus_planana_r%02d", r) }

func planAnaInvoicedAt(i int) time.Time {
	return planAnaOrigin.Add(time.Duration(i) * (planAnaSpanDays * 24 * time.Hour / planAnaInvoices))
}

// planHash is a deterministic stand-in for a random draw, so the corpus is the same on every machine.
func planHash(i, salt int) int {
	x := uint64(i)*0x9E3779B97F4A7C15 + uint64(salt)*0xBF58476D1CE4E5B9
	x ^= x >> 31
	x *= 0x94D049BB133111EB
	x ^= x >> 29
	return int(x % (1 << 30))
}

// planAnaBuyer gives a tenth of the invoices to one buyer, 15% to the next four, a quarter to the
// next 45 and the rest to a long tail. The customer's two child accounts take a share of its tail.
func planAnaBuyer(i int) int {
	switch {
	case planAnaRareBuyerInvoices[i]:
		return planAnaRareBuyer
	case i%500 == 7:
		return planAnaBuyers - 10
	case i%500 == 8:
		return planAnaBuyers - 9
	}
	r := planHash(i, 1) % 1000
	switch {
	case r < 100:
		return 0
	case r < 250:
		return 1 + planHash(i, 11)%4
	case r < 500:
		return 5 + planHash(i, 12)%45
	default:
		return 50 + planHash(i, 13)%(planAnaBuyers-60)
	}
}

func planAnaRep(i int) any {
	if planAnaRareRepInvoices[i] {
		return planAnaRepID(planAnaRareRep)
	}
	share := 3
	if !planAnaInvoicedAt(i).Before(planAnaRepsFrom) {
		share = 40
	}
	if planHash(i, 2)%100 >= share {
		return nil
	}
	r := planHash(i, 21) % 100
	switch {
	case r < 50:
		return planAnaRepID(0)
	case r < 75:
		return planAnaRepID(1)
	default:
		return planAnaRepID(2 + planHash(i, 22)%7)
	}
}

func planAnaItem(i, line int) int {
	if planAnaRareItemInvoices[i] && line == 0 {
		return planAnaRareItem
	}
	r := planHash(i*8+line, 4) % 1000
	switch {
	case r < 150:
		return 0
	case r < 400:
		return 1 + planHash(i*8+line, 41)%29
	default:
		return 30 + planHash(i*8+line, 42)%(planAnaItems-31)
	}
}

// planAnaItemLine puts the busiest items on the three largest product lines and the rare item alone
// on the rare line.
func planAnaItemLine(item int) int {
	switch {
	case item == planAnaRareItem:
		return planAnaRareLine
	case item < 30:
		return item % 3
	default:
		return 3 + item%(planAnaLines-4)
	}
}

var planAnaCorpusOnce sync.Once

func ensureAnalyticsCorpus(t *testing.T) {
	t.Helper()
	db := planDB(t)
	planAnaCorpusOnce.Do(func() {
		var have int
		require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM sales_line_fact WHERE account_id = ?", planAnaAccount).Scan(&have))
		var version string
		_ = db.QueryRow("SELECT name FROM account WHERE id = ?", planAnaAccount).Scan(&version)
		if have > 0 && version == planAnaCorpusVersion {
			return
		}
		t.Logf("seeding the sales plan corpus (%d invoices) and its rollups; it is kept for later runs", planAnaInvoices)

		exec := func(query string, args ...any) {
			_, err := db.Exec(query, args...)
			require.NoError(t, err)
		}
		exec("DELETE FROM sales_line_fact WHERE account_id = ?", planAnaAccount)
		exec("DELETE FROM sales_fact_rollup WHERE account_id = ?", planAnaAccount)
		exec("DELETE FROM account_relation WHERE owner_account_id = ?", planAnaAccount)
		exec("DELETE FROM account_group WHERE owner_account_id = ?", planAnaAccount)
		exec("DELETE FROM account WHERE id LIKE 'ac\\_planana\\_%'")
		exec(`INSERT INTO account (id, name, account_type_code, onboarding_status_code) VALUES (?, 'pending', 'company', 'active')
		      ON DUPLICATE KEY UPDATE name = 'pending'`, planAnaAccount)
		for _, g := range []string{"big", "small"} {
			exec(`INSERT INTO account_group (id, owner_account_id, name, commission_status_code, freight_status_code, account_group_type_code)
			      VALUES (?, ?, ?, 'commission_applied', 'billed_freight', 'type_group')`, "ag_planana_"+g, planAnaAccount, "Plan "+g)
		}

		var accVals, relVals []string
		var accArgs, relArgs []any
		for b := range planAnaBuyers {
			var group, parent any
			switch {
			case b < 200:
				group = "ag_planana_big"
			case b == planAnaRareBuyer:
				group = "ag_planana_small"
			}
			if b == planAnaBuyers-10 || b == planAnaBuyers-9 {
				parent = "ar_planana_00000"
			}
			accVals = append(accVals, "(?, ?, 'company', 'unclaimed')")
			accArgs = append(accArgs, planAnaBuyerID(b), fmt.Sprintf("Plan Buyer %04d", b))
			relVals = append(relVals, "(?, ?, ?, 'customer', ?, 'normal', ?, ?)")
			relArgs = append(relArgs, fmt.Sprintf("ar_planana_%05d", b), planAnaAccount, planAnaBuyerID(b), fmt.Sprintf("B%04d", b), group, parent)
		}
		exec(`INSERT INTO account (id, name, account_type_code, onboarding_status_code) VALUES `+strings.Join(accVals, ","), accArgs...)
		exec(`INSERT INTO account_relation (id, owner_account_id, counterparty_account_id, account_relation_role_code, external_number, priority_code, account_group_id, parent_account_relation_id)
		      VALUES `+strings.Join(relVals, ","), relArgs...)

		const batch = 500
		for start := 0; start < planAnaInvoices; start += batch {
			var vals []string
			var args []any
			for i := start; i < start+batch; i++ {
				at := planAnaInvoicedAt(i)
				var discount any
				if i%250 == 0 {
					discount = fmt.Sprintf("od_planana_%d", (i/250)%3)
				}
				for line := range 1 + planHash(i, 3)%6 {
					item := planAnaItem(i, line)
					qty := 1 + planHash(i*8+line, 5)%50
					cents := 100 + planHash(i*8+line, 6)%10_000
					total := fmt.Sprintf("%d.%04d", qty*cents/100, (qty*cents%100)*100+planHash(i, 7)%100)
					cost := fmt.Sprintf("%d.%04d", qty*cents*6/1000, planHash(i, 8)%10_000)
					vals = append(vals, "(?, ?, ?, ?, ?, 'sales_order', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1)")
					args = append(args, planAnaAccount, at, fmt.Sprintf("ivln_planana_%07d_%d", i, line), fmt.Sprintf("iv_planana_%07d", i),
						fmt.Sprintf("or_planana_%07d", i), planAnaBuyerID(planAnaBuyer(i)), planAnaRep(i), discount,
						fmt.Sprintf("pd_planana_%05d", item), planAnaItemID(item), planAnaLineID(planAnaItemLine(item)),
						qty, total, cost, at, at.Add(-72*time.Hour))
				}
			}
			exec(`INSERT INTO sales_line_fact (account_id, invoiced_at, invoice_line_id, invoice_id, sales_order_id, sales_order_type_code,
			      buyer_account_id, sales_rep_id, order_discount_id, product_id, item_id, product_line_id, quantity_base, total_invoiced, total_cost,
			      refreshed_at, ordered_at, is_priced) VALUES `+strings.Join(vals, ","), args...)
		}

		repo := NewSalesFactRepo(sqlc.New(db))
		ctx := context.Background()
		end := planAnaOrigin.AddDate(0, 0, planAnaSpanDays+1)
		for day := planAnaOrigin; day.Before(end); day = day.AddDate(0, 0, 1) {
			require.Nil(t, repo.RebuildRollupDay(ctx, domain.SalesRollupDay{AccountID: planAnaAccount, Day: day}))
			if day.AddDate(0, 0, 1).Day() == 1 || !day.AddDate(0, 0, 1).Before(end) {
				require.Nil(t, repo.RebuildRollupMonth(ctx, planAnaAccount, day))
			}
		}
		exec("UPDATE account SET name = ? WHERE id = ?", planAnaCorpusVersion, planAnaAccount)
		exec("ANALYZE TABLE sales_line_fact, sales_fact_rollup, account_relation")
	})
}

var planAnaInvoicesOnce sync.Once

// ensureAnalyticsInvoices adds the corpus's invoices and their sales orders, which the invoice page joins
// its lines back to.
func ensureAnalyticsInvoices(t *testing.T) {
	t.Helper()
	ensureAnalyticsCorpus(t)
	db := planDB(t)
	planAnaInvoicesOnce.Do(func() {
		var have int
		require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM invoice WHERE account_id = ?", planAnaAccount).Scan(&have))
		if have == planAnaInvoices {
			return
		}
		exec := func(query string, args ...any) {
			_, err := db.Exec(query, args...)
			require.NoError(t, err)
		}
		exec("DELETE FROM invoice WHERE account_id = ?", planAnaAccount)
		exec("DELETE FROM sales_order WHERE owner_account_id = ?", planAnaAccount)
		const batch = 1_000
		for start := 0; start < planAnaInvoices; start += batch {
			var invVals, soVals []string
			var invArgs, soArgs []any
			for i := start; i < start+batch; i++ {
				at := planAnaInvoicedAt(i)
				invVals = append(invVals, "(?, ?, ?, 'ad_planana', ?, ?, ?)")
				invArgs = append(invArgs, fmt.Sprintf("iv_planana_%07d", i), fmt.Sprintf("INV-%06d", i), fmt.Sprintf("or_planana_%07d", i), planAnaAccount, at, at)
				soVals = append(soVals, "(?, 'ad_planana', 'ad_planana', ?, 'normal', 'fulfilled', 'sales_order', ?, ?, ?, ?, ?)")
				soArgs = append(soArgs, fmt.Sprintf("or_planana_%07d", i), fmt.Sprintf("SO-%06d", i), planAnaBuyerID(planAnaBuyer(i)),
					planAnaAccount, planAnaAccount, at.Add(-72*time.Hour), at)
			}
			exec(`INSERT INTO invoice (id, number, sales_order_id, billing_address_id, account_id, created_at, updated_at) VALUES `+strings.Join(invVals, ","), invArgs...)
			exec(`INSERT INTO sales_order (id, billing_address_id, shipping_address_id, number, priority_code, sales_order_status_code, sales_order_type_code,
			      buyer_account_id, seller_account_id, owner_account_id, created_at, updated_at) VALUES `+strings.Join(soVals, ","), soArgs...)
		}
		exec("ANALYZE TABLE invoice, sales_order")
	})
}

// salesPlanRequest is one sales report request: its filters, the breakdown's group (empty for a
// summary), the summary's time zone, and whether the rollups have been built yet.
type salesPlanRequest struct {
	domain.SalesReportFilter
	groupBy constants.SalesBreakdownGroupBy
	tz      int32
	cold    bool
	// cursor is the invoice page's.
	cursor *string
}

func (p salesPlanRequest) repo(q *sqlc.Queries) *salesReportRepoImpl {
	mode := rollupAuto
	if p.cold {
		mode = rollupNever
	}
	return &salesReportRepoImpl{queries: q, mode: mode}
}

func (p salesPlanRequest) breakdown(ctx context.Context, q *sqlc.Queries) (*domain.SalesBreakdown, error) {
	b, apiErr := p.repo(q).GetBreakdown(ctx, domain.AnalyzeSalesBreakdownParams{SalesReportFilter: p.SalesReportFilter, GroupBy: p.groupBy, Limit: 25}, true)
	if apiErr != nil {
		return nil, apiErr
	}
	return b, nil
}

func (p salesPlanRequest) summary(ctx context.Context, q *sqlc.Queries) (*domain.SalesSummary, error) {
	s, apiErr := p.repo(q).GetSummary(ctx, domain.AnalyzeSalesSummaryParams{SalesReportFilter: p.SalesReportFilter, TZOffsetMinutes: p.tz}, true)
	if apiErr != nil {
		return nil, apiErr
	}
	return s, nil
}

func salesPlanFilterDims() []planDim[salesPlanRequest] {
	type v = planValue[salesPlanRequest]
	return []planDim[salesPlanRequest]{
		{"customer", []v{
			{"customer=large", func(p *salesPlanRequest) { p.CustomerIDs = []string{planAnaBuyerID(0)} }},
			{"customer=rare", func(p *salesPlanRequest) { p.CustomerIDs = []string{planAnaBuyerID(planAnaRareBuyer)} }},
		}},
		{"group", []v{
			{"group=big", func(p *salesPlanRequest) { p.CustomerGroupIDs = []string{"ag_planana_big"} }},
			{"group=small", func(p *salesPlanRequest) { p.CustomerGroupIDs = []string{"ag_planana_small"} }},
		}},
		{"line", []v{
			{"line=large", func(p *salesPlanRequest) { p.ProductLineIDs = []string{planAnaLineID(0)} }},
			{"line=rare", func(p *salesPlanRequest) { p.ProductLineIDs = []string{planAnaLineID(planAnaRareLine)} }},
			{"line=two", func(p *salesPlanRequest) { p.ProductLineIDs = []string{planAnaLineID(1), planAnaLineID(4)} }},
		}},
		{"item", []v{
			{"item=large", func(p *salesPlanRequest) { p.ItemIDs = []string{planAnaItemID(0)} }},
			{"item=rare", func(p *salesPlanRequest) { p.ItemIDs = []string{planAnaItemID(planAnaRareItem)} }},
		}},
		{"rep", []v{
			{"rep=large", func(p *salesPlanRequest) { p.SalesRepIDs = []string{planAnaRepID(0)} }},
			{"rep=rare", func(p *salesPlanRequest) { p.SalesRepIDs = []string{planAnaRepID(planAnaRareRep)} }},
		}},
	}
}

// salesPlanWindows are the periods a report asks for, ending at the corpus's last day in a US time
// zone: the last week, the last year against the year before, a month three years back, everything,
// and a window whose edges fall mid-hour.
func salesPlanWindows() []planValue[salesPlanRequest] {
	at := func(s string) time.Time {
		v, err := time.Parse(time.RFC3339Nano, s)
		if err != nil {
			panic(err)
		}
		return v
	}
	window := func(start, end string) func(*salesPlanRequest) {
		return func(p *salesPlanRequest) { p.StartsAt, p.EndsAt = at(start), at(end) }
	}
	return []planValue[salesPlanRequest]{
		{"7d", window("2026-09-23T05:00:00Z", "2026-09-30T04:59:59.999Z")},
		{"1y-vs-prior", func(p *salesPlanRequest) {
			p.StartsAt, p.EndsAt = at("2025-10-01T05:00:00Z"), at("2026-10-01T04:59:59.999Z")
			cs, ce := at("2024-10-01T05:00:00Z"), at("2025-10-01T04:59:59.999Z")
			p.ComparisonStartsAt, p.ComparisonEndsAt = &cs, &ce
		}},
		{"old-month", window("2023-03-01T05:00:00Z", "2023-04-01T04:59:59.999Z")},
		{"all-time", window("1900-01-01T00:00:00Z", "2026-10-01T04:59:59.999Z")},
		{"ragged", window("2025-06-15T13:37:12.345Z", "2026-02-03T08:12:59.999Z")},
	}
}

// salesPlanRequests is every filter value alone and every pair, in every window, for each group (one
// empty group for a summary), with the rollups built and before they are.
func salesPlanRequests(groups []constants.SalesBreakdownGroupBy, tz int32) []planCase[salesPlanRequest] {
	var out []planCase[salesPlanRequest]
	for _, cold := range []bool{false, true} {
		for _, g := range groups {
			base := salesPlanRequest{SalesReportFilter: domain.SalesReportFilter{AccountID: planAnaAccount}, groupBy: g, tz: tz, cold: cold}
			prefix := "rollups"
			if cold {
				prefix = "cold"
			}
			if g != "" {
				prefix += "/" + string(g)
			}
			for _, c := range planCases(base, salesPlanFilterDims(), salesPlanWindows()) {
				c.name = prefix + "/" + c.name
				out = append(out, c)
			}
		}
	}
	return out
}

// salesPlanScope is the part of sales_line_fact and sales_fact_rollup a request covers, as the
// report's own filters define it.
type salesPlanScope struct {
	buyers                 []string
	buyersFiltered         bool
	lines, items, reps     []string
	dimension              string
	lineKey                string
	lineKeys, dimensionIDs []string
}

func salesScopeOf(t *testing.T, db *sql.DB, p salesPlanRequest) salesPlanScope {
	t.Helper()
	repo := &salesReportRepoImpl{queries: sqlc.New(db)}
	buyers, filtered, apiErr := repo.resolveBuyers(context.Background(), p.SalesReportFilter)
	require.Nil(t, apiErr)
	s := salesPlanScope{buyers: buyers, buyersFiltered: filtered, lines: p.ProductLineIDs, items: p.ItemIDs, reps: p.SalesRepIDs, dimension: rollupDimTotal}
	if p.groupBy != "" {
		s.dimension = breakdownDimension[p.groupBy]
	}
	switch {
	case filtered && s.dimension == rollupDimBuyer:
		s.dimensionIDs = buyers
	case len(s.items) > 0 && s.dimension == rollupDimItem:
		s.dimensionIDs = s.items
	}
	switch {
	case len(s.lines) == 0:
	case s.dimension == rollupDimProductLine:
		s.dimensionIDs = s.lines
	case len(s.lines) == 1:
		s.lineKey = s.lines[0]
	default:
		s.lineKeys = s.lines
	}
	return s
}

// rollupAnswers is whether the rollups hold this request exactly: they are built, a summary's local
// days are whole UTC hours, and every buyer or item filter is on the dimension the report groups by
// (the rollups keep one dimension per row, so a buyer's sales of one item are only in the lines).
func (s salesPlanScope) rollupAnswers(p salesPlanRequest) bool {
	switch {
	case p.cold, p.groupBy == "" && p.tz%60 != 0:
		return false
	case s.buyersFiltered && s.dimension != rollupDimBuyer:
		return false
	case len(s.items) > 0 && s.dimension != rollupDimItem:
		return false
	}
	return true
}

// facts counts the account's lines in ranges, under whichever single filter matches fewest: the most
// an index can pin, every other filter being read and rejected.
//
// A sales rep or discount breakdown's own column is not a filter here. It drops the lines that belong
// to no group, but the window still covers them, and no key can pin "has a rep" ahead of the window:
// a range on the rep column leaves invoiced_at unusable, so reading by it means every rep's whole
// history. The rollups, which hold only lines with a group, are what make these breakdowns cheap.
func (s salesPlanScope) facts(t *testing.T, db *sql.DB, ranges []rawRange) float64 {
	t.Helper()
	if len(ranges) == 0 {
		return 0
	}
	if s.buyersFiltered && len(s.buyers) == 0 {
		return 0
	}
	var window []string
	var windowArgs []any
	for _, r := range ranges {
		op := "<"
		if r.toInclusive {
			op = "<="
		}
		window = append(window, "(invoiced_at >= ? AND invoiced_at "+op+" ?)")
		windowArgs = append(windowArgs, r.from, r.to)
	}
	base := "SELECT COUNT(*) FROM sales_line_fact WHERE account_id = ? AND sales_order_type_code = 'sales_order' AND (" + strings.Join(window, " OR ") + ")"
	baseArgs := append([]any{planAnaAccount}, windowArgs...)

	type filter struct {
		clause string
		args   []any
	}
	var filters []filter
	in := func(column string, ids []string) {
		if len(ids) > 0 {
			filters = append(filters, filter{column + " IN (" + placeholders(len(ids)) + ")", stringsToAny(ids)})
		}
	}
	if s.buyersFiltered {
		in("buyer_account_id", s.buyers)
	}
	in("product_line_id", s.lines)
	in("item_id", s.items)
	in("sales_rep_id", s.reps)
	if len(filters) == 0 {
		filters = append(filters, filter{"TRUE", nil})
	}
	floor := math.Inf(1)
	for _, f := range filters {
		var n float64
		require.NoError(t, db.QueryRow(base+" AND "+f.clause, append(append([]any{}, baseArgs...), f.args...)...).Scan(&n))
		floor = math.Min(floor, n)
	}
	return floor
}

// rollups counts the scope's rollup rows in ranges, under whichever single filter matches fewest.
func (s salesPlanScope) rollups(t *testing.T, db *sql.DB, ranges []rollupRange) float64 {
	t.Helper()
	if len(ranges) == 0 {
		return 0
	}
	clause, args, _ := rollupPredicate(planAnaAccount, rollupScope{dimension: s.dimension, lineKey: s.lineKey, lineKeys: s.lineKeys}, ranges)
	query := "SELECT COUNT(*) FROM sales_fact_rollup r WHERE " + clause
	candidates := [][]any{nil}
	extra := []string{""}
	if len(s.dimensionIDs) > 0 {
		extra = append(extra, " AND r.dimension_id IN ("+placeholders(len(s.dimensionIDs))+")")
		candidates = append(candidates, stringsToAny(s.dimensionIDs))
	}
	if len(s.reps) > 0 {
		extra = append(extra, " AND r.sales_rep_key IN ("+placeholders(len(s.reps))+")")
		candidates = append(candidates, stringsToAny(s.reps))
	}
	floor := math.Inf(1)
	for i := range extra {
		var n float64
		require.NoError(t, db.QueryRow(query+extra[i], append(append([]any{}, args...), candidates[i]...)...).Scan(&n))
		floor = math.Min(floor, n)
	}
	return floor
}

func salesPlanPeriods(p salesPlanRequest) [][2]time.Time {
	return reportWindows(p.SalesReportFilter)
}

func wholePeriod(w [2]time.Time) []rawRange {
	return []rawRange{{from: w[0], to: w[1], toInclusive: true}}
}

// salesBreakdownFloor is a breakdown's scope. Read from the rollups, it is each period's whole month
// and day buckets, and the lines of the stretches at its edges no bucket covers; a scope of several
// product lines also counts its invoices from the lines, over each whole period. Read from the lines,
// it is the lines of every period, in one statement.
func salesBreakdownFloor(t *testing.T, db *sql.DB, p salesPlanRequest) map[string]float64 {
	s := salesScopeOf(t, db, p)
	floors := map[string]float64{}
	if !s.rollupAnswers(p) {
		var ranges []rawRange
		for _, w := range salesPlanPeriods(p) {
			ranges = append(ranges, wholePeriod(w)...)
		}
		floors["f"] = s.facts(t, db, ranges)
		return floors
	}
	for _, w := range salesPlanPeriods(p) {
		plan := planWindow(w[0], w[1], breakdownGrains)
		floors["f"] += s.facts(t, db, plan.raws)
		if len(s.lineKeys) > 0 {
			floors["f"] += s.facts(t, db, wholePeriod(w))
		}
		floors["r"] += s.rollups(t, db, plan.rollups)
	}
	return floors
}

// salesSummaryFloor is a summary's scope: per period, a totals statement and a daily one. From the
// rollups the totals read whole month, day and hour buckets and the daily read hour buckets, each with
// the sub-hour edges from the lines (and, for several product lines, every line of the period for the
// invoice count); from the lines each reads the period's lines.
func salesSummaryFloor(t *testing.T, db *sql.DB, p salesPlanRequest) map[string]float64 {
	s := salesScopeOf(t, db, p)
	floors := map[string]float64{}
	for _, w := range salesPlanPeriods(p) {
		if !s.rollupAnswers(p) {
			floors["f"] += 2 * s.facts(t, db, wholePeriod(w))
			continue
		}
		for _, grains := range [][]string{totalGrains, {rollupGrainHour}} {
			plan := planWindow(w[0], w[1], grains)
			floors["f"] += s.facts(t, db, plan.raws)
			if len(s.lineKeys) > 0 {
				floors["f"] += s.facts(t, db, wholePeriod(w))
			}
			floors["r"] += s.rollups(t, db, plan.rollups)
		}
	}
	return floors
}

var salesPlanTables = []aggregateTable{
	{table: "sales_line_fact", scopeColumn: "account_id", from: "FROM sales_line_fact f", alias: "f"},
	{table: "sales_fact_rollup", scopeColumn: "account_id", from: "FROM sales_fact_rollup r", alias: "r"},
}

var salesBreakdownGroups = []constants.SalesBreakdownGroupBy{
	constants.SalesBreakdownGroupByCustomer, constants.SalesBreakdownGroupByCustomerGroup, constants.SalesBreakdownGroupByProduct,
	constants.SalesBreakdownGroupByProductLine, constants.SalesBreakdownGroupBySalesRep, constants.SalesBreakdownGroupByDiscount,
}

// TestSalesBreakdown_ReadsItsScope holds every breakdown, filter pair and window to reading no more
// sales lines and rollup rows than its scope (aggregatePlanSuite).
func TestSalesBreakdown_ReadsItsScope(t *testing.T) {
	ensureAnalyticsCorpus(t)
	salesRollupsReady.Store(true)
	aggregatePlanSuite[salesPlanRequest]{
		tables: salesPlanTables,
		cases:  salesPlanRequests(salesBreakdownGroups, 0),
		report: func(ctx context.Context, q *sqlc.Queries, p salesPlanRequest) error {
			_, err := p.breakdown(ctx, q)
			return err
		},
		floor: salesBreakdownFloor,
	}.run(t)
}

// TestSalesSummary_ReadsItsScope holds every summary, filter pair and window to reading no more sales
// lines and rollup rows than its scope, in a whole-hour time zone (which the rollups serve) and a
// half-hour one (which they cannot).
func TestSalesSummary_ReadsItsScope(t *testing.T) {
	ensureAnalyticsCorpus(t)
	salesRollupsReady.Store(true)
	cases := salesPlanRequests([]constants.SalesBreakdownGroupBy{""}, -300)
	for _, c := range salesPlanRequests([]constants.SalesBreakdownGroupBy{""}, -330) {
		if !c.params.cold {
			c.name = "half-hour-tz/" + c.name
			cases = append(cases, c)
		}
	}
	aggregatePlanSuite[salesPlanRequest]{
		tables: salesPlanTables,
		cases:  cases,
		report: func(ctx context.Context, q *sqlc.Queries, p salesPlanRequest) error {
			_, err := p.summary(ctx, q)
			return err
		},
		floor: salesSummaryFloor,
	}.run(t)
}

// unsignedCursors is b with its page cursors cut to their payload: the signature is keyed per process,
// and other tests in the package set a different key.
func unsignedCursors(b *domain.SalesBreakdown) *domain.SalesBreakdown {
	out := *b
	for _, c := range []**string{&out.PageInfo.NextCursor, &out.PageInfo.PrevCursor} {
		if *c != nil {
			payload, _, _ := strings.Cut(**c, ".")
			*c = &payload
		}
	}
	return &out
}

// TestSalesReports_ResultsUnchanged pins what every plan-tested sales report returns on the corpus, so
// a change made for its plan cannot change its answer: each request's result, read from the rollups
// and from the lines alone, must hash to the digest recorded before the change.
func TestSalesReports_ResultsUnchanged(t *testing.T) {
	ensureAnalyticsCorpus(t)
	salesRollupsReady.Store(true)
	db := planDB(t)
	q := sqlc.New(db)
	ctx := context.Background()

	digest := func(v any) string { return planDigest(t, v) }
	got := map[string]string{}
	// A request read from the rollups and the same one read cold share a key.
	record := func(report, name, d string) {
		key := report + "/" + strings.TrimPrefix(strings.TrimPrefix(name, "rollups/"), "cold/")
		if prev, ok := got[key]; ok {
			require.Equal(t, prev, d, "%s: the rollups and the lines disagree", key)
			return
		}
		got[key] = d
	}
	for _, c := range salesPlanRequests(salesBreakdownGroups, 0) {
		b, err := c.params.breakdown(ctx, q)
		require.NoError(t, err)
		record("breakdown", c.name, digest(unsignedCursors(b)))
	}
	for _, c := range salesPlanRequests([]constants.SalesBreakdownGroupBy{""}, -300) {
		s, err := c.params.summary(ctx, q)
		require.NoError(t, err)
		record("summary", c.name, digest(s))
	}
	checkPlanResults(t, "sales_reports.json", got)
}
