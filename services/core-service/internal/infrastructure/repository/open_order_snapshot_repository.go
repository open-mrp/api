package repository

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var openOrderSnapshotTracer = tracing.GetTracer("core-service.infrastructure.repository.open_order_snapshot")

const (
	// openOrderSyncName is the snapshot pass's row in sales_sync.
	openOrderSyncName = "open_orders"

	// openOrderRefreshBatch is how many orders one refresh reads from the source tables at a time: each
	// line is a few primary-key lookups into tables far larger than memory, so a batch of a few orders'
	// lines stays within the statement budget even when none of them is cached.
	openOrderRefreshBatch = 5

	// openOrderWriteBatch is how many rows one INSERT or DELETE carries.
	openOrderWriteBatch = 100
)

// openOrderSnapshotReady latches once the snapshot's first pass has completed; a completed pass is never undone.
var openOrderSnapshotReady atomic.Bool

// snapshotReady reports whether the order-book reports may read the snapshot.
func snapshotReady(ctx context.Context, q *sqlc.Queries) bool {
	if openOrderSnapshotReady.Load() {
		return true
	}
	row, found, err := (&salesFactRepoImpl{queries: q}).getSalesSync(ctx, openOrderSyncName)
	if err != nil || !found || !row.lastCompletedAt.Valid {
		return false
	}
	openOrderSnapshotReady.Store(true)
	return true
}

type openOrderSnapshotRepoImpl struct {
	queries *sqlc.Queries
}

func NewOpenOrderSnapshotRepo(queries *sqlc.Queries) domain.OpenOrderSnapshotRepo {
	return &openOrderSnapshotRepoImpl{queries: queries}
}

// snapshotTable is one snapshot table: its columns, the first being the row's id, and the statement reading
// the same columns from the source tables for a set of keys (keyed).
type snapshotTable struct {
	name    string
	columns []string
	// source selects the columns, in order, for the keys bound to its single IN list.
	source string
	// stored selects the snapshot's rows for the same keys.
	stored string
}

var (
	openOrderTable = snapshotTable{
		name: "open_order",
		columns: []string{"id", "owner_account_id", "sales_order_type_code", "sales_order_status_code", "number", "buyer_account_id",
			"sales_rep_id", "order_discount_id", "shipping_address_id", "customer_po_number", "issued_at", "completed_at", "first_ship_at", "promised_at"},
		source: `SELECT so.id, so.owner_account_id, so.sales_order_type_code, so.sales_order_status_code, so.number, so.buyer_account_id,
    so.sales_rep_id, so.order_discount_id, so.shipping_address_id, so.customer_po_number, so.issued_at, so.completed_at, so.first_ship_at, so.promised_at
FROM sales_order so
WHERE so.id IN (%s) AND so.sales_order_type_code = 'sales_order' AND so.sales_order_status_code = 'issued'`,
		stored: `SELECT id, owner_account_id, sales_order_type_code, sales_order_status_code, number, buyer_account_id,
    sales_rep_id, order_discount_id, shipping_address_id, customer_po_number, issued_at, completed_at, first_ship_at, promised_at
FROM open_order WHERE id IN (%s)`,
	}
	// The lines are read only for orders the order read found issued, so they need no status predicate.
	openOrderLineTable = snapshotTable{
		name: "open_order_line",
		columns: []string{"id", "account_id", "sales_order_id", "product_id",
			"quantity_id", "quantity_value", "quantity_unit_id",
			"unit_price_id", "price_value", "price_numerator_unit_id", "price_denominator_unit_id",
			"unit_cost_id", "cost_value", "cost_numerator_unit_id", "cost_denominator_unit_id"},
		source: `SELECT STRAIGHT_JOIN sol.id, so.owner_account_id, sol.sales_order_id, sol.product_id,
    sol.quantity_id, q.value, q.unit_id,
    sol.unit_price_id, rp.value, rp.numerator_unit_id, rp.denominator_unit_id,
    sol.unit_cost_id, rc.value, rc.numerator_unit_id, rc.denominator_unit_id
FROM sales_order so
JOIN sales_order_line sol FORCE INDEX (sales_order_line_order_product_idx) ON sol.sales_order_id = so.id
LEFT JOIN quantity q ON q.id = sol.quantity_id
LEFT JOIN rate rp ON rp.id = sol.unit_price_id
LEFT JOIN rate rc ON rc.id = sol.unit_cost_id
WHERE so.id IN (%s)`,
		stored: `SELECT id, account_id, sales_order_id, product_id,
    quantity_id, quantity_value, quantity_unit_id,
    unit_price_id, price_value, price_numerator_unit_id, price_denominator_unit_id,
    unit_cost_id, cost_value, cost_numerator_unit_id, cost_denominator_unit_id
FROM open_order_line WHERE sales_order_id IN (%s)`,
	}
	openOrderInvoicedTable = snapshotTable{
		name:    "open_order_invoiced",
		columns: []string{"id", "account_id", "sales_order_line_id", "sales_order_id", "invoice_id", "quantity_id", "quantity_value", "quantity_unit_id"},
		source: `SELECT STRAIGHT_JOIN il.id, so.owner_account_id, il.sales_order_line_id, sol.sales_order_id, il.invoice_id, il.quantity_id, q.value, q.unit_id
FROM sales_order so
JOIN sales_order_line sol FORCE INDEX (sales_order_line_order_product_idx) ON sol.sales_order_id = so.id
JOIN invoice_line il FORCE INDEX (invoice_line_sol_qty_idx) ON il.sales_order_line_id = sol.id
JOIN quantity q ON q.id = il.quantity_id
WHERE so.id IN (%s)`,
		stored: `SELECT id, account_id, sales_order_line_id, sales_order_id, invoice_id, quantity_id, quantity_value, quantity_unit_id
FROM open_order_invoiced WHERE sales_order_id IN (%s)`,
	}
)

