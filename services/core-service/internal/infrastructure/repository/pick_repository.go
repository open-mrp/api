package repository

import (
	"context"
	gosql "database/sql"
	"errors"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/pagination"
	"github.com/open-mrp/api/shared/safeconv"
	"github.com/open-mrp/api/shared/tracing"
)

var pickRepoTracer = tracing.GetTracer("core-service.pick_repository")

type pickRepoImpl struct {
	queries *sqlc.Queries
}

func NewPickRepo(queries *sqlc.Queries) domain.PickRepo {
	return &pickRepoImpl{queries: queries}
}

func pickCreatedAt(p *domain.Pick) time.Time { return p.CreatedAt }
func pickID(p *domain.Pick) string           { return p.ID }

// Stands in for a missing ship-by date so the sort key is never null; must match the COALESCE in pick.sql or the keyset skips rows.
var pickNoShipByDate = time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)

func pickShipByDate(p *domain.Pick) time.Time {
	if p.ShipByDate == nil {
		return pickNoShipByDate
	}
	return *p.ShipByDate
}

// parseDateFilter is parseDateString under the name the pick and shipment repositories already used. Kept as a one-liner rather than renamed at both call sites so the two date filters cannot drift apart again.
func parseDateFilter(s *string) gosql.NullTime {
	return parseDateString(s)
}

// Parses an inclusive end-date filter, pushing it to the last microsecond of the day so rows created
// during that day still match. Microseconds, not nanoseconds — that is all DATETIME(6) stores.
func parseEndDateFilter(s *string) gosql.NullTime {
	end := parseDateFilter(s)
	if !end.Valid {
		return end
	}
	end.Time = end.Time.Add(24*time.Hour - time.Microsecond)
	return end
}

func mapPickForwardRow(row sqlc.ListPicksForwardRow) *domain.Pick {
	var finishedAt *time.Time
	if row.FinishedAt.Valid {
		finishedAt = &row.FinishedAt.Time
	}
	return &domain.Pick{
		ID:                          row.ID,
		Number:                      row.Number,
		SalesOrderID:                row.SalesOrderID,
		SalesOrderNumber:            row.SalesOrderNumber,
		CustomerID:                  row.CustomerID,
		CustomerName:                row.CustomerName,
		CustomerNumber:              row.CustomerNumber,
		PriorityID:                  row.PriorityID,
		PriorityCode:                constants.PriorityCode(row.PriorityCode),
		PriorityName:                row.PriorityName,
		FinishedAt:                  finishedAt,
		CreatedAt:                   row.CreatedAt,
		UpdatedAt:                   row.UpdatedAt,
		LineCount:                   safeconv.Int64ToInt32(row.LineCount),
		LastShippedAt:               interfaceToTimePtr(row.LastShippedAt),
		PromisedAt:                  nullTimePtr(row.PromisedAt),
		CustomerPONumber:            nullStringToPtr(row.CustomerPoNumber),
		Note:                        nullStringToPtr(row.Note),
		CarrierID:                   nullStringToPtr(row.CarrierID),
		CarrierName:                 nullStringToPtr(row.CarrierName),
		CarrierIsPortalEnabled:      nullBoolPtr(row.CarrierIsPortalEnabled),
		CarrierCreatedAt:            nullTimePtr(row.CarrierCreatedAt),
		CarrierUpdatedAt:            nullTimePtr(row.CarrierUpdatedAt),
		ServiceLevelID:              nullStringToPtr(row.ServiceLevelID),
		ServiceLevelName:            nullStringToPtr(row.ServiceLevelName),
		ServiceLevelIsPortalEnabled: nullBoolPtr(row.ServiceLevelIsPortalEnabled),
		ServiceLevelToken:           nullStringToPtr(row.ServiceLevelToken),
		ServiceLevelCreatedAt:       nullTimePtr(row.ServiceLevelCreatedAt),
		ServiceLevelUpdatedAt:       nullTimePtr(row.ServiceLevelUpdatedAt),
		CarrierBillingType:          nullStringToPtr(row.CarrierBillingType),
		CarrierBillingAccount:       nullStringToPtr(row.CarrierBillingAccount),
		ShipByDate:                  nullTimePtr(row.ShipByDate),
		LeadTimeDays:                nullInt32Ptr(row.LeadTimeDays),
		LeadTimeSource:              leadTimeSourcePtr(row.LeadTimeSourceCode),
		TransitDays:                 nullInt32Ptr(row.TransitDays),
		TransitSource:               transitSourcePtr(row.TransitSourceCode),
		ShippingAddressID:           row.ShippingAddressID,
		ShippingAddressName:         nullStringToPtr(row.ShippingAddressName),
		ShippingAddressPhone:        nullStringToPtr(row.ShippingAddressPhone),
		ShippingAddressEmail:        nullStringToPtr(row.ShippingAddressEmail),
		ShippingAddressIsDropShip:   nullBoolPtr(row.ShippingAddressIsDropShip),
		ShippingAddressGeolocation:  nullStringToPtr(row.ShippingAddressGeolocationID),
		ShippingAddressStreetLine1:  nullStringToPtr(row.ShippingAddressStreetLine1),
		ShippingAddressStreetLine2:  nullStringToPtr(row.ShippingAddressStreetLine2),
		ShippingAddressLocality:     nullStringToPtr(row.ShippingAddressLocality),
		ShippingAddressState:        nullStringToPtr(row.ShippingAddressState),
		ShippingAddressPostalCode:   nullStringToPtr(row.ShippingAddressPostalCode),
		ShippingAddressCountry:      nullStringToPtr(row.ShippingAddressCountry),
	}
}

