package service

import (
	"context"
	"fmt"
	"time"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var analyticsSvcTracer = tracing.GetTracer("core-service.service.analytics")

type analyticsSvcImpl struct {
	repos           domain.RepoFactory
	reportRepos     domain.RepoFactory
	jobSvcFactory   domain.JobSvcFactory
	txManager       TransactionManager
	mediatorFactory domain.MediatorFactory
	cache           *AnalyticsCache
}

type AnalyticsSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// JobSvcFactory (optional; default: nil, exports unavailable) builds the job service export jobs are recorded with.
	JobSvcFactory domain.JobSvcFactory

	// TxManager (optional; default: nil, exports unavailable) runs the transaction an export job is accepted in.
	TxManager TransactionManager

	// ReportRepos (optional; default: Repos) builds the repositories reports read through; point it at a read replica. Access checks and settings still read Repos.
	ReportRepos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// Cache (optional; default: caching disabled) holds computed reports.
	Cache *AnalyticsCache
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
		reportRepos:     config.ReportRepos,
		jobSvcFactory:   config.JobSvcFactory,
		txManager:       config.TxManager,
		mediatorFactory: config.MediatorFactory,
		cache:           config.Cache,
	}
}

var disabledAnalyticsCache = func() *AnalyticsCache {
	c, err := NewAnalyticsCache(nil)
	if err != nil {
		panic(err)
	}
	return c
}()

func (s *analyticsSvcImpl) reportCache() *AnalyticsCache {
	if s.cache == nil {
		return disabledAnalyticsCache
	}
	return s.cache
}

// reports is the repository factory for report reads, which may lag the primary.
func (s *analyticsSvcImpl) reports() domain.RepoFactory {
	if s.reportRepos == nil {
		return s.repos
	}
	return s.reportRepos
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

	entries, apiErr := s.salesEntries(ctx, identity, params)
	return entries, tracing.Trace(span, apiErr)
}

// salesEntries reads the sales report rows the caller's role scopes it to, without checking a permission: the endpoint that asks for them has already decided the caller may see them.
func (s *analyticsSvcImpl) salesEntries(ctx context.Context, identity *types.Identity, params domain.AnalyzeSalesParams) ([]domain.SalesEntry, *apierror.APIError) {
	ctx, span := analyticsSvcTracer.Start(ctx, "service.analytics.sales_entries")
	defer span.End()

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

	entries, apiErr := cachedReport(ctx, s.reportCache().sales, analyticsReport{
		accountID: params.AccountID,
		family:    analyticsFamilySales,
		method:    "sales",
		params:    params,
		ttl:       s.reportCache().ttlForWindow(params.EndDate),
	}, func(ctx context.Context) ([]domain.SalesEntry, *apierror.APIError) {
		return s.reports().NewAnalyticsRepo().GetSalesEntries(ctx, params)
	})
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

	return s.reports().NewAnalyticsRepo().GetOpenBatchEntries(ctx, params)
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

	return s.reports().NewAnalyticsRepo().GetDeliveryAnalytics(ctx, params)
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
	if isCostManufacturingMetric(params.Type) {
		if apiErr := identity.CheckHasPermission(types.PermissionDomainCosts, types.ActionRead); apiErr != nil {
			return 0, tracing.Trace(span, apiErr)
		}
	}

	params.AccountID = identity.Target.AccountID

	return s.reports().NewAnalyticsRepo().GetManufacturingMetric(ctx, params)
}

// isCostManufacturingMetric reports whether the metric is the seller's cost or margin, which only callers holding costs:read may see.
func isCostManufacturingMetric(metric string) bool {
	return metric == "costsPerUnit" || metric == "margin"
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

	return s.reports().NewAnalyticsRepo().GetManufacturingBatch(ctx, params)
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

	entries, apiErr := s.reports().NewAnalyticsRepo().GetOrderEntries(ctx, params)
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
	if identity.IsSalesRep() && identity.Actor != nil && identity.Actor.ID != "" {
		accountUser, apiErr := s.repos.NewAccountUserRepo().FindByAccountAndUserID(ctx, identity.Actor.ID, params.AccountID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if accountUser != nil {
			params.SalesRepIDs = []string{accountUser.ID}
		}
	}
	if params.YearsBack < 0 {
		return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("years_back must be at least 1.", "years_back"))
	}
	if params.YearsBack == 0 {
		params.YearsBack = domain.DefaultQuarterlyOrdersYearsBack
	}
	params.IssuedFrom = quarterlyOrdersIssuedFrom(s.reportCache().cfg.Now(), params.YearsBack)

	return cachedReport(ctx, s.reportCache().quarterlyOrders, analyticsReport{
		accountID: params.AccountID,
		family:    analyticsFamilySales,
		method:    "quarterly_orders",
		params:    params,
		ttl:       s.reportCache().cfg.LiveTTL,
	}, func(ctx context.Context) ([]domain.YearlyQuarterlyData, *apierror.APIError) {
		return s.reports().NewAnalyticsRepo().GetQuarterlyOrders(ctx, params)
	})
}

