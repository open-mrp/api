package repository

import (
	"context"
	gosql "database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/safeconv"
	"github.com/augno/api/shared/tracing"
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

func buildSettlementSearchQuery(query *string) gosql.NullString {
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

func (r *settlementRepoImpl) List(ctx context.Context, params domain.ListSettlementsParams) (*domain.ListSettlementsResult, *apierror.APIError) {
	ctx, span := settlementRepoTracer.Start(ctx, "repository.settlement.list")
	defer span.End()

	searchQuery := buildSettlementSearchQuery(params.Query)

	startDate := gosql.NullTime{}
	if params.StartDate != nil {
		startDate = gosql.NullTime{Time: *params.StartDate, Valid: true}
	}
	endDate := gosql.NullTime{}
	if params.EndDate != nil {
		endDate = gosql.NullTime{Time: *params.EndDate, Valid: true}
	}

	includeTransactionFilter := len(params.TransactionIDs) > 0
	transactionIDs := params.TransactionIDs
	if len(transactionIDs) == 0 {
		transactionIDs = []string{""}
	}

	includeInvoiceFilter := len(params.InvoiceIDs) > 0
	invoiceIDs := params.InvoiceIDs
	if len(invoiceIDs) == 0 {
		invoiceIDs = []string{""}
	}

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListSettlementsBackward(ctx, sqlc.ListSettlementsBackwardParams{
				AccountID:                params.AccountID,
				SearchQuery:              searchQuery,
				SearchQuery_2:            searchQuery,
				IncludeTransactionFilter: includeTransactionFilter,
				TransactionIds:           transactionIDs,
				IncludeInvoiceFilter:     includeInvoiceFilter,
				InvoiceIds:               invoiceIDs,
				StartDate:                startDate,
				EndDate:                  endDate,
				CursorCreatedAt:          gosql.NullTime{Time: cur.OccurredAt, Valid: true},
				CursorID:                 gosql.NullString{String: cur.ID, Valid: true},
				Limit:                    params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			settlements := make([]*domain.SettlementSummary, len(rows))
			for i, row := range rows {
				settlements[i] = mapBackwardSettlementRow(row)
			}
			result, pageInfo := pagination.BuildPageString(settlements, params.Limit, cursorDir, settlementCreatedAt, settlementID)
			return &domain.ListSettlementsResult{Settlements: result, PageInfo: pageInfo}, nil
		}

		rows, err := r.queries.ListSettlementsForward(ctx, sqlc.ListSettlementsForwardParams{
			AccountID:                params.AccountID,
			SearchQuery:              searchQuery,
			SearchQuery_2:            searchQuery,
			IncludeTransactionFilter: includeTransactionFilter,
			TransactionIds:           transactionIDs,
			IncludeInvoiceFilter:     includeInvoiceFilter,
			InvoiceIds:               invoiceIDs,
			StartDate:                startDate,
			EndDate:                  endDate,
			CursorCreatedAt:          gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:                 gosql.NullString{String: cur.ID, Valid: true},
			Limit:                    params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		settlements := make([]*domain.SettlementSummary, len(rows))
		for i, row := range rows {
			settlements[i] = mapForwardSettlementRow(row)
		}
		result, pageInfo := pagination.BuildPageString(settlements, params.Limit, cursorDir, settlementCreatedAt, settlementID)
		return &domain.ListSettlementsResult{Settlements: result, PageInfo: pageInfo}, nil
	}

	// No cursor - forward from beginning
	rows, err := r.queries.ListSettlementsForward(ctx, sqlc.ListSettlementsForwardParams{
		AccountID:                params.AccountID,
		SearchQuery:              searchQuery,
		SearchQuery_2:            searchQuery,
		IncludeTransactionFilter: includeTransactionFilter,
		TransactionIds:           transactionIDs,
		IncludeInvoiceFilter:     includeInvoiceFilter,
		InvoiceIds:               invoiceIDs,
		StartDate:                startDate,
		EndDate:                  endDate,
		CursorCreatedAt:          gosql.NullTime{},
		CursorID:                 gosql.NullString{},
		Limit:                    params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	settlements := make([]*domain.SettlementSummary, len(rows))
	for i, row := range rows {
		settlements[i] = mapForwardSettlementRow(row)
	}
	result, pageInfo := pagination.BuildPageString(settlements, params.Limit, cursorDir, settlementCreatedAt, settlementID)
	return &domain.ListSettlementsResult{Settlements: result, PageInfo: pageInfo}, nil
}

func mapForwardSettlementRow(row sqlc.ListSettlementsForwardRow) *domain.SettlementSummary {
	s := &domain.SettlementSummary{
		ID:              row.ID,
		Number:          row.Number,
		AllocationCount: safeconv.Int64ToInt32(row.AllocationCount),
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}

	tp := decimalToString(row.TotalPayments)
	if tp != "0" {
		s.TotalPayments = &tp
	}
	tr := decimalToString(row.TotalRebates)
	if tr != "0" {
		s.TotalRebates = &tr
	}
	ta := decimalToString(row.TotalAdjustments)
	if ta != "0" {
		s.TotalAdjustments = &ta
	}
	tc := decimalToString(row.TotalCredits)
	if tc != "0" {
		s.TotalCredits = &tc
	}

	if row.InvoiceNumbers.Valid {
		s.InvoiceNumbers = splitGroupConcat(row.InvoiceNumbers.String)
	}
	if row.CustomerNames.Valid {
		s.CustomerNames = splitGroupConcat(row.CustomerNames.String)
	}

	return s
}

func mapBackwardSettlementRow(row sqlc.ListSettlementsBackwardRow) *domain.SettlementSummary {
	s := &domain.SettlementSummary{
		ID:              row.ID,
		Number:          row.Number,
		AllocationCount: safeconv.Int64ToInt32(row.AllocationCount),
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}

	tp := decimalToString(row.TotalPayments)
	if tp != "0" {
		s.TotalPayments = &tp
	}
	tr := decimalToString(row.TotalRebates)
	if tr != "0" {
		s.TotalRebates = &tr
	}
	ta := decimalToString(row.TotalAdjustments)
	if ta != "0" {
		s.TotalAdjustments = &ta
	}
	tc := decimalToString(row.TotalCredits)
	if tc != "0" {
		s.TotalCredits = &tc
	}

	if row.InvoiceNumbers.Valid {
		s.InvoiceNumbers = splitGroupConcat(row.InvoiceNumbers.String)
	}
	if row.CustomerNames.Valid {
		s.CustomerNames = splitGroupConcat(row.CustomerNames.String)
	}

	return s
}

func splitGroupConcat(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}

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

	rows, err := r.queries.GetSettlementAllocations(ctx, gosql.NullString{String: settlementID, Valid: true})
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

func (r *settlementRepoImpl) GetNextSettlementNumber(ctx context.Context, accountID string) (int64, *apierror.APIError) {
	ctx, span := settlementRepoTracer.Start(ctx, "repository.settlement.get_next_number")
	defer span.End()

	next, err := r.queries.GetNextSettlementNumber(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}
	switch v := next.(type) {
	case int64:
		return v, nil
	case int32:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case []uint8:
		var n int64
		for _, b := range v {
			n = n*10 + int64(b-'0')
		}
		return n, nil
	default:
		return 0, tracing.Trace(span, apierror.NewInternalError(fmt.Errorf("unexpected type %T for settlement number", next), "Failed to parse settlement number."))
	}
}

func (r *settlementRepoImpl) UpdateNextSettlementNumber(ctx context.Context, sysPropertyID, accountID string, value int64) *apierror.APIError {
	ctx, span := settlementRepoTracer.Start(ctx, "repository.settlement.update_next_number")
	defer span.End()

	err := r.queries.UpdateNextSettlementNumber(ctx, sqlc.UpdateNextSettlementNumberParams{
		ID:        sysPropertyID,
		AccountID: accountID,
		Value:     safeconv.Int64ToInt32(value),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *settlementRepoImpl) DeleteOrphanedAdjustmentTransactions(ctx context.Context, settlementID string) *apierror.APIError {
	ctx, span := settlementRepoTracer.Start(ctx, "repository.settlement.delete_orphaned_adjustments")
	defer span.End()

	err := r.queries.DeleteOrphanedAdjustmentTransactions(ctx, sqlc.DeleteOrphanedAdjustmentTransactionsParams{
		SettlementID: gosql.NullString{String: settlementID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *settlementRepoImpl) UpdateTransactionsFullyAllocated(ctx context.Context, transactionIDs []string, isFullyAllocated bool) *apierror.APIError {
	ctx, span := settlementRepoTracer.Start(ctx, "repository.settlement.update_transactions_fully_allocated")
	defer span.End()

	if len(transactionIDs) == 0 {
		return nil
	}

	err := r.queries.UpdateTransactionsFullyAllocated(ctx, sqlc.UpdateTransactionsFullyAllocatedParams{
		IsFullyAllocated: isFullyAllocated,
		TransactionIds:   transactionIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *settlementRepoImpl) UpdateInvoicePaymentStatus(ctx context.Context, invoiceID string, isPaidInFull, isOverPaid bool) *apierror.APIError {
	ctx, span := settlementRepoTracer.Start(ctx, "repository.settlement.update_invoice_payment_status")
	defer span.End()

	err := r.queries.UpdateInvoicePaymentStatus(ctx, sqlc.UpdateInvoicePaymentStatusParams{
		ID:           invoiceID,
		IsPaidInFull: isPaidInFull,
		IsOverPaid:   isOverPaid,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}
