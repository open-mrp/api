package repository

import (
	"context"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var quantityRepoTracer = tracing.GetTracer("core-service.quantity_repository")

type quantityRepoImpl struct {
	queries *sqlc.Queries
}

func NewQuantityRepo(queries *sqlc.Queries) domain.QuantityRepo {
	return &quantityRepoImpl{queries: queries}
}

func (r *quantityRepoImpl) Get(ctx context.Context, id string) (*domain.Quantity, *apierror.APIError) {
	ctx, span := quantityRepoTracer.Start(ctx, "repository.quantity.get")
	defer span.End()

	row, err := r.queries.GetQuantityWithUnit(ctx, id)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return &domain.Quantity{
		ID:               row.ID,
		Value:            row.Value,
		UnitID:           row.UnitID,
		UnitName:         row.UnitName,
		UnitAbbreviation: row.UnitAbbreviation,
		UnitType:         row.UnitType,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}, nil
}

func (r *quantityRepoImpl) Update(ctx context.Context, params domain.UpdateQuantityParams) (*domain.Quantity, *apierror.APIError) {
	ctx, span := quantityRepoTracer.Start(ctx, "repository.quantity.update")
	defer span.End()

	result, err := r.queries.UpdateQuantityByID(ctx, sqlc.UpdateQuantityByIDParams{
		Value:  toNullString(params.Value),
		UnitID: toNullString(params.UnitID),
		ID:     params.QuantityID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Quantity not found."))
	}

	return r.Get(ctx, params.QuantityID)
}
