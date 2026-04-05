package repository

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var unitGroupQueryRepoTracer = tracing.GetTracer("core-service.unit_group_query_repository")

type unitGroupQueryRepoImpl struct {
	queries *sqlc.Queries
}

func NewUnitGroupQueryRepo(queries *sqlc.Queries) domain.UnitGroupQueryRepo {
	return &unitGroupQueryRepoImpl{queries: queries}
}

func (r *unitGroupQueryRepoImpl) FindByItem(ctx context.Context, accountID, itemID string) (*domain.UnitGroup, *apierror.APIError) {
	ctx, span := unitGroupQueryRepoTracer.Start(ctx, "repository.unit_group_query.find_by_item")
	defer span.End()

	row, err := r.queries.FindUnitGroupByItem(ctx, sqlc.FindUnitGroupByItemParams{
		ItemID:    itemID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return &domain.UnitGroup{
		ID: row.ID,
		BaseUnit: domain.LightUnit{
			ID:           row.BaseUnitID,
			Abbreviation: row.BaseUnitAbbreviation,
			Type:         row.BaseUnitType,
		},
	}, nil
}
