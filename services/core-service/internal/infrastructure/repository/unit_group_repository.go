package repository

import (
	"context"
	gosql "database/sql"
	"slices"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/patch"
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

func mapUnitGroupBaseForwardRow(row sqlc.ListUnitGroupsForwardBaseRow) *domain.UnitGroupFull {
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
		AccountID: accountID,
		BaseUnit:  domain.LightUnit{ID: row.BaseUnitID},
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func mapUnitGroupBaseBackwardRow(row sqlc.ListUnitGroupsBackwardBaseRow) *domain.UnitGroupFull {
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
		AccountID: accountID,
		BaseUnit:  domain.LightUnit{ID: row.BaseUnitID},
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func mapUnitGroupBaseRow(row sqlc.GetUnitGroupBaseRow) *domain.UnitGroupFull {
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
		AccountID: accountID,
		BaseUnit:  domain.LightUnit{ID: row.BaseUnitID},
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func mapUnitGroupUnitBaseRow(row sqlc.ListUnitGroupUnitsBaseRow) *domain.UnitGroupUnit {
	return &domain.UnitGroupUnit{
		ID:                 row.ID,
		UnitID:             row.UnitID,
		UnitGroupID:        row.UnitGroupID,
		DiscountPercentage: row.DiscountPercentage,
		DiscountFixed:      row.DiscountFixed,
		IsVisible:          row.IsVisible,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}
}

func mapUnitGroupUnitBaseGetRow(row sqlc.GetUnitGroupUnitBaseRow) *domain.UnitGroupUnit {
	return &domain.UnitGroupUnit{
		ID:                 row.ID,
		UnitID:             row.UnitID,
		UnitGroupID:        row.UnitGroupID,
		DiscountPercentage: row.DiscountPercentage,
		DiscountFixed:      row.DiscountFixed,
		IsVisible:          row.IsVisible,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}
}

func mapGetUnitsByIDsRowToLightUnit(row sqlc.GetUnitsByIDsRow) domain.LightUnit {
	var acctID *string
	if row.AccountID.Valid {
		acctID = &row.AccountID.String
	}
	return domain.LightUnit{
		ID:                row.ID,
		Name:              row.Name,
		Abbreviation:      row.Abbreviation,
		Type:              row.UnitDimensionCode,
		RatioNumerator:    row.RatioNumerator,
		RatioDenominator:  row.RatioDenominator,
		OffsetNumerator:   row.OffsetNumerator,
		OffsetDenominator: row.OffsetDenominator,
		IsBaseUnit:        row.IsBaseUnit,
		AccountID:         acctID,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}

// stitchBaseUnits fetches the base_unit for each UnitGroupFull and populates it in place.
// Mappers must store the base_unit_id in BaseUnit.ID before calling this.
func (r *unitGroupRepoImpl) stitchBaseUnits(ctx context.Context, unitGroups []*domain.UnitGroupFull) *apierror.APIError {
	if len(unitGroups) == 0 {
		return nil
	}

	idSet := make(map[string]bool)
	baseUnitIDs := make([]string, 0, len(unitGroups))
	for _, ug := range unitGroups {
		if !idSet[ug.BaseUnit.ID] {
			idSet[ug.BaseUnit.ID] = true
			baseUnitIDs = append(baseUnitIDs, ug.BaseUnit.ID)
		}
	}

	rows, err := r.queries.GetUnitsByIDs(ctx, baseUnitIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}

	unitByID := make(map[string]domain.LightUnit, len(rows))
	for _, row := range rows {
		unitByID[row.ID] = mapGetUnitsByIDsRowToLightUnit(row)
	}

	for _, ug := range unitGroups {
		if u, ok := unitByID[ug.BaseUnit.ID]; ok {
			ug.BaseUnit = u
		}
	}

	return nil
}

// stitchUnitGroupUnits fetches unit conversions for each UnitGroupFull.
// When includes contains "associated_units", it also stitches unit details.
func (r *unitGroupRepoImpl) stitchUnitGroupUnits(ctx context.Context, unitGroups []*domain.UnitGroupFull, incs []string) *apierror.APIError {
	if len(unitGroups) == 0 {
		return nil
	}

	for _, ug := range unitGroups {
		rows, err := r.queries.ListUnitGroupUnitsBase(ctx, ug.ID)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return apiErr
		}
		units := make([]*domain.UnitGroupUnit, len(rows))
		for i, row := range rows {
			units[i] = mapUnitGroupUnitBaseRow(row)
		}
		ug.UnitConversions = units
	}

	// If the caller requested unit details on each UnitGroupUnit, stitch them in.
	if slices.Contains(incs, "associated_units") {
		for _, ug := range unitGroups {
			if apiErr := r.stitchUnitDetails(ctx, ug.UnitConversions); apiErr != nil {
				return apiErr
			}
		}
	}

	return nil
}

// stitchUnitDetails fetches unit metadata for a slice of UnitGroupUnit entries.
func (r *unitGroupRepoImpl) stitchUnitDetails(ctx context.Context, units []*domain.UnitGroupUnit) *apierror.APIError {
	if len(units) == 0 {
		return nil
	}

	idSet := make(map[string]bool)
	unitIDs := make([]string, 0, len(units))
	for _, u := range units {
		if !idSet[u.UnitID] {
			idSet[u.UnitID] = true
			unitIDs = append(unitIDs, u.UnitID)
		}
	}

	rows, err := r.queries.GetUnitsByIDs(ctx, unitIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}

	unitByID := make(map[string]domain.LightUnit, len(rows))
	for _, row := range rows {
		unitByID[row.ID] = mapGetUnitsByIDsRowToLightUnit(row)
	}

	for _, u := range units {
		if lu, ok := unitByID[u.UnitID]; ok {
			u.Unit = lu
		}
	}

	return nil
}

// applyUnitGroupStitches populates optional sub-resources based on includes.
func (r *unitGroupRepoImpl) applyUnitGroupStitches(ctx context.Context, unitGroups []*domain.UnitGroupFull, incs []string) *apierror.APIError {
	if len(unitGroups) == 0 {
		return nil
	}

	// base_unit is always fetched (it's a required sub-resource on UnitGroupFull)
	if apiErr := r.stitchBaseUnits(ctx, unitGroups); apiErr != nil {
		return apiErr
	}

	// associated_units: always fetched for structural completeness, unit details conditional
	return r.stitchUnitGroupUnits(ctx, unitGroups, incs)
}

func buildUnitGroupSearchParams(query *string) gosql.NullString {
	if query == nil || *query == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: "%" + db.EscapeLike(*query) + "%", Valid: true}
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
			rows, err := r.queries.ListUnitGroupsBackwardBase(ctx, sqlc.ListUnitGroupsBackwardBaseParams{
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
				unitGroups[i] = mapUnitGroupBaseBackwardRow(row)
			}
			if apiErr := r.applyUnitGroupStitches(ctx, unitGroups, params.Includes); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			result, pageInfo := pagination.BuildPageString(unitGroups, params.Limit, cursorDir, unitGroupCreatedAt, unitGroupID)
			return &domain.ListUnitGroupsResult{UnitGroups: result, PageInfo: pageInfo}, nil
		}

		rows, err := r.queries.ListUnitGroupsForwardBase(ctx, sqlc.ListUnitGroupsForwardBaseParams{
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
			unitGroups[i] = mapUnitGroupBaseForwardRow(row)
		}
		if apiErr := r.applyUnitGroupStitches(ctx, unitGroups, params.Includes); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		result, pageInfo := pagination.BuildPageString(unitGroups, params.Limit, cursorDir, unitGroupCreatedAt, unitGroupID)
		return &domain.ListUnitGroupsResult{UnitGroups: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListUnitGroupsForwardBase(ctx, sqlc.ListUnitGroupsForwardBaseParams{
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
		unitGroups[i] = mapUnitGroupBaseForwardRow(row)
	}
	if apiErr := r.applyUnitGroupStitches(ctx, unitGroups, params.Includes); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	result, pageInfo := pagination.BuildPageString(unitGroups, params.Limit, cursorDir, unitGroupCreatedAt, unitGroupID)
	return &domain.ListUnitGroupsResult{UnitGroups: result, PageInfo: pageInfo}, nil
}

func (r *unitGroupRepoImpl) Get(ctx context.Context, params domain.GetUnitGroupParams) (*domain.UnitGroupFull, *apierror.APIError) {
	ctx, span := unitGroupRepoTracer.Start(ctx, "repository.unit_group.get")
	defer span.End()

	row, err := r.queries.GetUnitGroupBase(ctx, sqlc.GetUnitGroupBaseParams{
		ID:        params.UnitGroupID,
		AccountID: gosql.NullString{String: params.AccountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	ug := mapUnitGroupBaseRow(row)

	if apiErr := r.applyUnitGroupStitches(ctx, []*domain.UnitGroupFull{ug}, params.Includes); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

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

	return r.Get(ctx, domain.GetUnitGroupParams{
		AccountID:   params.AccountID,
		UnitGroupID: id,
		Includes:    params.Includes,
	})
}

func (r *unitGroupRepoImpl) Update(ctx context.Context, params domain.UpdateUnitGroupParams) (*domain.UnitGroupFull, *apierror.APIError) {
	ctx, span := unitGroupRepoTracer.Start(ctx, "repository.unit_group.update")
	defer span.End()

	updateNotes := params.Notes.WasProvided()
	result, err := r.queries.UpdateUnitGroup(ctx, sqlc.UpdateUnitGroupParams{
		ID:          params.UnitGroupID,
		AccountID:   gosql.NullString{String: params.AccountID, Valid: true},
		Name:        toNullString(params.Name),
		UpdateNotes: updateNotes,
		Notes:       patch.StringToNullString(params.Notes),
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

	return r.Get(ctx, domain.GetUnitGroupParams{
		AccountID:   params.AccountID,
		UnitGroupID: params.UnitGroupID,
		Includes:    params.Includes,
	})
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

	row, err := r.queries.GetUnitGroupUnitBase(ctx, sqlc.GetUnitGroupUnitBaseParams{
		ID:          id,
		UnitGroupID: params.UnitGroupID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	u := mapUnitGroupUnitBaseGetRow(row)

	if slices.Contains(params.Includes, "unit") {
		if apiErr := r.stitchUnitDetails(ctx, []*domain.UnitGroupUnit{u}); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	return u, nil
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

func (r *unitGroupRepoImpl) ListUnits(ctx context.Context, unitGroupID string, incs []string) ([]*domain.UnitGroupUnit, *apierror.APIError) {
	ctx, span := unitGroupRepoTracer.Start(ctx, "repository.unit_group.list_units")
	defer span.End()

	rows, err := r.queries.ListUnitGroupUnitsBase(ctx, unitGroupID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	units := make([]*domain.UnitGroupUnit, len(rows))
	for i, row := range rows {
		units[i] = mapUnitGroupUnitBaseRow(row)
	}

	if slices.Contains(incs, "unit") {
		if apiErr := r.stitchUnitDetails(ctx, units); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	return units, nil
}

func (r *unitGroupRepoImpl) GetUnit(ctx context.Context, params domain.GetUnitGroupUnitParams) (*domain.UnitGroupUnit, *apierror.APIError) {
	ctx, span := unitGroupRepoTracer.Start(ctx, "repository.unit_group.get_unit")
	defer span.End()

	row, err := r.queries.GetUnitGroupUnitBase(ctx, sqlc.GetUnitGroupUnitBaseParams{
		ID:          params.UnitGroupUnitID,
		UnitGroupID: params.UnitGroupID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	u := mapUnitGroupUnitBaseGetRow(row)

	if slices.Contains(params.Includes, "unit") {
		if apiErr := r.stitchUnitDetails(ctx, []*domain.UnitGroupUnit{u}); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	return u, nil
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
