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
	"github.com/open-mrp/api/shared/safeconv"
	"github.com/open-mrp/api/shared/tracing"
)

// Transaction reads are assembled here rather than in sqlc: every list filter is optional, and a
// static query can only express that as `(flag = false OR col IN (...))`, which hides from the
// planner which index the request can use. Building the WHERE clause from the filters actually
// given lets an account's (account_id, created_at) or (account_id, customer_account_id, ...) index
// drive the read, and the keyset predicate below matches the ORDER BY exactly.

// transactionColumns reads one transaction with the names it is shown with. The customer relation is
// the account's customer relation only: a counterparty can hold several roles, and joining on the
// account pair alone returned the transaction once per role.
const transactionColumns = `
	t.id, t.number,
	q.id, CAST(q.value AS CHAR), q.unit_id, u.abbreviation,
	t.customer_account_id, ba.name, ar.external_number, ar.account_status_code, ar.commission_status_code, ar.created_at, ar.updated_at,
	t.responsible_user_id, au.id, usr.name, au.status_code, au.created_at, au.updated_at,
	t.note,
	tt.id, tt.code, tt.name,
	tm.id, tm.code, tm.name,
	at2.id, at2.code, at2.name,
	t.is_fully_allocated, t.stripe_payment_id, t.funds_received_at,
	(SELECT COUNT(*) FROM transaction_allocation ta WHERE ta.transaction_id = t.id),
	t.created_at, t.updated_at`

// The list's keys. Each per-filter key leads with the account and its filter and ends in list order,
// so a list on that filter reads a page and stops; transactionCreatedIndex is the one for no filter.
const (
	transactionCreatedIndex    = "transaction_account_id_created_at_idx"
	transactionStatusIndex     = "transaction_account_id_is_fully_allocated_created_at_id_idx"
	transactionTypeIndex       = "transaction_account_id_transaction_type_code_created_at_id_idx"
	transactionMethodIndex     = "transaction_account_id_transaction_method_code_created_at_id_idx"
	transactionAdjustmentIndex = "transaction_account_id_adjustment_type_code_created_at_id_idx"
	transactionCustomerIndex   = "transaction_account_id_customer_account_id_created_at_id_idx"
	// transactionFundsIndex ranges an account's transactions by when funds were received.
	transactionFundsIndex = "transaction_open_credits_idx"
)

// transactionListIndexes are every key that yields an account's transactions in list order.
var transactionListIndexes = []string{
	transactionCreatedIndex, transactionStatusIndex, transactionTypeIndex,
	transactionMethodIndex, transactionAdjustmentIndex, transactionCustomerIndex,
}

const transactionJoins = `
JOIN quantity q ON q.id = t.amount_id
-- LEFT, though every quantity has a unit and every type exists: an inner join to a tiny lookup lets
-- the planner drive from it, probing t once per lookup row instead of walking one index in order.
LEFT JOIN unit u ON u.id = q.unit_id
LEFT JOIN transaction_type tt ON tt.code = t.transaction_type_code
JOIN account ba ON ba.id = t.customer_account_id
LEFT JOIN account_relation ar ON ar.owner_account_id = t.account_id AND ar.counterparty_account_id = t.customer_account_id AND ar.account_relation_role_code = 'customer'
LEFT JOIN transaction_method tm ON tm.code = t.transaction_method_code
LEFT JOIN adjustment_type at2 ON at2.code = t.adjustment_type_code
-- responsible_user_id holds an account_user id, or a user id on rows the legacy dashboard wrote.
LEFT JOIN account_user au ON au.account_id = t.account_id AND (au.id = t.responsible_user_id OR au.user_id = t.responsible_user_id)
LEFT JOIN ` + "`user`" + ` usr ON usr.id = au.user_id`

type transactionScan struct {
	id, number                                                 string
	amountID, amountValue, amountUnitID, amountUnitAbbr        string
	customerID, customerName                                   string
	customerNumber, customerStatus, customerCommission         gosql.NullString
	customerCreatedAt, customerUpdatedAt                       gosql.NullTime
	responsibleUserID, responsibleAccountUserID                gosql.NullString
	responsibleName, responsibleStatus                         gosql.NullString
	responsibleCreatedAt, responsibleUpdatedAt                 gosql.NullTime
	note                                                       gosql.NullString
	typeID, typeCode, typeName                                 string
	methodID, methodCode, methodName                           gosql.NullString
	adjustmentID, adjustmentCode, adjustmentName, stripePaymID gosql.NullString
	isFullyAllocated                                           bool
	fundsReceivedAt                                            gosql.NullTime
	allocationCount                                            int32
	createdAt, updatedAt                                       time.Time
}

