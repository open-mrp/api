package repository

import (
	"context"
	gosql "database/sql"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/pagination"
	"github.com/open-mrp/api/shared/safeconv"
	"github.com/open-mrp/api/shared/tracing"
)

var deliveryRepoTracer = tracing.GetTracer("core-service.delivery_repository")

type deliveryRepoImpl struct {
	queries *sqlc.Queries
}

func NewDeliveryRepo(queries *sqlc.Queries) domain.DeliveryRepo {
	return &deliveryRepoImpl{queries: queries}
}

func deliveryCreatedAt(d *domain.DeliverySummary) time.Time { return d.CreatedAt }
func deliveryID(d *domain.DeliverySummary) string           { return d.ID }

func buildDeliverySearchParams(query *string) gosql.NullString {
	if query == nil || *query == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: "%" + db.EscapeLike(*query) + "%", Valid: true}
}

// fetchReceivingOrderRefs names the receiving order created for each of the given purchase orders, keyed by order id.
//
// One receiving order exists per issued purchase order, so this is how a delivery reaches the order it was received against. Batched over a slice so a page of deliveries costs one query.
func (r *deliveryRepoImpl) fetchReceivingOrderRefs(ctx context.Context, orderIDs []string) (map[string]domain.DocumentRef, *apierror.APIError) {
	if len(orderIDs) == 0 {
		return map[string]domain.DocumentRef{}, nil
	}

	rows, err := r.queries.ListReceivingOrderRefsForOrders(ctx, orderIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}

	refs := make(map[string]domain.DocumentRef, len(rows))
	for _, row := range rows {
		refs[row.OrderID] = domain.DocumentRef{ID: row.ID, Number: row.Number}
	}
	return refs, nil
}

