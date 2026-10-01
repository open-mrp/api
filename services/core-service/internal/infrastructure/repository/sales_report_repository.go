package repository

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/pagination"
	"github.com/open-mrp/api/shared/tracing"
)

var salesReportRepoTracer = tracing.GetTracer("core-service.infrastructure.repository.sales_report")

// Sales reports read sales_line_fact, whose amounts are the legacy per-line analytics expressions evaluated
// once, and sales_fact_rollup, its per-bucket sums. Every SUM below mirrors the dashboard's former aggregate
// term for term (including CASE ... ELSE 0 for period splits), so totals match what it reported. Names are
// joined after aggregation, onto a page of groups rather than every line.
//
// Summaries and breakdowns read whole UTC buckets from the rollups and only the partial buckets at a
// window's edges from the facts (sales_report_rollup.go); decimal sums are exact, so both paths return the
// same totals. Filters the rollups cannot answer exactly fall back to reading the facts alone.
type salesReportRepoImpl struct {
	queries *sqlc.Queries
	mode    rollupMode
}

func NewSalesReportRepo(queries *sqlc.Queries) domain.SalesReportRepo {
	return &salesReportRepoImpl{queries: queries, mode: rollupAuto}
}

func (r *salesReportRepoImpl) FactsReady(ctx context.Context) (bool, *apierror.APIError) {
	sync, apiErr := NewSalesFactRepo(r.queries).GetSync(ctx)
	if apiErr != nil {
		return false, apiErr
	}
	return sync.LastCompletedAt != nil, nil
}

// salesFactQuery accumulates a WHERE clause over sales_line_fact (alias f) and its arguments.
type salesFactQuery struct {
	where     strings.Builder
	args      []any
	accountID string
	// buyers are the buyer accounts the customer filters admit; buyersFiltered is false when neither is set.
	buyers         []string
	buyersFiltered bool
	// withoutBuyers is the clause and the count of its args before the buyer filter was added.
	withoutBuyers     string
	withoutBuyersArgs int
	// entityFiltered is set when a sales rep, product line, or item filter is.
	entityFiltered bool
	// empty is set when a filter resolved to no buyers, so nothing can match.
	empty bool
}

func (q *salesFactQuery) add(clause string, args ...any) {
	q.where.WriteString(" AND ")
	q.where.WriteString(clause)
	q.args = append(q.args, args...)
}

func (q *salesFactQuery) addIn(column string, ids []string) {
	if len(ids) == 0 {
		return
	}
	q.where.WriteString(" AND " + column + " IN (" + placeholders(len(ids)) + ")")
	for _, id := range ids {
		q.args = append(q.args, id)
	}
}

func placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

// periodCase is the predicate that attributes a line to a period, as the dashboard wrote it.
func periodCase(start, end time.Time) (string, []any) {
	return "f.invoiced_at >= ? AND f.invoiced_at <= ?", []any{start, end}
}

// newSalesFactQuery scopes to the account's sales orders and the entity filters. windows adds a date predicate matching any of the given [start, end] periods; none leaves the dates unbounded.
func (r *salesReportRepoImpl) newSalesFactQuery(ctx context.Context, f domain.SalesReportFilter, windows [][2]time.Time) (*salesFactQuery, *apierror.APIError) {
	q := &salesFactQuery{accountID: f.AccountID}
	q.where.WriteString("f.account_id = ? AND f.sales_order_type_code = 'sales_order'")
	q.args = append(q.args, f.AccountID)

	switch len(windows) {
	case 0:
	case 1:
		q.add("f.invoiced_at >= ? AND f.invoiced_at <= ?", windows[0][0], windows[0][1])
	default:
		parts := make([]string, len(windows))
		for i, w := range windows {
			parts[i] = "(f.invoiced_at >= ? AND f.invoiced_at <= ?)"
			q.args = append(q.args, w[0], w[1])
		}
		q.where.WriteString(" AND (" + strings.Join(parts, " OR ") + ")")
	}

	q.addIn("f.sales_rep_id", f.SalesRepIDs)
	q.addIn("f.product_line_id", f.ProductLineIDs)
	q.addIn("f.item_id", f.ItemIDs)
	q.withoutBuyers, q.withoutBuyersArgs = q.where.String(), len(q.args)
	q.entityFiltered = len(f.SalesRepIDs) > 0 || len(f.ProductLineIDs) > 0 || len(f.ItemIDs) > 0

	buyers, filtered, apiErr := r.resolveBuyers(ctx, f)
	if apiErr != nil {
		return nil, apiErr
	}
	if filtered {
		if len(buyers) == 0 {
			q.empty = true
		}
		q.addIn("f.buyer_account_id", buyers)
	}
	q.buyers, q.buyersFiltered = buyers, filtered
	return q, nil
}

