package repository

import (
	"context"
	gosql "database/sql"
	"errors"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/safeconv"
	"github.com/open-mrp/api/shared/tracing"
)

var carrierTransitEstimateRepoTracer = tracing.GetTracer("core-service.carrier_transit_estimate_repository")

type carrierTransitEstimateRepoImpl struct {
	queries *sqlc.Queries
}

func NewCarrierTransitEstimateRepo(queries *sqlc.Queries) domain.CarrierTransitEstimateRepo {
	return &carrierTransitEstimateRepoImpl{queries: queries}
}

func (r *carrierTransitEstimateRepoImpl) Resolve(ctx context.Context, accountID string, lane domain.TransitLane) (*domain.CarrierTransitCandidates, *apierror.APIError) {
	ctx, span := carrierTransitEstimateRepoTracer.Start(ctx, "repository.carrier_transit_estimate.resolve")
	defer span.End()

	row, err := r.queries.ResolveCarrierTransit(ctx, sqlc.ResolveCarrierTransitParams{
		AccountID:       accountID,
		OriginCountry:   lane.OriginCountry,
		OriginPostal:    lane.OriginPostal,
		DestCountry:     lane.DestCountry,
		DestPostal:      lane.DestPostal,
		CarrierOptionID: lane.CarrierOptionID,
		OptionAccountID: gosql.NullString{String: accountID, Valid: true},
	})
	if err != nil {
		// The service level is the join root, so no row means it does not exist or belongs to another account. Transit is simply unknown, which the caller already handles.
		if errors.Is(err, gosql.ErrNoRows) {
			return nil, nil
		}
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to resolve carrier transit."))
	}

	candidates := &domain.CarrierTransitCandidates{}
	if row.LaneTransitDays.Valid {
		days := int(row.LaneTransitDays.Int32)
		candidates.LaneDays = &days
	}
	if row.LaneSourceCode.Valid {
		candidates.LaneSourceCode = row.LaneSourceCode.String
	}
	if row.LaneRefreshedAt.Valid {
		refreshed := row.LaneRefreshedAt.Time
		candidates.LaneRefreshedAt = &refreshed
	}
	if row.DefaultTransitDays.Valid {
		days := int(row.DefaultTransitDays.Int32)
		candidates.ServiceLevelDefaultDays = &days
	}
	return candidates, nil
}

func (r *carrierTransitEstimateRepoImpl) Upsert(ctx context.Context, params domain.UpsertTransitEstimateParams) *apierror.APIError {
	ctx, span := carrierTransitEstimateRepoTracer.Start(ctx, "repository.carrier_transit_estimate.upsert")
	defer span.End()

	err := r.queries.UpsertCarrierTransitEstimate(ctx, sqlc.UpsertCarrierTransitEstimateParams{
		ID:              params.ID,
		AccountID:       params.AccountID,
		CarrierOptionID: params.Lane.CarrierOptionID,
		OriginCountry:   params.Lane.OriginCountry,
		OriginPostal:    params.Lane.OriginPostal,
		DestCountry:     params.Lane.DestCountry,
		DestPostal:      params.Lane.DestPostal,
		TransitDays:     safeconv.IntToInt32(params.TransitDays),
		SourceCode:      params.SourceCode,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}
