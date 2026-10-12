package repository

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var deliveryFactRepoTracer = tracing.GetTracer("core-service.infrastructure.repository.delivery_fact")

// deliveryFactSyncName is the delivery reconcile pass's row in sales_sync.
const deliveryFactSyncName = "delivery_fact"

// deliveryFactWriteBatch bounds the rows one fact or mark statement writes.
const deliveryFactWriteBatch = 100

// deliveryFactIDBatch bounds the ids one lookup by IN-list sends.
const deliveryFactIDBatch = 500

type deliveryFactRepoImpl struct {
	queries *sqlc.Queries
}

func NewDeliveryFactRepo(queries *sqlc.Queries) domain.DeliveryFactRepo {
	return &deliveryFactRepoImpl{queries: queries}
}

// deliveryFactColumns are delivery_order_fact's columns in the order deliveryFactScan reads them.
const deliveryFactColumns = `owner_account_id, ship_by_date, sales_order_id, product_line_key, sales_order_number,
buyer_account_id, sales_rep_id, sales_order_status_code, issued_at, first_ship_at, completed_at,
lead_time_days, lead_time_source_code, quantity_ordered, quantity_packed`

// deliveryFactSource sums each measured order's sale lines per product line from the source tables. The
// filters and sums are ListDeliveryPerformanceOrders' own, so summing an order's rows over its product
// lines reproduces that query's per-order quantities exactly. The order's id is its primary key, so its
// other columns are grouped with it.
const deliveryFactSource = `SELECT so.owner_account_id, so.ship_by_date, so.id, COALESCE(p.product_line_id, '') AS product_line_key,
    so.number, so.buyer_account_id, so.sales_rep_id, so.sales_order_status_code, so.issued_at, so.first_ship_at,
    so.completed_at, so.lead_time_days, so.lead_time_source_code,
    CAST(COALESCE(SUM(q.value), 0) AS DECIMAL(65,30)),
    CAST(COALESCE(SUM(
        (SELECT COALESCE(SUM(plq.value), 0) FROM pick_line pl
            JOIN quantity plq ON plq.id = pl.quantity_id
            WHERE pl.sales_order_line_id = sol.id AND pl.packed_at IS NOT NULL)
    ), 0) AS DECIMAL(65,30))
FROM sales_order so
JOIN sales_order_line sol ON sol.sales_order_id = so.id
JOIN quantity q ON q.id = sol.quantity_id
JOIN product p ON p.id = sol.product_id
WHERE so.id IN (%s)
  AND so.sales_order_type_code = 'sales_order'
  AND so.sales_order_status_code IN ('issued', 'fulfilled')
  AND so.ship_by_date IS NOT NULL
  AND p.product_type_code = 'sale'
GROUP BY so.id, product_line_key`

func (r *deliveryFactRepoImpl) ComputeFacts(ctx context.Context, salesOrderIDs []string) ([]domain.DeliveryOrderFact, *apierror.APIError) {
	ctx, span := deliveryFactRepoTracer.Start(ctx, "repository.delivery_fact.compute_facts")
	defer span.End()
	if len(salesOrderIDs) == 0 {
		return nil, nil
	}
	query := strings.Replace(deliveryFactSource, "%s", placeholders(len(salesOrderIDs)), 1)
	facts, err := r.scanFacts(ctx, query, stringsToAny(salesOrderIDs)...)
	return facts, tracing.Trace(span, db.MapSQLError(err))
}

func (r *deliveryFactRepoImpl) GetFacts(ctx context.Context, salesOrderIDs []string) ([]domain.DeliveryOrderFact, *apierror.APIError) {
	ctx, span := deliveryFactRepoTracer.Start(ctx, "repository.delivery_fact.get_facts")
	defer span.End()
	if len(salesOrderIDs) == 0 {
		return nil, nil
	}
	facts, err := r.scanFacts(ctx, `SELECT `+deliveryFactColumns+` FROM delivery_order_fact FORCE INDEX (delivery_order_fact_order_idx)
WHERE sales_order_id IN (`+placeholders(len(salesOrderIDs))+`)`, stringsToAny(salesOrderIDs)...)
	return facts, tracing.Trace(span, db.MapSQLError(err))
}