// resolveBuyers turns the customer and customer-group filters into the buyer accounts they admit, the way the dashboard's joins did: a customer matches itself and any account whose customer relation names it as parent; a group matches buyers whose customer relation is in it. filtered is false when neither filter is set.
func (r *salesReportRepoImpl) resolveBuyers(ctx context.Context, f domain.SalesReportFilter) (buyers []string, filtered bool, apiErr *apierror.APIError) {
	if len(f.CustomerIDs) == 0 && len(f.CustomerGroupIDs) == 0 {
		return nil, false, nil
	}
	var sets [][]string
	if len(f.CustomerIDs) > 0 {
		query := `SELECT child.counterparty_account_id FROM account_relation child
WHERE child.owner_account_id = ? AND child.account_relation_role_code = 'customer'
  AND child.parent_account_relation_id IN (
      SELECT parent.id FROM account_relation parent
      WHERE parent.owner_account_id = ? AND parent.account_relation_role_code = 'customer'
        AND parent.counterparty_account_id IN (` + placeholders(len(f.CustomerIDs)) + `))`
		args := []any{f.AccountID, f.AccountID}
		for _, id := range f.CustomerIDs {
			args = append(args, id)
		}
		children, apiErr := r.queryStrings(ctx, query, args...)
		if apiErr != nil {
			return nil, true, apiErr
		}
		sets = append(sets, append(append([]string{}, f.CustomerIDs...), children...))
	}
	if len(f.CustomerGroupIDs) > 0 {
		query := `SELECT counterparty_account_id FROM account_relation
WHERE owner_account_id = ? AND account_relation_role_code = 'customer'
  AND account_group_id IN (` + placeholders(len(f.CustomerGroupIDs)) + `)`
		args := []any{f.AccountID}
		for _, id := range f.CustomerGroupIDs {
			args = append(args, id)
		}
		members, apiErr := r.queryStrings(ctx, query, args...)
		if apiErr != nil {
			return nil, true, apiErr
		}
		sets = append(sets, members)
	}

	result := sets[0]
	for _, next := range sets[1:] {
		allowed := make(map[string]struct{}, len(next))
		for _, id := range next {
			allowed[id] = struct{}{}
		}
		kept := result[:0:0]
		for _, id := range result {
			if _, ok := allowed[id]; ok {
				kept = append(kept, id)
			}
		}
		result = kept
	}
	return dedupe(result), true, nil
}

func (r *salesReportRepoImpl) queryStrings(ctx context.Context, query string, args ...any) ([]string, *apierror.APIError) {
	rows, err := r.queries.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, db.MapSQLError(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, db.MapSQLError(err)
		}
		out = append(out, s)
	}
	return out, db.MapSQLError(rows.Err())
}

func reportWindows(f domain.SalesReportFilter) [][2]time.Time {
	windows := [][2]time.Time{{f.StartsAt, f.EndsAt}}
	if f.HasComparison() {
		windows = append(windows, [2]time.Time{*f.ComparisonStartsAt, *f.ComparisonEndsAt})
	}
	return windows
}

// totalsColumns selects one SalesTotals for the lines matching pred, as columns named <prefix>invoiced, <prefix>cost, <prefix>qty, <prefix>invoice_count and <prefix>line_count. pred's arguments must be bound five times, in order.
func totalsColumns(pred, prefix string) string {
	return fmt.Sprintf(`CAST(COALESCE(SUM(CASE WHEN %[1]s THEN f.total_invoiced END), 0) AS DECIMAL(65,30)) AS %[2]sinvoiced,
CAST(COALESCE(SUM(CASE WHEN %[1]s THEN f.total_cost END), 0) AS DECIMAL(65,30)) AS %[2]scost,
CAST(COALESCE(SUM(CASE WHEN %[1]s THEN f.quantity_base END), 0) AS DECIMAL(65,30)) AS %[2]sqty,
COUNT(DISTINCT CASE WHEN %[1]s THEN f.invoice_id END) AS %[2]sinvoice_count,
COUNT(CASE WHEN %[1]s THEN 1 END) AS %[2]sline_count`, pred, prefix)
}

func repeatArgs(args []any, n int) []any {
	out := make([]any, 0, len(args)*n)
	for range n {
		out = append(out, args...)
	}
	return out
}

// scannedTotals receives the five totals columns.
type scannedTotals struct {
	invoiced, cost, qty sql.NullString
	invoices, lines     int64
}

func (t *scannedTotals) dest() []any {
	return []any{&t.invoiced, &t.cost, &t.qty, &t.invoices, &t.lines}
}

func (t *scannedTotals) totals(includeCost bool) domain.SalesTotals {
	out := domain.SalesTotals{
		Invoiced:     exactDecimal(t.invoiced),
		Quantity:     exactDecimal(t.qty),
		InvoiceCount: t.invoices,
		LineCount:    t.lines,
	}
	if includeCost {
		cost := exactDecimal(t.cost)
		out.Cost = &cost
	}
	return out
}

func zeroTotals(includeCost bool) domain.SalesTotals {
	t := scannedTotals{}
	return t.totals(includeCost)
}

