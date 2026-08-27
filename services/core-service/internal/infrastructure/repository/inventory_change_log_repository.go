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
	"github.com/open-mrp/api/shared/tracing"
)

var inventoryChangeLogRepoTracer = tracing.GetTracer("core-service.inventory_change_log_repository")

type inventoryChangeLogRepoImpl struct {
	queries *sqlc.Queries
}

func NewInventoryChangeLogRepo(queries *sqlc.Queries) domain.InventoryChangeLogRepo {
	return &inventoryChangeLogRepoImpl{queries: queries}
}

func iclCreatedAt(icl *domain.InventoryChangeLog) time.Time { return icl.CreatedAt }
func iclID(icl *domain.InventoryChangeLog) string           { return icl.ID }

func mapICLForwardRow(row sqlc.ListInventoryChangeLogsForwardRow) *domain.InventoryChangeLog {
	icl := &domain.InventoryChangeLog{
		ID:                            row.ID,
		ItemID:                        row.ItemID,
		ItemSKU:                       row.ItemSku,
		ItemTypeCode:                  &row.ItemTypeCode,
		ItemCreatedAt:                 row.ItemCreatedAt,
		ItemUpdatedAt:                 row.ItemUpdatedAt,
		QuantityID:                    row.QuantityID,
		QuantityValue:                 row.QuantityValue,
		QuantityUnitID:                row.QuantityUnitID,
		QuantityUnitName:              row.QuantityUnitName,
		QuantityUnitAbbreviation:      row.QuantityUnitAbbreviation,
		QuantityUnitType:              row.QuantityUnitType,
		QuantityUnitRatioNumerator:    row.QuantityUnitRatioNumerator,
		QuantityUnitRatioDenominator:  row.QuantityUnitRatioDenominator,
		QuantityUnitOffsetNumerator:   row.QuantityUnitOffsetNumerator,
		QuantityUnitOffsetDenominator: row.QuantityUnitOffsetDenominator,
		QuantityUnitCreatedAt:         row.QuantityUnitCreatedAt,
		QuantityUnitUpdatedAt:         row.QuantityUnitUpdatedAt,
		ActionTypeCode:                row.ActionTypeCode,
		AccountID:                     row.AccountID,
		CreatedAt:                     row.CreatedAt,
		UpdatedAt:                     row.UpdatedAt,
	}

	if row.ScanningStationID.Valid {
		icl.ScanningStationID = &row.ScanningStationID.String
	}
	if row.ScanningStationName.Valid {
		icl.ScanningStationName = &row.ScanningStationName.String
	}
	if row.ScanningStationType.Valid {
		icl.ScanningStationType = &row.ScanningStationType.String
	}
	if row.ScanningStationCreatedAt.Valid {
		icl.ScanningStationCreatedAt = &row.ScanningStationCreatedAt.Time
	}
	if row.ScanningStationUpdatedAt.Valid {
		icl.ScanningStationUpdatedAt = &row.ScanningStationUpdatedAt.Time
	}
	if row.ResponsibleUserID.Valid {
		icl.ResponsibleUserID = &row.ResponsibleUserID.String
	}
	if row.ResponsibleUserName.Valid {
		icl.ResponsibleUserName = &row.ResponsibleUserName.String
	}
	if row.ResponsibleUserCreatedAt.Valid {
		icl.ResponsibleUserCreatedAt = &row.ResponsibleUserCreatedAt.Time
	}
	if row.ResponsibleUserUpdatedAt.Valid {
		icl.ResponsibleUserUpdatedAt = &row.ResponsibleUserUpdatedAt.Time
	}

	return icl
}

