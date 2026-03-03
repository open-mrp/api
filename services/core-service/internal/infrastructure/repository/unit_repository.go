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

var unitRepoTracer = tracing.GetTracer("core-service.unit_repository")

var unitDuplicateKeyMapping = db.DuplicateKeyMapping{
	"unit_account_id_name_key": func() *apierror.APIError {
		return apierror.NewConflictErrorWithParam("A unit with this name already exists.", "name")
	},
	"unit_account_id_abbreviation_key": func() *apierror.APIError {
		return apierror.NewConflictErrorWithParam("A unit with this abbreviation already exists.", "abbreviation")
	},
}

type unitRepoImpl struct {
	queries *sqlc.Queries
}

func NewUnitRepo(queries *sqlc.Queries) domain.UnitRepo {
	return &unitRepoImpl{queries: queries}
}

func unitCreatedAt(u *domain.Unit) time.Time { return u.CreatedAt }
func unitID(u *domain.Unit) string           { return u.ID }

func buildSearchParams(query *string) (gosql.NullString, gosql.NullString) {
	if query == nil || *query == "" {
		return gosql.NullString{}, gosql.NullString{}
	}
	return gosql.NullString{String: "%" + *query + "%", Valid: true},
		gosql.NullString{String: *query, Valid: true}
}

func buildDimensionFilter(t *string) gosql.NullString {
	if t == nil || *t == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: *t, Valid: true}
}

