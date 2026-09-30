package repository

import (
	"context"
	"database/sql"
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

// salesBuyerSummarySource is one account's qualifying sales per buyer, as the legacy new-customers report
// counted them: sales orders only, lines priced above zero, outside the shipping and misc product lines.
// first_ordered_at is the earliest of those lines' order dates; a buyer whose qualifying orders have no
// issue date has no summary. The %s is the buyer_account_id IN list.
const salesBuyerSummarySource = `SELECT f.buyer_account_id, MIN(f.ordered_at), CAST(COALESCE(SUM(f.total_invoiced), 0) AS DECIMAL(65,30))
FROM sales_line_fact f
JOIN product_line pl ON pl.id = f.product_line_id
WHERE f.account_id = ? AND f.buyer_account_id IN (%s)
  AND f.sales_order_type_code = 'sales_order' AND f.is_priced = 1
  AND LOWER(pl.name) NOT IN ('shipping', 'misc')
GROUP BY f.buyer_account_id
HAVING MIN(f.ordered_at) IS NOT NULL`

func (r *salesFactRepoImpl) MarkBuyers(ctx context.Context, buyers []domain.SalesBuyerKey) *apierror.APIError {
	ctx, span := salesFactRepoTracer.Start(ctx, "repository.sales_fact.mark_buyers")
	defer span.End()

	for start := 0; start < len(buyers); start += 500 {
		batch := buyers[start:min(start+500, len(buyers))]
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

func (r *salesFactRepoImpl) ClearBuyerDirty(ctx context.Context, mark domain.SalesBuyerDirtyMark) *apierror.APIError {
	ctx, span := salesFactRepoTracer.Start(ctx, "repository.sales_fact.clear_buyer_dirty")
	defer span.End()

	_, err := r.queries.DB().ExecContext(ctx, `DELETE FROM sales_fact_dirty WHERE scope_type = ? AND scope_id = ? AND marked_at = ?`,
		string(domain.SalesFactScopeBuyerSummary), buyerMarkID(mark.Buyer), mark.MarkedAt)
	return tracing.Trace(span, db.MapSQLError(err))
}

func (r *salesFactRepoImpl) RebuildBuyerSummaries(ctx context.Context, accountID string, buyerIDs []string) *apierror.APIError {
	ctx, span := salesFactRepoTracer.Start(ctx, "repository.sales_fact.rebuild_buyer_summaries")
	defer span.End()

	for start := 0; start < len(buyerIDs); start += 200 {
		batch := buyerIDs[start:min(start+200, len(buyerIDs))]
		args := append([]any{accountID}, stringsToAny(batch)...)
		rows, err := r.queries.DB().QueryContext(ctx, strings.Replace(salesBuyerSummarySource, "%s", placeholders(len(batch)), 1), args...)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
		type summary struct {
			buyer   string
			first   time.Time
			invoice string
		}
		var found []summary
		for rows.Next() {
			var s summary
			if err := rows.Scan(&s.buyer, &s.first, &s.invoice); err != nil {
				_ = rows.Close()
				return tracing.Trace(span, db.MapSQLError(err))
			}
			found = append(found, s)
		}
		_ = rows.Close()
		if apiErr := db.MapSQLError(rows.Err()); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}

		keep := make(map[string]struct{}, len(found))
		if len(found) > 0 {
			now := time.Now().UTC()
			values := strings.TrimSuffix(strings.Repeat("(?, ?, ?, ?, ?),", len(found)), ",")
			upsertArgs := make([]any, 0, 5*len(found))
			for _, s := range found {
				keep[s.buyer] = struct{}{}
				upsertArgs = append(upsertArgs, accountID, s.buyer, s.first, s.invoice, now)
			}
			_, err := r.queries.DB().ExecContext(ctx, `INSERT INTO sales_buyer_summary (account_id, buyer_account_id, first_ordered_at, total_invoiced, refreshed_at) VALUES `+
				values+` ON DUPLICATE KEY UPDATE first_ordered_at = VALUES(first_ordered_at), total_invoiced = VALUES(total_invoiced), refreshed_at = VALUES(refreshed_at)`, upsertArgs...)
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return tracing.Trace(span, apiErr)
			}
		}
		var gone []string
		for _, b := range batch {
			if _, ok := keep[b]; !ok {
				gone = append(gone, b)
			}
		}
		if len(gone) > 0 {
			_, err := r.queries.DB().ExecContext(ctx, `DELETE FROM sales_buyer_summary WHERE account_id = ? AND buyer_account_id IN (`+placeholders(len(gone))+`)`,
				append([]any{accountID}, stringsToAny(gone)...)...)
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return tracing.Trace(span, apiErr)
			}
		}
	}
	return nil
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

func (r *salesFactRepoImpl) DeleteBuyerSummariesRefreshedBefore(ctx context.Context, t time.Time) *apierror.APIError {
	ctx, span := salesFactRepoTracer.Start(ctx, "repository.sales_fact.delete_stale_buyer_summaries")
	defer span.End()

	_, err := r.queries.DB().ExecContext(ctx, `DELETE FROM sales_buyer_summary WHERE refreshed_at < ?`, t)
	return tracing.Trace(span, db.MapSQLError(err))
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
