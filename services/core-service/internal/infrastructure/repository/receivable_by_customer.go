package repository

import (
	"context"
	gosql "database/sql"
	"fmt"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/db"
	"github.com/open-mrp/apikit/pagination"
	"github.com/open-mrp/apikit/tracing"
)

// receivableBalanceSQL is an invoice's balance as of the allocation cutoff (one argument, NULL for none).
// Each line is priced as the dashboard's multiplyRate does and summed unrounded: the balance is rounded
// once, as the dashboard's receivables report does (see the line-pricing skill).
const receivableBalanceSQL = `ROUND(
    COALESCE((
        SELECT SUM(CASE WHEN q.unit_id = rt.denominator_unit_id THEN q.value * rt.value ELSE (q.value * qu.ratio_numerator / qu.ratio_denominator) * (rt.value / (rtu.ratio_numerator / rtu.ratio_denominator)) END)
        FROM invoice_line il
        JOIN quantity q ON q.id = il.quantity_id
        JOIN sales_order_line sol ON sol.id = il.sales_order_line_id
        JOIN rate rt ON rt.id = sol.unit_price_id
        JOIN unit qu ON qu.id = q.unit_id
        JOIN unit rtu ON rtu.id = rt.denominator_unit_id
        WHERE il.invoice_id = inv.id
    ), 0)
    -
    COALESCE((
        SELECT SUM(aq.value)
        FROM transaction_allocation ta
        JOIN quantity aq ON aq.id = ta.amount_id
        JOIN ` + "`transaction`" + ` txn ON txn.id = ta.transaction_id
        WHERE ta.invoice_id = inv.id
        AND txn.funds_received_at IS NOT NULL
        AND txn.funds_received_at < COALESCE(?, CAST('9999-12-31 23:59:59' AS DATETIME(6)))
    ), 0),
2)`

// customerReceivables reads one customer's receivables, in list order and at most limit of them.
//
// The customer is a property of the invoice's order, which no invoice key holds in list order, so it
// reads whichever set is smaller: the account's unpaid invoices, walked in list order and filtered to
// the customer, or the customer's orders, each looked up for its invoices and the few found sorted.
// Walking in order lets a page stop once it has its rows, so balances are computed only for the
// invoices it reads.
func (r *receivableRepoImpl) customerReceivables(ctx context.Context, accountID, customerID string, cutoff *time.Time,
	query *string, cursor *pagination.StringCursor, limit int32) ([]domain.ReceivableEntry, error) {
	q := r.queries.DB()
	walk, err := unpaidFewerThanOrders(ctx, q, accountID, []string{customerID})
	if err != nil {
		return nil, err
	}

	f := &listFilter{}
	f.add("inv.account_id = ?", accountID)
	f.add("so.buyer_account_id = ?", customerID)
	f.add("inv.is_paid_in_full = false")
	if cutoff != nil {
		f.add("inv.created_at < ?", *cutoff)
	}
	if search := buildReceivableSearchParam(query); search.Valid {
		f.add("(inv.number LIKE ? OR buyer.name LIKE ?)", search.String, search.String)
	}
	orderBy := f.keyset(cursor, "inv.created_at", "inv.id")

	// The orders' ids are read off the buyer key alone: an order's row is read only once it has an
	// unpaid invoice.
	from := `(SELECT so2.id FROM sales_order so2 FORCE INDEX (sales_order_buyer_account_id_idx) WHERE so2.buyer_account_id = ?) drv
JOIN invoice inv FORCE INDEX (` + invoiceOrderIndex + `) ON inv.sales_order_id = drv.id
JOIN sales_order so ON so.id = inv.sales_order_id`
	fromArgs := []any{customerID}
	hint := "JOIN_ORDER(so2, inv, so, ar, buyer)"
	if walk {
		from = `invoice inv FORCE INDEX (` + invoicePaidIndex + `)
JOIN sales_order so ON so.id = inv.sales_order_id`
		fromArgs = nil
		hint = "JOIN_ORDER(inv, so, ar, buyer)"
	}
	// An as-of listing drops what was settled by the cutoff, which only the balance shows.
	having := ""
	if cutoff != nil {
		having = "\nHAVING remaining_balance > 0"
	}
	stmt := `SELECT /*+ ` + hint + ` */ inv.id, inv.number, inv.is_paid_in_full, inv.created_at, so.customer_po_number,
    buyer.id, buyer.name, ar.external_number,
    ` + receivableBalanceSQL + ` AS remaining_balance
FROM ` + from + `
JOIN account_relation ar ON ar.owner_account_id = inv.account_id
    AND ar.counterparty_account_id = so.buyer_account_id
    AND ar.account_relation_role_code = 'customer'
JOIN account buyer ON buyer.id = so.buyer_account_id` + f.whereSQL() + having + `
ORDER BY ` + orderBy + `
LIMIT ?`

	args := append(append(append([]any{buildAllocationCutoffParam(cutoff)}, fromArgs...), f.args...), limit)
	rows, err := q.QueryContext(ctx, stmt, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.ReceivableEntry{}
	for rows.Next() {
		var e domain.ReceivableEntry
		var po gosql.NullString
		var balance float64
		if err := rows.Scan(&e.InvoiceID, &e.InvoiceNumber, &e.IsPaidInFull, &e.InvoicedAt, &po,
			&e.CustomerID, &e.CustomerName, &e.CustomerNumber, &balance); err != nil {
			return nil, err
		}
		e.PONumber = nullStringPtr(po)
		e.RemainingBalance = fmt.Sprintf("%.2f", balance)
		items = append(items, e)
	}
	return items, rows.Err()
}

func (r *receivableRepoImpl) ListByCustomer(ctx context.Context, params domain.ListReceivablesByCustomerParams) (*domain.ListReceivablesByCustomerResult, *apierror.APIError) {
	ctx, span := receivableRepoTracer.Start(ctx, "repository.receivable.list_by_customer")
	defer span.End()

	cursor, apiErr := decodeListCursor(params.Cursor)
	if apiErr != nil {
		return nil, apiErr
	}
	var cursorDir *pagination.Direction
	if cursor != nil {
		cursorDir = &cursor.Direction
	}
	items, err := r.customerReceivables(ctx, params.AccountID, params.CustomerAccountID, params.CutoffDate, params.Query, cursor, params.Limit+1)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	result, pageInfo := pagination.BuildPageString(items, params.Limit, cursorDir, receivableEntryCreatedAt, receivableEntryID)
	return &domain.ListReceivablesByCustomerResult{Items: result, PageString: pageInfo.NextCursor}, nil
}

func (r *receivableRepoImpl) ListAllByCustomer(ctx context.Context, accountID, customerAccountID string, cutoffDate *time.Time) ([]domain.ReceivableEntry, *apierror.APIError) {
	ctx, span := receivableRepoTracer.Start(ctx, "repository.receivable.list_all_by_customer")
	defer span.End()

	items, err := r.customerReceivables(ctx, accountID, customerAccountID, cutoffDate, nil, nil, 10000)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return items, nil
}
