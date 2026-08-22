package repository

import (
	"context"

	"github.com/open-mrp/api/services/billing-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/tracing"
)

var stripeEventLogRepoTracer = tracing.GetTracer("billing-service.stripe_event_log_repository")

type StripeEventLogRepo struct {
	queries *sqlc.Queries
}

func NewStripeEventLogRepo(queries *sqlc.Queries) *StripeEventLogRepo {
	return &StripeEventLogRepo{queries: queries}
}

func (r *StripeEventLogRepo) Exists(ctx context.Context, eventID, objectID string) (bool, *apierror.APIError) {
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

func (r *StripeEventLogRepo) Insert(ctx context.Context, eventID, eventType, objectID string) *apierror.APIError {
	ctx, span := stripeEventLogRepoTracer.Start(ctx, "repository.stripe_event_log.insert")
	defer span.End()

	recordID, genErr := id.GenID(id.StripeEventLogIDPrefix, nil)
	if genErr != nil {
		return tracing.Trace(span, apierror.NewInternalError(genErr, "Failed to generate stripe event log ID."))
	}

	insertErr := r.queries.InsertStripeEventLog(ctx, sqlc.InsertStripeEventLogParams{
		ID:        recordID,
		EventID:   eventID,
		EventType: eventType,
		ObjectID:  objectID,
	})
	if apiErr := db.MapSQLError(insertErr); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}
