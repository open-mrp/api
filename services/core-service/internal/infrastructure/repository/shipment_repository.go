package repository

import (
	"context"
	gosql "database/sql"
	"errors"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/pagination"
	"github.com/open-mrp/api/shared/tracing"
)

var shipmentRepoTracer = tracing.GetTracer("core-service.shipment_repository")

type shipmentRepoImpl struct {
	queries *sqlc.Queries
}

func NewShipmentRepo(queries *sqlc.Queries) domain.ShipmentRepo {
	return &shipmentRepoImpl{queries: queries}
}

func shipmentCreatedAt(s *domain.Shipment) time.Time { return s.CreatedAt }
func shipmentID(s *domain.Shipment) string           { return s.ID }

// Converts a list row to the detail row so both share one mapper — legal only while the two shipment
// queries select the same projection in the same order, and a compile error if they drift.
func shipmentListRow(row sqlc.GetShipmentsByIDsRow) sqlc.GetShipmentRow {
	return sqlc.GetShipmentRow(row)
}

func mapShipmentRow(row sqlc.GetShipmentRow) *domain.Shipment {
	shipment := &domain.Shipment{
		ID:                row.ID,
		Number:            row.Number,
		StatusCode:        row.StatusCode,
		StatusName:        row.StatusName,
		SalesOrderID:      row.SalesOrderID,
		SalesOrderNumber:  row.SalesOrderNumber,
		CustomerID:        row.CustomerID,
		CustomerName:      row.CustomerName,
		CustomerNumber:    row.CustomerNumber,
		CarrierID:         row.CarrierID,
		CarrierName:       row.CarrierName,
		ShippingAddressID: row.ShippingAddressID,
		PriorityCode:      row.PriorityCode,
		CaseCount:         row.CaseCount,
		IsReadyToShip:     row.IsReadyToShip.Valid && row.IsReadyToShip.Bool,
		AccountID:         row.AccountID,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}

	if row.CarrierCode.Valid {
		shipment.CarrierCode = &row.CarrierCode.String
	}
	if row.Note.Valid {
		shipment.Note = &row.Note.String
	}
	if row.BillOfLading.Valid {
		shipment.BillOfLading = &row.BillOfLading.String
	}
	if row.MasterTrackingNumber.Valid {
		shipment.MasterTrackingNumber = &row.MasterTrackingNumber.String
	}
	if row.ShippedAt.Valid {
		shipment.ShippedAt = &row.ShippedAt.Time
	}
	if row.CustomerPoNumber.Valid {
		shipment.CustomerPONumber = &row.CustomerPoNumber.String
	}
	if row.CarrierBillingType.Valid {
		shipment.CarrierBillingType = &row.CarrierBillingType.String
	}
	if row.CarrierBillingAccount.Valid {
		shipment.CarrierBillingAccount = &row.CarrierBillingAccount.String
	}
	carrierPortal := row.CarrierIsPortalEnabled
	shipment.CarrierIsPortalEnabled = &carrierPortal
	carrierCreatedAt := row.CarrierCreatedAt
	shipment.CarrierCreatedAt = &carrierCreatedAt
	carrierUpdatedAt := row.CarrierUpdatedAt
	shipment.CarrierUpdatedAt = &carrierUpdatedAt
	if row.CarrierOptionID.Valid {
		shipment.ServiceLevelID = &row.CarrierOptionID.String
	}
	if row.CarrierOptionName.Valid {
		shipment.ServiceLevelName = &row.CarrierOptionName.String
	}
	shipment.ServiceLevelIsPortalEnabled = nullBoolPtr(row.ServiceLevelIsPortalEnabled)
	shipment.ServiceLevelCreatedAt = nullTimePtr(row.ServiceLevelCreatedAt)
	shipment.ServiceLevelUpdatedAt = nullTimePtr(row.ServiceLevelUpdatedAt)
	if row.ShippingAddressName.Valid {
		shipment.ShippingAddressName = &row.ShippingAddressName.String
	}
	shipment.ShippingAddressPhone = nullStringPtr(row.ShippingAddressPhone)
	shipment.ShippingAddressEmail = nullStringPtr(row.ShippingAddressEmail)
	shipment.ShippingAddressIsDropShip = nullBoolPtr(row.ShippingAddressIsDropShip)
	shipment.ShippingAddressGeolocationID = nullStringPtr(row.ShippingAddressGeolocationID)
	shipment.ShippingAddressStreetLine1 = nullStringPtr(row.ShippingAddressStreetLine1)
	shipment.ShippingAddressStreetLine2 = nullStringPtr(row.ShippingAddressStreetLine2)
	shipment.ShippingAddressLocality = nullStringPtr(row.ShippingAddressLocality)
	shipment.ShippingAddressState = nullStringPtr(row.ShippingAddressState)
	shipment.ShippingAddressPostalCode = nullStringPtr(row.ShippingAddressPostalCode)
	shipment.ShippingAddressCountry = nullStringPtr(row.ShippingAddressCountry)
	if row.ShippedByID.Valid {
		shipment.ShippedByID = &row.ShippedByID.String
	}
	if row.ShippedByName.Valid {
		shipment.ShippedByName = &row.ShippedByName.String
	}
	if row.InvoiceID.Valid {
		shipment.InvoiceID = &row.InvoiceID.String
	}
	if row.InvoiceNumber.Valid {
		shipment.InvoiceNumber = &row.InvoiceNumber.String
	}
	if row.PickID.Valid {
		shipment.PickID = &row.PickID.String
	}
	shipment.PickNumber = nullStringToPtr(row.PickNumber)
	shipment.ServiceLevelToken = nullStringToPtr(row.ServiceLevelToken)
	shipment.CustomerStatusCode = nullStringToPtr(row.CustomerStatusCode)
	shipment.CustomerCommissionPolicy = nullStringToPtr(row.CustomerCommissionPolicy)
	if row.BillingAddressCountry.Valid {
		shipment.BillingAddressCountry = &row.BillingAddressCountry.String
	}
	if row.BillingAddressZip.Valid {
		shipment.BillingAddressZip = &row.BillingAddressZip.String
	}
	shipment.OrderCarrierBillingType = nullStringToPtr(row.OrderCarrierBillingType)
	shipment.OrderCarrierBillingAccount = nullStringToPtr(row.OrderCarrierBillingAccount)

	shipment.CustomerCreatedAt = row.CustomerCreatedAt
	shipment.CustomerUpdatedAt = row.CustomerUpdatedAt
	shipment.SalesOrderCreatedAt = row.SalesOrderCreatedAt
	shipment.SalesOrderUpdatedAt = row.SalesOrderUpdatedAt
	shipment.ShippingAddressCreatedAt = nullTimePtr(row.ShippingAddressCreatedAt)
	shipment.ShippingAddressUpdatedAt = nullTimePtr(row.ShippingAddressUpdatedAt)
	shipment.ShippedByStatusCode = nullStringToPtr(row.ShippedByStatusCode)
	shipment.ShippedByCreatedAt = nullTimePtr(row.ShippedByCreatedAt)
	shipment.ShippedByUpdatedAt = nullTimePtr(row.ShippedByUpdatedAt)
	shipment.InvoiceCreatedAt = nullTimePtr(row.InvoiceCreatedAt)
	shipment.InvoiceUpdatedAt = nullTimePtr(row.InvoiceUpdatedAt)
	shipment.PickCreatedAt = nullTimePtr(row.PickCreatedAt)
	shipment.PickUpdatedAt = nullTimePtr(row.PickUpdatedAt)

	return shipment
}

