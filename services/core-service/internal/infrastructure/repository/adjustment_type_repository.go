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

var adjustmentTypeRepoTracer = tracing.GetTracer("core-service.adjustment_type_repository")

type adjustmentTypeRepoImpl struct {
	queries *sqlc.Queries
}

func NewAdjustmentTypeRepo(queries *sqlc.Queries) domain.AdjustmentTypeRepo {
	return &adjustmentTypeRepoImpl{queries: queries}
}

func adjustmentTypeCreatedAt(at *domain.AdjustmentType) time.Time { return at.CreatedAt }
func adjustmentTypeID(at *domain.AdjustmentType) string           { return at.ID }

func buildAdjustmentTypeSearchParams(query *string) gosql.NullString {
	if query == nil || *query == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: "%" + *query + "%", Valid: true}
}

func mapAdjustmentTypeRow(row sqlc.AdjustmentType) *domain.AdjustmentType {
	return &domain.AdjustmentType{
		ID:        row.ID,
		Name:      row.Name,
		Code:      row.Code,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func (r *adjustmentTypeRepoImpl) List(ctx context.Context, params domain.ListAdjustmentTypesParams) (*domain.ListAdjustmentTypesResult, *apierror.APIError) {
	ctx, span := adjustmentTypeRepoTracer.Start(ctx, "repository.adjustment_type.list")
	defer span.End()

	searchQuery := buildAdjustmentTypeSearchParams(params.Query)

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListAdjustmentTypesBackward(ctx, sqlc.ListAdjustmentTypesBackwardParams{
				SearchQuery:     searchQuery,
				CursorCreatedAt: cur.OccurredAt,
				CursorID:        cur.ID,
				Limit:           params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			adjustmentTypes := make([]*domain.AdjustmentType, len(rows))
			for i, row := range rows {
				adjustmentTypes[i] = mapAdjustmentTypeRow(row)
			}
			result, pageInfo := pagination.BuildPageString(adjustmentTypes, params.Limit, cursorDir, adjustmentTypeCreatedAt, adjustmentTypeID)
			return &domain.ListAdjustmentTypesResult{AdjustmentTypes: result, PageInfo: pageInfo}, nil
		}

		// Forward
		rows, err := r.queries.ListAdjustmentTypesForward(ctx, sqlc.ListAdjustmentTypesForwardParams{
			SearchQuery:     searchQuery,
			CursorCreatedAt: gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:        gosql.NullString{String: cur.ID, Valid: true},
			Limit:           params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		adjustmentTypes := make([]*domain.AdjustmentType, len(rows))
		for i, row := range rows {
			adjustmentTypes[i] = mapAdjustmentTypeRow(row)
		}
		result, pageInfo := pagination.BuildPageString(adjustmentTypes, params.Limit, cursorDir, adjustmentTypeCreatedAt, adjustmentTypeID)
		return &domain.ListAdjustmentTypesResult{AdjustmentTypes: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListAdjustmentTypesForward(ctx, sqlc.ListAdjustmentTypesForwardParams{
		SearchQuery: searchQuery,
		Limit:       params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	adjustmentTypes := make([]*domain.AdjustmentType, len(rows))
	for i, row := range rows {
		adjustmentTypes[i] = mapAdjustmentTypeRow(row)
	}
	result, pageInfo := pagination.BuildPageString(adjustmentTypes, params.Limit, cursorDir, adjustmentTypeCreatedAt, adjustmentTypeID)
	return &domain.ListAdjustmentTypesResult{AdjustmentTypes: result, PageInfo: pageInfo}, nil
}
