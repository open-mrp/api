package repository

import (
	"context"
	"database/sql"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var unitQueryRepoTracer = tracing.GetTracer("core-service.unit_query_repository")

type unitQueryRepoImpl struct {
	queries *sqlc.Queries
}

func NewUnitQueryRepo(queries *sqlc.Queries) domain.UnitQueryRepo {
	return &unitQueryRepoImpl{queries: queries}
}

func (r *unitQueryRepoImpl) Find(ctx context.Context, accountID, id string) (*domain.LightUnit, *apierror.APIError) {
	ctx, span := unitQueryRepoTracer.Start(ctx, "repository.unit_query.find")
	defer span.End()

	row, err := r.queries.GetUnit(ctx, sqlc.GetUnitParams{
		ID:        id,
		AccountID: sql.NullString{String: accountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return &domain.LightUnit{
		ID:           row.ID,
		Abbreviation: row.Abbreviation,
		Type:         row.UnitDimensionCode,
	}, nil
}