func (r *shipmentRepoImpl) List(ctx context.Context, params domain.ListShipmentsParams) (*domain.ListShipmentsResult, *apierror.APIError) {
	ctx, span := shipmentRepoTracer.Start(ctx, "repository.shipment.list")
	defer span.End()

	q := shipmentListQuery{
		AccountID:      params.AccountID,
		Status:         nullStringPtr(toNullString(params.Status)),
		Search:         db.NullStringLikePtr(params.Query),
		ItemIDs:        params.ItemIDs,
		ProductLineIDs: params.ProductLineIDs,
		SalesRepIDs:    params.SalesRepIDs,
		StartDate:      parseDateFilter(params.StartDate),
		EndDate:        parseEndDateFilter(params.EndDate),
		Direction:      pagination.DirectionForward,
		Limit:          params.Limit + 1,
	}

	var cursorDir *pagination.Direction
	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationErrorWithParam("Invalid pagination cursor.", "cursor")
		}
		cursorDir = &cur.Direction
		q.Direction = cur.Direction
		q.CursorAt = gosql.NullTime{Time: cur.OccurredAt, Valid: true}
		q.CursorID = gosql.NullString{String: cur.ID, Valid: true}
	}

	buyerIDs, apiErr := r.buyerFilter(ctx, params)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	// A non-nil empty buyer set is a filter nothing can match.
	if buyerIDs != nil && len(buyerIDs) == 0 {
		return &domain.ListShipmentsResult{Shipments: []*domain.Shipment{}, PageInfo: pagination.PageInfo{}}, nil
	}
	q.BuyerIDs = buyerIDs
	if q.Drive, apiErr = r.chooseDrive(ctx, q); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	ids, apiErr := r.listIDs(ctx, q)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	shipments, apiErr := r.getByIDsInOrder(ctx, params.AccountID, ids)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	result, pageInfo := pagination.BuildPageString(shipments, params.Limit, cursorDir, shipmentCreatedAt, shipmentID)
	return &domain.ListShipmentsResult{Shipments: result, PageInfo: pageInfo}, nil
}

