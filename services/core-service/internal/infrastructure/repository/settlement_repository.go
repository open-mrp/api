package repository

import (
	"context"
	gosql "database/sql"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var settlementRepoTracer = tracing.GetTracer("core-service.settlement_repository")

type settlementRepoImpl struct {
	queries *sqlc.Queries
}

func NewSettlementRepo(queries *sqlc.Queries) domain.SettlementRepo {
	return &settlementRepoImpl{queries: queries}
}

func settlementCreatedAt(d *domain.SettlementSummary) time.Time { return d.CreatedAt }
func settlementID(d *domain.SettlementSummary) string           { return d.ID }

func (r *settlementRepoImpl) Get(ctx context.Context, accountID, settlementID string) (*domain.Settlement, *apierror.APIError) {
	ctx, span := settlementRepoTracer.Start(ctx, "repository.settlement.get")
	defer span.End()

	row, err := r.queries.GetSettlement(ctx, sqlc.GetSettlementParams{
		ID:        settlementID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	var note *string
	if row.Note.Valid {
		note = &row.Note.String
	}

	settlement := &domain.Settlement{
		ID:        row.ID,
		Number:    row.Number,
		Note:      note,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}

	if row.ResponsibleUserAccountUserID.Valid {
		settlement.ResponsibleUserID = &row.ResponsibleUserAccountUserID.String
		if row.ResponsibleUserName.Valid {
			settlement.ResponsibleUserName = &row.ResponsibleUserName.String
		}
	} else if row.ResponsibleUserID.Valid {
		settlement.ResponsibleUserID = &row.ResponsibleUserID.String
	}

	return settlement, nil
}

func (r *settlementRepoImpl) GetAllocations(ctx context.Context, settlementID string) ([]*domain.TransactionAllocation, *apierror.APIError) {
	ctx, span := settlementRepoTracer.Start(ctx, "repository.settlement.get_allocations")
	defer span.End()

	allocations, apiErr := loadAllocations(ctx, r.queries.DB(), "ta.settlement_id = ?", "ta.created_at ASC, ta.id ASC", settlementID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return allocations, nil
}

func (r *settlementRepoImpl) InsertSettlement(ctx context.Context, id, number string, params domain.CreateSettlementParams) *apierror.APIError {
	ctx, span := settlementRepoTracer.Start(ctx, "repository.settlement.insert")
	defer span.End()

	var responsibleUserID gosql.NullString
	if params.ResponsibleUserID != "" {
		responsibleUserID = gosql.NullString{String: params.ResponsibleUserID, Valid: true}
	}

	err := r.queries.InsertSettlement(ctx, sqlc.InsertSettlementParams{
		ID:                id,
		Number:            number,
		ResponsibleUserID: responsibleUserID,
		AccountID:         params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *settlementRepoImpl) Update(ctx context.Context, params domain.UpdateSettlementParams) (*domain.Settlement, *apierror.APIError) {
	ctx, span := settlementRepoTracer.Start(ctx, "repository.settlement.update")
	defer span.End()

	updateParams := sqlc.UpdateSettlementParams{
		ID:        params.SettlementID,
		AccountID: params.AccountID,
		ClearNote: params.ClearNote,
	}
	if params.Number != nil {
		updateParams.Number = gosql.NullString{String: *params.Number, Valid: true}
	}
	if params.Note != nil {
		updateParams.Note = gosql.NullString{String: *params.Note, Valid: true}
	}
	if params.ResponsibleUserID != nil {
		updateParams.ResponsibleUserID = gosql.NullString{String: *params.ResponsibleUserID, Valid: true}
	}

	err := r.queries.UpdateSettlement(ctx, updateParams)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.Get(ctx, params.AccountID, params.SettlementID)
}

func (r *settlementRepoImpl) Delete(ctx context.Context, accountID, settlementID string) *apierror.APIError {
	ctx, span := settlementRepoTracer.Start(ctx, "repository.settlement.delete")
	defer span.End()

	err := r.queries.DeleteSettlement(ctx, sqlc.DeleteSettlementParams{
		ID:        settlementID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *settlementRepoImpl) IsDuplicateNumber(ctx context.Context, accountID, number string, excludeID *string) (bool, *apierror.APIError) {
	ctx, span := settlementRepoTracer.Start(ctx, "repository.settlement.is_duplicate_number")
	defer span.End()

	var exclude gosql.NullString
	if excludeID != nil {
		exclude = gosql.NullString{String: *excludeID, Valid: true}
	}

	isDuplicate, err := r.queries.CheckSettlementNumberDuplicate(ctx, sqlc.CheckSettlementNumberDuplicateParams{
		AccountID: accountID,
		Number:    number,
		ExcludeID: exclude,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return isDuplicate, nil
}

func (r *settlementRepoImpl) GetDollarUnitID(ctx context.Context) (string, *apierror.APIError) {
	ctx, span := settlementRepoTracer.Start(ctx, "repository.settlement.get_dollar_unit_id")
	defer span.End()

	unitID, err := r.queries.GetDollarUnitID(ctx)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}
	return unitID, nil
}

func (r *settlementRepoImpl) CreateAllocation(ctx context.Context, allocationID, quantityID, settlementID, dollarUnitID string, params domain.CreateSettlementAllocationParams) *apierror.APIError {
	ctx, span := settlementRepoTracer.Start(ctx, "repository.settlement.create_allocation")
	defer span.End()

	// First create the quantity
	err := r.queries.InsertAllocationQuantity(ctx, sqlc.InsertAllocationQuantityParams{
		ID:     quantityID,
		Value:  params.Amount,
		UnitID: dollarUnitID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// Then create the allocation
	var note gosql.NullString
	if params.Note != nil {
		note = gosql.NullString{String: *params.Note, Valid: true}
	}

	err = r.queries.InsertTransactionAllocation(ctx, sqlc.InsertTransactionAllocationParams{
		ID:            allocationID,
		TransactionID: params.TransactionID,
		AmountID:      quantityID,
		InvoiceID:     params.InvoiceID,
		SettlementID:  gosql.NullString{String: settlementID, Valid: true},
		Note:          note,
		CreatedAt:     toNullTime(params.CreatedAt),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *settlementRepoImpl) DeleteAllocations(ctx context.Context, settlementID string) ([]*domain.TransactionAllocation, *apierror.APIError) {
	ctx, span := settlementRepoTracer.Start(ctx, "repository.settlement.delete_allocations")
	defer span.End()

	// First get the allocations before deleting
	rows, err := r.queries.DeleteSettlementAllocations(ctx, gosql.NullString{String: settlementID, Valid: true})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	allocations := make([]*domain.TransactionAllocation, len(rows))
	for i, row := range rows {
		var delNote *string
		if row.Note.Valid {
			delNote = &row.Note.String
		}
		allocations[i] = &domain.TransactionAllocation{
			ID:                row.ID,
			AmountID:          row.AmountID,
			AmountValue:       decimalToString(row.AmountValue),
			AmountUnitID:      row.AmountUnitID,
			AmountUnitAbbr:    row.AmountUnitAbbreviation,
			Note:              delNote,
			TransactionID:     row.TransactionID,
			TransactionNumber: row.TransactionNumber,
			TransactionType:   row.TransactionType,
			InvoiceID:         row.InvoiceID,
			InvoiceNumber:     row.InvoiceNumber,
			CreatedAt:         row.CreatedAt,
			UpdatedAt:         row.UpdatedAt,
		}
	}

	// Delete quantities first, then allocations
	if err := r.queries.DeleteQuantitiesBySettlementAllocations(ctx, gosql.NullString{String: settlementID, Valid: true}); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	if err := r.queries.DeleteTransactionAllocationsBySettlement(ctx, gosql.NullString{String: settlementID, Valid: true}); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	return allocations, nil
}

func (r *settlementRepoImpl) GetAllocationTransactionIDs(ctx context.Context, settlementID string) ([]string, *apierror.APIError) {
	ctx, span := settlementRepoTracer.Start(ctx, "repository.settlement.get_allocation_transaction_ids")
	defer span.End()

	ids, err := r.queries.GetSettlementAllocationTransactionIDs(ctx, gosql.NullString{String: settlementID, Valid: true})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return ids, nil
}

func (r *settlementRepoImpl) GetAllocationInvoiceIDs(ctx context.Context, settlementID string) ([]string, *apierror.APIError) {
	ctx, span := settlementRepoTracer.Start(ctx, "repository.settlement.get_allocation_invoice_ids")
	defer span.End()

	ids, err := r.queries.GetSettlementAllocationInvoiceIDs(ctx, gosql.NullString{String: settlementID, Valid: true})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return ids, nil
}

func (r *settlementRepoImpl) AllocateNextSettlementNumber(ctx context.Context, sysPropertyID, accountID string) (int64, *apierror.APIError) {
	ctx, span := settlementRepoTracer.Start(ctx, "repository.settlement.allocate_next_number")
	defer span.End()

	res, err := r.queries.AllocateNextSettlementNumber(ctx, sqlc.AllocateNextSettlementNumberParams{
		ID:        sysPropertyID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}

	number, err := res.LastInsertId()
	if err != nil {
		return 0, tracing.Trace(span, apierror.NewInternalError(err, "Failed to read the reserved settlement number."))
	}

	return number, nil
}

func (r *settlementRepoImpl) DeleteSettlementOwnedTransactions(ctx context.Context, accountID, settlementID string) *apierror.APIError {
	ctx, span := settlementRepoTracer.Start(ctx, "repository.settlement.delete_owned_transactions")
	defer span.End()

	err := r.queries.DeleteSettlementOwnedTransactions(ctx, sqlc.DeleteSettlementOwnedTransactionsParams{
		AccountID:    accountID,
		SettlementID: gosql.NullString{String: settlementID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *settlementRepoImpl) MarkTransactionsCreatedBySettlement(ctx context.Context, accountID, settlementID string, transactionIDs []string) *apierror.APIError {
	ctx, span := settlementRepoTracer.Start(ctx, "repository.settlement.mark_transactions_created_by_settlement")
	defer span.End()

	if len(transactionIDs) == 0 {
		return nil
	}
	err := r.queries.MarkTransactionsCreatedBySettlement(ctx, sqlc.MarkTransactionsCreatedBySettlementParams{
		SettlementID:   gosql.NullString{String: settlementID, Valid: true},
		AccountID:      accountID,
		TransactionIds: transactionIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *settlementRepoImpl) UpdateTransactionsFullyAllocated(ctx context.Context, accountID string, transactionIDs []string, isFullyAllocated bool) *apierror.APIError {
	ctx, span := settlementRepoTracer.Start(ctx, "repository.settlement.update_transactions_fully_allocated")
	defer span.End()

	if len(transactionIDs) == 0 {
		return nil
	}

	err := r.queries.UpdateTransactionsFullyAllocated(ctx, sqlc.UpdateTransactionsFullyAllocatedParams{
		AccountID:        accountID,
		IsFullyAllocated: isFullyAllocated,
		TransactionIds:   transactionIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *settlementRepoImpl) UpdateInvoicePaymentStatus(ctx context.Context, accountID, invoiceID string, isPaidInFull, isOverPaid bool) *apierror.APIError {
	ctx, span := settlementRepoTracer.Start(ctx, "repository.settlement.update_invoice_payment_status")
	defer span.End()

	err := r.queries.UpdateInvoicePaymentStatus(ctx, sqlc.UpdateInvoicePaymentStatusParams{
		AccountID:    accountID,
		ID:           invoiceID,
		IsPaidInFull: isPaidInFull,
		IsOverPaid:   isOverPaid,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

// GetInvoicePaymentTotals returns each invoice's invoiced total and the sum of its allocations, from which its payment flags are decided.
func (r *settlementRepoImpl) GetInvoicePaymentTotals(ctx context.Context, accountID string, invoiceIDs []string) ([]domain.PaymentTotals, *apierror.APIError) {
	ctx, span := settlementRepoTracer.Start(ctx, "repository.settlement.get_invoice_payment_totals")
	defer span.End()

	if len(invoiceIDs) == 0 {
		return nil, nil
	}

	rows, err := r.queries.GetInvoicePaymentTotals(ctx, sqlc.GetInvoicePaymentTotalsParams{AccountID: accountID, InvoiceIds: invoiceIDs})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	totals := make([]domain.PaymentTotals, len(rows))
	for i, row := range rows {
		totals[i] = domain.PaymentTotals{ID: row.InvoiceID, Total: decimalOrZero(row.InvoicedTotal), Allocated: decimalOrZero(row.AllocatedTotal)}
	}
	return totals, nil
}

// GetTransactionAllocationTotals returns each transaction's amount and the sum of its allocations, from which its fully-allocated flag is decided.
func (r *settlementRepoImpl) GetTransactionAllocationTotals(ctx context.Context, accountID string, transactionIDs []string) ([]domain.PaymentTotals, *apierror.APIError) {
	ctx, span := settlementRepoTracer.Start(ctx, "repository.settlement.get_transaction_allocation_totals")
	defer span.End()

	if len(transactionIDs) == 0 {
		return nil, nil
	}

	rows, err := r.queries.GetTransactionAllocationTotals(ctx, sqlc.GetTransactionAllocationTotalsParams{AccountID: accountID, TransactionIds: transactionIDs})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	totals := make([]domain.PaymentTotals, len(rows))
	for i, row := range rows {
		totals[i] = domain.PaymentTotals{ID: row.TransactionID, Total: decimalOrZero(row.Amount), Allocated: decimalOrZero(row.AllocatedTotal)}
	}
	return totals, nil
}

// decimalOrZero reads a DECIMAL expression sqlc could only type as any, as text; NULL reads as zero.
func decimalOrZero(v any) string {
	if s := decimalStringPtr(v); s != nil {
		return *s
	}
	return "0"
}