func (r *salesReportRepoImpl) GetSummary(ctx context.Context, params domain.AnalyzeSalesSummaryParams, includeCost bool) (*domain.SalesSummary, *apierror.APIError) {
	ctx, span := salesReportRepoTracer.Start(ctx, "repository.sales_report.get_summary")
	defer span.End()

	periods := reportWindows(params.SalesReportFilter)
	summary := &domain.SalesSummary{}
	totals := []*domain.SalesTotals{&summary.Current}
	daily := []*[]domain.SalesTotals{&summary.Daily}
	if params.HasComparison() {
		summary.Comparison = &domain.SalesTotals{}
		totals = append(totals, summary.Comparison)
		daily = append(daily, &summary.ComparisonDaily)
	}

	// Each period is read on its own, so a line in two overlapping periods counts in both, as it did when the dashboard fetched each period separately.
	tz := tzOffsetString(params.TZOffsetMinutes)
	// Hour buckets split cleanly into local days only when the offset is whole hours.
	useRollups := params.TZOffsetMinutes%60 == 0 && r.rollupsReady(ctx)
	for i, w := range periods {
		if useRollups {
			q, apiErr := r.newSalesFactQuery(ctx, params.SalesReportFilter, nil)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			if scope, ok := rollupScopeFor(params.SalesReportFilter, q, rollupDimTotal); ok && !q.empty {
				totalsQuery, totalsArgs := rollupPeriodTotals(q, scope, w[0], w[1])
				dailyQuery, dailyArgs := rollupPeriodDaily(q, scope, w[0], w[1], tz)
				if *totals[i], *daily[i], apiErr = r.summaryPeriod(ctx, totalsQuery, totalsArgs, dailyQuery, dailyArgs, includeCost); apiErr != nil {
					return nil, tracing.Trace(span, apiErr)
				}
				continue
			}
		}

		q, apiErr := r.newSalesFactQuery(ctx, params.SalesReportFilter, [][2]time.Time{w})
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if q.empty {
			*totals[i] = zeroTotals(includeCost)
			*daily[i] = []domain.SalesTotals{}
			continue
		}
		totalsQuery := "SELECT " + totalsColumns("TRUE", "") + " FROM sales_line_fact f WHERE " + q.where.String()
		dailyQuery := "SELECT DATE(CONVERT_TZ(f.invoiced_at, '+00:00', ?)) AS day, " + totalsColumns("TRUE", "") +
			" FROM sales_line_fact f WHERE " + q.where.String() + " GROUP BY day ORDER BY day"
		if *totals[i], *daily[i], apiErr = r.summaryPeriod(ctx, totalsQuery, q.args, dailyQuery, append([]any{tz}, q.args...), includeCost); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}
	return summary, nil
}

// summaryPeriod runs a period's totals query (the five totals columns) and its daily query (a day, then the five columns, per row); an empty daily query means no days.
func (r *salesReportRepoImpl) summaryPeriod(ctx context.Context, totalsQuery string, totalsArgs []any, dailyQuery string, dailyArgs []any, includeCost bool) (domain.SalesTotals, []domain.SalesTotals, *apierror.APIError) {
	var whole scannedTotals
	if err := r.queries.DB().QueryRowContext(ctx, totalsQuery, totalsArgs...).Scan(whole.dest()...); err != nil {
		return domain.SalesTotals{}, nil, db.MapSQLError(err)
	}
	days := []domain.SalesTotals{}
	if dailyQuery == "" {
		return whole.totals(includeCost), days, nil
	}
	rows, err := r.queries.DB().QueryContext(ctx, dailyQuery, dailyArgs...)
	if err != nil {
		return domain.SalesTotals{}, nil, db.MapSQLError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			day time.Time
			t   scannedTotals
		)
		if err := rows.Scan(append([]any{&day}, t.dest()...)...); err != nil {
			return domain.SalesTotals{}, nil, db.MapSQLError(err)
		}
		dt := t.totals(includeCost)
		dt.PeriodStart = &day
		days = append(days, dt)
	}
	if err := rows.Err(); err != nil {
		return domain.SalesTotals{}, nil, db.MapSQLError(err)
	}
	return whole.totals(includeCost), days, nil
}

func tzOffsetString(minutes int32) string {
	sign := '+'
	if minutes < 0 {
		sign = '-'
		minutes = -minutes
	}
	return fmt.Sprintf("%c%02d:%02d", sign, minutes/60, minutes%60)
}

// breakdownColumn is the fact column each dimension aggregates on first. Customer groups aggregate by buyer, then roll buyers up into their group.
var breakdownColumn = map[constants.SalesBreakdownGroupBy]string{
	constants.SalesBreakdownGroupByCustomer:      "f.buyer_account_id",
	constants.SalesBreakdownGroupByProduct:       "f.item_id",
	constants.SalesBreakdownGroupByProductLine:   "f.product_line_id",
	constants.SalesBreakdownGroupByCustomerGroup: "f.buyer_account_id",
	constants.SalesBreakdownGroupBySalesRep:      "f.sales_rep_id",
	constants.SalesBreakdownGroupByDiscount:      "f.order_discount_id",
}

// breakdownTotalColumns are the aggregate columns of a grouped row: c_ for the current period, p_ for the comparison period.
var breakdownTotalColumns = []string{"c_invoiced", "c_cost", "c_qty", "c_invoice_count", "c_line_count", "p_invoiced", "p_cost", "p_qty", "p_invoice_count", "p_line_count"}

