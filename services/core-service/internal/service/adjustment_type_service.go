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

var adjustmentTypeSvcTracer = tracing.GetTracer("core-service.adjustment_type_service")

type adjustmentTypeSvcImpl struct {
	repos domain.RepoFactory
}

type AdjustmentTypeSvcConfig struct {
	Repos domain.RepoFactory
}

func (c *AdjustmentTypeSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("adjustment type service: repos is required")
	}
	return nil
}

func NewAdjustmentTypeSvc(config *AdjustmentTypeSvcConfig) domain.AdjustmentTypeSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &adjustmentTypeSvcImpl{
		repos: config.Repos,
	}
}

func (s *adjustmentTypeSvcImpl) ListAdjustmentTypes(ctx context.Context, params domain.ListAdjustmentTypesParams) (*domain.ListAdjustmentTypesResult, *apierror.APIError) {
	ctx, span := adjustmentTypeSvcTracer.Start(ctx, "service.adjustment_type.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkAdjustmentTypeReadPermission(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewAdjustmentTypeRepo().List(ctx, params)
}

func (s *adjustmentTypeSvcImpl) BatchGetAdjustmentTypesByIDs(ctx context.Context, ids []string) ([]*domain.AdjustmentType, *apierror.APIError) {
	ctx, span := adjustmentTypeSvcTracer.Start(ctx, "service.adjustment_type.batch_get_by_ids")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainAdjustmentTypes, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if len(ids) == 0 {
		return nil, nil
	}
	return s.repos.NewAdjustmentTypeRepo().GetByIDs(ctx, ids)
}

// checkAdjustmentTypeReadPermission checks the appropriate read permission.
// Adjustment types are system-wide data; internal actors need adjustment_types:read.
// Non-internal actors (customer/supplier) do not carry role-based permissions and are
// permitted to read global lookup data.
func checkAdjustmentTypeReadPermission(identity *types.Identity) *apierror.APIError {
	if !identity.IsInternalActor() {
		return nil
	}
	return identity.CheckHasPermission(types.PermissionDomainAdjustmentTypes, types.ActionRead)
}