func mapForwardRow(row sqlc.ListUnitsForwardRow) *domain.Unit {
	var accountID *string
	if row.AccountID.Valid {
		accountID = &row.AccountID.String
	}
	return &domain.Unit{
		ID:                row.ID,
		Name:              row.Name,
		Abbreviation:      row.Abbreviation,
		UnitDimensionCode: row.UnitDimensionCode,
		RatioNumerator:    row.RatioNumerator,
		RatioDenominator:  row.RatioDenominator,
		OffsetNumerator:   row.OffsetNumerator,
		OffsetDenominator: row.OffsetDenominator,
		IsBaseUnit:        row.IsBaseUnit,
		AccountID:         accountID,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}

func mapGetUnitRow(row sqlc.GetUnitRow) *domain.Unit {
	var accountID *string
	if row.AccountID.Valid {
		accountID = &row.AccountID.String
	}
	return &domain.Unit{
		ID:                row.ID,
		Name:              row.Name,
		Abbreviation:      row.Abbreviation,
		UnitDimensionCode: row.UnitDimensionCode,
		RatioNumerator:    row.RatioNumerator,
		RatioDenominator:  row.RatioDenominator,
		OffsetNumerator:   row.OffsetNumerator,
		OffsetDenominator: row.OffsetDenominator,
		IsBaseUnit:        row.IsBaseUnit,
		AccountID:         accountID,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}

func toNullString(s *string) gosql.NullString {
	if s == nil {
		return gosql.NullString{}
	}
	return gosql.NullString{String: *s, Valid: true}
}

func mapBackwardRow(row sqlc.ListUnitsBackwardRow) *domain.Unit {
	var accountID *string
	if row.AccountID.Valid {
		accountID = &row.AccountID.String
	}
	return &domain.Unit{
		ID:                row.ID,
		Name:              row.Name,
		Abbreviation:      row.Abbreviation,
		UnitDimensionCode: row.UnitDimensionCode,
		RatioNumerator:    row.RatioNumerator,
		RatioDenominator:  row.RatioDenominator,
		OffsetNumerator:   row.OffsetNumerator,
		OffsetDenominator: row.OffsetDenominator,
		IsBaseUnit:        row.IsBaseUnit,
		AccountID:         accountID,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}

func (r *unitRepoImpl) List(ctx context.Context, params domain.ListUnitsParams) (*domain.ListUnitsResult, *apierror.APIError) {
	ctx, span := unitRepoTracer.Start(ctx, "repository.unit.list")
	defer span.End()

	searchQuery, searchExact := buildSearchParams(params.Query)
	dimensionFilter := buildDimensionFilter(params.Type)
	includeGroupFilter := len(params.UnitGroupIDs) > 0
	accountID := gosql.NullString{String: params.AccountID, Valid: true}

	// Ensure unitGroupIDs is never nil for the slice param
	unitGroupIDs := params.UnitGroupIDs
	if unitGroupIDs == nil {
		unitGroupIDs = []string{}
	}

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListUnitsBackward(ctx, sqlc.ListUnitsBackwardParams{
				AccountID:              accountID,
				UnitDimensionCode:      dimensionFilter,
				IncludeUnitGroupFilter: includeGroupFilter,
				UnitGroupIds:           unitGroupIDs,
				SearchQuery:            searchQuery,
				SearchExact:            searchExact,
				CursorCreatedAt:        cur.OccurredAt,
				CursorID:               cur.ID,
				Limit:                  params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			units := make([]*domain.Unit, len(rows))
			for i, row := range rows {
				units[i] = mapBackwardRow(row)
			}
			result, pageInfo := pagination.BuildPageString(units, params.Limit, cursorDir, unitCreatedAt, unitID)
			return &domain.ListUnitsResult{Units: result, PageInfo: pageInfo}, nil
		}

		// Forward
		rows, err := r.queries.ListUnitsForward(ctx, sqlc.ListUnitsForwardParams{
			AccountID:              accountID,
			UnitDimensionCode:      dimensionFilter,
			IncludeUnitGroupFilter: includeGroupFilter,
			UnitGroupIds:           unitGroupIDs,
			SearchQuery:            searchQuery,
			SearchExact:            searchExact,
			CursorCreatedAt:        gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:               gosql.NullString{String: cur.ID, Valid: true},
			Limit:                  params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		units := make([]*domain.Unit, len(rows))
		for i, row := range rows {
			units[i] = mapForwardRow(row)
		}
		result, pageInfo := pagination.BuildPageString(units, params.Limit, cursorDir, unitCreatedAt, unitID)
		return &domain.ListUnitsResult{Units: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListUnitsForward(ctx, sqlc.ListUnitsForwardParams{
		AccountID:              accountID,
		UnitDimensionCode:      dimensionFilter,
		IncludeUnitGroupFilter: includeGroupFilter,
		UnitGroupIds:           unitGroupIDs,
		SearchQuery:            searchQuery,
		SearchExact:            searchExact,
		Limit:                  params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	units := make([]*domain.Unit, len(rows))
	for i, row := range rows {
		units[i] = mapForwardRow(row)
	}
	result, pageInfo := pagination.BuildPageString(units, params.Limit, cursorDir, unitCreatedAt, unitID)
	return &domain.ListUnitsResult{Units: result, PageInfo: pageInfo}, nil
}

func (r *unitRepoImpl) Get(ctx context.Context, params domain.GetUnitParams) (*domain.Unit, *apierror.APIError) {
	ctx, span := unitRepoTracer.Start(ctx, "repository.unit.get")
	defer span.End()

	row, err := r.queries.GetUnit(ctx, sqlc.GetUnitParams{
		ID:        params.UnitID,
		AccountID: gosql.NullString{String: params.AccountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapGetUnitRow(row), nil
}

func (r *unitRepoImpl) Create(ctx context.Context, id string, params domain.CreateUnitParams) (*domain.Unit, *apierror.APIError) {
	ctx, span := unitRepoTracer.Start(ctx, "repository.unit.create")
	defer span.End()

	err := r.queries.InsertUnit(ctx, sqlc.InsertUnitParams{
		ID:                id,
		Name:              params.Name,
		Abbreviation:      params.Abbreviation,
		UnitDimensionCode: params.UnitDimensionCode,
		RatioNumerator:    params.RatioNumerator,
		RatioDenominator:  params.RatioDenominator,
		OffsetNumerator:   params.OffsetNumerator,
		OffsetDenominator: params.OffsetDenominator,
		IsBaseUnit:        params.IsBaseUnit,
		AccountID:         gosql.NullString{String: params.AccountID, Valid: true},
	})
	if apiErr := db.MapSQLErrorWithDuplicateKeys(err, unitDuplicateKeyMapping); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.Get(ctx, domain.GetUnitParams{AccountID: params.AccountID, UnitID: id})
}

func (r *unitRepoImpl) Update(ctx context.Context, params domain.UpdateUnitParams) (*domain.Unit, *apierror.APIError) {
	ctx, span := unitRepoTracer.Start(ctx, "repository.unit.update")
	defer span.End()

	result, err := r.queries.UpdateUnit(ctx, sqlc.UpdateUnitParams{
		ID:                params.UnitID,
		AccountID:         gosql.NullString{String: params.AccountID, Valid: true},
		Name:              toNullString(params.Name),
		Abbreviation:      toNullString(params.Abbreviation),
		RatioNumerator:    toNullString(params.RatioNumerator),
		RatioDenominator:  toNullString(params.RatioDenominator),
		OffsetNumerator:   toNullString(params.OffsetNumerator),
		OffsetDenominator: toNullString(params.OffsetDenominator),
	})
	if apiErr := db.MapSQLErrorWithDuplicateKeys(err, unitDuplicateKeyMapping); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Unit not found."))
	}

	return r.Get(ctx, domain.GetUnitParams{AccountID: params.AccountID, UnitID: params.UnitID})
}

func (r *unitRepoImpl) ExistsByName(ctx context.Context, accountID, name string, excludeID *string) (bool, *apierror.APIError) {
	ctx, span := unitRepoTracer.Start(ctx, "repository.unit.exists_by_name")
	defer span.End()

	count, err := r.queries.CountUnitsByName(ctx, sqlc.CountUnitsByNameParams{
		Name:      name,
		AccountID: gosql.NullString{String: accountID, Valid: true},
		ExcludeID: toNullString(excludeID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return count > 0, nil
}

func (r *unitRepoImpl) ExistsByAbbreviation(ctx context.Context, accountID, abbreviation string, excludeID *string) (bool, *apierror.APIError) {
	ctx, span := unitRepoTracer.Start(ctx, "repository.unit.exists_by_abbreviation")
	defer span.End()

	count, err := r.queries.CountUnitsByAbbreviation(ctx, sqlc.CountUnitsByAbbreviationParams{
		Abbreviation: abbreviation,
		AccountID:    gosql.NullString{String: accountID, Valid: true},
		ExcludeID:    toNullString(excludeID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return count > 0, nil
}

func (r *unitRepoImpl) Delete(ctx context.Context, params domain.DeleteUnitParams) *apierror.APIError {
	ctx, span := unitRepoTracer.Start(ctx, "repository.unit.delete")
	defer span.End()

	if err := r.queries.DeleteUnitGroupUnitsByUnitID(ctx, params.UnitID); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	result, err := r.queries.DeleteUnit(ctx, sqlc.DeleteUnitParams{
		ID:        params.UnitID,
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
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Unit not found."))
	}

	return nil
}