// quarterlyOrdersIssuedFrom is the start of the earliest of the last yearsBack calendar years, the current one included, in UTC.
func quarterlyOrdersIssuedFrom(now time.Time, yearsBack int32) time.Time {
	return time.Date(now.UTC().Year()-int(yearsBack-1), time.January, 1, 0, 0, 0, 0, time.UTC)
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

	return s.reports().NewAnalyticsRepo().GetMaterialAnalytics(ctx, params)
}

// checkInventoryReceiptAnalyticsReadPermission checks the appropriate read permission based on the target context: internal actors targeting a customer or supplier account need the relationship's read permission rather than the resource domain's.
func checkInventoryReceiptAnalyticsReadPermission(identity *types.Identity) *apierror.APIError {
	if !identity.IsInternalActor() {
		return nil
	}
	if identity.IsTargetCustomerAccount() {
		return identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionRead)
	}
	if identity.IsTargetSupplierAccount() {
		return identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionRead)
	}
	return identity.CheckHasPermission(types.PermissionDomainMaterials, types.ActionRead)
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
		if apiErr := checkInventoryReceiptAnalyticsReadPermission(identity); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		if !identity.IsTargetAccountSet() {
			return nil, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required."))
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

	return s.reports().NewAnalyticsRepo().GetInventoryReceiptAnalytics(ctx, params)
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

	return s.reports().NewAnalyticsRepo().GetNewCustomerEntries(ctx, params)
}

// GetDemandForecast returns per-item demand, revenue and sales history with seasonal-EMA forecasts and confidence bands.
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

	return cachedReport(ctx, s.reportCache().forecast, analyticsReport{
		accountID: params.AccountID,
		family:    analyticsFamilySales,
		method:    "demand_forecast",
		params:    params,
		ttl:       s.reportCache().cfg.LiveTTL,
	}, func(ctx context.Context) (*domain.DemandForecastResult, *apierror.APIError) {
		return s.buildDemandForecast(ctx, params)
	})
}

// AnalyzeOee computes Availability x Performance x Quality per department from planned time, logged downtime and the ideal cycle times the period's output earned.
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
	if apiErr := identity.CheckHasPermission(types.PermissionDomainMachineDowntime, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if params.EndDate.Before(params.StartDate) {
		return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("The period must end on or after it starts.", "ends_at"))
	}

	params.AccountID = identity.Target.AccountID

	return cachedReport(ctx, s.reportCache().oee, analyticsReport{
		accountID: params.AccountID,
		family:    analyticsFamilyProduction,
		method:    "oee",
		params:    params,
		ttl:       s.reportCache().ttlForWindow(params.EndDate),
	}, func(ctx context.Context) ([]domain.OeeDepartment, *apierror.APIError) {
		return s.buildOeeByDepartment(ctx, params)
	})
}

// AnalyzeOeeTrend computes the same OEE terms per production week over a window, rolled up across departments.
func (s *analyticsSvcImpl) AnalyzeOeeTrend(ctx context.Context, params domain.AnalyzeOeeTrendParams) ([]domain.OeeTrendPeriod, *apierror.APIError) {
	ctx, span := analyticsSvcTracer.Start(ctx, "service.analytics.analyze_oee_trend")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainMachineDowntime, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if params.EndDate.Before(params.StartDate) {
		return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("The period must end on or after it starts.", "ends_at"))
	}

	params.AccountID = identity.Target.AccountID

	return cachedReport(ctx, s.reportCache().oeeTrend, analyticsReport{
		accountID: params.AccountID,
		family:    analyticsFamilyProduction,
		method:    "oee_trend",
		params:    params,
		ttl:       s.reportCache().ttlForWindow(params.EndDate),
	}, func(ctx context.Context) ([]domain.OeeTrendPeriod, *apierror.APIError) {
		return s.buildOeeTrend(ctx, params)
	})
}

// AnalyzeWeeksOfSales returns on-hand inventory expressed as weeks of average sales per product line.
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

	return cachedReport(ctx, s.reportCache().weeksOfSales, analyticsReport{
		accountID: params.AccountID,
		family:    analyticsFamilySales,
		method:    "weeks_of_sales",
		params:    params,
		ttl:       s.reportCache().cfg.LiveTTL,
	}, func(ctx context.Context) (*domain.WeeksOfSalesResult, *apierror.APIError) {
		return s.buildWeeksOfSales(ctx, params)
	})
}

