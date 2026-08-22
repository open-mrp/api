package repository

import (
	"context"

	"github.com/open-mrp/api/services/platform-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/tracing"
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

func (r *cleanupRepoImpl) DeleteExpiredDeletedRecords(ctx context.Context, limit int) (int64, error) {
	ctx, span := tracing.StartSpan(ctx, cleanupRepoTracer, "repository.cleanup.delete_expired_deleted_records")
	defer span.End()

	result, err := r.queries.DeleteExpiredDeletedRecords(ctx, int32(limit)) // #nosec G115 - small config value
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

func (r *cleanupRepoImpl) DeleteExpiredRequestLogs(ctx context.Context, limit int) (int64, error) {
	ctx, span := tracing.StartSpan(ctx, cleanupRepoTracer, "repository.cleanup.delete_expired_request_logs")
	defer span.End()

	result, err := r.queries.DeleteExpiredRequestLogs(ctx, int32(limit)) // #nosec G115 - small config value
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

func (r *cleanupRepoImpl) DeleteExpiredAuditEvents(ctx context.Context, limit int) (int64, error) {
	ctx, span := tracing.StartSpan(ctx, cleanupRepoTracer, "repository.cleanup.delete_expired_audit_events")
	defer span.End()

	result, err := r.queries.DeleteExpiredAuditEvents(ctx, int32(limit)) // #nosec G115 - small config value
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
