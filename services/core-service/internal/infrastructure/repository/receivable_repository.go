package repository

import (
	"context"
	gosql "database/sql"
	"fmt"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/tracing"
)

var receivableRepoTracer = tracing.GetTracer("core-service.repository.receivable")

type receivableRepoImpl struct {
	queries *sqlc.Queries
}

func NewReceivableRepo(queries *sqlc.Queries) domain.ReceivableRepo {
	return &receivableRepoImpl{queries: queries}
}

func receivableEntryCreatedAt(e domain.ReceivableEntry) time.Time { return e.InvoicedAt }
func receivableEntryID(e domain.ReceivableEntry) string           { return e.InvoiceID }

// Bounds which allocations net against the balance. A cutoff-free run still needs a comparable
// upper bound, so it gets a sentinel far enough out that every funded allocation clears it.
// Leaves the allocation window open when no cutoff is asked for. A far-future sentinel cannot serve
// here: nanosecond precision overflows MySQL's DATETIME, and the comparison then matches nothing.
func buildAllocationCutoffParam(cutoffDate *time.Time) gosql.NullTime {
	if cutoffDate == nil {
		return gosql.NullTime{}
	}
	return gosql.NullTime{Time: *cutoffDate, Valid: true}
}

// Reports whether settled entries must be dropped: only an as-of run prunes them, so a plain
// listing still shows every unpaid invoice.
func requirePositiveBalance(cutoffDate *time.Time) bool {
	return cutoffDate != nil
}

func buildCutoffDateParam(cutoffDate *time.Time) gosql.NullTime {
	if cutoffDate == nil {
		return gosql.NullTime{}
	}
	return gosql.NullTime{Time: *cutoffDate, Valid: true}
}

func buildReceivableSearchParam(query *string) gosql.NullString {
	if query == nil || *query == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: "%" + db.EscapeLike(*query) + "%", Valid: true}
}

func mapReceivableForwardRow(row sqlc.ListReceivablesForwardRow) domain.ReceivableEntry {
	return domain.ReceivableEntry{
		InvoiceID:        row.ID,
		InvoiceNumber:    row.InvoiceNumber,
		PONumber:         nullStringPtr(row.PoNumber),
		InvoicedAt:       row.CreatedAt,
		CustomerID:       row.CustomerID,
		CustomerNumber:   row.CustomerNumber,
		CustomerName:     row.CustomerName,
		RemainingBalance: fmt.Sprintf("%.2f", row.RemainingBalance),
		IsPaidInFull:     row.IsPaidInFull,
	}
}

func mapReceivableBackwardRow(row sqlc.ListReceivablesBackwardRow) domain.ReceivableEntry {
	return domain.ReceivableEntry{
		InvoiceID:        row.ID,
		InvoiceNumber:    row.InvoiceNumber,
		PONumber:         nullStringPtr(row.PoNumber),
		InvoicedAt:       row.CreatedAt,
		CustomerID:       row.CustomerID,
		CustomerNumber:   row.CustomerNumber,
		CustomerName:     row.CustomerName,
		RemainingBalance: fmt.Sprintf("%.2f", row.RemainingBalance),
		IsPaidInFull:     row.IsPaidInFull,
	}
}

func mapReceivableByCustomerForwardRow(row sqlc.ListReceivablesByCustomerForwardRow) domain.ReceivableEntry {
	return domain.ReceivableEntry{
		InvoiceID:        row.ID,
		InvoiceNumber:    row.InvoiceNumber,
		PONumber:         nullStringPtr(row.PoNumber),
		InvoicedAt:       row.CreatedAt,
		CustomerID:       row.CustomerID,
		CustomerNumber:   row.CustomerNumber,
		CustomerName:     row.CustomerName,
		RemainingBalance: fmt.Sprintf("%.2f", row.RemainingBalance),
		IsPaidInFull:     row.IsPaidInFull,
	}
}

func mapReceivableByCustomerBackwardRow(row sqlc.ListReceivablesByCustomerBackwardRow) domain.ReceivableEntry {
	return domain.ReceivableEntry{
		InvoiceID:        row.ID,
		InvoiceNumber:    row.InvoiceNumber,
		PONumber:         nullStringPtr(row.PoNumber),
		InvoicedAt:       row.CreatedAt,
		CustomerID:       row.CustomerID,
		CustomerNumber:   row.CustomerNumber,
		CustomerName:     row.CustomerName,
		RemainingBalance: fmt.Sprintf("%.2f", row.RemainingBalance),
		IsPaidInFull:     row.IsPaidInFull,
	}
}