func (r *deliveryFactRepoImpl) scanFacts(ctx context.Context, query string, args ...any) ([]domain.DeliveryOrderFact, error) {
	rows, err := r.queries.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []domain.DeliveryOrderFact
	for rows.Next() {
		var (
			f                            domain.DeliveryOrderFact
			rep, source                  sql.NullString
			issued, firstShip, completed sql.NullTime
			leadTime                     sql.NullInt32
			ordered, packed              string
		)
		if err := rows.Scan(&f.OwnerAccountID, &f.ShipByDate, &f.SalesOrderID, &f.ProductLineKey, &f.SalesOrderNumber,
			&f.BuyerAccountID, &rep, &f.StatusCode, &issued, &firstShip, &completed, &leadTime, &source, &ordered, &packed); err != nil {
			return nil, err
		}
		f.SalesRepID, f.LeadTimeSourceCode = nullStringPtr(rep), nullStringPtr(source)
		f.IssuedAt, f.FirstShipAt, f.CompletedAt = nullTimePtr(issued), nullTimePtr(firstShip), nullTimePtr(completed)
		f.LeadTimeDays = nullInt32Ptr(leadTime)
		f.QuantityOrdered, f.QuantityPacked = ordered, packed
		out = append(out, f)
	}
	return out, rows.Err()
}

// WriteFacts writes upserts in multi-row statements and deletes by full key. A row whose ship-by date or
// product line moved has a new key, so it is an upsert of the new row and a delete of the old.
func (r *deliveryFactRepoImpl) WriteFacts(ctx context.Context, upserts, deletes []domain.DeliveryOrderFact) *apierror.APIError {
	ctx, span := deliveryFactRepoTracer.Start(ctx, "repository.delivery_fact.write_facts")
	defer span.End()

	now := time.Now().UTC()
	for start := 0; start < len(upserts); start += deliveryFactWriteBatch {
		batch := upserts[start:min(start+deliveryFactWriteBatch, len(upserts))]
		args := make([]any, 0, len(batch)*16)
		for _, f := range batch {
			args = append(args, f.OwnerAccountID, f.ShipByDate, f.SalesOrderID, f.ProductLineKey, f.SalesOrderNumber,
				f.BuyerAccountID, f.SalesRepID, f.StatusCode, f.IssuedAt, f.FirstShipAt, f.CompletedAt,
				f.LeadTimeDays, f.LeadTimeSourceCode, f.QuantityOrdered, f.QuantityPacked, now)
		}
		values := strings.TrimSuffix(strings.Repeat("(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?),", len(batch)), ",")
		_, err := r.queries.DB().ExecContext(ctx, `INSERT INTO delivery_order_fact (`+deliveryFactColumns+`, refreshed_at) VALUES `+values+`
ON DUPLICATE KEY UPDATE sales_order_number = VALUES(sales_order_number), buyer_account_id = VALUES(buyer_account_id),
sales_rep_id = VALUES(sales_rep_id), sales_order_status_code = VALUES(sales_order_status_code), issued_at = VALUES(issued_at),
first_ship_at = VALUES(first_ship_at), completed_at = VALUES(completed_at), lead_time_days = VALUES(lead_time_days),
lead_time_source_code = VALUES(lead_time_source_code), quantity_ordered = VALUES(quantity_ordered),
quantity_packed = VALUES(quantity_packed), refreshed_at = VALUES(refreshed_at)`, args...)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}
	for start := 0; start < len(deletes); start += deliveryFactWriteBatch {
		batch := deletes[start:min(start+deliveryFactWriteBatch, len(deletes))]
		args := make([]any, 0, len(batch)*4)
		for _, f := range batch {
			args = append(args, f.OwnerAccountID, f.ShipByDate, f.SalesOrderID, f.ProductLineKey)
		}
		match := strings.TrimSuffix(strings.Repeat("(owner_account_id = ? AND ship_by_date = ? AND sales_order_id = ? AND product_line_key = ?) OR ", len(batch)), " OR ")
		if _, err := r.queries.DB().ExecContext(ctx, `DELETE FROM delivery_order_fact WHERE `+match, args...); err != nil {
			return tracing.Trace(span, db.MapSQLError(err))
		}
	}
	return nil
}

