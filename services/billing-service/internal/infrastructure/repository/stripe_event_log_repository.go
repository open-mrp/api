package repository

import (
	"context"

	"github.com/augno/api/services/billing-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/tracing"
)

var stripeEventLogRepoTracer = tracing.GetTracer("billing-service.stripe_event_log_repository")

type StripeEventLogRepo struct {
	queries *sqlc.Queries
}

func NewStripeEventLogRepo(queries *sqlc.Queries) *StripeEventLogRepo {
	return &StripeEventLogRepo{queries: queries}
}

func (r *StripeEventLogRepo) Exists(ctx context.Context, eventID, objectID string) (bool, error) {
	ctx, span := stripeEventLogRepoTracer.Start(ctx, "repository.stripe_event_log.exists")
	defer span.End()

	exists, err := r.queries.StripeEventLogExists(ctx, sqlc.StripeEventLogExistsParams{
		EventID:  eventID,
		ObjectID: objectID,
	})
	if err != nil {
		span.RecordError(err)
		return false, err
	}

	return exists, nil
}

func (r *StripeEventLogRepo) Insert(ctx context.Context, eventID, eventType, objectID string) error {
	ctx, span := stripeEventLogRepoTracer.Start(ctx, "repository.stripe_event_log.insert")
	defer span.End()

	recordID, genErr := id.GenID(id.StripeEventLogIDPrefix, nil)
	if genErr != nil {
		span.RecordError(genErr)
		return genErr
	}

	insertErr := r.queries.InsertStripeEventLog(ctx, sqlc.InsertStripeEventLogParams{
		ID:        recordID,
		EventID:   eventID,
		EventType: eventType,
		ObjectID:  objectID,
	})
	if insertErr != nil {
		span.RecordError(insertErr)
		return insertErr
	}

	return nil
}
