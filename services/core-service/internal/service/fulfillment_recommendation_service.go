package service

import (
	"context"
	"sort"
	"time"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/scheduling"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

// recommendationHistoryMonths is how far back demand shape is measured.
//
// Twelve months rather than the forecast's twenty-four: the question is what this item is like *now*, and two years of history would let a SKU that has since gone dormant still read as steady.
const recommendationHistoryMonths = 12

// ListFulfillmentRecommendations works out which SKUs should be built to order and which to stock, and says why for each.
//
// Computed live rather than stored. A recommendation is only useful next to current demand, and a materialized one would need a job to keep it honest — where the durable artifact is the item setting a merchant writes after agreeing with it.
func (s *productionScheduleSvcImpl) ListFulfillmentRecommendations(ctx context.Context) ([]*domain.FulfillmentRecommendation, *apierror.APIError) {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.list_fulfillment_recommendations")
	defer span.End()

	identity, apiErr := s.readIdentity(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSchedules, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID

	settings, apiErr := s.loadEffectiveSettings(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	repo := s.repos.NewProductionScheduleInputRepo()
	asOf := time.Now().UTC()
	windowStart := asOf.AddDate(0, -recommendationHistoryMonths, 0)

	// Every sellable item is a candidate: the question "should this be made to order" is asked of finished goods, which is where the policy is resolved.
	products, apiErr := repo.GetAllSellableProducts(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if len(products) == 0 {
		return nil, nil
	}

	productIDs := make([]string, 0, len(products))
	itemByProduct := make(map[string]string, len(products))
	skuByItem := make(map[string]string, len(products))
	lineByItem := make(map[string]string, len(products))
	for _, p := range products {
		productIDs = append(productIDs, p.ProductID)
		itemByProduct[p.ProductID] = p.ItemID
		skuByItem[p.ItemID] = p.SKU
		if p.ProductLineID != nil {
			lineByItem[p.ItemID] = *p.ProductLineID
		}
	}
	sort.Strings(productIDs)

	demandRows, apiErr := repo.GetProductDemandByCustomer(ctx, domain.GetPooledOrderDemandParams{
		AccountID:   accountID,
		WindowStart: windowStart,
		WindowEnd:   asOf,
		ProductIDs:  productIDs,
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	profiles, apiErr := repo.GetCustomerFulfillmentProfiles(ctx, accountID, settings.DefaultCustomerLeadTimeDays)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	profileByCustomer := make(map[string]domain.CustomerFulfillmentProfile, len(profiles))
	for _, p := range profiles {
		profileByCustomer[p.CustomerAccountID] = p
	}

	costs, apiErr := repo.GetItemUnitCosts(ctx, accountID, sortedItemIDs(skuByItem))
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	linePolicies, apiErr := repo.ListProductLineFulfillmentPolicies(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	itemOverrides := map[string]string{}
	for itemID, setting := range settings.ItemSettings {
		if setting.FulfillmentPolicyCode != "" {
			itemOverrides[itemID] = setting.FulfillmentPolicyCode
		}
	}

	inputs := buildClassificationInputs(classificationSources{
		AsOf:              asOf,
		HistoryMonths:     recommendationHistoryMonths,
		DemandRows:        demandRows,
		ItemByProduct:     itemByProduct,
		SKUByItem:         skuByItem,
		ProfileByCustomer: profileByCustomer,
		UnitCostByItem:    costs,
		ProductionWeeks:   settings.Settings.DefaultConstraintLeadTimeWeeks + settings.Settings.FinishLeadTimeWeeks,
		Resolution: scheduling.PolicyResolutionInput{
			ItemOverrides:       itemOverrides,
			ProductLineByItem:   lineByItem,
			PolicyByProductLine: linePolicies,
			AccountDefault:      settings.DefaultFulfillmentPolicy,
		},
	})

	out := make([]*domain.FulfillmentRecommendation, 0, len(inputs))
	for _, in := range inputs {
		rec := scheduling.RecommendPolicy(in.input, settings.RecommendationThresholds)
		out = append(out, &domain.FulfillmentRecommendation{
			Recommendation:   rec,
			ProductLineID:    lineByItem[in.input.ItemID],
			MixedStreamShare: scheduling.MixedStreamShare(in.input.Customers, rec.RecommendedPolicy),
		})
	}

	// Changes first, then by SKU. A merchant opening this page is looking for what to act on, and a hundred unchanged rows above the first real suggestion is a page nobody reads twice.
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Changes() != out[j].Changes() {
			return out[i].Changes()
		}
		if out[i].SKU != out[j].SKU {
			return out[i].SKU < out[j].SKU
		}
		return out[i].ItemID < out[j].ItemID
	})

	return out, nil
}

// classificationSources is everything the per-item inputs are assembled from.
type classificationSources struct {
	AsOf              time.Time
	HistoryMonths     int
	DemandRows        []domain.CustomerDemandRow
	ItemByProduct     map[string]string
	SKUByItem         map[string]string
	ProfileByCustomer map[string]domain.CustomerFulfillmentProfile
	UnitCostByItem    map[string]float64
	ProductionWeeks   float64
	Resolution        scheduling.PolicyResolutionInput
}

type classificationInput struct {
	input scheduling.ClassificationInput
}

// buildClassificationInputs turns raw per-customer monthly rows into one classification input per item.
//
// Items with no demand at all in the window are still classified: "nothing has sold" is the most actionable verdict this produces, and dropping those rows would hide exactly the SKUs holding dead stock.
func buildClassificationInputs(src classificationSources) []classificationInput {
	type itemAccumulator struct {
		monthly       map[time.Time]float64
		byCustomer    map[string]float64
		lastSaleMonth time.Time
		annual        float64
	}

	acc := map[string]*itemAccumulator{}
	for itemID := range src.SKUByItem {
		acc[itemID] = &itemAccumulator{monthly: map[time.Time]float64{}, byCustomer: map[string]float64{}}
	}

	for _, row := range src.DemandRows {
		itemID, ok := src.ItemByProduct[row.ProductID]
		if !ok {
			continue
		}
		a := acc[itemID]
		if a == nil {
			continue
		}
		monthStart := time.Date(row.Year, time.Month(row.Month), 1, 0, 0, 0, 0, time.UTC)
		a.monthly[monthStart] += row.Quantity
		a.byCustomer[row.BuyerAccountID] += row.Quantity
		a.annual += row.Quantity
		if row.Quantity > 0 && monthStart.After(a.lastSaleMonth) {
			a.lastSaleMonth = monthStart
		}
	}

	itemIDs := sortedItemIDs(src.SKUByItem)
	out := make([]classificationInput, 0, len(itemIDs))

	for _, itemID := range itemIDs {
		a := acc[itemID]

		// A dense series, so a month with no demand counts as a gap rather than being absent from the history entirely — the interval measure depends on the zeroes being there.
		monthly := make([]float64, 0, src.HistoryMonths)
		cursor := monthStartOfUTC(src.AsOf).AddDate(0, -src.HistoryMonths, 0)
		for range src.HistoryMonths {
			monthly = append(monthly, a.monthly[cursor])
			cursor = cursor.AddDate(0, 1, 0)
		}

		monthsSince := monthsSinceLastSale(a.lastSaleMonth, src.AsOf, src.HistoryMonths)

		customers := make([]scheduling.CustomerDemand, 0, len(a.byCustomer))
		for customerID, units := range a.byCustomer {
			profile := src.ProfileByCustomer[customerID]
			customers = append(customers, scheduling.CustomerDemand{
				CustomerAccountID: customerID,
				CustomerName:      profile.CustomerName,
				Units:             units,
				LeadTimeDays:      profile.LeadTimeDays,
				FulfillmentPolicy: profile.FulfillmentPolicyCode,
			})
		}
		// Go randomizes map iteration and the classifier reports a top customer, so the order is pinned before it is handed over.
		sort.SliceStable(customers, func(i, j int) bool {
			return customers[i].CustomerAccountID < customers[j].CustomerAccountID
		})

		resolved := scheduling.ResolveFulfillmentPolicy(itemID, src.Resolution)

		out = append(out, classificationInput{input: scheduling.ClassificationInput{
			ItemID:                       itemID,
			SKU:                          src.SKUByItem[itemID],
			Monthly:                      monthly,
			MonthsObserved:               src.HistoryMonths,
			MonthsSinceLastSale:          monthsSince,
			AnnualDemand:                 a.annual,
			UnitCost:                     src.UnitCostByItem[itemID],
			TotalProductionLeadTimeWeeks: src.ProductionWeeks,
			Customers:                    customers,
			CurrentPolicy:                resolved.Policy,
		}})
	}

	return out
}

// monthsSinceLastSale reports how long since anything sold, capped at the observation window.
//
// An item that has never sold inside the window reports the whole window rather than an unbounded number: the history simply does not reach further back, and inventing a larger figure would claim knowledge the query never had.
func monthsSinceLastSale(lastSale, asOf time.Time, windowMonths int) int {
	if lastSale.IsZero() {
		return windowMonths
	}
	months := int(monthStartOfUTC(asOf).Sub(lastSale).Hours() / 24 / 30.44)
	if months < 0 {
		return 0
	}
	if months > windowMonths {
		return windowMonths
	}
	return months
}

func monthStartOfUTC(t time.Time) time.Time {
	utc := t.UTC()
	return time.Date(utc.Year(), utc.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func sortedItemIDs(skuByItem map[string]string) []string {
	out := make([]string, 0, len(skuByItem))
	for itemID := range skuByItem {
		out = append(out, itemID)
	}
	sort.Strings(out)
	return out
}

// ApplyFulfillmentRecommendations writes the recommended policy onto the named items.
//
// Explicit item IDs rather than "apply everything": a recommendation is advice, and adopting it in bulk without naming what is being adopted is how a plant changes what it builds by accident. The items a caller did not name are left exactly as they were.
func (s *productionScheduleSvcImpl) ApplyFulfillmentRecommendations(ctx context.Context, itemIDs []string) ([]*domain.FulfillmentRecommendation, *apierror.APIError) {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.apply_fulfillment_recommendations")
	defer span.End()

	if _, apiErr := s.writeIdentity(ctx, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if len(itemIDs) == 0 {
		return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("At least one item is required.", "item_ids"))
	}

	// Recomputed here rather than trusted from the request: a recommendation the caller read minutes ago may no longer be the advice, and applying a stale verdict would set a policy the engine would not give today.
	recommendations, apiErr := s.ListFulfillmentRecommendations(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	byItem := make(map[string]*domain.FulfillmentRecommendation, len(recommendations))
	for _, rec := range recommendations {
		byItem[rec.ItemID] = rec
	}

	applied := make([]*domain.FulfillmentRecommendation, 0, len(itemIDs))
	for _, itemID := range itemIDs {
		rec, ok := byItem[itemID]
		if !ok {
			return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("No recommendation exists for item "+itemID+".", "item_ids"))
		}

		policy := rec.RecommendedPolicy
		if _, apiErr := s.UpsertItemSetting(ctx, domain.UpsertItemSettingParams{
			ItemID:                itemID,
			FulfillmentPolicyCode: &policy,
		}); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		applied = append(applied, rec)
	}

	return applied, nil
}
