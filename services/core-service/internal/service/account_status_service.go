package service

import (
	"context"
	"fmt"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var accountStatusSvcTracer = tracing.GetTracer("core-service.account_status_service")

type accountStatusSvcImpl struct {
	repos domain.RepoFactory
}

type AccountStatusSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory
}

func (c *AccountStatusSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("account status service: repos is required")
	}
	return nil
}

func NewAccountStatusSvc(config *AccountStatusSvcConfig) domain.AccountStatusSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &accountStatusSvcImpl{
		repos: config.Repos,
	}
}

func (s *accountStatusSvcImpl) ListAccountStatuses(ctx context.Context, params domain.ListAccountStatusesParams) (*domain.ListAccountStatusesResult, *apierror.APIError) {
	ctx, span := accountStatusSvcTracer.Start(ctx, "service.account_status.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewAccountStatusRepo().List(ctx, params)
}

func (s *accountStatusSvcImpl) GetAccountStatus(ctx context.Context, identifier string) (*domain.AccountStatus, *apierror.APIError) {
	ctx, span := accountStatusSvcTracer.Start(ctx, "service.account_status.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewAccountStatusRepo().Get(ctx, identifier)
}

func (s *accountStatusSvcImpl) BatchGetAccountStatusesByIDs(ctx context.Context, ids []string) ([]*domain.AccountStatus, *apierror.APIError) {
	ctx, span := accountStatusSvcTracer.Start(ctx, "service.account_status.batch_get_by_ids")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if len(ids) == 0 {
		return nil, nil
	}
	return s.repos.NewAccountStatusRepo().GetByIDs(ctx, ids)
}