// buyerFilter resolves the customer and customer-group filters to the customers a shipment may be for.
// Nil means no filter; an empty, non-nil set means the filters exclude every customer.
func (r *shipmentRepoImpl) buyerFilter(ctx context.Context, params domain.ListShipmentsParams) ([]string, *apierror.APIError) {
	if len(params.CustomerGroupIDs) == 0 {
		if len(params.CustomerIDs) == 0 {
			return nil, nil
		}
		return params.CustomerIDs, nil
	}
	query := "SELECT DISTINCT counterparty_account_id FROM account_relation WHERE owner_account_id = ?" +
		" AND account_group_id IN (" + placeholders(len(params.CustomerGroupIDs)) + ")"
	args := append([]any{params.AccountID}, stringArgs(params.CustomerGroupIDs)...)
	if len(params.CustomerIDs) > 0 {
		query += " AND counterparty_account_id IN (" + placeholders(len(params.CustomerIDs)) + ")"
		args = append(args, stringArgs(params.CustomerIDs)...)
	}
	rows, err := r.queries.DB().QueryContext(ctx, query, args...)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	defer func() { _ = rows.Close() }()
	buyers := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, db.MapSQLError(err)
		}
		buyers = append(buyers, id)
	}
	if apiErr := db.MapSQLError(rows.Err()); apiErr != nil {
		return nil, apiErr
	}
	return buyers, nil
}

// chooseDrive picks the unordered filter with the fewest matches under shipmentMatchCap to drive the
// read, or walks a list-order key when none has so few. MySQL cannot estimate these sets, so they are
// counted. One customer is in list order on the buyer key and is never counted.
func (r *shipmentRepoImpl) chooseDrive(ctx context.Context, q shipmentListQuery) (shipmentDrive, *apierror.APIError) {
	candidates := map[shipmentDrive][]string{}
	if len(q.BuyerIDs) > 1 {
		candidates[shipmentDriveBuyers] = q.BuyerIDs
	}
	if len(q.ItemIDs) > 0 {
		candidates[shipmentDriveItems] = q.ItemIDs
	}
	if len(q.ProductLineIDs) > 0 {
		candidates[shipmentDriveProductLines] = q.ProductLineIDs
	}
	if len(q.SalesRepIDs) > 0 {
		candidates[shipmentDriveSalesReps] = q.SalesRepIDs
	}
	drive, fewest := shipmentDriveListOrder, shipmentMatchCap
	for _, candidate := range []shipmentDrive{shipmentDriveBuyers, shipmentDriveItems, shipmentDriveProductLines, shipmentDriveSalesReps} {
		ids, ok := candidates[candidate]
		if !ok {
			continue
		}
		query, args := buildShipmentMatchCountQuery(q.AccountID, candidate, ids)
		var count int
		if err := r.queries.DB().QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
			return drive, db.MapSQLError(err)
		}
		if count < fewest {
			drive, fewest = candidate, count
		}
	}
	return drive, nil
}

