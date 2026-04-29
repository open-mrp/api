package repository

import (
	"context"
	gosql "database/sql"
	"math/big"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/tracing"
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
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListAllocationEntriesBackward(ctx, sqlc.ListAllocationEntriesBackwardParams{
				AccountID:       params.AccountID,
				SearchQuery:     searchQuery,
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

	row, err := r.queries.GetTransactionAllocationByID(ctx, sqlc.GetTransactionAllocationByIDParams{
		ID:        allocationID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	var note *string
	if row.Note.Valid {
		note = &row.Note.String
	}

	return &domain.TransactionAllocation{
		ID:                row.ID,
		AmountID:          row.AmountID,
		AmountValue:       decimalToString(row.AmountValue),
		AmountUnitID:      row.AmountUnitID,
		AmountUnitAbbr:    row.AmountUnitAbbreviation,
		Note:              note,
		TransactionID:     row.TransactionID,
		TransactionNumber: row.TransactionNumber,
		TransactionType:   row.TransactionType,
		InvoiceID:         row.InvoiceID,
		InvoiceNumber:     row.InvoiceNumber,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}, nil
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

func (r *transactionAllocationRepoImpl) ListOpenCredits(ctx context.Context, params domain.ListOpenCreditsParams) (*domain.ListOpenCreditsResult, *apierror.APIError) {
	ctx, span := transactionAllocationRepoTracer.Start(ctx, "repository.transaction_allocation.list_open_credits")
	defer span.End()

	startDate := gosql.NullTime{}
	if params.StartDate != nil {
		startDate = gosql.NullTime{Time: *params.StartDate, Valid: true}
	}
	endDate := gosql.NullTime{}
	if params.EndDate != nil {
		endDate = gosql.NullTime{Time: *params.EndDate, Valid: true}
	}

	includeCustomerFilter := len(params.CustomerIDs) > 0
	customerIDs := params.CustomerIDs
	if len(customerIDs) == 0 {
		customerIDs = []string{""}
	}

	lim := params.Limit
	if lim <= 0 {
		lim = 100
	}
	if lim > 1000 {
		lim = 1000
	}

	var search interface{}
	if params.SearchQuery != nil && *params.SearchQuery != "" {
		search = *params.SearchQuery
	}

	cursorCreatedAt := gosql.NullTime{}
	cursorID := gosql.NullString{}
	if params.Cursor != nil && *params.Cursor != "" {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewValidationError("Invalid pagination cursor."))
		}
		cursorCreatedAt = gosql.NullTime{Time: cur.OccurredAt, Valid: true}
		cursorID = gosql.NullString{String: cur.ID, Valid: true}
	}

	rows, err := r.queries.ListOpenCredits(ctx, sqlc.ListOpenCreditsParams{
		AccountID:             params.AccountID,
		IncludeCustomerFilter: includeCustomerFilter,
		CustomerIds:           customerIDs,
		StartDate:             startDate,
		EndDate:               endDate,
		Search:                search,
		CursorCreatedAt:       cursorCreatedAt,
		CursorID:              cursorID,
		Limit:                 lim + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if len(rows) == 0 {
		return &domain.ListOpenCreditsResult{Entries: []*domain.OpenCreditEntry{}}, nil
	}

	// Collect transaction IDs for allocation lookup
	transactionIDs := make([]string, len(rows))
	for i, row := range rows {
		transactionIDs[i] = row.ID
	}

	// Get allocations for all open credits
	allocRows, err := r.queries.GetOpenCreditAllocations(ctx, transactionIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Group allocations by transaction ID
	allocsByTxn := make(map[string][]domain.InvoiceAllocationEntry)
	for _, aRow := range allocRows {
		allocsByTxn[aRow.TransactionID] = append(allocsByTxn[aRow.TransactionID], domain.InvoiceAllocationEntry{
			InvoiceNumber: aRow.InvoiceNumber,
			Amount:        decimalToString(aRow.Amount),
		})
	}

	entries := make([]*domain.OpenCreditEntry, len(rows))
	for i, row := range rows {
		originalAmount := decimalToString(row.OriginalAmount)
		allocatedAmount := decimalToString(row.AllocatedAmount)
		leftoverAmount := subtractDecimalStrings(originalAmount, allocatedAmount)

		entry := &domain.OpenCreditEntry{
			ID:                 row.ID,
			Number:             row.Number,
			OriginalAmount:     originalAmount,
			AllocatedAmount:    allocatedAmount,
			LeftoverAmount:     leftoverAmount,
			CustomerName:       row.CustomerName,
			TransactionType:    row.TransactionType,
			InvoiceAllocations: allocsByTxn[row.ID],
			CreatedAt:          row.CreatedAt,
		}

		if row.CustomerNumber.Valid {
			entry.CustomerNumber = &row.CustomerNumber.String
		}
		if row.TransactionMethod.Valid {
			entry.TransactionMethod = &row.TransactionMethod.String
		}
		if row.AdjustmentType.Valid {
			entry.AdjustmentType = &row.AdjustmentType.String
		}
		if row.ResponsibleUserName != "" {
			entry.ResponsibleUserName = &row.ResponsibleUserName
		}
		if row.Note.Valid {
			entry.Note = &row.Note.String
		}
		if row.StripePaymentID.Valid {
			entry.StripePaymentID = &row.StripePaymentID.String
		}
		if entry.InvoiceAllocations == nil {
			entry.InvoiceAllocations = []domain.InvoiceAllocationEntry{}
		}

		entries[i] = entry
	}

	hasExtra := len(entries) > int(lim)
	if hasExtra {
		entries = entries[:lim]
	}
	var pi pagination.PageInfo
	pi.HasNextPage = hasExtra
	if pi.HasNextPage && len(entries) > 0 {
		last := entries[len(entries)-1]
		nc := pagination.EncodeStringCursor(pagination.StringCursor{
			OccurredAt: last.CreatedAt,
			ID:         last.ID,
			Direction:  pagination.DirectionForward,
		})
		pi.NextCursor = &nc
	}

	return &domain.ListOpenCreditsResult{Entries: entries, PageInfo: pi}, nil
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
