package service

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var permissionGroupSvcTracer = tracing.GetTracer("core-service.permission_group_service")

type permissionGroupSvcImpl struct {
	repos domain.RepoFactory
}

type PermissionGroupSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory
}

func (c *PermissionGroupSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("permission group service: repos is required")
	}
	return nil
}

func NewPermissionGroupSvc(config *PermissionGroupSvcConfig) domain.PermissionGroupSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &permissionGroupSvcImpl{
		repos: config.Repos,
	}
}

func (s *permissionGroupSvcImpl) BatchGetPermissionGroupsByIDs(ctx context.Context, ids []string) ([]*domain.PermissionGroup, *apierror.APIError) {
	ctx, span := permissionGroupSvcTracer.Start(ctx, "service.permission_group.batch_get_by_ids")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainPermissions, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewPermissionGroupRepo().GetByIDs(ctx, ids)
}

func (s *permissionGroupSvcImpl) ListPermissionGroups(ctx context.Context, params domain.ListPermissionGroupsParams) (*domain.ListPermissionGroupsResult, *apierror.APIError) {
	ctx, span := permissionGroupSvcTracer.Start(ctx, "service.permission_group.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainPermissions, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewPermissionGroupRepo().List(ctx, params)
}
