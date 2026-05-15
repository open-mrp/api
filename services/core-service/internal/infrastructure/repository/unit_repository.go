package repository

import (
	"context"
	gosql "database/sql"
	"strings"
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
	if s == nil || *s == "" {
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

	ft := db.NewFulltextSearch(params.Query)
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
				SearchQuery:            ft.Fulltext,
				LikeQuery:              ft.Like,
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
			SearchQuery:            ft.Fulltext,
			LikeQuery:              ft.Like,
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
		SearchQuery:            ft.Fulltext,
		LikeQuery:              ft.Like,
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

func (r *unitRepoImpl) GetCurrencyBaseUnitID(ctx context.Context) (string, *apierror.APIError) {
	ctx, span := unitRepoTracer.Start(ctx, "repository.unit.get_currency_base_unit_id")
	defer span.End()

	id, err := r.queries.GetCurrencyBaseUnit(ctx)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}
	return id, nil
}

func (r *unitRepoImpl) GetDimensionCodes(ctx context.Context, ids []string) (map[string]string, *apierror.APIError) {
	ctx, span := unitRepoTracer.Start(ctx, "repository.unit.get_dimension_codes")
	defer span.End()

	if len(ids) == 0 {
		return map[string]string{}, nil
	}

	rows, err := r.queries.GetUnitDimensionCodes(ctx, ids)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make(map[string]string, len(rows))
	for _, row := range rows {
		out[row.ID] = row.UnitDimensionCode
	}
	return out, nil
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

func (r *unitRepoImpl) FindByAbbreviations(ctx context.Context, accountID string, abbreviations []string) ([]*domain.Unit, *apierror.APIError) {
	ctx, span := unitRepoTracer.Start(ctx, "repository.unit.find_by_abbreviations")
	defer span.End()

	// NOTE: sqlc generation for FindUnitsByAbbreviations currently only accepts accountID.
	// Filter abbreviations in-memory for now.
	rows, err := r.queries.FindUnitsByAbbreviations(ctx, gosql.NullString{String: accountID, Valid: true})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	abbrSet := make(map[string]struct{}, len(abbreviations))
	for _, a := range abbreviations {
		abbrSet[strings.ToLower(a)] = struct{}{}
	}

	units := make([]*domain.Unit, len(rows))
	out := units[:0]
	for _, row := range rows {
		var accID *string
		if row.AccountID.Valid {
			accID = &row.AccountID.String
		}
		if _, ok := abbrSet[strings.ToLower(row.Abbreviation)]; !ok {
			continue
		}
		out = append(out, &domain.Unit{
			ID:                row.ID,
			Name:              row.Name,
			Abbreviation:      row.Abbreviation,
			UnitDimensionCode: row.UnitDimensionCode,
			RatioNumerator:    row.RatioNumerator,
			RatioDenominator:  row.RatioDenominator,
			OffsetNumerator:   row.OffsetNumerator,
			OffsetDenominator: row.OffsetDenominator,
			IsBaseUnit:        row.IsBaseUnit,
			AccountID:         accID,
			CreatedAt:         row.CreatedAt,
			UpdatedAt:         row.UpdatedAt,
		})
	}
	return out, nil
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
