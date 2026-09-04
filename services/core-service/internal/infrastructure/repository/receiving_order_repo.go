package repository

import (
	"context"
	gosql "database/sql"
	"fmt"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/services/core-service/internal/ledgerlock"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/pagination"
	"github.com/open-mrp/api/shared/safeconv"
	"github.com/open-mrp/api/shared/tracing"
	"github.com/shopspring/decimal"
)

var receivingOrderRepoTracer = tracing.GetTracer("core-service.receiving_order_repository")

type receivingOrderRepoImpl struct {
	queries *sqlc.Queries
}

func NewReceivingOrderRepo(queries *sqlc.Queries) domain.ReceivingOrderRepo {
	return &receivingOrderRepoImpl{queries: queries}
}

func receivingOrderCreatedAt(d *domain.ReceivingOrderSummary) time.Time { return d.CreatedAt }
func receivingOrderID(d *domain.ReceivingOrderSummary) string           { return d.ID }

func buildReceivingOrderSearchParams(query *string) gosql.NullString {
	if query == nil || *query == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: "%" + db.EscapeLike(*query) + "%", Valid: true}
}

func receivingOrderInterfaceToString(v any) string {
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

func receivingOrderInterfaceToFloat64(v any) float64 {
	s := receivingOrderInterfaceToString(v)
	d, err := decimal.NewFromString(s)
	if err != nil {
		return 0
	}
	f, _ := d.Float64()
	return f
}

// --- Existing methods ---

func (r *receivingOrderRepoImpl) Create(ctx context.Context, id, number, orderID, accountID string) *apierror.APIError {
	ctx, span := receivingOrderRepoTracer.Start(ctx, "repository.receiving_order.create")
	defer span.End()

	err := r.queries.CreateReceivingOrder(ctx, sqlc.CreateReceivingOrderParams{
		ID:        id,
		Number:    number,
		OrderID:   orderID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *receivingOrderRepoImpl) CreateLine(ctx context.Context, id, receivingOrderID, quantityID, salesOrderLineID string) *apierror.APIError {
	ctx, span := receivingOrderRepoTracer.Start(ctx, "repository.receiving_order.create_line")
	defer span.End()

	err := r.queries.CreateReceivingOrderLine(ctx, sqlc.CreateReceivingOrderLineParams{
		ID:               id,
		ReceivingOrderID: receivingOrderID,
		QuantityID:       quantityID,
		SalesOrderLineID: salesOrderLineID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *receivingOrderRepoImpl) GetByOrderID(ctx context.Context, orderID string) (*string, *apierror.APIError) {
	ctx, span := receivingOrderRepoTracer.Start(ctx, "repository.receiving_order.get_by_order_id")
	defer span.End()

	id, err := r.queries.GetReceivingOrderByOrderID(ctx, orderID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			return nil, nil
		}
		return nil, tracing.Trace(span, apiErr)
	}

	return &id, nil
}

func (r *receivingOrderRepoImpl) DeleteLinesByOrderID(ctx context.Context, orderID string) *apierror.APIError {
	ctx, span := receivingOrderRepoTracer.Start(ctx, "repository.receiving_order.delete_lines_by_order_id")
	defer span.End()

	err := r.queries.DeleteReceivingOrderLinesByOrderID(ctx, orderID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *receivingOrderRepoImpl) DeleteByOrderID(ctx context.Context, orderID string) *apierror.APIError {
	ctx, span := receivingOrderRepoTracer.Start(ctx, "repository.receiving_order.delete_by_order_id")
	defer span.End()

	err := r.queries.DeleteReceivingOrderByOrderID(ctx, orderID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *receivingOrderRepoImpl) MarkComplete(ctx context.Context, orderID string) *apierror.APIError {
	ctx, span := receivingOrderRepoTracer.Start(ctx, "repository.receiving_order.mark_complete")
	defer span.End()

	err := r.queries.MarkReceivingOrderComplete(ctx, orderID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *receivingOrderRepoImpl) MarkIncomplete(ctx context.Context, orderID string) *apierror.APIError {
	ctx, span := receivingOrderRepoTracer.Start(ctx, "repository.receiving_order.mark_incomplete")
	defer span.End()

	err := r.queries.MarkReceivingOrderIncomplete(ctx, orderID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *receivingOrderRepoImpl) DeleteLinesByOrderLineID(ctx context.Context, salesOrderLineID string) *apierror.APIError {
	ctx, span := receivingOrderRepoTracer.Start(ctx, "repository.receiving_order.delete_lines_by_order_line_id")
	defer span.End()

	err := r.queries.DeleteReceivingOrderLinesByOrderLineID(ctx, salesOrderLineID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

// --- New methods ---

func (r *receivingOrderRepoImpl) List(ctx context.Context, params domain.ListReceivingOrdersParams) (*domain.ListReceivingOrdersResult, *apierror.APIError) {
	ctx, span := receivingOrderRepoTracer.Start(ctx, "repository.receiving_order.list")
	defer span.End()

	searchQuery := buildReceivingOrderSearchParams(params.Query)

	var status any
	if params.Status != nil && *params.Status != "" && *params.Status != "all" {
		status = *params.Status
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
	itemIDs := make([]gosql.NullString, len(params.ItemIDs))
	for i, itemID := range params.ItemIDs {
		itemIDs[i] = gosql.NullString{String: itemID, Valid: true}
	}
	if len(itemIDs) == 0 {
		itemIDs = []gosql.NullString{{}}
	}

	includeSupplierFilter := len(params.SupplierIDs) > 0
	supplierIDs := params.SupplierIDs
	if len(supplierIDs) == 0 {
		supplierIDs = []string{""}
	}

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationErrorWithParam("Invalid pagination cursor.", "cursor")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListReceivingOrdersBackward(ctx, sqlc.ListReceivingOrdersBackwardParams{
				AccountID:             params.AccountID,
				SearchQuery:           searchQuery,
				Status:                status,
				IncludeItemFilter:     includeItemFilter,
				ItemIds:               itemIDs,
				IncludeSupplierFilter: includeSupplierFilter,
				SupplierIds:           supplierIDs,
				StartDate:             startDate,
				EndDate:               endDate,
				CursorCreatedAt:       cur.OccurredAt,
				CursorID:              cur.ID,
				Limit:                 params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			orders := make([]*domain.ReceivingOrderSummary, len(rows))
			for i, row := range rows {
				orders[i] = mapBackwardReceivingOrderRow(row)
			}
			result, pageInfo := pagination.BuildPageString(orders, params.Limit, cursorDir, receivingOrderCreatedAt, receivingOrderID)
			if apiErr := r.attachTotals(ctx, result); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			return &domain.ListReceivingOrdersResult{ReceivingOrders: result, PageInfo: pageInfo}, nil
		}

		// Forward with cursor
		rows, err := r.queries.ListReceivingOrdersForward(ctx, sqlc.ListReceivingOrdersForwardParams{
			AccountID:             params.AccountID,
			SearchQuery:           searchQuery,
			Status:                status,
			IncludeItemFilter:     includeItemFilter,
			ItemIds:               itemIDs,
			IncludeSupplierFilter: includeSupplierFilter,
			SupplierIds:           supplierIDs,
			StartDate:             startDate,
			EndDate:               endDate,
			CursorCreatedAt:       gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:              gosql.NullString{String: cur.ID, Valid: true},
			Limit:                 params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		orders := make([]*domain.ReceivingOrderSummary, len(rows))
		for i, row := range rows {
			orders[i] = mapForwardReceivingOrderRow(row)
		}
		if apiErr := r.attachTotals(ctx, orders); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		result, pageInfo := pagination.BuildPageString(orders, params.Limit, cursorDir, receivingOrderCreatedAt, receivingOrderID)
		return &domain.ListReceivingOrdersResult{ReceivingOrders: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListReceivingOrdersForward(ctx, sqlc.ListReceivingOrdersForwardParams{
		AccountID:             params.AccountID,
		SearchQuery:           searchQuery,
		Status:                status,
		IncludeItemFilter:     includeItemFilter,
		ItemIds:               itemIDs,
		IncludeSupplierFilter: includeSupplierFilter,
		SupplierIds:           supplierIDs,
		StartDate:             startDate,
		EndDate:               endDate,
		Limit:                 params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	orders := make([]*domain.ReceivingOrderSummary, len(rows))
	for i, row := range rows {
		orders[i] = mapForwardReceivingOrderRow(row)
	}
	if apiErr := r.attachTotals(ctx, orders); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	result, pageInfo := pagination.BuildPageString(orders, params.Limit, cursorDir, receivingOrderCreatedAt, receivingOrderID)
	return &domain.ListReceivingOrdersResult{ReceivingOrders: result, PageInfo: pageInfo}, nil
}

// fetchReceivingOrderTotals aggregates the given orders' lines in one query, keyed by order id.
//
// Orders whose lines carry no price, or that have no lines at all, are simply absent from the result; callers treat a missing entry as zero rather than as an error.
func (r *receivingOrderRepoImpl) fetchReceivingOrderTotals(ctx context.Context, orderIDs []string) (map[string]*domain.ReceivingOrderTotals, *apierror.APIError) {
	if len(orderIDs) == 0 {
		return map[string]*domain.ReceivingOrderTotals{}, nil
	}

	rows, err := r.queries.GetReceivingOrderTotals(ctx, orderIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}

	totals := make(map[string]*domain.ReceivingOrderTotals, len(rows))
	for _, row := range rows {
		totals[row.ReceivingOrderID] = &domain.ReceivingOrderTotals{
			OrderedAmount:  sqlValueToString(row.OrderedAmount),
			StockedAmount:  sqlValueToString(row.StockedAmount),
			RejectedAmount: sqlValueToString(row.RejectedAmount),
		}
	}
	return totals, nil
}

func (r *receivingOrderRepoImpl) Get(ctx context.Context, accountID, receivingOrderID string) (*domain.ReceivingOrder, *apierror.APIError) {
	ctx, span := receivingOrderRepoTracer.Start(ctx, "repository.receiving_order.get")
	defer span.End()

	row, err := r.queries.GetReceivingOrderByID(ctx, sqlc.GetReceivingOrderByIDParams{
		ID:        receivingOrderID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	lineRows, err := r.queries.ListReceivingOrderLinesByOrderID(ctx, receivingOrderID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	lines := make([]*domain.ReceivingOrderLine, len(lineRows))
	for i, lr := range lineRows {
		lines[i] = mapReceivingOrderLineRow(lr)
	}

	var completedAt *time.Time
	if row.CompletedAt.Valid {
		completedAt = &row.CompletedAt.Time
	}
	var supplierID *string
	if row.SupplierID.Valid {
		supplierID = &row.SupplierID.String
	}
	var supplierName *string
	if row.SupplierName.Valid {
		supplierName = &row.SupplierName.String
	}
	var supplierNumber *string
	if row.SupplierNumber.Valid {
		supplierNumber = &row.SupplierNumber.String
	}
	var note *string
	if row.Note.Valid {
		note = &row.Note.String
	}

	totals, apiErr := r.fetchReceivingOrderTotals(ctx, []string{row.ID})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	deliveries, apiErr := r.fetchDeliveryRefs(ctx, []string{row.PurchaseOrderID})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return &domain.ReceivingOrder{
		ID:                  row.ID,
		Number:              row.Number,
		PurchaseOrderID:     row.PurchaseOrderID,
		PurchaseOrderNumber: row.PurchaseOrderNumber,
		PurchaseOrderStatus: row.PurchaseOrderStatus,
		SupplierID:          supplierID,
		SupplierName:        supplierName,
		SupplierNumber:      supplierNumber,
		Note:                note,
		Lines:               lines,
		Totals:              totals[row.ID],
		Deliveries:          deliveries[row.PurchaseOrderID],
		CompletedAt:         completedAt,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}, nil
}

func (r *receivingOrderRepoImpl) ListLines(ctx context.Context, receivingOrderID string) ([]*domain.ReceivingOrderLine, *apierror.APIError) {
	ctx, span := receivingOrderRepoTracer.Start(ctx, "repository.receiving_order.list_lines")
	defer span.End()

	rows, err := r.queries.ListReceivingOrderLinesByOrderID(ctx, receivingOrderID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	lines := make([]*domain.ReceivingOrderLine, len(rows))
	for i, row := range rows {
		lines[i] = mapReceivingOrderLineRow(row)
	}

	return lines, nil
}

func (r *receivingOrderRepoImpl) FindUnstockedLineIDs(ctx context.Context, receivingOrderID, accountID string, enforceNonZero bool) ([]domain.UnstockedLine, *apierror.APIError) {
	ctx, span := receivingOrderRepoTracer.Start(ctx, "repository.receiving_order.find_unstocked_line_ids")
	defer span.End()

	rows, err := r.queries.FindUnstockedLineIDs(ctx, sqlc.FindUnstockedLineIDsParams{
		ReceivingOrderID: receivingOrderID,
		AccountID:        accountID,
		EnforceNonZero:   enforceNonZero,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	result := make([]domain.UnstockedLine, len(rows))
	for i, row := range rows {
		result[i] = domain.UnstockedLine{
			ID:          row.ID,
			OrderLineID: row.OrderLineID,
		}
	}

	return result, nil
}

func (r *receivingOrderRepoImpl) StockLines(ctx context.Context, lineIDs []string, accountID string) *apierror.APIError {
	ctx, span := receivingOrderRepoTracer.Start(ctx, "repository.receiving_order.stock_lines")
	defer span.End()

	err := r.queries.StockReceivingOrderLines(ctx, lineIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *receivingOrderRepoImpl) MarkCompleteIfAllStocked(ctx context.Context, receivingOrderID, accountID string) (bool, *apierror.APIError) {
	ctx, span := receivingOrderRepoTracer.Start(ctx, "repository.receiving_order.mark_complete_if_all_stocked")
	defer span.End()

	unstockedCount, err := r.queries.CheckAllLinesStocked(ctx, sqlc.CheckAllLinesStockedParams{
		ReceivingOrderID: receivingOrderID,
		AccountID:        accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	if unstockedCount > 0 {
		return false, nil
	}

	err = r.queries.MarkReceivingOrderCompleteByID(ctx, sqlc.MarkReceivingOrderCompleteByIDParams{
		ID:        receivingOrderID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return true, nil
}

func (r *receivingOrderRepoImpl) MarkIncompleteByID(ctx context.Context, receivingOrderID, accountID string) *apierror.APIError {
	ctx, span := receivingOrderRepoTracer.Start(ctx, "repository.receiving_order.mark_incomplete_by_id")
	defer span.End()

	err := r.queries.MarkReceivingOrderIncompleteByID(ctx, sqlc.MarkReceivingOrderIncompleteByIDParams{
		ID:        receivingOrderID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *receivingOrderRepoImpl) BulkCreateForRemainingQuantities(ctx context.Context, receivingOrderID string, orderLineIDs []string, accountID string) *apierror.APIError {
	ctx, span := receivingOrderRepoTracer.Start(ctx, "repository.receiving_order.bulk_create_for_remaining_quantities")
	defer span.End()

	rows, err := r.queries.GetOrderedQuantityForLine(ctx, orderLineIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	for _, row := range rows {
		ordered, oErr := decimal.NewFromString(row.OrderedValue)
		if oErr != nil {
			return tracing.Trace(span, apierror.NewInternalError(oErr, "Failed to parse ordered quantity value."))
		}

		receivedStr := receivingOrderInterfaceToString(row.ReceivedTotal)
		received, rErr := decimal.NewFromString(receivedStr)
		if rErr != nil {
			return tracing.Trace(span, apierror.NewInternalError(rErr, "Failed to parse received quantity value."))
		}

		remaining := ordered.Sub(received)
		if remaining.LessThanOrEqual(decimal.Zero) {
			continue
		}

		quantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
		if apiErr != nil {
			return tracing.Trace(span, apiErr)
		}

		lineID, apiErr := id.GenID(id.ReceivingOrderLineIDPrefix, nil)
		if apiErr != nil {
			return tracing.Trace(span, apiErr)
		}

		if createErr := r.queries.CreateQuantity(ctx, sqlc.CreateQuantityParams{
			ID:     quantityID,
			Value:  remaining.String(),
			UnitID: row.UnitID,
		}); createErr != nil {
			if apiErr := db.MapSQLError(createErr); apiErr != nil {
				return tracing.Trace(span, apiErr)
			}
		}

		if createErr := r.queries.CreateReceivingOrderLine(ctx, sqlc.CreateReceivingOrderLineParams{
			ID:               lineID,
			ReceivingOrderID: receivingOrderID,
			QuantityID:       quantityID,
			SalesOrderLineID: row.OrderLineID,
		}); createErr != nil {
			if apiErr := db.MapSQLError(createErr); apiErr != nil {
				return tracing.Trace(span, apiErr)
			}
		}
	}

	return nil
}

func (r *receivingOrderRepoImpl) BulkReceiveRemainingQuantities(ctx context.Context, receivingOrderID string, orderLineIDs []string, accountID string) *apierror.APIError {
	ctx, span := receivingOrderRepoTracer.Start(ctx, "repository.receiving_order.bulk_receive_remaining_quantities")
	defer span.End()

	rows, err := r.queries.GetOrderedQuantityForLine(ctx, orderLineIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// Build a map of order line ID -> remaining quantity
	remainingByOrderLine := make(map[string]decimal.Decimal)
	for _, row := range rows {
		ordered, oErr := decimal.NewFromString(row.OrderedValue)
		if oErr != nil {
			return tracing.Trace(span, apierror.NewInternalError(oErr, "Failed to parse ordered quantity value."))
		}

		receivedStr := receivingOrderInterfaceToString(row.ReceivedTotal)
		received, rErr := decimal.NewFromString(receivedStr)
		if rErr != nil {
			return tracing.Trace(span, apierror.NewInternalError(rErr, "Failed to parse received quantity value."))
		}

		remaining := ordered.Sub(received)
		if remaining.GreaterThan(decimal.Zero) {
			remainingByOrderLine[row.OrderLineID] = remaining
		}
	}

	if len(remainingByOrderLine) == 0 {
		return nil
	}

	// Find unstocked lines for this receiving order
	unstockedLines, apiErr := r.FindUnstockedLineIDs(ctx, receivingOrderID, accountID, false)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// Update the quantity of each unstocked line to the remaining amount
	for _, line := range unstockedLines {
		remaining, ok := remainingByOrderLine[line.OrderLineID]
		if !ok {
			continue
		}

		updateErr := r.queries.UpdateReceivingOrderLineQuantity(ctx, sqlc.UpdateReceivingOrderLineQuantityParams{
			QuantityValue: remaining.String(),
			LineID:        line.ID,
		})
		if updateApiErr := db.MapSQLError(updateErr); updateApiErr != nil {
			return tracing.Trace(span, updateApiErr)
		}
	}

	return nil
}

func (r *receivingOrderRepoImpl) VoidAllLines(ctx context.Context, receivingOrderID, accountID string) *apierror.APIError {
	ctx, span := receivingOrderRepoTracer.Start(ctx, "repository.receiving_order.void_all_lines")
	defer span.End()

	err := r.queries.VoidAllReceivingOrderLines(ctx, sqlc.VoidAllReceivingOrderLinesParams{
		ReceivingOrderID: receivingOrderID,
		AccountID:        accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *receivingOrderRepoImpl) DeleteDuplicateLines(ctx context.Context, receivingOrderID, accountID string) *apierror.APIError {
	ctx, span := receivingOrderRepoTracer.Start(ctx, "repository.receiving_order.delete_duplicate_lines")
	defer span.End()

	err := r.queries.DeleteDuplicateReceivingOrderLines(ctx, sqlc.DeleteDuplicateReceivingOrderLinesParams{
		ReceivingOrderID: receivingOrderID,
		AccountID:        accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *receivingOrderRepoImpl) UpdateLineQuantity(ctx context.Context, lineID string, quantityValue string) *apierror.APIError {
	ctx, span := receivingOrderRepoTracer.Start(ctx, "repository.receiving_order.update_line_quantity")
	defer span.End()

	err := r.queries.UpdateReceivingOrderLineQuantity(ctx, sqlc.UpdateReceivingOrderLineQuantityParams{
		QuantityValue: quantityValue,
		LineID:        lineID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *receivingOrderRepoImpl) VoidLine(ctx context.Context, lineID, accountID string) *apierror.APIError {
	ctx, span := receivingOrderRepoTracer.Start(ctx, "repository.receiving_order.void_line")
	defer span.End()

	err := r.queries.VoidReceivingOrderLine(ctx, sqlc.VoidReceivingOrderLineParams{
		LineID:    lineID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *receivingOrderRepoImpl) GetLine(ctx context.Context, lineID string) (*domain.ReceivingOrderLine, *apierror.APIError) {
	ctx, span := receivingOrderRepoTracer.Start(ctx, "repository.receiving_order.get_line")
	defer span.End()

	row, err := r.queries.GetReceivingOrderLine(ctx, lineID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapGetReceivingOrderLineRow(row), nil
}

func (r *receivingOrderRepoImpl) IsLineInReceivingOrder(ctx context.Context, lineID, receivingOrderID string) (bool, *apierror.APIError) {
	ctx, span := receivingOrderRepoTracer.Start(ctx, "repository.receiving_order.is_line_in_receiving_order")
	defer span.End()

	exists, err := r.queries.IsReceivingOrderLineInOrder(ctx, sqlc.IsReceivingOrderLineInOrderParams{
		LineID:           lineID,
		ReceivingOrderID: receivingOrderID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return exists, nil
}

func (r *receivingOrderRepoImpl) CalculateQuantityYetToBeReceived(ctx context.Context, lineID, accountID string) (string, string, *apierror.APIError) {
	ctx, span := receivingOrderRepoTracer.Start(ctx, "repository.receiving_order.calculate_quantity_yet_to_be_received")
	defer span.End()

	row, err := r.queries.CalculateQuantityYetToBeReceived(ctx, lineID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", "", tracing.Trace(span, apiErr)
	}

	ordered, oErr := decimal.NewFromString(row.OrderedValue)
	if oErr != nil {
		return "", "", tracing.Trace(span, apierror.NewInternalError(oErr, "Failed to parse ordered quantity value."))
	}

	receivedStr := receivingOrderInterfaceToString(row.ReceivedTotal)
	received, rErr := decimal.NewFromString(receivedStr)
	if rErr != nil {
		return "", "", tracing.Trace(span, apierror.NewInternalError(rErr, "Failed to parse received quantity value."))
	}

	remaining := ordered.Sub(received)
	if remaining.LessThan(decimal.Zero) {
		remaining = decimal.Zero
	}

	return remaining.String(), row.UnitID, nil
}

func (r *receivingOrderRepoImpl) IsInAccount(ctx context.Context, accountID, receivingOrderID string) (bool, *apierror.APIError) {
	ctx, span := receivingOrderRepoTracer.Start(ctx, "repository.receiving_order.is_in_account")
	defer span.End()

	exists, err := r.queries.IsReceivingOrderInAccount(ctx, sqlc.IsReceivingOrderInAccountParams{
		ReceivingOrderID: receivingOrderID,
		AccountID:        accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return exists, nil
}

func (r *receivingOrderRepoImpl) GetLineUnitPrices(ctx context.Context, receivingOrderID string) ([]domain.ReceivingOrderLineUnitPrice, *apierror.APIError) {
	ctx, span := receivingOrderRepoTracer.Start(ctx, "repository.receiving_order.get_line_unit_prices")
	defer span.End()

	rows, err := r.queries.GetReceivingOrderLineUnitPrice(ctx, receivingOrderID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	result := make([]domain.ReceivingOrderLineUnitPrice, len(rows))
	for i, row := range rows {
		itemID := ""
		if row.ItemID.Valid {
			itemID = row.ItemID.String
		}
		result[i] = domain.ReceivingOrderLineUnitPrice{
			ReceivingOrderLineID:       row.ReceivingOrderLineID,
			ItemID:                     itemID,
			UnitPriceValue:             row.UnitPriceValue,
			UnitPriceNumeratorUnitID:   row.UnitPriceNumeratorUnitID,
			UnitPriceDenominatorUnitID: row.UnitPriceDenominatorUnitID,
			QuantityUnitID:             row.QuantityUnitID,
		}
	}

	return result, nil
}

func (r *receivingOrderRepoImpl) GetPurchaseOrderID(ctx context.Context, receivingOrderID, accountID string) (string, *apierror.APIError) {
	ctx, span := receivingOrderRepoTracer.Start(ctx, "repository.receiving_order.get_purchase_order_id")
	defer span.End()

	poID, err := r.queries.GetPurchaseOrderIDForReceivingOrder(ctx, sqlc.GetPurchaseOrderIDForReceivingOrderParams{
		ReceivingOrderID: receivingOrderID,
		AccountID:        accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	return poID, nil
}

func (r *receivingOrderRepoImpl) UpsertLot(ctx context.Context, lotID, accountID, itemID, lotNumber string) (string, *apierror.APIError) {
	ctx, span := receivingOrderRepoTracer.Start(ctx, "repository.receiving_order.upsert_lot")
	defer span.End()

	// INSERT IGNORE — if the lot already exists, the insert is silently skipped.
	err := r.queries.UpsertLot(ctx, sqlc.UpsertLotParams{
		ID:        lotID,
		AccountID: accountID,
		ItemID:    itemID,
		LotNumber: lotNumber,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	// Fetch the actual lot ID (may be existing, not the one we passed).
	existingID, err := r.queries.GetLotByKey(ctx, sqlc.GetLotByKeyParams{
		AccountID: accountID,
		ItemID:    itemID,
		LotNumber: lotNumber,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	return existingID, nil
}

func (r *receivingOrderRepoImpl) LockItemForLedger(ctx context.Context, itemID string) *apierror.APIError {
	if err := r.queries.LockItemForLedger(ctx, itemID); err != nil {
		return db.MapSQLError(err)
	}
	return nil
}

func (r *receivingOrderRepoImpl) InsertInventoryReceiptForDelivery(ctx context.Context, scope *ledgerlock.Scope, receiptID, accountID, itemID, quantityID, unitCostID string, storageLocationID, lotID, orderID *string) *apierror.APIError {
	// The backstop; the acquisition belongs at the top of the caller's transaction. See ledgerlock.
	if apiErr := scope.EnsureLocked(ctx, r, itemID); apiErr != nil {
		return apiErr
	}

	ctx, span := receivingOrderRepoTracer.Start(ctx, "repository.receiving_order.insert_inventory_receipt_for_delivery")
	defer span.End()

	storageLocNull := gosql.NullString{}
	if storageLocationID != nil {
		storageLocNull = gosql.NullString{String: *storageLocationID, Valid: true}
	}
	lotNull := gosql.NullString{}
	if lotID != nil {
		lotNull = gosql.NullString{String: *lotID, Valid: true}
	}
	orderNull := gosql.NullString{}
	if orderID != nil {
		orderNull = gosql.NullString{String: *orderID, Valid: true}
	}

	err := r.queries.InsertInventoryReceiptForDelivery(ctx, sqlc.InsertInventoryReceiptForDeliveryParams{
		ID:                receiptID,
		AccountID:         accountID,
		ItemID:            itemID,
		QuantityID:        quantityID,
		UnitCostID:        unitCostID,
		StorageLocationID: storageLocNull,
		LotID:             lotNull,
		OrderID:           orderNull,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *receivingOrderRepoImpl) MarkPurchaseOrderFulfilled(ctx context.Context, purchaseOrderID, accountID string) *apierror.APIError {
	ctx, span := receivingOrderRepoTracer.Start(ctx, "repository.receiving_order.mark_purchase_order_fulfilled")
	defer span.End()

	err := r.queries.MarkPurchaseOrderFulfilled(ctx, sqlc.MarkPurchaseOrderFulfilledParams{
		ID:        purchaseOrderID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *receivingOrderRepoImpl) HasUnstockedLineForOrderLine(ctx context.Context, salesOrderLineID string) (bool, *apierror.APIError) {
	ctx, span := receivingOrderRepoTracer.Start(ctx, "repository.receiving_order.has_unstocked_line_for_order_line")
	defer span.End()

	hasUnstocked, err := r.queries.HasUnstockedReceivingOrderLineForOrderLine(ctx, salesOrderLineID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return hasUnstocked, nil
}

func (r *receivingOrderRepoImpl) CreateLineForRemainingQuantity(ctx context.Context, receivingOrderID, salesOrderLineID, accountID string) *apierror.APIError {
	ctx, span := receivingOrderRepoTracer.Start(ctx, "repository.receiving_order.create_line_for_remaining_quantity")
	defer span.End()

	// Check if there's already an unstocked receiving order line for this PO line
	hasUnstocked, apiErr := r.HasUnstockedLineForOrderLine(ctx, salesOrderLineID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if hasUnstocked {
		// An open receiving order line already exists for this PO line
		return nil
	}

	// Calculate remaining quantity
	rows, err := r.queries.GetOrderedQuantityForLine(ctx, []string{salesOrderLineID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if len(rows) == 0 {
		return nil
	}

	row := rows[0]
	ordered, oErr := decimal.NewFromString(row.OrderedValue)
	if oErr != nil {
		return tracing.Trace(span, apierror.NewInternalError(oErr, "Failed to parse ordered quantity value."))
	}

	receivedStr := receivingOrderInterfaceToString(row.ReceivedTotal)
	received, rErr := decimal.NewFromString(receivedStr)
	if rErr != nil {
		return tracing.Trace(span, apierror.NewInternalError(rErr, "Failed to parse received quantity value."))
	}

	remaining := ordered.Sub(received)
	if remaining.LessThanOrEqual(decimal.Zero) {
		return nil
	}

	quantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	lineID, apiErr := id.GenID(id.ReceivingOrderLineIDPrefix, nil)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if createErr := r.queries.CreateQuantity(ctx, sqlc.CreateQuantityParams{
		ID:     quantityID,
		Value:  remaining.String(),
		UnitID: row.UnitID,
	}); createErr != nil {
		if apiErr := db.MapSQLError(createErr); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	if createErr := r.queries.CreateReceivingOrderLine(ctx, sqlc.CreateReceivingOrderLineParams{
		ID:               lineID,
		ReceivingOrderID: receivingOrderID,
		QuantityID:       quantityID,
		SalesOrderLineID: salesOrderLineID,
	}); createErr != nil {
		if apiErr := db.MapSQLError(createErr); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	return nil
}

func (r *receivingOrderRepoImpl) GetAllocationSumForIssue(ctx context.Context, issueID string) (string, *apierror.APIError) {
	ctx, span := receivingOrderRepoTracer.Start(ctx, "repository.receiving_order.get_allocation_sum_for_issue")
	defer span.End()

	totalAllocated, err := r.queries.GetAllocationSumForIssue(ctx, issueID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "0", tracing.Trace(span, apiErr)
	}

	return receivingOrderInterfaceToString(totalAllocated), nil
}

// --- Mapping helpers ---

func mapForwardReceivingOrderRow(row sqlc.ListReceivingOrdersForwardRow) *domain.ReceivingOrderSummary {
	var completedAt *time.Time
	if row.CompletedAt.Valid {
		completedAt = &row.CompletedAt.Time
	}
	var supplierID *string
	if row.SupplierID.Valid {
		supplierID = &row.SupplierID.String
	}
	var supplierName *string
	if row.SupplierName.Valid {
		supplierName = &row.SupplierName.String
	}
	var supplierNumber *string
	if row.SupplierNumber.Valid {
		supplierNumber = &row.SupplierNumber.String
	}

	return &domain.ReceivingOrderSummary{
		ID:                  row.ID,
		Number:              row.Number,
		PurchaseOrderID:     row.PurchaseOrderID,
		PurchaseOrderNumber: row.PurchaseOrderNumber,
		PurchaseOrderStatus: row.PurchaseOrderStatus,
		SupplierID:          supplierID,
		SupplierName:        supplierName,
		SupplierNumber:      supplierNumber,
		LineCount:           safeconv.Int64ToInt32(row.LineCount),
		CompletedAt:         completedAt,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}
}

func mapBackwardReceivingOrderRow(row sqlc.ListReceivingOrdersBackwardRow) *domain.ReceivingOrderSummary {
	var completedAt *time.Time
	if row.CompletedAt.Valid {
		completedAt = &row.CompletedAt.Time
	}
	var supplierID *string
	if row.SupplierID.Valid {
		supplierID = &row.SupplierID.String
	}
	var supplierName *string
	if row.SupplierName.Valid {
		supplierName = &row.SupplierName.String
	}
	var supplierNumber *string
	if row.SupplierNumber.Valid {
		supplierNumber = &row.SupplierNumber.String
	}

	return &domain.ReceivingOrderSummary{
		ID:                  row.ID,
		Number:              row.Number,
		PurchaseOrderID:     row.PurchaseOrderID,
		PurchaseOrderNumber: row.PurchaseOrderNumber,
		PurchaseOrderStatus: row.PurchaseOrderStatus,
		SupplierID:          supplierID,
		SupplierName:        supplierName,
		SupplierNumber:      supplierNumber,
		LineCount:           safeconv.Int64ToInt32(row.LineCount),
		CompletedAt:         completedAt,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}
}

func mapReceivingOrderLineRow(row sqlc.ListReceivingOrderLinesByOrderIDRow) *domain.ReceivingOrderLine {
	line := &domain.ReceivingOrderLine{
		ID:                        row.ID,
		QuantityID:                row.QuantityID,
		QuantityValue:             row.QuantityValue,
		QuantityUnitID:            row.QuantityUnitID,
		QuantityUnitAbbreviation:  row.QuantityUnitAbbreviation,
		OrderLineID:               row.OrderLineID,
		OrderLineQuantityID:       row.OrderLineQuantityID,
		OrderLineQuantityOrdered:  row.OrderLineQuantityOrdered,
		OrderLineUnitID:           row.OrderLineUnitID,
		OrderLineUnitAbbreviation: row.OrderLineUnitAbbreviation,
		CreatedAt:                 row.CreatedAt,
		UpdatedAt:                 row.UpdatedAt,
	}

	if row.StockedAt.Valid {
		line.StockedAt = &row.StockedAt.Time
	}
	if row.OrderLineProductID.Valid {
		line.OrderLineProductID = &row.OrderLineProductID.String
	}
	if row.OrderLineItemNumber.Valid {
		line.OrderLineItemNumber = &row.OrderLineItemNumber.Int32
	}
	if row.OrderLineItemID.Valid {
		line.OrderLineItemID = &row.OrderLineItemID.String
	}
	if row.OrderLineItemSku.Valid {
		line.OrderLineItemSKU = &row.OrderLineItemSku.String
	}
	if row.OrderLineItemDescription.Valid {
		line.OrderLineItemDescription = &row.OrderLineItemDescription.String
	}
	if row.RejectedQuantityValue != nil {
		var val string
		switch v := row.RejectedQuantityValue.(type) {
		case []byte:
			val = string(v)
		case string:
			val = v
		}
		if val != "" {
			line.RejectedQuantityValue = &val
		}
	}

	return line
}

func mapGetReceivingOrderLineRow(row sqlc.GetReceivingOrderLineRow) *domain.ReceivingOrderLine {
	line := &domain.ReceivingOrderLine{
		ID:                        row.ID,
		QuantityID:                row.QuantityID,
		QuantityValue:             row.QuantityValue,
		QuantityUnitID:            row.QuantityUnitID,
		QuantityUnitAbbreviation:  row.QuantityUnitAbbreviation,
		OrderLineID:               row.OrderLineID,
		OrderLineQuantityID:       row.OrderLineQuantityID,
		OrderLineQuantityOrdered:  row.OrderLineQuantityOrdered,
		OrderLineUnitID:           row.OrderLineUnitID,
		OrderLineUnitAbbreviation: row.OrderLineUnitAbbreviation,
		CreatedAt:                 row.CreatedAt,
		UpdatedAt:                 row.UpdatedAt,
	}

	if row.StockedAt.Valid {
		line.StockedAt = &row.StockedAt.Time
	}
	if row.OrderLineProductID.Valid {
		line.OrderLineProductID = &row.OrderLineProductID.String
	}
	if row.OrderLineItemNumber.Valid {
		line.OrderLineItemNumber = &row.OrderLineItemNumber.Int32
	}
	if row.OrderLineItemID.Valid {
		line.OrderLineItemID = &row.OrderLineItemID.String
	}
	if row.OrderLineItemSku.Valid {
		line.OrderLineItemSKU = &row.OrderLineItemSku.String
	}
	if row.OrderLineItemDescription.Valid {
		line.OrderLineItemDescription = &row.OrderLineItemDescription.String
	}
	if row.RejectedQuantityValue != nil {
		var val string
		switch v := row.RejectedQuantityValue.(type) {
		case []byte:
			val = string(v)
		case string:
			val = v
		}
		if val != "" {
			line.RejectedQuantityValue = &val
		}
	}

	return line
}

// fetchDeliveryRefs names the deliveries booked against each of the given purchase orders, keyed by order id.
//
// Keyed on the purchase order because that is the column deliveries carry; a receiving order is one-to-one with the order it was created for.
func (r *receivingOrderRepoImpl) fetchDeliveryRefs(ctx context.Context, orderIDs []string) (map[string][]domain.DocumentRef, *apierror.APIError) {
	if len(orderIDs) == 0 {
		return map[string][]domain.DocumentRef{}, nil
	}

	rows, err := r.queries.ListDeliveryRefsForOrders(ctx, orderIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}

	refs := make(map[string][]domain.DocumentRef, len(orderIDs))
	for _, row := range rows {
		refs[row.OrderID] = append(refs[row.OrderID], domain.DocumentRef{ID: row.ID, Number: row.Number, Status: row.Status})
	}
	return refs, nil
}

// attachTotals fills in each summary's totals from a single aggregate over the whole page, so a list of orders costs one extra query rather than one per row.
func (r *receivingOrderRepoImpl) attachTotals(ctx context.Context, orders []*domain.ReceivingOrderSummary) *apierror.APIError {
	if len(orders) == 0 {
		return nil
	}

	ids := make([]string, len(orders))
	for i, o := range orders {
		ids[i] = o.ID
	}

	totals, apiErr := r.fetchReceivingOrderTotals(ctx, ids)
	if apiErr != nil {
		return apiErr
	}

	orderIDs := make([]string, len(orders))
	for i, o := range orders {
		orderIDs[i] = o.PurchaseOrderID
	}
	deliveries, apiErr := r.fetchDeliveryRefs(ctx, orderIDs)
	if apiErr != nil {
		return apiErr
	}

	for _, o := range orders {
		o.Totals = totals[o.ID]
		o.Deliveries = deliveries[o.PurchaseOrderID]
	}
	return nil
}