func (r *salesReportRepoImpl) GetBreakdown(ctx context.Context, params domain.AnalyzeSalesBreakdownParams, includeCost bool) (*domain.SalesBreakdown, *apierror.APIError) {
	ctx, span := salesReportRepoTracer.Start(ctx, "repository.sales_report.get_breakdown")
	defer span.End()

	cur, apiErr := decodeBreakdownCursor(params.Cursor)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	column, ok := breakdownColumn[params.GroupBy]
	if !ok {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError(fmt.Sprintf("Unknown sales breakdown group %q.", params.GroupBy)))
	}
	// The rollup path filters facts without a window: its raw stretches bring their own.
	var windows [][2]time.Time
	useRollups := r.rollupsReady(ctx)
	if !useRollups {
		windows = reportWindows(params.SalesReportFilter)
	}
	q, apiErr := r.newSalesFactQuery(ctx, params.SalesReportFilter, windows)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if q.empty {
		return &domain.SalesBreakdown{}, nil
	}
	switch params.GroupBy {
	case constants.SalesBreakdownGroupBySalesRep:
		q.add("f.sales_rep_id IS NOT NULL")
	case constants.SalesBreakdownGroupByDiscount:
		q.add("f.order_discount_id IS NOT NULL")
	}

	var (
		grouped string
		args    []any
	)
	if useRollups {
		scope, ok := rollupScopeFor(params.SalesReportFilter, q, breakdownDimension[params.GroupBy])
		if ok {
			grouped, args = rollupBreakdownGrouped(q, scope, params, column)
		} else {
			// The filters need the lines themselves; add back the window the rollup path left off.
			q, apiErr = r.newSalesFactQuery(ctx, params.SalesReportFilter, reportWindows(params.SalesReportFilter))
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			switch params.GroupBy {
			case constants.SalesBreakdownGroupBySalesRep:
				q.add("f.sales_rep_id IS NOT NULL")
			case constants.SalesBreakdownGroupByDiscount:
				q.add("f.order_discount_id IS NOT NULL")
			}
		}
	}
	if grouped == "" {
		grouped, args = factBreakdownGrouped(q, params, column)
	}

	// Customer groups re-total their buyers' rows. Decimal addition is exact, and an invoice has one buyer, so every sum (invoice counts included) equals totalling the group's lines directly.
	if params.GroupBy == constants.SalesBreakdownGroupByCustomerGroup {
		sums := make([]string, len(breakdownTotalColumns))
		for i, c := range breakdownTotalColumns {
			sums[i] = fmt.Sprintf("SUM(b.%[1]s) AS %[1]s", c)
		}
		grouped = `SELECT ar.account_group_id AS k, ` + strings.Join(sums, ", ") + `
FROM (` + grouped + `) b
JOIN account_relation ar ON ar.owner_account_id = ? AND ar.counterparty_account_id = b.k AND ar.account_relation_role_code = 'customer'
WHERE ar.account_group_id IS NOT NULL
GROUP BY ar.account_group_id`
		args = append(args, params.AccountID)
	}

	// A product with invoiced lines stays listed at a zero total, as the dashboard's product view did; the other views drop groups whose current total is zero.
	having := "g.c_invoiced <> 0"
	if params.GroupBy == constants.SalesBreakdownGroupByProduct {
		having = "g.c_line_count > 0"
	}

	// Groups are ranked by current revenue, then key. A page is a keyset over that ranking, compared as
	// exact decimals, so a group never repeats or goes missing between pages; a backward page reads the
	// ranking in reverse and is turned back around below.
	order, pageOrder := "g.c_invoiced DESC, g.k ASC", "p.c_invoiced DESC, p.k ASC"
	if cur != nil {
		if cur.Direction == pagination.DirectionBackward {
			having += " AND (g.c_invoiced > CAST(? AS DECIMAL(65,30)) OR (g.c_invoiced = CAST(? AS DECIMAL(65,30)) AND g.k < ?))"
			order, pageOrder = "g.c_invoiced ASC, g.k DESC", "p.c_invoiced ASC, p.k DESC"
		} else {
			having += " AND (g.c_invoiced < CAST(? AS DECIMAL(65,30)) OR (g.c_invoiced = CAST(? AS DECIMAL(65,30)) AND g.k > ?))"
		}
		args = append(args, cur.Value, cur.Value, cur.ID)
	}

	label, joins := breakdownLabel(params.GroupBy)
	query := fmt.Sprintf(`SELECT p.k, %s, p.%s FROM (
  SELECT g.* FROM (%s) g WHERE %s ORDER BY %s LIMIT ?
) p %s
ORDER BY %s`, label, strings.Join(breakdownTotalColumns, ", p."), grouped, having, order, joins, pageOrder)
	args = append(args, params.Limit+1)

	rows, err := r.queries.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	defer rows.Close()
	out := &domain.SalesBreakdown{}
	var ranks []string // each group's current revenue as stored, for its cursor
	for rows.Next() {
		var (
			g                           domain.SalesBreakdownGroup
			labelVal, description, unit sql.NullString
			current, comparison         scannedTotals
		)
		dest := append([]any{&g.Key, &labelVal, &description, &unit}, current.dest()...)
		dest = append(dest, comparison.dest()...)
		if err := rows.Scan(dest...); err != nil {
			return nil, tracing.Trace(span, db.MapSQLError(err))
		}
		g.Label = labelVal.String
		g.Description, g.UnitAbbreviation = nullStringPtr(description), nullStringPtr(unit)
		g.Totals = current.totals(includeCost)
		if params.HasComparison() {
			c := comparison.totals(includeCost)
			g.Comparison = &c
		}
		out.Groups = append(out.Groups, g)
		ranks = append(ranks, current.invoiced.String)
	}
	if err := rows.Err(); err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	out.Groups, out.PageInfo = breakdownPage(out.Groups, ranks, params.Limit, cur)
	return out, nil
}

