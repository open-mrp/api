package mediator

import (
	"context"
	"fmt"

	"github.com/augno/api/services/core-service/internal/domain"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var readAccessMedTracer = tracing.GetTracer("core-service.read_access_mediator")

type readAccessMedImpl struct {
	repos domain.RepoFactory
}

type ReadAccessMedConfig struct {
	// Repos (required) is the repository factory for access checks.
	Repos domain.RepoFactory
}

func (c *ReadAccessMedConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("read access mediator: repos is required")
	}
	return nil
}

func NewReadAccessMed(config *ReadAccessMedConfig) domain.ReadAccessMed {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &readAccessMedImpl{
		repos: config.Repos,
	}
}

// CheckReadAccess verifies that the actor account has owner-side read access to the target account. Same-account access is always allowed. Cross-account access requires an account_relation row in the actor→target direction (the actor is the owner of the relation). Use this for endpoints that expose the owner's view of a counterparty account (e.g. a merchant reading data scoped to one of their customers).
//
//  1. Allow access when the actor and target accounts are the same.
//  2. Require an actor→target account relation; otherwise return an authorization error.
func (m *readAccessMedImpl) CheckReadAccess(ctx context.Context, actorAccountID, targetAccountID string) *apierror.APIError {
	ctx, span := readAccessMedTracer.Start(ctx, "mediator.read_access.check")
	defer span.End()

	if actorAccountID == targetAccountID {
		return nil
	}

	hasRelation, apiErr := m.repos.NewAccountRelationRepo().HasRelation(ctx, actorAccountID, targetAccountID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if !hasRelation {
		return tracing.Trace(span, apierror.NewAuthorizationError("You cannot access this account."))
	}

	return nil
}

// CheckCounterpartyReadAccess verifies access in either direction. It is intended for customer/supplier portal endpoints where the counterparty (e.g. a customer) reads data on the owner's account (e.g. a vendor), and the account_relation row is stored owner→counterparty. Only use this on endpoints that explicitly scope returned data to the counterparty — otherwise it leaks cross-tenant data.
//
//  1. Allow access when the actor and target accounts are the same.
//  2. Check for an account relation in the actor→target direction.
//  3. Fall back to checking the target→actor direction.
//  4. Return an authorization error when no relation exists in either direction.
func (m *readAccessMedImpl) CheckCounterpartyReadAccess(ctx context.Context, actorAccountID, targetAccountID string) *apierror.APIError {
	ctx, span := readAccessMedTracer.Start(ctx, "mediator.read_access.check_counterparty")
	defer span.End()

	if actorAccountID == targetAccountID {
		return nil
	}

	repo := m.repos.NewAccountRelationRepo()

	hasRelation, apiErr := repo.HasRelation(ctx, actorAccountID, targetAccountID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if !hasRelation {
		hasRelation, apiErr = repo.HasRelation(ctx, targetAccountID, actorAccountID)
		if apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	if !hasRelation {
		return tracing.Trace(span, apierror.NewAuthorizationError("You cannot access this account."))
	}

	return nil
}
