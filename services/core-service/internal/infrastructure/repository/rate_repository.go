package repository

import (
	"context"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var rateRepoTracer = tracing.GetTracer("core-service.rate_repository")

type rateRepoImpl struct {
	queries *sqlc.Queries
}

func NewRateRepo(queries *sqlc.Queries) domain.RateRepo {
	return &rateRepoImpl{queries: queries}
}

func (r *rateRepoImpl) Get(ctx context.Context, id string) (*domain.Rate, *apierror.APIError) {
	ctx, span := rateRepoTracer.Start(ctx, "repository.rate.get")
	defer span.End()

	row, err := r.queries.GetRateWithUnits(ctx, id)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return &domain.Rate{
		ID:                          row.ID,
		Value:                       row.Value,
		NumeratorUnitID:             row.NumeratorUnitID,
		NumeratorUnitName:           row.NumeratorUnitName,
		NumeratorUnitAbbreviation:   row.NumeratorUnitAbbreviation,
		NumeratorUnitType:           row.NumeratorUnitType,
		DenominatorUnitID:           row.DenominatorUnitID,
		DenominatorUnitName:         row.DenominatorUnitName,
		DenominatorUnitAbbreviation: row.DenominatorUnitAbbreviation,
		DenominatorUnitType:         row.DenominatorUnitType,
		CreatedAt:                   row.CreatedAt,
		UpdatedAt:                   row.UpdatedAt,
	}, nil
}

func (r *rateRepoImpl) Update(ctx context.Context, params domain.UpdateRateParams) (*domain.Rate, *apierror.APIError) {
	ctx, span := rateRepoTracer.Start(ctx, "repository.rate.update")
	defer span.End()

	result, err := r.queries.UpdateRateByID(ctx, sqlc.UpdateRateByIDParams{
		Value:             toNullString(params.Value),
		NumeratorUnitID:   toNullString(params.NumeratorUnitID),
		DenominatorUnitID: toNullString(params.DenominatorUnitID),
		ID:                params.RateID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Rate not found."))
	}

	return r.Get(ctx, params.RateID)
}
