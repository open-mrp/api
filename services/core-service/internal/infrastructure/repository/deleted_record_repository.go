package repository

import (
	"context"
	"encoding/json"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var deletedRecordRepoTracer = tracing.GetTracer("core-service.deleted_record_repository")

type deletedRecordRepoImpl struct {
	queries *sqlc.Queries
}

func NewDeletedRecordRepo(queries *sqlc.Queries) domain.DeletedRecordRepo {
	return &deletedRecordRepoImpl{queries: queries}
}

func (r *deletedRecordRepoImpl) Create(ctx context.Context, resourceType constants.DeletedRecordResourceType, resourceID string, data any) *apierror.APIError {
	ctx, span := deletedRecordRepoTracer.Start(ctx, "repository.deleted_record.create")
	defer span.End()

	serializedData, err := json.Marshal(data)
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to serialize deleted record data."))
	}

	if apiErr := db.MapSQLError(r.queries.InsertDeletedRecord(ctx, sqlc.InsertDeletedRecordParams{
		ResourceType: string(resourceType),
		ResourceID:   resourceID,
		Data:         serializedData,
	})); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *deletedRecordRepoImpl) Exists(ctx context.Context, resourceType constants.DeletedRecordResourceType, resourceID string) (bool, *apierror.APIError) {
	ctx, span := deletedRecordRepoTracer.Start(ctx, "repository.deleted_record.exists")
	defer span.End()

	count, err := r.queries.CountDeletedRecordsByResourceAndResourceID(ctx, sqlc.CountDeletedRecordsByResourceAndResourceIDParams{
		ResourceType: string(resourceType),
		ResourceID:   resourceID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return count > 0, nil
}