func mapPickBackwardRow(row sqlc.ListPicksBackwardRow) *domain.Pick {
	var finishedAt *time.Time
	if row.FinishedAt.Valid {
		finishedAt = &row.FinishedAt.Time
	}
	return &domain.Pick{
		ID:                          row.ID,
		Number:                      row.Number,
		SalesOrderID:                row.SalesOrderID,
		SalesOrderNumber:            row.SalesOrderNumber,
		CustomerID:                  row.CustomerID,
		CustomerName:                row.CustomerName,
		CustomerNumber:              row.CustomerNumber,
		PriorityID:                  row.PriorityID,
		PriorityCode:                constants.PriorityCode(row.PriorityCode),
		PriorityName:                row.PriorityName,
		FinishedAt:                  finishedAt,
		CreatedAt:                   row.CreatedAt,
		UpdatedAt:                   row.UpdatedAt,
		LineCount:                   safeconv.Int64ToInt32(row.LineCount),
		LastShippedAt:               interfaceToTimePtr(row.LastShippedAt),
		PromisedAt:                  nullTimePtr(row.PromisedAt),
		CustomerPONumber:            nullStringToPtr(row.CustomerPoNumber),
		Note:                        nullStringToPtr(row.Note),
		CarrierID:                   nullStringToPtr(row.CarrierID),
		CarrierName:                 nullStringToPtr(row.CarrierName),
		CarrierIsPortalEnabled:      nullBoolPtr(row.CarrierIsPortalEnabled),
		CarrierCreatedAt:            nullTimePtr(row.CarrierCreatedAt),
		CarrierUpdatedAt:            nullTimePtr(row.CarrierUpdatedAt),
		ServiceLevelID:              nullStringToPtr(row.ServiceLevelID),
		ServiceLevelName:            nullStringToPtr(row.ServiceLevelName),
		ServiceLevelIsPortalEnabled: nullBoolPtr(row.ServiceLevelIsPortalEnabled),
		ServiceLevelToken:           nullStringToPtr(row.ServiceLevelToken),
		ServiceLevelCreatedAt:       nullTimePtr(row.ServiceLevelCreatedAt),
		ServiceLevelUpdatedAt:       nullTimePtr(row.ServiceLevelUpdatedAt),
		CarrierBillingType:          nullStringToPtr(row.CarrierBillingType),
		CarrierBillingAccount:       nullStringToPtr(row.CarrierBillingAccount),
		ShipByDate:                  nullTimePtr(row.ShipByDate),
		LeadTimeDays:                nullInt32Ptr(row.LeadTimeDays),
		LeadTimeSource:              leadTimeSourcePtr(row.LeadTimeSourceCode),
		TransitDays:                 nullInt32Ptr(row.TransitDays),
		TransitSource:               transitSourcePtr(row.TransitSourceCode),
		ShippingAddressID:           row.ShippingAddressID,
		ShippingAddressName:         nullStringToPtr(row.ShippingAddressName),
		ShippingAddressPhone:        nullStringToPtr(row.ShippingAddressPhone),
		ShippingAddressEmail:        nullStringToPtr(row.ShippingAddressEmail),
		ShippingAddressIsDropShip:   nullBoolPtr(row.ShippingAddressIsDropShip),
		ShippingAddressGeolocation:  nullStringToPtr(row.ShippingAddressGeolocationID),
		ShippingAddressStreetLine1:  nullStringToPtr(row.ShippingAddressStreetLine1),
		ShippingAddressStreetLine2:  nullStringToPtr(row.ShippingAddressStreetLine2),
		ShippingAddressLocality:     nullStringToPtr(row.ShippingAddressLocality),
		ShippingAddressState:        nullStringToPtr(row.ShippingAddressState),
		ShippingAddressPostalCode:   nullStringToPtr(row.ShippingAddressPostalCode),
		ShippingAddressCountry:      nullStringToPtr(row.ShippingAddressCountry),
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
	var productID *string
	if row.ProductID.Valid {
		productID = &row.ProductID.String
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
		OrderLineProductID:                   productID,
		OrderLineItemID:                      nullStringToPtr(row.OrderLineItemID),
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
	endDate := parseEndDateFilter(params.EndDate)

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

	// An unset sort means ship-by date, so a caller that never sends the parameter still gets the urgent-first order.
	sortByShipBy := params.Sort != constants.PickSortCreatedAt
	sortKey := pickCreatedAt
	if sortByShipBy {
		sortKey = pickShipByDate
	}

	forwardParams := sqlc.ListPicksForwardParams{
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
		SortByShipBy:               sortByShipBy,
		Limit:                      params.Limit + 1,
	}

	var cursorDir *pagination.Direction
	var picks []*domain.Pick
	backward := false

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationErrorWithParam("Invalid pagination cursor.", "cursor")
		}
		cursorDir = &cur.Direction
		backward = cur.Direction == pagination.DirectionBackward

		if backward {
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
				SortByShipBy:               sortByShipBy,
				CursorCreatedAt:            cur.OccurredAt,
				CursorShipByDate:           cur.OccurredAt,
				CursorID:                   cur.ID,
				Limit:                      params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			picks = make([]*domain.Pick, len(rows))
			for i, row := range rows {
				picks[i] = mapPickBackwardRow(row)
			}
		} else {
			// Only one of the two cursor columns is read; the sort decides which.
			forwardParams.CursorCreatedAt = gosql.NullTime{Time: cur.OccurredAt, Valid: true}
			forwardParams.CursorShipByDate = gosql.NullTime{Time: cur.OccurredAt, Valid: true}
			forwardParams.CursorID = gosql.NullString{String: cur.ID, Valid: true}
		}
	}

	if !backward {
		rows, err := r.queries.ListPicksForward(ctx, forwardParams)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		picks = make([]*domain.Pick, len(rows))
		for i, row := range rows {
			picks[i] = mapPickForwardRow(row)
		}
	}

	result, pageInfo := pagination.BuildPageString(picks, params.Limit, cursorDir, sortKey, pickID)
	if apiErr := r.attachProgress(ctx, result); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
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

	pick := &domain.Pick{
		ID:                          row.ID,
		Number:                      row.Number,
		SalesOrderID:                row.SalesOrderID,
		SalesOrderNumber:            row.SalesOrderNumber,
		AccountID:                   accountID,
		CustomerID:                  row.CustomerID,
		CustomerName:                row.CustomerName,
		CustomerNumber:              row.CustomerNumber,
		PriorityID:                  row.PriorityID,
		PriorityCode:                constants.PriorityCode(row.PriorityCode),
		PriorityName:                row.PriorityName,
		FinishedAt:                  finishedAt,
		CreatedAt:                   row.CreatedAt,
		UpdatedAt:                   row.UpdatedAt,
		LineCount:                   safeconv.Int64ToInt32(row.LineCount),
		LastShippedAt:               interfaceToTimePtr(row.LastShippedAt),
		PromisedAt:                  nullTimePtr(row.PromisedAt),
		CustomerPONumber:            nullStringToPtr(row.CustomerPoNumber),
		Note:                        nullStringToPtr(row.Note),
		CarrierID:                   nullStringToPtr(row.CarrierID),
		CarrierName:                 nullStringToPtr(row.CarrierName),
		CarrierIsPortalEnabled:      nullBoolPtr(row.CarrierIsPortalEnabled),
		CarrierCreatedAt:            nullTimePtr(row.CarrierCreatedAt),
		CarrierUpdatedAt:            nullTimePtr(row.CarrierUpdatedAt),
		ServiceLevelID:              nullStringToPtr(row.ServiceLevelID),
		ServiceLevelName:            nullStringToPtr(row.ServiceLevelName),
		ServiceLevelIsPortalEnabled: nullBoolPtr(row.ServiceLevelIsPortalEnabled),
		ServiceLevelToken:           nullStringToPtr(row.ServiceLevelToken),
		ServiceLevelCreatedAt:       nullTimePtr(row.ServiceLevelCreatedAt),
		ServiceLevelUpdatedAt:       nullTimePtr(row.ServiceLevelUpdatedAt),
		CarrierBillingType:          nullStringToPtr(row.CarrierBillingType),
		CarrierBillingAccount:       nullStringToPtr(row.CarrierBillingAccount),
		ShipByDate:                  nullTimePtr(row.ShipByDate),
		LeadTimeDays:                nullInt32Ptr(row.LeadTimeDays),
		LeadTimeSource:              leadTimeSourcePtr(row.LeadTimeSourceCode),
		TransitDays:                 nullInt32Ptr(row.TransitDays),
		TransitSource:               transitSourcePtr(row.TransitSourceCode),
		ShippingAddressID:           row.ShippingAddressID,
		ShippingAddressName:         nullStringToPtr(row.ShippingAddressName),
		ShippingAddressPhone:        nullStringToPtr(row.ShippingAddressPhone),
		ShippingAddressEmail:        nullStringToPtr(row.ShippingAddressEmail),
		ShippingAddressIsDropShip:   nullBoolPtr(row.ShippingAddressIsDropShip),
		ShippingAddressGeolocation:  nullStringToPtr(row.ShippingAddressGeolocationID),
		ShippingAddressStreetLine1:  nullStringToPtr(row.ShippingAddressStreetLine1),
		ShippingAddressStreetLine2:  nullStringToPtr(row.ShippingAddressStreetLine2),
		ShippingAddressLocality:     nullStringToPtr(row.ShippingAddressLocality),
		ShippingAddressState:        nullStringToPtr(row.ShippingAddressState),
		ShippingAddressPostalCode:   nullStringToPtr(row.ShippingAddressPostalCode),
		ShippingAddressCountry:      nullStringToPtr(row.ShippingAddressCountry),
	}

	if apiErr := r.attachProgress(ctx, []*domain.Pick{pick}); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return pick, nil
}

