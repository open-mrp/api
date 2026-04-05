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

var analyticsSvcTracer = tracing.GetTracer("core-service.service.analytics")

type analyticsSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
}

type AnalyticsSvcConfig struct {
	Repos           domain.RepoFactory
	MediatorFactory domain.MediatorFactory
}

func (c *AnalyticsSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("analytics service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("analytics service: mediator factory is required")
	}
	return nil
}

func NewAnalyticsSvc(config *AnalyticsSvcConfig) domain.AnalyticsSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &analyticsSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
	}
}

func (s *analyticsSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *analyticsSvcImpl) AnalyzeSales(ctx context.Context, params domain.AnalyzeSalesParams) ([]domain.SalesEntry, *apierror.APIError) {
	ctx, span := analyticsSvcTracer.Start(ctx, "service.analytics.analyze_sales")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainInvoices, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID
	isSalesRep := identity.IsSalesRep()

	// If the user is a sales rep, restrict to only their own sales data.
	if isSalesRep && identity.Actor != nil && identity.Actor.ID != "" {
		accountUser, apiErr := s.repos.NewAccountUserRepo().FindByAccountAndUserID(ctx, identity.Actor.ID, params.AccountID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if accountUser != nil {
			params.SalesRepIDs = []string{accountUser.ID}
		}
	}

	entries, apiErr := s.repos.NewAnalyticsRepo().GetSalesEntries(ctx, params)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Sales reps should not see cost data.
	if isSalesRep {
		for i := range entries {
			entries[i].UnitCost = 0
		}
	}

	return entries, nil
}

func (s *analyticsSvcImpl) AnalyzeOpenBatches(ctx context.Context, params domain.AnalyzeOpenBatchesParams) ([]domain.OpenBatchEntry, *apierror.APIError) {
	ctx, span := analyticsSvcTracer.Start(ctx, "service.analytics.analyze_open_batches")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainBatches, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewAnalyticsRepo().GetOpenBatchEntries(ctx, params)
}

func (s *analyticsSvcImpl) AnalyzeProductionCosts(ctx context.Context, params domain.AnalyzeProductionCostsParams) ([]domain.ProductionCostEntry, *apierror.APIError) {
	ctx, span := analyticsSvcTracer.Start(ctx, "service.analytics.analyze_production_costs")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainBatches, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewAnalyticsRepo().GetProductionCostEntries(ctx, params)
}

func (s *analyticsSvcImpl) AnalyzeDeliveries(ctx context.Context, params domain.AnalyzeDeliveriesParams) (*domain.DeliveryAnalyticsResult, *apierror.APIError) {
	ctx, span := analyticsSvcTracer.Start(ctx, "service.analytics.analyze_deliveries")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainInvoices, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewAnalyticsRepo().GetDeliveryAnalytics(ctx, params)
}

func (s *analyticsSvcImpl) AnalyzeManufacturing(ctx context.Context, params domain.AnalyzeManufacturingParams) (float64, *apierror.APIError) {
	ctx, span := analyticsSvcTracer.Start(ctx, "service.analytics.analyze_manufacturing")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return 0, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainInvoices, types.ActionRead); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewAnalyticsRepo().GetManufacturingMetric(ctx, params)
}

func (s *analyticsSvcImpl) AnalyzeManufacturingBatch(ctx context.Context, params domain.AnalyzeManufacturingBatchParams) (*domain.ManufacturingBatchResult, *apierror.APIError) {
	ctx, span := analyticsSvcTracer.Start(ctx, "service.analytics.analyze_manufacturing_batch")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainInvoices, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewAnalyticsRepo().GetManufacturingBatch(ctx, params)
}

