package repository

import (
	"context"

	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/services/auth-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var registrationQueueRepoTracer = tracing.GetTracer("auth-service.registration_queue_repository")

type registrationQueueRepoImpl struct {
	db *sqlc.Queries
}

func NewRegistrationQueueRepo(db *sqlc.Queries) domain.RegistrationQueueRepo {
	return &registrationQueueRepoImpl{db: db}
}

func (r *registrationQueueRepoImpl) Create(ctx context.Context, email, name, planCode, registrationSessionID string) (bool, *apierror.APIError) {
	ctx, span := registrationQueueRepoTracer.Start(ctx, "repository.registration_queue.create")
	defer span.End()

	rowsAffected, err := r.db.CreateRegistrationQueueEntry(ctx, sqlc.CreateRegistrationQueueEntryParams{
		Email:                 email,
		Name:                  name,
		PlanCode:              planCode,
		RegistrationSessionID: registrationSessionID,
	})

	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return rowsAffected > 0, nil
}
