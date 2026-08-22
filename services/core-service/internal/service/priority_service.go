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

var prioritySvcTracer = tracing.GetTracer("core-service.priority_service")

type prioritySvcImpl struct {
	repos domain.RepoFactory
}

type PrioritySvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory
}

func (c *PrioritySvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("priority service: repos is required")
	}
	return nil
}

func NewPrioritySvc(config *PrioritySvcConfig) domain.PrioritySvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &prioritySvcImpl{
		repos: config.Repos,
	}
}

// ListPriorities returns a paginated list of priorities.
//
// 1. Extract and validate the caller's identity and actor type.
// 2. Check priorities:read permission.
// 3. Query the priority repository with pagination params.
func (s *prioritySvcImpl) ListPriorities(ctx context.Context, params domain.ListPrioritiesParams) (*domain.ListPrioritiesResult, *apierror.APIError) {
	ctx, span := prioritySvcTracer.Start(ctx, "service.priority.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainPriorities, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewPriorityRepo().List(ctx, params)
}

// GetPriority retrieves a single priority by ID.
//
// 1. Extract and validate the caller's identity and actor type.
// 2. Check priorities:read permission.
// 3. Fetch the priority from the repository by ID or code.
func (s *prioritySvcImpl) GetPriority(ctx context.Context, identifier string) (*domain.Priority, *apierror.APIError) {
	ctx, span := prioritySvcTracer.Start(ctx, "service.priority.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainPriorities, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewPriorityRepo().Get(ctx, identifier)
}

// BatchGetPrioritiesByIDs returns priorities by ID for the api-gateway include resolver. Priorities are system-wide so there's no per-caller scoping; we still require the caller to be an authenticated internal actor with the priorities:read permission.
func (s *prioritySvcImpl) BatchGetPrioritiesByIDs(ctx context.Context, ids []string) ([]*domain.Priority, *apierror.APIError) {
	ctx, span := prioritySvcTracer.Start(ctx, "service.priority.batch_get_by_ids")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainPriorities, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if len(ids) == 0 {
		return nil, nil
	}
	return s.repos.NewPriorityRepo().GetByIDs(ctx, ids)
}
