package repository

import (
	"context"
	gosql "database/sql"
	"math/big"
	"strings"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/pagination"
	"github.com/open-mrp/api/shared/tracing"
)

var transactionAllocationRepoTracer = tracing.GetTracer("core-service.transaction_allocation_repository")

type transactionAllocationRepoImpl struct {
	queries *sqlc.Queries
}

func NewTransactionAllocationRepo(queries *sqlc.Queries) domain.TransactionAllocationRepo {
	return &transactionAllocationRepoImpl{queries: queries}
}

func allocationEntryCreatedAt(d *domain.AllocationEntry) time.Time { return d.CreatedAt }
func allocationEntryID(d *domain.AllocationEntry) string           { return d.ID }

func buildAllocationSearchQuery(query *string) gosql.NullString {
	if query == nil {
		return gosql.NullString{}
	}
	q := strings.TrimSpace(*query)
	if q == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: q, Valid: true}
}

// searchLike is the query for a contains match on the customer name, with LIKE wildcards escaped.
func searchLike(q gosql.NullString) any {
	if !q.Valid {
		return nil
	}
	return db.EscapeLike(q.String)
}

func (r *transactionAllocationRepoImpl) ListEntries(ctx context.Context, params domain.ListAllocationEntriesParams) (*domain.ListAllocationEntriesResult, *apierror.APIError) {
	ctx, span := transactionAllocationRepoTracer.Start(ctx, "repository.transaction_allocation.list_entries")
	defer span.End()

	searchQuery := buildAllocationSearchQuery(params.Query)

	startDate := gosql.NullTime{}
	if params.StartDate != nil {
		startDate = gosql.NullTime{Time: *params.StartDate, Valid: true}
	}
	endDate := gosql.NullTime{}
	if params.EndDate != nil {
		endDate = gosql.NullTime{Time: *params.EndDate, Valid: true}
	}

	transactionType := gosql.NullString{}
	if params.TransactionType != nil {
		transactionType = gosql.NullString{String: *params.TransactionType, Valid: true}
	}

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationErrorWithParam("Invalid pagination cursor.", "cursor")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListAllocationEntriesBackward(ctx, sqlc.ListAllocationEntriesBackwardParams{
				AccountID:       params.AccountID,
				SearchQuery:     searchQuery,
				SearchLike:      searchLike(searchQuery),
				TransactionType: transactionType,
				StartDate:       startDate,
				EndDate:         endDate,
				CursorCreatedAt: gosql.NullTime{Time: cur.OccurredAt, Valid: true},
				CursorID:        gosql.NullString{String: cur.ID, Valid: true},
				Limit:           params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			entries := make([]*domain.AllocationEntry, len(rows))
			for i, row := range rows {
				entries[i] = mapBackwardAllocationEntryRow(row)
			}
			result, pageInfo := pagination.BuildPageString(entries, params.Limit, cursorDir, allocationEntryCreatedAt, allocationEntryID)
			return &domain.ListAllocationEntriesResult{Entries: result, PageInfo: pageInfo}, nil
		}

		rows, err := r.queries.ListAllocationEntriesForward(ctx, sqlc.ListAllocationEntriesForwardParams{
			AccountID:       params.AccountID,
			SearchQuery:     searchQuery,
			SearchLike:      searchLike(searchQuery),
			TransactionType: transactionType,
			StartDate:       startDate,
			EndDate:         endDate,
			CursorCreatedAt: gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:        gosql.NullString{String: cur.ID, Valid: true},
			Limit:           params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		entries := make([]*domain.AllocationEntry, len(rows))
		for i, row := range rows {
			entries[i] = mapForwardAllocationEntryRow(row)
		}
		result, pageInfo := pagination.BuildPageString(entries, params.Limit, cursorDir, allocationEntryCreatedAt, allocationEntryID)
		return &domain.ListAllocationEntriesResult{Entries: result, PageInfo: pageInfo}, nil
	}

	// No cursor - forward from beginning
	rows, err := r.queries.ListAllocationEntriesForward(ctx, sqlc.ListAllocationEntriesForwardParams{
		AccountID:       params.AccountID,
		SearchQuery:     searchQuery,
		SearchLike:      searchLike(searchQuery),
		TransactionType: transactionType,
		StartDate:       startDate,
		EndDate:         endDate,
		CursorCreatedAt: gosql.NullTime{},
		CursorID:        gosql.NullString{},
		Limit:           params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	entries := make([]*domain.AllocationEntry, len(rows))
	for i, row := range rows {
		entries[i] = mapForwardAllocationEntryRow(row)
	}
	result, pageInfo := pagination.BuildPageString(entries, params.Limit, cursorDir, allocationEntryCreatedAt, allocationEntryID)
	return &domain.ListAllocationEntriesResult{Entries: result, PageInfo: pageInfo}, nil
}

func mapForwardAllocationEntryRow(row sqlc.ListAllocationEntriesForwardRow) *domain.AllocationEntry {
	entry := &domain.AllocationEntry{
		ID:              row.ID,
		AmountValue:     decimalToString(row.AmountValue),
		AmountUnitAbbr:  row.AmountUnitAbbreviation,
		CustomerID:      row.CustomerID,
		CustomerName:    row.CustomerName,
		TransactionID:   row.TransactionID,
		TransactionType: row.TransactionType,
		InvoiceID:       row.InvoiceID,
		InvoiceNumber:   row.InvoiceNumber,
		CreatedAt:       row.CreatedAt,
	}
	if row.CustomerNumber.Valid {
		entry.CustomerNumber = &row.CustomerNumber.String
	}
	if row.Note.Valid {
		entry.Note = &row.Note.String
	}
	if row.TransactionMethod.Valid {
		entry.TransactionMethod = &row.TransactionMethod.String
	}
	if row.AdjustmentType.Valid {
		entry.AdjustmentType = &row.AdjustmentType.String
	}
	return entry
}

func mapBackwardAllocationEntryRow(row sqlc.ListAllocationEntriesBackwardRow) *domain.AllocationEntry {
	entry := &domain.AllocationEntry{
		ID:              row.ID,
		AmountValue:     decimalToString(row.AmountValue),
		AmountUnitAbbr:  row.AmountUnitAbbreviation,
		CustomerID:      row.CustomerID,
		CustomerName:    row.CustomerName,
		TransactionID:   row.TransactionID,
		TransactionType: row.TransactionType,
		InvoiceID:       row.InvoiceID,
		InvoiceNumber:   row.InvoiceNumber,
		CreatedAt:       row.CreatedAt,
	}
	if row.CustomerNumber.Valid {
		entry.CustomerNumber = &row.CustomerNumber.String
	}
	if row.Note.Valid {
		entry.Note = &row.Note.String
	}
	if row.TransactionMethod.Valid {
		entry.TransactionMethod = &row.TransactionMethod.String
	}
	if row.AdjustmentType.Valid {
		entry.AdjustmentType = &row.AdjustmentType.String
	}
	return entry
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