// buildWeeksOfSales measures the trailing PeriodInWeeks up to now, so its result drifts with the clock.
func (s *analyticsSvcImpl) buildWeeksOfSales(ctx context.Context, params domain.AnalyzeWeeksOfSalesParams) (*domain.WeeksOfSalesResult, *apierror.APIError) {
	ctx, span := analyticsSvcTracer.Start(ctx, "service.analytics.build_weeks_of_sales")
	defer span.End()

	repo := s.reports().NewAnalyticsRepo()

	// 1. Get sale-type product item IDs and their product line IDs.
	productItems, apiErr := repo.GetSaleProductItemIDs(ctx, params.AccountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if len(productItems) == 0 {
		return &domain.WeeksOfSalesResult{Items: nil, Count: 0}, nil
	}

	// 2. Build maps: productLine -> []itemID, collect unique product line IDs.
	itemsByProductLine := make(map[string][]string)
	uniquePLIDs := make(map[string]bool)
	var allItemIDs []string
	for _, pi := range productItems {
		if pi.ProductLineID != nil {
			plID := *pi.ProductLineID
			itemsByProductLine[plID] = append(itemsByProductLine[plID], pi.ItemID)
			uniquePLIDs[plID] = true
		}
		allItemIDs = append(allItemIDs, pi.ItemID)
	}

	plIDs := make([]string, 0, len(uniquePLIDs))
	for plID := range uniquePLIDs {
		plIDs = append(plIDs, plID)
	}
	if len(plIDs) == 0 {
		return &domain.WeeksOfSalesResult{Items: nil, Count: 0}, nil
	}

	// 3. Get product line info (names).
	plInfoRows, apiErr := repo.GetProductLineInfo(ctx, params.AccountID, plIDs)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// 4. Get on-hand inventory for all items, each in its dimension's base unit.
	inventoryRows, apiErr := repo.GetWeeksOfSalesOnHand(ctx, params.AccountID, allItemIDs)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	invMap := make(map[string]float64, len(inventoryRows))
	for _, row := range inventoryRows {
		invMap[row.ItemID] = row.OnHand
	}

	// 5. For each product line, compute metrics.
	weeks := params.PeriodInWeeks
	if weeks < 1 {
		weeks = 4
	}
	endDate := time.Now()
	startDate := endDate.Add(-time.Duration(weeks) * 7 * 24 * time.Hour)

	orderRows, apiErr := repo.GetOrderQuantitiesByProductLines(ctx, domain.GetOrderQuantitiesByProductLinesParams{
		AccountID:      params.AccountID,
		ProductLineIDs: plIDs,
		StartDate:      startDate,
		EndDate:        endDate,
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	ordersByProductLine := make(map[string]domain.OrderQuantityByProductLineRow, len(orderRows))
	for _, row := range orderRows {
		ordersByProductLine[row.ProductLineID] = row
	}

	var items []domain.WeeksOfSalesItem
	for _, plInfo := range plInfoRows {
		orderRow := ordersByProductLine[plInfo.ID]
		unitAbbrev := orderRow.UnitAbbreviation
		unitType := orderRow.UnitType

		// Stock and sales are both stated in the product line's base unit.
		var onHand float64
		for _, iid := range itemsByProductLine[plInfo.ID] {
			onHand += invMap[iid]
		}
		if orderRow.BaseRatio != 0 {
			onHand /= orderRow.BaseRatio
		}

		avgSales := orderRow.TotalQuantity / float64(weeks)
		var wos float64
		if avgSales > 0 {
			wos = onHand / avgSales
		}

		items = append(items, domain.WeeksOfSalesItem{
			ProductLineID:                        plInfo.ID,
			ProductLineName:                      plInfo.Name,
			QuantityOnHand:                       onHand,
			QuantityOnHandUnitAbbreviation:       unitAbbrev,
			QuantityOnHandUnitType:               unitType,
			AverageSalesQuantity:                 avgSales,
			AverageSalesQuantityUnitAbbreviation: unitAbbrev,
			AverageSalesQuantityUnitType:         unitType,
			WeeksOfSales:                         wos,
		})
	}

	return &domain.WeeksOfSalesResult{
		Items: items,
		Count: int64(len(items)),
	}, nil
}

// AnalyzeScheduleAttainment measures actual production against the plan that was live at the time.
func (s *analyticsSvcImpl) AnalyzeScheduleAttainment(ctx context.Context, params domain.AnalyzeScheduleAttainmentParams) (*domain.ScheduleAttainmentResult, *apierror.APIError) {
	ctx, span := analyticsSvcTracer.Start(ctx, "service.analytics.analyze_schedule_attainment")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	// Attainment is a property of the schedule, so it is gated on the schedule domain rather than on whatever analytics happens to use elsewhere.
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSchedules, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if params.EndDate.Before(params.StartDate) {
		return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("The period must end on or after it starts.", "ends_at"))
	}

	params.AccountID = identity.Target.AccountID
	if params.GroupBy == "" {
		params.GroupBy = string(constants.AttainmentGroupByWeek)
	}

	return cachedReport(ctx, s.reportCache().attainment, analyticsReport{
		accountID: params.AccountID,
		family:    analyticsFamilyProduction,
		method:    "schedule_attainment",
		params:    params,
		ttl:       s.reportCache().ttlForWindow(params.EndDate),
	}, func(ctx context.Context) (*domain.ScheduleAttainmentResult, *apierror.APIError) {
		return s.buildScheduleAttainment(ctx, params)
	})
}