func (r *openOrderSnapshotRepoImpl) RefreshOrders(ctx context.Context, orderIDs []string) ([]string, int, *apierror.APIError) {
	ctx, span := openOrderSnapshotTracer.Start(ctx, "repository.open_order_snapshot.refresh_orders")
	defer span.End()

	accounts := map[string]bool{}
	changed := 0
	for start := 0; start < len(orderIDs); start += openOrderRefreshBatch {
		n, err := r.refreshBatch(ctx, orderIDs[start:min(start+openOrderRefreshBatch, len(orderIDs))], accounts)
		changed += n
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return slices.Sorted(maps.Keys(accounts)), changed, tracing.Trace(span, apiErr)
		}
	}
	return slices.Sorted(maps.Keys(accounts)), changed, nil
}

// refreshBatch syncs one batch of orders. Orders are written before their lines, and lines before their
// invoiced rows, so a report read in between sees an order with fewer lines rather than lines without
// their order; deletes run in the reverse order. The account of every row written or deleted is added to
// accounts.
func (r *openOrderSnapshotRepoImpl) refreshBatch(ctx context.Context, orderIDs []string, accounts map[string]bool) (int, error) {
	orders, err := r.readSource(ctx, openOrderTable, orderIDs)
	if err != nil {
		return 0, err
	}
	issued := make([]string, 0, len(orders))
	for _, row := range orders {
		issued = append(issued, row.id())
	}
	var lines, invoiced []snapshotRow
	if len(issued) > 0 {
		if lines, err = r.readSource(ctx, openOrderLineTable, issued); err != nil {
			return 0, err
		}
		if invoiced, err = r.readSource(ctx, openOrderInvoicedTable, issued); err != nil {
			return 0, err
		}
	}

	type diff struct {
		table          snapshotTable
		upsert, delete []snapshotRow
	}
	var diffs []diff
	for _, d := range []struct {
		table snapshotTable
		want  []snapshotRow
	}{{openOrderTable, orders}, {openOrderLineTable, lines}, {openOrderInvoicedTable, invoiced}} {
		have, err := r.readStored(ctx, d.table, orderIDs)
		if err != nil {
			return 0, err
		}
		upsert, del := diffSnapshotRows(have, d.want)
		diffs = append(diffs, diff{d.table, upsert, del})
	}

	changed := 0
	for _, d := range diffs {
		if err := r.upsert(ctx, d.table, d.upsert); err != nil {
			return changed, err
		}
		changed += len(d.upsert)
		addSnapshotAccounts(accounts, d.upsert)
	}
	for _, d := range slices.Backward(diffs) {
		if err := r.delete(ctx, d.table, d.delete); err != nil {
			return changed, err
		}
		changed += len(d.delete)
		addSnapshotAccounts(accounts, d.delete)
	}
	return changed, nil
}

// snapshotRow is one row's column values as the driver returned them; the first is its id.
type snapshotRow []any

func (row snapshotRow) id() string { return snapshotValueString(row[0]) }

// addSnapshotAccounts adds each row's account, which every snapshot table keeps second.
func addSnapshotAccounts(accounts map[string]bool, rows []snapshotRow) {
	for _, row := range rows {
		accounts[snapshotValueString(row[1])] = true
	}
}