func (s *transactionScan) dest() []any {
	return []any{
		&s.id, &s.number,
		&s.amountID, &s.amountValue, &s.amountUnitID, &s.amountUnitAbbr,
		&s.customerID, &s.customerName, &s.customerNumber, &s.customerStatus, &s.customerCommission, &s.customerCreatedAt, &s.customerUpdatedAt,
		&s.responsibleUserID, &s.responsibleAccountUserID, &s.responsibleName, &s.responsibleStatus, &s.responsibleCreatedAt, &s.responsibleUpdatedAt,
		&s.note,
		&s.typeID, &s.typeCode, &s.typeName,
		&s.methodID, &s.methodCode, &s.methodName,
		&s.adjustmentID, &s.adjustmentCode, &s.adjustmentName,
		&s.isFullyAllocated, &s.stripePaymID, &s.fundsReceivedAt,
		&s.allocationCount,
		&s.createdAt, &s.updatedAt,
	}
}

func (s *transactionScan) transaction() *domain.Transaction {
	t := &domain.Transaction{
		ID:                        s.id,
		Number:                    s.number,
		AmountID:                  s.amountID,
		AmountValue:               s.amountValue,
		AmountUnitID:              s.amountUnitID,
		AmountUnitAbbr:            s.amountUnitAbbr,
		CustomerID:                &s.customerID,
		CustomerNumber:            nullStringPtr(s.customerNumber),
		CustomerStatusCode:        nullStringPtr(s.customerStatus),
		CustomerCommissionPolicy:  nullStringPtr(s.customerCommission),
		CustomerCreatedAt:         nullTimePtr(s.customerCreatedAt),
		CustomerUpdatedAt:         nullTimePtr(s.customerUpdatedAt),
		ResponsibleUserName:       nonEmptyPtr(s.responsibleName),
		ResponsibleUserStatusCode: nullStringPtr(s.responsibleStatus),
		ResponsibleUserCreatedAt:  nullTimePtr(s.responsibleCreatedAt),
		ResponsibleUserUpdatedAt:  nullTimePtr(s.responsibleUpdatedAt),
		Note:                      nullStringPtr(s.note),
		TransactionTypeID:         s.typeID,
		TransactionTypeCode:       s.typeCode,
		TransactionTypeName:       s.typeName,
		TransactionMethodID:       nullStringPtr(s.methodID),
		TransactionMethodCode:     nullStringPtr(s.methodCode),
		TransactionMethodName:     nullStringPtr(s.methodName),
		AdjustmentTypeID:          nullStringPtr(s.adjustmentID),
		AdjustmentTypeCode:        nullStringPtr(s.adjustmentCode),
		AdjustmentTypeName:        nullStringPtr(s.adjustmentName),
		IsFullyAllocated:          s.isFullyAllocated,
		StripePaymentID:           nullStringPtr(s.stripePaymID),
		FundsReceivedAt:           nullTimePtr(s.fundsReceivedAt),
		AllocationCount:           s.allocationCount,
		CreatedAt:                 s.createdAt,
		UpdatedAt:                 s.updatedAt,
	}
	if s.customerName != "" {
		t.CustomerName = &s.customerName
	}
	// Prefer the resolved account_user id; a legacy row may name a user with no account_user match.
	if s.responsibleAccountUserID.Valid {
		t.ResponsibleUserID = &s.responsibleAccountUserID.String
	} else {
		t.ResponsibleUserID = nullStringPtr(s.responsibleUserID)
	}
	return t
}

