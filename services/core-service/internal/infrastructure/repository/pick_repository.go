package repository

import (
	"context"
	gosql "database/sql"
	"errors"
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

var pickRepoTracer = tracing.GetTracer("core-service.pick_repository")

type pickRepoImpl struct {
	queries *sqlc.Queries
}

func NewPickRepo(queries *sqlc.Queries) domain.PickRepo {
	return &pickRepoImpl{queries: queries}
}

func pickCreatedAt(p *domain.PickSummary) time.Time { return p.CreatedAt }
func pickID(p *domain.PickSummary) string           { return p.ID }

// parseDateFilter is parseDateString under the name the pick and shipment repositories already used. Kept as a one-liner rather than renamed at both call sites so the two date filters cannot drift apart again.
func parseDateFilter(s *string) gosql.NullTime {
	return parseDateString(s)
}

func mapPickForwardRow(row sqlc.ListPicksForwardRow) *domain.PickSummary {
	var finishedAt *time.Time
	if row.FinishedAt.Valid {
		finishedAt = &row.FinishedAt.Time
	}
	return &domain.PickSummary{
		ID:               row.ID,
		Number:           row.Number,
		SalesOrderID:     row.SalesOrderID,
		SalesOrderNumber: row.SalesOrderNumber,
		CustomerID:       row.CustomerID,
		CustomerName:     row.CustomerName,
		CustomerNumber:   row.CustomerNumber,
		PriorityID:       row.PriorityID,
		PriorityCode:     constants.PriorityCode(row.PriorityCode),
		PriorityName:     row.PriorityName,
		FinishedAt:       finishedAt,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
}

func mapPickBackwardRow(row sqlc.ListPicksBackwardRow) *domain.PickSummary {
	var finishedAt *time.Time
	if row.FinishedAt.Valid {
		finishedAt = &row.FinishedAt.Time
	}
	return &domain.PickSummary{
		ID:               row.ID,
		Number:           row.Number,
		SalesOrderID:     row.SalesOrderID,
		SalesOrderNumber: row.SalesOrderNumber,
		CustomerID:       row.CustomerID,
		CustomerName:     row.CustomerName,
		CustomerNumber:   row.CustomerNumber,
		PriorityID:       row.PriorityID,
		PriorityCode:     constants.PriorityCode(row.PriorityCode),
		PriorityName:     row.PriorityName,
		FinishedAt:       finishedAt,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
}

func mapGetPickLinesRow(row sqlc.GetPickLinesRow) *domain.PickLine {
	var packedAt *time.Time
	if row.PackedAt.Valid {
		packedAt = &row.PackedAt.Time
	}
	var lineItemNumber int32
	if row.LineItemNumber.Valid {
		lineItemNumber = row.LineItemNumber.Int32
	}
	var productDescription *string
	if row.ProductDescription.Valid {
		productDescription = &row.ProductDescription.String
	}
	return &domain.PickLine{
		ID:                                   row.ID,
		PickID:                               row.PickID,
		SalesOrderLineID:                     row.SalesOrderLineID,
		QuantityID:                           row.QuantityID,
		QuantityValue:                        row.QuantityValue,
		QuantityUnitID:                       row.QuantityUnitID,
		QuantityUnitName:                     row.QuantityUnitName,
		QuantityUnitAbbreviation:             row.QuantityUnitAbbreviation,
		PackedAt:                             packedAt,
		CreatedAt:                            row.CreatedAt,
		UpdatedAt:                            row.UpdatedAt,
		OrderLineItemNumber:                  lineItemNumber,
		OrderLineSKU:                         row.ProductSku,
		OrderLineDescription:                 productDescription,
		OrderedQuantityID:                    row.OrderedQuantityID,
		OrderedQuantityValue:                 row.OrderedQuantityValue,
		OrderedQuantityUnitID:                row.OrderedQuantityUnitID,
		OrderedQuantityUnitName:              row.OrderedQuantityUnitName,
		OrderedQuantityUnitAbbrev:            row.OrderedQuantityUnitAbbreviation,
		UnitPriceID:                          row.UnitPriceID,
		UnitPriceValue:                       row.UnitPriceValue,
		UnitPriceNumeratorUnitID:             row.UnitPriceNumeratorUnitID,
		UnitPriceNumeratorUnitAbbreviation:   row.UnitPriceNumeratorUnitAbbreviation,
		UnitPriceDenominatorUnitID:           row.UnitPriceDenominatorUnitID,
		UnitPriceDenominatorUnitAbbreviation: row.UnitPriceDenominatorUnitAbbreviation,
	}
}

func mapFindLinesToPackRow(row sqlc.FindLinesToPackRow) *domain.PickLine {
	var packedAt *time.Time
	if row.PackedAt.Valid {
		packedAt = &row.PackedAt.Time
	}
	var lineItemNumber int32
	if row.LineItemNumber.Valid {
		lineItemNumber = row.LineItemNumber.Int32
	}
	var productDescription *string
	if row.ProductDescription.Valid {
		productDescription = &row.ProductDescription.String
	}
	return &domain.PickLine{
		ID:                        row.ID,
		PickID:                    row.PickID,
		SalesOrderLineID:          row.SalesOrderLineID,
		QuantityID:                row.QuantityID,
		QuantityValue:             row.QuantityValue,
		QuantityUnitID:            row.QuantityUnitID,
		QuantityUnitName:          row.QuantityUnitName,
		QuantityUnitAbbreviation:  row.QuantityUnitAbbreviation,
		PackedAt:                  packedAt,
		CreatedAt:                 row.CreatedAt,
		UpdatedAt:                 row.UpdatedAt,
		OrderLineItemNumber:       lineItemNumber,
		OrderLineSKU:              row.ProductSku,
		OrderLineDescription:      productDescription,
		OrderedQuantityID:         row.OrderedQuantityID,
		OrderedQuantityValue:      row.OrderedQuantityValue,
		OrderedQuantityUnitID:     row.OrderedQuantityUnitID,
		OrderedQuantityUnitName:   row.OrderedQuantityUnitName,
		OrderedQuantityUnitAbbrev: row.OrderedQuantityUnitAbbreviation,
	}
}

func (r *pickRepoImpl) List(ctx context.Context, params domain.ListPicksParams) (*domain.ListPicksResult, *apierror.APIError) {
	ctx, span := pickRepoTracer.Start(ctx, "repository.pick.list")
	defer span.End()

	searchQuery := db.NullStringLikePtr(params.Query)
	statusFilter := toNullString(params.Status)
	startDate := parseDateFilter(params.StartDate)
	endDate := parseDateFilter(params.EndDate)

	customerIDs := params.CustomerIDs
	if customerIDs == nil {
		customerIDs = []string{}
	}
	customerGroupIDs := toNullStringSlice(params.CustomerGroupIDs)
	if customerGroupIDs == nil {
		customerGroupIDs = []gosql.NullString{}
	}
	departmentIDs := params.DepartmentIDs
	if departmentIDs == nil {
		departmentIDs = []string{}
	}
	productLineIDs := toNullStringSlice(params.ProductLineIDs)
	if productLineIDs == nil {
		productLineIDs = []gosql.NullString{}
	}

	includeCustomerFilter := len(params.CustomerIDs) > 0
	includeCustomerGroupFilter := len(params.CustomerGroupIDs) > 0
	includeDepartmentFilter := len(params.DepartmentIDs) > 0
	includeProductLineFilter := len(params.ProductLineIDs) > 0

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationErrorWithParam("Invalid pagination cursor.", "cursor")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListPicksBackward(ctx, sqlc.ListPicksBackwardParams{
				AccountID:                  params.AccountID,
				SearchQuery:                searchQuery,
				Status:                     statusFilter,
				IncludeCustomerFilter:      includeCustomerFilter,
				CustomerIds:                customerIDs,
				IncludeCustomerGroupFilter: includeCustomerGroupFilter,
				CustomerGroupIds:           customerGroupIDs,
				IncludeDepartmentFilter:    includeDepartmentFilter,
				DepartmentIds:              departmentIDs,
				IncludeProductLineFilter:   includeProductLineFilter,
				ProductLineIds:             productLineIDs,
				StartDate:                  startDate,
				EndDate:                    endDate,
				CursorCreatedAt:            cur.OccurredAt,
				CursorID:                   cur.ID,
				Limit:                      params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			picks := make([]*domain.PickSummary, len(rows))
			for i, row := range rows {
				picks[i] = mapPickBackwardRow(row)
			}
			result, pageInfo := pagination.BuildPageString(picks, params.Limit, cursorDir, pickCreatedAt, pickID)
			return &domain.ListPicksResult{Picks: result, PageInfo: pageInfo}, nil
		}

		// Forward with cursor
		rows, err := r.queries.ListPicksForward(ctx, sqlc.ListPicksForwardParams{
			AccountID:                  params.AccountID,
			SearchQuery:                searchQuery,
			Status:                     statusFilter,
			IncludeCustomerFilter:      includeCustomerFilter,
			CustomerIds:                customerIDs,
			IncludeCustomerGroupFilter: includeCustomerGroupFilter,
			CustomerGroupIds:           customerGroupIDs,
			IncludeDepartmentFilter:    includeDepartmentFilter,
			DepartmentIds:              departmentIDs,
			IncludeProductLineFilter:   includeProductLineFilter,
			ProductLineIds:             productLineIDs,
			StartDate:                  startDate,
			EndDate:                    endDate,
			CursorCreatedAt:            gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:                   gosql.NullString{String: cur.ID, Valid: true},
			Limit:                      params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		picks := make([]*domain.PickSummary, len(rows))
		for i, row := range rows {
			picks[i] = mapPickForwardRow(row)
		}
		result, pageInfo := pagination.BuildPageString(picks, params.Limit, cursorDir, pickCreatedAt, pickID)
		return &domain.ListPicksResult{Picks: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListPicksForward(ctx, sqlc.ListPicksForwardParams{
		AccountID:                  params.AccountID,
		SearchQuery:                searchQuery,
		Status:                     statusFilter,
		IncludeCustomerFilter:      includeCustomerFilter,
		CustomerIds:                customerIDs,
		IncludeCustomerGroupFilter: includeCustomerGroupFilter,
		CustomerGroupIds:           customerGroupIDs,
		IncludeDepartmentFilter:    includeDepartmentFilter,
		DepartmentIds:              departmentIDs,
		IncludeProductLineFilter:   includeProductLineFilter,
		ProductLineIds:             productLineIDs,
		StartDate:                  startDate,
		EndDate:                    endDate,
		Limit:                      params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	picks := make([]*domain.PickSummary, len(rows))
	for i, row := range rows {
		picks[i] = mapPickForwardRow(row)
	}
	result, pageInfo := pagination.BuildPageString(picks, params.Limit, cursorDir, pickCreatedAt, pickID)
	return &domain.ListPicksResult{Picks: result, PageInfo: pageInfo}, nil
}

func (r *pickRepoImpl) Get(ctx context.Context, accountID, pickID string) (*domain.Pick, *apierror.APIError) {
	ctx, span := pickRepoTracer.Start(ctx, "repository.pick.get")
	defer span.End()

	row, err := r.queries.GetPick(ctx, sqlc.GetPickParams{PickID: pickID, AccountID: accountID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	var finishedAt *time.Time
	if row.FinishedAt.Valid {
		finishedAt = &row.FinishedAt.Time
	}

	return &domain.Pick{
		ID:               row.ID,
		Number:           row.Number,
		SalesOrderID:     row.SalesOrderID,
		SalesOrderNumber: row.SalesOrderNumber,
		AccountID:        accountID,
		CustomerID:       row.CustomerID,
		CustomerName:     row.CustomerName,
		CustomerNumber:   row.CustomerNumber,
		PriorityID:       row.PriorityID,
		PriorityCode:     constants.PriorityCode(row.PriorityCode),
		PriorityName:     row.PriorityName,
		FinishedAt:       finishedAt,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}, nil
}

func (r *pickRepoImpl) GetLines(ctx context.Context, pickID string) ([]*domain.PickLine, *apierror.APIError) {
	ctx, span := pickRepoTracer.Start(ctx, "repository.pick.get_lines")
	defer span.End()

	rows, err := r.queries.GetPickLines(ctx, pickID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	lines := make([]*domain.PickLine, len(rows))
	for i, row := range rows {
		lines[i] = mapGetPickLinesRow(row)
	}
	return lines, nil
}

func (r *pickRepoImpl) GetDepartments(ctx context.Context, pickID string) ([]*domain.PickDepartment, *apierror.APIError) {
	ctx, span := pickRepoTracer.Start(ctx, "repository.pick.get_departments")
	defer span.End()

	rows, err := r.queries.GetPickDepartments(ctx, pickID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	departments := make([]*domain.PickDepartment, len(rows))
	for i, row := range rows {
		departments[i] = &domain.PickDepartment{
			ID:   row.ID,
			Name: row.Name,
		}
	}
	return departments, nil
}

func (r *pickRepoImpl) UpdateNumber(ctx context.Context, accountID, pickID, number string) *apierror.APIError {
	ctx, span := pickRepoTracer.Start(ctx, "repository.pick.update_number")
	defer span.End()

	if err := r.queries.UpdatePickNumber(ctx, sqlc.UpdatePickNumberParams{
		Number:    number,
		PickID:    pickID,
		AccountID: accountID,
	}); err != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}
	return nil
}

func (r *pickRepoImpl) UpdateFinishedAt(ctx context.Context, accountID, pickID string, finishedAt time.Time) *apierror.APIError {
	ctx, span := pickRepoTracer.Start(ctx, "repository.pick.update_finished_at")
	defer span.End()

	if err := r.queries.UpdatePickFinishedAt(ctx, sqlc.UpdatePickFinishedAtParams{
		FinishedAt: gosql.NullTime{Time: finishedAt, Valid: true},
		PickID:     pickID,
		AccountID:  accountID,
	}); err != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}
	return nil
}

func (r *pickRepoImpl) HasShippedItems(ctx context.Context, accountID, pickID string) (bool, *apierror.APIError) {
	ctx, span := pickRepoTracer.Start(ctx, "repository.pick.has_shipped_items")
	defer span.End()

	hasShipped, err := r.queries.HasShippedItems(ctx, sqlc.HasShippedItemsParams{
		PickID:    pickID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return hasShipped, nil
}

func (r *pickRepoImpl) VoidAllLines(ctx context.Context, pickID string) *apierror.APIError {
	ctx, span := pickRepoTracer.Start(ctx, "repository.pick.void_all_lines")
	defer span.End()

	if err := r.queries.VoidAllPickLines(ctx, pickID); err != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}
	return nil
}

func (r *pickRepoImpl) DeleteDuplicatePickLines(ctx context.Context, accountID, pickID string) *apierror.APIError {
	ctx, span := pickRepoTracer.Start(ctx, "repository.pick.delete_duplicate_pick_lines")
	defer span.End()

	if err := r.queries.DeleteDuplicatePickLines(ctx, sqlc.DeleteDuplicatePickLinesParams{
		PickID:    pickID,
		AccountID: accountID,
	}); err != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}
	return nil
}

func (r *pickRepoImpl) ClearFinishedAt(ctx context.Context, accountID, pickID string) *apierror.APIError {
	ctx, span := pickRepoTracer.Start(ctx, "repository.pick.clear_finished_at")
	defer span.End()

	if err := r.queries.ClearPickFinishedAt(ctx, sqlc.ClearPickFinishedAtParams{
		PickID:    pickID,
		AccountID: accountID,
	}); err != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}
	return nil
}

func (r *pickRepoImpl) PickAllLines(ctx context.Context, pickID string) *apierror.APIError {
	ctx, span := pickRepoTracer.Start(ctx, "repository.pick.pick_all_lines")
	defer span.End()

	if err := r.queries.PickAllLines(ctx, sqlc.PickAllLinesParams{PickID: pickID}); err != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}
	return nil
}

func (r *pickRepoImpl) GetShipmentNumbers(ctx context.Context, params domain.GetPickShipmentsParams) (*domain.PickShipmentsResult, *apierror.APIError) {
	ctx, span := pickRepoTracer.Start(ctx, "repository.pick.get_shipment_numbers")
	defer span.End()

	var searchQuery gosql.NullString
	if params.Query != nil && *params.Query != "" {
		searchQuery = gosql.NullString{String: "%" + db.EscapeLike(*params.Query) + "%", Valid: true}
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 100
	}

	offset := max(params.Offset, 0)

	numbers, err := r.queries.GetPickShipmentNumbers(ctx, sqlc.GetPickShipmentNumbersParams{
		PickID:      params.PickID,
		AccountID:   params.AccountID,
		SearchQuery: searchQuery,
		Limit:       limit,
		Offset:      offset,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	count, err := r.queries.CountPickShipmentNumbers(ctx, sqlc.CountPickShipmentNumbersParams{
		PickID:      params.PickID,
		AccountID:   params.AccountID,
		SearchQuery: searchQuery,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return &domain.PickShipmentsResult{
		ShipmentNumbers: numbers,
		Count:           safeconv.Int64ToInt32(count),
	}, nil
}

func (r *pickRepoImpl) IsInAccount(ctx context.Context, accountID, pickID string) (bool, *apierror.APIError) {
	ctx, span := pickRepoTracer.Start(ctx, "repository.pick.is_in_account")
	defer span.End()

	exists, err := r.queries.IsPickInAccount(ctx, sqlc.IsPickInAccountParams{
		PickID:    pickID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return exists, nil
}

func (r *pickRepoImpl) FindLinesToPack(ctx context.Context, pickID string) ([]*domain.PickLine, *apierror.APIError) {
	ctx, span := pickRepoTracer.Start(ctx, "repository.pick.find_lines_to_pack")
	defer span.End()

	rows, err := r.queries.FindLinesToPack(ctx, pickID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	lines := make([]*domain.PickLine, len(rows))
	for i, row := range rows {
		lines[i] = mapFindLinesToPackRow(row)
	}
	return lines, nil
}

func (r *pickRepoImpl) PackLines(ctx context.Context, pickID string) *apierror.APIError {
	ctx, span := pickRepoTracer.Start(ctx, "repository.pick.pack_lines")
	defer span.End()

	if err := r.queries.PackPickLines(ctx, pickID); err != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}
	return nil
}

func (r *pickRepoImpl) MarkFinishedIfAllPacked(ctx context.Context, pickID string) *apierror.APIError {
	ctx, span := pickRepoTracer.Start(ctx, "repository.pick.mark_finished_if_all_packed")
	defer span.End()

	if err := r.queries.MarkPickFinishedIfAllPacked(ctx, pickID); err != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}
	return nil
}

func (r *pickRepoImpl) CloseOpenPickLines(ctx context.Context, pickID string) *apierror.APIError {
	ctx, span := pickRepoTracer.Start(ctx, "repository.pick.close_open_pick_lines")
	defer span.End()

	if err := r.queries.CloseOpenPickLines(ctx, pickID); err != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}
	return nil
}

func (r *pickRepoImpl) ReopenIncompletePickLines(ctx context.Context, pickID string) *apierror.APIError {
	ctx, span := pickRepoTracer.Start(ctx, "repository.pick.reopen_incomplete_pick_lines")
	defer span.End()

	if err := r.queries.ReopenIncompletePickLines(ctx, pickID); err != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}
	return nil
}

func (r *pickRepoImpl) CountLines(ctx context.Context, pickID string) (int64, *apierror.APIError) {
	ctx, span := pickRepoTracer.Start(ctx, "repository.pick.count_lines")
	defer span.End()

	count, err := r.queries.CountPickLinesByPick(ctx, pickID)
	if err != nil {
		return 0, tracing.Trace(span, db.MapSQLError(err))
	}
	return count, nil
}

func (r *pickRepoImpl) CountShipmentsByOrder(ctx context.Context, salesOrderID string) (int64, *apierror.APIError) {
	ctx, span := pickRepoTracer.Start(ctx, "repository.pick.count_shipments_by_order")
	defer span.End()

	count, err := r.queries.CountShipmentsByOrder(ctx, salesOrderID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}
	return count, nil
}

func (r *pickRepoImpl) GetSalesOrderForPick(ctx context.Context, accountID, pickID string) (*domain.PickSalesOrder, *apierror.APIError) {
	ctx, span := pickRepoTracer.Start(ctx, "repository.pick.get_sales_order_for_pick")
	defer span.End()

	row, err := r.queries.GetSalesOrderForPick(ctx, sqlc.GetSalesOrderForPickParams{
		PickID:    pickID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	var serviceLevelID *string
	if row.CarrierOptionID.Valid {
		serviceLevelID = &row.CarrierOptionID.String
	}

	return &domain.PickSalesOrder{
		ID:                row.ID,
		Number:            row.Number,
		CarrierID:         row.CarrierID.String,
		ServiceLevelID:    serviceLevelID,
		ShippingAddressID: row.ShippingAddressID,
	}, nil
}

func (r *pickRepoImpl) CreateShipment(ctx context.Context, params domain.CreateShipmentFromPickParams) *apierror.APIError {
	ctx, span := pickRepoTracer.Start(ctx, "repository.pick.create_shipment")
	defer span.End()

	if err := r.queries.CreateShipment(ctx, sqlc.CreateShipmentParams{
		ID:                 params.ID,
		Number:             params.Number,
		SalesOrderID:       params.SalesOrderID,
		CarrierID:          gosql.NullString{String: params.CarrierID, Valid: params.CarrierID != ""},
		CarrierOptionID:    toNullString(params.ServiceLevelID),
		ShippingAddressID:  gosql.NullString{String: params.ShippingAddressID, Valid: params.ShippingAddressID != ""},
		ShipmentStatusCode: params.StatusCode,
		AccountID:          params.AccountID,
	}); err != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}
	return nil
}

func (r *pickRepoImpl) CreateShipmentLine(ctx context.Context, params domain.CreateShipmentLineParams) *apierror.APIError {
	ctx, span := pickRepoTracer.Start(ctx, "repository.pick.create_shipment_line")
	defer span.End()

	if err := r.queries.CreateShipmentLine(ctx, sqlc.CreateShipmentLineParams{
		ID:               params.ID,
		ShipmentID:       params.ShipmentID,
		SalesOrderLineID: params.SalesOrderLineID,
		QuantityID:       params.QuantityID,
	}); err != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}
	return nil
}

func (r *pickRepoImpl) CreateShippingCase(ctx context.Context, params domain.CreateShippingCaseParams) *apierror.APIError {
	ctx, span := pickRepoTracer.Start(ctx, "repository.pick.create_shipping_case")
	defer span.End()

	if err := r.queries.CreateShippingCase(ctx, sqlc.CreateShippingCaseParams{
		ID:              params.ID,
		Number:          params.Number,
		FreightAmountID: params.FreightAmountID,
		FreightWeightID: params.FreightWeightID,
		ShipmentID:      params.ShipmentID,
		CarrierID:       gosql.NullString{String: params.CarrierID, Valid: params.CarrierID != ""},
		AccountID:       params.AccountID,
	}); err != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}
	return nil
}

func (r *pickRepoImpl) CreateQuantity(ctx context.Context, id, value, unitID string) *apierror.APIError {
	ctx, span := pickRepoTracer.Start(ctx, "repository.pick.create_quantity")
	defer span.End()

	if err := r.queries.CreateQuantity(ctx, sqlc.CreateQuantityParams{
		ID:     id,
		Value:  value,
		UnitID: unitID,
	}); err != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}
	return nil
}

func (r *pickRepoImpl) FindIDByShipmentOrder(ctx context.Context, accountID, shipmentID string) (string, *apierror.APIError) {
	ctx, span := pickRepoTracer.Start(ctx, "repository.pick.find_id_by_shipment_order")
	defer span.End()

	pickID, err := r.queries.FindPickIDByShipmentOrder(ctx, sqlc.FindPickIDByShipmentOrderParams{
		AccountID:  accountID,
		ShipmentID: shipmentID,
	})
	if err != nil {
		if errors.Is(err, gosql.ErrNoRows) {
			return "", nil
		}
		return "", tracing.Trace(span, apierror.NewInternalError(err, "Failed to find pick by shipment order."))
	}
	return pickID, nil
}
