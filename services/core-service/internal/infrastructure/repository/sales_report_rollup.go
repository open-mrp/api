package repository

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
)

// salesRollupsReady latches once the rollup backfill has completed; a completed pass is never undone.
var salesRollupsReady atomic.Bool

// rollupMode selects how a report reads.
type rollupMode int

const (
	// rollupAuto reads rollups once the backfill has completed and the filters allow it, otherwise facts.
	rollupAuto rollupMode = iota
	// rollupNever always reads facts; tests compare it against rollupAuto.
	rollupNever
)

func (r *salesReportRepoImpl) rollupsReady(ctx context.Context) bool {
	if r.mode == rollupNever {
		return false
	}
	if salesRollupsReady.Load() {
		return true
	}
	sync, apiErr := NewSalesFactRepo(r.queries).GetRollupSync(ctx)
	if apiErr != nil || sync.LastCompletedAt == nil {
		return false
	}
	salesRollupsReady.Store(true)
	return true
}

// rollupScope is the slice of sales_fact_rollup a report reads.
type rollupScope struct {
	dimension string
	// lineKey is '' for every product line, or the one product line the report is filtered to.
	lineKey string
	// lineKeys, when set, are the several product lines the report is filtered to. Their rows' money,
	// quantities and lines add up, since a line has one product line; their invoice counts do not (an
	// invoice can span two of them), so those are counted from the facts instead (see invoiceCountRows).
	lineKeys []string
	// dimensionIDs restricts the groups read; nil reads all.
	dimensionIDs []string
	salesRepIDs  []string
}

// breakdownDimension is the rollup dimension each breakdown reads. Customer groups read buyers and roll them up, as the fact path does.
var breakdownDimension = map[constants.SalesBreakdownGroupBy]string{
	constants.SalesBreakdownGroupByCustomer:      rollupDimBuyer,
	constants.SalesBreakdownGroupByProduct:       rollupDimItem,
	constants.SalesBreakdownGroupByProductLine:   rollupDimProductLine,
	constants.SalesBreakdownGroupByCustomerGroup: rollupDimBuyer,
	constants.SalesBreakdownGroupBySalesRep:      rollupDimSalesRep,
	constants.SalesBreakdownGroupByDiscount:      rollupDimDiscount,
}

// rollupScopeFor maps a report's filters onto the rollup rows of dimension, or reports false when rollups cannot answer them exactly. A filter on the dimension itself selects its groups; a sales rep filter is part of every row's key; each product line has its own rows. Any other filter needs the lines themselves.
func rollupScopeFor(f domain.SalesReportFilter, q *salesFactQuery, dimension string) (rollupScope, bool) {
	s := rollupScope{dimension: dimension, salesRepIDs: f.SalesRepIDs}
	if q.buyersFiltered {
		if dimension != rollupDimBuyer {
			return s, false
		}
		s.dimensionIDs = q.buyers
	}
	if len(f.ItemIDs) > 0 {
		if dimension != rollupDimItem {
			return s, false
		}
		s.dimensionIDs = f.ItemIDs
	}
	switch {
	case len(f.ProductLineIDs) == 0:
	case dimension == rollupDimProductLine:
		s.dimensionIDs = f.ProductLineIDs
	case len(f.ProductLineIDs) == 1:
		s.lineKey = f.ProductLineIDs[0]
	default:
		s.lineKeys = f.ProductLineIDs
	}
	return s, true
}

// from is the FROM item for s's rows. Left alone, a filtered scope reads the dimension's every bucket
// rather than look up its groups or sales rep; forcing their keys stops that.
func (s rollupScope) from() string {
	var keys []string
	if s.dimensionIDs != nil {
		keys = append(keys, "sales_fact_rollup_dimension_id_idx")
	}
	if len(s.salesRepIDs) > 0 {
		keys = append(keys, "sales_fact_rollup_sales_rep_idx")
	}
	if len(keys) == 0 {
		return "FROM sales_fact_rollup r"
	}
	return "FROM sales_fact_rollup r FORCE INDEX (" + strings.Join(keys, ", ") + ")"
}

// rollupRange is a run of whole buckets of one grain, bucket_start in [from, to).
type rollupRange struct {
	grain    string
	from, to time.Time
}

// rawRange is a stretch of a window no whole bucket covers, read from sales_line_fact: invoiced_at in [from, to), or [from, to] at the window's end.
type rawRange struct {
	from, to    time.Time
	toInclusive bool
}

type windowPlan struct {
	rollups []rollupRange
	raws    []rawRange
}