func (r *deliveryFactRepoImpl) ListSalesOrderIDsAfter(ctx context.Context, afterID string, limit int32) ([]string, *apierror.APIError) {
	ctx, span := deliveryFactRepoTracer.Start(ctx, "repository.delivery_fact.list_sales_orders_after")
	defer span.End()
	ids, err := selectStrings(ctx, r.queries.DB(), `SELECT id FROM sales_order WHERE id > ? ORDER BY id LIMIT ?`, afterID, limit)
	return ids, tracing.Trace(span, db.MapSQLError(err))
}

func (r *deliveryFactRepoImpl) ListFactSalesOrderIDsAfter(ctx context.Context, afterID string, limit int32) ([]string, *apierror.APIError) {
	ctx, span := deliveryFactRepoTracer.Start(ctx, "repository.delivery_fact.list_fact_sales_orders_after")
	defer span.End()
	ids, err := selectStrings(ctx, r.queries.DB(), `SELECT DISTINCT sales_order_id FROM delivery_order_fact FORCE INDEX (delivery_order_fact_order_idx)
WHERE sales_order_id > ? ORDER BY sales_order_id LIMIT ?`, afterID, limit)
	return ids, tracing.Trace(span, db.MapSQLError(err))
}

func (r *deliveryFactRepoImpl) FilterExistingSalesOrderIDs(ctx context.Context, salesOrderIDs []string) ([]string, *apierror.APIError) {
	ctx, span := deliveryFactRepoTracer.Start(ctx, "repository.delivery_fact.filter_existing_sales_orders")
	defer span.End()
	var out []string
	for start := 0; start < len(salesOrderIDs); start += deliveryFactIDBatch {
		batch := salesOrderIDs[start:min(start+deliveryFactIDBatch, len(salesOrderIDs))]
		ids, err := selectStrings(ctx, r.queries.DB(), `SELECT id FROM sales_order WHERE id IN (`+placeholders(len(batch))+`)`, stringsToAny(batch)...)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		out = append(out, ids...)
	}
	return out, nil
}

// deliveryFactResolvers select the orders a scope's ids reach, each id list spliced in at %s and each
// statement ending in LIMIT ?.
var deliveryFactResolvers = map[domain.DeliveryFactScope]string{
	domain.DeliveryFactScopeSalesOrderLine: `SELECT DISTINCT sales_order_id FROM sales_order_line WHERE id IN (%s) LIMIT ?`,
	domain.DeliveryFactScopePick:           `SELECT sales_order_id FROM pick WHERE id IN (%s) AND sales_order_id IS NOT NULL LIMIT ?`,
	domain.DeliveryFactScopePickLine: `SELECT DISTINCT sol.sales_order_id FROM pick_line pl
JOIN sales_order_line sol ON sol.id = pl.sales_order_line_id WHERE pl.id IN (%s) LIMIT ?`,
	domain.DeliveryFactScopeQuantity: `SELECT sales_order_id FROM sales_order_line WHERE quantity_id IN (%[1]s)
UNION SELECT sol.sales_order_id FROM pick_line pl JOIN sales_order_line sol ON sol.id = pl.sales_order_line_id
WHERE pl.quantity_id IN (%[1]s) LIMIT ?`,
	domain.DeliveryFactScopeProduct: `SELECT DISTINCT sales_order_id FROM sales_order_line WHERE product_id IN (%s) LIMIT ?`,
}

