package repository

import (
	"context"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/pagination"
	"github.com/open-mrp/api/shared/tracing"
)

// List pages an account's settlements newest first and summarizes each over all of its allocations.
//
// The page is chosen first, from settlement alone, so the (account_id, created_at) index yields it
// in order and stops at the limit; only that page's allocations are then aggregated. Aggregating in
// the same statement grouped every settlement in the account before the LIMIT applied. A filter on
// transactions or invoices selects the settlements that have such an allocation; the summary still
// covers the settlement's whole allocation list, as the dashboard showed it.
func (r *settlementRepoImpl) List(ctx context.Context, params domain.ListSettlementsParams) (*domain.ListSettlementsResult, *apierror.APIError) {
	ctx, span := settlementRepoTracer.Start(ctx, "repository.settlement.list")
	defer span.End()

	cur, apiErr := decodeTransactionCursor(params.Cursor)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	f := &transactionFilter{}
	f.add("s.account_id = ?", params.AccountID)
	if term := db.AllWordsPrefixQuery(params.Query); term != "" {
		f.add("MATCH(s.number) AGAINST(? IN BOOLEAN MODE)", term)
	}
	if len(params.TransactionIDs) > 0 || len(params.InvoiceIDs) > 0 {
		clause := "EXISTS (SELECT 1 FROM transaction_allocation fa WHERE fa.settlement_id = s.id"
		var args []any
		if len(params.TransactionIDs) > 0 {
			clause += " AND fa.transaction_id IN (" + placeholders(len(params.TransactionIDs)) + ")"
			args = append(args, stringArgs(params.TransactionIDs)...)
		}
		if len(params.InvoiceIDs) > 0 {
			clause += " AND fa.invoice_id IN (" + placeholders(len(params.InvoiceIDs)) + ")"
			args = append(args, stringArgs(params.InvoiceIDs)...)
		}
		f.add(clause+")", args...)
	}
	if params.StartDate != nil {
		f.add("s.created_at >= ?", *params.StartDate)
	}
	if params.EndDate != nil {
		f.add("s.created_at <= ?", *params.EndDate)
	}
	orderBy := "s.created_at DESC, s.id DESC"
	if cur != nil {
		if cur.Direction == pagination.DirectionBackward {
			f.add("(s.created_at > ? OR (s.created_at = ? AND s.id > ?))", cur.OccurredAt, cur.OccurredAt, cur.ID)
			orderBy = "s.created_at ASC, s.id ASC"
		} else {
			f.add("(s.created_at < ? OR (s.created_at = ? AND s.id < ?))", cur.OccurredAt, cur.OccurredAt, cur.ID)
		}
	}

	query := "SELECT s.id, s.number, s.created_at, s.updated_at FROM settlement s\nWHERE " + strings.Join(f.where, "\nAND ") + "\nORDER BY " + orderBy + "\nLIMIT ?"
	rows, err := r.queries.DB().QueryContext(ctx, query, append(f.args, params.Limit+1)...)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	var page []*domain.SettlementSummary
	for rows.Next() {
		var s domain.SettlementSummary
		if err := rows.Scan(&s.ID, &s.Number, &s.CreatedAt, &s.UpdatedAt); err != nil {
			_ = rows.Close()
			return nil, tracing.Trace(span, db.MapSQLError(err))
		}
		page = append(page, &s)
	}
	_ = rows.Close()
	if apiErr := db.MapSQLError(rows.Err()); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	result, pageInfo := pagination.BuildPageString(page, params.Limit, cursorDirection(cur), settlementCreatedAt, settlementID)
	if apiErr := r.summarize(ctx, result); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return &domain.ListSettlementsResult{Settlements: result, PageInfo: pageInfo}, nil
}

