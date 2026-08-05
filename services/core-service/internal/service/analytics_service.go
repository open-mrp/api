package service

import (
	"context"
	"fmt"
	"time"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var analyticsSvcTracer = tracing.GetTracer("core-service.service.analytics")

type analyticsSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
}

type AnalyticsSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
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

	return s.buildDemandForecast(ctx, params)
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

	params.AccountID = identity.Target.AccountID

	return s.buildOeeByDepartment(ctx, params)
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

	params.AccountID = identity.Target.AccountID

	return s.buildOeeTrend(ctx, params)
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

	repo := s.repos.NewAnalyticsRepo()

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

	// 4. Get on-hand inventory for all items.
	inventoryRows, apiErr := s.repos.NewInventoryQueryRepo().FetchOnHandInventoryBulk(ctx, allItemIDs, params.AccountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Build inventory map: itemID -> onHandQuantity.
	invMap := make(map[string]float64)
	for _, row := range inventoryRows {
		invMap[row.ItemID] = row.OnHandQuantity
	}

	// 5. For each product line, compute metrics.
	weeks := params.PeriodInWeeks
	if weeks < 1 {
		weeks = 4
	}
	endDate := time.Now()
	startDate := endDate.Add(-time.Duration(weeks) * 7 * 24 * time.Hour)

	var items []domain.WeeksOfSalesItem
	for _, plInfo := range plInfoRows {
		// Get order quantity for this product line in the period.
		orderRow, apiErr := repo.GetOrderQuantityByProductLine(ctx, domain.GetOrderQuantityByProductLineParams{
			AccountID:     params.AccountID,
			ProductLineID: plInfo.ID,
			StartDate:     startDate,
			EndDate:       endDate,
		})
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		totalDemand := orderRow.TotalQuantity
		unitAbbrev := orderRow.UnitAbbreviation
		unitType := orderRow.UnitType

		// Sum on-hand for items in this product line.
		itemIDsForLine := itemsByProductLine[plInfo.ID]
		var onHand float64
		for _, iid := range itemIDsForLine {
			onHand += invMap[iid]
		}

		avgSales := totalDemand / float64(weeks)
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

	return s.buildScheduleAttainment(ctx, params)
}