func (r *deliveryRepoImpl) List(ctx context.Context, params domain.ListDeliveriesParams) (*domain.ListDeliveriesResult, *apierror.APIError) {
	ctx, span := deliveryRepoTracer.Start(ctx, "repository.delivery.list")
	defer span.End()

	searchQuery := buildDeliverySearchParams(params.Query)

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
	itemIDs := make([]gosql.NullString, len(params.ItemIDs))
	for i, id := range params.ItemIDs {
		itemIDs[i] = gosql.NullString{String: id, Valid: true}
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
			rows, err := r.queries.ListDeliveriesBackward(ctx, sqlc.ListDeliveriesBackwardParams{
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
			deliveries := make([]*domain.DeliverySummary, len(rows))
			for i, row := range rows {
				deliveries[i] = mapBackwardDeliveryRow(row)
			}
			result, pageInfo := pagination.BuildPageString(deliveries, params.Limit, cursorDir, deliveryCreatedAt, deliveryID)
			if apiErr := r.attachReceivingOrderRefs(ctx, result); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			if apiErr := r.attachReceivingOrderRefs(ctx, result); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			if apiErr := r.attachReceivingOrderRefs(ctx, result); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			return &domain.ListDeliveriesResult{Deliveries: result, PageInfo: pageInfo}, nil
		}

		// Forward with cursor
		rows, err := r.queries.ListDeliveriesForward(ctx, sqlc.ListDeliveriesForwardParams{
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
		deliveries := make([]*domain.DeliverySummary, len(rows))
		for i, row := range rows {
			deliveries[i] = mapForwardDeliveryRow(row)
		}
		result, pageInfo := pagination.BuildPageString(deliveries, params.Limit, cursorDir, deliveryCreatedAt, deliveryID)
		return &domain.ListDeliveriesResult{Deliveries: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListDeliveriesForward(ctx, sqlc.ListDeliveriesForwardParams{
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

	deliveries := make([]*domain.DeliverySummary, len(rows))
	for i, row := range rows {
		deliveries[i] = mapForwardDeliveryRow(row)
	}
	result, pageInfo := pagination.BuildPageString(deliveries, params.Limit, cursorDir, deliveryCreatedAt, deliveryID)
	return &domain.ListDeliveriesResult{Deliveries: result, PageInfo: pageInfo}, nil
}

func (r *deliveryRepoImpl) Get(ctx context.Context, params domain.GetDeliveryParams) (*domain.Delivery, *apierror.APIError) {
	ctx, span := deliveryRepoTracer.Start(ctx, "repository.delivery.get")
	defer span.End()

	row, err := r.queries.GetDelivery(ctx, sqlc.GetDeliveryParams{
		ID:        params.DeliveryID,
		AccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	lineRows, err := r.queries.ListDeliveryLines(ctx, params.DeliveryID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	lines := make([]*domain.DeliveryLine, len(lineRows))
	for i, lr := range lineRows {
		lines[i] = mapDeliveryLineRow(lr)
	}

	var acceptedAt *time.Time
	if row.AcceptedAt.Valid {
		acceptedAt = &row.AcceptedAt.Time
	}
	var rejectedAt *time.Time
	if row.RejectedAt.Valid {
		rejectedAt = &row.RejectedAt.Time
	}

	refs, apiErr := r.fetchReceivingOrderRefs(ctx, []string{row.PurchaseOrderID})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	ref := refs[row.PurchaseOrderID]

	return &domain.Delivery{
		ID:                   row.ID,
		Number:               row.Number,
		PurchaseOrderID:      row.PurchaseOrderID,
		PurchaseOrderNumber:  row.PurchaseOrderNumber,
		ReceivingOrderID:     ref.ID,
		ReceivingOrderNumber: ref.Number,
		Status:               row.DeliveryStatusCode,
		Lines:                lines,
		AcceptedAt:           acceptedAt,
		RejectedAt:           rejectedAt,
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}, nil
}

func (r *deliveryRepoImpl) CountByPurchaseOrder(ctx context.Context, purchaseOrderID string) (int64, *apierror.APIError) {
	ctx, span := deliveryRepoTracer.Start(ctx, "repository.delivery.count_by_purchase_order")
	defer span.End()

	count, err := r.queries.CountDeliveriesByPurchaseOrder(ctx, purchaseOrderID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}

	return count, nil
}

func (r *deliveryRepoImpl) CreateDelivery(ctx context.Context, id, number, salesOrderID, accountID, statusCode string, acceptedAt, rejectedAt *time.Time) *apierror.APIError {
	ctx, span := deliveryRepoTracer.Start(ctx, "repository.delivery.create")
	defer span.End()

	acceptedAtNull := gosql.NullTime{}
	if acceptedAt != nil {
		acceptedAtNull = gosql.NullTime{Time: *acceptedAt, Valid: true}
	}
	rejectedAtNull := gosql.NullTime{}
	if rejectedAt != nil {
		rejectedAtNull = gosql.NullTime{Time: *rejectedAt, Valid: true}
	}

	err := r.queries.InsertDelivery(ctx, sqlc.InsertDeliveryParams{
		ID:                 id,
		Number:             number,
		SalesOrderID:       salesOrderID,
		AccountID:          accountID,
		DeliveryStatusCode: statusCode,
		AcceptedAt:         acceptedAtNull,
		RejectedAt:         rejectedAtNull,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *deliveryRepoImpl) CreateDeliveryLine(ctx context.Context, id, deliveryID, receivingOrderLineID, quantityID, unitCostID string, storageLocationID, lotID *string, acceptedAt, rejectedAt *time.Time) *apierror.APIError {
	ctx, span := deliveryRepoTracer.Start(ctx, "repository.delivery.create_line")
	defer span.End()

	storageLocNull := gosql.NullString{}
	if storageLocationID != nil {
		storageLocNull = gosql.NullString{String: *storageLocationID, Valid: true}
	}
	lotNull := gosql.NullString{}
	if lotID != nil {
		lotNull = gosql.NullString{String: *lotID, Valid: true}
	}
	acceptedAtNull := gosql.NullTime{}
	if acceptedAt != nil {
		acceptedAtNull = gosql.NullTime{Time: *acceptedAt, Valid: true}
	}
	rejectedAtNull := gosql.NullTime{}
	if rejectedAt != nil {
		rejectedAtNull = gosql.NullTime{Time: *rejectedAt, Valid: true}
	}

	err := r.queries.InsertDeliveryLine(ctx, sqlc.InsertDeliveryLineParams{
		ID:                   id,
		DeliveryID:           deliveryID,
		ReceivingOrderLineID: receivingOrderLineID,
		QuantityID:           quantityID,
		UnitCostID:           unitCostID,
		StorageLocationID:    storageLocNull,
		LotID:                lotNull,
		AcceptedAt:           acceptedAtNull,
		RejectedAt:           rejectedAtNull,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func mapForwardDeliveryRow(row sqlc.ListDeliveriesForwardRow) *domain.DeliverySummary {
	var acceptedAt *time.Time
	if row.AcceptedAt.Valid {
		acceptedAt = &row.AcceptedAt.Time
	}
	var rejectedAt *time.Time
	if row.RejectedAt.Valid {
		rejectedAt = &row.RejectedAt.Time
	}
	return &domain.DeliverySummary{
		ID:                  row.ID,
		Number:              row.Number,
		PurchaseOrderID:     row.PurchaseOrderID,
		PurchaseOrderNumber: row.PurchaseOrderNumber,
		Status:              row.DeliveryStatusCode,
		LineCount:           safeconv.Int64ToInt32(row.LineCount),
		AcceptedAt:          acceptedAt,
		RejectedAt:          rejectedAt,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}
}

func mapBackwardDeliveryRow(row sqlc.ListDeliveriesBackwardRow) *domain.DeliverySummary {
	var acceptedAt *time.Time
	if row.AcceptedAt.Valid {
		acceptedAt = &row.AcceptedAt.Time
	}
	var rejectedAt *time.Time
	if row.RejectedAt.Valid {
		rejectedAt = &row.RejectedAt.Time
	}
	return &domain.DeliverySummary{
		ID:                  row.ID,
		Number:              row.Number,
		PurchaseOrderID:     row.PurchaseOrderID,
		PurchaseOrderNumber: row.PurchaseOrderNumber,
		Status:              row.DeliveryStatusCode,
		LineCount:           safeconv.Int64ToInt32(row.LineCount),
		AcceptedAt:          acceptedAt,
		RejectedAt:          rejectedAt,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}
}

func mapDeliveryLineRow(row sqlc.ListDeliveryLinesRow) *domain.DeliveryLine {
	line := &domain.DeliveryLine{
		ID:                        row.ID,
		QuantityID:                row.QuantityID,
		QuantityValue:             row.QuantityValue,
		QuantityUnitID:            row.QuantityUnitID,
		QuantityUnitAbbreviation:  row.QuantityUnitAbbreviation,
		UnitCostID:                row.UnitCostID,
		UnitCostValue:             row.UnitCostValue,
		UnitCostNumeratorUnitID:   row.UnitCostNumeratorUnitID,
		UnitCostDenominatorUnitID: row.UnitCostDenominatorUnitID,
		CreatedAt:                 row.CreatedAt,
		UpdatedAt:                 row.UpdatedAt,
	}

	if row.AcceptedAt.Valid {
		line.AcceptedAt = &row.AcceptedAt.Time
	}
	if row.RejectedAt.Valid {
		line.RejectedAt = &row.RejectedAt.Time
	}
	if row.ItemID.Valid {
		line.ItemID = &row.ItemID.String
	}
	if row.ItemSku.Valid {
		line.ItemSKU = &row.ItemSku.String
	}
	if row.ItemDescription.Valid {
		line.ItemDescription = &row.ItemDescription.String
	}
	if row.StorageLocationID.Valid {
		line.LocationID = &row.StorageLocationID.String
	}
	if row.StorageLocationName.Valid {
		line.LocationName = &row.StorageLocationName.String
	}
	if row.LotID.Valid {
		line.LotID = &row.LotID.String
	}
	if row.LotNumber.Valid {
		line.LotNumber = &row.LotNumber.String
	}

	return line
}

// attachReceivingOrderRefs fills in each summary's receiving order from a single lookup over the whole page.
func (r *deliveryRepoImpl) attachReceivingOrderRefs(ctx context.Context, deliveries []*domain.DeliverySummary) *apierror.APIError {
	if len(deliveries) == 0 {
		return nil
	}

	orderIDs := make([]string, len(deliveries))
	for i, d := range deliveries {
		orderIDs[i] = d.PurchaseOrderID
	}

	refs, apiErr := r.fetchReceivingOrderRefs(ctx, orderIDs)
	if apiErr != nil {
		return apiErr
	}
	for _, d := range deliveries {
		if ref, ok := refs[d.PurchaseOrderID]; ok {
			d.ReceivingOrderID, d.ReceivingOrderNumber = ref.ID, ref.Number
		}
	}
	return nil
}
