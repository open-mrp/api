package repository

import (
	"context"
	gosql "database/sql"
	"fmt"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/safeconv"
	"github.com/augno/api/shared/tracing"
)

var invoiceRepoTracer = tracing.GetTracer("core-service.invoice_repository")

type invoiceRepoImpl struct {
	queries *sqlc.Queries
}

func NewInvoiceRepo(queries *sqlc.Queries) domain.InvoiceRepo {
	return &invoiceRepoImpl{queries: queries}
}

func invoiceCreatedAt(d *domain.InvoiceSummary) time.Time { return d.CreatedAt }
func invoiceID(d *domain.InvoiceSummary) string           { return d.ID }

func customerInvoiceCreatedAt(d *domain.InvoiceForPayment) time.Time { return d.CreatedAt }
func customerInvoiceID(d *domain.InvoiceForPayment) string           { return d.ID }

func buildInvoiceSearchParams(query *string) gosql.NullString {
	if query == nil || *query == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: "%" + db.EscapeLike(*query) + "%", Valid: true}
}

func toNullStringSlice(ids []string) []gosql.NullString {
	if len(ids) == 0 {
		return []gosql.NullString{{}}
	}
	result := make([]gosql.NullString, len(ids))
	for i, id := range ids {
		result[i] = gosql.NullString{String: id, Valid: true}
	}
	return result
}

func decimalToString(v any) string {
	if v == nil {
		return "0"
	}
	switch val := v.(type) {
	case []byte:
		return string(val)
	case string:
		return val
	default:
		return fmt.Sprintf("%v", val)
	}
}