// planWindow splits the inclusive window [start, end] into the largest whole UTC buckets of grains (largest first) that fit inside it, and the raw stretches left at its edges. Lines are stored to the millisecond, so a bucket fits when it ends no later than a millisecond past end.
func planWindow(start, end time.Time, grains []string) windowPlan {
	var plan windowPlan
	finest := grains[len(grains)-1]
	limit := end.Add(time.Millisecond)
	cur := start.UTC()
	for !cur.After(end) {
		took := false
		for _, g := range grains {
			if next := nextBucket(cur, g); isBucketStart(cur, g) && !next.After(limit) {
				plan.addRollup(g, cur, next)
				cur, took = next, true
				break
			}
		}
		if took {
			continue
		}
		if isBucketStart(cur, finest) {
			// Not even the finest bucket fits: the rest of the window is shorter than one.
			plan.addRaw(cur, end, true)
			break
		}
		boundary := nextBucket(bucketStart(cur, finest), finest)
		if !boundary.Before(limit) {
			plan.addRaw(cur, end, true)
			break
		}
		plan.addRaw(cur, boundary, false)
		cur = boundary
	}
	return plan
}

func (p *windowPlan) addRollup(grain string, from, to time.Time) {
	if n := len(p.rollups); n > 0 && p.rollups[n-1].grain == grain && p.rollups[n-1].to.Equal(from) {
		p.rollups[n-1].to = to
		return
	}
	p.rollups = append(p.rollups, rollupRange{grain: grain, from: from, to: to})
}

func (p *windowPlan) addRaw(from, to time.Time, toInclusive bool) {
	if n := len(p.raws); n > 0 && !p.raws[n-1].toInclusive && p.raws[n-1].to.Equal(from) {
		p.raws[n-1].to, p.raws[n-1].toInclusive = to, toInclusive
		return
	}
	p.raws = append(p.raws, rawRange{from: from, to: to, toInclusive: toInclusive})
}

func bucketStart(t time.Time, grain string) time.Time {
	t = t.UTC()
	switch grain {
	case rollupGrainMonth:
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	case rollupGrainDay:
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	default:
		return t.Truncate(time.Hour)
	}
}

func isBucketStart(t time.Time, grain string) bool {
	return bucketStart(t, grain).Equal(t)
}

func nextBucket(t time.Time, grain string) time.Time {
	switch grain {
	case rollupGrainMonth:
		return t.AddDate(0, 1, 0)
	case rollupGrainDay:
		return t.AddDate(0, 0, 1)
	default:
		return t.Add(time.Hour)
	}
}

// rollupPredicate is the WHERE clause selecting a scope's buckets for the plan, over sales_fact_rollup aliased r; ok is false when the plan has no whole buckets.
func rollupPredicate(accountID string, s rollupScope, ranges []rollupRange) (clause string, args []any, ok bool) {
	if len(ranges) == 0 {
		return "", nil, false
	}
	var sb strings.Builder
	if len(s.lineKeys) > 0 {
		sb.WriteString("r.account_id = ? AND r.sales_order_type_code = 'sales_order' AND r.dimension = ? AND r.product_line_key IN (" + placeholders(len(s.lineKeys)) + ") AND (")
		args = append([]any{accountID, s.dimension}, stringsToAny(s.lineKeys)...)
	} else {
		sb.WriteString("r.account_id = ? AND r.sales_order_type_code = 'sales_order' AND r.dimension = ? AND r.product_line_key = ? AND (")
		args = []any{accountID, s.dimension, s.lineKey}
	}
	for i, rg := range ranges {
		if i > 0 {
			sb.WriteString(" OR ")
		}
		sb.WriteString("(r.grain = ? AND r.bucket_start >= ? AND r.bucket_start < ?)")
		args = append(args, rg.grain, rg.from, rg.to)
	}
	sb.WriteString(")")
	if len(s.salesRepIDs) > 0 {
		sb.WriteString(" AND r.sales_rep_key IN (" + placeholders(len(s.salesRepIDs)) + ")")
		args = append(args, stringsToAny(s.salesRepIDs)...)
	}
	if s.dimensionIDs != nil {
		sb.WriteString(" AND r.dimension_id IN (" + placeholders(len(s.dimensionIDs)) + ")")
		args = append(args, stringsToAny(s.dimensionIDs)...)
	}
	return sb.String(), args, true
}

// rawPredicate is the WHERE clause adding a plan's raw stretches to q's filters over sales_line_fact aliased f; ok is false when the plan has none.
func rawPredicate(q *salesFactQuery, ranges []rawRange) (clause string, args []any, ok bool) {
	if len(ranges) == 0 {
		return "", nil, false
	}
	parts := make([]string, len(ranges))
	args = append(args, q.args...)
	for i, rg := range ranges {
		op := "<"
		if rg.toInclusive {
			op = "<="
		}
		parts[i] = "(f.invoiced_at >= ? AND f.invoiced_at " + op + " ?)"
		args = append(args, rg.from, rg.to)
	}
	return q.where.String() + " AND (" + strings.Join(parts, " OR ") + ")", args, true
}