func transactionSummaryOf(t *domain.Transaction) *domain.TransactionSummary {
	return &domain.TransactionSummary{
		ID:                       t.ID,
		Number:                   t.Number,
		AmountID:                 t.AmountID,
		AmountValue:              t.AmountValue,
		AmountUnitID:             t.AmountUnitID,
		AmountUnitAbbr:           t.AmountUnitAbbr,
		CustomerID:               t.CustomerID,
		CustomerName:             t.CustomerName,
		CustomerNumber:           t.CustomerNumber,
		CustomerStatusCode:       t.CustomerStatusCode,
		CustomerCommissionPolicy: t.CustomerCommissionPolicy,
		CustomerCreatedAt:        t.CustomerCreatedAt,
		CustomerUpdatedAt:        t.CustomerUpdatedAt,
		TransactionTypeCode:      t.TransactionTypeCode,
		TransactionTypeName:      t.TransactionTypeName,
		TransactionTypeID:        t.TransactionTypeID,
		TransactionMethodCode:    t.TransactionMethodCode,
		TransactionMethodName:    t.TransactionMethodName,
		TransactionMethodID:      t.TransactionMethodID,
		AdjustmentTypeCode:       t.AdjustmentTypeCode,
		AdjustmentTypeName:       t.AdjustmentTypeName,
		AdjustmentTypeID:         t.AdjustmentTypeID,
		IsFullyAllocated:         t.IsFullyAllocated,
		FundsReceivedAt:          t.FundsReceivedAt,
		AllocationCount:          t.AllocationCount,
		CreatedAt:                t.CreatedAt,
		UpdatedAt:                t.UpdatedAt,
	}
}

// transactionFilter accumulates a WHERE clause and its arguments.
type transactionFilter struct {
	where []string
	args  []any
	// indexes, when set, are FORCE INDEX'd on the transaction table the page is chosen from.
	indexes []string
}

func (f *transactionFilter) add(clause string, args ...any) {
	f.where = append(f.where, clause)
	f.args = append(f.args, args...)
}

func (f *transactionFilter) in(column string, values []string) {
	if len(values) == 0 {
		return
	}
	f.add(column+" IN ("+placeholders(len(values))+")", stringArgs(values)...)
}

// search matches transaction numbers the way the dashboard did: every word of the query must begin
// a word of the number.
func (f *transactionFilter) search(query *string) {
	if term := db.AllWordsPrefixQuery(query); term != "" {
		f.add("MATCH(t.number) AGAINST(? IN BOOLEAN MODE)", term)
	}
}

func (f *transactionFilter) status(status *string) {
	if status == nil {
		return
	}
	switch *status {
	case "allocated":
		f.add("t.is_fully_allocated = 1")
	case "unallocated":
		f.add("t.is_fully_allocated = 0")
	}
}

// transactionKeyFilter is an equality filter with a list-order key of its own.
type transactionKeyFilter struct {
	index, column string
	values        []string
}

// transactionListIndexHint is the keys a list may be read from, or none to leave the choice to the
// planner. Left to itself, the planner reaches for the number key, merges single-column keys, or
// walks created_at past every row a filter rejects, and sorts every match.
//   - A number search is answered by its FULLTEXT key, which no hint can name; it reads every match.
//   - A funds-received range filters a column the list does not sort by, so no key can stop at a
//     page. The funds key reads just the range (status included, which it leads with). The customer
//     key is offered too, for a customer narrower than the range; a type, method, or adjustment key
//     is not, because walking a common one in list order reads past every row outside the range.
//   - Otherwise each single-valued filter's key both narrows and orders, and the planner picks among
//     them. The created_at key is offered only alone: forced beside another, the planner may swap to
//     it for the order and walk it from the account's newest row, past a deep page's cursor.
//   - A multi-valued filter has no key that yields its values in list order, and the planner cannot
//     tell a rare one from a common one. Its presence returns every filter as counted, for
//     transactionCountedHint to settle by counting; indexes is then the fallback for when all are common.
func transactionListIndexHint(params domain.ListTransactionsParams, customerIDs []string) (indexes []string, counted []transactionKeyFilter) {
	if db.AllWordsPrefixQuery(params.Query) != "" {
		return nil, nil
	}
	if params.StartDate != nil || params.EndDate != nil {
		if len(customerIDs) > 0 {
			return []string{transactionFundsIndex, transactionCustomerIndex}, nil
		}
		return []string{transactionFundsIndex}, nil
	}
	var filters []transactionKeyFilter
	if params.Status != nil && (*params.Status == "allocated" || *params.Status == "unallocated") {
		allocated := "0"
		if *params.Status == "allocated" {
			allocated = "1"
		}
		filters = append(filters, transactionKeyFilter{transactionStatusIndex, "is_fully_allocated", []string{allocated}})
	}
	for _, f := range []transactionKeyFilter{
		{transactionTypeIndex, "transaction_type_code", params.TypeCodes},
		{transactionMethodIndex, "transaction_method_code", params.MethodCodes},
		{transactionAdjustmentIndex, "adjustment_type_code", params.AdjustmentTypeCodes},
		{transactionCustomerIndex, "customer_account_id", customerIDs},
	} {
		if len(f.values) > 0 {
			filters = append(filters, f)
		}
	}
	multiValued := false
	for _, f := range filters {
		if len(f.values) == 1 {
			indexes = append(indexes, f.index)
		} else {
			multiValued = true
		}
	}
	if len(indexes) == 0 {
		indexes = []string{transactionCreatedIndex}
	}
	if multiValued {
		return indexes, filters
	}
	return indexes, nil
}