func (r *deliveryFactRepoImpl) ResolveSalesOrderIDs(ctx context.Context, scope domain.DeliveryFactScope, scopeIDs []string, limit int) ([]string, *apierror.APIError) {
	ctx, span := deliveryFactRepoTracer.Start(ctx, "repository.delivery_fact.resolve_sales_orders")
	defer span.End()

	if scope == domain.DeliveryFactScopeSalesOrder {
		return scopeIDs, nil
	}
	resolver, ok := deliveryFactResolvers[scope]
	if !ok {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unknown delivery fact scope "+string(scope)+"."))
	}
	var out []string
	for start := 0; start < len(scopeIDs) && len(out) <= limit; start += deliveryFactIDBatch {
		batch := scopeIDs[start:min(start+deliveryFactIDBatch, len(scopeIDs))]
		list := placeholders(len(batch))
		args := stringsToAny(batch)
		query := strings.ReplaceAll(strings.ReplaceAll(resolver, "%[1]s", list), "%s", list)
		if scope == domain.DeliveryFactScopeQuantity {
			args = append(args, stringsToAny(batch)...)
		}
		ids, err := selectStrings(ctx, r.queries.DB(), query, append(args, limit+1-len(out))...)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		out = append(out, ids...)
	}
	slices.Sort(out)
	return slices.Compact(out), nil
}

func (r *deliveryFactRepoImpl) MarkDirty(ctx context.Context, scope domain.DeliveryFactScope, scopeID, accountID string) *apierror.APIError {
	ctx, span := deliveryFactRepoTracer.Start(ctx, "repository.delivery_fact.mark_dirty")
	defer span.End()
	return tracing.Trace(span, db.MapSQLError(r.mark(ctx, scope, accountID, []string{scopeID})))
}

func (r *deliveryFactRepoImpl) MarkSalesOrdersDirty(ctx context.Context, accountID string, salesOrderIDs []string) *apierror.APIError {
	ctx, span := deliveryFactRepoTracer.Start(ctx, "repository.delivery_fact.mark_sales_orders_dirty")
	defer span.End()
	return tracing.Trace(span, db.MapSQLError(r.mark(ctx, domain.DeliveryFactScopeSalesOrder, accountID, salesOrderIDs)))
}

func (r *deliveryFactRepoImpl) mark(ctx context.Context, scope domain.DeliveryFactScope, accountID string, scopeIDs []string) error {
	for start := 0; start < len(scopeIDs); start += deliveryFactWriteBatch {
		batch := scopeIDs[start:min(start+deliveryFactWriteBatch, len(scopeIDs))]
		args := make([]any, 0, 3*len(batch))
		for _, id := range batch {
			args = append(args, string(scope), id, accountID)
		}
		values := strings.TrimSuffix(strings.Repeat("(?, ?, ?, NOW(3)),", len(batch)), ",")
		if _, err := r.queries.DB().ExecContext(ctx, `INSERT INTO delivery_fact_dirty (scope_type, scope_id, account_id, marked_at) VALUES `+
			values+` ON DUPLICATE KEY UPDATE marked_at = VALUES(marked_at)`, args...); err != nil {
			return err
		}
	}
	return nil
}

func (r *deliveryFactRepoImpl) ListDirty(ctx context.Context, limit int32) ([]domain.DeliveryFactDirtyMark, *apierror.APIError) {
	ctx, span := deliveryFactRepoTracer.Start(ctx, "repository.delivery_fact.list_dirty")
	defer span.End()

	rows, err := r.queries.DB().QueryContext(ctx, `SELECT scope_type, scope_id, account_id, marked_at FROM delivery_fact_dirty
ORDER BY marked_at LIMIT ?`, limit)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	defer func() { _ = rows.Close() }()
	var marks []domain.DeliveryFactDirtyMark
	for rows.Next() {
		var m domain.DeliveryFactDirtyMark
		if err := rows.Scan(&m.ScopeType, &m.ScopeID, &m.AccountID, &m.MarkedAt); err != nil {
			return nil, tracing.Trace(span, db.MapSQLError(err))
		}
		marks = append(marks, m)
	}
	return marks, tracing.Trace(span, db.MapSQLError(rows.Err()))
}