// rollupPeriodRows selects a period's contributions as (k, per, inv, cost, qty, ic, lc) rows: its whole buckets, and one pre-grouped row per group for its raw stretches. keyCol is the fact column the rows are keyed on and bucketKey the rollup column, so the two halves line up; "”" keys every row alike, for a period's overall totals. start and end are the period, which a multi-line scope counts its invoices over.
func rollupPeriodRows(q *salesFactQuery, s rollupScope, plan windowPlan, start, end time.Time, per, keyCol, bucketKey string) (string, []any) {
	var parts []string
	var args []any
	// A multi-line scope takes its invoice counts from invoiceCountRows alone.
	bucketIC, rawIC := "r.invoice_count", "COUNT(DISTINCT f.invoice_id)"
	if len(s.lineKeys) > 0 {
		bucketIC, rawIC = "0", "0"
	}
	if clause, a, ok := rollupPredicate(q.accountID, s, plan.rollups); ok {
		parts = append(parts, fmt.Sprintf(`SELECT %s AS k, '%s' AS per, r.total_invoiced AS inv, r.total_cost AS cost, r.quantity_base AS qty, %s AS ic, r.line_count AS lc
%s WHERE %s`, bucketKey, per, bucketIC, s.from(), clause))
		args = append(args, a...)
	}
	group := ""
	if keyCol != "''" {
		group = " GROUP BY " + keyCol
	}
	if clause, a, ok := rawPredicate(q, plan.raws); ok {
		parts = append(parts, fmt.Sprintf(`SELECT %s AS k, '%s' AS per, SUM(f.total_invoiced) AS inv, SUM(f.total_cost) AS cost, SUM(f.quantity_base) AS qty, %s AS ic, COUNT(*) AS lc
%s WHERE %s%s`, keyCol, per, rawIC, q.from(), clause, group))
		args = append(args, a...)
	}
	if len(s.lineKeys) > 0 && len(parts) > 0 {
		clause, a := invoiceCountPredicate(q, start, end)
		parts = append(parts, fmt.Sprintf(`SELECT %s AS k, '%s' AS per, 0 AS inv, 0 AS cost, 0 AS qty, COUNT(DISTINCT f.invoice_id) AS ic, 0 AS lc
%s WHERE %s%s`, keyCol, per, q.from(), clause, group))
		args = append(args, a...)
	}
	return strings.Join(parts, "\nUNION ALL\n"), args
}

// invoiceCountPredicate is q's filters over the whole period [start, end], for counting a multi-line scope's
// distinct invoices from the facts. q carries the product line filter, so sales_line_fact_product_line_idx
// reads only those lines' entries in the period.
func invoiceCountPredicate(q *salesFactQuery, start, end time.Time) (string, []any) {
	return q.where.String() + " AND f.invoiced_at >= ? AND f.invoiced_at <= ?", append(append([]any{}, q.args...), start, end)
}

// rollupTotalsColumns totals a union of period rows (alias u) into the columns totalsColumns names, for the rows whose per matches.
func rollupTotalsColumns(per, prefix string) string {
	return fmt.Sprintf(`CAST(COALESCE(SUM(CASE WHEN u.per = '%[1]s' THEN u.inv END), 0) AS DECIMAL(65,30)) AS %[2]sinvoiced,
CAST(COALESCE(SUM(CASE WHEN u.per = '%[1]s' THEN u.cost END), 0) AS DECIMAL(65,30)) AS %[2]scost,
CAST(COALESCE(SUM(CASE WHEN u.per = '%[1]s' THEN u.qty END), 0) AS DECIMAL(65,30)) AS %[2]sqty,
CAST(COALESCE(SUM(CASE WHEN u.per = '%[1]s' THEN u.ic END), 0) AS SIGNED) AS %[2]sinvoice_count,
CAST(COALESCE(SUM(CASE WHEN u.per = '%[1]s' THEN u.lc END), 0) AS SIGNED) AS %[2]sline_count`, per, prefix)
}

var breakdownGrains = []string{rollupGrainMonth, rollupGrainDay}

