package analyticsep

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	"google.golang.org/grpc"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
)

const (
	// pricingPageSize is the page size used to walk account prices, customers and products.
	pricingPageSize = 500
	// maxPricingPrices caps how many contracted prices are examined in one call.
	maxPricingPrices = 20000
	// maxPricingProducts caps the catalog loaded for cost lookup.
	maxPricingProducts = 20000
)

var (
	defaultTargetGrossMargin = decimal.RequireFromString("0.30")
	defaultOutlierTolerance  = decimal.RequireFromString("0.15")
)

// AnalyzeCustomerPricing audits contracted prices for outliers and thin margins.
func (m *analyticsSvcImpl) AnalyzeCustomerPricing(ctx context.Context, req *AnalyzeCustomerPricingRequest) (*apiresource.AnalyzeCustomerPricingResponse, *apierror.APIError) {
	targetMargin, apiErr := parsePricingFraction(req.TargetGrossMargin.Ptr(), defaultTargetGrossMargin, "target_gross_margin")
	if apiErr != nil {
		return nil, apiErr
	}
	outlierTolerance, apiErr := parsePricingFraction(req.OutlierTolerance.Ptr(), defaultOutlierTolerance, "outlier_tolerance")
	if apiErr != nil {
		return nil, apiErr
	}

	notes := make([]string, 0)

	prices, truncated, apiErr := m.listAllAccountPrices(ctx)
	if apiErr != nil {
		return nil, apiErr
	}
	if truncated {
		notes = append(notes, fmt.Sprintf("Only the first %d contracted prices were examined.", maxPricingPrices))
	}

	customers, apiErr := m.listPricingCustomers(ctx)
	if apiErr != nil {
		return nil, apiErr
	}

	costs, costNotes, apiErr := m.buildPricingCostIndex(ctx)
	if apiErr != nil {
		return nil, apiErr
	}
	notes = append(notes, costNotes...)

	candidates := buildPricingCandidates(prices, customers, costs)
	findings := analyzePricing(candidates, targetMargin, outlierTolerance)
	findings = filterPricingFindings(findings, customers, req)

	return presentPricingAnalysis(ctx, candidates, findings, notes), nil
}

// parsePricingFraction reads an optional 0..1 fraction, falling back to the default.
func parsePricingFraction(raw *string, fallback decimal.Decimal, param string) (decimal.Decimal, *apierror.APIError) {
	if raw == nil || *raw == "" {
		return fallback, nil
	}
	value, err := decimal.NewFromString(*raw)
	if err != nil {
		return decimal.Zero, apierror.NewValidationErrorWithParam("Must be a decimal between 0 and 1.", param)
	}
	if value.IsNegative() || value.GreaterThan(decimal.NewFromInt(1)) {
		return decimal.Zero, apierror.NewValidationErrorWithParam("Must be between 0 and 1.", param)
	}
	return value, nil
}

func (m *analyticsSvcImpl) listAllAccountPrices(ctx context.Context) ([]*pb.AccountPriceInfo, bool, *apierror.APIError) {
	out := make([]*pb.AccountPriceInfo, 0, pricingPageSize)
	var cursor *string

	for {
		req := &pb.ListAccountPricesRequest{Limit: pricingPageSize, Cursor: cursor}
		resp, apiErr := grpcutil.CallRPC(ctx, analyticsSvcTracer, "service.analytics.customer_pricing.prices", domain.ServiceName,
			func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListAccountPricesResponse, error) {
				return m.coreClient.ListAccountPrices(ctx, req, opts...)
			})
		if apiErr != nil {
			return nil, false, apiErr
		}

		out = append(out, resp.GetAccountPrices()...)
		if len(out) >= maxPricingPrices {
			return out[:maxPricingPrices], true, nil
		}

		page := resp.GetPageInfo()
		if page == nil || !page.GetHasNextPage() || page.GetNextCursor() == "" {
			break
		}
		next := page.GetNextCursor()
		cursor = &next
	}
	return out, false, nil
}