func mapICLBackwardRow(row sqlc.ListInventoryChangeLogsBackwardRow) *domain.InventoryChangeLog {
	icl := &domain.InventoryChangeLog{
		ID:                            row.ID,
		ItemID:                        row.ItemID,
		ItemSKU:                       row.ItemSku,
		ItemTypeCode:                  &row.ItemTypeCode,
		ItemCreatedAt:                 row.ItemCreatedAt,
		ItemUpdatedAt:                 row.ItemUpdatedAt,
		QuantityID:                    row.QuantityID,
		QuantityValue:                 row.QuantityValue,
		QuantityUnitID:                row.QuantityUnitID,
		QuantityUnitName:              row.QuantityUnitName,
		QuantityUnitAbbreviation:      row.QuantityUnitAbbreviation,
		QuantityUnitType:              row.QuantityUnitType,
		QuantityUnitRatioNumerator:    row.QuantityUnitRatioNumerator,
		QuantityUnitRatioDenominator:  row.QuantityUnitRatioDenominator,
		QuantityUnitOffsetNumerator:   row.QuantityUnitOffsetNumerator,
		QuantityUnitOffsetDenominator: row.QuantityUnitOffsetDenominator,
		QuantityUnitCreatedAt:         row.QuantityUnitCreatedAt,
		QuantityUnitUpdatedAt:         row.QuantityUnitUpdatedAt,
		ActionTypeCode:                row.ActionTypeCode,
		AccountID:                     row.AccountID,
		CreatedAt:                     row.CreatedAt,
		UpdatedAt:                     row.UpdatedAt,
	}

	if row.ScanningStationID.Valid {
		icl.ScanningStationID = &row.ScanningStationID.String
	}
	if row.ScanningStationName.Valid {
		icl.ScanningStationName = &row.ScanningStationName.String
	}
	if row.ScanningStationType.Valid {
		icl.ScanningStationType = &row.ScanningStationType.String
	}
	if row.ScanningStationCreatedAt.Valid {
		icl.ScanningStationCreatedAt = &row.ScanningStationCreatedAt.Time
	}
	if row.ScanningStationUpdatedAt.Valid {
		icl.ScanningStationUpdatedAt = &row.ScanningStationUpdatedAt.Time
	}
	if row.ResponsibleUserID.Valid {
		icl.ResponsibleUserID = &row.ResponsibleUserID.String
	}
	if row.ResponsibleUserName.Valid {
		icl.ResponsibleUserName = &row.ResponsibleUserName.String
	}
	if row.ResponsibleUserCreatedAt.Valid {
		icl.ResponsibleUserCreatedAt = &row.ResponsibleUserCreatedAt.Time
	}
	if row.ResponsibleUserUpdatedAt.Valid {
		icl.ResponsibleUserUpdatedAt = &row.ResponsibleUserUpdatedAt.Time
	}

	return icl
}

func mapGetICLRow(row sqlc.GetInventoryChangeLogRow) *domain.InventoryChangeLog {
	icl := &domain.InventoryChangeLog{
		ID:                            row.ID,
		ItemID:                        row.ItemID,
		ItemSKU:                       row.ItemSku,
		ItemTypeCode:                  &row.ItemTypeCode,
		ItemCreatedAt:                 row.ItemCreatedAt,
		ItemUpdatedAt:                 row.ItemUpdatedAt,
		QuantityID:                    row.QuantityID,
		QuantityValue:                 row.QuantityValue,
		QuantityUnitID:                row.QuantityUnitID,
		QuantityUnitName:              row.QuantityUnitName,
		QuantityUnitAbbreviation:      row.QuantityUnitAbbreviation,
		QuantityUnitType:              row.QuantityUnitType,
		QuantityUnitRatioNumerator:    row.QuantityUnitRatioNumerator,
		QuantityUnitRatioDenominator:  row.QuantityUnitRatioDenominator,
		QuantityUnitOffsetNumerator:   row.QuantityUnitOffsetNumerator,
		QuantityUnitOffsetDenominator: row.QuantityUnitOffsetDenominator,
		QuantityUnitCreatedAt:         row.QuantityUnitCreatedAt,
		QuantityUnitUpdatedAt:         row.QuantityUnitUpdatedAt,
		ActionTypeCode:                row.ActionTypeCode,
		AccountID:                     row.AccountID,
		CreatedAt:                     row.CreatedAt,
		UpdatedAt:                     row.UpdatedAt,
	}

	if row.ScanningStationID.Valid {
		icl.ScanningStationID = &row.ScanningStationID.String
	}
	if row.ScanningStationName.Valid {
		icl.ScanningStationName = &row.ScanningStationName.String
	}
	if row.ScanningStationType.Valid {
		icl.ScanningStationType = &row.ScanningStationType.String
	}
	if row.ScanningStationCreatedAt.Valid {
		icl.ScanningStationCreatedAt = &row.ScanningStationCreatedAt.Time
	}
	if row.ScanningStationUpdatedAt.Valid {
		icl.ScanningStationUpdatedAt = &row.ScanningStationUpdatedAt.Time
	}
	if row.ResponsibleUserID.Valid {
		icl.ResponsibleUserID = &row.ResponsibleUserID.String
	}
	if row.ResponsibleUserName.Valid {
		icl.ResponsibleUserName = &row.ResponsibleUserName.String
	}
	if row.ResponsibleUserCreatedAt.Valid {
		icl.ResponsibleUserCreatedAt = &row.ResponsibleUserCreatedAt.Time
	}
	if row.ResponsibleUserUpdatedAt.Valid {
		icl.ResponsibleUserUpdatedAt = &row.ResponsibleUserUpdatedAt.Time
	}

	return icl
}

