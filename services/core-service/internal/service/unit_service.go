package service

import (
	"context"
	"fmt"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var unitSvcTracer = tracing.GetTracer("core-service.unit_service")

type unitSvcImpl struct {
	repos domain.RepoFactory
}

type UnitSvcConfig struct {
	Repos domain.RepoFactory
}

func (c *UnitSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("unit service: repos is required")
	}
	return nil
}

func NewUnitSvc(config *UnitSvcConfig) domain.UnitSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &unitSvcImpl{
		repos: config.Repos,
	}
}

func (s *unitSvcImpl) ListUnits(ctx context.Context, params domain.ListUnitsParams) (*domain.ListUnitsResult, *apierror.APIError) {
	ctx, span := unitSvcTracer.Start(ctx, "service.unit.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := types.CheckIsInternalActor(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := types.CheckHasPermission(identity, types.PermissionDomainUnits, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if identity.TargetAccountID == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Target account ID is required."))
	}

	params.AccountID = *identity.TargetAccountID

	return s.repos.NewUnitRepo().List(ctx, params)
}