func (r *shipmentRepoImpl) listIDs(ctx context.Context, q shipmentListQuery) ([]string, *apierror.APIError) {
	query, args := buildShipmentListQuery(q)
	rows, err := r.queries.DB().QueryContext(ctx, query, args...)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	defer func() { _ = rows.Close() }()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, db.MapSQLError(err)
		}
		ids = append(ids, id)
	}
	if apiErr := db.MapSQLError(rows.Err()); apiErr != nil {
		return nil, apiErr
	}
	return ids, nil
}

// getByIDsInOrder hydrates ids in the order given. A shipment deleted since its id was read is skipped.
func (r *shipmentRepoImpl) getByIDsInOrder(ctx context.Context, accountID string, ids []string) ([]*domain.Shipment, *apierror.APIError) {
	if len(ids) == 0 {
		return []*domain.Shipment{}, nil
	}
	rows, err := r.queries.GetShipmentsByIDs(ctx, sqlc.GetShipmentsByIDsParams{Ids: ids, AccountID: accountID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	byID := make(map[string]*domain.Shipment, len(rows))
	for _, row := range rows {
		byID[row.ID] = mapShipmentRow(shipmentListRow(row))
	}
	ordered := make([]*domain.Shipment, 0, len(ids))
	for _, id := range ids {
		if shipment, ok := byID[id]; ok {
			ordered = append(ordered, shipment)
		}
	}
	return ordered, nil
}

func (r *shipmentRepoImpl) Get(ctx context.Context, params domain.GetShipmentParams) (*domain.Shipment, *apierror.APIError) {
	ctx, span := shipmentRepoTracer.Start(ctx, "repository.shipment.get")
	defer span.End()

	row, err := r.queries.GetShipment(ctx, sqlc.GetShipmentParams{
		ID:        params.ShipmentID,
		AccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapShipmentRow(row), nil
}

func (r *shipmentRepoImpl) Update(ctx context.Context, params domain.UpdateShipmentParams) (*domain.Shipment, *apierror.APIError) {
	ctx, span := shipmentRepoTracer.Start(ctx, "repository.shipment.update")
	defer span.End()

	_, err := r.queries.UpdateShipment(ctx, sqlc.UpdateShipmentParams{
		Note:                      toNullString(params.Note),
		Number:                    toNullString(params.Number),
		MasterTrackingNumber:      toNullString(params.MasterTrackingNumber.ValuePtr()),
		ClearMasterTrackingNumber: params.MasterTrackingNumber.IsClear(),
		CarrierID:                 toNullString(params.CarrierID),
		CarrierOptionID:           toNullString(params.ServiceLevelID.ValuePtr()),
		ID:                        params.ShipmentID,
		AccountID:                 params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.Get(ctx, domain.GetShipmentParams{
		ShipmentID: params.ShipmentID,
		AccountID:  params.AccountID,
	})
}

// Re-points every shipment on an order to the given ship-to address. Split from the carrier sync because
// an order with no carrier still owes its shipments the address change (legacy updateShipToByOrder).
func (r *shipmentRepoImpl) SyncShipToForOrder(ctx context.Context, accountID, salesOrderID, shippingAddressID string) *apierror.APIError {
	ctx, span := shipmentRepoTracer.Start(ctx, "repository.shipment.sync_ship_to_for_order")
	defer span.End()

	err := r.queries.SyncShipmentShipToForOrder(ctx, sqlc.SyncShipmentShipToForOrderParams{
		ShippingAddressID: shippingAddressID,
		SalesOrderID:      salesOrderID,
		AccountID:         accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

// SyncShippingForOrder re-points every shipment on an order to the given carrier, service level, and ship-to address. Used by the out-of-band shipping-updated consumer to keep shipments in sync with the order.
func (r *shipmentRepoImpl) SyncShippingForOrder(ctx context.Context, params domain.SyncShipmentShippingParams) *apierror.APIError {
	ctx, span := shipmentRepoTracer.Start(ctx, "repository.shipment.sync_shipping_for_order")
	defer span.End()

	err := r.queries.SyncShipmentShippingForOrder(ctx, sqlc.SyncShipmentShippingForOrderParams{
		CarrierID:         params.CarrierID,
		CarrierOptionID:   toNullString(params.ServiceLevelID),
		ShippingAddressID: params.ShippingAddressID,
		SalesOrderID:      params.SalesOrderID,
		AccountID:         params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// The cases carry their own carrier, and their tracking links are built from it, so they move
	// with the shipments rather than being left pointing at the previous carrier.
	err = r.queries.RepointShippingCasesToCarrierByOrder(ctx, sqlc.RepointShippingCasesToCarrierByOrderParams{
		CarrierID:    params.CarrierID,
		SalesOrderID: params.SalesOrderID,
		AccountID:    params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *shipmentRepoImpl) Delete(ctx context.Context, accountID, shipmentID string) *apierror.APIError {
	ctx, span := shipmentRepoTracer.Start(ctx, "repository.shipment.delete")
	defer span.End()

	err := r.queries.DeleteShipment(ctx, sqlc.DeleteShipmentParams{
		ID:        shipmentID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *shipmentRepoImpl) MarkShipped(ctx context.Context, accountID, shipmentID, shippedByID string) *apierror.APIError {
	ctx, span := shipmentRepoTracer.Start(ctx, "repository.shipment.mark_shipped")
	defer span.End()

	changed, err := r.queries.MarkShipmentShipped(ctx, sqlc.MarkShipmentShippedParams{
		ShippedByID: gosql.NullString{String: shippedByID, Valid: shippedByID != ""},
		ID:          shipmentID,
		AccountID:   accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if changed == 0 {
		return tracing.Trace(span, apierror.NewConflictErrorWithParam("Shipment has already been shipped.", "id"))
	}
	return nil
}

func (r *shipmentRepoImpl) MarkVoided(ctx context.Context, accountID, shipmentID string) *apierror.APIError {
	ctx, span := shipmentRepoTracer.Start(ctx, "repository.shipment.mark_voided")
	defer span.End()

	changed, err := r.queries.MarkShipmentVoided(ctx, sqlc.MarkShipmentVoidedParams{
		ID:        shipmentID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if changed == 0 {
		return tracing.Trace(span, apierror.NewConflictErrorWithParam("Shipment is not in shipped status.", "id"))
	}
	return nil
}

func (r *shipmentRepoImpl) FindInvoiceIDByShipment(ctx context.Context, accountID, shipmentID string) (*string, *apierror.APIError) {
	ctx, span := shipmentRepoTracer.Start(ctx, "repository.shipment.find_invoice_id_by_shipment")
	defer span.End()

	invoiceID, err := r.queries.FindInvoiceIDByShipment(ctx, sqlc.FindInvoiceIDByShipmentParams{
		ShipmentID: shipmentID,
		AccountID:  accountID,
	})
	if err != nil {
		if errors.Is(err, gosql.ErrNoRows) {
			return nil, nil
		}
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to find invoice by shipment."))
	}
	return &invoiceID, nil
}

func (r *shipmentRepoImpl) LinkInvoice(ctx context.Context, accountID, shipmentID, invoiceID string) *apierror.APIError {
	ctx, span := shipmentRepoTracer.Start(ctx, "repository.shipment.link_invoice")
	defer span.End()

	err := r.queries.LinkShipmentInvoice(ctx, sqlc.LinkShipmentInvoiceParams{
		InvoiceID: gosql.NullString{String: invoiceID, Valid: true},
		ID:        shipmentID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *shipmentRepoImpl) SetMasterTracking(ctx context.Context, accountID, shipmentID, trackingNumber string) *apierror.APIError {
	ctx, span := shipmentRepoTracer.Start(ctx, "repository.shipment.set_master_tracking")
	defer span.End()

	err := r.queries.SetShipmentMasterTracking(ctx, sqlc.SetShipmentMasterTrackingParams{
		MasterTrackingNumber: gosql.NullString{String: trackingNumber, Valid: trackingNumber != ""},
		ID:                   shipmentID,
		AccountID:            accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *shipmentRepoImpl) IsInAccount(ctx context.Context, accountID, shipmentID string) (bool, *apierror.APIError) {
	ctx, span := shipmentRepoTracer.Start(ctx, "repository.shipment.is_in_account")
	defer span.End()

	exists, err := r.queries.CheckShipmentInAccount(ctx, sqlc.CheckShipmentInAccountParams{
		ID:        shipmentID,
		AccountID: accountID,
	})
	if err != nil {
		return false, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check shipment in account."))
	}
	return exists, nil
}
