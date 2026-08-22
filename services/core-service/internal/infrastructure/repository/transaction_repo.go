package repository

import (
	"context"
	gosql "database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/pagination"
	"github.com/open-mrp/api/shared/safeconv"
	"github.com/open-mrp/api/shared/tracing"
)

var transactionRepoTracer = tracing.GetTracer("core-service.infrastructure.repository.transaction")

type transactionRepoImpl struct {
	queries *sqlc.Queries
}

func NewTransactionRepo(queries *sqlc.Queries) domain.TransactionRepo {
	return &transactionRepoImpl{queries: queries}
}

func (r *transactionRepoImpl) Create(
	ctx context.Context,
	txID, number, typeCode, accountID, customerAccountID string,
	stripePaymentID *string, methodCode *string, adjustmentTypeCode *string, responsibleUserID *string, note *string,
	amountValue string, amountUnitID string,
) *apierror.APIError {
	ctx, span := transactionRepoTracer.Start(ctx, "repository.transaction.create")
	defer span.End()

	// Create the quantity record for the transaction amount.
	quantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	err := r.queries.InsertTransactionQuantity(ctx, sqlc.InsertTransactionQuantityParams{
		ID:     quantityID,
		Value:  amountValue,
		UnitID: amountUnitID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// Create the transaction record.
	err = r.queries.InsertTransaction(ctx, sqlc.InsertTransactionParams{
		ID:                    txID,
		Number:                number,
		TransactionTypeCode:   typeCode,
		StripePaymentID:       toNullString(stripePaymentID),
		CustomerAccountID:     customerAccountID,
		AccountID:             accountID,
		TransactionMethodCode: toNullString(methodCode),
		AdjustmentTypeCode:    toNullString(adjustmentTypeCode),
		ResponsibleUserID:     toNullString(responsibleUserID),
		Note:                  toNullString(note),
		AmountID:              quantityID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *transactionRepoImpl) FindByStripePaymentID(ctx context.Context, stripePaymentID string) (*domain.TransactionRecord, *apierror.APIError) {
	ctx, span := transactionRepoTracer.Start(ctx, "repository.transaction.find_by_stripe_payment_id")
	defer span.End()

	row, err := r.queries.FindTransactionByStripePaymentID(ctx, gosql.NullString{String: stripePaymentID, Valid: true})
	if errors.Is(err, gosql.ErrNoRows) {
		// A payment intent with no recorded transaction is the caller's normal existence-check case, not an error.
		return nil, nil
	}
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return &domain.TransactionRecord{
		ID:       row.ID,
		Number:   row.Number,
		AmountID: row.AmountID,
	}, nil
}

func (r *transactionRepoImpl) UpdateFundsReceivedByStripePaymentIDs(ctx context.Context, accountID string, stripePaymentIDs []string, fundsReceivedAt time.Time) *apierror.APIError {
	ctx, span := transactionRepoTracer.Start(ctx, "repository.transaction.update_funds_received_by_stripe_payment_ids")
	defer span.End()

	if len(stripePaymentIDs) == 0 {
		return nil
	}

	err := r.queries.UpdateTransactionFundsReceivedByStripePaymentIDs(ctx, sqlc.UpdateTransactionFundsReceivedByStripePaymentIDsParams{
		FundsReceivedAt: gosql.NullTime{Time: fundsReceivedAt, Valid: true},
		AccountID:       accountID,
		StripePaymentIds: func() []gosql.NullString {
			out := make([]gosql.NullString, len(stripePaymentIDs))
			for i, id := range stripePaymentIDs {
				out[i] = gosql.NullString{String: id, Valid: true}
			}
			return out
		}(),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *transactionRepoImpl) UpdateNote(ctx context.Context, txID, note string) *apierror.APIError {
	ctx, span := transactionRepoTracer.Start(ctx, "repository.transaction.update_note")
	defer span.End()

	err := r.queries.UpdateTransactionNote(ctx, sqlc.UpdateTransactionNoteParams{
		ID:   txID,
		Note: gosql.NullString{String: note, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *transactionRepoImpl) Delete(ctx context.Context, txID string) *apierror.APIError {
	ctx, span := transactionRepoTracer.Start(ctx, "repository.transaction.delete")
	defer span.End()

	err := r.queries.DeleteTransaction(ctx, txID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *transactionRepoImpl) DeleteAllocations(ctx context.Context, transactionID string) *apierror.APIError {
	ctx, span := transactionRepoTracer.Start(ctx, "repository.transaction.delete_allocations")
	defer span.End()

	err := r.queries.DeleteTransactionAllocationsByTransactionID(ctx, transactionID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *transactionRepoImpl) DeleteQuantity(ctx context.Context, quantityID string) *apierror.APIError {
	ctx, span := transactionRepoTracer.Start(ctx, "repository.transaction.delete_quantity")
	defer span.End()

	err := r.queries.DeleteTransactionQuantity(ctx, quantityID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *transactionRepoImpl) FetchAndIncrementTransactionNumber(ctx context.Context, accountID string) (string, *apierror.APIError) {
	ctx, span := transactionRepoTracer.Start(ctx, "repository.transaction.fetch_and_increment_number")
	defer span.End()

	sysPropertyID, apiErr := id.GenID(id.SysPropertyIDPrefix, nil)
	if apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	// One statement reserves the number under a row lock. The previous read-then-write pair
	// let two payments recorded at the same moment be assigned the same number, and the
	// duplicate check that followed could not see a number the other request had not written yet.
	res, err := r.queries.AllocateNextTransactionNumber(ctx, sqlc.AllocateNextTransactionNumberParams{
		ID:        sysPropertyID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	number, err := res.LastInsertId()
	if err != nil {
		return "", tracing.Trace(span, apierror.NewInternalError(err, "Failed to read the reserved transaction number."))
	}

	return strconv.FormatInt(number, 10), nil
}

func transactionCreatedAt(d *domain.TransactionSummary) time.Time { return d.CreatedAt }
func transactionID(d *domain.TransactionSummary) string           { return d.ID }

func accountTransactionCreatedAt(d *domain.Transaction) time.Time { return d.CreatedAt }
func accountTransactionID(d *domain.Transaction) string           { return d.ID }

func buildTransactionSearchQuery(query *string) gosql.NullString {
	if query == nil || *query == "" {
		return gosql.NullString{}
	}
	sanitized := db.SanitizeFulltextBoolean(*query)
	if sanitized == "" {
		return gosql.NullString{}
	}
	term := sanitized + "*"
	return gosql.NullString{String: term, Valid: true}
}

func toNullStrings(values []string) []gosql.NullString {
	result := make([]gosql.NullString, len(values))
	for i, v := range values {
		result[i] = gosql.NullString{String: v, Valid: true}
	}
	return result
}

func (r *transactionRepoImpl) List(ctx context.Context, params domain.ListTransactionsParams) (*domain.ListTransactionsResult, *apierror.APIError) {
	ctx, span := transactionRepoTracer.Start(ctx, "repository.transaction.list")
	defer span.End()

	searchQuery := buildTransactionSearchQuery(params.Query)
	startDate := gosql.NullTime{}
	if params.StartDate != nil {
		startDate = gosql.NullTime{Time: *params.StartDate, Valid: true}
	}
	endDate := gosql.NullTime{}
	if params.EndDate != nil {
		endDate = gosql.NullTime{Time: *params.EndDate, Valid: true}
	}

	status := toNullString(params.Status)

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationErrorWithParam("Invalid pagination cursor.", "cursor")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListTransactionsBackward(ctx, sqlc.ListTransactionsBackwardParams{
				AccountID:                        params.AccountID,
				Cursor:                           cur.ID,
				Query:                            searchQuery,
				Status:                           status,
				IncludeTypeCodesFilter:           len(params.TypeCodes) > 0,
				TypeCodes:                        params.TypeCodes,
				IncludeAdjustmentTypeCodesFilter: len(params.AdjustmentTypeCodes) > 0,
				AdjustmentTypeCodes:              toNullStrings(params.AdjustmentTypeCodes),
				IncludeMethodCodesFilter:         len(params.MethodCodes) > 0,
				MethodCodes:                      toNullStrings(params.MethodCodes),
				IncludeCustomerIdsFilter:         len(params.CustomerIDs) > 0,
				CustomerIds:                      params.CustomerIDs,
				IncludeCustomerGroupIdsFilter:    len(params.CustomerGroupIDs) > 0,
				CustomerGroupIds:                 toNullStrings(params.CustomerGroupIDs),
				StartDate:                        startDate,
				EndDate:                          endDate,
				Limit:                            params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			transactions := make([]*domain.TransactionSummary, len(rows))
			for i, row := range rows {
				transactions[i] = mapBackwardTransactionRow(row)
			}
			result, pageInfo := pagination.BuildPageString(transactions, params.Limit, cursorDir, transactionCreatedAt, transactionID)
			return &domain.ListTransactionsResult{Transactions: result, PageInfo: pageInfo}, nil
		}

		rows, err := r.queries.ListTransactionsForward(ctx, sqlc.ListTransactionsForwardParams{
			AccountID:                        params.AccountID,
			Cursor:                           gosql.NullString{String: cur.ID, Valid: true},
			Query:                            searchQuery,
			Status:                           status,
			IncludeTypeCodesFilter:           len(params.TypeCodes) > 0,
			TypeCodes:                        params.TypeCodes,
			IncludeAdjustmentTypeCodesFilter: len(params.AdjustmentTypeCodes) > 0,
			AdjustmentTypeCodes:              toNullStrings(params.AdjustmentTypeCodes),
			IncludeMethodCodesFilter:         len(params.MethodCodes) > 0,
			MethodCodes:                      toNullStrings(params.MethodCodes),
			IncludeCustomerIdsFilter:         len(params.CustomerIDs) > 0,
			CustomerIds:                      params.CustomerIDs,
			IncludeCustomerGroupIdsFilter:    len(params.CustomerGroupIDs) > 0,
			CustomerGroupIds:                 toNullStrings(params.CustomerGroupIDs),
			StartDate:                        startDate,
			EndDate:                          endDate,
			Limit:                            params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		transactions := make([]*domain.TransactionSummary, len(rows))
		for i, row := range rows {
			transactions[i] = mapForwardTransactionRow(row)
		}
		result, pageInfo := pagination.BuildPageString(transactions, params.Limit, cursorDir, transactionCreatedAt, transactionID)
		return &domain.ListTransactionsResult{Transactions: result, PageInfo: pageInfo}, nil
	}

	// No cursor - forward from beginning
	rows, err := r.queries.ListTransactionsForward(ctx, sqlc.ListTransactionsForwardParams{
		AccountID:                        params.AccountID,
		Cursor:                           gosql.NullString{},
		Query:                            searchQuery,
		Status:                           status,
		IncludeTypeCodesFilter:           len(params.TypeCodes) > 0,
		TypeCodes:                        params.TypeCodes,
		IncludeAdjustmentTypeCodesFilter: len(params.AdjustmentTypeCodes) > 0,
		AdjustmentTypeCodes:              toNullStrings(params.AdjustmentTypeCodes),
		IncludeMethodCodesFilter:         len(params.MethodCodes) > 0,
		MethodCodes:                      toNullStrings(params.MethodCodes),
		IncludeCustomerIdsFilter:         len(params.CustomerIDs) > 0,
		CustomerIds:                      params.CustomerIDs,
		IncludeCustomerGroupIdsFilter:    len(params.CustomerGroupIDs) > 0,
		CustomerGroupIds:                 toNullStrings(params.CustomerGroupIDs),
		StartDate:                        startDate,
		EndDate:                          endDate,
		Limit:                            params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	transactions := make([]*domain.TransactionSummary, len(rows))
	for i, row := range rows {
		transactions[i] = mapForwardTransactionRow(row)
	}
	result, pageInfo := pagination.BuildPageString(transactions, params.Limit, cursorDir, transactionCreatedAt, transactionID)
	return &domain.ListTransactionsResult{Transactions: result, PageInfo: pageInfo}, nil
}

func mapForwardTransactionRow(row sqlc.ListTransactionsForwardRow) *domain.TransactionSummary {
	s := &domain.TransactionSummary{
		ID:                  row.ID,
		Number:              row.Number,
		AmountID:            row.AmountID,
		AmountValue:         decimalToString(row.AmountValue),
		AmountUnitID:        row.AmountUnitID,
		AmountUnitAbbr:      row.AmountUnitAbbreviation,
		TransactionTypeID:   row.TransactionTypeID,
		TransactionTypeCode: row.TransactionTypeCode,
		TransactionTypeName: row.TransactionTypeName,
		IsFullyAllocated:    row.IsFullyAllocated,
		AllocationCount:     safeconv.Int64ToInt32(row.AllocationCount),
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}

	s.CustomerID = &row.CustomerAccountID
	if row.CustomerName != "" {
		s.CustomerName = &row.CustomerName
	}
	if row.CustomerNumber != "" {
		s.CustomerNumber = &row.CustomerNumber
	}
	if row.CustomerStatus.Valid {
		s.CustomerStatusCode = &row.CustomerStatus.String
	}
	if row.CustomerCommissionPolicy.Valid {
		s.CustomerCommissionPolicy = &row.CustomerCommissionPolicy.String
	}
	customerCreatedAt := row.CustomerCreatedAt
	s.CustomerCreatedAt = &customerCreatedAt
	customerUpdatedAt := row.CustomerUpdatedAt
	s.CustomerUpdatedAt = &customerUpdatedAt

	if row.TransactionMethodID.Valid {
		s.TransactionMethodID = &row.TransactionMethodID.String
	}
	if row.TransactionMethodCode.Valid {
		s.TransactionMethodCode = &row.TransactionMethodCode.String
	}
	if row.TransactionMethodName.Valid {
		s.TransactionMethodName = &row.TransactionMethodName.String
	}
	if row.AdjustmentTypeID.Valid {
		s.AdjustmentTypeID = &row.AdjustmentTypeID.String
	}
	if row.AdjustmentTypeCode.Valid {
		s.AdjustmentTypeCode = &row.AdjustmentTypeCode.String
	}
	if row.AdjustmentTypeName.Valid {
		s.AdjustmentTypeName = &row.AdjustmentTypeName.String
	}
	return s
}

func mapBackwardTransactionRow(row sqlc.ListTransactionsBackwardRow) *domain.TransactionSummary {
	s := &domain.TransactionSummary{
		ID:                  row.ID,
		Number:              row.Number,
		AmountID:            row.AmountID,
		AmountValue:         decimalToString(row.AmountValue),
		AmountUnitID:        row.AmountUnitID,
		AmountUnitAbbr:      row.AmountUnitAbbreviation,
		TransactionTypeID:   row.TransactionTypeID,
		TransactionTypeCode: row.TransactionTypeCode,
		TransactionTypeName: row.TransactionTypeName,
		IsFullyAllocated:    row.IsFullyAllocated,
		AllocationCount:     safeconv.Int64ToInt32(row.AllocationCount),
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}

	s.CustomerID = &row.CustomerAccountID
	if row.CustomerName != "" {
		s.CustomerName = &row.CustomerName
	}
	if row.CustomerNumber != "" {
		s.CustomerNumber = &row.CustomerNumber
	}
	if row.CustomerStatus.Valid {
		s.CustomerStatusCode = &row.CustomerStatus.String
	}
	if row.CustomerCommissionPolicy.Valid {
		s.CustomerCommissionPolicy = &row.CustomerCommissionPolicy.String
	}
	customerCreatedAt := row.CustomerCreatedAt
	s.CustomerCreatedAt = &customerCreatedAt
	customerUpdatedAt := row.CustomerUpdatedAt
	s.CustomerUpdatedAt = &customerUpdatedAt

	if row.TransactionMethodID.Valid {
		s.TransactionMethodID = &row.TransactionMethodID.String
	}
	if row.TransactionMethodCode.Valid {
		s.TransactionMethodCode = &row.TransactionMethodCode.String
	}
	if row.TransactionMethodName.Valid {
		s.TransactionMethodName = &row.TransactionMethodName.String
	}
	if row.AdjustmentTypeID.Valid {
		s.AdjustmentTypeID = &row.AdjustmentTypeID.String
	}
	if row.AdjustmentTypeCode.Valid {
		s.AdjustmentTypeCode = &row.AdjustmentTypeCode.String
	}
	if row.AdjustmentTypeName.Valid {
		s.AdjustmentTypeName = &row.AdjustmentTypeName.String
	}
	return s
}

func (r *transactionRepoImpl) Get(ctx context.Context, accountID, transactionID string) (*domain.Transaction, *apierror.APIError) {
	ctx, span := transactionRepoTracer.Start(ctx, "repository.transaction.get")
	defer span.End()

	row, err := r.queries.FindTransactionByID(ctx, sqlc.FindTransactionByIDParams{
		ID:        transactionID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapTransactionRow(row), nil
}

func mapTransactionRow(row sqlc.FindTransactionByIDRow) *domain.Transaction {
	t := &domain.Transaction{
		ID:                  row.ID,
		Number:              row.Number,
		AmountID:            row.AmountID,
		AmountValue:         decimalToString(row.AmountValue),
		AmountUnitID:        row.AmountUnitID,
		AmountUnitAbbr:      row.AmountUnitAbbreviation,
		TransactionTypeID:   row.TransactionTypeID,
		TransactionTypeCode: row.TransactionTypeCode,
		TransactionTypeName: row.TransactionTypeName,
		IsFullyAllocated:    row.IsFullyAllocated,
		AllocationCount:     safeconv.Int64ToInt32(row.AllocationCount),
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}

	t.CustomerID = &row.CustomerAccountID
	if row.CustomerName != "" {
		t.CustomerName = &row.CustomerName
	}
	if row.CustomerNumber != "" {
		t.CustomerNumber = &row.CustomerNumber
	}
	if row.CustomerStatus.Valid {
		t.CustomerStatusCode = &row.CustomerStatus.String
	}
	if row.CustomerCommissionPolicy.Valid {
		t.CustomerCommissionPolicy = &row.CustomerCommissionPolicy.String
	}
	customerCreatedAt := row.CustomerCreatedAt
	t.CustomerCreatedAt = &customerCreatedAt
	customerUpdatedAt := row.CustomerUpdatedAt
	t.CustomerUpdatedAt = &customerUpdatedAt

	// Prefer the account_user id resolved by the query; legacy rows store a user id in responsible_user_id and may have no account_user match.
	if row.ResponsibleAccountUserID.Valid {
		t.ResponsibleUserID = &row.ResponsibleAccountUserID.String
	} else if row.ResponsibleUserID.Valid {
		t.ResponsibleUserID = &row.ResponsibleUserID.String
	}
	name := row.ResponsibleUserName
	if name != "" {
		t.ResponsibleUserName = &name
	}
	if row.ResponsibleUserStatus.Valid {
		t.ResponsibleUserStatusCode = &row.ResponsibleUserStatus.String
	}
	if row.ResponsibleUserCreatedAt.Valid {
		t.ResponsibleUserCreatedAt = &row.ResponsibleUserCreatedAt.Time
	}
	if row.ResponsibleUserUpdatedAt.Valid {
		t.ResponsibleUserUpdatedAt = &row.ResponsibleUserUpdatedAt.Time
	}

	if row.Note.Valid {
		t.Note = &row.Note.String
	}
	if row.TransactionMethodID.Valid {
		t.TransactionMethodID = &row.TransactionMethodID.String
	}
	if row.TransactionMethodCode.Valid {
		t.TransactionMethodCode = &row.TransactionMethodCode.String
	}
	if row.TransactionMethodName.Valid {
		t.TransactionMethodName = &row.TransactionMethodName.String
	}
	if row.AdjustmentTypeID.Valid {
		t.AdjustmentTypeID = &row.AdjustmentTypeID.String
	}
	if row.AdjustmentTypeCode.Valid {
		t.AdjustmentTypeCode = &row.AdjustmentTypeCode.String
	}
	if row.AdjustmentTypeName.Valid {
		t.AdjustmentTypeName = &row.AdjustmentTypeName.String
	}
	if row.StripePaymentID.Valid {
		t.StripePaymentID = &row.StripePaymentID.String
	}
	return t
}

func (r *transactionRepoImpl) GetAllocations(ctx context.Context, transactionID string) ([]*domain.TransactionAllocation, *apierror.APIError) {
	ctx, span := transactionRepoTracer.Start(ctx, "repository.transaction.get_allocations")
	defer span.End()

	rows, err := r.queries.GetTransactionAllocations(ctx, transactionID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	allocations := make([]*domain.TransactionAllocation, len(rows))
	for i, row := range rows {
		var allocNote *string
		if row.Note.Valid {
			allocNote = &row.Note.String
		}
		allocations[i] = &domain.TransactionAllocation{
			ID:                row.ID,
			AmountID:          row.AmountID,
			AmountValue:       decimalToString(row.AmountValue),
			AmountUnitID:      row.AmountUnitID,
			AmountUnitAbbr:    row.AmountUnitAbbreviation,
			Note:              allocNote,
			TransactionID:     row.TransactionID,
			TransactionNumber: row.TransactionNumber,
			TransactionType:   row.TransactionType,
			InvoiceID:         row.InvoiceID,
			InvoiceNumber:     row.InvoiceNumber,
			CreatedAt:         row.CreatedAt,
			UpdatedAt:         row.UpdatedAt,
		}
	}

	return allocations, nil
}

func (r *transactionRepoImpl) Update(ctx context.Context, params domain.UpdateTransactionParams) (*domain.Transaction, *apierror.APIError) {
	ctx, span := transactionRepoTracer.Start(ctx, "repository.transaction.update")
	defer span.End()

	updateParams := sqlc.UpdateTransactionParams{
		ID:                     params.TransactionID,
		AccountID:              params.AccountID,
		ClearTransactionMethod: params.ClearTransactionMethod,
		ClearAdjustmentType:    params.ClearAdjustmentType,
		ClearResponsibleUser:   params.ClearResponsibleUser,
	}
	if params.Number != nil {
		updateParams.Number = gosql.NullString{String: *params.Number, Valid: true}
	}
	if params.Note != nil {
		updateParams.UpdateNote = gosql.NullString{String: "1", Valid: true}
		updateParams.Note = gosql.NullString{String: *params.Note, Valid: true}
	}
	if params.TransactionMethodCode != nil {
		updateParams.TransactionMethodCode = gosql.NullString{String: *params.TransactionMethodCode, Valid: true}
	}
	if params.AdjustmentTypeCode != nil {
		updateParams.AdjustmentTypeCode = gosql.NullString{String: *params.AdjustmentTypeCode, Valid: true}
	}
	if params.ResponsibleUserID != nil {
		updateParams.ResponsibleUserID = gosql.NullString{String: *params.ResponsibleUserID, Valid: true}
	}
	if params.IsFullyAllocated != nil {
		updateParams.IsFullyAllocated = gosql.NullBool{Bool: *params.IsFullyAllocated, Valid: true}
	}
	err := r.queries.UpdateTransaction(ctx, updateParams)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Update amount if provided
	if params.Amount != nil {
		amountID, qErr := r.queries.GetTransactionAmountID(ctx, params.TransactionID)
		if apiErr := db.MapSQLError(qErr); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		qErr = r.queries.UpdateTransactionQuantity(ctx, sqlc.UpdateTransactionQuantityParams{
			ID:    amountID,
			Value: *params.Amount,
		})
		if apiErr := db.MapSQLError(qErr); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	return r.Get(ctx, params.AccountID, params.TransactionID)
}

func (r *transactionRepoImpl) ExistsByNumber(ctx context.Context, accountID, number string, excludeID *string) (bool, *apierror.APIError) {
	ctx, span := transactionRepoTracer.Start(ctx, "repository.transaction.exists_by_number")
	defer span.End()

	cnt, err := r.queries.ExistsTransactionByNumber(ctx, sqlc.ExistsTransactionByNumberParams{
		AccountID: accountID,
		Number:    number,
		ExcludeID: toNullString(excludeID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return cnt > 0, nil
}

func (r *transactionRepoImpl) ResolveResponsibleUserID(ctx context.Context, accountID, userOrAccountUserID string) (string, *apierror.APIError) {
	ctx, span := transactionRepoTracer.Start(ctx, "repository.transaction.resolve_responsible_user_id")
	defer span.End()

	resolvedID, err := r.queries.ResolveResponsibleUserID(ctx, sqlc.ResolveResponsibleUserIDParams{
		AccountID:           accountID,
		UserOrAccountUserID: userOrAccountUserID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}
	return resolvedID, nil
}

func (r *transactionRepoImpl) ListByCustomer(ctx context.Context, params domain.ListAccountTransactionsParams) (*domain.ListAccountTransactionsResult, *apierror.APIError) {
	ctx, span := transactionRepoTracer.Start(ctx, "repository.transaction.list_by_customer")
	defer span.End()

	searchQuery := buildTransactionSearchQuery(params.Query)
	status := toNullString(params.Status)
	txType := toNullString(params.Type)

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationErrorWithParam("Invalid pagination cursor.", "cursor")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListAccountTransactionsBackward(ctx, sqlc.ListAccountTransactionsBackwardParams{
				AccountID:         params.AccountID,
				CustomerAccountID: params.CustomerAccountID,
				Cursor:            cur.ID,
				Query:             searchQuery,
				Status:            status,
				Type:              txType,
				Limit:             params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			transactions := make([]*domain.Transaction, len(rows))
			for i, row := range rows {
				transactions[i] = mapAccountTransactionBackwardRow(row)
			}
			result, pageInfo := pagination.BuildPageString(transactions, params.Limit, cursorDir, accountTransactionCreatedAt, accountTransactionID)
			return &domain.ListAccountTransactionsResult{Transactions: result, PageInfo: pageInfo}, nil
		}

		rows, err := r.queries.ListAccountTransactionsForward(ctx, sqlc.ListAccountTransactionsForwardParams{
			AccountID:         params.AccountID,
			CustomerAccountID: params.CustomerAccountID,
			Cursor:            gosql.NullString{String: cur.ID, Valid: true},
			Query:             searchQuery,
			Status:            status,
			Type:              txType,
			Limit:             params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		transactions := make([]*domain.Transaction, len(rows))
		for i, row := range rows {
			transactions[i] = mapAccountTransactionForwardRow(row)
		}
		result, pageInfo := pagination.BuildPageString(transactions, params.Limit, cursorDir, accountTransactionCreatedAt, accountTransactionID)
		return &domain.ListAccountTransactionsResult{Transactions: result, PageInfo: pageInfo}, nil
	}

	// No cursor
	rows, err := r.queries.ListAccountTransactionsForward(ctx, sqlc.ListAccountTransactionsForwardParams{
		AccountID:         params.AccountID,
		CustomerAccountID: params.CustomerAccountID,
		Cursor:            gosql.NullString{},
		Query:             searchQuery,
		Status:            status,
		Type:              txType,
		Limit:             params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	transactions := make([]*domain.Transaction, len(rows))
	for i, row := range rows {
		transactions[i] = mapAccountTransactionForwardRow(row)
	}
	result, pageInfo := pagination.BuildPageString(transactions, params.Limit, cursorDir, accountTransactionCreatedAt, accountTransactionID)
	return &domain.ListAccountTransactionsResult{Transactions: result, PageInfo: pageInfo}, nil
}

func mapAccountTransactionForwardRow(row sqlc.ListAccountTransactionsForwardRow) *domain.Transaction {
	t := &domain.Transaction{
		ID:                  row.ID,
		Number:              row.Number,
		AmountID:            row.AmountID,
		AmountValue:         decimalToString(row.AmountValue),
		AmountUnitID:        row.AmountUnitID,
		AmountUnitAbbr:      row.AmountUnitAbbreviation,
		TransactionTypeID:   row.TransactionTypeID,
		TransactionTypeCode: row.TransactionTypeCode,
		TransactionTypeName: row.TransactionTypeName,
		IsFullyAllocated:    row.IsFullyAllocated,
		AllocationCount:     safeconv.Int64ToInt32(row.AllocationCount),
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}

	t.CustomerID = &row.CustomerAccountID
	if row.CustomerName != "" {
		t.CustomerName = &row.CustomerName
	}
	if row.CustomerNumber != "" {
		t.CustomerNumber = &row.CustomerNumber
	}
	if row.CustomerStatus.Valid {
		t.CustomerStatusCode = &row.CustomerStatus.String
	}
	if row.CustomerCommissionPolicy.Valid {
		t.CustomerCommissionPolicy = &row.CustomerCommissionPolicy.String
	}
	customerCreatedAt := row.CustomerCreatedAt
	t.CustomerCreatedAt = &customerCreatedAt
	customerUpdatedAt := row.CustomerUpdatedAt
	t.CustomerUpdatedAt = &customerUpdatedAt

	// Prefer the account_user id resolved by the query; legacy rows store a user id in responsible_user_id and may have no account_user match.
	if row.ResponsibleAccountUserID.Valid {
		t.ResponsibleUserID = &row.ResponsibleAccountUserID.String
	} else if row.ResponsibleUserID.Valid {
		t.ResponsibleUserID = &row.ResponsibleUserID.String
	}
	name := row.ResponsibleUserName
	if name != "" {
		t.ResponsibleUserName = &name
	}
	if row.ResponsibleUserStatus.Valid {
		t.ResponsibleUserStatusCode = &row.ResponsibleUserStatus.String
	}
	if row.ResponsibleUserCreatedAt.Valid {
		t.ResponsibleUserCreatedAt = &row.ResponsibleUserCreatedAt.Time
	}
	if row.ResponsibleUserUpdatedAt.Valid {
		t.ResponsibleUserUpdatedAt = &row.ResponsibleUserUpdatedAt.Time
	}

	if row.Note.Valid {
		t.Note = &row.Note.String
	}
	if row.TransactionMethodID.Valid {
		t.TransactionMethodID = &row.TransactionMethodID.String
	}
	if row.TransactionMethodCode.Valid {
		t.TransactionMethodCode = &row.TransactionMethodCode.String
	}
	if row.TransactionMethodName.Valid {
		t.TransactionMethodName = &row.TransactionMethodName.String
	}
	if row.AdjustmentTypeID.Valid {
		t.AdjustmentTypeID = &row.AdjustmentTypeID.String
	}
	if row.AdjustmentTypeCode.Valid {
		t.AdjustmentTypeCode = &row.AdjustmentTypeCode.String
	}
	if row.AdjustmentTypeName.Valid {
		t.AdjustmentTypeName = &row.AdjustmentTypeName.String
	}
	if row.StripePaymentID.Valid {
		t.StripePaymentID = &row.StripePaymentID.String
	}
	return t
}

func mapAccountTransactionBackwardRow(row sqlc.ListAccountTransactionsBackwardRow) *domain.Transaction {
	t := &domain.Transaction{
		ID:                  row.ID,
		Number:              row.Number,
		AmountID:            row.AmountID,
		AmountValue:         decimalToString(row.AmountValue),
		AmountUnitID:        row.AmountUnitID,
		AmountUnitAbbr:      row.AmountUnitAbbreviation,
		TransactionTypeID:   row.TransactionTypeID,
		TransactionTypeCode: row.TransactionTypeCode,
		TransactionTypeName: row.TransactionTypeName,
		IsFullyAllocated:    row.IsFullyAllocated,
		AllocationCount:     safeconv.Int64ToInt32(row.AllocationCount),
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}

	t.CustomerID = &row.CustomerAccountID
	if row.CustomerName != "" {
		t.CustomerName = &row.CustomerName
	}
	if row.CustomerNumber != "" {
		t.CustomerNumber = &row.CustomerNumber
	}
	if row.CustomerStatus.Valid {
		t.CustomerStatusCode = &row.CustomerStatus.String
	}
	if row.CustomerCommissionPolicy.Valid {
		t.CustomerCommissionPolicy = &row.CustomerCommissionPolicy.String
	}
	customerCreatedAt := row.CustomerCreatedAt
	t.CustomerCreatedAt = &customerCreatedAt
	customerUpdatedAt := row.CustomerUpdatedAt
	t.CustomerUpdatedAt = &customerUpdatedAt

	// Prefer the account_user id resolved by the query; legacy rows store a user id in responsible_user_id and may have no account_user match.
	if row.ResponsibleAccountUserID.Valid {
		t.ResponsibleUserID = &row.ResponsibleAccountUserID.String
	} else if row.ResponsibleUserID.Valid {
		t.ResponsibleUserID = &row.ResponsibleUserID.String
	}
	name := row.ResponsibleUserName
	if name != "" {
		t.ResponsibleUserName = &name
	}
	if row.ResponsibleUserStatus.Valid {
		t.ResponsibleUserStatusCode = &row.ResponsibleUserStatus.String
	}
	if row.ResponsibleUserCreatedAt.Valid {
		t.ResponsibleUserCreatedAt = &row.ResponsibleUserCreatedAt.Time
	}
	if row.ResponsibleUserUpdatedAt.Valid {
		t.ResponsibleUserUpdatedAt = &row.ResponsibleUserUpdatedAt.Time
	}

	if row.Note.Valid {
		t.Note = &row.Note.String
	}
	if row.TransactionMethodID.Valid {
		t.TransactionMethodID = &row.TransactionMethodID.String
	}
	if row.TransactionMethodCode.Valid {
		t.TransactionMethodCode = &row.TransactionMethodCode.String
	}
	if row.TransactionMethodName.Valid {
		t.TransactionMethodName = &row.TransactionMethodName.String
	}
	if row.AdjustmentTypeID.Valid {
		t.AdjustmentTypeID = &row.AdjustmentTypeID.String
	}
	if row.AdjustmentTypeCode.Valid {
		t.AdjustmentTypeCode = &row.AdjustmentTypeCode.String
	}
	if row.AdjustmentTypeName.Valid {
		t.AdjustmentTypeName = &row.AdjustmentTypeName.String
	}
	if row.StripePaymentID.Valid {
		t.StripePaymentID = &row.StripePaymentID.String
	}
	return t
}

func (r *transactionRepoImpl) GetDollarUnitID(ctx context.Context) (string, *apierror.APIError) {
	ctx, span := transactionRepoTracer.Start(ctx, "repository.transaction.get_dollar_unit_id")
	defer span.End()

	unitID, err := r.queries.GetDollarUnitIDForTransaction(ctx)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}
	return unitID, nil
}
