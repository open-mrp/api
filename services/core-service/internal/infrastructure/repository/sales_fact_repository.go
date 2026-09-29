package repository

import (
	"context"
	"database/sql"
	"errors"
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
quantity_base, total_invoiced, total_cost, refreshed_at) VALUES `)
		args := make([]any, 0, len(batch)*16)
		for i, f := range batch {
			if i > 0 {
				sb.WriteString(",")
			}
			sb.WriteString("(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)")
			args = append(args, f.AccountID, f.InvoicedAt, f.InvoiceLineID, f.InvoiceID, f.SalesOrderID,
				f.SalesOrderTypeCode, f.BuyerAccountID, f.SalesRepID, f.OrderDiscountID, f.ProductID, f.ItemID, f.ProductLineID,
				f.QuantityBase, f.TotalInvoiced, f.TotalCost, now)
		}
		sb.WriteString(` ON DUPLICATE KEY UPDATE account_id = VALUES(account_id), invoiced_at = VALUES(invoiced_at),
invoice_id = VALUES(invoice_id), sales_order_id = VALUES(sales_order_id), sales_order_type_code = VALUES(sales_order_type_code),
buyer_account_id = VALUES(buyer_account_id), sales_rep_id = VALUES(sales_rep_id), order_discount_id = VALUES(order_discount_id),
product_id = VALUES(product_id), item_id = VALUES(item_id), product_line_id = VALUES(product_line_id),
quantity_base = VALUES(quantity_base), total_invoiced = VALUES(total_invoiced), total_cost = VALUES(total_cost),
refreshed_at = VALUES(refreshed_at)`)
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

func (r *salesFactRepoImpl) ResolveInvoiceIDs(ctx context.Context, scope domain.SalesFactScope, scopeIDs []string) ([]string, *apierror.APIError) {
	ctx, span := salesFactRepoTracer.Start(ctx, "repository.sales_fact.resolve_invoice_ids")
	defer span.End()
	if len(scopeIDs) == 0 {
		return nil, nil
	}

	var (
		ids []string
		err error
	)
	switch scope {
	case domain.SalesFactScopeInvoice:
		return scopeIDs, nil
	case domain.SalesFactScopeSalesOrder:
		ids, err = r.queries.ListInvoiceIDsBySalesOrders(ctx, scopeIDs)
	case domain.SalesFactScopeSalesOrderLine:
		ids, err = r.queries.ListInvoiceIDsBySalesOrderLines(ctx, scopeIDs)
	case domain.SalesFactScopeProduct:
		ids, err = r.queries.ListInvoiceIDsByProducts(ctx, toNullStringSlice(scopeIDs))
	default:
		return nil, tracing.Trace(span, apierror.NewInternalError(nil, fmt.Sprintf("Unknown sales fact scope %q.", scope)))
	}
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return ids, nil
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

	row, err := r.queries.GetSalesFactSync(ctx, salesFactSyncName)
	if errors.Is(err, sql.ErrNoRows) {
		return &domain.SalesFactSync{}, nil
	}
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	sync := &domain.SalesFactSync{
		PassStartedAt:   nullTimePtr(row.PassStartedAt),
		LastCompletedAt: nullTimePtr(row.LastCompletedAt),
	}
	if row.CursorCreatedAt.Valid && row.CursorInvoiceID.Valid {
		sync.Cursor = &domain.SalesFactInvoiceCursor{InvoiceID: row.CursorInvoiceID.String, CreatedAt: row.CursorCreatedAt.Time}
	}
	return sync, nil
}

func (r *salesFactRepoImpl) SaveSync(ctx context.Context, sync domain.SalesFactSync) *apierror.APIError {
	ctx, span := salesFactRepoTracer.Start(ctx, "repository.sales_fact.save_sync")
	defer span.End()

	params := sqlc.UpsertSalesFactSyncParams{
		Name:            salesFactSyncName,
		PassStartedAt:   toNullTime(sync.PassStartedAt),
		LastCompletedAt: toNullTime(sync.LastCompletedAt),
	}
	if sync.Cursor != nil {
		params.CursorCreatedAt = sql.NullTime{Time: sync.Cursor.CreatedAt, Valid: true}
		params.CursorInvoiceID = sql.NullString{String: sync.Cursor.InvoiceID, Valid: true}
	}
	if apiErr := db.MapSQLError(r.queries.UpsertSalesFactSync(ctx, params)); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
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
