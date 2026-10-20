package mediator

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var editAccessMedTracer = tracing.GetTracer("core-service.edit_access_mediator")

type editAccessMedImpl struct {
	repos domain.RepoFactory
}

type EditAccessMedConfig struct {
	// Repos (required) is the repository factory for access checks.
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

// CheckEditAccess verifies that the actor account has edit access to the target account. Same-account access is always allowed. Cross-account access requires: the target is not self-managed, a relation exists between the accounts, and the target has no other owner relations.
//
//  1. Allow access when the actor and target accounts are the same.
//  2. Reject when the target signed up to OpenMRP on its own.
//  3. Require a relation between the actor and target accounts.
//  4. Reject when the target has owner relations with other accounts.
func (m *editAccessMedImpl) CheckEditAccess(ctx context.Context, actorAccountID, targetAccountID string) *apierror.APIError {
	ctx, span := editAccessMedTracer.Start(ctx, "mediator.edit_access.check")
	defer span.End()

	// Same-account access is always allowed.
	if actorAccountID == targetAccountID {
		return nil
	}

	// A self-managed account is altered only by its own users.
	selfManaged, apiErr := m.repos.NewAccountRepo().IsSelfManaged(ctx, targetAccountID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if selfManaged {
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
