package repository

import (
	"context"
	gosql "database/sql"
	"strings"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/pagination"
	"github.com/open-mrp/api/shared/tracing"
)

// The entry list's keys, in list order. Only one is ever forced: beside another, the planner may walk
// it from the newest row past a deep page's cursor.
const (
	allocationCreatedIndex = "transaction_allocation_account_created_idx"
	allocationTypeIndex    = "transaction_allocation_account_type_created_idx"
	// allocationTransactionIndex and allocationInvoiceIndex find a resolved search's allocations.
	allocationTransactionIndex = "transaction_allocation_transaction_id_created_at_id_idx"
	allocationInvoiceIndex     = "transaction_allocation_invoice_id_idx"
)

// allocationSearchResolveLimit is the most transactions a search resolves to; a commoner search finds a
// page sooner by walking the list and matching row by row.
var allocationSearchResolveLimit = 1000

// allocationSearchJoins are what the dashboard's search reads, for a search matched row by row.
const allocationSearchJoins = `
JOIN invoice inv ON inv.id = ta.invoice_id
JOIN ` + "`transaction`" + ` t ON t.id = ta.transaction_id
JOIN account cust_acct ON cust_acct.id = t.customer_account_id
LEFT JOIN account_relation ar ON ar.counterparty_account_id = t.customer_account_id
    AND ar.owner_account_id = t.account_id
    AND ar.account_relation_role_code = 'customer'`

// The dashboard found an entry by its invoice or transaction number, or by its customer.
const allocationSearchMatch = `(inv.number = ? OR t.number = ? OR ar.external_number = ? OR cust_acct.name LIKE CONCAT('%', ?, '%'))`

const allocationEntryColumns = `
SELECT ta.id, t.note, ta.created_at, q.value, qu.abbreviation,
	t.customer_account_id, cust_acct.name, ar.external_number,
	t.id, t.transaction_type_code, t.transaction_method_code, t.adjustment_type_code,
	inv.id, inv.number`

func allocationEntryCreatedAt(d *domain.AllocationEntry) time.Time { return d.CreatedAt }
func allocationEntryID(d *domain.AllocationEntry) string           { return d.ID }

// ListEntries pages the account's allocations, newest first, choosing the page before joining it.
func (r *transactionAllocationRepoImpl) ListEntries(ctx context.Context, params domain.ListAllocationEntriesParams) (*domain.ListAllocationEntriesResult, *apierror.APIError) {
	ctx, span := transactionAllocationRepoTracer.Start(ctx, "repository.transaction_allocation.list_entries")
	defer span.End()

	cur, apiErr := decodeTransactionCursor(params.Cursor)
	if apiErr != nil {
		return nil, apiErr
	}

	f := &transactionFilter{}
	f.add("ta.account_id = ?", params.AccountID)
	f.indexes = []string{allocationCreatedIndex}
	if params.TransactionType != nil {
		f.add("ta.transaction_type_code = ?", *params.TransactionType)
		f.indexes = []string{allocationTypeIndex}
	}
	if params.StartDate != nil {
		f.add("ta.created_at >= ?", *params.StartDate)
	}
	if params.EndDate != nil {
		f.add("ta.created_at <= ?", *params.EndDate)
	}

	var joins string
	if q := strings.TrimSpace(derefString(params.Query)); q != "" {
		search, err := r.resolveAllocationSearch(ctx, params.AccountID, q)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		switch {
		case search == nil:
			joins = allocationSearchJoins
			f.add(allocationSearchMatch, q, q, q, db.EscapeLike(q))
		case len(search.transactionIDs) == 0 && len(search.invoiceIDs) == 0:
			return &domain.ListAllocationEntriesResult{Entries: []*domain.AllocationEntry{}, PageInfo: pagination.PageInfo{}}, nil
		default:
			// The transaction and invoice keys hold only the matches; a list key would walk the account.
			var match []string
			f.indexes = nil
			if len(search.transactionIDs) > 0 {
				match = append(match, "ta.transaction_id IN ("+placeholders(len(search.transactionIDs))+")")
				f.args = append(f.args, stringArgs(search.transactionIDs)...)
				f.indexes = append(f.indexes, allocationTransactionIndex)
			}
			if len(search.invoiceIDs) > 0 {
				match = append(match, "ta.invoice_id IN ("+placeholders(len(search.invoiceIDs))+")")
				f.args = append(f.args, stringArgs(search.invoiceIDs)...)
				f.indexes = append(f.indexes, allocationInvoiceIndex)
			}
			f.where = append(f.where, "("+strings.Join(match, " OR ")+")")
		}
	}

	orderBy := "ta.created_at DESC, ta.id DESC"
	if cur != nil {
		if cur.Direction == pagination.DirectionBackward {
			f.add("(ta.created_at > ? OR (ta.created_at = ? AND ta.id > ?))", cur.OccurredAt, cur.OccurredAt, cur.ID)
			orderBy = "ta.created_at ASC, ta.id ASC"
		} else {
			f.add("(ta.created_at < ? OR (ta.created_at = ? AND ta.id < ?))", cur.OccurredAt, cur.OccurredAt, cur.ID)
		}
	}

	entries, err := r.queryAllocationEntries(ctx, f, joins, orderBy, params.Limit+1)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	result, pageInfo := pagination.BuildPageString(entries, params.Limit, cursorDirection(cur), allocationEntryCreatedAt, allocationEntryID)
	return &domain.ListAllocationEntriesResult{Entries: result, PageInfo: pageInfo}, nil
}

