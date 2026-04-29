package repository

import (
	"context"
	gosql "database/sql"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/tracing"
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

func mapICLBackwardRow(row sqlc.ListInventoryChangeLogsBackwardRow) *domain.InventoryChangeLog {
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

func mapGetICLRow(row sqlc.GetInventoryChangeLogRow) *domain.InventoryChangeLog {
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

func buildICLSearchQuery(query *string) gosql.NullString {
	if query == nil || *query == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: "%" + db.EscapeLike(*query) + "%", Valid: true}
}

func (r *inventoryChangeLogRepoImpl) List(ctx context.Context, params domain.ListInventoryChangeLogsParams) (*domain.ListInventoryChangeLogsResult, *apierror.APIError) {
	ctx, span := inventoryChangeLogRepoTracer.Start(ctx, "repository.inventory_change_log.list")
	defer span.End()

	searchQuery := buildICLSearchQuery(params.Query)
	includeItemFilter, itemIDs, includeActionTypeFilter, actionTypeCodes, includeUserFilter, changedByUserIDs := buildICLFilterParams(listICLFilterAdapter{params})

	var startDate, endDate gosql.NullTime
	if params.StartDate != nil {
		startDate = gosql.NullTime{Time: *params.StartDate, Valid: true}
	}
	if params.EndDate != nil {
		endDate = gosql.NullTime{Time: *params.EndDate, Valid: true}
	}

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
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
				SearchQuery:             searchQuery,
				CursorCreatedAt:         cur.OccurredAt,
				CursorID:                cur.ID,
				Limit:                   params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			items := make([]*domain.InventoryChangeLog, len(rows))
			for i, row := range rows {
				items[i] = mapICLBackwardRow(row)
			}
			result, pageInfo := pagination.BuildPageString(items, params.Limit, cursorDir, iclCreatedAt, iclID)
			return &domain.ListInventoryChangeLogsResult{Items: result, PageInfo: pageInfo}, nil
		}

		// Forward
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
			SearchQuery:             searchQuery,
			CursorCreatedAt:         gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:                gosql.NullString{String: cur.ID, Valid: true},
			Limit:                   params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		items := make([]*domain.InventoryChangeLog, len(rows))
		for i, row := range rows {
			items[i] = mapICLForwardRow(row)
		}
		result, pageInfo := pagination.BuildPageString(items, params.Limit, cursorDir, iclCreatedAt, iclID)
		return &domain.ListInventoryChangeLogsResult{Items: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
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
		SearchQuery:             searchQuery,
		Limit:                   params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	items := make([]*domain.InventoryChangeLog, len(rows))
	for i, row := range rows {
		items[i] = mapICLForwardRow(row)
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
