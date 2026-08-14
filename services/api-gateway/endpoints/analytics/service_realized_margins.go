package analyticsep

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
)

// AnalyzeRealizedMargins audits invoiced sales for thin margins and prices well below what other customers achieved.
func (m *analyticsSvcImpl) AnalyzeRealizedMargins(ctx context.Context, req *AnalyzeRealizedMarginsRequest) (*apiresource.AnalyzeRealizedMarginsResponse, *apierror.APIError) {
	targetMargin, apiErr := parsePricingFraction(req.TargetGrossMargin.Ptr(), defaultTargetGrossMargin, "target_gross_margin")
	if apiErr != nil {
		return nil, apiErr
	}
	outlierTolerance, apiErr := parsePricingFraction(req.OutlierTolerance.Ptr(), defaultOutlierTolerance, "outlier_tolerance")
	if apiErr != nil {
		return nil, apiErr
	}
	if req.EndDate.Before(req.StartDate) {
		return nil, apierror.NewValidationErrorWithParam("Must not be before starts_at.", "ends_at")
	}

	// The peer benchmark is drawn from every customer that bought the SKU, so the request's customer filters are deliberately not sent to the query — narrowing the population would compare a customer against a benchmark it helped set.
	pbReq := &pb.AnalyzeSalesRequest{
		StartDate:      timestamppb.New(req.StartDate),
		EndDate:        timestamppb.New(req.EndDate),
		ProductLineIds: req.ProductLineIDs,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, analyticsSvcTracer, "service.analytics.realized_margins.sales", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.AnalyzeSalesResponse, error) {
			return m.coreClient.AnalyzeSales(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	entries := resp.GetEntries()
	lines := make([]realizedLine, 0, len(entries))
	for _, entry := range entries {
		lines = append(lines, realizedLine{
			CustomerID:      entry.GetCustomerId(),
			CustomerName:    entry.GetCustomerName(),
			CustomerNo:      entry.GetCustomerNumber(),
			CustomerGroup:   entry.GetCustomerGroupName(),
			CustomerGroupID: entry.GetCustomerTypeGroupId(),
			ItemID:          entry.GetItemId(),
			SKU:             entry.GetProductSku(),
			ProductLineID:   entry.GetProductLineId(),
			ProductLine:     entry.GetProductLine(),
			UnitAbbr:        entry.GetUnit(),
			Quantity:        decimal.NewFromFloat(entry.GetQuantityInvoiced()),
			Revenue:         decimal.NewFromFloat(entry.GetTotalInvoiced()),
			Cost:            decimal.NewFromFloat(entry.GetTotalCost()),
		})
	}

	aggregates := aggregateRealizedLines(lines)
	findings := analyzeRealizedMargins(aggregates, targetMargin, outlierTolerance)
	findings = filterRealizedFindings(findings, req)

	notes := make([]string, 0)
	if len(entries) == 0 {
		notes = append(notes, "No invoiced sales fall in this window.")
	}
	medians := realizedPeerMedians(aggregates)
	singleBuyer := 0
	for _, aggregate := range aggregates {
		if _, ok := medians[aggregate.ItemID]; !ok {
			singleBuyer++
		}
	}
	if singleBuyer > 0 {
		notes = append(notes, fmt.Sprintf("%d customer/SKU pairs had no other buyer to compare against, so only their margin was checked.", singleBuyer))
	}

	return presentRealizedMargins(ctx, len(entries), aggregates, findings, notes), nil
}

// filterRealizedFindings narrows the reported findings after scoring, so the peer benchmark still reflects every customer that bought the SKU.
func filterRealizedFindings(findings []realizedFinding, req *AnalyzeRealizedMarginsRequest) []realizedFinding {
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

// presentRealizedMargins builds the response, leaving every expandable field nil and stashing the foreign keys the include resolver needs.
func presentRealizedMargins(ctx context.Context, lineCount int, aggregates []realizedAggregate, findings []realizedFinding, notes []string) *apiresource.AnalyzeRealizedMarginsResponse {
	meta := resourcekit.GetLoadMeta(ctx)
	presented := make([]apiresource.RealizedMarginFinding, 0, len(findings))
	belowPeer, belowMargin := 0, 0

	for _, finding := range findings {
		id := finding.CustomerID + ":" + finding.ItemID

		meta.Set(constants.ObjectTypeRealizedMarginFinding, id, "customer_id", finding.CustomerID)
		meta.Set(constants.ObjectTypeRealizedMarginFinding, id, "customer_group_id", finding.CustomerGroupID)
		meta.Set(constants.ObjectTypeRealizedMarginFinding, id, "item_id", finding.ItemID)
		meta.Set(constants.ObjectTypeRealizedMarginFinding, id, "product_line_id", finding.ProductLineID)

		item := apiresource.RealizedMarginFinding{
			ID:               id,
			Object:           constants.ObjectTypeRealizedMarginFinding,
			Reason:           pricingFindingReason(finding.BelowPeerMedian, finding.BelowTargetMargin),
			QuantityInvoiced: computedQuantity(finding.Quantity, finding.UnitAbbr),
			// The invoiced-sales payload carries no currency unit, so money renders as a plain formatted amount rather than guessing a symbol.
			Revenue:          computedQuantity(finding.Revenue, ""),
			Cost:             computedQuantity(finding.Cost, ""),
			AverageUnitPrice: computedMoneyRate(finding.AveragePrice, "", finding.UnitAbbr),
			LineCount:        finding.LineCount,
		}
		if finding.HasPeerMedian {
			item.PeerMedianPrice = computedMoneyRate(finding.PeerMedianPrice, "", finding.UnitAbbr)
			fraction := finding.BelowPeerFraction.StringFixed(4)
			item.BelowPeerMedianFraction = &fraction
		}
		if finding.HasGrossMargin {
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
	for _, aggregate := range aggregates {
		if !aggregate.Cost.IsPositive() {
			notAssessed++
		}
	}

	return &apiresource.AnalyzeRealizedMarginsResponse{
		Object:   constants.ObjectTypeAnalyzeRealizedMarginsResponse,
		Findings: apiresource.NewList(presented, apiresource.PageInfo{}),
		Summary: apiresource.RealizedMarginSummary{
			Object:                 constants.ObjectTypeRealizedMarginSummary,
			LinesAnalyzed:          lineCount,
			RelationshipsAnalyzed:  len(aggregates),
			BelowPeerMedianCount:   belowPeer,
			BelowTargetMarginCount: belowMargin,
			MarginNotAssessedCount: notAssessed,
			Notes:                  notes,
		},
	}
}

// computedQuantity renders an amount as an unpersisted quantity. The unit sub-object is left for the include resolver; display_value carries the readable form.
func computedQuantity(value decimal.Decimal, unitAbbr string) *apiresource.ComputedQuantity {
	display := value.StringFixed(2)
	if unitAbbr != "" {
		display += " " + unitAbbr
	}
	return &apiresource.ComputedQuantity{
		Object:       constants.ObjectTypeComputedQuantity,
		Value:        value.StringFixed(4),
		DisplayValue: display,
	}
}