func (r *receivableRepoImpl) List(ctx context.Context, params domain.ListReceivablesParams) (*domain.ListReceivablesResult, *apierror.APIError) {
	ctx, span := receivableRepoTracer.Start(ctx, "repository.receivable.list")
	defer span.End()

	allocationCutoff := buildAllocationCutoffParam(params.CutoffDate)
	positiveOnly := requirePositiveBalance(params.CutoffDate)
	cutoffDate := buildCutoffDateParam(params.CutoffDate)
	searchQuery := buildReceivableSearchParam(params.Query)

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationErrorWithParam("Invalid pagination cursor.", "cursor")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListReceivablesBackward(ctx, sqlc.ListReceivablesBackwardParams{
				AllocationCutoffDate:   allocationCutoff,
				RequirePositiveBalance: positiveOnly,
				AccountID:              params.AccountID,
				CutoffDate:             cutoffDate,
				SearchQuery:            searchQuery,
				CursorCreatedAt:        cur.OccurredAt,
				CursorID:               cur.ID,
				Limit:                  params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			items := make([]domain.ReceivableEntry, len(rows))
			for i, row := range rows {
				items[i] = mapReceivableBackwardRow(row)
			}
			result, pageInfo := pagination.BuildPageString(items, params.Limit, cursorDir, receivableEntryCreatedAt, receivableEntryID)
			return &domain.ListReceivablesResult{Items: result, PageString: pageInfo.NextCursor}, nil
		}

		// Forward with cursor
		rows, err := r.queries.ListReceivablesForward(ctx, sqlc.ListReceivablesForwardParams{
			AllocationCutoffDate:   allocationCutoff,
			RequirePositiveBalance: positiveOnly,
			AccountID:              params.AccountID,
			CutoffDate:             cutoffDate,
			SearchQuery:            searchQuery,
			CursorCreatedAt:        gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:               gosql.NullString{String: cur.ID, Valid: true},
			Limit:                  params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		items := make([]domain.ReceivableEntry, len(rows))
		for i, row := range rows {
			items[i] = mapReceivableForwardRow(row)
		}
		result, pageInfo := pagination.BuildPageString(items, params.Limit, cursorDir, receivableEntryCreatedAt, receivableEntryID)
		return &domain.ListReceivablesResult{Items: result, PageString: pageInfo.NextCursor}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListReceivablesForward(ctx, sqlc.ListReceivablesForwardParams{
		AllocationCutoffDate:   allocationCutoff,
		RequirePositiveBalance: positiveOnly,
		AccountID:              params.AccountID,
		CutoffDate:             cutoffDate,
		SearchQuery:            searchQuery,
		Limit:                  params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	items := make([]domain.ReceivableEntry, len(rows))
	for i, row := range rows {
		items[i] = mapReceivableForwardRow(row)
	}
	result, pageInfo := pagination.BuildPageString(items, params.Limit, cursorDir, receivableEntryCreatedAt, receivableEntryID)
	return &domain.ListReceivablesResult{Items: result, PageString: pageInfo.NextCursor}, nil
}

func (r *receivableRepoImpl) ListByCustomer(ctx context.Context, params domain.ListReceivablesByCustomerParams) (*domain.ListReceivablesByCustomerResult, *apierror.APIError) {
	ctx, span := receivableRepoTracer.Start(ctx, "repository.receivable.list_by_customer")
	defer span.End()

	allocationCutoff := buildAllocationCutoffParam(params.CutoffDate)
	positiveOnly := requirePositiveBalance(params.CutoffDate)
	cutoffDate := buildCutoffDateParam(params.CutoffDate)
	searchQuery := buildReceivableSearchParam(params.Query)

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationErrorWithParam("Invalid pagination cursor.", "cursor")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListReceivablesByCustomerBackward(ctx, sqlc.ListReceivablesByCustomerBackwardParams{
				AllocationCutoffDate:   allocationCutoff,
				RequirePositiveBalance: positiveOnly,
				AccountID:              params.AccountID,
				CustomerAccountID:      params.CustomerAccountID,
				CutoffDate:             cutoffDate,
				SearchQuery:            searchQuery,
				CursorCreatedAt:        cur.OccurredAt,
				CursorID:               cur.ID,
				Limit:                  params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			items := make([]domain.ReceivableEntry, len(rows))
			for i, row := range rows {
				items[i] = mapReceivableByCustomerBackwardRow(row)
			}
			result, pageInfo := pagination.BuildPageString(items, params.Limit, cursorDir, receivableEntryCreatedAt, receivableEntryID)
			return &domain.ListReceivablesByCustomerResult{Items: result, PageString: pageInfo.NextCursor}, nil
		}

		// Forward with cursor
		rows, err := r.queries.ListReceivablesByCustomerForward(ctx, sqlc.ListReceivablesByCustomerForwardParams{
			AllocationCutoffDate:   allocationCutoff,
			RequirePositiveBalance: positiveOnly,
			AccountID:              params.AccountID,
			CustomerAccountID:      params.CustomerAccountID,
			CutoffDate:             cutoffDate,
			SearchQuery:            searchQuery,
			CursorCreatedAt:        gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:               gosql.NullString{String: cur.ID, Valid: true},
			Limit:                  params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		items := make([]domain.ReceivableEntry, len(rows))
		for i, row := range rows {
			items[i] = mapReceivableByCustomerForwardRow(row)
		}
		result, pageInfo := pagination.BuildPageString(items, params.Limit, cursorDir, receivableEntryCreatedAt, receivableEntryID)
		return &domain.ListReceivablesByCustomerResult{Items: result, PageString: pageInfo.NextCursor}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListReceivablesByCustomerForward(ctx, sqlc.ListReceivablesByCustomerForwardParams{
		AllocationCutoffDate:   allocationCutoff,
		RequirePositiveBalance: positiveOnly,
		AccountID:              params.AccountID,
		CustomerAccountID:      params.CustomerAccountID,
		CutoffDate:             cutoffDate,
		SearchQuery:            searchQuery,
		Limit:                  params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	items := make([]domain.ReceivableEntry, len(rows))
	for i, row := range rows {
		items[i] = mapReceivableByCustomerForwardRow(row)
	}
	result, pageInfo := pagination.BuildPageString(items, params.Limit, cursorDir, receivableEntryCreatedAt, receivableEntryID)
	return &domain.ListReceivablesByCustomerResult{Items: result, PageString: pageInfo.NextCursor}, nil
}

func (r *receivableRepoImpl) ListAllByCustomer(ctx context.Context, accountID, customerAccountID string, cutoffDate *time.Time) ([]domain.ReceivableEntry, *apierror.APIError) {
	ctx, span := receivableRepoTracer.Start(ctx, "repository.receivable.list_all_by_customer")
	defer span.End()

	allocationCutoff := buildAllocationCutoffParam(cutoffDate)
	positiveOnly := requirePositiveBalance(cutoffDate)
	cutoffDateParam := buildCutoffDateParam(cutoffDate)

	rows, err := r.queries.ListReceivablesByCustomerForward(ctx, sqlc.ListReceivablesByCustomerForwardParams{
		AllocationCutoffDate:   allocationCutoff,
		RequirePositiveBalance: positiveOnly,
		AccountID:              accountID,
		CustomerAccountID:      customerAccountID,
		CutoffDate:             cutoffDateParam,
		Limit:                  10000,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	items := make([]domain.ReceivableEntry, len(rows))
	for i, row := range rows {
		items[i] = mapReceivableByCustomerForwardRow(row)
	}
	return items, nil
}

func (r *receivableRepoImpl) ListOpenCreditsByCustomer(ctx context.Context, accountID, customerAccountID string) ([]domain.OpenCredit, *apierror.APIError) {
	ctx, span := receivableRepoTracer.Start(ctx, "repository.receivable.list_open_credits_by_customer")
	defer span.End()

	rows, err := r.queries.GetOpenCreditsByCustomer(ctx, sqlc.GetOpenCreditsByCustomerParams{
		AccountID:         accountID,
		CustomerAccountID: customerAccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	items := make([]domain.OpenCredit, len(rows))
	for i, row := range rows {
		items[i] = domain.OpenCredit{
			ID:             row.ID,
			Number:         row.Number,
			CreatedAt:      row.CreatedAt,
			OriginalAmount: row.OriginalAmount,
			LeftoverAmount: fmt.Sprintf("%.2f", row.LeftoverAmount),
		}
	}
	return items, nil
}
