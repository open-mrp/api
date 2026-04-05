package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var productionRunQueryRepoTracer = tracing.GetTracer("core-service.production_run_query_repository")

type productionRunQueryRepoImpl struct {
	queries *sqlc.Queries
}

func NewProductionRunQueryRepo(queries *sqlc.Queries) domain.ProductionRunQueryRepo {
	return &productionRunQueryRepoImpl{queries: queries}
}

func (r *productionRunQueryRepoImpl) Start(ctx context.Context, accountID, id string) *apierror.APIError {
	ctx, span := productionRunQueryRepoTracer.Start(ctx, "repository.production_run_query.start")
	defer span.End()

	err := r.queries.StartProductionRun(ctx, sqlc.StartProductionRunParams{
		ID:        id,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *productionRunQueryRepoImpl) CloseIfAllBatchesScannedOrDeleted(ctx context.Context, accountID, id string) *apierror.APIError {
	ctx, span := productionRunQueryRepoTracer.Start(ctx, "repository.production_run_query.close_if_all_batches_scanned_or_deleted")
	defer span.End()

	count, err := r.queries.CountUnscannedOrUndeletedBatchesByRun(ctx, sqlc.CountUnscannedOrUndeletedBatchesByRunParams{
		ProductionRunID: sql.NullString{String: id, Valid: true},
		AccountID:       accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if count > 0 {
		return nil
	}

	err = r.queries.CloseProductionRun(ctx, sqlc.CloseProductionRunParams{
		ID:        id,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *productionRunQueryRepoImpl) Create(ctx context.Context, id, responsibleUserID, number, accountID string) *apierror.APIError {
	ctx, span := productionRunQueryRepoTracer.Start(ctx, "repository.production_run_query.create")
	defer span.End()

	err := r.queries.CreateProductionRun(ctx, sqlc.CreateProductionRunParams{
		ID:                id,
		ResponsibleUserID: responsibleUserID,
		Number:            number,
		AccountID:         accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *productionRunQueryRepoImpl) GetNextNumber(ctx context.Context, accountID string) (string, *apierror.APIError) {
	ctx, span := productionRunQueryRepoTracer.Start(ctx, "repository.production_run_query.get_next_number")
	defer span.End()

	nextNum, err := r.queries.GetNextProductionRunNumber(ctx, accountID)
	if err != nil {
		return "", tracing.Trace(span, db.MapSQLError(err))
	}

	return fmt.Sprintf("%d", nextNum), nil
}
