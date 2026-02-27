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