// listPricingCustomers indexes customers by account id so a price can be attributed, and so a price recorded against a parent can be attributed to its children too.
func (m *analyticsSvcImpl) listPricingCustomers(ctx context.Context) (map[string]*pb.CustomerProto, *apierror.APIError) {
	customers := make(map[string]*pb.CustomerProto)
	var cursor *string

	for {
		req := &pb.ListCustomersRequest{Limit: pricingPageSize, Cursor: cursor}
		resp, apiErr := grpcutil.CallRPC(ctx, analyticsSvcTracer, "service.analytics.customer_pricing.customers", domain.ServiceName,
			func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListCustomersResponse, error) {
				return m.coreClient.ListCustomers(ctx, req, opts...)
			})
		if apiErr != nil {
			return nil, apiErr
		}

		for _, customer := range resp.GetCustomers() {
			customers[customer.GetId()] = customer
		}

		page := resp.GetPageInfo()
		if page == nil || !page.GetHasNextPage() || page.GetNextCursor() == "" {
			break
		}
		next := page.GetNextCursor()
		cursor = &next
	}
	return customers, nil
}

// pricingCostEntry is one product's cost, kept with the facts needed to decide which contracted prices it belongs to.
type pricingCostEntry struct {
	productLineID   string
	attributeIDs    []string
	unitCost        decimal.Decimal
	denominatorUnit string
}

// buildPricingCostIndex loads the catalog once and indexes cost by product line, so a price's margin can be checked against the products it actually applies to.
func (m *analyticsSvcImpl) buildPricingCostIndex(ctx context.Context) (map[string][]pricingCostEntry, []string, *apierror.APIError) {
	index := make(map[string][]pricingCostEntry)
	notes := make([]string, 0)
	missingCost := 0
	loaded := 0
	var cursor *string

	for {
		req := &pb.ListProductsFullRequest{Limit: pricingPageSize, Cursor: cursor}
		resp, apiErr := grpcutil.CallRPC(ctx, analyticsSvcTracer, "service.analytics.customer_pricing.products", domain.ServiceName,
			func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListProductsFullResponse, error) {
				return m.coreClient.ListProductsFull(ctx, req, opts...)
			})
		if apiErr != nil {
			return nil, nil, apiErr
		}

		for _, product := range resp.GetProducts() {
			lineID := product.GetProductLineId()
			if lineID == "" {
				continue
			}
			loaded++

			cost := product.GetItem().GetUnitCost()
			value, err := decimal.NewFromString(cost.GetValue())
			if err != nil || !value.IsPositive() {
				missingCost++
				continue
			}

			attributeIDs := make([]string, 0, len(product.GetItem().GetAttributes()))
			for _, attribute := range product.GetItem().GetAttributes() {
				attributeIDs = append(attributeIDs, attribute.GetId())
			}

			index[lineID] = append(index[lineID], pricingCostEntry{
				productLineID:   lineID,
				attributeIDs:    attributeIDs,
				unitCost:        value,
				denominatorUnit: cost.GetDenominatorUnitId(),
			})
		}

		if loaded >= maxPricingProducts {
			notes = append(notes, fmt.Sprintf("Costs were read from the first %d products only.", maxPricingProducts))
			break
		}

		page := resp.GetPageInfo()
		if page == nil || !page.GetHasNextPage() || page.GetNextCursor() == "" {
			break
		}
		next := page.GetNextCursor()
		cursor = &next
	}

	if missingCost > 0 {
		notes = append(notes, fmt.Sprintf("%d products have no unit cost recorded, so prices covering only those products were not margin-checked.", missingCost))
	}
	return index, notes, nil
}

