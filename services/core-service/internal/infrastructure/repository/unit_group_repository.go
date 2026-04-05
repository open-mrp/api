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

var unitGroupRepoTracer = tracing.GetTracer("core-service.unit_group_repository")

type unitGroupRepoImpl struct {
	queries *sqlc.Queries
}

func NewUnitGroupRepo(queries *sqlc.Queries) domain.UnitGroupRepo {
	return &unitGroupRepoImpl{queries: queries}
}

func unitGroupCreatedAt(ug *domain.UnitGroupFull) time.Time { return ug.CreatedAt }
func unitGroupID(ug *domain.UnitGroupFull) string           { return ug.ID }

func mapLightUnit(id, name, abbreviation, unitType, ratioNum, ratioDenom, offsetNum, offsetDenom string, isBaseUnit bool, accountID gosql.NullString, createdAt, updatedAt time.Time) domain.LightUnit {
	var acctID *string
	if accountID.Valid {
		acctID = &accountID.String
	}
	return domain.LightUnit{
		ID:                id,
		Name:              name,
		Abbreviation:      abbreviation,
		Type:              unitType,
		RatioNumerator:    ratioNum,
		RatioDenominator:  ratioDenom,
		OffsetNumerator:   offsetNum,
		OffsetDenominator: offsetDenom,
		IsBaseUnit:        isBaseUnit,
		AccountID:         acctID,
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
	}
}