func (r *invoiceRepoImpl) CountSince(ctx context.Context, accountID string, since time.Time) (int64, *apierror.APIError) {
	ctx, span := invoiceRepoTracer.Start(ctx, "repository.invoice.count_since")
	defer span.End()

	count, err := r.queries.CountInvoicesByAccountSince(ctx, sqlc.CountInvoicesByAccountSinceParams{
		AccountID: accountID,
		Since:     since,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}
	return count, nil
}

func (r *invoiceRepoImpl) List(ctx context.Context, params domain.ListInvoicesParams) (*domain.ListInvoicesResult, *apierror.APIError) {
	ctx, span := invoiceRepoTracer.Start(ctx, "repository.invoice.list")
	defer span.End()

	searchQuery := buildInvoiceSearchParams(params.Query)

	status := gosql.NullString{}
	if params.Status != nil && *params.Status != "" && *params.Status != "all" {
		status = gosql.NullString{String: *params.Status, Valid: true}
	}

	startDate := gosql.NullTime{}
	if params.StartDate != nil {
		startDate = gosql.NullTime{Time: *params.StartDate, Valid: true}
	}
	endDate := gosql.NullTime{}
	if params.EndDate != nil {
		endDate = gosql.NullTime{Time: *params.EndDate, Valid: true}
	}

	includeItemFilter := len(params.ItemIDs) > 0
	itemIDs := toNullStringSlice(params.ItemIDs)

	includeCustomerFilter := len(params.CustomerIDs) > 0
	customerIDs := params.CustomerIDs
	if len(customerIDs) == 0 {
		customerIDs = []string{""}
	}

	includeProductLineFilter := len(params.ProductLineIDs) > 0
	productLineIDs := toNullStringSlice(params.ProductLineIDs)

	includeCustomerGroupFilter := len(params.CustomerGroupIDs) > 0
	customerGroupIDs := toNullStringSlice(params.CustomerGroupIDs)

	includeSalesRepFilter := len(params.SalesRepIDs) > 0
	salesRepIDs := toNullStringSlice(params.SalesRepIDs)

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationErrorWithParam("Invalid pagination cursor.", "cursor")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListInvoicesBackward(ctx, sqlc.ListInvoicesBackwardParams{
				AccountID:                  params.AccountID,
				SearchQuery:                searchQuery,
				Status:                     status,
				IncludeItemFilter:          includeItemFilter,
				ItemIds:                    itemIDs,
				IncludeCustomerFilter:      includeCustomerFilter,
				CustomerIds:                customerIDs,
				IncludeProductLineFilter:   includeProductLineFilter,
				ProductLineIds:             productLineIDs,
				IncludeCustomerGroupFilter: includeCustomerGroupFilter,
				CustomerGroupIds:           customerGroupIDs,
				IncludeSalesRepFilter:      includeSalesRepFilter,
				SalesRepIds:                salesRepIDs,
				StartDate:                  startDate,
				EndDate:                    endDate,
				CursorCreatedAt:            cur.OccurredAt,
				CursorID:                   cur.ID,
				Limit:                      params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			invoices := make([]*domain.InvoiceSummary, len(rows))
			for i, row := range rows {
				invoices[i] = mapBackwardInvoiceRow(row)
			}
			result, pageInfo := pagination.BuildPageString(invoices, params.Limit, cursorDir, invoiceCreatedAt, invoiceID)
			return &domain.ListInvoicesResult{Invoices: result, PageInfo: pageInfo}, nil
		}

		// Forward with cursor
		rows, err := r.queries.ListInvoicesForward(ctx, sqlc.ListInvoicesForwardParams{
			AccountID:                  params.AccountID,
			SearchQuery:                searchQuery,
			Status:                     status,
			IncludeItemFilter:          includeItemFilter,
			ItemIds:                    itemIDs,
			IncludeCustomerFilter:      includeCustomerFilter,
			CustomerIds:                customerIDs,
			IncludeProductLineFilter:   includeProductLineFilter,
			ProductLineIds:             productLineIDs,
			IncludeCustomerGroupFilter: includeCustomerGroupFilter,
			CustomerGroupIds:           customerGroupIDs,
			IncludeSalesRepFilter:      includeSalesRepFilter,
			SalesRepIds:                salesRepIDs,
			StartDate:                  startDate,
			EndDate:                    endDate,
			CursorCreatedAt:            gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:                   gosql.NullString{String: cur.ID, Valid: true},
			Limit:                      params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		invoices := make([]*domain.InvoiceSummary, len(rows))
		for i, row := range rows {
			invoices[i] = mapForwardInvoiceRow(row)
		}
		result, pageInfo := pagination.BuildPageString(invoices, params.Limit, cursorDir, invoiceCreatedAt, invoiceID)
		return &domain.ListInvoicesResult{Invoices: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListInvoicesForward(ctx, sqlc.ListInvoicesForwardParams{
		AccountID:                  params.AccountID,
		SearchQuery:                searchQuery,
		Status:                     status,
		IncludeItemFilter:          includeItemFilter,
		ItemIds:                    itemIDs,
		IncludeCustomerFilter:      includeCustomerFilter,
		CustomerIds:                customerIDs,
		IncludeProductLineFilter:   includeProductLineFilter,
		ProductLineIds:             productLineIDs,
		IncludeCustomerGroupFilter: includeCustomerGroupFilter,
		CustomerGroupIds:           customerGroupIDs,
		IncludeSalesRepFilter:      includeSalesRepFilter,
		SalesRepIds:                salesRepIDs,
		StartDate:                  startDate,
		EndDate:                    endDate,
		Limit:                      params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	invoices := make([]*domain.InvoiceSummary, len(rows))
	for i, row := range rows {
		invoices[i] = mapForwardInvoiceRow(row)
	}
	result, pageInfo := pagination.BuildPageString(invoices, params.Limit, cursorDir, invoiceCreatedAt, invoiceID)
	return &domain.ListInvoicesResult{Invoices: result, PageInfo: pageInfo}, nil
}

func (r *invoiceRepoImpl) Get(ctx context.Context, params domain.GetInvoiceParams) (*domain.Invoice, *apierror.APIError) {
	ctx, span := invoiceRepoTracer.Start(ctx, "repository.invoice.get")
	defer span.End()

	row, err := r.queries.GetInvoice(ctx, sqlc.GetInvoiceParams{
		ID:        params.InvoiceID,
		AccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	invoice := &domain.Invoice{
		ID:                    row.ID,
		Number:                row.Number,
		OrderID:               row.OrderID,
		OrderNumber:           row.OrderNumber,
		CustomerID:            row.CustomerID,
		BillingAddressID:      row.BillingAddressID,
		BillingAddressCountry: row.BillingAddressCountry,
		IsPaidInFull:          row.IsPaidInFull,
		IsOverPaid:            row.IsOverPaid,
		IsEdiSent:             row.IsEdiSent,
		HasBeenSent:           row.HasBeenSent,
		AcceptsInvoiceEmails:  row.AcceptsInvoiceEmails != 0,
		CreatedAt:             row.CreatedAt,
		UpdatedAt:             row.UpdatedAt,
	}

	if row.PaymentTermID.Valid {
		invoice.PaymentTermID = &row.PaymentTermID.String
	}

	if row.Note.Valid {
		invoice.Note = &row.Note.String
	}
	invoice.BillingAddressName = &row.BillingAddressName
	if row.BillingAddressLine1.Valid {
		invoice.BillingAddressLine1 = &row.BillingAddressLine1.String
	}
	if row.BillingAddressLine2.Valid {
		invoice.BillingAddressLine2 = &row.BillingAddressLine2.String
	}
	if row.BillingAddressCity.Valid {
		invoice.BillingAddressCity = &row.BillingAddressCity.String
	}
	if row.BillingAddressState.Valid {
		invoice.BillingAddressState = &row.BillingAddressState.String
	}
	if row.BillingAddressZip.Valid {
		invoice.BillingAddressZip = &row.BillingAddressZip.String
	}
	if row.ShipmentID.Valid {
		invoice.ShipmentID = &row.ShipmentID.String
	}
	if row.ShipmentNumber.Valid {
		invoice.ShipmentNumber = &row.ShipmentNumber.String
	}

	return invoice, nil
}

func (r *invoiceRepoImpl) GetLines(ctx context.Context, invoiceID string) ([]*domain.InvoiceLine, *apierror.APIError) {
	ctx, span := invoiceRepoTracer.Start(ctx, "repository.invoice.get_lines")
	defer span.End()

	rows, err := r.queries.GetInvoiceLines(ctx, invoiceID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	lines := make([]*domain.InvoiceLine, len(rows))
	for i, row := range rows {
		line := &domain.InvoiceLine{
			ID:               row.ID,
			QuantityID:       row.QuantityID,
			QuantityValue:    row.QuantityValue,
			QuantityUnitID:   row.QuantityUnitID,
			QuantityUnitAbbr: row.QuantityUnitAbbreviation,
			UnitPriceID:      row.UnitPriceID,
			UnitPriceValue:   row.UnitPriceValue,
			UnitPriceNumUnit: row.UnitPriceNumeratorUnitID,
			UnitPriceDenUnit: row.UnitPriceDenominatorUnitID,
			OrderLineID:      row.OrderLineID,
			CreatedAt:        row.CreatedAt,
			UpdatedAt:        row.UpdatedAt,
		}
		if row.OrderLineItemID.Valid {
			line.OrderLineItemID = &row.OrderLineItemID.String
		}
		if row.OrderLineItemSku.Valid {
			line.OrderLineItemSKU = &row.OrderLineItemSku.String
		}
		lines[i] = line
	}

	return lines, nil
}

func (r *invoiceRepoImpl) GetAllocations(ctx context.Context, invoiceID string) ([]*domain.InvoiceAllocation, *apierror.APIError) {
	ctx, span := invoiceRepoTracer.Start(ctx, "repository.invoice.get_allocations")
	defer span.End()

	rows, err := r.queries.GetInvoiceAllocations(ctx, invoiceID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	allocations := make([]*domain.InvoiceAllocation, len(rows))
	for i, row := range rows {
		alloc := &domain.InvoiceAllocation{
			ID:             row.ID,
			TransactionID:  row.TransactionID,
			AmountID:       row.AmountID,
			AmountValue:    row.AmountValue,
			AmountUnitID:   row.AmountUnitID,
			AmountUnitAbbr: row.AmountUnitAbbreviation,
			CreatedAt:      row.CreatedAt,
			UpdatedAt:      row.UpdatedAt,
		}
		if row.Note.Valid {
			alloc.Note = &row.Note.String
		}
		allocations[i] = alloc
	}

	return allocations, nil
}

func (r *invoiceRepoImpl) Update(ctx context.Context, params domain.UpdateInvoiceParams) (*domain.InvoiceSummary, *apierror.APIError) {
	ctx, span := invoiceRepoTracer.Start(ctx, "repository.invoice.update")
	defer span.End()

	updateParams := sqlc.UpdateInvoiceParams{
		ID:        params.InvoiceID,
		AccountID: params.AccountID,
	}
	if params.Note != nil {
		updateParams.Note = gosql.NullString{String: *params.Note, Valid: true}
	}
	if params.HasBeenSent != nil {
		updateParams.HasBeenSent = gosql.NullBool{Bool: *params.HasBeenSent, Valid: true}
	}
	if params.IsEdiSent != nil {
		updateParams.IsEdiSent = gosql.NullBool{Bool: *params.IsEdiSent, Valid: true}
	}
	if params.IsPaidInFull != nil {
		updateParams.IsPaidInFull = gosql.NullBool{Bool: *params.IsPaidInFull, Valid: true}
	}

	err := r.queries.UpdateInvoice(ctx, updateParams)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Re-fetch the updated invoice summary
	row, err := r.queries.GetInvoiceSummaryByID(ctx, sqlc.GetInvoiceSummaryByIDParams{
		ID:        params.InvoiceID,
		AccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapSummaryByIDRow(row), nil
}

func (r *invoiceRepoImpl) IsDuplicateNumber(ctx context.Context, accountID, number string) (bool, *apierror.APIError) {
	ctx, span := invoiceRepoTracer.Start(ctx, "repository.invoice.is_duplicate_number")
	defer span.End()

	cnt, err := r.queries.IsDuplicateInvoiceNumber(ctx, sqlc.IsDuplicateInvoiceNumberParams{
		AccountID: accountID,
		Number:    number,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return cnt > 0, nil
}

func (r *invoiceRepoImpl) GetEmailRecipients(ctx context.Context, invoiceID string) ([]string, *apierror.APIError) {
	ctx, span := invoiceRepoTracer.Start(ctx, "repository.invoice.get_email_recipients")
	defer span.End()

	rows, err := r.queries.GetInvoiceEmailRecipients(ctx, invoiceID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	emails := make([]string, 0, len(rows))
	for _, row := range rows {
		if row.Valid {
			emails = append(emails, row.String)
		}
	}

	return emails, nil
}

func (r *invoiceRepoImpl) MarkEmailSent(ctx context.Context, accountID, invoiceID string) *apierror.APIError {
	ctx, span := invoiceRepoTracer.Start(ctx, "repository.invoice.mark_email_sent")
	defer span.End()

	err := r.queries.MarkInvoiceEmailSent(ctx, sqlc.MarkInvoiceEmailSentParams{
		InvoiceID: invoiceID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *invoiceRepoImpl) DeleteLinesByInvoice(ctx context.Context, invoiceID string) *apierror.APIError {
	ctx, span := invoiceRepoTracer.Start(ctx, "repository.invoice.delete_lines_by_invoice")
	defer span.End()

	err := r.queries.DeleteInvoiceLinesByInvoice(ctx, invoiceID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *invoiceRepoImpl) Delete(ctx context.Context, accountID, invoiceID string) *apierror.APIError {
	ctx, span := invoiceRepoTracer.Start(ctx, "repository.invoice.delete")
	defer span.End()

	err := r.queries.DeleteInvoice(ctx, sqlc.DeleteInvoiceParams{
		ID:        invoiceID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *invoiceRepoImpl) ListByCustomer(ctx context.Context, params domain.ListCustomerInvoicesParams) (*domain.ListCustomerInvoicesResult, *apierror.APIError) {
	ctx, span := invoiceRepoTracer.Start(ctx, "repository.invoice.list_by_customer")
	defer span.End()

	searchQuery := buildInvoiceSearchParams(params.Query)

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationErrorWithParam("Invalid pagination cursor.", "cursor")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListCustomerInvoicesBackward(ctx, sqlc.ListCustomerInvoicesBackwardParams{
				AccountID:            params.AccountID,
				IncludeChildAccounts: params.IncludeChildAccounts,
				CustomerAccountID:    params.CustomerAccountID,
				SearchQuery:          searchQuery,
				CursorCreatedAt:      cur.OccurredAt,
				CursorID:             cur.ID,
				Limit:                params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			invoices := make([]*domain.InvoiceForPayment, len(rows))
			for i, row := range rows {
				invoices[i] = mapBackwardCustomerInvoiceRow(row)
			}
			result, pageInfo := pagination.BuildPageString(invoices, params.Limit, cursorDir, customerInvoiceCreatedAt, customerInvoiceID)
			return &domain.ListCustomerInvoicesResult{Invoices: result, PageInfo: pageInfo}, nil
		}

		// Forward with cursor
		rows, err := r.queries.ListCustomerInvoicesForward(ctx, sqlc.ListCustomerInvoicesForwardParams{
			AccountID:            params.AccountID,
			IncludeChildAccounts: params.IncludeChildAccounts,
			CustomerAccountID:    params.CustomerAccountID,
			SearchQuery:          searchQuery,
			CursorCreatedAt:      gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:             gosql.NullString{String: cur.ID, Valid: true},
			Limit:                params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		invoices := make([]*domain.InvoiceForPayment, len(rows))
		for i, row := range rows {
			invoices[i] = mapForwardCustomerInvoiceRow(row)
		}
		result, pageInfo := pagination.BuildPageString(invoices, params.Limit, cursorDir, customerInvoiceCreatedAt, customerInvoiceID)
		return &domain.ListCustomerInvoicesResult{Invoices: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListCustomerInvoicesForward(ctx, sqlc.ListCustomerInvoicesForwardParams{
		AccountID:            params.AccountID,
		IncludeChildAccounts: params.IncludeChildAccounts,
		CustomerAccountID:    params.CustomerAccountID,
		SearchQuery:          searchQuery,
		Limit:                params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	invoices := make([]*domain.InvoiceForPayment, len(rows))
	for i, row := range rows {
		invoices[i] = mapForwardCustomerInvoiceRow(row)
	}
	result, pageInfo := pagination.BuildPageString(invoices, params.Limit, cursorDir, customerInvoiceCreatedAt, customerInvoiceID)
	return &domain.ListCustomerInvoicesResult{Invoices: result, PageInfo: pageInfo}, nil
}

// Mapping helpers

func mapForwardInvoiceRow(row sqlc.ListInvoicesForwardRow) *domain.InvoiceSummary {
	s := &domain.InvoiceSummary{
		ID:                    row.ID,
		Number:                row.Number,
		IsPaidInFull:          row.IsPaidInFull,
		IsEdiSent:             row.IsEdiSent,
		HasBeenSent:           row.HasBeenSent,
		CustomerID:            row.CustomerID,
		CustomerName:          row.CustomerName,
		CustomerNumber:        row.CustomerNumber,
		CustomerIsEdiEnabled:  row.CustomerIsEdiEnabled,
		OrderID:               row.OrderID,
		OrderNumber:           row.OrderNumber,
		BillingAddressID:      row.BillingAddressID,
		BillingAddressCountry: row.BillingAddressCountry,
		PriorityCode:          constants.PriorityCode(row.PriorityCode),
		LineCount:             safeconv.Int64ToInt32(row.LineCount),
		TotalInvoiced:         decimalToString(row.TotalInvoiced),
		AcceptsInvoiceEmails:  row.AcceptsInvoiceEmails != 0,
		CreatedAt:             row.CreatedAt,
		UpdatedAt:             row.UpdatedAt,
	}
	if row.Note.Valid {
		s.Note = &row.Note.String
	}
	if row.ShipmentID.Valid {
		s.ShipmentID = &row.ShipmentID.String
	}
	s.BillingAddressName = &row.BillingAddressName
	if row.BillingAddressLine1.Valid {
		s.BillingAddressLine1 = &row.BillingAddressLine1.String
	}
	if row.BillingAddressLine2.Valid {
		s.BillingAddressLine2 = &row.BillingAddressLine2.String
	}
	if row.BillingAddressCity.Valid {
		s.BillingAddressCity = &row.BillingAddressCity.String
	}
	if row.BillingAddressState.Valid {
		s.BillingAddressState = &row.BillingAddressState.String
	}
	if row.BillingAddressZip.Valid {
		s.BillingAddressZip = &row.BillingAddressZip.String
	}
	if row.PaymentTermID.Valid {
		s.PaymentTermID = &row.PaymentTermID.String
	}
	if row.PaymentTermName.Valid {
		s.PaymentTermName = &row.PaymentTermName.String
	}
	s.PaymentTermIsActive = nullBoolPtr(row.PaymentTermIsActive)
	s.CustomerStatusCode = nullStringToPtr(row.CustomerStatusCode)
	s.CustomerCommissionPolicy = nullStringToPtr(row.CustomerCommissionPolicy)
	return s
}

func mapBackwardInvoiceRow(row sqlc.ListInvoicesBackwardRow) *domain.InvoiceSummary {
	s := &domain.InvoiceSummary{
		ID:                    row.ID,
		Number:                row.Number,
		IsPaidInFull:          row.IsPaidInFull,
		IsEdiSent:             row.IsEdiSent,
		HasBeenSent:           row.HasBeenSent,
		CustomerID:            row.CustomerID,
		CustomerName:          row.CustomerName,
		CustomerNumber:        row.CustomerNumber,
		CustomerIsEdiEnabled:  row.CustomerIsEdiEnabled,
		OrderID:               row.OrderID,
		OrderNumber:           row.OrderNumber,
		BillingAddressID:      row.BillingAddressID,
		BillingAddressCountry: row.BillingAddressCountry,
		PriorityCode:          constants.PriorityCode(row.PriorityCode),
		LineCount:             safeconv.Int64ToInt32(row.LineCount),
		TotalInvoiced:         decimalToString(row.TotalInvoiced),
		AcceptsInvoiceEmails:  row.AcceptsInvoiceEmails != 0,
		CreatedAt:             row.CreatedAt,
		UpdatedAt:             row.UpdatedAt,
	}
	if row.Note.Valid {
		s.Note = &row.Note.String
	}
	if row.ShipmentID.Valid {
		s.ShipmentID = &row.ShipmentID.String
	}
	s.BillingAddressName = &row.BillingAddressName
	if row.BillingAddressLine1.Valid {
		s.BillingAddressLine1 = &row.BillingAddressLine1.String
	}
	if row.BillingAddressLine2.Valid {
		s.BillingAddressLine2 = &row.BillingAddressLine2.String
	}
	if row.BillingAddressCity.Valid {
		s.BillingAddressCity = &row.BillingAddressCity.String
	}
	if row.BillingAddressState.Valid {
		s.BillingAddressState = &row.BillingAddressState.String
	}
	if row.BillingAddressZip.Valid {
		s.BillingAddressZip = &row.BillingAddressZip.String
	}
	if row.PaymentTermID.Valid {
		s.PaymentTermID = &row.PaymentTermID.String
	}
	if row.PaymentTermName.Valid {
		s.PaymentTermName = &row.PaymentTermName.String
	}
	s.PaymentTermIsActive = nullBoolPtr(row.PaymentTermIsActive)
	s.CustomerStatusCode = nullStringToPtr(row.CustomerStatusCode)
	s.CustomerCommissionPolicy = nullStringToPtr(row.CustomerCommissionPolicy)
	return s
}

func mapSummaryByIDRow(row sqlc.GetInvoiceSummaryByIDRow) *domain.InvoiceSummary {
	s := &domain.InvoiceSummary{
		ID:                    row.ID,
		Number:                row.Number,
		IsPaidInFull:          row.IsPaidInFull,
		IsEdiSent:             row.IsEdiSent,
		HasBeenSent:           row.HasBeenSent,
		CustomerID:            row.CustomerID,
		CustomerName:          row.CustomerName,
		CustomerNumber:        row.CustomerNumber,
		CustomerIsEdiEnabled:  row.CustomerIsEdiEnabled,
		OrderID:               row.OrderID,
		OrderNumber:           row.OrderNumber,
		BillingAddressID:      row.BillingAddressID,
		BillingAddressCountry: row.BillingAddressCountry,
		PriorityCode:          constants.PriorityCode(row.PriorityCode),
		LineCount:             safeconv.Int64ToInt32(row.LineCount),
		TotalInvoiced:         decimalToString(row.TotalInvoiced),
		AcceptsInvoiceEmails:  row.AcceptsInvoiceEmails != 0,
		CreatedAt:             row.CreatedAt,
		UpdatedAt:             row.UpdatedAt,
	}
	if row.Note.Valid {
		s.Note = &row.Note.String
	}
	if row.ShipmentID.Valid {
		s.ShipmentID = &row.ShipmentID.String
	}
	s.BillingAddressName = &row.BillingAddressName
	if row.BillingAddressLine1.Valid {
		s.BillingAddressLine1 = &row.BillingAddressLine1.String
	}
	if row.BillingAddressLine2.Valid {
		s.BillingAddressLine2 = &row.BillingAddressLine2.String
	}
	if row.BillingAddressCity.Valid {
		s.BillingAddressCity = &row.BillingAddressCity.String
	}
	if row.BillingAddressState.Valid {
		s.BillingAddressState = &row.BillingAddressState.String
	}
	if row.BillingAddressZip.Valid {
		s.BillingAddressZip = &row.BillingAddressZip.String
	}
	if row.PaymentTermID.Valid {
		s.PaymentTermID = &row.PaymentTermID.String
	}
	if row.PaymentTermName.Valid {
		s.PaymentTermName = &row.PaymentTermName.String
	}
	s.PaymentTermIsActive = nullBoolPtr(row.PaymentTermIsActive)
	s.CustomerStatusCode = nullStringToPtr(row.CustomerStatusCode)
	s.CustomerCommissionPolicy = nullStringToPtr(row.CustomerCommissionPolicy)
	return s
}

func mapForwardCustomerInvoiceRow(row sqlc.ListCustomerInvoicesForwardRow) *domain.InvoiceForPayment {
	inv := &domain.InvoiceForPayment{
		ID:              row.ID,
		Number:          row.Number,
		CustomerID:      row.CustomerID,
		CustomerName:    row.CustomerName,
		CustomerNumber:  row.CustomerNumber,
		IsParentAccount: row.ParentAccountRelationID.Valid,
		InvoiceTotal:    decimalToString(row.TotalInvoiced),
		IsPaidInFull:    row.IsPaidInFull,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
	if row.CustomerPoNumber.Valid {
		inv.CustomerPO = &row.CustomerPoNumber.String
	}
	if row.ParentAccountID.Valid {
		inv.ParentAccountID = &row.ParentAccountID.String
	}
	if row.BillingAddressID.Valid {
		inv.BillingAddressID = &row.BillingAddressID.String
	}
	if row.BillingAddressName.Valid {
		inv.BillingAddressName = &row.BillingAddressName.String
	}
	inv.IsPrepaid = row.CustomerPaymentTermID.Valid && row.CustomerPaymentTermID.String == "prepaid"
	return inv
}

func mapBackwardCustomerInvoiceRow(row sqlc.ListCustomerInvoicesBackwardRow) *domain.InvoiceForPayment {
	inv := &domain.InvoiceForPayment{
		ID:              row.ID,
		Number:          row.Number,
		CustomerID:      row.CustomerID,
		CustomerName:    row.CustomerName,
		CustomerNumber:  row.CustomerNumber,
		IsParentAccount: row.ParentAccountRelationID.Valid,
		InvoiceTotal:    decimalToString(row.TotalInvoiced),
		IsPaidInFull:    row.IsPaidInFull,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
	if row.CustomerPoNumber.Valid {
		inv.CustomerPO = &row.CustomerPoNumber.String
	}
	if row.ParentAccountID.Valid {
		inv.ParentAccountID = &row.ParentAccountID.String
	}
	if row.BillingAddressID.Valid {
		inv.BillingAddressID = &row.BillingAddressID.String
	}
	if row.BillingAddressName.Valid {
		inv.BillingAddressName = &row.BillingAddressName.String
	}
	inv.IsPrepaid = row.CustomerPaymentTermID.Valid && row.CustomerPaymentTermID.String == "prepaid"
	return inv
}