func (r *openOrderSnapshotRepoImpl) readSource(ctx context.Context, t snapshotTable, keys []string) ([]snapshotRow, error) {
	return r.readRows(ctx, fmt.Sprintf(t.source, placeholders(len(keys))), len(t.columns), stringArgs(keys))
}

func (r *openOrderSnapshotRepoImpl) readStored(ctx context.Context, t snapshotTable, keys []string) ([]snapshotRow, error) {
	return r.readRows(ctx, fmt.Sprintf(t.stored, placeholders(len(keys))), len(t.columns), stringArgs(keys))
}

func (r *openOrderSnapshotRepoImpl) readRows(ctx context.Context, query string, width int, args []any) ([]snapshotRow, error) {
	rows, err := r.queries.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []snapshotRow
	for rows.Next() {
		row := make(snapshotRow, width)
		dest := make([]any, width)
		for i := range row {
			dest[i] = &row[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// diffSnapshotRows returns the wanted rows that are missing or differ from the stored ones, and the stored
// rows no longer wanted.
func diffSnapshotRows(have, want []snapshotRow) (upsert, del []snapshotRow) {
	stored := make(map[string]snapshotRow, len(have))
	for _, row := range have {
		stored[row.id()] = row
	}
	for _, row := range want {
		old, ok := stored[row.id()]
		delete(stored, row.id())
		if !ok || !snapshotRowsEqual(old, row) {
			upsert = append(upsert, row)
		}
	}
	for _, row := range have {
		if _, gone := stored[row.id()]; gone {
			del = append(del, row)
		}
	}
	return upsert, del
}

func snapshotRowsEqual(a, b snapshotRow) bool {
	for i := range a {
		if !snapshotValuesEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

func snapshotValuesEqual(a, b any) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	if at, ok := a.(time.Time); ok {
		bt, ok := b.(time.Time)
		return ok && at.Equal(bt)
	}
	ab, aok := a.([]byte)
	bb, bok := b.([]byte)
	if aok && bok {
		return bytes.Equal(ab, bb)
	}
	return snapshotValueString(a) == snapshotValueString(b)
}

func snapshotValueString(v any) string {
	if b, ok := v.([]byte); ok {
		return string(b)
	}
	return fmt.Sprint(v)
}

func (r *openOrderSnapshotRepoImpl) upsert(ctx context.Context, t snapshotTable, rows []snapshotRow) error {
	row := "(" + placeholders(len(t.columns)) + "),"
	updates := make([]string, len(t.columns))
	for i, c := range t.columns {
		updates[i] = c + " = VALUES(" + c + ")"
	}
	for start := 0; start < len(rows); start += openOrderWriteBatch {
		batch := rows[start:min(start+openOrderWriteBatch, len(rows))]
		args := make([]any, 0, len(batch)*len(t.columns))
		for _, row := range batch {
			args = append(args, row...)
		}
		_, err := r.queries.DB().ExecContext(ctx, "INSERT INTO "+t.name+" ("+strings.Join(t.columns, ", ")+") VALUES "+
			strings.TrimSuffix(strings.Repeat(row, len(batch)), ",")+" ON DUPLICATE KEY UPDATE "+strings.Join(updates, ", "), args...)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *openOrderSnapshotRepoImpl) delete(ctx context.Context, t snapshotTable, rows []snapshotRow) error {
	for start := 0; start < len(rows); start += openOrderWriteBatch {
		batch := rows[start:min(start+openOrderWriteBatch, len(rows))]
		ids := make([]any, len(batch))
		for i, row := range batch {
			ids[i] = row[0]
		}
		if _, err := r.queries.DB().ExecContext(ctx, "DELETE FROM "+t.name+" WHERE id IN ("+placeholders(len(ids))+")", ids...); err != nil {
			return err
		}
	}
	return nil
}

func (r *openOrderSnapshotRepoImpl) ListIssuedOrderIDsAfter(ctx context.Context, afterID string, limit int32) ([]string, *apierror.APIError) {
	ids, err := selectStrings(ctx, r.queries.DB(), `SELECT id FROM sales_order FORCE INDEX (sales_order_sales_order_status_code_idx)
WHERE sales_order_status_code = 'issued' AND id > ? AND sales_order_type_code = 'sales_order' ORDER BY id LIMIT ?`, afterID, limit)
	return ids, db.MapSQLError(err)
}

func (r *openOrderSnapshotRepoImpl) ListSnapshotOrderIDsAfter(ctx context.Context, afterID string, limit int32) ([]string, *apierror.APIError) {
	ids, err := selectStrings(ctx, r.queries.DB(), `SELECT id FROM open_order FORCE INDEX (open_order_id_key) WHERE id > ? ORDER BY id LIMIT ?`, afterID, limit)
	return ids, db.MapSQLError(err)
}

func (r *openOrderSnapshotRepoImpl) FilterNotIssued(ctx context.Context, orderIDs []string) ([]string, *apierror.APIError) {
	if len(orderIDs) == 0 {
		return nil, nil
	}
	issued, err := selectStrings(ctx, r.queries.DB(), `SELECT id FROM sales_order WHERE id IN (`+placeholders(len(orderIDs))+`)
AND sales_order_type_code = 'sales_order' AND sales_order_status_code = 'issued'`, stringArgs(orderIDs)...)
	if err != nil {
		return nil, db.MapSQLError(err)
	}
	var out []string
	for _, id := range orderIDs {
		if !slices.Contains(issued, id) {
			out = append(out, id)
		}
	}
	return out, nil
}

func (r *openOrderSnapshotRepoImpl) ResolveOrders(ctx context.Context, scope domain.SalesFactScope, scopeIDs []string) ([]string, *apierror.APIError) {
	ctx, span := openOrderSnapshotTracer.Start(ctx, "repository.open_order_snapshot.resolve_orders")
	defer span.End()
	if len(scopeIDs) == 0 {
		return nil, nil
	}
	in := placeholders(len(scopeIDs))
	args := stringArgs(scopeIDs)
	var (
		ids []string
		err error
	)
	q := r.queries.DB()
	switch scope {
	case domain.SalesFactScopeSalesOrder:
		return scopeIDs, nil
	case domain.SalesFactScopeSalesOrderLine:
		// A deleted line is found in the snapshot, a new one on its order.
		ids, err = selectStrings(ctx, q, `SELECT sales_order_id FROM open_order_line WHERE id IN (`+in+`)
UNION SELECT sales_order_id FROM sales_order_line WHERE id IN (`+in+`)`, append(args, args...)...)
	case domain.SalesFactScopeQuantity:
		ids, err = selectStrings(ctx, q, `SELECT sales_order_id FROM open_order_line WHERE quantity_id IN (`+in+`)
UNION SELECT sales_order_id FROM open_order_invoiced WHERE quantity_id IN (`+in+`)`, append(args, args...)...)
	case domain.SalesFactScopeRate:
		ids, err = selectStrings(ctx, q, `SELECT sales_order_id FROM open_order_line WHERE unit_price_id IN (`+in+`)
UNION SELECT sales_order_id FROM open_order_line WHERE unit_cost_id IN (`+in+`)`, append(args, args...)...)
	case domain.SalesFactScopeInvoice:
		// A deleted invoice is found in the snapshot, a new one through its lines.
		ids, err = selectStrings(ctx, q, `SELECT sales_order_id FROM open_order_invoiced WHERE invoice_id IN (`+in+`)
UNION SELECT ol.sales_order_id FROM invoice_line il JOIN open_order_line ol ON ol.id = il.sales_order_line_id WHERE il.invoice_id IN (`+in+`)`,
			append(args, args...)...)
	case domain.SalesFactScopeBuyer:
		ids, err = selectStrings(ctx, q, `SELECT id FROM open_order WHERE buyer_account_id IN (`+in+`)`, args...)
	default:
		// Products, items and units are joined live by the reports, so the snapshot holds nothing of theirs.
		return nil, nil
	}
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return ids, nil
}

func (r *openOrderSnapshotRepoImpl) GetSync(ctx context.Context) (*domain.OpenOrderSnapshotSync, *apierror.APIError) {
	row, _, err := (&salesFactRepoImpl{queries: r.queries}).getSalesSync(ctx, openOrderSyncName)
	if err != nil {
		return nil, db.MapSQLError(err)
	}
	return &domain.OpenOrderSnapshotSync{PassStartedAt: nullTimePtr(row.passStartedAt), LastCompletedAt: nullTimePtr(row.lastCompletedAt)}, nil
}

func (r *openOrderSnapshotRepoImpl) SaveSync(ctx context.Context, s domain.OpenOrderSnapshotSync) *apierror.APIError {
	row := salesSyncRow{passStartedAt: optionalNullTime(s.PassStartedAt), lastCompletedAt: optionalNullTime(s.LastCompletedAt)}
	return db.MapSQLError((&salesFactRepoImpl{queries: r.queries}).saveSalesSync(ctx, openOrderSyncName, row))
}

func optionalNullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}
