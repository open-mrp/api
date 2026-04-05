package mediator

import (
	"context"
	"fmt"

	"github.com/augno/api/services/core-service/internal/domain"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var editAccessMedTracer = tracing.GetTracer("core-service.edit_access_mediator")

type editAccessMedImpl struct {
	repos domain.RepoFactory
}

type EditAccessMedConfig struct {
	Repos domain.RepoFactory
}

func (c *EditAccessMedConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("edit access mediator: repos is required")
	}
	return nil
}

func NewEditAccessMed(config *EditAccessMedConfig) domain.EditAccessMed {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &editAccessMedImpl{
		repos: config.Repos,
	}
}

func (m *editAccessMedImpl) CheckEditAccess(ctx context.Context, actorAccountID, targetAccountID string) *apierror.APIError {
	ctx, span := editAccessMedTracer.Start(ctx, "mediator.edit_access.check")
	defer span.End()

	// Same-account access is always allowed.
	if actorAccountID == targetAccountID {
		return nil
	}

	// Target must not have an active billing plan.
	hasPlan, apiErr := m.repos.NewAccountRepo().HasActiveBillingPlan(ctx, targetAccountID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if hasPlan {
		return tracing.Trace(span, apierror.NewAuthorizationError("You cannot alter this account directly."))
	}

	// A relation must exist between the actor and target.
	relationRepo := m.repos.NewAccountRelationRepo()
	hasRelation, apiErr := relationRepo.HasRelation(ctx, actorAccountID, targetAccountID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if !hasRelation {
		return tracing.Trace(span, apierror.NewAuthorizationError("You cannot access this account."))
	}

	// Target must not have relations with other owners.
	otherCount, apiErr := relationRepo.CountOtherOwnerRelations(ctx, targetAccountID, actorAccountID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if otherCount > 0 {
		return tracing.Trace(span, apierror.NewAuthorizationError("You cannot alter this account directly."))
	}

	return nil
}
