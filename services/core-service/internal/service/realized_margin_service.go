package service

import (
	"context"

	"github.com/shopspring/decimal"

	"github.com/augno/api/services/core-service/internal/domain"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

// AnalyzeRealizedMargins audits what customers were actually invoiced, flagging prices well below what other customers achieved on the same SKU and sales that do not clear a target gross margin.
//
// The invoiced lines are read and rolled up here rather than shipped out whole: a year of trading across a few thousand customers and SKUs is hundreds of thousands of lines, far past what one response can carry. Callers get one row per customer and SKU instead.
func (s *analyticsSvcImpl) AnalyzeRealizedMargins(ctx context.Context, params domain.AnalyzeRealizedMarginsParams) (*domain.RealizedMarginAnalysis, *apierror.APIError) {
	ctx, span := analyticsSvcTracer.Start(ctx, "service.analytics.analyze_realized_margins")
	defer span.End()

	targetMargin, apiErr := parseAnalysisFraction(params.TargetGrossMargin, "0.30", "target_gross_margin")
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	outlierTolerance, apiErr := parseAnalysisFraction(params.OutlierTolerance, "0.15", "outlier_tolerance")
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// The peer benchmark is drawn from every customer that bought the SKU, so the
	// caller's customer filters are deliberately not pushed into the query — narrowing
	// the population would compare a customer against a benchmark it helped set.
	entries, apiErr := s.AnalyzeSales(ctx, domain.AnalyzeSalesParams{
		StartDate:      params.StartDate,
		EndDate:        params.EndDate,
		ProductLineIDs: params.ProductLineIDs,
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	lines := make([]realizedLine, 0, len(entries))
	for _, entry := range entries {
		lines = append(lines, realizedLine{
			CustomerID:      entry.CustomerID,
			CustomerName:    entry.CustomerName,
			CustomerNo:      entry.CustomerNumber,
			CustomerGroup:   derefString(entry.CustomerGroupName),
			CustomerGroupID: derefString(entry.CustomerTypeGroupID),
			ItemID:          entry.ItemID,
			SKU:             entry.ProductSku,
			ProductLineID:   derefString(entry.ProductLineID),
			ProductLine:     derefString(entry.ProductLine),
			UnitAbbr:        entry.Unit,
			Quantity:        decimal.NewFromFloat(entry.QuantityInvoiced),
			Revenue:         decimal.NewFromFloat(entry.TotalInvoiced),
			Cost:            decimal.NewFromFloat(entry.TotalCost),
		})
	}

	aggregates := aggregateRealizedLines(lines)
	findings := analyzeRealizedMargins(aggregates, targetMargin, outlierTolerance)
	findings = filterRealizedFindings(findings, params.CustomerIDs, params.CustomerGroupIDs)

	return buildRealizedMarginAnalysis(len(entries), aggregates, findings), nil
}

// filterRealizedFindings narrows the reported findings after scoring, so the peer benchmark still reflects every customer that bought the SKU.
func filterRealizedFindings(findings []realizedFinding, customerIDs, customerGroupIDs []string) []realizedFinding {
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

	kept := make([]realizedFinding, 0, len(findings))
	for _, finding := range findings {
		if _, ok := wantedCustomers[finding.CustomerID]; ok {
			kept = append(kept, finding)
			continue
		}
		if _, ok := wantedGroups[finding.CustomerGroupID]; ok {
			kept = append(kept, finding)
		}
	}
	return kept
}

// buildRealizedMarginAnalysis converts the scored findings into the domain result the transport presents.
func buildRealizedMarginAnalysis(lineCount int, aggregates []realizedAggregate, findings []realizedFinding) *domain.RealizedMarginAnalysis {
	out := &domain.RealizedMarginAnalysis{
		LinesAnalyzed:         lineCount,
		RelationshipsAnalyzed: len(aggregates),
		Findings:              make([]domain.RealizedMarginFinding, 0, len(findings)),
	}

	for _, finding := range findings {
		item := domain.RealizedMarginFinding{
			CustomerID:       finding.CustomerID,
			CustomerGroupID:  finding.CustomerGroupID,
			ItemID:           finding.ItemID,
			ProductLineID:    finding.ProductLineID,
			UnitAbbreviation: finding.UnitAbbr,
			QuantityInvoiced: finding.Quantity.StringFixed(4),
			Revenue:          finding.Revenue.StringFixed(2),
			Cost:             finding.Cost.StringFixed(2),
			AverageUnitPrice: finding.AveragePrice.StringFixed(4),
			LineCount:        finding.LineCount,
			Reason:           pricingFindingReason(finding.BelowPeerMedian, finding.BelowTargetMargin),
		}
		if finding.HasPeerMedian {
			median := finding.PeerMedianPrice.StringFixed(4)
			fraction := finding.BelowPeerFraction.StringFixed(4)
			item.PeerMedianPrice = &median
			item.BelowPeerMedianFraction = &fraction
		}
		if finding.HasGrossMargin {
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

	for _, aggregate := range aggregates {
		if !aggregate.Cost.IsPositive() {
			out.MarginNotAssessedCount++
		}
	}
	return out
}

// pricingFindingReason names the combination of checks that failed. A finding only exists when at least one failed, so there is no "neither" case to represent.
func pricingFindingReason(belowPeer, belowMargin bool) string {
	switch {
	case belowPeer && belowMargin:
		return "below_peer_median_and_target_margin"
	case belowMargin:
		return "below_target_margin"
	default:
		return "below_peer_median"
	}
}

// parseAnalysisFraction reads an optional 0..1 fraction, falling back to a default.
func parseAnalysisFraction(raw *string, fallback, param string) (decimal.Decimal, *apierror.APIError) {
	value := fallback
	if raw != nil && *raw != "" {
		value = *raw
	}
	parsed, err := decimal.NewFromString(value)
	if err != nil {
		return decimal.Zero, apierror.NewValidationErrorWithParam("Must be a decimal between 0 and 1.", param)
	}
	if parsed.IsNegative() || parsed.GreaterThan(decimal.NewFromInt(1)) {
		return decimal.Zero, apierror.NewValidationErrorWithParam("Must be between 0 and 1.", param)
	}
	return parsed, nil
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
