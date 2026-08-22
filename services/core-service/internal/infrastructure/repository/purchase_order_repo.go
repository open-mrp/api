package repository

import (
	"context"
	gosql "database/sql"
	"strconv"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/pagination"
	"github.com/open-mrp/api/shared/safeconv"
	"github.com/open-mrp/api/shared/tracing"
)

var purchaseOrderRepoTracer = tracing.GetTracer("core-service.purchase_order_repository")

type purchaseOrderRepoImpl struct {
	queries *sqlc.Queries
}

func NewPurchaseOrderRepo(queries *sqlc.Queries) domain.PurchaseOrderRepo {
	return &purchaseOrderRepoImpl{queries: queries}
}

func purchaseOrderCreatedAt(d *domain.PurchaseOrderSummary) time.Time { return d.CreatedAt }
func purchaseOrderID(d *domain.PurchaseOrderSummary) string           { return d.ID }

func buildPurchaseOrderSearchParams(query *string) gosql.NullString {
	if query == nil || *query == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: "%" + db.EscapeLike(*query) + "%", Valid: true}
}

func buildPurchaseOrderListFilters(params domain.ListPurchaseOrdersParams) (
	includeStatusFilter bool, statusCodes []string,
	includeItemFilter bool, itemIDs []gosql.NullString,
	includeSupplierFilter bool, supplierIDs []string,
) {
	includeStatusFilter = len(params.StatusCodes) > 0
	statusCodes = params.StatusCodes
	if len(statusCodes) == 0 {
		statusCodes = []string{""}
	}

	includeItemFilter = len(params.ItemIDs) > 0
	itemIDs = toNullStringSlice(params.ItemIDs)

	includeSupplierFilter = len(params.SupplierIDs) > 0
	supplierIDs = params.SupplierIDs
	if len(supplierIDs) == 0 {
		supplierIDs = []string{""}
	}

	return
}

