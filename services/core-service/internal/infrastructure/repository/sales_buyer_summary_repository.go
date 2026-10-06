package repository

import (
	"context"
	"database/sql"
	"fmt"
	"math/big"
	"slices"
	"strings"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

// salesBuyerSummarySyncName is the buyer summary sweep's row in sales_sync.
const salesBuyerSummarySyncName = "sales_buyer_summary"

// buyerMarkID is a buyer summary mark's scope_id. A customer account can buy from several accounts, so the
// id names the pair; scope_id alone is the mark table's key within a scope type.
func buyerMarkID(b domain.SalesBuyerKey) string {
	return b.AccountID + "/" + b.BuyerAccountID
}

// salesBuyerSummaryPartial is one account's qualifying sales per buyer within an optional invoiced_at window,
// as the legacy new-customers report counted them: sales orders only, lines priced above zero, outside the
// shipping and misc product lines. The first %s is the buyer_account_id IN list, the second the window.
// The inner query reads sales_line_fact_buyer_summary_idx alone; product line names join only to its groups.
const salesBuyerSummaryPartial = `SELECT g.buyer_account_id, MIN(g.first_ordered_at), SUM(g.total_invoiced)
FROM (
  SELECT f.buyer_account_id, f.product_line_id, MIN(f.ordered_at) AS first_ordered_at, SUM(f.total_invoiced) AS total_invoiced
  FROM sales_line_fact f FORCE INDEX (sales_line_fact_buyer_summary_idx)
  WHERE f.account_id = ? AND f.buyer_account_id IN (%s) AND f.sales_order_type_code = 'sales_order' AND f.is_priced = 1%s
  GROUP BY f.buyer_account_id, f.product_line_id
) g
JOIN product_line pl ON pl.id = g.product_line_id
WHERE LOWER(pl.name) NOT IN ('shipping', 'misc')
GROUP BY g.buyer_account_id`

// salesBuyerSummaryLineBudget is how many facts one summary query may read, keeping it under 25ms. A var so tests can split small buyers.
var salesBuyerSummaryLineBudget int64 = 3000

// salesBuyerSummaryMaxBuyers caps the buyers in one query whose line counts the rollups do not know.
const salesBuyerSummaryMaxBuyers = 20

// buyerSummaryRead is one summary query: a set of buyers, over one invoiced_at window (zero bounds are open).
type buyerSummaryRead struct {
	buyers   []string
	from, to time.Time
}

// planBuyerSummaryReads groups buyers into reads of about salesBuyerSummaryLineBudget facts each, splitting a
// buyer with more into windows of whole months. Lines per month come from the buyer rollups, an upper bound.
func planBuyerSummaryReads(buyerIDs []string, monthLines map[string]map[time.Time]int64) []buyerSummaryRead {
	var reads []buyerSummaryRead
	var group []string
	var groupLines int64
	flush := func() {
		if len(group) > 0 {
			reads = append(reads, buyerSummaryRead{buyers: group})
			group, groupLines = nil, 0
		}
	}
	for _, b := range buyerIDs {
		var total int64
		for _, n := range monthLines[b] {
			total += n
		}
		if total <= salesBuyerSummaryLineBudget {
			if groupLines+total > salesBuyerSummaryLineBudget || len(group) == salesBuyerSummaryMaxBuyers {
				flush()
			}
			group = append(group, b)
			groupLines += total
			continue
		}
		months := make([]time.Time, 0, len(monthLines[b]))
		for m := range monthLines[b] {
			months = append(months, m)
		}
		slices.SortFunc(months, func(x, y time.Time) int { return x.Compare(y) })
		var from time.Time
		var lines int64
		for i, m := range months {
			lines += monthLines[b][m]
			if i == len(months)-1 {
				break
			}
			if next := months[i+1]; lines+monthLines[b][next] > salesBuyerSummaryLineBudget {
				reads = append(reads, buyerSummaryRead{buyers: []string{b}, from: from, to: next})
				from, lines = next, 0
			}
		}
		reads = append(reads, buyerSummaryRead{buyers: []string{b}, from: from})
	}
	flush()
	return reads
}

// buyerMonthLines is each buyer's facts per UTC month, as the buyer rollups count them.
func (r *salesFactRepoImpl) buyerMonthLines(ctx context.Context, accountID string, buyerIDs []string) (map[string]map[time.Time]int64, error) {
	out := map[string]map[time.Time]int64{}
	for start := 0; start < len(buyerIDs); start += salesFactWriteBatch {
		batch := buyerIDs[start:min(start+salesFactWriteBatch, len(buyerIDs))]
		rows, err := r.queries.DB().QueryContext(ctx, `SELECT dimension_id, bucket_start, SUM(line_count)
FROM sales_fact_rollup FORCE INDEX (sales_fact_rollup_dimension_id_idx)
WHERE account_id = ? AND dimension = 'buyer' AND product_line_key = '' AND dimension_id IN (`+placeholders(len(batch))+`) AND grain = 'month'
GROUP BY dimension_id, bucket_start`, append([]any{accountID}, stringsToAny(batch)...)...)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var (
				buyer string
				month time.Time
				n     int64
			)
			if err := rows.Scan(&buyer, &month, &n); err != nil {
				_ = rows.Close()
				return nil, err
			}
			if out[buyer] == nil {
				out[buyer] = map[time.Time]int64{}
			}
			out[buyer][month.UTC()] = n
		}
		_ = rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// buyerSummaryTotal accumulates one buyer's partial reads the way MIN and SUM over all of them would.
type buyerSummaryTotal struct {
	first *time.Time
	total *big.Rat
}

func (r *salesFactRepoImpl) readBuyerSummaries(ctx context.Context, accountID string, read buyerSummaryRead, into map[string]*buyerSummaryTotal) error {
	window := ""
	args := append([]any{accountID}, stringsToAny(read.buyers)...)
	if !read.from.IsZero() {
		window += " AND f.invoiced_at >= ?"
		args = append(args, read.from)
	}
	if !read.to.IsZero() {
		window += " AND f.invoiced_at < ?"
		args = append(args, read.to)
	}
	rows, err := r.queries.DB().QueryContext(ctx, fmt.Sprintf(salesBuyerSummaryPartial, placeholders(len(read.buyers)), window), args...)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var (
			buyer string
			first sql.NullTime
			total sql.NullString
		)
		if err := rows.Scan(&buyer, &first, &total); err != nil {
			return err
		}
		t := into[buyer]
		if t == nil {
			t = &buyerSummaryTotal{}
			into[buyer] = t
		}
		if first.Valid && (t.first == nil || first.Time.Before(*t.first)) {
			f := first.Time
			t.first = &f
		}
		if err := addDecimal(&t.total, total); err != nil {
			return err
		}
	}
	return rows.Err()
}

func (r *salesFactRepoImpl) MarkBuyers(ctx context.Context, buyers []domain.SalesBuyerKey) *apierror.APIError {
	ctx, span := salesFactRepoTracer.Start(ctx, "repository.sales_fact.mark_buyers")
	defer span.End()

	for start := 0; start < len(buyers); start += salesFactWriteBatch {
		batch := buyers[start:min(start+salesFactWriteBatch, len(buyers))]
		args := make([]any, 0, 3*len(batch))
		for _, b := range batch {
			args = append(args, string(domain.SalesFactScopeBuyerSummary), buyerMarkID(b), b.AccountID)
		}
		values := strings.TrimSuffix(strings.Repeat("(?, ?, ?, NOW(3)),", len(batch)), ",")
		_, err := r.queries.DB().ExecContext(ctx, `INSERT INTO sales_fact_dirty (scope_type, scope_id, account_id, marked_at) VALUES `+
			values+` ON DUPLICATE KEY UPDATE marked_at = VALUES(marked_at)`, args...)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}
	return nil
}

func (r *salesFactRepoImpl) ListBuyerDirty(ctx context.Context, limit int32) ([]domain.SalesBuyerDirtyMark, *apierror.APIError) {
	ctx, span := salesFactRepoTracer.Start(ctx, "repository.sales_fact.list_buyer_dirty")
	defer span.End()

	rows, err := r.queries.DB().QueryContext(ctx, `SELECT account_id, scope_id, marked_at FROM sales_fact_dirty
WHERE scope_type = ? ORDER BY marked_at LIMIT ?`, string(domain.SalesFactScopeBuyerSummary), limit)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	defer func() { _ = rows.Close() }()
	var marks []domain.SalesBuyerDirtyMark
	for rows.Next() {
		var (
			m       domain.SalesBuyerDirtyMark
			scopeID string
		)
		if err := rows.Scan(&m.Buyer.AccountID, &scopeID, &m.MarkedAt); err != nil {
			return nil, tracing.Trace(span, db.MapSQLError(err))
		}
		m.Buyer.BuyerAccountID = strings.TrimPrefix(scopeID, m.Buyer.AccountID+"/")
		marks = append(marks, m)
	}
	return marks, tracing.Trace(span, db.MapSQLError(rows.Err()))
}

func (r *salesFactRepoImpl) ClearBuyerDirty(ctx context.Context, marks []domain.SalesBuyerDirtyMark) *apierror.APIError {
	ctx, span := salesFactRepoTracer.Start(ctx, "repository.sales_fact.clear_buyer_dirty")
	defer span.End()

	keys := make([]dirtyMarkKey, len(marks))
	for i, m := range marks {
		keys[i] = dirtyMarkKey{scopeType: string(domain.SalesFactScopeBuyerSummary), scopeID: buyerMarkID(m.Buyer), markedAt: m.MarkedAt}
	}
	return tracing.Trace(span, db.MapSQLError(r.clearDirtyMarks(ctx, keys)))
}

func (r *salesFactRepoImpl) RebuildBuyerSummaries(ctx context.Context, accountID string, buyerIDs []string) *apierror.APIError {
	ctx, span := salesFactRepoTracer.Start(ctx, "repository.sales_fact.rebuild_buyer_summaries")
	defer span.End()
	if len(buyerIDs) == 0 {
		return nil
	}

	monthLines, err := r.buyerMonthLines(ctx, accountID, buyerIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	totals := map[string]*buyerSummaryTotal{}
	for _, read := range planBuyerSummaryReads(buyerIDs, monthLines) {
		if apiErr := db.MapSQLError(r.readBuyerSummaries(ctx, accountID, read, totals)); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	// A buyer whose qualifying orders have no issue date has no summary.
	now := time.Now().UTC()
	var upsertArgs []any
	var gone []string
	for _, b := range buyerIDs {
		t := totals[b]
		if t == nil || t.first == nil {
			gone = append(gone, b)
			continue
		}
		total := new(big.Rat)
		if t.total != nil {
			total = t.total
		}
		upsertArgs = append(upsertArgs, accountID, b, *t.first, total.FloatString(30), now)
	}
	for start := 0; start < len(upsertArgs); start += 5 * salesFactWriteBatch {
		batch := upsertArgs[start:min(start+5*salesFactWriteBatch, len(upsertArgs))]
		values := strings.TrimSuffix(strings.Repeat("(?, ?, ?, ?, ?),", len(batch)/5), ",")
		_, err := r.queries.DB().ExecContext(ctx, `INSERT INTO sales_buyer_summary (account_id, buyer_account_id, first_ordered_at, total_invoiced, refreshed_at) VALUES `+
			values+` ON DUPLICATE KEY UPDATE first_ordered_at = VALUES(first_ordered_at), total_invoiced = VALUES(total_invoiced), refreshed_at = VALUES(refreshed_at)`, batch...)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}
	keys := make([]domain.SalesBuyerKey, len(gone))
	for i, b := range gone {
		keys[i] = domain.SalesBuyerKey{AccountID: accountID, BuyerAccountID: b}
	}
	return tracing.Trace(span, r.DeleteBuyerSummaries(ctx, keys))
}

func (r *salesFactRepoImpl) NextBuyers(ctx context.Context, after domain.SalesBuyerKey, limit int32) ([]domain.SalesBuyerKey, *apierror.APIError) {
	ctx, span := salesFactRepoTracer.Start(ctx, "repository.sales_fact.next_buyers")
	defer span.End()

	// The leading account_id >= bound keeps it a range read of sales_line_fact_buyer_idx.
	rows, err := r.queries.DB().QueryContext(ctx, `SELECT account_id, buyer_account_id FROM sales_line_fact
WHERE account_id >= ? AND (account_id > ? OR buyer_account_id > ?)
GROUP BY account_id, buyer_account_id
ORDER BY account_id, buyer_account_id
LIMIT ?`, after.AccountID, after.AccountID, after.BuyerAccountID, limit)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	defer func() { _ = rows.Close() }()
	var out []domain.SalesBuyerKey
	for rows.Next() {
		var k domain.SalesBuyerKey
		if err := rows.Scan(&k.AccountID, &k.BuyerAccountID); err != nil {
			return nil, tracing.Trace(span, db.MapSQLError(err))
		}
		out = append(out, k)
	}
	return out, tracing.Trace(span, db.MapSQLError(rows.Err()))
}

func (r *salesFactRepoImpl) DeleteBuyerSummaries(ctx context.Context, keys []domain.SalesBuyerKey) *apierror.APIError {
	ctx, span := salesFactRepoTracer.Start(ctx, "repository.sales_fact.delete_buyer_summaries")
	defer span.End()

	for start := 0; start < len(keys); start += salesFactWriteBatch {
		batch := keys[start:min(start+salesFactWriteBatch, len(keys))]
		args := make([]any, 0, 2*len(batch))
		for _, k := range batch {
			args = append(args, k.AccountID, k.BuyerAccountID)
		}
		tuples := strings.TrimSuffix(strings.Repeat("(?, ?),", len(batch)), ",")
		_, err := r.queries.DB().ExecContext(ctx, `DELETE FROM sales_buyer_summary WHERE (account_id, buyer_account_id) IN (`+tuples+`)`, args...)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}
	return nil
}

func (r *salesFactRepoImpl) ListBuyerSummaryRefreshes(ctx context.Context, after domain.SalesBuyerKey, through *domain.SalesBuyerKey) (map[domain.SalesBuyerKey]time.Time, *apierror.APIError) {
	ctx, span := salesFactRepoTracer.Start(ctx, "repository.sales_fact.list_buyer_summary_refreshes")
	defer span.End()

	query := `SELECT account_id, buyer_account_id, refreshed_at FROM sales_buyer_summary
WHERE account_id >= ? AND (account_id > ? OR buyer_account_id > ?)`
	args := []any{after.AccountID, after.AccountID, after.BuyerAccountID}
	if through != nil {
		query += ` AND account_id <= ? AND (account_id < ? OR buyer_account_id <= ?)`
		args = append(args, through.AccountID, through.AccountID, through.BuyerAccountID)
	}
	rows, err := r.queries.DB().QueryContext(ctx, query, args...)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	defer func() { _ = rows.Close() }()
	out := map[domain.SalesBuyerKey]time.Time{}
	for rows.Next() {
		var (
			k  domain.SalesBuyerKey
			at time.Time
		)
		if err := rows.Scan(&k.AccountID, &k.BuyerAccountID, &at); err != nil {
			return nil, tracing.Trace(span, db.MapSQLError(err))
		}
		out[k] = at
	}
	return out, tracing.Trace(span, db.MapSQLError(rows.Err()))
}

func (r *salesFactRepoImpl) LatestBuyerFactRefreshes(ctx context.Context, accountID string, buyerIDs []string) (map[string]time.Time, *apierror.APIError) {
	ctx, span := salesFactRepoTracer.Start(ctx, "repository.sales_fact.latest_buyer_fact_refreshes")
	defer span.End()

	out := map[string]time.Time{}
	for start := 0; start < len(buyerIDs); start += salesFactWriteBatch {
		batch := buyerIDs[start:min(start+salesFactWriteBatch, len(buyerIDs))]
		rows, err := r.queries.DB().QueryContext(ctx, `SELECT buyer_account_id, MAX(refreshed_at)
FROM sales_line_fact FORCE INDEX (sales_line_fact_buyer_refreshed_idx)
WHERE account_id = ? AND buyer_account_id IN (`+placeholders(len(batch))+`)
GROUP BY buyer_account_id`, append([]any{accountID}, stringsToAny(batch)...)...)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		for rows.Next() {
			var (
				buyer string
				at    time.Time
			)
			if err := rows.Scan(&buyer, &at); err != nil {
				_ = rows.Close()
				return nil, tracing.Trace(span, db.MapSQLError(err))
			}
			out[buyer] = at
		}
		_ = rows.Close()
		if apiErr := db.MapSQLError(rows.Err()); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}
	return out, nil
}

func (r *salesFactRepoImpl) GetBuyerSummarySync(ctx context.Context) (*domain.SalesBuyerSummarySync, *apierror.APIError) {
	ctx, span := salesFactRepoTracer.Start(ctx, "repository.sales_fact.get_buyer_summary_sync")
	defer span.End()

	row, found, err := r.getSalesSync(ctx, salesBuyerSummarySyncName)
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	if !found {
		return &domain.SalesBuyerSummarySync{}, nil
	}
	sync := &domain.SalesBuyerSummarySync{
		FactsSince: nullTimePtr(row.factsSince), PassStartedAt: nullTimePtr(row.passStartedAt), LastCompletedAt: nullTimePtr(row.lastCompletedAt),
	}
	if row.cursorAccountID.Valid && row.cursorBuyerAccountID.Valid {
		sync.Cursor = &domain.SalesBuyerKey{AccountID: row.cursorAccountID.String, BuyerAccountID: row.cursorBuyerAccountID.String}
	}
	return sync, nil
}

func (r *salesFactRepoImpl) SaveBuyerSummarySync(ctx context.Context, sync domain.SalesBuyerSummarySync) *apierror.APIError {
	ctx, span := salesFactRepoTracer.Start(ctx, "repository.sales_fact.save_buyer_summary_sync")
	defer span.End()

	row := salesSyncRow{factsSince: toNullTime(sync.FactsSince), passStartedAt: toNullTime(sync.PassStartedAt), lastCompletedAt: toNullTime(sync.LastCompletedAt)}
	if sync.Cursor != nil {
		row.cursorAccountID = sql.NullString{String: sync.Cursor.AccountID, Valid: true}
		row.cursorBuyerAccountID = sql.NullString{String: sync.Cursor.BuyerAccountID, Valid: true}
	}
	return tracing.Trace(span, db.MapSQLError(r.saveSalesSync(ctx, salesBuyerSummarySyncName, row)))
}