func (s *analyticsSvcImpl) AnalyzeOrders(ctx context.Context, params domain.AnalyzeOrdersParams) ([]domain.OrderEntry, *apierror.APIError) {
	ctx, span := analyticsSvcTracer.Start(ctx, "service.analytics.analyze_orders")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSalesOrders, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	// If the user is a sales rep, restrict results to only their own data.
	if params.IsSalesRep && identity.Actor != nil && identity.Type == types.IdentityActorTypeUser {
		accountUser, apiErr := s.repos.NewAccountUserRepo().FindByAccountAndUserID(ctx, identity.Actor.ID, params.AccountID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if accountUser != nil {
			params.SalesRepIDs = []string{accountUser.ID}
		}
	}

	entries, apiErr := s.repos.NewAnalyticsRepo().GetOrderEntries(ctx, params)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// For sales reps, sanitize cost data.
	if params.IsSalesRep {
		for i := range entries {
			entries[i].UnitCost = 0
		}
	}

	return entries, nil
}

func (s *analyticsSvcImpl) AnalyzeQuarterlyOrders(ctx context.Context, params domain.AnalyzeQuarterlyOrdersParams) ([]domain.YearlyQuarterlyData, *apierror.APIError) {
	ctx, span := analyticsSvcTracer.Start(ctx, "service.analytics.analyze_quarterly_orders")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainInvoices, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewAnalyticsRepo().GetQuarterlyOrders(ctx, params)
}

func (s *analyticsSvcImpl) AnalyzeMaterials(ctx context.Context, params domain.AnalyzeMaterialsParams) ([]domain.MaterialAnalyticsEntry, *apierror.APIError) {
	ctx, span := analyticsSvcTracer.Start(ctx, "service.analytics.analyze_materials")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainMaterials, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewAnalyticsRepo().GetMaterialAnalytics(ctx, params)
}

func (s *analyticsSvcImpl) AnalyzeInventoryReceipts(ctx context.Context, params domain.AnalyzeInventoryReceiptsParams) ([]domain.InventoryReceiptEntry, *apierror.APIError) {
	ctx, span := analyticsSvcTracer.Start(ctx, "service.analytics.analyze_inventory_receipts")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if identity.IsInternalActor() {
		if apiErr := identity.CheckHasPermission(types.PermissionDomainMaterials, types.ActionRead); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		if !identity.IsTargetAccountSet() {
			return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
		}

		if identity.IsExternalTarget() {
			meds := s.mediators()
			if apiErr := meds.ReadAccess.CheckReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
		}

		params.AccountID = identity.Target.AccountID
	} else if identity.IsCustomerUser() {
		actorAccountID := identity.ActorAccountID()
		if actorAccountID == nil {
			return nil, tracing.Trace(span, apierror.NewAuthenticationError("Actor account ID is required."))
		}
		params.AccountID = *actorAccountID
	} else {
		return nil, tracing.Trace(span, apierror.NewValidationError("Invalid actor type."))
	}

	return s.repos.NewAnalyticsRepo().GetInventoryReceiptAnalytics(ctx, params)
}

func (s *analyticsSvcImpl) GetNewCustomersAnalytics(ctx context.Context, params domain.GetNewCustomersAnalyticsParams) ([]domain.NewCustomerEntry, *apierror.APIError) {
	ctx, span := analyticsSvcTracer.Start(ctx, "service.analytics.get_new_customers_analytics")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewAnalyticsRepo().GetNewCustomerEntries(ctx, params)
}

func (s *analyticsSvcImpl) GetDemandForecast(ctx context.Context, params domain.GetDemandForecastParams) (*domain.DemandForecastResult, *apierror.APIError) {
	ctx, span := analyticsSvcTracer.Start(ctx, "service.analytics.get_demand_forecast")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainInvoices, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewAnalyticsRepo().GetDemandForecast(ctx, params)
}

func (s *analyticsSvcImpl) AnalyzeOee(ctx context.Context, params domain.AnalyzeOeeParams) ([]domain.OeeDepartment, *apierror.APIError) {
	ctx, span := analyticsSvcTracer.Start(ctx, "service.analytics.analyze_oee")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainInvoices, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewAnalyticsRepo().GetOeeByDepartment(ctx, params)
}

func (s *analyticsSvcImpl) AnalyzeWeeksOfSales(ctx context.Context, params domain.AnalyzeWeeksOfSalesParams) (*domain.WeeksOfSalesResult, *apierror.APIError) {
	ctx, span := analyticsSvcTracer.Start(ctx, "service.analytics.analyze_weeks_of_sales")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainInventory, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewAnalyticsRepo().GetWeeksOfSales(ctx, params)
}
