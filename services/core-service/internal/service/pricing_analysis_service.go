package service

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/augno/api/shared/appctx"

	"github.com/augno/api/services/core-service/internal/domain"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

// pricingSweepPageSize walks the contracted prices, customers and catalog. These are local reads rather than round trips, so the page size only bounds memory, not latency.
const pricingSweepPageSize = 1000

// AnalyzeCustomerPricing audits contracted prices for outliers and thin margins.
//
// The whole sweep runs here because it reads three collections end to end — every contracted price, every customer, and the catalog's costs. Walking those from outside the service meant a request's worth of paginated round trips, which is what made this time out; in-process they are ordinary queries.
func (s *analyticsSvcImpl) AnalyzeCustomerPricing(ctx context.Context, params domain.AnalyzeCustomerPricingParams) (*domain.CustomerPricingAnalysis, *apierror.APIError) {
	ctx, span := analyticsSvcTracer.Start(ctx, "service.analytics.analyze_customer_pricing")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID

	targetMargin, apiErr := parseAnalysisFraction(params.TargetGrossMargin, "0.30", "target_gross_margin")
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	outlierTolerance, apiErr := parseAnalysisFraction(params.OutlierTolerance, "0.15", "outlier_tolerance")
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	prices, apiErr := s.sweepAccountPrices(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	customers, apiErr := s.sweepCustomers(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	costs, notes, apiErr := s.buildPricingCostIndex(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	units, apiErr := s.sweepUnits(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	candidates := buildPricingCandidates(prices, customers, costs, units)
	findings := analyzePricing(candidates, targetMargin, outlierTolerance)
	findings = filterPricingFindings(findings, customers, params.CustomerIDs, params.CustomerGroupIDs)

	return buildCustomerPricingAnalysis(candidates, findings, notes), nil
}

func (s *analyticsSvcImpl) sweepAccountPrices(ctx context.Context, accountID string) ([]*domain.AccountPrice, *apierror.APIError) {
	repo := s.repos.NewAccountPriceRepo()
	out := make([]*domain.AccountPrice, 0)
	var cursor *string
	for {
		result, apiErr := repo.List(ctx, domain.ListAccountPricesParams{
			AccountID: accountID,
			Limit:     pricingSweepPageSize,
			Cursor:    cursor,
		})
		if apiErr != nil {
			return nil, apiErr
		}
		out = append(out, result.AccountPrices...)
		if !result.PageInfo.HasNextPage || result.PageInfo.NextCursor == nil {
			return out, nil
		}
		cursor = result.PageInfo.NextCursor
	}
}

// sweepCustomers indexes customers by account id so a price can be attributed, and so a price recorded against a parent can be attributed to its children too.
func (s *analyticsSvcImpl) sweepCustomers(ctx context.Context, accountID string) (map[string]*domain.Customer, *apierror.APIError) {
	repo := s.repos.NewCustomerRepo()
	out := make(map[string]*domain.Customer)
	var cursor *string
	for {
		result, apiErr := repo.List(ctx, domain.ListCustomersParams{
			AccountID: accountID,
			Limit:     pricingSweepPageSize,
			Cursor:    cursor,
		})
		if apiErr != nil {
			return nil, apiErr
		}
		for _, customer := range result.Items {
			out[customer.ID] = customer
		}
		if !result.PageInfo.HasNextPage || result.PageInfo.NextCursor == nil {
			return out, nil
		}
		cursor = result.PageInfo.NextCursor
	}
}

// sweepUnits indexes the account's units by id so a cost recorded on one basis can be restated on the basis a price is contracted against.
func (s *analyticsSvcImpl) sweepUnits(ctx context.Context, accountID string) (map[string]*domain.Unit, *apierror.APIError) {
	units, apiErr := s.repos.NewUnitRepo().Export(ctx, domain.ExportUnitsParams{AccountID: accountID})
	if apiErr != nil {
		return nil, apiErr
	}
	out := make(map[string]*domain.Unit, len(units))
	for _, unit := range units {
		out[unit.ID] = unit
	}
	return out, nil
}

// pricingCostEntry is one product's cost, kept with the facts needed to decide which contracted prices it belongs to.
type pricingCostEntry struct {
	productLineID   string
	attributeIDs    []string
	unitCost        decimal.Decimal
	denominatorUnit string
}

// buildPricingCostIndex loads the catalog once and indexes cost by product line, so a price's margin can be checked against the products it actually applies to.
func (s *analyticsSvcImpl) buildPricingCostIndex(ctx context.Context, accountID string) (map[string][]pricingCostEntry, []string, *apierror.APIError) {
	products, apiErr := s.repos.NewProductRepo().Export(ctx, domain.ExportProductsParams{AccountID: accountID})
	if apiErr != nil {
		return nil, nil, apiErr
	}

	index := make(map[string][]pricingCostEntry)
	notes := make([]string, 0)
	// The two ways a cost can be unusable are worth separating: one is a gap in the catalog, the other is a cost deliberately entered as zero. Reporting them as one number sent people looking for missing data that was not missing.
	noCostRecorded := 0
	zeroCost := 0
	priced := 0

	for _, product := range products {
		if product.ProductLineID == nil || *product.ProductLineID == "" || product.Item == nil {
			continue
		}
		cost := product.Item.UnitCost
		if cost == nil {
			noCostRecorded++
			continue
		}
		value, err := decimal.NewFromString(cost.Value)
		if err != nil || !value.IsPositive() {
			zeroCost++
			continue
		}
		priced++

		attributeIDs := make([]string, 0, len(product.Item.Attributes))
		for _, attribute := range product.Item.Attributes {
			attributeIDs = append(attributeIDs, attribute.ID)
		}

		lineID := *product.ProductLineID
		index[lineID] = append(index[lineID], pricingCostEntry{
			productLineID:   lineID,
			attributeIDs:    attributeIDs,
			unitCost:        value,
			denominatorUnit: cost.DenominatorUnitID,
		})
	}

	if noCostRecorded > 0 {
		notes = append(notes, costCoverageNote("no unit cost recorded", noCostRecorded, priced+noCostRecorded+zeroCost))
	}
	if zeroCost > 0 {
		notes = append(notes, costCoverageNote("a unit cost of zero", zeroCost, priced+noCostRecorded+zeroCost))
	}
	return index, notes, nil
}

// costCoverageNote reports a coverage gap against the catalog it was measured over. A bare count reads as alarming on a large catalog and as trivial on a small one, and neither reading is available without the denominator.
func costCoverageNote(reason string, n, total int) string {
	return fmt.Sprintf("%d of %d products (%s) have %s, so prices covering only those products were not margin-checked.", n, total, percentOf(n, total), reason)
}

func percentOf(n, total int) string {
	if total == 0 {
		return "0%"
	}
	return fmt.Sprintf("%.1f%%", float64(n)*100/float64(total))
}

// buildPricingCandidates turns each contracted price into a comparable candidate, fanning a parent's price out to its children the way the pricing engine does.
func buildPricingCandidates(
	prices []*domain.AccountPrice,
	customers map[string]*domain.Customer,
	costs map[string][]pricingCostEntry,
	units map[string]*domain.Unit,
) []pricingCandidate {
	childrenByParent := make(map[string][]*domain.Customer)
	for _, customer := range customers {
		if customer.ParentAccountID != nil && *customer.ParentAccountID != "" {
			childrenByParent[*customer.ParentAccountID] = append(childrenByParent[*customer.ParentAccountID], customer)
		}
	}

	candidates := make([]pricingCandidate, 0, len(prices))
	for _, price := range prices {
		value, err := decimal.NewFromString(price.RateValue)
		if err != nil || !value.IsPositive() {
			continue
		}

		attributeIDs := make([]string, 0, len(price.Attributes))
		for _, attribute := range price.Attributes {
			attributeIDs = append(attributeIDs, attribute.ID)
		}

		unitCost, hasCost := representativeCost(costs[price.ProductLineID], price.ProductLineID, attributeIDs, price.DenominatorUnitID, units)

		base := pricingCandidate{
			AccountPriceID:    price.ID,
			ProductLineID:     price.ProductLineID,
			AttributeKey:      attributeKeyFor(attributeIDs),
			AttributeIDs:      attributeIDs,
			NumeratorUnitAbbr: price.NumeratorUnitAbbr,
			Value:             value,
			NumeratorUnitID:   price.NumeratorUnitID,
			DenominatorUnit:   price.DenominatorUnitID,
			DenominatorLabel:  price.DenominatorUnitAbbr,
			UnitCost:          unitCost,
			HasUnitCost:       hasCost,
		}

		candidate := base
		candidate.CustomerID = price.RecipientAccountID
		candidate.CustomerName = price.RecipientAccountName
		candidate.CustomerNo = price.RecipientAccountNumber
		candidates = append(candidates, candidate)

		// A price on a parent account also prices its children's orders, so each child is audited on it too — otherwise a deep discount hides one level up.
		for _, child := range childrenByParent[price.RecipientAccountID] {
			inherited := base
			inherited.CustomerID = child.ID
			inherited.CustomerName = child.Name
			inherited.Inherited = true
			candidates = append(candidates, inherited)
		}
	}
	return candidates
}

// representativeCost is the median cost of the products a price applies to, restated on the price's own per-unit basis. Median so one oddly-costed SKU in a wide line cannot swing the margin verdict.
//
// Costs are routinely recorded per each while prices are contracted per pair or per dozen. Dropping those rather than converting them left most prices unassessed even though their products all had costs, so the basis is converted the way the pricing engine converts one.
func representativeCost(entries []pricingCostEntry, lineID string, priceAttributeIDs []string, denominatorUnit string, units map[string]*domain.Unit) (decimal.Decimal, bool) {
	matched := make([]decimal.Decimal, 0)
	for _, entry := range entries {
		if !productMatchesPrice(entry.productLineID, entry.attributeIDs, lineID, priceAttributeIDs) {
			continue
		}
		cost, ok := costOnBasis(entry.unitCost, entry.denominatorUnit, denominatorUnit, units)
		if !ok {
			continue
		}
		matched = append(matched, cost)
	}
	if len(matched) == 0 {
		return decimal.Zero, false
	}
	return medianOf(matched), true
}

// costOnBasis restates a cost recorded per one costUnit as a cost per one priceUnit.
//
// A rate's denominator converts inversely to a quantity: if one pair is two each, a cost of $3 per each is $6 per pair. So the factor is how many cost units make up one price unit.
func costOnBasis(cost decimal.Decimal, costUnitID, priceUnitID string, units map[string]*domain.Unit) (decimal.Decimal, bool) {
	if costUnitID == priceUnitID {
		return cost, true
	}
	costUnit, priceUnit := units[costUnitID], units[priceUnitID]
	if costUnit == nil || priceUnit == nil {
		return decimal.Zero, false
	}
	// Units of different dimensions share no base measure, so there is no conversion to make.
	if costUnit.UnitDimensionCode != priceUnit.UnitDimensionCode {
		return decimal.Zero, false
	}
	// An affine unit (one with an offset, as temperature has) does not carry a single multiplier, so a rate expressed against it cannot be rescaled by one factor. Refusing beats reporting a confidently wrong margin.
	if !isLinearUnit(costUnit) || !isLinearUnit(priceUnit) {
		return decimal.Zero, false
	}
	// One price unit is ratio(price)/ratio(cost) cost units. Both ratios are fractions, so the whole factor is applied as a single rational and the division is done last — dividing first turns a ratio like one-twelfth into a repeating decimal and a $36/dozen cost comes back as $2.9999999999999988 per each.
	priceNum, priceDen := linearRatioOf(priceUnit)
	costNum, costDen := linearRatioOf(costUnit)
	factorNum := priceNum.Mul(costDen)
	factorDen := priceDen.Mul(costNum)
	if !factorNum.IsPositive() || !factorDen.IsPositive() {
		return decimal.Zero, false
	}
	return cost.Mul(factorNum).Div(factorDen), true
}

// linearRatioOf is a unit's size relative to its group's base measure, as an unreduced fraction. A base unit is one base measure by definition, whatever its stored ratio columns say — the same rule normalizeQuantity applies.
func linearRatioOf(u *domain.Unit) (decimal.Decimal, decimal.Decimal) {
	if u.IsBaseUnit {
		return decimal.NewFromInt(1), decimal.NewFromInt(1)
	}
	return parseDecimal(u.RatioNumerator), parseDecimal(u.RatioDenominator)
}

func isLinearUnit(u *domain.Unit) bool {
	if u.IsBaseUnit {
		return true
	}
	return offsetOf(pricingUnitOf(u)).IsZero()
}

// pricingUnitOf adapts a catalog unit to the conversion shape the pricing engine already uses, so the analysis converts by exactly the same rules an order does.
func pricingUnitOf(u *domain.Unit) *domain.PricingUnit {
	return &domain.PricingUnit{
		ID:                u.ID,
		RatioNumerator:    u.RatioNumerator,
		RatioDenominator:  u.RatioDenominator,
		OffsetNumerator:   u.OffsetNumerator,
		OffsetDenominator: u.OffsetDenominator,
		IsBaseUnit:        u.IsBaseUnit,
	}
}

// filterPricingFindings narrows the reported findings to the requested customers. Applied after scoring so peer medians are still computed across everyone — a benchmark drawn only from the customers being audited would be no benchmark at all.
func filterPricingFindings(findings []pricingFinding, customers map[string]*domain.Customer, customerIDs, customerGroupIDs []string) []pricingFinding {
	if len(customerIDs) == 0 && len(customerGroupIDs) == 0 {
		return findings
	}

	wantedCustomers := make(map[string]struct{}, len(customerIDs))
	for _, id := range customerIDs {
		wantedCustomers[id] = struct{}{}
	}
	wantedGroups := make(map[string]struct{}, len(customerGroupIDs))
	for _, id := range customerGroupIDs {
		wantedGroups[id] = struct{}{}
	}

	kept := make([]pricingFinding, 0, len(findings))
	for _, finding := range findings {
		if _, ok := wantedCustomers[finding.CustomerID]; ok {
			kept = append(kept, finding)
			continue
		}
		customer := customers[finding.CustomerID]
		if customer == nil || len(wantedGroups) == 0 {
			continue
		}
		if customer.TypeGroupID != nil {
			if _, ok := wantedGroups[*customer.TypeGroupID]; ok {
				kept = append(kept, finding)
			}
		}
	}
	return kept
}

func buildCustomerPricingAnalysis(candidates []pricingCandidate, findings []pricingFinding, notes []string) *domain.CustomerPricingAnalysis {
	out := &domain.CustomerPricingAnalysis{
		PricesAnalyzed: len(candidates),
		Notes:          notes,
		Findings:       make([]domain.CustomerPricingFinding, 0, len(findings)),
	}

	for _, finding := range findings {
		origin := "direct"
		if finding.Inherited {
			origin = "inherited"
		}
		item := domain.CustomerPricingFinding{
			AccountPriceID:    finding.AccountPriceID,
			CustomerID:        finding.CustomerID,
			ProductLineID:     finding.ProductLineID,
			AttributeIDs:      finding.AttributeIDs,
			UnitPrice:         finding.Value.StringFixed(4),
			NumeratorUnitID:   finding.NumeratorUnitID,
			NumeratorUnitAbbr: finding.NumeratorUnitAbbr,
			DenominatorUnitID: finding.DenominatorUnit,
			DenominatorAbbr:   finding.DenominatorLabel,
			Origin:            origin,
			Reason:            pricingFindingReason(finding.BelowPeerMedian, finding.BelowTargetMargin),
		}
		if finding.HasPeerMedian {
			median := finding.PeerMedian.StringFixed(4)
			fraction := finding.BelowPeerFraction.StringFixed(4)
			item.PeerMedianPrice = &median
			item.BelowPeerMedianFraction = &fraction
		}
		if finding.HasUnitCost {
			margin := finding.GrossMargin.StringFixed(4)
			item.GrossMargin = &margin
		}
		if finding.BelowPeerMedian {
			out.BelowPeerMedianCount++
		}
		if finding.BelowTargetMargin {
			out.BelowTargetMarginCount++
		}
		out.Findings = append(out.Findings, item)
	}

	for _, candidate := range candidates {
		if !candidate.HasUnitCost {
			out.MarginNotAssessedCount++
		}
	}
	return out
}