// factBreakdownGrouped totals the matching lines per group straight from sales_line_fact, one row per group key with c_ and p_ totals columns.
func factBreakdownGrouped(q *salesFactQuery, params domain.AnalyzeSalesBreakdownParams, column string) (string, []any) {
	cur, curArgs := periodCase(params.StartsAt, params.EndsAt)
	args := repeatArgs(curArgs, 5)
	// Without a comparison period its columns are all zero: FALSE matches no line.
	cmp, cmpArgs := "FALSE", []any(nil)
	if params.HasComparison() {
		cmp, cmpArgs = periodCase(*params.ComparisonStartsAt, *params.ComparisonEndsAt)
	}
	args = append(args, repeatArgs(cmpArgs, 5)...)
	grouped := fmt.Sprintf("SELECT %s AS k, %s, %s FROM sales_line_fact f WHERE %s GROUP BY %s",
		column, totalsColumns(cur, "c_"), totalsColumns(cmp, "p_"), q.where.String(), column)
	return grouped, append(args, q.args...)
}

func decodeBreakdownCursor(cursor *string) (*pagination.ValueCursor, *apierror.APIError) {
	if cursor == nil || *cursor == "" {
		return nil, nil
	}
	c, err := pagination.DecodeValueCursor(*cursor)
	if err != nil {
		return nil, apierror.NewParameterInvalidError("The cursor is invalid.", "cursor")
	}
	return &c, nil
}

// breakdownPage trims a page read one past limit (in reverse for a backward page) and cursors its ends.
// ranks holds each group's revenue as stored, in the order read.
func breakdownPage(groups []domain.SalesBreakdownGroup, ranks []string, limit int32, cur *pagination.ValueCursor) ([]domain.SalesBreakdownGroup, pagination.PageInfo) {
	var info pagination.PageInfo
	more := len(groups) > int(limit)
	if more {
		groups, ranks = groups[:limit], ranks[:limit]
	}
	backward := cur != nil && cur.Direction == pagination.DirectionBackward
	if backward {
		slices.Reverse(groups)
		slices.Reverse(ranks)
	}
	// Forward: more rows lie ahead, and a cursor means rows lie behind. Backward, the reverse.
	info.HasNextPage, info.HasPrevPage = more, cur != nil
	if backward {
		info.HasNextPage, info.HasPrevPage = true, more
	}
	if len(groups) == 0 {
		return groups, pagination.PageInfo{}
	}
	if info.HasNextPage {
		last := len(groups) - 1
		next := pagination.EncodeValueCursor(pagination.ValueCursor{Value: ranks[last], ID: groups[last].Key, Direction: pagination.DirectionForward})
		info.NextCursor = &next
	}
	if info.HasPrevPage {
		prev := pagination.EncodeValueCursor(pagination.ValueCursor{Value: ranks[0], ID: groups[0].Key, Direction: pagination.DirectionBackward})
		info.PrevCursor = &prev
	}
	return groups, info
}

// keysetCursor decodes a (timestamp, id) cursor, returning its direction or nil for the first page.
func keysetCursor(cursor *string) (*pagination.StringCursor, *apierror.APIError) {
	if cursor == nil || *cursor == "" {
		return nil, nil
	}
	c, err := pagination.DecodeStringCursor(*cursor)
	if err != nil {
		return nil, apierror.NewParameterInvalidError("The cursor is invalid.", "cursor")
	}
	return &c, nil
}

// keysetPredicate seeks past a cursor in (at DESC, id DESC) order: forward continues older, backward walks back toward newer (read ascending, then reversed by BuildPageString).
func keysetPredicate(c *pagination.StringCursor, atCol, idCol string) (clause string, args []any, order string) {
	if c == nil {
		return "", nil, "DESC"
	}
	if c.Direction == pagination.DirectionBackward {
		return fmt.Sprintf("(%[1]s > ? OR (%[1]s = ? AND %[2]s > ?))", atCol, idCol), []any{c.OccurredAt, c.OccurredAt, c.ID}, "ASC"
	}
	return fmt.Sprintf("(%[1]s < ? OR (%[1]s = ? AND %[2]s < ?))", atCol, idCol), []any{c.OccurredAt, c.OccurredAt, c.ID}, "DESC"
}

func cursorDirection(c *pagination.StringCursor) *pagination.Direction {
	if c == nil {
		return nil
	}
	return &c.Direction
}

// breakdownLabel returns the label, description and unit select-list (in that order) for a page of groups aliased p, and the joins it needs.
func breakdownLabel(groupBy constants.SalesBreakdownGroupBy) (string, string) {
	const noProduct = "NULL, NULL"
	switch groupBy {
	case constants.SalesBreakdownGroupByCustomer:
		return "a.name, " + noProduct, "LEFT JOIN account a ON a.id = p.k"
	case constants.SalesBreakdownGroupByProduct:
		return "it.sku, it.description, bu.abbreviation",
			`LEFT JOIN item it ON it.id = p.k
LEFT JOIN item_category ic ON ic.id = it.item_category_id
LEFT JOIN unit_group ug ON ug.id = ic.unit_group_id
LEFT JOIN unit bu ON bu.id = ug.base_unit_id`
	case constants.SalesBreakdownGroupByProductLine:
		return "pl.name, " + noProduct, "LEFT JOIN product_line pl ON pl.id = p.k"
	case constants.SalesBreakdownGroupByCustomerGroup:
		return "ag.name, " + noProduct, "LEFT JOIN account_group ag ON ag.id = p.k"
	case constants.SalesBreakdownGroupBySalesRep:
		return "u.username, " + noProduct, "LEFT JOIN account_user au ON au.id = p.k LEFT JOIN `user` u ON u.id = au.user_id"
	default:
		return "od.code, " + noProduct, "LEFT JOIN order_discount od ON od.id = p.k"
	}
}

