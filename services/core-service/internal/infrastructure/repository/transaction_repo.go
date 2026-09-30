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
	createdAt, fundsReceivedAt *time.Time,
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
		FundsReceivedAt:       toNullTime(fundsReceivedAt),
		CreatedAt:             toNullTime(createdAt),
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
	} else if params.ClearNote {
		updateParams.UpdateNote = gosql.NullString{String: "1", Valid: true}
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
	updateParams.CreatedAt = toNullTime(params.CreatedAt)
	updateParams.FundsReceivedAt = toNullTime(params.FundsReceivedAt)
	updateParams.ClearFundsReceivedAt = params.ClearFundsReceivedAt
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

func (r *transactionRepoImpl) ResolveResponsibleUserID(ctx context.Context, accountID, userOrAccountUserID string) (string, bool, *apierror.APIError) {
	ctx, span := transactionRepoTracer.Start(ctx, "repository.transaction.resolve_responsible_user_id")
	defer span.End()

	row, err := r.queries.ResolveResponsibleUserID(ctx, sqlc.ResolveResponsibleUserIDParams{
		AccountID:           accountID,
		UserOrAccountUserID: userOrAccountUserID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", false, tracing.Trace(span, apiErr)
	}
	return row.ID, row.IsActive.Valid && row.IsActive.Bool, nil
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