// buildPricingCandidates turns each contracted price into a comparable candidate, fanning a parent's price out to its children the way the pricing engine does.
func buildPricingCandidates(
	prices []*pb.AccountPriceInfo,
	customers map[string]*pb.CustomerProto,
	costs map[string][]pricingCostEntry,
) []pricingCandidate {
	childrenByParent := make(map[string][]*pb.CustomerProto)
	for _, customer := range customers {
		if parent := customer.GetParentAccount(); parent != nil && parent.GetId() != "" {
			childrenByParent[parent.GetId()] = append(childrenByParent[parent.GetId()], customer)
		}
	}

	candidates := make([]pricingCandidate, 0, len(prices))
	for _, price := range prices {
		recipientID := price.GetRecipientAccount().GetId()
		if recipientID == "" {
			continue
		}

		value, err := decimal.NewFromString(price.GetRate().GetValue())
		if err != nil || !value.IsPositive() {
			continue
		}

		attributeIDs := make([]string, 0, len(price.GetAttributes()))
		for _, attribute := range price.GetAttributes() {
			attributeIDs = append(attributeIDs, attribute.GetId())
		}

		lineID := price.GetProductLine().GetId()
		denominatorUnit := price.GetRate().GetDenominatorUnit().GetId()
		unitCost, hasCost := representativeCost(costs[lineID], lineID, attributeIDs, denominatorUnit)

		base := pricingCandidate{
			AccountPriceID:    price.GetId(),
			ProductLineID:     lineID,
			AttributeKey:      attributeKeyFor(attributeIDs),
			AttributeIDs:      attributeIDs,
			NumeratorUnitAbbr: price.GetRate().GetNumeratorUnit().GetAbbreviation(),
			Value:             value,
			NumeratorUnitID:   price.GetRate().GetNumeratorUnit().GetId(),
			DenominatorUnit:   denominatorUnit,
			DenominatorLabel:  price.GetRate().GetDenominatorUnit().GetAbbreviation(),
			UnitCost:          unitCost,
			HasUnitCost:       hasCost,
		}

		recipient := customers[recipientID]
		candidate := base
		candidate.CustomerID = recipientID
		candidate.CustomerName = recipient.GetName()
		candidate.CustomerNo = recipient.GetNumber()
		candidates = append(candidates, candidate)

		// A price on a parent account also prices its children's orders, so each child is audited on it too — otherwise a deep discount hides one level up.
		for _, child := range childrenByParent[recipientID] {
			inherited := base
			inherited.CustomerID = child.GetId()
			inherited.CustomerName = child.GetName()
			inherited.CustomerNo = child.GetNumber()
			inherited.Inherited = true
			candidates = append(candidates, inherited)
		}
	}
	return candidates
}

// representativeCost is the median cost of the products a price applies to. Median so one oddly-costed SKU in a wide line cannot swing the margin verdict.
//
// Costs in a different denominator unit from the price are skipped rather than converted: a wrong conversion would produce a confidently wrong margin.
func representativeCost(entries []pricingCostEntry, lineID string, priceAttributeIDs []string, denominatorUnit string) (decimal.Decimal, bool) {
	matched := make([]decimal.Decimal, 0)
	for _, entry := range entries {
		if entry.denominatorUnit != denominatorUnit {
			continue
		}
		if !productMatchesPrice(entry.productLineID, entry.attributeIDs, lineID, priceAttributeIDs) {
			continue
		}
		matched = append(matched, entry.unitCost)
	}
	if len(matched) == 0 {
		return decimal.Zero, false
	}
	return medianOf(matched), true
}

// filterPricingFindings narrows the reported findings to the requested customers. Applied after scoring so peer medians are still computed across everyone — a benchmark drawn only from the customers being audited would be no benchmark at all.
func filterPricingFindings(findings []pricingFinding, customers map[string]*pb.CustomerProto, req *AnalyzeCustomerPricingRequest) []pricingFinding {
	if len(req.CustomerIDs) == 0 && len(req.CustomerGroupIDs) == 0 {
		return findings
	}

	wantedCustomers := make(map[string]struct{}, len(req.CustomerIDs))
	for _, id := range req.CustomerIDs {
		wantedCustomers[id] = struct{}{}
	}
	wantedGroups := make(map[string]struct{}, len(req.CustomerGroupIDs))
	for _, id := range req.CustomerGroupIDs {
		wantedGroups[id] = struct{}{}
	}

	kept := make([]pricingFinding, 0, len(findings))
	for _, finding := range findings {
		if _, ok := wantedCustomers[finding.CustomerID]; ok {
			kept = append(kept, finding)
			continue
		}
		if len(wantedGroups) == 0 {
			continue
		}
		customer := customers[finding.CustomerID]
		if customer == nil {
			continue
		}
		if inAnyPricingGroup(customer, wantedGroups) {
			kept = append(kept, finding)
		}
	}
	return kept
}

// inAnyPricingGroup reports group membership the way the pricing engine resolves it: the customer's own group, plus every price group on the relation.
func inAnyPricingGroup(customer *pb.CustomerProto, wanted map[string]struct{}) bool {
	if group := customer.GetTypeGroup(); group != nil {
		if _, ok := wanted[group.GetId()]; ok {
			return true
		}
	}
	for _, group := range customer.GetPriceGroups() {
		if _, ok := wanted[group.GetId()]; ok {
			return true
		}
	}
	return false
}

