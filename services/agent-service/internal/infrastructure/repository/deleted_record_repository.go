package repository

import (
	"context"
	"encoding/json"

	"github.com/open-mrp/api/services/agent-service/internal/domain"
	"github.com/open-mrp/api/services/agent-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var deletedRecordRepoTracer = tracing.GetTracer("agent-service.deleted_record_repository")

type deletedRecordRepoImpl struct {
	queries *sqlc.Queries
}

func NewDeletedRecordRepo(queries *sqlc.Queries) domain.DeletedRecordRepo {
	return &deletedRecordRepoImpl{queries: queries}
}

// CreateInAccount records the snapshot with the owning account added under account_id, which ExistsInAccount matches.
func (r *deletedRecordRepoImpl) CreateInAccount(ctx context.Context, resourceType constants.DeletedRecordResourceType, resourceID, accountID string, data any) *apierror.APIError {
	ctx, span := deletedRecordRepoTracer.Start(ctx, "repository.deleted_record.create_in_account")
	defer span.End()

	serialized, err := json.Marshal(data)
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to serialize deleted record data."))
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(serialized, &fields); err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "A deleted record's snapshot must be an object."))
	}
	owner, err := json.Marshal(accountID)
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to serialize deleted record owner."))
	}
	fields["account_id"] = owner
	serializedData, err := json.Marshal(fields)
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

func (r *deletedRecordRepoImpl) ExistsInAccount(ctx context.Context, resourceType constants.DeletedRecordResourceType, resourceID, accountID string) (bool, *apierror.APIError) {
	ctx, span := deletedRecordRepoTracer.Start(ctx, "repository.deleted_record.exists_in_account")
	defer span.End()

	count, err := r.queries.CountDeletedRecordsInAccount(ctx, sqlc.CountDeletedRecordsInAccountParams{
		ResourceType: string(resourceType),
		ResourceID:   resourceID,
		AccountID:    accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return count > 0, nil
}
