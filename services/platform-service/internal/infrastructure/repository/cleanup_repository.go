package repository

import (
	"context"

	"github.com/augno/api/services/platform-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"
)

var cleanupRepoTracer = tracing.GetTracer("platform-service.cleanup_repository")

type cleanupRepoImpl struct {
	queries *sqlc.Queries
}

// NewCleanupRepo creates a new cleanup repository for the idempotency key cleanup worker.
func NewCleanupRepo(queries *sqlc.Queries) messaging.CleanupRepo {
	return &cleanupRepoImpl{queries: queries}
}

func (r *cleanupRepoImpl) DeleteExpiredIdempotencyKeys(ctx context.Context, limit int) (int64, error) {
	ctx, span := tracing.StartSpan(ctx, cleanupRepoTracer, "repository.cleanup.delete_expired_idempotency_keys")
	defer span.End()

	result, err := r.queries.DeleteExpiredIdempotencyKeys(ctx, int32(limit)) // #nosec G115 - small config value
	if err != nil {
		span.RecordError(err)
		return 0, err
	}

	count, err := result.RowsAffected()
	if err != nil {
		span.RecordError(err)
		return 0, err
	}

	return count, nil
}

func (r *cleanupRepoImpl) DeleteExpiredServiceIdempotencyKeys(ctx context.Context, limit int) (int64, error) {
	ctx, span := tracing.StartSpan(ctx, cleanupRepoTracer, "repository.cleanup.delete_expired_service_idempotency_keys")
	defer span.End()

	result, err := r.queries.DeleteExpiredServiceIdempotencyKeys(ctx, int32(limit)) // #nosec G115 - small config value
	if err != nil {
		span.RecordError(err)
		return 0, err
	}

	count, err := result.RowsAffected()
	if err != nil {
		span.RecordError(err)
		return 0, err
	}

	return count, nil
}