func (r *transactionAllocationRepoImpl) queryAllocationEntries(ctx context.Context, f *transactionFilter, joins, orderBy string, limit int32) ([]*domain.AllocationEntry, error) {
	var sb strings.Builder
	sb.WriteString(allocationEntryColumns)
	sb.WriteString("\nFROM (SELECT ta.id FROM transaction_allocation ta")
	if len(f.indexes) > 0 {
		sb.WriteString(" FORCE INDEX (" + strings.Join(f.indexes, ", ") + ")")
	}
	sb.WriteString(joins)
	sb.WriteString("\nWHERE " + strings.Join(f.where, "\nAND "))
	sb.WriteString("\nORDER BY " + orderBy + "\nLIMIT ?")
	sb.WriteString(`) page
JOIN transaction_allocation ta ON ta.id = page.id
JOIN quantity q ON q.id = ta.amount_id
JOIN unit qu ON qu.id = q.unit_id
JOIN ` + "`transaction`" + ` t ON t.id = ta.transaction_id
JOIN invoice inv ON inv.id = ta.invoice_id
JOIN account cust_acct ON cust_acct.id = t.customer_account_id
LEFT JOIN account_relation ar ON ar.counterparty_account_id = t.customer_account_id
    AND ar.owner_account_id = t.account_id
    AND ar.account_relation_role_code = 'customer'
ORDER BY ` + orderBy)

	rows, err := r.queries.DB().QueryContext(ctx, sb.String(), append(f.args, limit)...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []*domain.AllocationEntry
	for rows.Next() {
		var (
			e                                    domain.AllocationEntry
			note, custNumber, method, adjustment gosql.NullString
		)
		if err := rows.Scan(&e.ID, &note, &e.CreatedAt, &e.AmountValue, &e.AmountUnitAbbr,
			&e.CustomerID, &e.CustomerName, &custNumber,
			&e.TransactionID, &e.TransactionType, &method, &adjustment,
			&e.InvoiceID, &e.InvoiceNumber); err != nil {
			return nil, err
		}
		e.Note, e.CustomerNumber = nullStringPtr(note), nullStringPtr(custNumber)
		e.TransactionMethod, e.AdjustmentType = nullStringPtr(method), nullStringPtr(adjustment)
		out = append(out, &e)
	}
	return out, rows.Err()
}

// allocationSearch is what a search matches: allocations of these transactions or to these invoices.
type allocationSearch struct {
	transactionIDs, invoiceIDs []string
}

// resolveAllocationSearch finds the invoices and transactions q names, matching customer names against
// the account's few thousand customers rather than every allocation. It returns nil past
// allocationSearchResolveLimit. Invoices are the account's: a settlement allocates to no other.
func (r *transactionAllocationRepoImpl) resolveAllocationSearch(ctx context.Context, accountID, q string) (*allocationSearch, error) {
	dbtx := r.queries.DB()
	var search allocationSearch
	var err error

	search.invoiceIDs, err = queryStrings(ctx, dbtx, "SELECT id FROM invoice WHERE account_id = ? AND number = ?", accountID, q)
	if err != nil {
		return nil, err
	}
	customers, err := queryStrings(ctx, dbtx, `
SELECT c.customer_account_id
FROM (SELECT DISTINCT customer_account_id FROM `+"`transaction`"+` WHERE account_id = ?) c
JOIN account cust_acct ON cust_acct.id = c.customer_account_id
LEFT JOIN account_relation ar ON ar.counterparty_account_id = c.customer_account_id
    AND ar.owner_account_id = ?
    AND ar.account_relation_role_code = 'customer'
WHERE ar.external_number = ? OR cust_acct.name LIKE CONCAT('%', ?, '%')`, accountID, accountID, q, db.EscapeLike(q))
	if err != nil {
		return nil, err
	}

	query := "SELECT id FROM `transaction` WHERE account_id = ? AND number = ?"
	args := []any{accountID, q}
	if len(customers) > 0 {
		query += "\nUNION\nSELECT id FROM `transaction` WHERE account_id = ? AND customer_account_id IN (" + placeholders(len(customers)) + ")"
		args = append(append(args, accountID), stringArgs(customers)...)
	}
	search.transactionIDs, err = queryStrings(ctx, dbtx, query+"\nLIMIT ?", append(args, allocationSearchResolveLimit+1)...)
	if err != nil {
		return nil, err
	}
	if len(search.transactionIDs) > allocationSearchResolveLimit {
		return nil, nil
	}
	return &search, nil
}

// queryStrings reads one string column.
func queryStrings(ctx context.Context, dbtx sqlc.DBTX, query string, args ...any) ([]string, error) {
	rows, err := dbtx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