// presentPricingAnalysis builds the response, leaving every expandable field nil and stashing the foreign keys the include resolver needs. Nothing here fabricates a sub-resource: an unexpanded relation serializes as null.
func presentPricingAnalysis(ctx context.Context, candidates []pricingCandidate, findings []pricingFinding, notes []string) *apiresource.AnalyzeCustomerPricingResponse {
	meta := resourcekit.GetLoadMeta(ctx)
	presented := make([]apiresource.CustomerPricingFinding, 0, len(findings))
	belowPeer, belowMargin := 0, 0

	for _, finding := range findings {
		id := finding.AccountPriceID + ":" + finding.CustomerID

		meta.Set(constants.ObjectTypeCustomerPricingFinding, id, "customer_id", finding.CustomerID)
		meta.Set(constants.ObjectTypeCustomerPricingFinding, id, "product_line_id", finding.ProductLineID)
		meta.Set(constants.ObjectTypeCustomerPricingFinding, id, "attribute_ids", finding.AttributeIDs)
		meta.Set(constants.ObjectTypeCustomerPricingFinding, id, "numerator_unit_id", finding.NumeratorUnitID)
		meta.Set(constants.ObjectTypeCustomerPricingFinding, id, "denominator_unit_id", finding.DenominatorUnit)

		origin := constants.AccountPriceOriginDirect
		if finding.Inherited {
			origin = constants.AccountPriceOriginInherited
		}

		item := apiresource.CustomerPricingFinding{
			ID:             id,
			Object:         constants.ObjectTypeCustomerPricingFinding,
			AccountPriceID: finding.AccountPriceID,
			Reason:         pricingFindingReason(finding.BelowPeerMedian, finding.BelowTargetMargin),
			Origin:         origin,
			UnitPrice:      computedMoneyRate(finding.Value, finding.NumeratorUnitAbbr, finding.DenominatorLabel),
		}
		if finding.HasPeerMedian {
			item.PeerMedianPrice = computedMoneyRate(finding.PeerMedian, finding.NumeratorUnitAbbr, finding.DenominatorLabel)
			fraction := finding.BelowPeerFraction.StringFixed(4)
			item.BelowPeerMedianFraction = &fraction
		}
		if finding.HasUnitCost {
			margin := finding.GrossMargin.StringFixed(4)
			item.GrossMargin = &margin
		}
		if finding.BelowPeerMedian {
			belowPeer++
		}
		if finding.BelowTargetMargin {
			belowMargin++
		}
		presented = append(presented, item)
	}

	notAssessed := 0
	for _, candidate := range candidates {
		if !candidate.HasUnitCost {
			notAssessed++
		}
	}

	return &apiresource.AnalyzeCustomerPricingResponse{
		Object:   constants.ObjectTypeAnalyzeCustomerPricingResponse,
		Findings: apiresource.NewList(presented, apiresource.PageInfo{}),
		Summary: apiresource.CustomerPricingSummary{
			Object:                 constants.ObjectTypeCustomerPricingSummary,
			PricesAnalyzed:         len(candidates),
			BelowPeerMedianCount:   belowPeer,
			BelowTargetMarginCount: belowMargin,
			MarginNotAssessedCount: notAssessed,
			Notes:                  notes,
		},
	}
}

// pricingFindingReason names the combination of checks that failed. A finding only exists when at least one failed, so there is no "neither" case to represent.
func pricingFindingReason(belowPeer, belowMargin bool) constants.PricingFindingReason {
	switch {
	case belowPeer && belowMargin:
		return constants.PricingFindingReasonBelowPeerMedianAndTargetMargin
	case belowMargin:
		return constants.PricingFindingReasonBelowTargetMargin
	default:
		return constants.PricingFindingReasonBelowPeerMedian
	}
}

// computedMoneyRate renders a price as an unpersisted rate. The unit sub-objects are left for the include resolver to load; display_value carries the readable form so a caller that does not expand them still sees the basis.
func computedMoneyRate(value decimal.Decimal, numeratorAbbr, denominatorAbbr string) *apiresource.ComputedRate {
	display := value.StringFixed(2)
	if numeratorAbbr != "" {
		display = numeratorAbbr + display
	}
	if denominatorAbbr != "" {
		display += " / " + denominatorAbbr
	}
	return &apiresource.ComputedRate{
		Object:       constants.ObjectTypeComputedRate,
		Value:        value.StringFixed(4),
		DisplayValue: display,
	}
}