func (r *purchaseOrderRepoImpl) List(ctx context.Context, params domain.ListPurchaseOrdersParams) (*domain.ListPurchaseOrdersResult, *apierror.APIError) {
	ctx, span := purchaseOrderRepoTracer.Start(ctx, "repository.purchase_order.list")
	defer span.End()

	searchQuery := buildPurchaseOrderSearchParams(params.Query)
	startDate := parseDateString(params.StartDate)
	endDate := parseDateString(params.EndDate)

	includeStatusFilter, statusCodes,
		includeItemFilter, itemIDs,
		includeSupplierFilter, supplierIDs := buildPurchaseOrderListFilters(params)

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationErrorWithParam("Invalid pagination cursor.", "cursor")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListPurchaseOrdersBackward(ctx, sqlc.ListPurchaseOrdersBackwardParams{
				AccountID:             params.AccountID,
				SearchQuery:           searchQuery,
				IncludeStatusFilter:   includeStatusFilter,
				StatusCodes:           statusCodes,
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
			orders := make([]*domain.PurchaseOrderSummary, len(rows))
			for i, row := range rows {
				orders[i] = mapBackwardPurchaseOrderRow(row)
			}
			result, pageInfo := pagination.BuildPageString(orders, params.Limit, cursorDir, purchaseOrderCreatedAt, purchaseOrderID)
			return &domain.ListPurchaseOrdersResult{PurchaseOrders: result, PageInfo: pageInfo}, nil
		}

		// Forward with cursor
		rows, err := r.queries.ListPurchaseOrdersForward(ctx, sqlc.ListPurchaseOrdersForwardParams{
			AccountID:             params.AccountID,
			SearchQuery:           searchQuery,
			IncludeStatusFilter:   includeStatusFilter,
			StatusCodes:           statusCodes,
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
		orders := make([]*domain.PurchaseOrderSummary, len(rows))
		for i, row := range rows {
			orders[i] = mapForwardPurchaseOrderRow(row)
		}
		result, pageInfo := pagination.BuildPageString(orders, params.Limit, cursorDir, purchaseOrderCreatedAt, purchaseOrderID)
		return &domain.ListPurchaseOrdersResult{PurchaseOrders: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListPurchaseOrdersForward(ctx, sqlc.ListPurchaseOrdersForwardParams{
		AccountID:             params.AccountID,
		SearchQuery:           searchQuery,
		IncludeStatusFilter:   includeStatusFilter,
		StatusCodes:           statusCodes,
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

	orders := make([]*domain.PurchaseOrderSummary, len(rows))
	for i, row := range rows {
		orders[i] = mapForwardPurchaseOrderRow(row)
	}
	result, pageInfo := pagination.BuildPageString(orders, params.Limit, cursorDir, purchaseOrderCreatedAt, purchaseOrderID)
	return &domain.ListPurchaseOrdersResult{PurchaseOrders: result, PageInfo: pageInfo}, nil
}

func (r *purchaseOrderRepoImpl) Get(ctx context.Context, accountID, purchaseOrderID string) (*domain.PurchaseOrder, *apierror.APIError) {
	ctx, span := purchaseOrderRepoTracer.Start(ctx, "repository.purchase_order.get")
	defer span.End()

	row, err := r.queries.GetPurchaseOrder(ctx, sqlc.GetPurchaseOrderParams{
		SalesOrderID: purchaseOrderID,
		AccountID:    accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapGetPurchaseOrderRow(row), nil
}

func (r *purchaseOrderRepoImpl) GetLines(ctx context.Context, salesOrderID string) ([]*domain.PurchaseOrderLine, *apierror.APIError) {
	ctx, span := purchaseOrderRepoTracer.Start(ctx, "repository.purchase_order.get_lines")
	defer span.End()

	rows, err := r.queries.GetPurchaseOrderLines(ctx, salesOrderID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	lines := make([]*domain.PurchaseOrderLine, len(rows))
	for i, row := range rows {
		lines[i] = mapPurchaseOrderLinesRow(row)
		lines[i].SalesOrderID = salesOrderID
	}

	return lines, nil
}

func (r *purchaseOrderRepoImpl) Create(ctx context.Context, poID string, params domain.CreatePurchaseOrderParams) (*domain.PurchaseOrder, *apierror.APIError) {
	ctx, span := purchaseOrderRepoTracer.Start(ctx, "repository.purchase_order.create")
	defer span.End()

	err := r.queries.CreatePurchaseOrder(ctx, sqlc.CreatePurchaseOrderParams{
		ID:                    poID,
		Number:                params.Number,
		Note:                  toNullString(params.Note),
		BillingAddressID:      params.BillingAddressID,
		ShippingAddressID:     params.ShippingAddressID,
		CarrierID:             toNullString(params.CarrierID),
		CarrierOptionID:       toNullString(params.ServiceLevelID),
		CarrierBillingType:    toNullString(params.CarrierBillingType),
		CarrierBillingAccount: toNullString(params.CarrierBillingAccount),
		PriorityCode:          params.PriorityCode,
		ShippingTermID:        toNullString(params.ShippingTermID),
		SalesOrderStatusCode:  params.SalesOrderStatusCode,
		PaymentTermID:         toNullString(params.PaymentTermID),
		BuyerAccountID:        params.AccountID,
		SellerAccountID:       params.SupplierAccountID,
		OwnerAccountID:        params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.Get(ctx, params.AccountID, poID)
}

func (r *purchaseOrderRepoImpl) Update(ctx context.Context, params domain.UpdatePurchaseOrderParams) (*domain.PurchaseOrder, *apierror.APIError) {
	ctx, span := purchaseOrderRepoTracer.Start(ctx, "repository.purchase_order.update")
	defer span.End()

	promisedAt := gosql.NullTime{}
	if params.PromisedAt != nil {
		t, parseErr := time.Parse("2006-01-02", *params.PromisedAt)
		if parseErr == nil {
			promisedAt = gosql.NullTime{Time: t, Valid: true}
		}
	}

	err := r.queries.UpdatePurchaseOrder(ctx, sqlc.UpdatePurchaseOrderParams{
		Note:              toNullString(params.Note),
		Number:            toNullString(params.Number),
		PriorityCode:      toNullString(params.PriorityCode),
		BillingAddressID:  toNullString(params.BillingAddressID),
		ShippingAddressID: toNullString(params.ShippingAddressID),
		PromisedAt:        promisedAt,
		ID:                params.PurchaseOrderID,
		AccountID:         params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.Get(ctx, params.AccountID, params.PurchaseOrderID)
}

func (r *purchaseOrderRepoImpl) Delete(ctx context.Context, accountID, purchaseOrderID string) *apierror.APIError {
	ctx, span := purchaseOrderRepoTracer.Start(ctx, "repository.purchase_order.delete")
	defer span.End()

	err := r.queries.DeletePurchaseOrder(ctx, sqlc.DeletePurchaseOrderParams{
		ID:        purchaseOrderID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *purchaseOrderRepoImpl) UpdateStatus(ctx context.Context, accountID, purchaseOrderID, statusCode string, issuedAt, completedAt *time.Time) *apierror.APIError {
	ctx, span := purchaseOrderRepoTracer.Start(ctx, "repository.purchase_order.update_status")
	defer span.End()

	issuedAtNull := gosql.NullTime{}
	if issuedAt != nil {
		issuedAtNull = gosql.NullTime{Time: *issuedAt, Valid: true}
	}
	completedAtNull := gosql.NullTime{}
	if completedAt != nil {
		completedAtNull = gosql.NullTime{Time: *completedAt, Valid: true}
	}

	err := r.queries.UpdatePurchaseOrderStatus(ctx, sqlc.UpdatePurchaseOrderStatusParams{
		StatusCode:  statusCode,
		IssuedAt:    issuedAtNull,
		CompletedAt: completedAtNull,
		ID:          purchaseOrderID,
		AccountID:   accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *purchaseOrderRepoImpl) IsDuplicateOrderNumber(ctx context.Context, accountID, number string, excludeID *string) (bool, *apierror.APIError) {
	ctx, span := purchaseOrderRepoTracer.Start(ctx, "repository.purchase_order.is_duplicate_order_number")
	defer span.End()

	cnt, err := r.queries.IsDuplicatePurchaseOrderNumber(ctx, sqlc.IsDuplicatePurchaseOrderNumberParams{
		AccountID: accountID,
		Number:    number,
		ExcludeID: toNullString(excludeID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return cnt > 0, nil
}

func (r *purchaseOrderRepoImpl) GetNextOrderNumber(ctx context.Context, accountID string) (string, *apierror.APIError) {
	ctx, span := purchaseOrderRepoTracer.Start(ctx, "repository.purchase_order.get_next_order_number")
	defer span.End()

	sysPropertyID, apiErr := id.GenID(id.SysPropertyIDPrefix, nil)
	if apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	// Atomically reserve the next number: the upsert row-locks the per-account counter and returns the
	// reserved value via LAST_INSERT_ID, so concurrent creates serialize instead of racing on MAX+1.
	res, err := r.queries.AllocateNextPurchaseOrderNumber(ctx, sqlc.AllocateNextPurchaseOrderNumberParams{
		ID:        sysPropertyID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}
	number, err := res.LastInsertId()
	if err != nil {
		return "", tracing.Trace(span, apierror.NewInternalError(err, "Failed to read the reserved purchase order number."))
	}

	return strconv.FormatInt(number, 10), nil
}

func (r *purchaseOrderRepoImpl) GetSupplierID(ctx context.Context, accountID, purchaseOrderID string) (string, *apierror.APIError) {
	ctx, span := purchaseOrderRepoTracer.Start(ctx, "repository.purchase_order.get_supplier_id")
	defer span.End()

	supplierID, err := r.queries.GetPurchaseOrderSupplierID(ctx, sqlc.GetPurchaseOrderSupplierIDParams{
		SalesOrderID: purchaseOrderID,
		AccountID:    accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	return supplierID, nil
}

func (r *purchaseOrderRepoImpl) UpdateAcknowledgmentSent(ctx context.Context, accountID, purchaseOrderID string) *apierror.APIError {
	ctx, span := purchaseOrderRepoTracer.Start(ctx, "repository.purchase_order.update_acknowledgment_sent")
	defer span.End()

	err := r.queries.UpdatePurchaseOrderAcknowledgmentSent(ctx, sqlc.UpdatePurchaseOrderAcknowledgmentSentParams{
		ID:        purchaseOrderID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *purchaseOrderRepoImpl) CreateEmailContact(ctx context.Context, contactID, salesOrderID, accountUserID, notificationTypeCode string) *apierror.APIError {
	ctx, span := purchaseOrderRepoTracer.Start(ctx, "repository.purchase_order.create_email_contact")
	defer span.End()

	err := r.queries.CreateOrderEmailContact(ctx, sqlc.CreateOrderEmailContactParams{
		ID:                   contactID,
		SalesOrderID:         salesOrderID,
		AccountUserID:        accountUserID,
		NotificationTypeCode: notificationTypeCode,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *purchaseOrderRepoImpl) DeleteEmailContactsByOrder(ctx context.Context, salesOrderID string) *apierror.APIError {
	ctx, span := purchaseOrderRepoTracer.Start(ctx, "repository.purchase_order.delete_email_contacts_by_order")
	defer span.End()

	err := r.queries.DeleteOrderEmailContactsByOrder(ctx, salesOrderID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *purchaseOrderRepoImpl) GetEmailContacts(ctx context.Context, salesOrderID string) ([]*domain.PurchaseOrderEmailContact, *apierror.APIError) {
	ctx, span := purchaseOrderRepoTracer.Start(ctx, "repository.purchase_order.get_email_contacts")
	defer span.End()

	rows, err := r.queries.GetOrderEmailContacts(ctx, salesOrderID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	contacts := make([]*domain.PurchaseOrderEmailContact, len(rows))
	for i, row := range rows {
		contacts[i] = &domain.PurchaseOrderEmailContact{
			ID:            row.ID,
			AccountUserID: row.AccountUserID,
		}
	}

	return contacts, nil
}

func (r *purchaseOrderRepoImpl) DeleteCascade(ctx context.Context, accountID, purchaseOrderID string) *apierror.APIError {
	ctx, span := purchaseOrderRepoTracer.Start(ctx, "repository.purchase_order.delete_cascade")
	defer span.End()

	// Delete receiving order lines (via order)
	if err := r.queries.DeleteReceivingOrderLinesByOrderID(ctx, purchaseOrderID); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	// Delete receiving order (by order)
	if err := r.queries.DeleteReceivingOrderByOrderID(ctx, purchaseOrderID); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	// Delete purchase order lines
	if err := r.queries.DeletePurchaseOrderLinesBySalesOrder(ctx, purchaseOrderID); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	// Delete order email contacts
	if err := r.queries.DeleteOrderEmailContactsByOrder(ctx, purchaseOrderID); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	// Delete the purchase order itself
	if err := r.queries.DeletePurchaseOrder(ctx, sqlc.DeletePurchaseOrderParams{
		ID:        purchaseOrderID,
		AccountID: accountID,
	}); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	return nil
}

func (r *purchaseOrderRepoImpl) GetSubmissionRecipients(ctx context.Context, purchaseOrderID string) ([]string, *apierror.APIError) {
	ctx, span := purchaseOrderRepoTracer.Start(ctx, "repository.purchase_order.get_submission_recipients")
	defer span.End()

	rows, err := r.queries.GetPurchaseOrderSubmissionRecipients(ctx, purchaseOrderID)
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

func (r *purchaseOrderRepoImpl) MarkSubmissionSent(ctx context.Context, accountID, purchaseOrderID string) *apierror.APIError {
	ctx, span := purchaseOrderRepoTracer.Start(ctx, "repository.purchase_order.mark_submission_sent")
	defer span.End()

	err := r.queries.MarkPurchaseOrderSubmissionSent(ctx, sqlc.MarkPurchaseOrderSubmissionSentParams{
		ID:        purchaseOrderID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

// Mapping helpers

func mapGetPurchaseOrderRow(row sqlc.GetPurchaseOrderRow) *domain.PurchaseOrder {
	po := &domain.PurchaseOrder{
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
		SupplierName:         row.SupplierName,
		SupplierNumber:       row.SupplierNumber,
		StatusName:           row.StatusName,
		TypeName:             row.TypeName,
		PriorityName:         row.PriorityName,
	}

	po.Note = nullStringToPtr(row.Note)
	po.CarrierID = nullStringToPtr(row.CarrierID)
	po.ServiceLevelID = nullStringToPtr(row.CarrierOptionID)
	po.CarrierBillingType = nullStringToPtr(row.CarrierBillingType)
	po.CarrierBillingAccount = nullStringToPtr(row.CarrierBillingAccount)
	po.ShippingTermID = nullStringToPtr(row.ShippingTermID)
	po.PaymentTermID = nullStringToPtr(row.PaymentTermID)

	if row.IssuedAt.Valid {
		po.IssuedAt = &row.IssuedAt.Time
	}
	if row.CompletedAt.Valid {
		po.CompletedAt = &row.CompletedAt.Time
	}
	if row.PromisedAt.Valid {
		po.PromisedAt = &row.PromisedAt.Time
	}

	po.BillToName = nullStringToPtr(row.BillToName)
	po.BillToStreetLine1 = nullStringToPtr(row.BillToStreetLine1)
	po.BillToStreetLine2 = nullStringToPtr(row.BillToStreetLine2)
	po.BillToLocality = nullStringToPtr(row.BillToLocality)
	po.BillToState = nullStringToPtr(row.BillToState)
	po.BillToPostalCode = nullStringToPtr(row.BillToPostalCode)
	po.BillToCountry = nullStringToPtr(row.BillToCountry)
	po.BillToPhone = nullStringToPtr(row.BillToPhone)
	po.BillToEmail = nullStringToPtr(row.BillToEmail)
	po.ShipToName = nullStringToPtr(row.ShipToName)
	po.ShipToStreetLine1 = nullStringToPtr(row.ShipToStreetLine1)
	po.ShipToStreetLine2 = nullStringToPtr(row.ShipToStreetLine2)
	po.ShipToLocality = nullStringToPtr(row.ShipToLocality)
	po.ShipToState = nullStringToPtr(row.ShipToState)
	po.ShipToPostalCode = nullStringToPtr(row.ShipToPostalCode)
	po.ShipToCountry = nullStringToPtr(row.ShipToCountry)
	po.ShipToPhone = nullStringToPtr(row.ShipToPhone)
	po.ShipToEmail = nullStringToPtr(row.ShipToEmail)

	po.BillToIsDropShip = nullBoolPtr(row.BillToIsDropShip)
	po.BillToCreatedAt = nullTimePtr(row.BillToCreatedAt)
	po.BillToUpdatedAt = nullTimePtr(row.BillToUpdatedAt)
	po.ShipToIsDropShip = nullBoolPtr(row.ShipToIsDropShip)
	po.ShipToCreatedAt = nullTimePtr(row.ShipToCreatedAt)
	po.ShipToUpdatedAt = nullTimePtr(row.ShipToUpdatedAt)

	po.CarrierName = nullStringToPtr(row.CarrierName)
	po.CarrierIsPortalEnabled = nullBoolPtr(row.CarrierIsPortalEnabled)
	po.CarrierCreatedAt = nullTimePtr(row.CarrierCreatedAt)
	po.CarrierUpdatedAt = nullTimePtr(row.CarrierUpdatedAt)
	po.ServiceLevelName = nullStringToPtr(row.CarrierOptionName)
	po.ServiceLevelIsPortalEnabled = nullBoolPtr(row.ServiceLevelIsPortalEnabled)
	po.ServiceLevelToken = nullStringToPtr(row.ServiceLevelToken)
	po.ServiceLevelCreatedAt = nullTimePtr(row.ServiceLevelCreatedAt)
	po.ServiceLevelUpdatedAt = nullTimePtr(row.ServiceLevelUpdatedAt)
	po.PaymentTermName = nullStringToPtr(row.PaymentTermName)
	po.PaymentTermIsActive = nullBoolPtr(row.PaymentTermIsActive)
	po.PaymentTermCreatedAt = nullTimePtr(row.PaymentTermCreatedAt)
	po.PaymentTermUpdatedAt = nullTimePtr(row.PaymentTermUpdatedAt)
	po.ShippingTermName = nullStringToPtr(row.ShippingTermName)
	po.ShippingTermIsFreightExempt = nullBoolPtr(row.ShippingTermIsFreightExempt)
	po.ShippingTermIsCarrierRate = nullBoolPtr(row.ShippingTermIsCarrierRate)
	po.ShippingTermCreatedAt = nullTimePtr(row.ShippingTermCreatedAt)
	po.ShippingTermUpdatedAt = nullTimePtr(row.ShippingTermUpdatedAt)
	po.ReceivingOrderID = nullStringToPtr(row.ReceivingOrderID)
	po.PriorityID = &row.PriorityID

	return po
}

func mapForwardPurchaseOrderRow(row sqlc.ListPurchaseOrdersForwardRow) *domain.PurchaseOrderSummary {
	s := &domain.PurchaseOrderSummary{
		ID:                   row.ID,
		Number:               row.Number,
		StatusCode:           row.StatusCode,
		StatusName:           row.StatusName,
		TypeCode:             row.TypeCode,
		TypeName:             row.TypeName,
		SupplierID:           row.SupplierID,
		SupplierName:         row.SupplierName,
		SupplierNumber:       row.SupplierNumber,
		LineCount:            safeconv.Int64ToInt32(row.LineCount),
		IsAcknowledgmentSent: row.IsAcknowledgmentSent,
		PriorityCode:         constants.PriorityCode(row.PriorityCode),
		PriorityName:         row.PriorityName,
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}
	s.PriorityID = &row.PriorityID
	if row.IssuedAt.Valid {
		s.IssuedAt = &row.IssuedAt.Time
	}
	if row.CompletedAt.Valid {
		s.CompletedAt = &row.CompletedAt.Time
	}
	return s
}

func mapBackwardPurchaseOrderRow(row sqlc.ListPurchaseOrdersBackwardRow) *domain.PurchaseOrderSummary {
	s := &domain.PurchaseOrderSummary{
		ID:                   row.ID,
		Number:               row.Number,
		StatusCode:           row.StatusCode,
		StatusName:           row.StatusName,
		TypeCode:             row.TypeCode,
		TypeName:             row.TypeName,
		SupplierID:           row.SupplierID,
		SupplierName:         row.SupplierName,
		SupplierNumber:       row.SupplierNumber,
		LineCount:            safeconv.Int64ToInt32(row.LineCount),
		IsAcknowledgmentSent: row.IsAcknowledgmentSent,
		PriorityCode:         constants.PriorityCode(row.PriorityCode),
		PriorityName:         row.PriorityName,
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}
	s.PriorityID = &row.PriorityID
	if row.IssuedAt.Valid {
		s.IssuedAt = &row.IssuedAt.Time
	}
	if row.CompletedAt.Valid {
		s.CompletedAt = &row.CompletedAt.Time
	}
	return s
}

func mapPurchaseOrderLinesRow(row sqlc.GetPurchaseOrderLinesRow) *domain.PurchaseOrderLine {
	line := &domain.PurchaseOrderLine{
		ID:                           row.ID,
		ProductSKU:                   row.ProductSku,
		QuantityID:                   row.QuantityID,
		QuantityValue:                row.QuantityValue,
		QuantityUnitID:               row.QuantityUnitID,
		QuantityUnitName:             row.QuantityUnitName,
		QuantityUnitAbbreviation:     row.QuantityUnitAbbreviation,
		QuantityUnitType:             row.QuantityUnitType,
		UnitPriceID:                  row.UnitPriceID,
		UnitPriceValue:               row.UnitPriceValue,
		UnitPriceNumeratorUnitID:     row.UnitPriceNumeratorUnitID,
		UnitPriceNumeratorUnitAbbr:   row.UnitPriceNumeratorUnitAbbreviation,
		UnitPriceDenominatorUnitID:   row.UnitPriceDenominatorUnitID,
		UnitPriceDenominatorUnitAbbr: row.UnitPriceDenominatorUnitAbbreviation,
		CreatedAt:                    row.CreatedAt,
		UpdatedAt:                    row.UpdatedAt,
	}

	if row.LineItemNumber.Valid {
		line.LineItemNumber = row.LineItemNumber.Int32
	}
	if row.ProductDescription.Valid {
		line.ProductDescription = &row.ProductDescription.String
	}
	if row.ProductID.Valid {
		line.ProductID = &row.ProductID.String
	}
	if row.ItemID.Valid {
		line.ItemID = &row.ItemID.String
	}
	if row.ItemSku.Valid {
		line.ItemSKU = &row.ItemSku.String
	}

	// Quantity received
	receivedVal := decimalToString(row.QuantityReceivedValue)
	line.QuantityReceivedValue = &receivedVal

	// Unit cost (nullable)
	if row.UnitCostID.Valid {
		line.UnitCostID = &row.UnitCostID.String
	}
	if row.UnitCostValue.Valid {
		line.UnitCostValue = &row.UnitCostValue.String
	}
	if row.UnitCostNumeratorUnitID.Valid {
		line.UnitCostNumeratorUnitID = &row.UnitCostNumeratorUnitID.String
	}
	if row.UnitCostNumeratorUnitAbbreviation.Valid {
		line.UnitCostNumeratorUnitAbbr = &row.UnitCostNumeratorUnitAbbreviation.String
	}
	if row.UnitCostDenominatorUnitID.Valid {
		line.UnitCostDenominatorUnitID = &row.UnitCostDenominatorUnitID.String
	}
	if row.UnitCostDenominatorUnitAbbreviation.Valid {
		line.UnitCostDenominatorUnitAbbr = &row.UnitCostDenominatorUnitAbbreviation.String
	}

	return line
}
