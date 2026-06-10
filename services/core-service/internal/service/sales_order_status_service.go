package service

import (
	"context"
	"fmt"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var salesOrderStatusSvcTracer = tracing.GetTracer("core-service.sales_order_status_service")

type salesOrderStatusSvcImpl struct {
	repos domain.RepoFactory
}

type SalesOrderStatusSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory
}

func (c *SalesOrderStatusSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("sales order status service: repos is required")
	}
	return nil
}

func NewSalesOrderStatusSvc(config *SalesOrderStatusSvcConfig) domain.SalesOrderStatusSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &salesOrderStatusSvcImpl{
		repos: config.Repos,
	}
}

func (s *salesOrderStatusSvcImpl) ListSalesOrderStatuses(ctx context.Context, params domain.ListSalesOrderStatusesParams) (*domain.ListSalesOrderStatusesResult, *apierror.APIError) {
	ctx, span := salesOrderStatusSvcTracer.Start(ctx, "service.sales_order_status.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAuthenticated(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewSalesOrderStatusRepo().List(ctx, params)
}

func (s *salesOrderStatusSvcImpl) BatchGetSalesOrderStatusesByIDs(ctx context.Context, ids []string) ([]*domain.SalesOrderStatus, *apierror.APIError) {
	ctx, span := salesOrderStatusSvcTracer.Start(ctx, "service.sales_order_status.batch_get_by_ids")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsAuthenticated(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if len(ids) == 0 {
		return nil, nil
	}
	return s.repos.NewSalesOrderStatusRepo().GetByIDs(ctx, ids)
}