func mapListAllICLRow(row sqlc.ListAllInventoryChangeLogsRow) *domain.InventoryChangeLog {
	icl := &domain.InventoryChangeLog{
		ID:                       row.ID,
		ItemID:                   row.ItemID,
		ItemSKU:                  row.ItemSku,
		ItemTypeCode:             &row.ItemTypeCode,
		QuantityID:               row.QuantityID,
		QuantityValue:            row.QuantityValue,
		QuantityUnitID:           row.QuantityUnitID,
		QuantityUnitName:         row.QuantityUnitName,
		QuantityUnitAbbreviation: row.QuantityUnitAbbreviation,
		QuantityUnitType:         row.QuantityUnitType,
		ActionTypeCode:           row.ActionTypeCode,
		AccountID:                row.AccountID,
		CreatedAt:                row.CreatedAt,
		UpdatedAt:                row.UpdatedAt,
	}

	if row.ScanningStationID.Valid {
		icl.ScanningStationID = &row.ScanningStationID.String
	}
	if row.ScanningStationName.Valid {
		icl.ScanningStationName = &row.ScanningStationName.String
	}
	if row.ScanningStationType.Valid {
		icl.ScanningStationType = &row.ScanningStationType.String
	}
	if row.ResponsibleUserID.Valid {
		icl.ResponsibleUserID = &row.ResponsibleUserID.String
	}
	if row.ResponsibleUserName.Valid {
		icl.ResponsibleUserName = &row.ResponsibleUserName.String
	}

	return icl
}

func buildICLFilterParams(params interface {
	getFilterSlices() ([]string, []string, []string)
}) (bool, []string, bool, []string, bool, []gosql.NullString) {
	itemIDs, actionTypeCodes, changedByUserIDsRaw := params.getFilterSlices()

	includeItemFilter := len(itemIDs) > 0
	includeActionTypeFilter := len(actionTypeCodes) > 0
	includeUserFilter := len(changedByUserIDsRaw) > 0

	if itemIDs == nil {
		itemIDs = []string{}
	}
	if actionTypeCodes == nil {
		actionTypeCodes = []string{}
	}

	changedByUserIDs := make([]gosql.NullString, len(changedByUserIDsRaw))
	for i, id := range changedByUserIDsRaw {
		changedByUserIDs[i] = gosql.NullString{String: id, Valid: true}
	}
	if len(changedByUserIDs) == 0 {
		changedByUserIDs = []gosql.NullString{{}}
	}

	return includeItemFilter, itemIDs, includeActionTypeFilter, actionTypeCodes, includeUserFilter, changedByUserIDs
}

type listICLFilterAdapter struct {
	params domain.ListInventoryChangeLogsParams
}

func (a listICLFilterAdapter) getFilterSlices() ([]string, []string, []string) {
	return a.params.ItemIDs, a.params.ActionTypeCodes, a.params.ChangedByUserIDs
}

type exportICLFilterAdapter struct {
	params domain.ExportInventoryChangeLogsParams
}

func (a exportICLFilterAdapter) getFilterSlices() ([]string, []string, []string) {
	return a.params.ItemIDs, a.params.ActionTypeCodes, a.params.ChangedByUserIDs
}

// iclSearchMatches holds the ids a free-text term resolved to, one set per dimension the log is searchable by.
//
// Empty is meaningful and distinct from absent: a term that matched no item still has to exclude every row, which is why matched() reports whether anything was found at all rather than the caller testing the sets.
type iclSearchMatches struct {
	itemIDs    []string
	userIDs    []string
	stationIDs []string
}

func (m iclSearchMatches) matched() bool {
	return len(m.itemIDs) > 0 || len(m.userIDs) > 0 || len(m.stationIDs) > 0
}