func mapForwardUnitGroupRow(row sqlc.ListUnitGroupsForwardRow) *domain.UnitGroupFull {
	var accountID *string
	if row.AccountID.Valid {
		accountID = &row.AccountID.String
	}
	var notes *string
	if row.Notes.Valid {
		notes = &row.Notes.String
	}
	return &domain.UnitGroupFull{
		ID:        row.ID,
		Name:      row.Name,
		Notes:     notes,
		Type:      row.UnitTypeCode,
		BaseUnit:  mapLightUnit(row.BaseUnitID, row.BaseUnitName, row.BaseUnitAbbreviation, row.BaseUnitType, row.BaseUnitRatioNumerator, row.BaseUnitRatioDenominator, row.BaseUnitOffsetNumerator, row.BaseUnitOffsetDenominator, row.BaseUnitIsBaseUnit, row.BaseUnitAccountID, row.BaseUnitCreatedAt, row.BaseUnitUpdatedAt),
		AccountID: accountID,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func mapBackwardUnitGroupRow(row sqlc.ListUnitGroupsBackwardRow) *domain.UnitGroupFull {
	var accountID *string
	if row.AccountID.Valid {
		accountID = &row.AccountID.String
	}
	var notes *string
	if row.Notes.Valid {
		notes = &row.Notes.String
	}
	return &domain.UnitGroupFull{
		ID:        row.ID,
		Name:      row.Name,
		Notes:     notes,
		Type:      row.UnitTypeCode,
		BaseUnit:  mapLightUnit(row.BaseUnitID, row.BaseUnitName, row.BaseUnitAbbreviation, row.BaseUnitType, row.BaseUnitRatioNumerator, row.BaseUnitRatioDenominator, row.BaseUnitOffsetNumerator, row.BaseUnitOffsetDenominator, row.BaseUnitIsBaseUnit, row.BaseUnitAccountID, row.BaseUnitCreatedAt, row.BaseUnitUpdatedAt),
		AccountID: accountID,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func mapGetUnitGroupRow(row sqlc.GetUnitGroupRow) *domain.UnitGroupFull {
	var accountID *string
	if row.AccountID.Valid {
		accountID = &row.AccountID.String
	}
	var notes *string
	if row.Notes.Valid {
		notes = &row.Notes.String
	}
	return &domain.UnitGroupFull{
		ID:        row.ID,
		Name:      row.Name,
		Notes:     notes,
		Type:      row.UnitTypeCode,
		BaseUnit:  mapLightUnit(row.BaseUnitID, row.BaseUnitName, row.BaseUnitAbbreviation, row.BaseUnitType, row.BaseUnitRatioNumerator, row.BaseUnitRatioDenominator, row.BaseUnitOffsetNumerator, row.BaseUnitOffsetDenominator, row.BaseUnitIsBaseUnit, row.BaseUnitAccountID, row.BaseUnitCreatedAt, row.BaseUnitUpdatedAt),
		AccountID: accountID,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func mapUnitGroupUnitRow(row sqlc.ListUnitGroupUnitsRow) *domain.UnitGroupUnit {
	return &domain.UnitGroupUnit{
		ID:                 row.ID,
		UnitID:             row.UnitID,
		UnitGroupID:        row.UnitGroupID,
		DiscountPercentage: row.DiscountPercentage,
		DiscountFixed:      row.DiscountFixed,
		IsVisible:          row.IsVisible,
		Unit:               mapLightUnit(row.UnitID, row.UnitName, row.UnitAbbreviation, row.UnitType, row.UnitRatioNumerator, row.UnitRatioDenominator, row.UnitOffsetNumerator, row.UnitOffsetDenominator, row.UnitIsBaseUnit, row.UnitAccountID, row.UnitCreatedAt, row.UnitUpdatedAt),
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}
}

func mapGetUnitGroupUnitRow(row sqlc.GetUnitGroupUnitRow) *domain.UnitGroupUnit {
	return &domain.UnitGroupUnit{
		ID:                 row.ID,
		UnitID:             row.UnitID,
		UnitGroupID:        row.UnitGroupID,
		DiscountPercentage: row.DiscountPercentage,
		DiscountFixed:      row.DiscountFixed,
		IsVisible:          row.IsVisible,
		Unit:               mapLightUnit(row.UnitID, row.UnitName, row.UnitAbbreviation, row.UnitType, row.UnitRatioNumerator, row.UnitRatioDenominator, row.UnitOffsetNumerator, row.UnitOffsetDenominator, row.UnitIsBaseUnit, row.UnitAccountID, row.UnitCreatedAt, row.UnitUpdatedAt),
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}
}

func (r *unitGroupRepoImpl) fetchUnitGroupUnits(ctx context.Context, unitGroupID string) ([]*domain.UnitGroupUnit, *apierror.APIError) {
	rows, err := r.queries.ListUnitGroupUnits(ctx, unitGroupID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}

	units := make([]*domain.UnitGroupUnit, len(rows))
	for i, row := range rows {
		units[i] = mapUnitGroupUnitRow(row)
	}
	return units, nil
}

func buildUnitGroupSearchParams(query *string) gosql.NullString {
	if query == nil || *query == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: "%" + *query + "%", Valid: true}
}

func buildUnitTypeFilter(t *string) gosql.NullString {
	if t == nil || *t == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: *t, Valid: true}
}

func (r *unitGroupRepoImpl) List(ctx context.Context, params domain.ListUnitGroupsParams) (*domain.ListUnitGroupsResult, *apierror.APIError) {
	ctx, span := unitGroupRepoTracer.Start(ctx, "repository.unit_group.list")
	defer span.End()

	searchQuery := buildUnitGroupSearchParams(params.Query)
	typeFilter := buildUnitTypeFilter(params.Type)
	accountID := gosql.NullString{String: params.AccountID, Valid: true}

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListUnitGroupsBackward(ctx, sqlc.ListUnitGroupsBackwardParams{
				AccountID:       accountID,
				UnitTypeCode:    typeFilter,
				SearchQuery:     searchQuery,
				CursorCreatedAt: cur.OccurredAt,
				CursorID:        cur.ID,
				Limit:           params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			unitGroups := make([]*domain.UnitGroupFull, len(rows))
			for i, row := range rows {
				unitGroups[i] = mapBackwardUnitGroupRow(row)
			}
			result, pageInfo := pagination.BuildPageString(unitGroups, params.Limit, cursorDir, unitGroupCreatedAt, unitGroupID)
			return &domain.ListUnitGroupsResult{UnitGroups: result, PageInfo: pageInfo}, nil
		}

		rows, err := r.queries.ListUnitGroupsForward(ctx, sqlc.ListUnitGroupsForwardParams{
			AccountID:       accountID,
			UnitTypeCode:    typeFilter,
			SearchQuery:     searchQuery,
			CursorCreatedAt: gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:        gosql.NullString{String: cur.ID, Valid: true},
			Limit:           params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		unitGroups := make([]*domain.UnitGroupFull, len(rows))
		for i, row := range rows {
			unitGroups[i] = mapForwardUnitGroupRow(row)
		}
		result, pageInfo := pagination.BuildPageString(unitGroups, params.Limit, cursorDir, unitGroupCreatedAt, unitGroupID)
		return &domain.ListUnitGroupsResult{UnitGroups: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListUnitGroupsForward(ctx, sqlc.ListUnitGroupsForwardParams{
		AccountID:    accountID,
		UnitTypeCode: typeFilter,
		SearchQuery:  searchQuery,
		Limit:        params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	unitGroups := make([]*domain.UnitGroupFull, len(rows))
	for i, row := range rows {
		unitGroups[i] = mapForwardUnitGroupRow(row)
	}
	result, pageInfo := pagination.BuildPageString(unitGroups, params.Limit, cursorDir, unitGroupCreatedAt, unitGroupID)
	return &domain.ListUnitGroupsResult{UnitGroups: result, PageInfo: pageInfo}, nil
}

func (r *unitGroupRepoImpl) Get(ctx context.Context, params domain.GetUnitGroupParams) (*domain.UnitGroupFull, *apierror.APIError) {
	ctx, span := unitGroupRepoTracer.Start(ctx, "repository.unit_group.get")
	defer span.End()

	row, err := r.queries.GetUnitGroup(ctx, sqlc.GetUnitGroupParams{
		ID:        params.UnitGroupID,
		AccountID: gosql.NullString{String: params.AccountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	ug := mapGetUnitGroupRow(row)

	conversions, apiErr := r.fetchUnitGroupUnits(ctx, ug.ID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	ug.UnitConversions = conversions

	return ug, nil
}

func (r *unitGroupRepoImpl) Create(ctx context.Context, id string, params domain.CreateUnitGroupParams) (*domain.UnitGroupFull, *apierror.APIError) {
	ctx, span := unitGroupRepoTracer.Start(ctx, "repository.unit_group.create")
	defer span.End()

	err := r.queries.InsertUnitGroup(ctx, sqlc.InsertUnitGroupParams{
		ID:           id,
		Name:         params.Name,
		Notes:        toNullString(params.Notes),
		UnitTypeCode: params.Type,
		BaseUnitID:   params.BaseUnitID,
		AccountID:    gosql.NullString{String: params.AccountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.Get(ctx, domain.GetUnitGroupParams{AccountID: params.AccountID, UnitGroupID: id})
}

func (r *unitGroupRepoImpl) Update(ctx context.Context, params domain.UpdateUnitGroupParams) (*domain.UnitGroupFull, *apierror.APIError) {
	ctx, span := unitGroupRepoTracer.Start(ctx, "repository.unit_group.update")
	defer span.End()

	updateNotes := params.Notes != nil
	var notesVal gosql.NullString
	if updateNotes {
		notesVal = toNullString(*params.Notes)
	}

	result, err := r.queries.UpdateUnitGroup(ctx, sqlc.UpdateUnitGroupParams{
		ID:          params.UnitGroupID,
		AccountID:   gosql.NullString{String: params.AccountID, Valid: true},
		Name:        toNullString(params.Name),
		UpdateNotes: updateNotes,
		Notes:       notesVal,
		BaseUnitID:  toNullString(params.BaseUnitID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Unit group not found."))
	}

	return r.Get(ctx, domain.GetUnitGroupParams{AccountID: params.AccountID, UnitGroupID: params.UnitGroupID})
}

func (r *unitGroupRepoImpl) Delete(ctx context.Context, params domain.DeleteUnitGroupParams) *apierror.APIError {
	ctx, span := unitGroupRepoTracer.Start(ctx, "repository.unit_group.delete")
	defer span.End()

	result, err := r.queries.DeleteUnitGroup(ctx, sqlc.DeleteUnitGroupParams{
		ID:        params.UnitGroupID,
		AccountID: gosql.NullString{String: params.AccountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Unit group not found."))
	}

	return nil
}

func (r *unitGroupRepoImpl) ExistsByName(ctx context.Context, accountID, name string, excludeID *string) (bool, *apierror.APIError) {
	ctx, span := unitGroupRepoTracer.Start(ctx, "repository.unit_group.exists_by_name")
	defer span.End()

	count, err := r.queries.CountUnitGroupsByName(ctx, sqlc.CountUnitGroupsByNameParams{
		Name:      name,
		AccountID: gosql.NullString{String: accountID, Valid: true},
		ExcludeID: toNullString(excludeID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return count > 0, nil
}

func (r *unitGroupRepoImpl) UpsertUnitGroupUnit(ctx context.Context, id string, params domain.UpsertUnitGroupUnitParams) (*domain.UnitGroupUnit, *apierror.APIError) {
	ctx, span := unitGroupRepoTracer.Start(ctx, "repository.unit_group.upsert_unit_group_unit")
	defer span.End()

	err := r.queries.UpsertUnitGroupUnit(ctx, sqlc.UpsertUnitGroupUnitParams{
		ID:                 id,
		UnitGroupID:        params.UnitGroupID,
		UnitID:             params.UnitID,
		DiscountPercentage: params.DiscountPercentage,
		DiscountFixed:      params.DiscountFixed,
		IsVisible:          params.IsVisible,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	row, err := r.queries.GetUnitGroupUnit(ctx, sqlc.GetUnitGroupUnitParams{
		ID:          id,
		UnitGroupID: params.UnitGroupID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapGetUnitGroupUnitRow(row), nil
}

func (r *unitGroupRepoImpl) DeleteUnitGroupUnit(ctx context.Context, params domain.DeleteUnitGroupUnitParams) *apierror.APIError {
	ctx, span := unitGroupRepoTracer.Start(ctx, "repository.unit_group.delete_unit_group_unit")
	defer span.End()

	result, err := r.queries.DeleteUnitGroupUnitByID(ctx, sqlc.DeleteUnitGroupUnitByIDParams{
		ID:          params.UnitGroupUnitID,
		UnitGroupID: params.UnitGroupID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Unit group unit not found."))
	}

	return nil
}

func (r *unitGroupRepoImpl) ListUnits(ctx context.Context, unitGroupID string) ([]*domain.UnitGroupUnit, *apierror.APIError) {
	return r.fetchUnitGroupUnits(ctx, unitGroupID)
}

func (r *unitGroupRepoImpl) GetUnit(ctx context.Context, params domain.GetUnitGroupUnitParams) (*domain.UnitGroupUnit, *apierror.APIError) {
	ctx, span := unitGroupRepoTracer.Start(ctx, "repository.unit_group.get_unit")
	defer span.End()

	row, err := r.queries.GetUnitGroupUnit(ctx, sqlc.GetUnitGroupUnitParams{
		ID:          params.UnitGroupUnitID,
		UnitGroupID: params.UnitGroupID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapGetUnitGroupUnitRow(row), nil
}

func (r *unitGroupRepoImpl) DeleteAllUnitGroupUnits(ctx context.Context, accountID, unitGroupID string) *apierror.APIError {
	ctx, span := unitGroupRepoTracer.Start(ctx, "repository.unit_group.delete_all_unit_group_units")
	defer span.End()

	if err := r.queries.DeleteAllUnitGroupUnits(ctx, unitGroupID); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	return nil
}