// rollupBreakdownGrouped is the rollup form of the per-group totals GetBreakdown pages over: one row per group key with c_ and p_ totals columns.
func rollupBreakdownGrouped(q *salesFactQuery, s rollupScope, params domain.AnalyzeSalesBreakdownParams, column string) (string, []any) {
	periods := []struct {
		per        string
		start, end time.Time
	}{{"c", params.StartsAt, params.EndsAt}}
	if params.HasComparison() {
		periods = append(periods, struct {
			per        string
			start, end time.Time
		}{"p", *params.ComparisonStartsAt, *params.ComparisonEndsAt})
	}
	var parts []string
	var args []any
	for _, p := range periods {
		sql, a := rollupPeriodRows(q, s, planWindow(p.start, p.end, breakdownGrains), p.start, p.end, p.per, column, "r.dimension_id")
		if sql != "" {
			parts = append(parts, sql)
			args = append(args, a...)
		}
	}
	if len(parts) == 0 {
		// An empty window: select no groups, with the columns the pager expects.
		return "SELECT '' AS k, " + totalsColumns("FALSE", "c_") + ", " + totalsColumns("FALSE", "p_") + " FROM sales_line_fact f WHERE FALSE", nil
	}
	return fmt.Sprintf("SELECT u.k, %s, %s FROM (\n%s\n) u GROUP BY u.k",
		rollupTotalsColumns("c", "c_"), rollupTotalsColumns("p", "p_"), strings.Join(parts, "\nUNION ALL\n")), args
}

var totalGrains = []string{rollupGrainMonth, rollupGrainDay, rollupGrainHour}

// rollupPeriodTotals selects one period's totals as the five totals columns.
func rollupPeriodTotals(q *salesFactQuery, s rollupScope, start, end time.Time) (string, []any) {
	sql, args := rollupPeriodRows(q, s, planWindow(start, end, totalGrains), start, end, "c", "''", "''")
	if sql == "" {
		return "SELECT " + totalsColumns("FALSE", "") + " FROM sales_line_fact f WHERE FALSE", nil
	}
	return fmt.Sprintf("SELECT %s FROM (\n%s\n) u", rollupTotalsColumns("c", ""), sql), args
}

// rollupPeriodDaily selects one period's totals per local day, from hour buckets, which fall wholly inside a local day when the offset is a whole number of hours.
func rollupPeriodDaily(q *salesFactQuery, s rollupScope, start, end time.Time, tz string) (string, []any) {
	plan := planWindow(start, end, []string{rollupGrainHour})
	day := func(col string) string { return "DATE(CONVERT_TZ(" + col + ", '+00:00', ?))" }
	var parts []string
	var args []any
	// A multi-line scope takes its invoice counts from the per-day count below alone.
	bucketIC, rawIC := "r.invoice_count", "COUNT(DISTINCT f.invoice_id)"
	if len(s.lineKeys) > 0 {
		bucketIC, rawIC = "0", "0"
	}
	if clause, a, ok := rollupPredicate(q.accountID, s, plan.rollups); ok {
		parts = append(parts, `SELECT `+day("r.bucket_start")+` AS day, r.total_invoiced AS inv, r.total_cost AS cost, r.quantity_base AS qty, `+bucketIC+` AS ic, r.line_count AS lc
`+s.from()+` WHERE `+clause)
		args = append(append(args, tz), a...)
	}
	if clause, a, ok := rawPredicate(q, plan.raws); ok {
		parts = append(parts, `SELECT `+day("f.invoiced_at")+` AS day, SUM(f.total_invoiced) AS inv, SUM(f.total_cost) AS cost, SUM(f.quantity_base) AS qty, `+rawIC+` AS ic, COUNT(*) AS lc
`+q.from()+` WHERE `+clause+` GROUP BY day`)
		args = append(append(args, tz), a...)
	}
	if len(parts) == 0 {
		return "", nil
	}
	if len(s.lineKeys) > 0 {
		// An invoice falls on one local day, so its day's distinct count holds it exactly once.
		clause, a := invoiceCountPredicate(q, start, end)
		parts = append(parts, `SELECT `+day("f.invoiced_at")+` AS day, 0 AS inv, 0 AS cost, 0 AS qty, COUNT(DISTINCT f.invoice_id) AS ic, 0 AS lc
`+q.from()+` WHERE `+clause+` GROUP BY day`)
		args = append(append(args, tz), a...)
	}
	return fmt.Sprintf(`SELECT u.day,
CAST(COALESCE(SUM(u.inv), 0) AS DECIMAL(65,30)), CAST(COALESCE(SUM(u.cost), 0) AS DECIMAL(65,30)), CAST(COALESCE(SUM(u.qty), 0) AS DECIMAL(65,30)),
CAST(COALESCE(SUM(u.ic), 0) AS SIGNED), CAST(COALESCE(SUM(u.lc), 0) AS SIGNED)
FROM (
%s
) u GROUP BY u.day ORDER BY u.day`, strings.Join(parts, "\nUNION ALL\n")), args
}
