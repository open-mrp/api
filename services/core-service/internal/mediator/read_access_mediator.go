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