// Fills the picked/packed fractions on already-built picks. It sits beside the other roll-ups in
// the repository so no caller can hand back a pick that silently reports zero progress.
func (r *pickRepoImpl) attachProgress(ctx context.Context, picks []*domain.Pick) *apierror.APIError {
	if len(picks) == 0 {
		return nil
	}
	ids := make([]string, len(picks))
	for i, p := range picks {
		ids[i] = p.ID
	}
	progress, apiErr := r.GetProgress(ctx, ids)
	if apiErr != nil {
		return apiErr
	}
	for _, p := range picks {
		p.PickedCompletion = progress[p.ID].PickedCompletion
		p.PackedCompletion = progress[p.ID].PackedCompletion
	}
	return nil
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

func (r *pickRepoImpl) GetProgress(ctx context.Context, pickIDs []string) (map[string]domain.PickProgress, *apierror.APIError) {
	ctx, span := pickRepoTracer.Start(ctx, "repository.pick.get_progress")
	defer span.End()

	if len(pickIDs) == 0 {
		return map[string]domain.PickProgress{}, nil
	}

	rows, err := r.queries.GetPickProgress(ctx, pickIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	progress := make(map[string]domain.PickProgress, len(rows))
	for _, row := range rows {
		ordered := parseDecimalOrZero(decimalToString(row.QuantityOrdered))
		progress[row.PickID] = domain.PickProgress{
			PickedCompletion: completionFraction(decimalToString(row.QuantityPicked), ordered),
			PackedCompletion: completionFraction(decimalToString(row.QuantityPacked), ordered),
		}
	}
	return progress, nil
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

func (r *pickRepoImpl) GetShipmentIDs(ctx context.Context, accountID, pickID string) ([]string, *apierror.APIError) {
	ids, err := r.queries.GetShipmentIDsByPick(ctx, sqlc.GetShipmentIDsByPickParams{
		PickID:    pickID,
		AccountID: accountID,
	})
	if err != nil {
		return nil, apierror.NewInternalError(err, "failed to list shipment ids for pick")
	}
	return ids, nil
}

// Converts the stored lead-time source code, which is nullable, into the typed constant the
// resource exposes. An unrecognised code reads as absent rather than inventing a value.
func leadTimeSourcePtr(ns gosql.NullString) *constants.LeadTimeSource {
	if !ns.Valid || ns.String == "" {
		return nil
	}
	v := constants.LeadTimeSource(ns.String)
	if !v.IsValid() {
		return nil
	}
	return &v
}

// Converts the stored transit source code the same way its lead-time counterpart does.
func transitSourcePtr(ns gosql.NullString) *constants.TransitSource {
	if !ns.Valid || ns.String == "" {
		return nil
	}
	v := constants.TransitSource(ns.String)
	if !v.IsValid() {
		return nil
	}
	return &v
}
