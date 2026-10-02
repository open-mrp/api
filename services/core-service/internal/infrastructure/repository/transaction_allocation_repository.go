package repository

import (
	"context"
	"math/big"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var transactionAllocationRepoTracer = tracing.GetTracer("core-service.transaction_allocation_repository")

type transactionAllocationRepoImpl struct {
	queries *sqlc.Queries
}

func NewTransactionAllocationRepo(queries *sqlc.Queries) domain.TransactionAllocationRepo {
	return &transactionAllocationRepoImpl{queries: queries}
}

func (r *transactionAllocationRepoImpl) GetByID(ctx context.Context, accountID, allocationID string) (*domain.TransactionAllocation, *apierror.APIError) {
	ctx, span := transactionAllocationRepoTracer.Start(ctx, "repository.transaction_allocation.get_by_id")
	defer span.End()

	allocations, apiErr := loadAllocations(ctx, r.queries.DB(), "ta.id = ? AND t.account_id = ?", "ta.id", allocationID, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if len(allocations) == 0 {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Transaction allocation not found."))
	}
	return allocations[0], nil
}

// UpdateCreatedAt re-dates an allocation.
func (r *transactionAllocationRepoImpl) UpdateCreatedAt(ctx context.Context, accountID, allocationID string, createdAt time.Time) *apierror.APIError {
	ctx, span := transactionAllocationRepoTracer.Start(ctx, "repository.transaction_allocation.update_created_at")
	defer span.End()

	_, err := r.queries.DB().ExecContext(ctx, `
UPDATE transaction_allocation ta
JOIN `+"`transaction`"+` t ON t.id = ta.transaction_id
SET ta.created_at = ?, ta.updated_at = NOW(3)
WHERE ta.id = ? AND t.account_id = ?`, createdAt, allocationID, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *transactionAllocationRepoImpl) UpdateAmount(ctx context.Context, amountID, newValue string) *apierror.APIError {
	ctx, span := transactionAllocationRepoTracer.Start(ctx, "repository.transaction_allocation.update_amount")
	defer span.End()

	err := r.queries.UpdateAllocationAmount(ctx, sqlc.UpdateAllocationAmountParams{
		ID:    amountID,
		Value: newValue,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *transactionAllocationRepoImpl) Delete(ctx context.Context, accountID, allocationID string) *apierror.APIError {
	ctx, span := transactionAllocationRepoTracer.Start(ctx, "repository.transaction_allocation.delete")
	defer span.End()

	// Delete the quantity first
	err := r.queries.DeleteTransactionAllocationQuantity(ctx, allocationID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// Then delete the allocation
	err = r.queries.DeleteTransactionAllocation(ctx, sqlc.DeleteTransactionAllocationParams{
		ID:        allocationID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *transactionAllocationRepoImpl) GetDollarUnitID(ctx context.Context) (string, *apierror.APIError) {
	ctx, span := transactionAllocationRepoTracer.Start(ctx, "repository.transaction_allocation.get_dollar_unit_id")
	defer span.End()

	unitID, err := r.queries.GetDollarUnitID(ctx)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}
	return unitID, nil
}

// subtractDecimalStrings subtracts b from a using arbitrary precision, returning the result as a string.
func subtractDecimalStrings(a, b string) string {
	aRat, ok := new(big.Rat).SetString(a)
	if !ok {
		return "0"
	}
	bRat, ok := new(big.Rat).SetString(b)
	if !ok {
		return "0"
	}
	result := new(big.Rat).Sub(aRat, bRat)
	return result.FloatString(30)
}
