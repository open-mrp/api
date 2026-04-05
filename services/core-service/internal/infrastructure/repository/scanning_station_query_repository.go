package repository

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var scanningStationQueryRepoTracer = tracing.GetTracer("core-service.scanning_station_query_repository")

type scanningStationQueryRepoImpl struct {
	queries *sqlc.Queries
}

func NewScanningStationQueryRepo(queries *sqlc.Queries) domain.ScanningStationQueryRepo {
	return &scanningStationQueryRepoImpl{queries: queries}
}

func (r *scanningStationQueryRepoImpl) IsInAccount(ctx context.Context, accountID, id string) (bool, *apierror.APIError) {
	ctx, span := scanningStationQueryRepoTracer.Start(ctx, "repository.scanning_station_query.is_in_account")
	defer span.End()

	count, err := r.queries.IsScanningStationInAccount(ctx, sqlc.IsScanningStationInAccountParams{
		ID:        id,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return count > 0, nil
}

func (r *scanningStationQueryRepoImpl) FindType(ctx context.Context, accountID, id string) (string, *apierror.APIError) {
	ctx, span := scanningStationQueryRepoTracer.Start(ctx, "repository.scanning_station_query.find_type")
	defer span.End()

	stationType, err := r.queries.GetScanningStationType(ctx, sqlc.GetScanningStationTypeParams{
		ID:        id,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	return stationType, nil
}