func (r *salesReportRepoImpl) GetInvoicePage(ctx context.Context, params domain.AnalyzeSalesInvoicesParams) (*domain.SalesInvoicePage, *apierror.APIError) {
	ctx, span := salesReportRepoTracer.Start(ctx, "repository.sales_report.get_invoice_page")
	defer span.End()

	cursor, apiErr := keysetCursor(params.Cursor)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	q, apiErr := r.newSalesFactQuery(ctx, params.SalesReportFilter, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if q.empty {
		return &domain.SalesInvoicePage{}, nil
	}

	// Walk the account's invoices newest first on invoice_account_created_idx and keep those with a matching
	// sale line, stopping at the page size. Left to itself the optimizer turns the EXISTS into a semi-join
	// that reads every fact the account has before the LIMIT applies; NO_SEMIJOIN keeps it a per-invoice
	// probe of sales_line_fact_invoice_idx.
	var sb strings.Builder
	args := []any{params.AccountID, params.StartsAt, params.EndsAt}
	sb.WriteString(`SELECT i.id, i.number, i.created_at, so.buyer_account_id, COALESCE(a.name, '') FROM invoice i FORCE INDEX (invoice_account_created_idx)
JOIN sales_order so ON so.id = i.sales_order_id
LEFT JOIN account a ON a.id = so.buyer_account_id
WHERE i.account_id = ? AND i.created_at >= ? AND i.created_at <= ?`)
	seek, seekArgs, order := keysetPredicate(cursor, "i.created_at", "i.id")
	if seek != "" {
		sb.WriteString(" AND " + seek)
		args = append(args, seekArgs...)
	}
	sb.WriteString(" AND EXISTS (SELECT /*+ NO_SEMIJOIN() */ 1 FROM sales_line_fact f WHERE f.invoice_id = i.id AND ")
	sb.WriteString(q.where.String())
	args = append(args, q.args...)
	sb.WriteString(") ORDER BY i.created_at " + order + ", i.id " + order + " LIMIT ?")
	args = append(args, params.Limit+1)

	rows, err := r.queries.DB().QueryContext(ctx, sb.String(), args...)
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	page := &domain.SalesInvoicePage{}
	for rows.Next() {
		var inv domain.SalesInvoiceSummary
		if err := rows.Scan(&inv.InvoiceID, &inv.InvoiceNumber, &inv.InvoicedAt, &inv.CustomerID, &inv.CustomerName); err != nil {
			_ = rows.Close()
			return nil, tracing.Trace(span, db.MapSQLError(err))
		}
		page.Invoices = append(page.Invoices, inv)
	}
	_ = rows.Close()
	if err := rows.Err(); err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	page.Invoices, page.PageInfo = pagination.BuildPageString(page.Invoices, params.Limit, cursorDirection(cursor),
		func(inv domain.SalesInvoiceSummary) time.Time { return inv.InvoicedAt },
		func(inv domain.SalesInvoiceSummary) string { return inv.InvoiceID })
	if len(page.Invoices) == 0 {
		return page, nil
	}

	ids := make([]string, len(page.Invoices))
	for i, inv := range page.Invoices {
		ids[i] = inv.InvoiceID
	}
	totalsQuery := `SELECT f.invoice_id, COUNT(DISTINCT f.item_id), CAST(COALESCE(SUM(f.total_invoiced), 0) AS DECIMAL(65,30))
FROM sales_line_fact f WHERE ` + q.where.String() + ` AND f.invoice_id IN (` + placeholders(len(ids)) + `) GROUP BY f.invoice_id`
	totalsArgs := append(append([]any{}, q.args...), stringsToAny(ids)...)
	trows, err := r.queries.DB().QueryContext(ctx, totalsQuery, totalsArgs...)
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	defer trows.Close()
	type totals struct {
		items    int64
		invoiced string
	}
	byInvoice := make(map[string]totals, len(ids))
	for trows.Next() {
		var (
			id    string
			items int64
			total sql.NullString
		)
		if err := trows.Scan(&id, &items, &total); err != nil {
			return nil, tracing.Trace(span, db.MapSQLError(err))
		}
		byInvoice[id] = totals{items: items, invoiced: exactDecimal(total)}
	}
	if err := trows.Err(); err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	for i := range page.Invoices {
		t, ok := byInvoice[page.Invoices[i].InvoiceID]
		if !ok {
			t.invoiced = "0"
		}
		page.Invoices[i].ItemCount = t.items
		page.Invoices[i].Invoiced = t.invoiced
	}
	return page, nil
}

func stringsToAny(ids []string) []any {
	out := make([]any, len(ids))
	for i, id := range ids {
		out[i] = id
	}
	return out
}

// salesLineColumns are the legacy sales-entry row, derived from a page of facts (f) instead of re-pricing: unit amounts divide the stored totals by the stored base quantity, exactly as the dashboard's expressions did.
const salesLineColumns = `f.invoice_line_id, so.issued_at, so.completed_at, so.first_ship_at, so.promised_at,
f.invoiced_at, f.invoice_id, i.number, so.customer_po_number, so.number, f.sales_order_id,
f.sales_rep_id, sr_user.username, f.buyer_account_id, parent_ar.counterparty_account_id, buyer.name,
ar.external_number, buyer.created_at, ar.account_group_id, ag.name, f.product_line_id, fg.product_type_code,
f.item_id, pb.sku, pb.description, ic.name, pl.name, bu_unit.abbreviation,
f.quantity_base, f.total_invoiced, f.total_cost, CAST(f.total_invoiced - f.total_cost AS DECIMAL(65,30)),
CAST(f.total_invoiced / NULLIF(f.quantity_base, 0) AS DECIMAL(65,30)),
CAST(f.total_cost / NULLIF(f.quantity_base, 0) AS DECIMAL(65,30)),
CAST((f.total_invoiced - f.total_cost) / NULLIF(f.quantity_base, 0) AS DECIMAL(65,30)),
geo.state, geo.locality, geo.postal_code, geo.country, od.code`

// salesLineIndexHint is the fact keys a page may be read from, each in list order: each set filter's,
// and the clustered key only when no filter has a single value. A key pinned to one value stops at a
// page or reads just its range; walking the account past every line a rare combination rejects reads
// the account whole.
func salesLineIndexHint(f domain.SalesReportFilter, buyers []string, buyersFiltered bool) []string {
	buyerValues := 0
	if buyersFiltered {
		buyerValues = len(buyers)
	}
	var keys []string
	pinned := false
	for _, filter := range []struct {
		key    string
		values int
	}{
		{"sales_line_fact_buyer_idx", buyerValues},
		{"sales_line_fact_sales_rep_idx", len(f.SalesRepIDs)},
		{"sales_line_fact_item_idx", len(f.ItemIDs)},
		{"sales_line_fact_account_product_line_idx", len(f.ProductLineIDs)},
	} {
		if filter.values > 0 {
			keys = append(keys, filter.key)
		}
		pinned = pinned || filter.values == 1
	}
	if !pinned {
		keys = append(keys, "PRIMARY")
	}
	return keys
}

// salesLineBuyerBranches is the most buyers a page is read buyer by buyer for. A customer filter admits
// the customer's child accounts too, and no key yields several buyers' lines in one order: read
// together they are a range to sort whole, read apart each stops at a page. With a sales rep, item, or
// product line filter as well, that filter's key is read instead.
const salesLineBuyerBranches = 8

// salesLinePageQuery is the statement choosing one page of facts (alias-free columns of sales_line_fact),
// its args, and the direction it is read in.
func salesLinePageQuery(q *salesFactQuery, f domain.SalesReportFilter, cursor *pagination.StringCursor, limit int32) (string, []any, string) {
	seek, seekArgs, order := keysetPredicate(cursor, "f.invoiced_at", "f.invoice_line_id")
	orderBy := func(alias string) string {
		return alias + ".invoiced_at " + order + ", " + alias + ".invoice_line_id " + order
	}
	if q.buyersFiltered && len(q.buyers) > 1 && len(q.buyers) <= salesLineBuyerBranches && !q.entityFiltered {
		var args []any
		branches := make([]string, len(q.buyers))
		for i, buyer := range q.buyers {
			where := q.withoutBuyers + " AND f.buyer_account_id = ?"
			args = append(append(args, q.args[:q.withoutBuyersArgs]...), buyer)
			if seek != "" {
				where += " AND " + seek
				args = append(args, seekArgs...)
			}
			branches[i] = "(SELECT f.* FROM sales_line_fact f FORCE INDEX (sales_line_fact_buyer_idx) WHERE " + where +
				" ORDER BY " + orderBy("f") + " LIMIT ?)"
			args = append(args, limit)
		}
		return "SELECT u.* FROM (" + strings.Join(branches, "\nUNION ALL ") + ") u ORDER BY " + orderBy("u") + " LIMIT ?",
			append(args, limit), order
	}
	if seek != "" {
		q.add(seek, seekArgs...)
	}
	// Intersecting two filters' keys reads both ranges whole; one key with the other filter residual
	// stops at a page, or reads no more than its own range.
	return "SELECT /*+ SET_VAR(optimizer_switch = 'index_merge_intersection=off') */ f.* FROM sales_line_fact f FORCE INDEX (" +
		strings.Join(salesLineIndexHint(f, q.buyers, q.buyersFiltered), ", ") + ") WHERE " +
		q.where.String() + " ORDER BY " + orderBy("f") + " LIMIT ?", append(q.args, limit), order
}

func (r *salesReportRepoImpl) GetLinePage(ctx context.Context, params domain.ListSalesLinesParams) (*domain.SalesLinePage, *apierror.APIError) {
	ctx, span := salesReportRepoTracer.Start(ctx, "repository.sales_report.get_line_page")
	defer span.End()

	cursor, apiErr := keysetCursor(params.Cursor)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	var windows [][2]time.Time
	if params.HasWindow {
		windows = [][2]time.Time{{params.StartsAt, params.EndsAt}}
	}
	q, apiErr := r.newSalesFactQuery(ctx, params.SalesReportFilter, windows)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if q.empty {
		return &domain.SalesLinePage{}, nil
	}
	pageQuery, args, order := salesLinePageQuery(q, params.SalesReportFilter, cursor, params.Limit+1)

	// The page is chosen from the facts alone; only its rows are joined out.
	query := `SELECT ` + salesLineColumns + ` FROM (` + pageQuery + `) f
JOIN invoice i ON i.id = f.invoice_id
JOIN sales_order so ON so.id = f.sales_order_id
JOIN product fg ON fg.id = f.product_id
JOIN item pb ON pb.id = f.item_id
JOIN item_category ic ON ic.id = pb.item_category_id
JOIN product_line pl ON pl.id = f.product_line_id
LEFT JOIN unit_group ug ON ug.id = ic.unit_group_id
LEFT JOIN unit bu_unit ON bu_unit.id = ug.base_unit_id
LEFT JOIN account buyer ON buyer.id = f.buyer_account_id
LEFT JOIN account_relation ar ON ar.owner_account_id = so.owner_account_id
    AND ar.counterparty_account_id = so.buyer_account_id
    AND ar.account_relation_role_code = 'customer'
LEFT JOIN account_relation parent_ar ON parent_ar.id = ar.parent_account_relation_id
    AND parent_ar.owner_account_id = ar.owner_account_id
LEFT JOIN account_group ag ON ag.id = ar.account_group_id
LEFT JOIN account_user sr ON sr.id = f.sales_rep_id
LEFT JOIN ` + "`user`" + ` sr_user ON sr_user.id = sr.user_id
LEFT JOIN address ship_addr ON ship_addr.id = so.shipping_address_id
LEFT JOIN geolocation geo ON geo.id = ship_addr.geolocation_id
LEFT JOIN order_discount od ON od.id = f.order_discount_id
ORDER BY f.invoiced_at ` + order + `, f.invoice_line_id ` + order

	rows, err := r.queries.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	defer rows.Close()
	page := &domain.SalesLinePage{}
	for rows.Next() {
		var (
			e                                                           domain.SalesEntry
			issuedAt, completedAt, firstShipAt, promisedAt              sql.NullTime
			customerPO, repID, repName, parentID, customerName, custNum sql.NullString
			groupID, groupName, productLineID, description, unit        sql.NullString
			qty, total, cost, profit, unitPrice, unitCost, unitProfit   sql.NullString
			state, city, zip, country, discount                         sql.NullString
			customerCreatedAt                                           sql.NullTime
		)
		if err := rows.Scan(&e.ID, &issuedAt, &completedAt, &firstShipAt, &promisedAt,
			&e.InvoiceDate, &e.InvoiceID, &e.InvoiceNumber, &customerPO, &e.SalesOrderNumber, &e.SalesOrderID,
			&repID, &repName, &e.CustomerID, &parentID, &customerName,
			&custNum, &customerCreatedAt, &groupID, &groupName, &productLineID, &e.ProductTypeCode,
			&e.ItemID, &e.ProductSku, &description, &e.CategoryName, &e.ProductLine, &unit,
			&qty, &total, &cost, &profit, &unitPrice, &unitCost, &unitProfit,
			&state, &city, &zip, &country, &discount); err != nil {
			return nil, tracing.Trace(span, db.MapSQLError(err))
		}
		e.IssuedAt, e.CompletedAt, e.FirstShipAt, e.PromisedAt = nullTimePtr(issuedAt), nullTimePtr(completedAt), nullTimePtr(firstShipAt), nullTimePtr(promisedAt)
		e.CustomerPO, e.SalesRepID, e.SalesRepUsername, e.ParentCustomerID = nullStringPtr(customerPO), nullStringPtr(repID), nullStringPtr(repName), nullStringPtr(parentID)
		e.CustomerName, e.CustomerNumber = customerName.String, custNum.String
		e.CustomerCreatedAt = customerCreatedAt.Time
		e.CustomerTypeGroupID, e.CustomerGroupName, e.ProductLineID = nullStringPtr(groupID), nullStringPtr(groupName), nullStringPtr(productLineID)
		e.ProductDescription, e.Unit = nullStringPtr(description), unit.String
		e.QuantityInvoiced, e.TotalInvoiced, e.TotalCost, e.TotalProfit = decimalToFloat(qty), decimalToFloat(total), decimalToFloat(cost), decimalToFloat(profit)
		e.UnitPrice, e.UnitCost, e.UnitProfit = decimalToFloat(unitPrice), decimalToFloat(unitCost), decimalToFloat(unitProfit)
		e.ShipToState, e.ShipToCity, e.ShipToPostalCode, e.ShipToCountry = nullStringPtr(state), nullStringPtr(city), nullStringPtr(zip), nullStringPtr(country)
		e.OrderDiscountCode = nullStringPtr(discount)
		page.Lines = append(page.Lines, e)
	}
	if err := rows.Err(); err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	page.Lines, page.PageInfo = pagination.BuildPageString(page.Lines, params.Limit, cursorDirection(cursor),
		func(e domain.SalesEntry) time.Time { return e.InvoiceDate },
		func(e domain.SalesEntry) string { return e.ID })
	return page, nil
}

// exactDecimal renders a DECIMAL result without its trailing zeros, keeping every significant digit; NULL is zero.
func exactDecimal(ns sql.NullString) string {
	if !ns.Valid || ns.String == "" {
		return "0"
	}
	v := ns.String
	if strings.Contains(v, ".") {
		v = strings.TrimRight(strings.TrimRight(v, "0"), ".")
	}
	if v == "" || v == "-" || v == "-0" {
		return "0"
	}
	return v
}

// decimalToFloat parses a DECIMAL result to the nearest float64, the same rounding Decimal.toNumber() applied in the dashboard; NULL is zero, as it was there.
func decimalToFloat(ns sql.NullString) float64 {
	if !ns.Valid {
		return 0
	}
	f, err := strconv.ParseFloat(ns.String, 64)
	if err != nil {
		return 0
	}
	return f
}