// transactionCountedRatio sets how many matches, in pages, make a filter common.
const transactionCountedRatio = 40

// transactionCountedHint picks the key of the filter matching the fewest rows: a single-valued one's
// key stops at the page, a multi-valued one's is read whole and sorted, and either reads no more than
// the filter matches. When every filter matches many rows, they are common enough that fallback (the
// single-valued keys, or created_at) finds a page quickly in list order. Each count is capped, reading
// at most that many index entries.
func (r *transactionRepoImpl) transactionCountedHint(ctx context.Context, accountID string, limit int32, filters []transactionKeyFilter, fallback []string) ([]string, error) {
	capped := int64(transactionCountedRatio * (limit + 1))
	best, bestCount := "", capped
	for _, f := range filters {
		args := append(append([]any{accountID}, stringArgs(f.values)...), capped)
		var n int64
		err := r.queries.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM (SELECT 1 FROM `transaction` FORCE INDEX ("+f.index+
			") WHERE account_id = ? AND "+f.column+" IN ("+placeholders(len(f.values))+") LIMIT ?) matches", args...).Scan(&n)
		if err != nil {
			return nil, err
		}
		if n < bestCount {
			best, bestCount = f.index, n
		}
	}
	if best == "" {
		return fallback, nil
	}
	return []string{best}, nil
}

// page applies the keyset for a cursor and returns the ORDER BY. Rows are read newest first; a
// backward page reads the rows after the cursor oldest first, and BuildPageString reverses them.
func (f *transactionFilter) page(cursor *pagination.StringCursor) string {
	if cursor == nil {
		return "t.created_at DESC, t.id DESC"
	}
	if cursor.Direction == pagination.DirectionBackward {
		f.add("(t.created_at > ? OR (t.created_at = ? AND t.id > ?))", cursor.OccurredAt, cursor.OccurredAt, cursor.ID)
		return "t.created_at ASC, t.id ASC"
	}
	f.add("(t.created_at < ? OR (t.created_at = ? AND t.id < ?))", cursor.OccurredAt, cursor.OccurredAt, cursor.ID)
	return "t.created_at DESC, t.id DESC"
}

func (r *transactionRepoImpl) queryTransactions(ctx context.Context, f *transactionFilter, orderBy string, limit int32) ([]*domain.Transaction, error) {
	// The page is chosen from the transaction table alone and joined after: a filter no key serves in
	// list order reads every match, and joining each one (account_user alone fans out to every user of
	// the account) before the sort multiplied that by the joins.
	var sb strings.Builder
	sb.WriteString("SELECT")
	sb.WriteString(transactionColumns)
	sb.WriteString("\nFROM (SELECT t.id FROM `transaction` t")
	if len(f.indexes) > 0 {
		sb.WriteString(" FORCE INDEX (" + strings.Join(f.indexes, ", ") + ")")
	}
	if len(f.where) > 0 {
		sb.WriteString("\nWHERE ")
		sb.WriteString(strings.Join(f.where, "\nAND "))
	}
	args := f.args
	if orderBy != "" {
		sb.WriteString("\nORDER BY " + orderBy)
	}
	if limit > 0 {
		sb.WriteString("\nLIMIT ?")
		args = append(args, limit)
	}
	sb.WriteString(") page\nJOIN `transaction` t ON t.id = page.id")
	sb.WriteString(transactionJoins)
	if orderBy != "" {
		sb.WriteString("\nORDER BY " + orderBy)
	}

	rows, err := r.queries.DB().QueryContext(ctx, sb.String(), args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []*domain.Transaction
	for rows.Next() {
		var s transactionScan
		if err := rows.Scan(s.dest()...); err != nil {
			return nil, err
		}
		out = append(out, s.transaction())
	}
	return out, rows.Err()
}

func decodeTransactionCursor(cursor *string) (*pagination.StringCursor, *apierror.APIError) {
	if cursor == nil {
		return nil, nil
	}
	cur, err := pagination.DecodeStringCursor(*cursor)
	if err != nil {
		return nil, apierror.NewValidationErrorWithParam("Invalid pagination cursor.", "cursor")
	}
	return &cur, nil
}

func (r *transactionRepoImpl) List(ctx context.Context, params domain.ListTransactionsParams) (*domain.ListTransactionsResult, *apierror.APIError) {
	ctx, span := transactionRepoTracer.Start(ctx, "repository.transaction.list")
	defer span.End()

	cur, apiErr := decodeTransactionCursor(params.Cursor)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	f := &transactionFilter{}
	f.add("t.account_id = ?", params.AccountID)
	f.search(params.Query)
	f.status(params.Status)
	f.in("t.transaction_type_code", params.TypeCodes)
	f.in("t.adjustment_type_code", params.AdjustmentTypeCodes)
	f.in("t.transaction_method_code", params.MethodCodes)
	customerIDs := params.CustomerIDs
	if len(params.CustomerGroupIDs) > 0 {
		// Resolved up front so the customer key serves it; filtering on the joined relation left the
		// planner to drive from account_relation and read every transaction of the group's customers.
		groupCustomerIDs, err := r.customersInGroups(ctx, params.AccountID, params.CustomerGroupIDs)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if len(customerIDs) > 0 {
			groupCustomerIDs = intersectStrings(customerIDs, groupCustomerIDs)
		}
		if len(groupCustomerIDs) == 0 {
			return &domain.ListTransactionsResult{Transactions: []*domain.TransactionSummary{}, PageInfo: pagination.PageInfo{}}, nil
		}
		customerIDs = groupCustomerIDs
	}
	f.in("t.customer_account_id", customerIDs)
	if params.StartDate != nil || params.EndDate != nil {
		if params.Status == nil || (*params.Status != "allocated" && *params.Status != "unallocated") {
			// Both statuses, spelled out so the funds key, which leads with status, can range the dates.
			f.add("t.is_fully_allocated IN (0, 1)")
		}
		if params.StartDate != nil {
			f.add("t.funds_received_at >= ?", *params.StartDate)
		}
		if params.EndDate != nil {
			f.add("t.funds_received_at <= ?", *params.EndDate)
		}
	}
	indexes, counted := transactionListIndexHint(params, customerIDs)
	if len(counted) > 0 {
		var err error
		if indexes, err = r.transactionCountedHint(ctx, params.AccountID, params.Limit, counted, indexes); err != nil {
			return nil, tracing.Trace(span, db.MapSQLError(err))
		}
	}
	f.indexes = indexes
	orderBy := f.page(cur)

	rows, err := r.queryTransactions(ctx, f, orderBy, params.Limit+1)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	summaries := make([]*domain.TransactionSummary, len(rows))
	for i, t := range rows {
		summaries[i] = transactionSummaryOf(t)
	}
	result, pageInfo := pagination.BuildPageString(summaries, params.Limit, cursorDirection(cur), transactionCreatedAt, transactionID)
	return &domain.ListTransactionsResult{Transactions: result, PageInfo: pageInfo}, nil
}

func (r *transactionRepoImpl) Get(ctx context.Context, accountID, transactionID string) (*domain.Transaction, *apierror.APIError) {
	ctx, span := transactionRepoTracer.Start(ctx, "repository.transaction.get")
	defer span.End()

	f := &transactionFilter{}
	f.add("t.id = ?", transactionID)
	f.add("t.account_id = ?", accountID)
	rows, err := r.queryTransactions(ctx, f, "", 1)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if len(rows) == 0 {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Transaction not found."))
	}
	return rows[0], nil
}

func (r *transactionRepoImpl) GetByIDs(ctx context.Context, accountID string, transactionIDs []string) ([]*domain.Transaction, *apierror.APIError) {
	ctx, span := transactionRepoTracer.Start(ctx, "repository.transaction.get_by_ids")
	defer span.End()

	if len(transactionIDs) == 0 {
		return nil, nil
	}
	f := &transactionFilter{}
	f.add("t.account_id = ?", accountID)
	f.in("t.id", transactionIDs)
	rows, err := r.queryTransactions(ctx, f, "", safeconv.IntToInt32(len(transactionIDs)))
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return rows, nil
}

// ListByCustomer lists a customer's transactions together with those of the customer's direct child
// accounts, as the dashboard's settle flow reads them.
func (r *transactionRepoImpl) ListByCustomer(ctx context.Context, params domain.ListAccountTransactionsParams) (*domain.ListAccountTransactionsResult, *apierror.APIError) {
	ctx, span := transactionRepoTracer.Start(ctx, "repository.transaction.list_by_customer")
	defer span.End()

	cur, apiErr := decodeTransactionCursor(params.Cursor)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Resolved up front so the read below is an IN list on (account_id, customer_account_id, ...)
	// rather than a subquery evaluated against every transaction in the account.
	customerIDs, err := r.customerWithChildren(ctx, params.AccountID, params.CustomerAccountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	f := &transactionFilter{}
	f.add("t.account_id = ?", params.AccountID)
	f.in("t.customer_account_id", customerIDs)
	f.search(params.Query)
	f.status(params.Status)
	if params.Status != nil && *params.Status == "unallocated" {
		f.add("t.funds_received_at IS NOT NULL")
	}
	if params.Type != nil {
		f.add("t.transaction_type_code = ?", *params.Type)
	}
	f.indexes = accountTransactionIndexHint(params)
	orderBy := f.page(cur)

	rows, err := r.queryTransactions(ctx, f, orderBy, params.Limit+1)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	result, pageInfo := pagination.BuildPageString(rows, params.Limit, cursorDirection(cur), accountTransactionCreatedAt, accountTransactionID)

	if params.WithAllocations && len(result) > 0 {
		ids := make([]string, len(result))
		for i, t := range result {
			ids[i] = t.ID
		}
		byTransaction, apiErr := r.allocationsFor(ctx, ids)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		for _, t := range result {
			t.Allocations = byTransaction[t.ID]
		}
	}
	return &domain.ListAccountTransactionsResult{Transactions: result, PageInfo: pageInfo}, nil
}

// accountTransactionIndexHint is the customer key plus the key of each other filter given; unhinted,
// the planner sorts every match from the single-column customer key. The created_at key is left out:
// forced beside another, it is walked from the newest row past a deep page's cursor.
func accountTransactionIndexHint(params domain.ListAccountTransactionsParams) []string {
	if db.AllWordsPrefixQuery(params.Query) != "" {
		return nil
	}
	indexes := []string{transactionCustomerIndex}
	if params.Status != nil && (*params.Status == "allocated" || *params.Status == "unallocated") {
		indexes = append(indexes, transactionStatusIndex)
	}
	if params.Type != nil {
		indexes = append(indexes, transactionTypeIndex)
	}
	return indexes
}

// customersInGroups is the account's customers in any of groupIDs.
func (r *transactionRepoImpl) customersInGroups(ctx context.Context, accountID string, groupIDs []string) ([]string, error) {
	args := append([]any{accountID}, stringArgs(groupIDs)...)
	rows, err := r.queries.DB().QueryContext(ctx, `
SELECT counterparty_account_id
FROM account_relation
WHERE owner_account_id = ?
AND account_relation_role_code = 'customer'
AND account_group_id IN (`+placeholders(len(groupIDs))+`)`, args...)
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

// customerWithChildren returns the customer and its direct child customer accounts.
func (r *transactionRepoImpl) customerWithChildren(ctx context.Context, accountID, customerID string) ([]string, error) {
	rows, err := r.queries.DB().QueryContext(ctx, `
SELECT child.counterparty_account_id
FROM account_relation parent
JOIN account_relation child ON child.parent_account_relation_id = parent.id
	AND child.owner_account_id = parent.owner_account_id
	AND child.account_relation_role_code = 'customer'
WHERE parent.owner_account_id = ?
AND parent.counterparty_account_id = ?
AND parent.account_relation_role_code = 'customer'`, accountID, customerID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	ids := []string{customerID}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// allocationColumns reads an allocation together with what its transaction, invoice and settlement
// are shown with.
const allocationColumns = `
SELECT ta.id, taq.id, CAST(taq.value AS CHAR), taq.unit_id, tau.abbreviation, ta.note,
	ta.transaction_id, t.number, t.transaction_type_code, t.transaction_method_code, t.adjustment_type_code,
	t.customer_account_id, t.created_at,
	ta.invoice_id, COALESCE(i.number, ''),
	ta.settlement_id, s.number,
	ta.created_at, ta.updated_at
FROM transaction_allocation ta
JOIN quantity taq ON taq.id = ta.amount_id
JOIN unit tau ON tau.id = taq.unit_id
JOIN ` + "`transaction`" + ` t ON t.id = ta.transaction_id
LEFT JOIN invoice i ON i.id = ta.invoice_id
LEFT JOIN settlement s ON s.id = ta.settlement_id`

func scanAllocations(rows *gosql.Rows) ([]*domain.TransactionAllocation, error) {
	defer func() { _ = rows.Close() }()
	var out []*domain.TransactionAllocation
	for rows.Next() {
		var a domain.TransactionAllocation
		var note, method, adjustment, settlementID, settlementNumber gosql.NullString
		if err := rows.Scan(
			&a.ID, &a.AmountID, &a.AmountValue, &a.AmountUnitID, &a.AmountUnitAbbr, &note,
			&a.TransactionID, &a.TransactionNumber, &a.TransactionType, &method, &adjustment,
			&a.TransactionCustomerID, &a.TransactionCreatedAt,
			&a.InvoiceID, &a.InvoiceNumber,
			&settlementID, &settlementNumber,
			&a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, err
		}
		a.Note = nullStringPtr(note)
		a.TransactionMethodCode = nullStringPtr(method)
		a.TransactionAdjustmentType = nullStringPtr(adjustment)
		a.SettlementID = nullStringPtr(settlementID)
		a.SettlementNumber = nullStringPtr(settlementNumber)
		out = append(out, &a)
	}
	return out, rows.Err()
}

func (r *transactionRepoImpl) GetAllocations(ctx context.Context, transactionID string) ([]*domain.TransactionAllocation, *apierror.APIError) {
	ctx, span := transactionRepoTracer.Start(ctx, "repository.transaction.get_allocations")
	defer span.End()

	byTransaction, apiErr := r.allocationsFor(ctx, []string{transactionID})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return byTransaction[transactionID], nil
}

// allocationsFor loads the allocations drawn from each of the given transactions, newest first.
func (r *transactionRepoImpl) allocationsFor(ctx context.Context, transactionIDs []string) (map[string][]*domain.TransactionAllocation, *apierror.APIError) {
	allocations, apiErr := loadAllocations(ctx, r.queries.DB(), "ta.transaction_id IN ("+placeholders(len(transactionIDs))+")", "ta.created_at DESC, ta.id DESC", stringArgs(transactionIDs)...)
	if apiErr != nil {
		return nil, apiErr
	}
	byTransaction := make(map[string][]*domain.TransactionAllocation, len(transactionIDs))
	for _, a := range allocations {
		byTransaction[a.TransactionID] = append(byTransaction[a.TransactionID], a)
	}
	return byTransaction, nil
}

// loadAllocations reads the allocations matching where, with what their transaction, invoice and
// settlement are shown with.
func loadAllocations(ctx context.Context, q sqlc.DBTX, where, orderBy string, args ...any) ([]*domain.TransactionAllocation, *apierror.APIError) {
	rows, err := q.QueryContext(ctx, allocationColumns+"\nWHERE "+where+"\nORDER BY "+orderBy, args...)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	allocations, err := scanAllocations(rows)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	return allocations, nil
}

func nonEmptyPtr(v gosql.NullString) *string {
	if !v.Valid || v.String == "" {
		return nil
	}
	return &v.String
}

func stringArgs(values []string) []any {
	args := make([]any, len(values))
	for i, v := range values {
		args[i] = v
	}
	return args
}

// CustomerExists reports whether the counterparty is one of the account's customers.
func (r *transactionRepoImpl) CustomerExists(ctx context.Context, accountID, customerID string) (bool, *apierror.APIError) {
	var n int
	err := r.queries.DB().QueryRowContext(ctx, `
SELECT COUNT(*) FROM account_relation
WHERE owner_account_id = ? AND counterparty_account_id = ? AND account_relation_role_code = 'customer'`,
		accountID, customerID).Scan(&n)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, apiErr
	}
	return n > 0, nil
}
