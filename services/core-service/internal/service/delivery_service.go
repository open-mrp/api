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

var deliverySvcTracer = tracing.GetTracer("core-service.delivery_service")

type deliverySvcImpl struct {
	repos domain.RepoFactory
}

type DeliverySvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory
}

func (c *DeliverySvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("delivery service: repos is required")
	}
	return nil
}

func NewDeliverySvc(config *DeliverySvcConfig) domain.DeliverySvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &deliverySvcImpl{
		repos: config.Repos,
	}
}

func (s *deliverySvcImpl) ListDeliveries(ctx context.Context, params domain.ListDeliveriesParams) (*domain.ListDeliveriesResult, *apierror.APIError) {
	ctx, span := deliverySvcTracer.Start(ctx, "service.delivery.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainDeliveries, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	repo := s.repos.NewDeliveryRepo()
	result, apiErr := repo.List(ctx, params)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Expand lines per delivery only when requested (so the list can serve the
	// lines.item array filter). Get returns the full delivery with its lines.
	for _, include := range params.Includes {
		if include == "lines" {
			for _, summary := range result.Deliveries {
				full, apiErr := repo.Get(ctx, domain.GetDeliveryParams{AccountID: params.AccountID, DeliveryID: summary.ID})
				if apiErr != nil {
					return nil, tracing.Trace(span, apiErr)
				}
				summary.Lines = full.Lines
			}
			break
		}
	}

	return result, nil
}

func (s *deliverySvcImpl) GetDelivery(ctx context.Context, params domain.GetDeliveryParams) (*domain.Delivery, *apierror.APIError) {
	ctx, span := deliverySvcTracer.Start(ctx, "service.delivery.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainDeliveries, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewDeliveryRepo().Get(ctx, params)
}
