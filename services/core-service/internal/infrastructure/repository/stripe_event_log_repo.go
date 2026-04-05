package repository

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var stripeEventLogRepoTracer = tracing.GetTracer("core-service.infrastructure.repository.stripe_event_log")

type stripeEventLogRepoImpl struct {
	queries *sqlc.Queries
}

func NewStripeEventLogRepo(queries *sqlc.Queries) domain.StripeEventLogRepo {
	return &stripeEventLogRepoImpl{queries: queries}
}

func (r *stripeEventLogRepoImpl) Exists(ctx context.Context, eventID, objectID string) (bool, *apierror.APIError) {
	ctx, span := stripeEventLogRepoTracer.Start(ctx, "repository.stripe_event_log.exists")
	defer span.End()

	exists, err := r.queries.StripeEventLogExists(ctx, sqlc.StripeEventLogExistsParams{
		EventID:  eventID,
		ObjectID: objectID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return exists, nil
}

func (r *stripeEventLogRepoImpl) Create(ctx context.Context, id, eventID, objectID, eventType string) *apierror.APIError {
	ctx, span := stripeEventLogRepoTracer.Start(ctx, "repository.stripe_event_log.create")
	defer span.End()

	err := r.queries.InsertStripeEventLog(ctx, sqlc.InsertStripeEventLogParams{
		ID:        id,
		EventID:   eventID,
		ObjectID:  objectID,
		EventType: eventType,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}
