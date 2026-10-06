package repository

import (
	"context"
	"database/sql"
	"strconv"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/tracing"
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

func (r *productionRunQueryRepoImpl) Reopen(ctx context.Context, accountID, id string) *apierror.APIError {
	ctx, span := productionRunQueryRepoTracer.Start(ctx, "repository.production_run_query.reopen")
	defer span.End()

	err := r.queries.ReopenProductionRun(ctx, sqlc.ReopenProductionRunParams{
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

	// Shares the counter with the other production-run repository: two allocators on one series would
	// race each other, so both go through numberCounter.
	sysPropertyID, apiErr := id.GenID(id.SysPropertyIDPrefix, nil)
	if apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}
	number, apiErr := productionRunNumbers(r.queries, accountID).next(ctx, sysPropertyID)
	if apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}
	return strconv.FormatInt(number, 10), nil
}