func (r *deliveryFactRepoImpl) ClearDirty(ctx context.Context, marks []domain.DeliveryFactDirtyMark) *apierror.APIError {
	ctx, span := deliveryFactRepoTracer.Start(ctx, "repository.delivery_fact.clear_dirty")
	defer span.End()

	for start := 0; start < len(marks); start += deliveryFactWriteBatch {
		batch := marks[start:min(start+deliveryFactWriteBatch, len(marks))]
		args := make([]any, 0, 3*len(batch))
		for _, m := range batch {
			args = append(args, string(m.ScopeType), m.ScopeID, m.MarkedAt)
		}
		match := strings.TrimSuffix(strings.Repeat("(scope_type = ? AND scope_id = ? AND marked_at = ?) OR ", len(batch)), " OR ")
		if _, err := r.queries.DB().ExecContext(ctx, `DELETE FROM delivery_fact_dirty WHERE `+match, args...); err != nil {
			return tracing.Trace(span, db.MapSQLError(err))
		}
	}
	return nil
}

func (r *deliveryFactRepoImpl) GetSync(ctx context.Context) (*domain.DeliveryFactSync, *apierror.APIError) {
	ctx, span := deliveryFactRepoTracer.Start(ctx, "repository.delivery_fact.get_sync")
	defer span.End()

	var (
		cursor             sql.NullString
		started, completed sql.NullTime
	)
	err := r.queries.DB().QueryRowContext(ctx, `SELECT cursor_sales_order_id, pass_started_at, last_completed_at FROM sales_sync WHERE name = ?`,
		deliveryFactSyncName).Scan(&cursor, &started, &completed)
	if errors.Is(err, sql.ErrNoRows) {
		return &domain.DeliveryFactSync{}, nil
	}
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return &domain.DeliveryFactSync{CursorSalesOrderID: nullStringPtr(cursor), PassStartedAt: nullTimePtr(started), LastCompletedAt: nullTimePtr(completed)}, nil
}

func (r *deliveryFactRepoImpl) SaveSync(ctx context.Context, sync domain.DeliveryFactSync) *apierror.APIError {
	ctx, span := deliveryFactRepoTracer.Start(ctx, "repository.delivery_fact.save_sync")
	defer span.End()

	_, err := r.queries.DB().ExecContext(ctx, `INSERT INTO sales_sync (name, cursor_sales_order_id, pass_started_at, last_completed_at, updated_at)
VALUES (?, ?, ?, ?, NOW(3))
ON DUPLICATE KEY UPDATE cursor_sales_order_id = VALUES(cursor_sales_order_id), pass_started_at = VALUES(pass_started_at),
    last_completed_at = VALUES(last_completed_at), updated_at = VALUES(updated_at)`,
		deliveryFactSyncName, sync.CursorSalesOrderID, toNullTime(sync.PassStartedAt), toNullTime(sync.LastCompletedAt))
	return tracing.Trace(span, db.MapSQLError(err))
}

func (r *deliveryFactRepoImpl) RestartReconcile(ctx context.Context) *apierror.APIError {
	ctx, span := deliveryFactRepoTracer.Start(ctx, "repository.delivery_fact.restart_reconcile")
	defer span.End()

	sync, apiErr := r.GetSync(ctx)
	if apiErr != nil || sync.LastCompletedAt == nil {
		return tracing.Trace(span, apiErr)
	}
	start, now := "", time.Now().UTC()
	sync.CursorSalesOrderID, sync.PassStartedAt = &start, &now
	return tracing.Trace(span, r.SaveSync(ctx, *sync))
}