// resolveICLSearch turns a free-text term into the id sets the list query filters on.
//
//  1. Returns nil when there is no term, which selects the unfiltered list path.
//  2. Matches the term against item SKU, user name and scanning station name, each on its own table so each can use its own index.
//  3. Returns the three id sets, any of which may be empty.
//
// Resolving here rather than matching a LIKE inside the list query is what keeps that query on an index: see SearchInventoryChangeLogsForward for the measured difference.
func (r *inventoryChangeLogRepoImpl) resolveICLSearch(ctx context.Context, accountID string, query *string) (*iclSearchMatches, *apierror.APIError) {
	if query == nil || *query == "" {
		return nil, nil
	}
	pattern := "%" + db.EscapeLike(*query) + "%"

	itemIDs, err := r.queries.ResolveInventoryChangeLogSearchItemIDs(ctx, sqlc.ResolveInventoryChangeLogSearchItemIDsParams{
		AccountID:   accountID,
		SearchQuery: pattern,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}

	userIDs, err := r.queries.ResolveInventoryChangeLogSearchUserIDs(ctx, gosql.NullString{String: pattern, Valid: true})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}

	stationIDs, err := r.queries.ResolveInventoryChangeLogSearchStationIDs(ctx, sqlc.ResolveInventoryChangeLogSearchStationIDsParams{
		AccountID:   accountID,
		SearchQuery: pattern,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}

	return &iclSearchMatches{itemIDs: itemIDs, userIDs: userIDs, stationIDs: stationIDs}, nil
}

// nullStrings adapts an id set for a nullable column, which sqlc types as []sql.NullString.
func nullStrings(ids []string) []gosql.NullString {
	out := make([]gosql.NullString, len(ids))
	for i, id := range ids {
		out[i] = gosql.NullString{String: id, Valid: true}
	}
	return out
}

func (r *inventoryChangeLogRepoImpl) List(ctx context.Context, params domain.ListInventoryChangeLogsParams) (*domain.ListInventoryChangeLogsResult, *apierror.APIError) {
	ctx, span := inventoryChangeLogRepoTracer.Start(ctx, "repository.inventory_change_log.list")
	defer span.End()

	search, apiErr := r.resolveICLSearch(ctx, params.AccountID, params.Query)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	// A term that matched nothing on any dimension has no page to return, and asking the log for one would scan the account to prove it.
	if search != nil && !search.matched() {
		return &domain.ListInventoryChangeLogsResult{Items: []*domain.InventoryChangeLog{}, PageInfo: pagination.PageInfo{}}, nil
	}

	includeItemFilter, itemIDs, includeActionTypeFilter, actionTypeCodes, includeUserFilter, changedByUserIDs := buildICLFilterParams(listICLFilterAdapter{params})

	var startDate, endDate gosql.NullTime
	if params.StartDate != nil {
		startDate = gosql.NullTime{Time: *params.StartDate, Valid: true}
	}
	if params.EndDate != nil {
		endDate = gosql.NullTime{Time: *params.EndDate, Valid: true}
	}

	limit := params.Limit + 1

	var cursorDir *pagination.Direction
	var cursorCreatedAt gosql.NullTime
	var cursorID gosql.NullString
	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationErrorWithParam("Invalid pagination cursor.", "cursor")
		}
		cursorDir = &cur.Direction
		cursorCreatedAt = gosql.NullTime{Time: cur.OccurredAt, Valid: true}
		cursorID = gosql.NullString{String: cur.ID, Valid: true}
	}

	backward := cursorDir != nil && *cursorDir == pagination.DirectionBackward

	var items []*domain.InventoryChangeLog

	switch {
	case backward && search != nil:
		rows, err := r.queries.SearchInventoryChangeLogsBackward(ctx, sqlc.SearchInventoryChangeLogsBackwardParams{
			AccountID:               params.AccountID,
			SearchItemIds:           search.itemIDs,
			SearchUserIds:           nullStrings(search.userIDs),
			SearchStationIds:        nullStrings(search.stationIDs),
			IncludeItemFilter:       includeItemFilter,
			ItemIds:                 itemIDs,
			IncludeActionTypeFilter: includeActionTypeFilter,
			ActionTypeCodes:         actionTypeCodes,
			IncludeUserFilter:       includeUserFilter,
			ChangedByUserIds:        changedByUserIDs,
			StartDate:               startDate,
			EndDate:                 endDate,
			CursorCreatedAt:         cursorCreatedAt.Time,
			CursorID:                cursorID.String,
			Limit:                   limit,
			Limit_2:                 limit,
			Limit_3:                 limit,
			Limit_4:                 limit,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		items = make([]*domain.InventoryChangeLog, len(rows))
		for i, row := range rows {
			items[i] = mapICLBackwardRow(sqlc.ListInventoryChangeLogsBackwardRow(row))
		}

	case backward:
		rows, err := r.queries.ListInventoryChangeLogsBackward(ctx, sqlc.ListInventoryChangeLogsBackwardParams{
			AccountID:               params.AccountID,
			IncludeItemFilter:       includeItemFilter,
			ItemIds:                 itemIDs,
			IncludeActionTypeFilter: includeActionTypeFilter,
			ActionTypeCodes:         actionTypeCodes,
			IncludeUserFilter:       includeUserFilter,
			ChangedByUserIds:        changedByUserIDs,
			StartDate:               startDate,
			EndDate:                 endDate,
			CursorCreatedAt:         cursorCreatedAt.Time,
			CursorID:                cursorID.String,
			Limit:                   limit,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		items = make([]*domain.InventoryChangeLog, len(rows))
		for i, row := range rows {
			items[i] = mapICLBackwardRow(row)
		}

	case search != nil:
		rows, err := r.queries.SearchInventoryChangeLogsForward(ctx, sqlc.SearchInventoryChangeLogsForwardParams{
			AccountID:               params.AccountID,
			SearchItemIds:           search.itemIDs,
			SearchUserIds:           nullStrings(search.userIDs),
			SearchStationIds:        nullStrings(search.stationIDs),
			IncludeItemFilter:       includeItemFilter,
			ItemIds:                 itemIDs,
			IncludeActionTypeFilter: includeActionTypeFilter,
			ActionTypeCodes:         actionTypeCodes,
			IncludeUserFilter:       includeUserFilter,
			ChangedByUserIds:        changedByUserIDs,
			StartDate:               startDate,
			EndDate:                 endDate,
			CursorCreatedAt:         cursorCreatedAt,
			CursorID:                cursorID,
			Limit:                   limit,
			Limit_2:                 limit,
			Limit_3:                 limit,
			Limit_4:                 limit,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		items = make([]*domain.InventoryChangeLog, len(rows))
		for i, row := range rows {
			items[i] = mapICLForwardRow(sqlc.ListInventoryChangeLogsForwardRow(row))
		}

	default:
		rows, err := r.queries.ListInventoryChangeLogsForward(ctx, sqlc.ListInventoryChangeLogsForwardParams{
			AccountID:               params.AccountID,
			IncludeItemFilter:       includeItemFilter,
			ItemIds:                 itemIDs,
			IncludeActionTypeFilter: includeActionTypeFilter,
			ActionTypeCodes:         actionTypeCodes,
			IncludeUserFilter:       includeUserFilter,
			ChangedByUserIds:        changedByUserIDs,
			StartDate:               startDate,
			EndDate:                 endDate,
			CursorCreatedAt:         cursorCreatedAt,
			CursorID:                cursorID,
			Limit:                   limit,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		items = make([]*domain.InventoryChangeLog, len(rows))
		for i, row := range rows {
			items[i] = mapICLForwardRow(row)
		}
	}

	result, pageInfo := pagination.BuildPageString(items, params.Limit, cursorDir, iclCreatedAt, iclID)
	return &domain.ListInventoryChangeLogsResult{Items: result, PageInfo: pageInfo}, nil
}

func (r *inventoryChangeLogRepoImpl) Get(ctx context.Context, accountID, id string) (*domain.InventoryChangeLog, *apierror.APIError) {
	ctx, span := inventoryChangeLogRepoTracer.Start(ctx, "repository.inventory_change_log.get")
	defer span.End()

	row, err := r.queries.GetInventoryChangeLog(ctx, sqlc.GetInventoryChangeLogParams{
		ID:        id,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapGetICLRow(row), nil
}

func (r *inventoryChangeLogRepoImpl) ListAll(ctx context.Context, params domain.ExportInventoryChangeLogsParams) ([]*domain.InventoryChangeLog, *apierror.APIError) {
	ctx, span := inventoryChangeLogRepoTracer.Start(ctx, "repository.inventory_change_log.list_all")
	defer span.End()

	includeItemFilter, itemIDs, includeActionTypeFilter, actionTypeCodes, includeUserFilter, changedByUserIDs := buildICLFilterParams(exportICLFilterAdapter{params})

	var startDate, endDate gosql.NullTime
	if params.StartDate != nil {
		startDate = gosql.NullTime{Time: *params.StartDate, Valid: true}
	}
	if params.EndDate != nil {
		endDate = gosql.NullTime{Time: *params.EndDate, Valid: true}
	}

	rows, err := r.queries.ListAllInventoryChangeLogs(ctx, sqlc.ListAllInventoryChangeLogsParams{
		AccountID:               params.AccountID,
		IncludeItemFilter:       includeItemFilter,
		ItemIds:                 itemIDs,
		IncludeActionTypeFilter: includeActionTypeFilter,
		ActionTypeCodes:         actionTypeCodes,
		IncludeUserFilter:       includeUserFilter,
		ChangedByUserIds:        changedByUserIDs,
		StartDate:               startDate,
		EndDate:                 endDate,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	items := make([]*domain.InventoryChangeLog, len(rows))
	for i, row := range rows {
		items[i] = mapListAllICLRow(row)
	}

	return items, nil
}
