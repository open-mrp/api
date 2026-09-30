package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var salesFactRepoTracer = tracing.GetTracer("core-service.infrastructure.repository.sales_fact")

const salesFactSyncName = "reconcile"

// salesFactUpsertBatch bounds one multi-row upsert: 500 rows x 16 columns stays far below MySQL's 65,535 placeholder limit.
const salesFactUpsertBatch = 500

type salesFactRepoImpl struct {
	queries *sqlc.Queries
}

func NewSalesFactRepo(queries *sqlc.Queries) domain.SalesFactRepo {
	return &salesFactRepoImpl{queries: queries}
}

func (r *salesFactRepoImpl) ComputeFacts(ctx context.Context, invoiceIDs []string) ([]domain.SalesLineFact, *apierror.APIError) {
	ctx, span := salesFactRepoTracer.Start(ctx, "repository.sales_fact.compute_facts")
	defer span.End()
	if len(invoiceIDs) == 0 {
		return nil, nil
	}

	rows, err := r.queries.SelectSalesFactSource(ctx, invoiceIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	out := make([]domain.SalesLineFact, len(rows))
	for i, row := range rows {
		out[i] = domain.SalesLineFact{
			AccountID:          row.AccountID,
			InvoicedAt:         row.InvoicedAt,
			InvoiceLineID:      row.InvoiceLineID,
			InvoiceID:          row.InvoiceID,
			SalesOrderID:       row.SalesOrderID,
			SalesOrderTypeCode: row.SalesOrderTypeCode,
			BuyerAccountID:     row.BuyerAccountID,
			SalesRepID:         nullStringPtr(row.SalesRepID),
			OrderDiscountID:    nullStringPtr(row.OrderDiscountID),
			ProductID:          row.ProductID,
			ItemID:             row.ItemID,
			ProductLineID:      row.ProductLineID.String,
			QuantityBase:       decimalStringPtr(row.QuantityBase),
			TotalInvoiced:      decimalStringPtr(row.TotalInvoiced),
			TotalCost:          decimalStringPtr(row.TotalCost),
			OrderedAt:          nullTimePtr(row.OrderedAt),
			IsPriced:           row.IsPriced == 1,
		}
	}
	return out, nil
}

func (r *salesFactRepoImpl) GetFacts(ctx context.Context, invoiceIDs []string) ([]domain.SalesLineFact, *apierror.APIError) {
	ctx, span := salesFactRepoTracer.Start(ctx, "repository.sales_fact.get_facts")
	defer span.End()
	if len(invoiceIDs) == 0 {
		return nil, nil
	}

	rows, err := r.queries.SelectSalesFactsByInvoiceIDs(ctx, invoiceIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	out := make([]domain.SalesLineFact, len(rows))
	for i, row := range rows {
		out[i] = domain.SalesLineFact{
			AccountID:          row.AccountID,
			InvoicedAt:         row.InvoicedAt,
			InvoiceLineID:      row.InvoiceLineID,
			InvoiceID:          row.InvoiceID,
			SalesOrderID:       row.SalesOrderID,
			SalesOrderTypeCode: row.SalesOrderTypeCode,
			BuyerAccountID:     row.BuyerAccountID,
			SalesRepID:         nullStringPtr(row.SalesRepID),
			OrderDiscountID:    nullStringPtr(row.OrderDiscountID),
			ProductID:          row.ProductID,
			ItemID:             row.ItemID,
			ProductLineID:      row.ProductLineID,
			QuantityBase:       nullStringPtr(row.QuantityBase),
			TotalInvoiced:      nullStringPtr(row.TotalInvoiced),
			TotalCost:          nullStringPtr(row.TotalCost),
			OrderedAt:          nullTimePtr(row.OrderedAt),
			IsPriced:           row.IsPriced,
		}
	}
	return out, nil
}

// UpsertFacts writes facts in multi-row statements; a fact whose invoice_line_id exists is overwritten in place, including its clustered-key columns.
func (r *salesFactRepoImpl) UpsertFacts(ctx context.Context, facts []domain.SalesLineFact) *apierror.APIError {
	ctx, span := salesFactRepoTracer.Start(ctx, "repository.sales_fact.upsert_facts")
	defer span.End()

	now := time.Now().UTC()
	for start := 0; start < len(facts); start += salesFactUpsertBatch {
		batch := facts[start:min(start+salesFactUpsertBatch, len(facts))]
		var sb strings.Builder
		sb.WriteString(`INSERT INTO sales_line_fact (account_id, invoiced_at, invoice_line_id, invoice_id, sales_order_id,
sales_order_type_code, buyer_account_id, sales_rep_id, order_discount_id, product_id, item_id, product_line_id,
quantity_base, total_invoiced, total_cost, ordered_at, is_priced, refreshed_at) VALUES `)
		args := make([]any, 0, len(batch)*18)
		for i, f := range batch {
			if i > 0 {
				sb.WriteString(",")
			}
			sb.WriteString("(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)")
			args = append(args, f.AccountID, f.InvoicedAt, f.InvoiceLineID, f.InvoiceID, f.SalesOrderID,
				f.SalesOrderTypeCode, f.BuyerAccountID, f.SalesRepID, f.OrderDiscountID, f.ProductID, f.ItemID, f.ProductLineID,
				f.QuantityBase, f.TotalInvoiced, f.TotalCost, f.OrderedAt, f.IsPriced, now)
		}
		sb.WriteString(` ON DUPLICATE KEY UPDATE account_id = VALUES(account_id), invoiced_at = VALUES(invoiced_at),
invoice_id = VALUES(invoice_id), sales_order_id = VALUES(sales_order_id), sales_order_type_code = VALUES(sales_order_type_code),
buyer_account_id = VALUES(buyer_account_id), sales_rep_id = VALUES(sales_rep_id), order_discount_id = VALUES(order_discount_id),
product_id = VALUES(product_id), item_id = VALUES(item_id), product_line_id = VALUES(product_line_id),
quantity_base = VALUES(quantity_base), total_invoiced = VALUES(total_invoiced), total_cost = VALUES(total_cost),
ordered_at = VALUES(ordered_at), is_priced = VALUES(is_priced), refreshed_at = VALUES(refreshed_at)`)
		if _, err := r.queries.DB().ExecContext(ctx, sb.String(), args...); err != nil {
			return tracing.Trace(span, db.MapSQLError(err))
		}
	}
	return nil
}

func (r *salesFactRepoImpl) DeleteFacts(ctx context.Context, invoiceLineIDs []string) *apierror.APIError {
	ctx, span := salesFactRepoTracer.Start(ctx, "repository.sales_fact.delete_facts")
	defer span.End()
	if len(invoiceLineIDs) == 0 {
		return nil
	}
	if apiErr := db.MapSQLError(r.queries.DeleteSalesFactsByLineIDs(ctx, invoiceLineIDs)); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *salesFactRepoImpl) ListInvoicesAfter(ctx context.Context, after *domain.SalesFactInvoiceCursor, limit int32) ([]domain.SalesFactInvoiceCursor, *apierror.APIError) {
	ctx, span := salesFactRepoTracer.Start(ctx, "repository.sales_fact.list_invoices_after")
	defer span.End()

	// The zero time.Time interpolates as 0000-00-00, which strict mode rejects; start before any invoice instead.
	params := sqlc.ListInvoicesForFactSweepParams{AfterCreatedAt: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC), AfterID: "", Limit: limit}
	if after != nil {
		params.AfterCreatedAt = after.CreatedAt
		params.AfterID = after.InvoiceID
	}
	rows, err := r.queries.ListInvoicesForFactSweep(ctx, params)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	out := make([]domain.SalesFactInvoiceCursor, len(rows))
	for i, row := range rows {
		out[i] = domain.SalesFactInvoiceCursor{InvoiceID: row.ID, CreatedAt: row.CreatedAt}
	}
	return out, nil
}

func (r *salesFactRepoImpl) ListInvoiceIDsCreatedSince(ctx context.Context, since time.Time, limit int32) ([]string, *apierror.APIError) {
	ctx, span := salesFactRepoTracer.Start(ctx, "repository.sales_fact.list_invoice_ids_created_since")
	defer span.End()

	ids, err := r.queries.ListInvoiceIDsCreatedSince(ctx, sqlc.ListInvoiceIDsCreatedSinceParams{CreatedAt: since, Limit: limit})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return ids, nil
}

func (r *salesFactRepoImpl) ListFactInvoiceIDsAfter(ctx context.Context, afterInvoiceID string, limit int32) ([]string, *apierror.APIError) {
	ctx, span := salesFactRepoTracer.Start(ctx, "repository.sales_fact.list_fact_invoice_ids_after")
	defer span.End()

	ids, err := r.queries.ListSalesFactInvoiceIDsAfter(ctx, sqlc.ListSalesFactInvoiceIDsAfterParams{InvoiceID: afterInvoiceID, Limit: limit})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return ids, nil
}

func (r *salesFactRepoImpl) FilterExistingInvoiceIDs(ctx context.Context, invoiceIDs []string) ([]string, *apierror.APIError) {
	ctx, span := salesFactRepoTracer.Start(ctx, "repository.sales_fact.filter_existing_invoice_ids")
	defer span.End()
	if len(invoiceIDs) == 0 {
		return nil, nil
	}

	ids, err := r.queries.ListExistingInvoiceIDs(ctx, invoiceIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return ids, nil
}

func (r *salesFactRepoImpl) ResolveInvoiceIDs(ctx context.Context, accountID string, scope domain.SalesFactScope, scopeIDs []string) ([]string, *apierror.APIError) {
	ctx, span := salesFactRepoTracer.Start(ctx, "repository.sales_fact.resolve_invoice_ids")
	defer span.End()
	if len(scopeIDs) == 0 {
		return nil, nil
	}

	var (
		ids []string
		err error
	)
	in := placeholders(len(scopeIDs))
	args := stringArgs(scopeIDs)
	switch scope {
	case domain.SalesFactScopeInvoice:
		return scopeIDs, nil
	case domain.SalesFactScopeSalesOrder:
		ids, err = r.queries.ListInvoiceIDsBySalesOrders(ctx, scopeIDs)
	case domain.SalesFactScopeSalesOrderLine:
		ids, err = r.queries.ListInvoiceIDsBySalesOrderLines(ctx, scopeIDs)
	case domain.SalesFactScopeProduct:
		ids, err = r.queries.ListInvoiceIDsByProducts(ctx, toNullStringSlice(scopeIDs))
	case domain.SalesFactScopeQuantity:
		ids, err = r.selectIDs(ctx, `SELECT DISTINCT invoice_id FROM invoice_line WHERE quantity_id IN (`+in+`)`, args...)
	case domain.SalesFactScopeRate:
		ids, err = r.selectIDs(ctx, `SELECT il.invoice_id FROM sales_order_line sol JOIN invoice_line il ON il.sales_order_line_id = sol.id WHERE sol.unit_price_id IN (`+in+`)
UNION SELECT il.invoice_id FROM sales_order_line sol JOIN invoice_line il ON il.sales_order_line_id = sol.id WHERE sol.unit_cost_id IN (`+in+`)`, append(args, args...)...)
	case domain.SalesFactScopeItem:
		ids, err = r.selectIDs(ctx, `SELECT DISTINCT il.invoice_id FROM product p
JOIN sales_order_line sol ON sol.product_id = p.id
JOIN invoice_line il ON il.sales_order_line_id = sol.id
WHERE p.item_id IN (`+in+`)`, args...)
	case domain.SalesFactScopeBuyer:
		// The buyer's orders now belong to another customer, so the invoices are found by the facts
		// that still name it.
		ids, err = r.selectIDs(ctx, `SELECT DISTINCT invoice_id FROM sales_line_fact WHERE account_id = ? AND buyer_account_id IN (`+in+`)`,
			append([]any{accountID}, args...)...)
	default:
		return nil, tracing.Trace(span, apierror.NewInternalError(nil, fmt.Sprintf("Unknown sales fact scope %q.", scope)))
	}
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return ids, nil
}

func (r *salesFactRepoImpl) selectIDs(ctx context.Context, query string, args ...any) ([]string, error) {
	rows, err := r.queries.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *salesFactRepoImpl) MarkInvoicesDirty(ctx context.Context, accountID string, invoiceIDs []string) *apierror.APIError {
	ctx, span := salesFactRepoTracer.Start(ctx, "repository.sales_fact.mark_invoices_dirty")
	defer span.End()

	for start := 0; start < len(invoiceIDs); start += 500 {
		batch := invoiceIDs[start:min(start+500, len(invoiceIDs))]
		values := strings.Repeat("(?, ?, ?, NOW(3)),", len(batch))
		args := make([]any, 0, 3*len(batch))
		for _, id := range batch {
			args = append(args, string(domain.SalesFactScopeInvoice), id, accountID)
		}
		_, err := r.queries.DB().ExecContext(ctx, `INSERT INTO sales_fact_dirty (scope_type, scope_id, account_id, marked_at) VALUES `+
			strings.TrimSuffix(values, ",")+` ON DUPLICATE KEY UPDATE marked_at = VALUES(marked_at)`, args...)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}
	return nil
}

func (r *salesFactRepoImpl) RestartReconcile(ctx context.Context) *apierror.APIError {
	ctx, span := salesFactRepoTracer.Start(ctx, "repository.sales_fact.restart_reconcile")
	defer span.End()

	row, found, err := r.getSalesSync(ctx, salesFactSyncName)
	if err != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}
	// Until the backfill has run there is nothing to restart: the backfill is the first pass.
	if !found || !row.lastCompletedAt.Valid {
		return nil
	}
	row.cursorCreatedAt = sql.NullTime{Time: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true}
	row.cursorInvoiceID = sql.NullString{String: "", Valid: true}
	row.passStartedAt = sql.NullTime{Time: time.Now().UTC(), Valid: true}
	return tracing.Trace(span, db.MapSQLError(r.saveSalesSync(ctx, salesFactSyncName, row)))
}

// rollupMarkID is a rollup day mark's scope_id in sales_fact_dirty: the account and the UTC day.
func rollupMarkID(d domain.SalesRollupDay) string {
	return d.AccountID + "/" + truncateUTCDay(d.Day).Format(time.DateOnly)
}

func (r *salesFactRepoImpl) MarkRollupDays(ctx context.Context, days []domain.SalesRollupDay) *apierror.APIError {
	ctx, span := salesFactRepoTracer.Start(ctx, "repository.sales_fact.mark_rollup_days")
	defer span.End()

	for start := 0; start < len(days); start += 500 {
		batch := days[start:min(start+500, len(days))]
		args := make([]any, 0, 3*len(batch))
		for _, d := range batch {
			args = append(args, string(domain.SalesFactScopeRollupDay), rollupMarkID(d), d.AccountID)
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

func (r *salesFactRepoImpl) ListRollupDirty(ctx context.Context, limit int32) ([]domain.SalesRollupDirtyMark, *apierror.APIError) {
	ctx, span := salesFactRepoTracer.Start(ctx, "repository.sales_fact.list_rollup_dirty")
	defer span.End()

	if err := r.adoptLegacyRollupMarks(ctx, limit); err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	rows, err := r.queries.DB().QueryContext(ctx, `SELECT account_id, scope_id, marked_at FROM sales_fact_dirty
WHERE scope_type = ? ORDER BY marked_at LIMIT ?`, string(domain.SalesFactScopeRollupDay), limit)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	defer func() { _ = rows.Close() }()
	var marks []domain.SalesRollupDirtyMark
	for rows.Next() {
		var (
			m       domain.SalesRollupDirtyMark
			scopeID string
		)
		if err := rows.Scan(&m.Day.AccountID, &scopeID, &m.MarkedAt); err != nil {
			return nil, tracing.Trace(span, db.MapSQLError(err))
		}
		day, err := time.ParseInLocation(time.DateOnly, strings.TrimPrefix(scopeID, m.Day.AccountID+"/"), time.UTC)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Unreadable rollup day mark "+scopeID+"."))
		}
		m.Day.Day = day
		marks = append(marks, m)
	}
	return marks, tracing.Trace(span, db.MapSQLError(rows.Err()))
}

// adoptLegacyRollupMarks moves marks from sales_rollup_dirty, where the rollup day marks were kept before
// they joined sales_fact_dirty, so pods on the previous release that mark days during a deploy are not lost.
// A mark moves only if it was not re-marked after it was read, like any clear.
// TODO(sales-sync): drop with sales_rollup_dirty once every deployment has moved.
func (r *salesFactRepoImpl) adoptLegacyRollupMarks(ctx context.Context, limit int32) error {
	rows, err := r.queries.DB().QueryContext(ctx, `SELECT account_id, day, marked_at FROM sales_rollup_dirty ORDER BY marked_at LIMIT ?`, limit)
	if err != nil {
		return err
	}
	type legacyMark struct {
		accountID string
		day       time.Time
		markedAt  time.Time
	}
	var legacy []legacyMark
	for rows.Next() {
		var m legacyMark
		if err := rows.Scan(&m.accountID, &m.day, &m.markedAt); err != nil {
			_ = rows.Close()
			return err
		}
		legacy = append(legacy, m)
	}
	_ = rows.Close()
	if err := rows.Err(); err != nil || len(legacy) == 0 {
		return err
	}
	args := make([]any, 0, 4*len(legacy))
	for _, m := range legacy {
		args = append(args, string(domain.SalesFactScopeRollupDay), rollupMarkID(domain.SalesRollupDay{AccountID: m.accountID, Day: m.day}), m.accountID, m.markedAt)
	}
	values := strings.TrimSuffix(strings.Repeat("(?, ?, ?, ?),", len(legacy)), ",")
	if _, err := r.queries.DB().ExecContext(ctx, `INSERT INTO sales_fact_dirty (scope_type, scope_id, account_id, marked_at) VALUES `+
		values+` ON DUPLICATE KEY UPDATE marked_at = GREATEST(marked_at, VALUES(marked_at))`, args...); err != nil {
		return err
	}
	for _, m := range legacy {
		if _, err := r.queries.DB().ExecContext(ctx, `DELETE FROM sales_rollup_dirty WHERE account_id = ? AND day = ? AND marked_at = ?`,
			m.accountID, m.day, m.markedAt); err != nil {
			return err
		}
	}
	return nil
}

func (r *salesFactRepoImpl) ClearRollupDirty(ctx context.Context, mark domain.SalesRollupDirtyMark) *apierror.APIError {
	ctx, span := salesFactRepoTracer.Start(ctx, "repository.sales_fact.clear_rollup_dirty")
	defer span.End()

	_, err := r.queries.DB().ExecContext(ctx, `DELETE FROM sales_fact_dirty WHERE scope_type = ? AND scope_id = ? AND marked_at = ?`,
		string(domain.SalesFactScopeRollupDay), rollupMarkID(mark.Day), mark.MarkedAt)
	return tracing.Trace(span, db.MapSQLError(err))
}

func (r *salesFactRepoImpl) MarkDirty(ctx context.Context, scope domain.SalesFactScope, scopeID, accountID string) *apierror.APIError {
	ctx, span := salesFactRepoTracer.Start(ctx, "repository.sales_fact.mark_dirty")
	defer span.End()

	err := r.queries.MarkSalesFactDirty(ctx, sqlc.MarkSalesFactDirtyParams{ScopeType: string(scope), ScopeID: scopeID, AccountID: accountID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *salesFactRepoImpl) ListDirty(ctx context.Context, limit int32) ([]domain.SalesFactDirtyMark, *apierror.APIError) {
	ctx, span := salesFactRepoTracer.Start(ctx, "repository.sales_fact.list_dirty")
	defer span.End()

	rows, err := r.queries.ListSalesFactDirty(ctx, limit)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	out := make([]domain.SalesFactDirtyMark, len(rows))
	for i, row := range rows {
		out[i] = domain.SalesFactDirtyMark{
			ScopeType: domain.SalesFactScope(row.ScopeType),
			ScopeID:   row.ScopeID,
			AccountID: row.AccountID,
			MarkedAt:  row.MarkedAt,
		}
	}
	return out, nil
}

func (r *salesFactRepoImpl) ClearDirty(ctx context.Context, mark domain.SalesFactDirtyMark) *apierror.APIError {
	ctx, span := salesFactRepoTracer.Start(ctx, "repository.sales_fact.clear_dirty")
	defer span.End()

	err := r.queries.ClearSalesFactDirty(ctx, sqlc.ClearSalesFactDirtyParams{
		ScopeType: string(mark.ScopeType),
		ScopeID:   mark.ScopeID,
		MarkedAt:  mark.MarkedAt,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *salesFactRepoImpl) GetSync(ctx context.Context) (*domain.SalesFactSync, *apierror.APIError) {
	ctx, span := salesFactRepoTracer.Start(ctx, "repository.sales_fact.get_sync")
	defer span.End()

	row, found, err := r.getSalesSync(ctx, salesFactSyncName)
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	if !found {
		return &domain.SalesFactSync{}, nil
	}
	sync := &domain.SalesFactSync{PassStartedAt: nullTimePtr(row.passStartedAt), LastCompletedAt: nullTimePtr(row.lastCompletedAt)}
	if row.cursorCreatedAt.Valid && row.cursorInvoiceID.Valid {
		sync.Cursor = &domain.SalesFactInvoiceCursor{InvoiceID: row.cursorInvoiceID.String, CreatedAt: row.cursorCreatedAt.Time}
	}
	return sync, nil
}

func (r *salesFactRepoImpl) SaveSync(ctx context.Context, sync domain.SalesFactSync) *apierror.APIError {
	ctx, span := salesFactRepoTracer.Start(ctx, "repository.sales_fact.save_sync")
	defer span.End()

	row := salesSyncRow{passStartedAt: toNullTime(sync.PassStartedAt), lastCompletedAt: toNullTime(sync.LastCompletedAt)}
	if sync.Cursor != nil {
		row.cursorCreatedAt = sql.NullTime{Time: sync.Cursor.CreatedAt, Valid: true}
		row.cursorInvoiceID = sql.NullString{String: sync.Cursor.InvoiceID, Valid: true}
	}
	return tracing.Trace(span, db.MapSQLError(r.saveSalesSync(ctx, salesFactSyncName, row)))
}

// decimalStringPtr reads a DECIMAL expression that sqlc could only type as any: the driver returns it as text, or nil for NULL.
func decimalStringPtr(v any) *string {
	switch t := v.(type) {
	case nil:
		return nil
	case []byte:
		s := string(t)
		return &s
	case string:
		return &t
	default:
		s := fmt.Sprint(t)
		return &s
	}
}