// summarize fills in each settlement's allocation count, totals by transaction type, invoice numbers
// and customer names, from one read of the page's allocations.
func (r *settlementRepoImpl) summarize(ctx context.Context, settlements []*domain.SettlementSummary) *apierror.APIError {
	if len(settlements) == 0 {
		return nil
	}
	byID := make(map[string]*domain.SettlementSummary, len(settlements))
	ids := make([]string, len(settlements))
	for i, s := range settlements {
		byID[s.ID] = s
		ids[i] = s.ID
	}

	rows, err := r.queries.DB().QueryContext(ctx, `
SELECT ta.settlement_id, t.transaction_type_code, CAST(q.value AS CHAR), COALESCE(inv.number, ''), buyer.name
FROM transaction_allocation ta
JOIN quantity q ON q.id = ta.amount_id
JOIN `+"`transaction`"+` t ON t.id = ta.transaction_id
JOIN account buyer ON buyer.id = t.customer_account_id
LEFT JOIN invoice inv ON inv.id = ta.invoice_id
WHERE ta.settlement_id IN (`+placeholders(len(ids))+`)
ORDER BY ta.created_at ASC, ta.id ASC`, stringArgs(ids)...)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}
	defer func() { _ = rows.Close() }()

	type totals struct{ payments, rebates, adjustments, credits decimal.Decimal }
	sums := make(map[string]*totals, len(settlements))
	seenInvoice := map[string]map[string]bool{}
	seenCustomer := map[string]map[string]bool{}
	for rows.Next() {
		var settlementID, typeCode, amount, invoiceNumber, customer string
		if err := rows.Scan(&settlementID, &typeCode, &amount, &invoiceNumber, &customer); err != nil {
			return db.MapSQLError(err)
		}
		s := byID[settlementID]
		s.AllocationCount++
		t := sums[settlementID]
		if t == nil {
			t = &totals{}
			sums[settlementID] = t
			seenInvoice[settlementID] = map[string]bool{}
			seenCustomer[settlementID] = map[string]bool{}
		}
		value, err := decimal.NewFromString(amount)
		if err != nil {
			return apierror.NewInternalError(err, "Allocation amount is not a number.")
		}
		switch typeCode {
		case "payment":
			t.payments = t.payments.Add(value)
		case "rebate":
			t.rebates = t.rebates.Add(value)
		case "adjustment":
			t.adjustments = t.adjustments.Add(value)
		case "credit_memo":
			t.credits = t.credits.Add(value)
		}
		if invoiceNumber != "" && !seenInvoice[settlementID][invoiceNumber] {
			seenInvoice[settlementID][invoiceNumber] = true
			s.InvoiceNumbers = append(s.InvoiceNumbers, invoiceNumber)
		}
		if !seenCustomer[settlementID][customer] {
			seenCustomer[settlementID][customer] = true
			s.CustomerNames = append(s.CustomerNames, customer)
		}
	}
	if apiErr := db.MapSQLError(rows.Err()); apiErr != nil {
		return apiErr
	}

	// A total of zero is shown as no total, as the dashboard did.
	for id, t := range sums {
		s := byID[id]
		s.TotalPayments = nonZeroDecimal(t.payments)
		s.TotalRebates = nonZeroDecimal(t.rebates)
		s.TotalAdjustments = nonZeroDecimal(t.adjustments)
		s.TotalCredits = nonZeroDecimal(t.credits)
	}
	return nil
}

func nonZeroDecimal(d decimal.Decimal) *string {
	if d.IsZero() {
		return nil
	}
	s := d.String()
	return &s
}

func (r *settlementRepoImpl) LockPaymentFlagRows(ctx context.Context, accountID string, transactionIDs, invoiceIDs []string) *apierror.APIError {
	ctx, span := settlementRepoTracer.Start(ctx, "repository.settlement.lock_payment_flag_rows")
	defer span.End()

	for _, lock := range []struct {
		table string
		ids   []string
	}{{"`transaction`", transactionIDs}, {"invoice", invoiceIDs}} {
		if len(lock.ids) == 0 {
			continue
		}
		rows, err := r.queries.DB().QueryContext(ctx, "SELECT id FROM "+lock.table+" WHERE account_id = ? AND id IN ("+
			placeholders(len(lock.ids))+") ORDER BY id FOR UPDATE", append([]any{accountID}, stringArgs(lock.ids)...)...)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
		_ = rows.Close()
	}
	return nil
}
