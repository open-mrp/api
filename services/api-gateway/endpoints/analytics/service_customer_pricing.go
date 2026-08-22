package analyticsep

import (
	"context"

	"google.golang.org/grpc"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
)

// AnalyzeCustomerPricing audits contracted prices for outliers and thin margins.
//
// The sweep runs in core-service: it reads every contracted price, every customer and the catalog's costs, and walking those from here meant a request's worth of paginated round trips.
func (m *analyticsSvcImpl) AnalyzeCustomerPricing(ctx context.Context, req *AnalyzeCustomerPricingRequest) (*apiresource.AnalyzeCustomerPricingResponse, *apierror.APIError) {
	pbReq := &pb.AnalyzeCustomerPricingRequest{
		CustomerIds:       req.CustomerIDs,
		CustomerGroupIds:  req.CustomerGroupIDs,
		TargetGrossMargin: req.TargetGrossMargin.Ptr(),
		OutlierTolerance:  req.OutlierTolerance.Ptr(),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, analyticsSvcTracer, "service.analytics.customer_pricing", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.AnalyzeCustomerPricingResponse, error) {
			return m.coreClient.AnalyzeCustomerPricing(ctx, pbReq, opts...)
		}, grpcutil.WithTimeout(grpcutil.AnalyticsOperationTimeout))
	if apiErr != nil {
		return nil, apiErr
	}

	return presentCustomerPricing(ctx, resp), nil
}

// presentCustomerPricing builds the response, leaving every expandable field nil and stashing the foreign keys the include resolver needs. Nothing here fabricates a sub-resource: an unexpanded relation serializes as null.
func presentCustomerPricing(ctx context.Context, resp *pb.AnalyzeCustomerPricingResponse) *apiresource.AnalyzeCustomerPricingResponse {
	meta := resourcekit.GetLoadMeta(ctx)
	presented := make([]apiresource.CustomerPricingFinding, 0, len(resp.GetFindings()))

	for _, f := range resp.GetFindings() {
		id := f.GetAccountPriceId() + ":" + f.GetCustomerId()

		meta.Set(constants.ObjectTypeCustomerPricingFinding, id, "customer_id", f.GetCustomerId())
		meta.Set(constants.ObjectTypeCustomerPricingFinding, id, "product_line_id", f.GetProductLineId())
		meta.Set(constants.ObjectTypeCustomerPricingFinding, id, "attribute_ids", f.GetAttributeIds())
		meta.Set(constants.ObjectTypeCustomerPricingFinding, id, "numerator_unit_id", f.GetNumeratorUnitId())
		meta.Set(constants.ObjectTypeCustomerPricingFinding, id, "denominator_unit_id", f.GetDenominatorUnitId())

		item := apiresource.CustomerPricingFinding{
			ID:             id,
			Object:         constants.ObjectTypeCustomerPricingFinding,
			AccountPriceID: f.GetAccountPriceId(),
			Reason:         constants.PricingFindingReason(f.GetReason()),
			Origin:         constants.AccountPriceOrigin(f.GetOrigin()),
			UnitPrice:      computedRate(f.GetUnitPrice(), f.GetNumeratorUnitAbbr(), f.GetDenominatorAbbr()),
		}
		if f.PeerMedianPrice != nil {
			item.PeerMedianPrice = computedRate(f.GetPeerMedianPrice(), f.GetNumeratorUnitAbbr(), f.GetDenominatorAbbr())
			item.BelowPeerMedianFraction = f.BelowPeerMedianFraction
		}
		item.GrossMargin = f.GrossMargin
		presented = append(presented, item)
	}

	notes := resp.GetNotes()
	if notes == nil {
		notes = []string{}
	}

	return &apiresource.AnalyzeCustomerPricingResponse{
		Object:   constants.ObjectTypeAnalyzeCustomerPricingResponse,
		Findings: apiresource.NewList(presented, apiresource.PageInfo{}),
		Summary: apiresource.CustomerPricingSummary{
			Object:                 constants.ObjectTypeCustomerPricingSummary,
			PricesAnalyzed:         int(resp.GetPricesAnalyzed()),
			BelowPeerMedianCount:   int(resp.GetBelowPeerMedianCount()),
			BelowTargetMarginCount: int(resp.GetBelowTargetMarginCount()),
			MarginNotAssessedCount: int(resp.GetMarginNotAssessedCount()),
			Notes:                  notes,
		},
	}
}
