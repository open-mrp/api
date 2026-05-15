package repository

import (
	"context"
	gosql "database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/shopspring/decimal"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/safeconv"
	"github.com/augno/api/shared/tracing"
)

var salesOrderRepoTracer = tracing.GetTracer("core-service.sales_order_repository")

type salesOrderRepoImpl struct {
	queries *sqlc.Queries
}

func NewSalesOrderRepo(queries *sqlc.Queries) domain.SalesOrderRepo {
	return &salesOrderRepoImpl{queries: queries}
}

func salesOrderCreatedAt(d *domain.SalesOrderSummary) time.Time { return d.CreatedAt }
func salesOrderID(d *domain.SalesOrderSummary) string           { return d.ID }

func (r *salesOrderRepoImpl) NoteFirstShipAt(ctx context.Context, accountID, salesOrderID string) *apierror.APIError {
	ctx, span := salesOrderRepoTracer.Start(ctx, "repository.sales_order.note_first_ship_at")
	defer span.End()

	err := r.queries.NoteSalesOrderFirstShipAt(ctx, sqlc.NoteSalesOrderFirstShipAtParams{
		ID:        salesOrderID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func buildSalesOrderSearchParams(query *string) gosql.NullString {
	if query == nil || *query == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: "%" + db.EscapeLike(*query) + "%", Valid: true}
}

func parseDateString(s *string) gosql.NullTime {
	if s == nil || *s == "" {
		return gosql.NullTime{}
	}
	t, err := time.Parse("2006-01-02", *s)
	if err != nil {
		return gosql.NullTime{}
	}
	return gosql.NullTime{Time: t, Valid: true}
}

func buildSalesOrderListFilters(params domain.ListSalesOrdersParams) (
	includeStatusFilter bool, statusCodes []string,
	includeItemFilter bool, itemIDs []gosql.NullString,
	includeProductLineFilter bool, productLineIDs []gosql.NullString,
	includeCustomerFilter bool, customerIDs []string,
	includeCustomerGroupFilter bool, customerGroupIDs []gosql.NullString,
	includeSalesRepFilter bool, salesRepIDs []gosql.NullString,
) {
	// 'all' is a wildcard meaning "no status filter" (matching Dashboard behavior)
	statusCodes = make([]string, 0, len(params.StatusCodes))
	for _, code := range params.StatusCodes {
		if code == "all" {
			statusCodes = statusCodes[:0]
			break
		}
		statusCodes = append(statusCodes, code)
	}
	includeStatusFilter = len(statusCodes) > 0
	if len(statusCodes) == 0 {
		statusCodes = []string{""}
	}

	includeItemFilter = len(params.ItemIDs) > 0
	itemIDs = toNullStringSlice(params.ItemIDs)

	includeProductLineFilter = len(params.ProductLineIDs) > 0
	productLineIDs = toNullStringSlice(params.ProductLineIDs)

	includeCustomerFilter = len(params.CustomerIDs) > 0
	customerIDs = params.CustomerIDs
	if len(customerIDs) == 0 {
		customerIDs = []string{""}
	}

	includeCustomerGroupFilter = len(params.CustomerGroupIDs) > 0
	customerGroupIDs = toNullStringSlice(params.CustomerGroupIDs)

	includeSalesRepFilter = len(params.SalesRepIDs) > 0
	salesRepIDs = toNullStringSlice(params.SalesRepIDs)

	return
}

func (r *salesOrderRepoImpl) List(ctx context.Context, params domain.ListSalesOrdersParams) (*domain.ListSalesOrdersResult, *apierror.APIError) {
	ctx, span := salesOrderRepoTracer.Start(ctx, "repository.sales_order.list")
	defer span.End()

	searchQuery := buildSalesOrderSearchParams(params.Query)
	startDate := parseDateString(params.StartDate)
	endDate := parseDateString(params.EndDate)

	buyerAccountID := gosql.NullString{}
	if params.BuyerAccountID != nil {
		buyerAccountID = gosql.NullString{String: *params.BuyerAccountID, Valid: true}
	}

	includeStatusFilter, statusCodes,
		includeItemFilter, itemIDs,
		includeProductLineFilter, productLineIDs,
		includeCustomerFilter, customerIDs,
		includeCustomerGroupFilter, customerGroupIDs,
		includeSalesRepFilter, salesRepIDs := buildSalesOrderListFilters(params)

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListSalesOrdersBackward(ctx, sqlc.ListSalesOrdersBackwardParams{
				AccountID:                  params.AccountID,
				BuyerAccountID:             buyerAccountID,
				SearchQuery:                searchQuery,
				IncludeStatusFilter:        includeStatusFilter,
				StatusCodes:                statusCodes,
				IncludeItemFilter:          includeItemFilter,
				ItemIds:                    itemIDs,
				IncludeProductLineFilter:   includeProductLineFilter,
				ProductLineIds:             productLineIDs,
				IncludeCustomerFilter:      includeCustomerFilter,
				CustomerIds:                customerIDs,
				IncludeCustomerGroupFilter: includeCustomerGroupFilter,
				CustomerGroupIds:           customerGroupIDs,
				IncludeSalesRepFilter:      includeSalesRepFilter,
				SalesRepIds:                salesRepIDs,
				StartDate:                  startDate,
				EndDate:                    endDate,
				ExcludeInternalOrders:      params.ExcludeInternalOrders,
				CursorCreatedAt:            cur.OccurredAt,
				CursorID:                   cur.ID,
				Limit:                      params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			orders := make([]*domain.SalesOrderSummary, len(rows))
			for i, row := range rows {
				orders[i] = mapBackwardSalesOrderRow(row)
			}
			result, pageInfo := pagination.BuildPageString(orders, params.Limit, cursorDir, salesOrderCreatedAt, salesOrderID)
			return &domain.ListSalesOrdersResult{SalesOrders: result, PageInfo: pageInfo}, nil
		}

		// Forward with cursor
		rows, err := r.queries.ListSalesOrdersForward(ctx, sqlc.ListSalesOrdersForwardParams{
			AccountID:                  params.AccountID,
			BuyerAccountID:             buyerAccountID,
			SearchQuery:                searchQuery,
			IncludeStatusFilter:        includeStatusFilter,
			StatusCodes:                statusCodes,
			IncludeItemFilter:          includeItemFilter,
			ItemIds:                    itemIDs,
			IncludeProductLineFilter:   includeProductLineFilter,
			ProductLineIds:             productLineIDs,
			IncludeCustomerFilter:      includeCustomerFilter,
			CustomerIds:                customerIDs,
			IncludeCustomerGroupFilter: includeCustomerGroupFilter,
			CustomerGroupIds:           customerGroupIDs,
			IncludeSalesRepFilter:      includeSalesRepFilter,
			SalesRepIds:                salesRepIDs,
			StartDate:                  startDate,
			EndDate:                    endDate,
			ExcludeInternalOrders:      params.ExcludeInternalOrders,
			CursorCreatedAt:            gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:                   gosql.NullString{String: cur.ID, Valid: true},
			Limit:                      params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		orders := make([]*domain.SalesOrderSummary, len(rows))
		for i, row := range rows {
			orders[i] = mapForwardSalesOrderRow(row)
		}
		result, pageInfo := pagination.BuildPageString(orders, params.Limit, cursorDir, salesOrderCreatedAt, salesOrderID)
		return &domain.ListSalesOrdersResult{SalesOrders: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListSalesOrdersForward(ctx, sqlc.ListSalesOrdersForwardParams{
		AccountID:                  params.AccountID,
		BuyerAccountID:             buyerAccountID,
		SearchQuery:                searchQuery,
		IncludeStatusFilter:        includeStatusFilter,
		StatusCodes:                statusCodes,
		IncludeItemFilter:          includeItemFilter,
		ItemIds:                    itemIDs,
		IncludeProductLineFilter:   includeProductLineFilter,
		ProductLineIds:             productLineIDs,
		IncludeCustomerFilter:      includeCustomerFilter,
		CustomerIds:                customerIDs,
		IncludeCustomerGroupFilter: includeCustomerGroupFilter,
		CustomerGroupIds:           customerGroupIDs,
		IncludeSalesRepFilter:      includeSalesRepFilter,
		SalesRepIds:                salesRepIDs,
		StartDate:                  startDate,
		EndDate:                    endDate,
		ExcludeInternalOrders:      params.ExcludeInternalOrders,
		Limit:                      params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	orders := make([]*domain.SalesOrderSummary, len(rows))
	for i, row := range rows {
		orders[i] = mapForwardSalesOrderRow(row)
	}
	result, pageInfo := pagination.BuildPageString(orders, params.Limit, cursorDir, salesOrderCreatedAt, salesOrderID)
	return &domain.ListSalesOrdersResult{SalesOrders: result, PageInfo: pageInfo}, nil
}

func (r *salesOrderRepoImpl) Get(ctx context.Context, accountID, salesOrderID string) (*domain.SalesOrder, *apierror.APIError) {
	ctx, span := salesOrderRepoTracer.Start(ctx, "repository.sales_order.get")
	defer span.End()

	row, err := r.queries.GetSalesOrder(ctx, sqlc.GetSalesOrderParams{
		SalesOrderID: salesOrderID,
		AccountID:    accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapGetSalesOrderRow(row), nil
}

func (r *salesOrderRepoImpl) GetForCustomer(ctx context.Context, accountID, buyerAccountID, salesOrderID string) (*domain.SalesOrder, *apierror.APIError) {
	ctx, span := salesOrderRepoTracer.Start(ctx, "repository.sales_order.get_for_customer")
	defer span.End()

	row, err := r.queries.GetSalesOrderForCustomer(ctx, sqlc.GetSalesOrderForCustomerParams{
		SalesOrderID:   salesOrderID,
		AccountID:      accountID,
		BuyerAccountID: buyerAccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapGetSalesOrderForCustomerRow(row), nil
}

func (r *salesOrderRepoImpl) GetLines(ctx context.Context, salesOrderID string) ([]*domain.SalesOrderLine, *apierror.APIError) {
	ctx, span := salesOrderRepoTracer.Start(ctx, "repository.sales_order.get_lines")
	defer span.End()

	rows, err := r.queries.GetSalesOrderLines(ctx, salesOrderID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	lines := make([]*domain.SalesOrderLine, len(rows))
	for i, row := range rows {
		lines[i] = mapSalesOrderLinesRow(row)
		lines[i].SalesOrderID = salesOrderID
	}

	return lines, nil
}

func (r *salesOrderRepoImpl) Create(ctx context.Context, soID string, params domain.CreateSalesOrderParams) (*domain.SalesOrder, *apierror.APIError) {
	ctx, span := salesOrderRepoTracer.Start(ctx, "repository.sales_order.create")
	defer span.End()

	err := r.queries.CreateSalesOrder(ctx, sqlc.CreateSalesOrderParams{
		ID:                    soID,
		Number:                params.Number,
		CustomerPoNumber:      toNullString(params.CustomerPONumber),
		Note:                  toNullString(params.Note),
		BillingAddressID:      params.BillingAddressID,
		ShippingAddressID:     params.ShippingAddressID,
		CarrierID:             toNullString(params.CarrierID),
		CarrierOptionID:       toNullString(params.ServiceLevelID),
		CarrierBillingType:    toNullString(params.CarrierBillingType),
		CarrierBillingAccount: toNullString(params.CarrierBillingAccount),
		PriorityCode:          params.PriorityCode,
		SalesRepID:            toNullString(params.SalesRepID),
		ShippingTermID:        toNullString(params.ShippingTermID),
		SalesOrderStatusCode:  params.SalesOrderStatusCode,
		SalesOrderTypeCode:    params.SalesOrderTypeCode,
		PaymentTermID:         toNullString(params.PaymentTermID),
		OrderDiscountID:       toNullString(params.OrderDiscountID),
		BuyerAccountID:        params.BuyerAccountID,
		SellerAccountID:       params.SellerAccountID,
		OwnerAccountID:        params.OwnerAccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.Get(ctx, params.AccountID, soID)
}

func (r *salesOrderRepoImpl) Update(ctx context.Context, params domain.UpdateSalesOrderParams) (*domain.SalesOrder, *apierror.APIError) {
	ctx, span := salesOrderRepoTracer.Start(ctx, "repository.sales_order.update")
	defer span.End()

	promisedAtNull := gosql.NullTime{}
	if params.PromisedAt != nil {
		promisedAtNull = gosql.NullTime{Time: *params.PromisedAt, Valid: true}
	}

	err := r.queries.UpdateSalesOrder(ctx, sqlc.UpdateSalesOrderParams{
		Number:                toNullString(params.Number),
		CustomerPoNumber:      toNullString(params.CustomerPONumber),
		Note:                  toNullString(params.Note),
		CarrierID:             toNullString(params.CarrierID),
		CarrierOptionID:       toNullString(params.ServiceLevelID),
		CarrierBillingType:    toNullString(params.CarrierBillingType),
		CarrierBillingAccount: toNullString(params.CarrierBillingAccount),
		PriorityCode:          toNullString(params.PriorityCode),
		SalesRepID:            toNullString(params.SalesRepID),
		ShippingTermID:        toNullString(params.ShippingTermID),
		PaymentTermID:         toNullString(params.PaymentTermID),
		OrderDiscountID:       toNullString(params.OrderDiscountID),
		IsAcknowledgmentSent:  toNullBool(params.IsAcknowledgmentSent),
		PromisedAt:            promisedAtNull,
		BuyerAccountID:        toNullString(params.BuyerAccountID),
		BillingAddressID:      toNullString(params.BillingAddressID),
		ShippingAddressID:     toNullString(params.ShippingAddressID),
		ID:                    params.SalesOrderID,
		AccountID:             params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.Get(ctx, params.AccountID, params.SalesOrderID)
}

func (r *salesOrderRepoImpl) Delete(ctx context.Context, accountID, salesOrderID string) *apierror.APIError {
	ctx, span := salesOrderRepoTracer.Start(ctx, "repository.sales_order.delete")
	defer span.End()

	err := r.queries.DeleteSalesOrder(ctx, sqlc.DeleteSalesOrderParams{
		ID:        salesOrderID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *salesOrderRepoImpl) UpdateStatus(ctx context.Context, accountID, salesOrderID, statusCode string, issuedAt, completedAt *time.Time) *apierror.APIError {
	ctx, span := salesOrderRepoTracer.Start(ctx, "repository.sales_order.update_status")
	defer span.End()

	issuedAtNull := gosql.NullTime{}
	if issuedAt != nil {
		issuedAtNull = gosql.NullTime{Time: *issuedAt, Valid: true}
	}
	completedAtNull := gosql.NullTime{}
	if completedAt != nil {
		completedAtNull = gosql.NullTime{Time: *completedAt, Valid: true}
	}

	err := r.queries.UpdateSalesOrderStatus(ctx, sqlc.UpdateSalesOrderStatusParams{
		StatusCode:  statusCode,
		IssuedAt:    issuedAtNull,
		CompletedAt: completedAtNull,
		ID:          salesOrderID,
		AccountID:   accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *salesOrderRepoImpl) IsOrderForCustomer(ctx context.Context, salesOrderID, buyerAccountID string) (bool, *apierror.APIError) {
	ctx, span := salesOrderRepoTracer.Start(ctx, "repository.sales_order.is_order_for_customer")
	defer span.End()

	exists, err := r.queries.IsOrderForCustomer(ctx, sqlc.IsOrderForCustomerParams{
		SalesOrderID:   salesOrderID,
		BuyerAccountID: buyerAccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return exists, nil
}

func (r *salesOrderRepoImpl) IsDuplicateOrderNumber(ctx context.Context, accountID, number string, excludeID *string) (bool, *apierror.APIError) {
	ctx, span := salesOrderRepoTracer.Start(ctx, "repository.sales_order.is_duplicate_order_number")
	defer span.End()

	cnt, err := r.queries.IsDuplicateOrderNumber(ctx, sqlc.IsDuplicateOrderNumberParams{
		AccountID: accountID,
		Number:    number,
		ExcludeID: toNullString(excludeID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return cnt > 0, nil
}

func (r *salesOrderRepoImpl) IsDuplicateCustomerPO(ctx context.Context, accountID, buyerAccountID, customerPO string, excludeID *string) (bool, *apierror.APIError) {
	ctx, span := salesOrderRepoTracer.Start(ctx, "repository.sales_order.is_duplicate_customer_po")
	defer span.End()

	cnt, err := r.queries.IsDuplicateCustomerPO(ctx, sqlc.IsDuplicateCustomerPOParams{
		AccountID:        accountID,
		BuyerAccountID:   buyerAccountID,
		CustomerPoNumber: gosql.NullString{String: customerPO, Valid: true},
		ExcludeID:        toNullString(excludeID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return cnt > 0, nil
}

func (r *salesOrderRepoImpl) GetNextOrderNumber(ctx context.Context, accountID string) (string, *apierror.APIError) {
	ctx, span := salesOrderRepoTracer.Start(ctx, "repository.sales_order.get_next_order_number")
	defer span.End()

	nextNumber, err := r.queries.GetNextOrderNumber(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	numberStr := decimalToString(nextNumber)
	var numberInt int32
	_, parseErr := fmt.Sscanf(numberStr, "%d", &numberInt)
	if parseErr != nil {
		return "", tracing.Trace(span, apierror.NewInternalError(parseErr, "Failed to parse next order number."))
	}

	sysPropertyID, apiErr := id.GenID(id.SysPropertyIDPrefix, nil)
	if apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	err = r.queries.UpdateNextOrderNumber(ctx, sqlc.UpdateNextOrderNumberParams{
		ID:        sysPropertyID,
		AccountID: accountID,
		Value:     numberInt,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	return numberStr, nil
}

func (r *salesOrderRepoImpl) GetPickID(ctx context.Context, salesOrderID string) (*string, *apierror.APIError) {
	ctx, span := salesOrderRepoTracer.Start(ctx, "repository.sales_order.get_pick_id")
	defer span.End()

	pickID, err := r.queries.GetSalesOrderPickID(ctx, salesOrderID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			return nil, nil
		}
		return nil, tracing.Trace(span, apiErr)
	}

	return &pickID, nil
}

func (r *salesOrderRepoImpl) DeleteCascade(ctx context.Context, accountID, salesOrderID string) *apierror.APIError {
	ctx, span := salesOrderRepoTracer.Start(ctx, "repository.sales_order.delete_cascade")
	defer span.End()

	// Delete quantities associated with pick lines for this sales order
	if err := r.queries.DeleteQuantitiesByPickLines(ctx, salesOrderID); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	// Delete pick lines for this sales order
	if err := r.queries.DeletePickLinesBySalesOrder(ctx, salesOrderID); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	// Delete pick for this sales order
	if err := r.queries.DeletePickBySalesOrder(ctx, salesOrderID); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	// Delete order email contacts
	if err := r.queries.DeleteOrderEmailContacts(ctx, salesOrderID); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	// Delete order payment intents
	if err := r.queries.DeleteOrderPaymentIntents(ctx, salesOrderID); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	// Delete shipment line quantities before shipment lines
	if err := r.queries.DeleteShipmentLineQuantitiesBySalesOrder(ctx, salesOrderID); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	// Delete shipment lines for this sales order
	if err := r.queries.DeleteShipmentLinesBySalesOrder(ctx, salesOrderID); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	// Delete invoice line quantities before invoice lines
	if err := r.queries.DeleteInvoiceLineQuantitiesBySalesOrder(ctx, salesOrderID); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	// Delete invoice lines for this sales order
	if err := r.queries.DeleteInvoiceLinesBySalesOrder(ctx, salesOrderID); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	// Release reserved inventory issues for this sales order
	if err := r.queries.DeleteReservedInventoryIssuesBySalesOrder(ctx, sqlc.DeleteReservedInventoryIssuesBySalesOrderParams{
		SalesOrderID: gosql.NullString{String: salesOrderID, Valid: true},
		AccountID:    accountID,
	}); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	// Delete sales order lines
	if err := r.queries.DeleteSalesOrderLinesBySalesOrder(ctx, salesOrderID); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	// Delete the sales order itself
	if err := r.queries.DeleteSalesOrder(ctx, sqlc.DeleteSalesOrderParams{
		ID:        salesOrderID,
		AccountID: accountID,
	}); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	return nil
}

func (r *salesOrderRepoImpl) CreatePick(ctx context.Context, pickID, number, salesOrderID, accountID string) *apierror.APIError {
	ctx, span := salesOrderRepoTracer.Start(ctx, "repository.sales_order.create_pick")
	defer span.End()

	err := r.queries.CreatePick(ctx, sqlc.CreatePickParams{
		ID:           pickID,
		Number:       number,
		SalesOrderID: salesOrderID,
		AccountID:    accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *salesOrderRepoImpl) CreatePickLine(ctx context.Context, pickLineID, pickID, quantityID, salesOrderLineID string) *apierror.APIError {
	ctx, span := salesOrderRepoTracer.Start(ctx, "repository.sales_order.create_pick_line")
	defer span.End()

	err := r.queries.CreatePickLineForOrderLine(ctx, sqlc.CreatePickLineForOrderLineParams{
		ID:               pickLineID,
		PickID:           pickID,
		QuantityID:       quantityID,
		SalesOrderLineID: salesOrderLineID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *salesOrderRepoImpl) DeleteQuantitiesByPickLines(ctx context.Context, salesOrderID string) *apierror.APIError {
	ctx, span := salesOrderRepoTracer.Start(ctx, "repository.sales_order.delete_quantities_by_pick_lines")
	defer span.End()

	if err := r.queries.DeleteQuantitiesByPickLines(ctx, salesOrderID); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	return nil
}

func (r *salesOrderRepoImpl) DeletePickLinesBySalesOrder(ctx context.Context, salesOrderID string) *apierror.APIError {
	ctx, span := salesOrderRepoTracer.Start(ctx, "repository.sales_order.delete_pick_lines_by_sales_order")
	defer span.End()

	if err := r.queries.DeletePickLinesBySalesOrder(ctx, salesOrderID); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	return nil
}

func (r *salesOrderRepoImpl) DeletePickBySalesOrder(ctx context.Context, salesOrderID string) *apierror.APIError {
	ctx, span := salesOrderRepoTracer.Start(ctx, "repository.sales_order.delete_pick_by_sales_order")
	defer span.End()

	if err := r.queries.DeletePickBySalesOrder(ctx, salesOrderID); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	return nil
}

func (r *salesOrderRepoImpl) GetSaleLinesForIssue(ctx context.Context, salesOrderID string) ([]domain.SalesOrderSaleLineForIssue, *apierror.APIError) {
	ctx, span := salesOrderRepoTracer.Start(ctx, "repository.sales_order.get_sale_lines_for_issue")
	defer span.End()

	rows, err := r.queries.GetSalesOrderSaleLinesForIssue(ctx, salesOrderID)
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}

	lines := make([]domain.SalesOrderSaleLineForIssue, len(rows))
	for i, row := range rows {
		lines[i] = domain.SalesOrderSaleLineForIssue{
			ID:             row.ID,
			QuantityValue:  row.QuantityValue,
			QuantityUnitID: row.QuantityUnitID,
		}
		if row.ItemID.Valid {
			lines[i].ItemID = &row.ItemID.String
		}
	}

	return lines, nil
}

func (r *salesOrderRepoImpl) CreateReservedInventoryIssue(ctx context.Context, issueID, accountID, itemID, quantityID, orderID string) *apierror.APIError {
	ctx, span := salesOrderRepoTracer.Start(ctx, "repository.sales_order.create_reserved_inventory_issue")
	defer span.End()

	err := r.queries.CreateReservedInventoryIssueForSalesOrder(ctx, sqlc.CreateReservedInventoryIssueForSalesOrderParams{
		ID:         issueID,
		AccountID:  accountID,
		ItemID:     itemID,
		QuantityID: quantityID,
		OrderID:    gosql.NullString{String: orderID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *salesOrderRepoImpl) DeleteInventoryAllocationsByReservedIssues(ctx context.Context, accountID, salesOrderID string) *apierror.APIError {
	ctx, span := salesOrderRepoTracer.Start(ctx, "repository.sales_order.delete_inventory_allocations_by_reserved_issues")
	defer span.End()

	err := r.queries.DeleteInventoryAllocationsByReservedSalesOrderIssues(ctx, sqlc.DeleteInventoryAllocationsByReservedSalesOrderIssuesParams{
		SalesOrderID: gosql.NullString{String: salesOrderID, Valid: true},
		AccountID:    accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *salesOrderRepoImpl) DeleteReservedInventoryIssues(ctx context.Context, accountID, salesOrderID string) *apierror.APIError {
	ctx, span := salesOrderRepoTracer.Start(ctx, "repository.sales_order.delete_reserved_inventory_issues")
	defer span.End()

	err := r.queries.DeleteReservedInventoryIssuesBySalesOrder(ctx, sqlc.DeleteReservedInventoryIssuesBySalesOrderParams{
		SalesOrderID: gosql.NullString{String: salesOrderID, Valid: true},
		AccountID:    accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

// Mapping helpers

func mapGetSalesOrderRow(row sqlc.GetSalesOrderRow) *domain.SalesOrder {
	so := &domain.SalesOrder{
		ID:                   row.ID,
		Number:               row.Number,
		IsAcknowledgmentSent: row.IsAcknowledgmentSent,
		BillingAddressID:     row.BillingAddressID,
		ShippingAddressID:    row.ShippingAddressID,
		PriorityCode:         constants.PriorityCode(row.PriorityCode),
		SalesOrderStatusCode: row.SalesOrderStatusCode,
		SalesOrderTypeCode:   row.SalesOrderTypeCode,
		BuyerAccountID:       row.BuyerAccountID,
		SellerAccountID:      row.SellerAccountID,
		OwnerAccountID:       row.OwnerAccountID,
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
		CustomerName:         row.CustomerName,
		CustomerNumber:       row.CustomerNumber,
		StatusName:           row.StatusName,
		TypeName:             row.TypeName,
		PriorityName:         row.PriorityName,
	}

	so.CustomerPONumber = nullStringToPtr(row.CustomerPoNumber)
	so.Note = nullStringToPtr(row.Note)
	so.CarrierID = nullStringToPtr(row.CarrierID)
	so.ServiceLevelID = nullStringToPtr(row.CarrierOptionID)
	so.CarrierBillingType = nullStringToPtr(row.CarrierBillingType)
	so.CarrierBillingAccount = nullStringToPtr(row.CarrierBillingAccount)
	so.SalesRepID = nullStringToPtr(row.SalesRepID)
	so.ShippingTermID = nullStringToPtr(row.ShippingTermID)
	so.PaymentTermID = nullStringToPtr(row.PaymentTermID)
	so.ProductionRunID = nullStringToPtr(row.ProductionRunID)
	so.OrderDiscountID = nullStringToPtr(row.OrderDiscountID)

	if row.IssuedAt.Valid {
		so.IssuedAt = &row.IssuedAt.Time
	}
	if row.CompletedAt.Valid {
		so.CompletedAt = &row.CompletedAt.Time
	}
	if row.FirstShipAt.Valid {
		so.FirstShipAt = &row.FirstShipAt.Time
	}
	if row.ExpiredAt.Valid {
		so.ExpiredAt = &row.ExpiredAt.Time
	}
	if row.PromisedAt.Valid {
		so.PromisedAt = &row.PromisedAt.Time
	}

	so.BillToName = nullStringToPtr(row.BillToName)
	so.BillToIsDropShip = nullBoolPtr(row.BillToIsDropShip)
	so.BillToGeolocationID = nullStringToPtr(row.BillToGeolocationID)
	so.BillToStreetLine1 = nullStringToPtr(row.BillToStreetLine1)
	so.BillToStreetLine2 = nullStringToPtr(row.BillToStreetLine2)
	so.BillToLocality = nullStringToPtr(row.BillToLocality)
	so.BillToState = nullStringToPtr(row.BillToState)
	so.BillToPostalCode = nullStringToPtr(row.BillToPostalCode)
	so.BillToCountry = nullStringToPtr(row.BillToCountry)
	so.BillToPhone = nullStringToPtr(row.BillToPhone)
	so.BillToEmail = nullStringToPtr(row.BillToEmail)
	so.BillToCreatedAt = nullTimePtr(row.BillToCreatedAt)
	so.BillToUpdatedAt = nullTimePtr(row.BillToUpdatedAt)
	so.ShipToName = nullStringToPtr(row.ShipToName)
	so.ShipToIsDropShip = nullBoolPtr(row.ShipToIsDropShip)
	so.ShipToGeolocationID = nullStringToPtr(row.ShipToGeolocationID)
	so.ShipToStreetLine1 = nullStringToPtr(row.ShipToStreetLine1)
	so.ShipToStreetLine2 = nullStringToPtr(row.ShipToStreetLine2)
	so.ShipToLocality = nullStringToPtr(row.ShipToLocality)
	so.ShipToState = nullStringToPtr(row.ShipToState)
	so.ShipToPostalCode = nullStringToPtr(row.ShipToPostalCode)
	so.ShipToCountry = nullStringToPtr(row.ShipToCountry)
	so.ShipToPhone = nullStringToPtr(row.ShipToPhone)
	so.ShipToEmail = nullStringToPtr(row.ShipToEmail)
	so.ShipToCreatedAt = nullTimePtr(row.ShipToCreatedAt)
	so.ShipToUpdatedAt = nullTimePtr(row.ShipToUpdatedAt)

	so.CustomerCreatedAt = &row.CustomerCreatedAt
	so.CustomerUpdatedAt = &row.CustomerUpdatedAt

	so.CarrierName = nullStringToPtr(row.CarrierName)
	so.CarrierIsPortalEnabled = nullBoolPtr(row.CarrierIsPortalEnabled)
	so.CarrierCreatedAt = nullTimePtr(row.CarrierCreatedAt)
	so.CarrierUpdatedAt = nullTimePtr(row.CarrierUpdatedAt)
	so.ServiceLevelName = nullStringToPtr(row.CarrierOptionName)
	so.ServiceLevelIsPortalEnabled = nullBoolPtr(row.ServiceLevelIsPortalEnabled)
	so.ServiceLevelToken = nullStringToPtr(row.ServiceLevelToken)
	so.ServiceLevelCreatedAt = nullTimePtr(row.ServiceLevelCreatedAt)
	so.ServiceLevelUpdatedAt = nullTimePtr(row.ServiceLevelUpdatedAt)
	so.SalesRepName = nullStringToPtr(row.SalesRepName)
	so.PaymentTermName = nullStringToPtr(row.PaymentTermName)
	so.PaymentTermIsActive = nullBoolPtr(row.PaymentTermIsActive)
	so.PaymentTermCreatedAt = nullTimePtr(row.PaymentTermCreatedAt)
	so.PaymentTermUpdatedAt = nullTimePtr(row.PaymentTermUpdatedAt)
	so.ShippingTermName = nullStringToPtr(row.ShippingTermName)
	so.ShippingTermIsFreightExempt = nullBoolPtr(row.ShippingTermIsFreightExempt)
	so.ShippingTermIsCarrierRate = nullBoolPtr(row.ShippingTermIsCarrierRate)
	so.ShippingTermCreatedAt = nullTimePtr(row.ShippingTermCreatedAt)
	so.ShippingTermUpdatedAt = nullTimePtr(row.ShippingTermUpdatedAt)
	so.OrderDiscountName = nullStringToPtr(row.OrderDiscountName)
	so.OrderDiscountCode = nullStringToPtr(row.OrderDiscountCode)
	if row.OrderDiscountPercentage.Valid {
		s := strconv.FormatFloat(row.OrderDiscountPercentage.Float64, 'f', -1, 64)
		so.OrderDiscountPercentage = &s
	}
	if row.OrderDiscountAmount.Valid {
		s := strconv.FormatFloat(row.OrderDiscountAmount.Float64, 'f', -1, 64)
		so.OrderDiscountAmount = &s
	}
	so.OrderDiscountDiscountType = nullStringToPtr(row.OrderDiscountDiscountType)
	if row.OrderDiscountID.Valid {
		cnt := safeconv.Int64ToInt32(row.OrderDiscountOrderCount)
		so.OrderDiscountOrderCount = &cnt
	}
	so.OrderDiscountCreatedAt = nullTimePtr(row.OrderDiscountCreatedAt)
	so.OrderDiscountUpdatedAt = nullTimePtr(row.OrderDiscountUpdatedAt)
	so.PickID = nullStringToPtr(row.PickID)
	so.PriorityID = &row.PriorityID
	so.CustomerStatusCode = nullStringToPtr(row.CustomerStatusCode)
	so.CustomerCommissionPolicy = nullStringToPtr(row.CustomerCommissionPolicy)

	return so
}

func mapGetSalesOrderForCustomerRow(row sqlc.GetSalesOrderForCustomerRow) *domain.SalesOrder {
	so := &domain.SalesOrder{
		ID:                   row.ID,
		Number:               row.Number,
		IsAcknowledgmentSent: row.IsAcknowledgmentSent,
		BillingAddressID:     row.BillingAddressID,
		ShippingAddressID:    row.ShippingAddressID,
		PriorityCode:         constants.PriorityCode(row.PriorityCode),
		SalesOrderStatusCode: row.SalesOrderStatusCode,
		SalesOrderTypeCode:   row.SalesOrderTypeCode,
		BuyerAccountID:       row.BuyerAccountID,
		SellerAccountID:      row.SellerAccountID,
		OwnerAccountID:       row.OwnerAccountID,
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
		CustomerName:         row.CustomerName,
		CustomerNumber:       row.CustomerNumber,
		StatusName:           row.StatusName,
		TypeName:             row.TypeName,
		PriorityName:         row.PriorityName,
	}

	so.CustomerPONumber = nullStringToPtr(row.CustomerPoNumber)
	so.Note = nullStringToPtr(row.Note)
	so.CarrierID = nullStringToPtr(row.CarrierID)
	so.ServiceLevelID = nullStringToPtr(row.CarrierOptionID)
	so.CarrierBillingType = nullStringToPtr(row.CarrierBillingType)
	so.CarrierBillingAccount = nullStringToPtr(row.CarrierBillingAccount)
	so.SalesRepID = nullStringToPtr(row.SalesRepID)
	so.ShippingTermID = nullStringToPtr(row.ShippingTermID)
	so.PaymentTermID = nullStringToPtr(row.PaymentTermID)
	so.ProductionRunID = nullStringToPtr(row.ProductionRunID)
	so.OrderDiscountID = nullStringToPtr(row.OrderDiscountID)

	if row.IssuedAt.Valid {
		so.IssuedAt = &row.IssuedAt.Time
	}
	if row.CompletedAt.Valid {
		so.CompletedAt = &row.CompletedAt.Time
	}
	if row.FirstShipAt.Valid {
		so.FirstShipAt = &row.FirstShipAt.Time
	}
	if row.ExpiredAt.Valid {
		so.ExpiredAt = &row.ExpiredAt.Time
	}
	if row.PromisedAt.Valid {
		so.PromisedAt = &row.PromisedAt.Time
	}

	so.BillToName = nullStringToPtr(row.BillToName)
	so.BillToIsDropShip = nullBoolPtr(row.BillToIsDropShip)
	so.BillToGeolocationID = nullStringToPtr(row.BillToGeolocationID)
	so.BillToStreetLine1 = nullStringToPtr(row.BillToStreetLine1)
	so.BillToStreetLine2 = nullStringToPtr(row.BillToStreetLine2)
	so.BillToLocality = nullStringToPtr(row.BillToLocality)
	so.BillToState = nullStringToPtr(row.BillToState)
	so.BillToPostalCode = nullStringToPtr(row.BillToPostalCode)
	so.BillToCountry = nullStringToPtr(row.BillToCountry)
	so.BillToPhone = nullStringToPtr(row.BillToPhone)
	so.BillToEmail = nullStringToPtr(row.BillToEmail)
	so.BillToCreatedAt = nullTimePtr(row.BillToCreatedAt)
	so.BillToUpdatedAt = nullTimePtr(row.BillToUpdatedAt)
	so.ShipToName = nullStringToPtr(row.ShipToName)
	so.ShipToIsDropShip = nullBoolPtr(row.ShipToIsDropShip)
	so.ShipToGeolocationID = nullStringToPtr(row.ShipToGeolocationID)
	so.ShipToStreetLine1 = nullStringToPtr(row.ShipToStreetLine1)
	so.ShipToStreetLine2 = nullStringToPtr(row.ShipToStreetLine2)
	so.ShipToLocality = nullStringToPtr(row.ShipToLocality)
	so.ShipToState = nullStringToPtr(row.ShipToState)
	so.ShipToPostalCode = nullStringToPtr(row.ShipToPostalCode)
	so.ShipToCountry = nullStringToPtr(row.ShipToCountry)
	so.ShipToPhone = nullStringToPtr(row.ShipToPhone)
	so.ShipToEmail = nullStringToPtr(row.ShipToEmail)
	so.ShipToCreatedAt = nullTimePtr(row.ShipToCreatedAt)
	so.ShipToUpdatedAt = nullTimePtr(row.ShipToUpdatedAt)

	so.CustomerCreatedAt = &row.CustomerCreatedAt
	so.CustomerUpdatedAt = &row.CustomerUpdatedAt

	so.CarrierName = nullStringToPtr(row.CarrierName)
	so.CarrierIsPortalEnabled = nullBoolPtr(row.CarrierIsPortalEnabled)
	so.CarrierCreatedAt = nullTimePtr(row.CarrierCreatedAt)
	so.CarrierUpdatedAt = nullTimePtr(row.CarrierUpdatedAt)
	so.ServiceLevelName = nullStringToPtr(row.CarrierOptionName)
	so.ServiceLevelIsPortalEnabled = nullBoolPtr(row.ServiceLevelIsPortalEnabled)
	so.ServiceLevelToken = nullStringToPtr(row.ServiceLevelToken)
	so.ServiceLevelCreatedAt = nullTimePtr(row.ServiceLevelCreatedAt)
	so.ServiceLevelUpdatedAt = nullTimePtr(row.ServiceLevelUpdatedAt)
	so.SalesRepName = nullStringToPtr(row.SalesRepName)
	so.PaymentTermName = nullStringToPtr(row.PaymentTermName)
	so.PaymentTermIsActive = nullBoolPtr(row.PaymentTermIsActive)
	so.PaymentTermCreatedAt = nullTimePtr(row.PaymentTermCreatedAt)
	so.PaymentTermUpdatedAt = nullTimePtr(row.PaymentTermUpdatedAt)
	so.ShippingTermName = nullStringToPtr(row.ShippingTermName)
	so.ShippingTermIsFreightExempt = nullBoolPtr(row.ShippingTermIsFreightExempt)
	so.ShippingTermIsCarrierRate = nullBoolPtr(row.ShippingTermIsCarrierRate)
	so.ShippingTermCreatedAt = nullTimePtr(row.ShippingTermCreatedAt)
	so.ShippingTermUpdatedAt = nullTimePtr(row.ShippingTermUpdatedAt)
	so.OrderDiscountName = nullStringToPtr(row.OrderDiscountName)
	so.OrderDiscountCode = nullStringToPtr(row.OrderDiscountCode)
	if row.OrderDiscountPercentage.Valid {
		s := strconv.FormatFloat(row.OrderDiscountPercentage.Float64, 'f', -1, 64)
		so.OrderDiscountPercentage = &s
	}
	if row.OrderDiscountAmount.Valid {
		s := strconv.FormatFloat(row.OrderDiscountAmount.Float64, 'f', -1, 64)
		so.OrderDiscountAmount = &s
	}
	so.OrderDiscountDiscountType = nullStringToPtr(row.OrderDiscountDiscountType)
	if row.OrderDiscountID.Valid {
		cnt := safeconv.Int64ToInt32(row.OrderDiscountOrderCount)
		so.OrderDiscountOrderCount = &cnt
	}
	so.OrderDiscountCreatedAt = nullTimePtr(row.OrderDiscountCreatedAt)
	so.OrderDiscountUpdatedAt = nullTimePtr(row.OrderDiscountUpdatedAt)
	so.PickID = nullStringToPtr(row.PickID)
	so.PriorityID = &row.PriorityID
	so.CustomerStatusCode = nullStringToPtr(row.CustomerStatusCode)
	so.CustomerCommissionPolicy = nullStringToPtr(row.CustomerCommissionPolicy)

	return so
}

func mapForwardSalesOrderRow(row sqlc.ListSalesOrdersForwardRow) *domain.SalesOrderSummary {
	s := &domain.SalesOrderSummary{
		ID:                   row.ID,
		Number:               row.Number,
		StatusCode:           row.StatusCode,
		StatusName:           row.StatusName,
		TypeCode:             row.TypeCode,
		TypeName:             row.TypeName,
		CustomerID:           row.CustomerID,
		CustomerName:         row.CustomerName,
		CustomerNumber:       row.CustomerNumber,
		LineCount:            safeconv.Int64ToInt32(row.LineCount),
		IsAcknowledgmentSent: row.IsAcknowledgmentSent,
		PriorityCode:         constants.PriorityCode(row.PriorityCode),
		PriorityName:         row.PriorityName,
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}
	s.CustomerPONumber = nullStringToPtr(row.CustomerPoNumber)
	s.PriorityID = &row.PriorityID
	s.CustomerStatusCode = nullStringToPtr(row.CustomerStatusCode)
	s.CustomerCommissionPolicy = nullStringToPtr(row.CustomerCommissionPolicy)
	if row.IssuedAt.Valid {
		s.IssuedAt = &row.IssuedAt.Time
	}
	if row.CompletedAt.Valid {
		s.CompletedAt = &row.CompletedAt.Time
	}
	return s
}

func mapBackwardSalesOrderRow(row sqlc.ListSalesOrdersBackwardRow) *domain.SalesOrderSummary {
	s := &domain.SalesOrderSummary{
		ID:                   row.ID,
		Number:               row.Number,
		StatusCode:           row.StatusCode,
		StatusName:           row.StatusName,
		TypeCode:             row.TypeCode,
		TypeName:             row.TypeName,
		CustomerID:           row.CustomerID,
		CustomerName:         row.CustomerName,
		CustomerNumber:       row.CustomerNumber,
		LineCount:            safeconv.Int64ToInt32(row.LineCount),
		IsAcknowledgmentSent: row.IsAcknowledgmentSent,
		PriorityCode:         constants.PriorityCode(row.PriorityCode),
		PriorityName:         row.PriorityName,
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}
	s.CustomerPONumber = nullStringToPtr(row.CustomerPoNumber)
	s.PriorityID = &row.PriorityID
	s.CustomerStatusCode = nullStringToPtr(row.CustomerStatusCode)
	s.CustomerCommissionPolicy = nullStringToPtr(row.CustomerCommissionPolicy)
	if row.IssuedAt.Valid {
		s.IssuedAt = &row.IssuedAt.Time
	}
	if row.CompletedAt.Valid {
		s.CompletedAt = &row.CompletedAt.Time
	}
	return s
}

func (r *salesOrderRepoImpl) CheckPaymentStatus(ctx context.Context, salesOrderID string) (bool, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, salesOrderRepoTracer, "repository.sales_order.check_payment_status")
	defer span.End()

	result, err := r.queries.CheckSalesOrderPaymentStatus(ctx, sqlc.CheckSalesOrderPaymentStatusParams{
		SalesOrderID: salesOrderID,
	})
	if err != nil {
		return false, db.MapSQLError(err)
	}
	return result.Valid && result.Bool, nil
}

func (r *salesOrderRepoImpl) GetLinesForBOM(ctx context.Context, salesOrderID string) ([]domain.SalesOrderLineForBOM, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, salesOrderRepoTracer, "repository.sales_order.get_lines_for_bom")
	defer span.End()

	rows, err := r.queries.GetSalesOrderLinesForBOM(ctx, salesOrderID)
	if err != nil {
		return nil, db.MapSQLError(err)
	}

	lines := make([]domain.SalesOrderLineForBOM, 0, len(rows))
	for _, row := range rows {
		if !row.ItemID.Valid {
			continue
		}
		qty, parseErr := decimal.NewFromString(row.QuantityValue)
		if parseErr != nil {
			return nil, apierror.NewInternalError(parseErr, "Invalid quantity value.")
		}
		lines = append(lines, domain.SalesOrderLineForBOM{
			ID:             row.ID,
			ItemID:         row.ItemID.String,
			QuantityValue:  qty,
			QuantityUnitID: row.QuantityUnitID,
		})
	}

	return lines, nil
}

func (r *salesOrderRepoImpl) SetProductionRunID(ctx context.Context, accountID, salesOrderID, productionRunID string) *apierror.APIError {
	ctx, span := tracing.StartSpan(ctx, salesOrderRepoTracer, "repository.sales_order.set_production_run_id")
	defer span.End()

	err := r.queries.SetSalesOrderProductionRunID(ctx, sqlc.SetSalesOrderProductionRunIDParams{
		ProductionRunID: gosql.NullString{String: productionRunID, Valid: true},
		ID:              salesOrderID,
		AccountID:       accountID,
	})
	if err != nil {
		return db.MapSQLError(err)
	}
	return nil
}

func (r *salesOrderRepoImpl) GetAcknowledgementRecipients(ctx context.Context, salesOrderID string) ([]string, *apierror.APIError) {
	ctx, span := salesOrderRepoTracer.Start(ctx, "repository.sales_order.get_acknowledgement_recipients")
	defer span.End()

	rows, err := r.queries.GetOrderAcknowledgementRecipients(ctx, salesOrderID)
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

func (r *salesOrderRepoImpl) MarkAcknowledgementSent(ctx context.Context, accountID, salesOrderID string) *apierror.APIError {
	ctx, span := salesOrderRepoTracer.Start(ctx, "repository.sales_order.mark_acknowledgement_sent")
	defer span.End()

	err := r.queries.MarkAcknowledgementSent(ctx, sqlc.MarkAcknowledgementSentParams{
		ID:        salesOrderID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *salesOrderRepoImpl) CreateEmailContact(ctx context.Context, id, salesOrderID, accountUserID, notificationTypeCode string) *apierror.APIError {
	ctx, span := salesOrderRepoTracer.Start(ctx, "repository.sales_order.create_email_contact")
	defer span.End()

	err := r.queries.CreateSalesOrderEmailContact(ctx, sqlc.CreateSalesOrderEmailContactParams{
		ID:                   id,
		SalesOrderID:         salesOrderID,
		AccountUserID:        accountUserID,
		NotificationTypeCode: notificationTypeCode,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *salesOrderRepoImpl) DeleteEmailContactsByOrderAndType(ctx context.Context, salesOrderID, notificationTypeCode string) *apierror.APIError {
	ctx, span := salesOrderRepoTracer.Start(ctx, "repository.sales_order.delete_email_contacts_by_order_and_type")
	defer span.End()

	err := r.queries.DeleteSalesOrderEmailContactsByOrderAndType(ctx, sqlc.DeleteSalesOrderEmailContactsByOrderAndTypeParams{
		SalesOrderID:         salesOrderID,
		NotificationTypeCode: notificationTypeCode,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *salesOrderRepoImpl) MarkUnfulfilled(ctx context.Context, accountID, salesOrderID string) *apierror.APIError {
	ctx, span := salesOrderRepoTracer.Start(ctx, "repository.sales_order.mark_unfulfilled")
	defer span.End()

	err := r.queries.MarkSalesOrderUnfulfilled(ctx, sqlc.MarkSalesOrderUnfulfilledParams{
		ID:        salesOrderID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *salesOrderRepoImpl) HasShippedShipment(ctx context.Context, salesOrderID string) (bool, *apierror.APIError) {
	ctx, span := salesOrderRepoTracer.Start(ctx, "repository.sales_order.has_shipped_shipment")
	defer span.End()

	result, err := r.queries.HasShippedShipmentForSalesOrder(ctx, salesOrderID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return result, nil
}
