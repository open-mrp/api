package repository

import (
	"context"
	gosql "database/sql"
	"strings"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/pagination"
	"github.com/open-mrp/api/shared/tracing"
)

func openCreditFundsReceivedAt(e *domain.OpenCreditEntry) time.Time { return e.FundsReceivedAt }
func openCreditID(e *domain.OpenCreditEntry) string                 { return e.ID }

// ListOpenCredits pages the account's open credits, most recently received first: money received, not
// marked fully allocated, and with part of it still unapplied. The keyset follows the order, so a page
// can be read in either direction.
func (r *transactionAllocationRepoImpl) ListOpenCredits(ctx context.Context, params domain.ListOpenCreditsParams) (*domain.ListOpenCreditsResult, *apierror.APIError) {
	ctx, span := transactionAllocationRepoTracer.Start(ctx, "repository.transaction_allocation.list_open_credits")
	defer span.End()

	limit := params.Limit
	if limit <= 0 {
		limit = 100
	}
	limit = min(limit, 1000)

	cur, apiErr := decodeTransactionCursor(params.Cursor)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	f := &transactionFilter{}
	f.add("t.account_id = ?", params.AccountID)
	f.add("t.is_fully_allocated = 0")
	f.add("t.funds_received_at IS NOT NULL")
	f.add("q.value > COALESCE((SELECT SUM(q3.value) FROM transaction_allocation ta3 JOIN quantity q3 ON q3.id = ta3.amount_id WHERE ta3.transaction_id = t.id), 0)")
	f.in("t.customer_account_id", params.CustomerIDs)
	if params.StartDate != nil {
		f.add("t.funds_received_at >= ?", *params.StartDate)
	}
	if params.EndDate != nil {
		f.add("t.funds_received_at < ?", *params.EndDate)
	}
	if params.SearchQuery != nil && strings.TrimSpace(*params.SearchQuery) != "" {
		like := "%" + db.EscapeLike(strings.TrimSpace(*params.SearchQuery)) + "%"
		f.add("(t.id LIKE ? OR t.number LIKE ? OR cust.name LIKE ? OR COALESCE(t.note, '') LIKE ?)", like, like, like, like)
	}
	orderBy := "t.funds_received_at DESC, t.id DESC"
	if cur != nil {
		if cur.Direction == pagination.DirectionBackward {
			f.add("(t.funds_received_at > ? OR (t.funds_received_at = ? AND t.id > ?))", cur.OccurredAt, cur.OccurredAt, cur.ID)
			orderBy = "t.funds_received_at ASC, t.id ASC"
		} else {
			f.add("(t.funds_received_at < ? OR (t.funds_received_at = ? AND t.id < ?))", cur.OccurredAt, cur.OccurredAt, cur.ID)
		}
	}

	query := `SELECT t.id, t.number, t.note, t.stripe_payment_id, t.created_at, t.funds_received_at, tt.name,
	CAST(q.value AS CHAR), t.customer_account_id, cust.name, ar.external_number, tm.name, adjt.name, COALESCE(usr.username, ''),
	CAST(COALESCE((SELECT SUM(q2.value) FROM transaction_allocation ta2 JOIN quantity q2 ON q2.id = ta2.amount_id WHERE ta2.transaction_id = t.id), 0) AS CHAR)
FROM ` + "`transaction`" + ` t
JOIN quantity q ON q.id = t.amount_id
JOIN account cust ON cust.id = t.customer_account_id
JOIN transaction_type tt ON tt.code = t.transaction_type_code
LEFT JOIN account_relation ar ON ar.counterparty_account_id = t.customer_account_id AND ar.owner_account_id = t.account_id AND ar.account_relation_role_code = 'customer'
LEFT JOIN transaction_method tm ON tm.code = t.transaction_method_code
LEFT JOIN adjustment_type adjt ON adjt.code = t.adjustment_type_code
-- responsible_user_id holds an account_user id, or a user id on rows the legacy dashboard wrote.
LEFT JOIN account_user au ON au.account_id = t.account_id AND (au.id = t.responsible_user_id OR au.user_id = t.responsible_user_id)
LEFT JOIN ` + "`user`" + ` usr ON usr.id = au.user_id
WHERE ` + strings.Join(f.where, " AND ") + `
ORDER BY ` + orderBy + `
LIMIT ?`

	rows, err := r.queries.DB().QueryContext(ctx, query, append(f.args, limit+1)...)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	var entries []*domain.OpenCreditEntry
	for rows.Next() {
		var (
			e                                     domain.OpenCreditEntry
			note, stripe, custNumber, method, adj gosql.NullString
			responsible                           string
			funds                                 time.Time
		)
		if err := rows.Scan(&e.ID, &e.Number, &note, &stripe, &e.CreatedAt, &funds, &e.TransactionType,
			&e.OriginalAmount, &e.CustomerID, &e.CustomerName, &custNumber, &method, &adj, &responsible, &e.AllocatedAmount); err != nil {
			_ = rows.Close()
			return nil, tracing.Trace(span, db.MapSQLError(err))
		}
		e.FundsReceivedAt = funds
		e.Note, e.StripePaymentID, e.CustomerNumber = nullStringPtr(note), nullStringPtr(stripe), nullStringPtr(custNumber)
		e.TransactionMethod, e.AdjustmentType = nullStringPtr(method), nullStringPtr(adj)
		if responsible != "" {
			e.ResponsibleUserName = &responsible
		}
		e.LeftoverAmount = subtractDecimalStrings(e.OriginalAmount, e.AllocatedAmount)
		entries = append(entries, &e)
	}
	_ = rows.Close()
	if apiErr := db.MapSQLError(rows.Err()); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	result, pageInfo := pagination.BuildPageString(entries, limit, cursorDirection(cur), openCreditFundsReceivedAt, openCreditID)
	if apiErr := r.attachOpenCreditAllocations(ctx, result); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return &domain.ListOpenCreditsResult{Entries: result, PageInfo: pageInfo}, nil
}

func (r *transactionAllocationRepoImpl) attachOpenCreditAllocations(ctx context.Context, entries []*domain.OpenCreditEntry) *apierror.APIError {
	if len(entries) == 0 {
		return nil
	}
	ids := make([]string, len(entries))
	for i, e := range entries {
		ids[i] = e.ID
	}
	rows, err := r.queries.GetOpenCreditAllocations(ctx, ids)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}
	byTxn := make(map[string][]domain.InvoiceAllocationEntry)
	for _, row := range rows {
		byTxn[row.TransactionID] = append(byTxn[row.TransactionID], domain.InvoiceAllocationEntry{
			InvoiceNumber: row.InvoiceNumber,
			Amount:        decimalToString(row.Amount),
		})
	}
	for _, e := range entries {
		e.InvoiceAllocations = byTxn[e.ID]
		if e.InvoiceAllocations == nil {
			e.InvoiceAllocations = []domain.InvoiceAllocationEntry{}
		}
	}
	return nil
}
