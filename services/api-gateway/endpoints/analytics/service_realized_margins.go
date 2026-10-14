package analyticsep

import (
	"context"
	"github.com/open-mrp/apikit/include"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
)

// AnalyzeRealizedMargins audits invoiced sales for thin margins and prices well below what other customers achieved.
//
// The roll-up happens in core-service: a year of trading across thousands of customers and SKUs is hundreds of thousands of invoiced lines, and pulling them here to group them in memory is what made this time out.
func (m *analyticsSvcImpl) AnalyzeRealizedMargins(ctx context.Context, req *AnalyzeRealizedMarginsRequest) (*apiresource.AnalyzeRealizedMarginsResponse, *apierror.APIError) {
	if req.EndDate.Before(req.StartDate) {
		return nil, apierror.NewFieldError("ends_at", apierror.CodeInvalidFormat, "Must not be before starts_at.")
	}

	pbReq := &pb.AnalyzeRealizedMarginsRequest{
		StartDate:         timestamppb.New(req.StartDate),
		EndDate:           timestamppb.New(req.EndDate),
		CustomerIds:       req.CustomerIDs,
		CustomerGroupIds:  req.CustomerGroupIDs,
		ProductLineIds:    req.ProductLineIDs,
		TargetGrossMargin: req.TargetGrossMargin.Ptr(),
		OutlierTolerance:  req.OutlierTolerance.Ptr(),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, analyticsSvcTracer, "service.analytics.realized_margins", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.AnalyzeRealizedMarginsResponse, error) {
			return m.coreClient.AnalyzeRealizedMargins(ctx, pbReq, opts...)
		}, grpcutil.WithTimeout(grpcutil.AnalyticsOperationTimeout))
	if apiErr != nil {
		return nil, apiErr
	}

	return presentRealizedMargins(ctx, resp), nil
}

// presentRealizedMargins builds the response, leaving every expandable field nil and stashing the foreign keys the include resolver needs.
func presentRealizedMargins(ctx context.Context, resp *pb.AnalyzeRealizedMarginsResponse) *apiresource.AnalyzeRealizedMarginsResponse {
	meta := include.GetLoadMeta(ctx)
	presented := make([]apiresource.RealizedMarginFinding, 0, len(resp.GetFindings()))

	for _, f := range resp.GetFindings() {
		id := f.GetCustomerId() + ":" + f.GetItemId()

		meta.Set(constants.ObjectTypeRealizedMarginFinding, id, "customer_id", f.GetCustomerId())
		meta.Set(constants.ObjectTypeRealizedMarginFinding, id, "customer_group_id", f.GetCustomerGroupId())
		meta.Set(constants.ObjectTypeRealizedMarginFinding, id, "item_id", f.GetItemId())
		meta.Set(constants.ObjectTypeRealizedMarginFinding, id, "product_line_id", f.GetProductLineId())

		item := apiresource.RealizedMarginFinding{
			ID:               id,
			Object:           constants.ObjectTypeRealizedMarginFinding,
			Reason:           constants.PricingFindingReason(f.GetReason()),
			QuantityInvoiced: analyticsQuantity(f.GetQuantityInvoiced(), f.GetUnitId()),
			Revenue:          apiresource.NewMoney(f.GetRevenue()),
			Cost:             apiresource.NewMoney(f.GetCost()),
			AverageUnitPrice: analyticsPrice(f.GetAverageUnitPrice(), f.GetUnitId()),
			LineCount:        int(f.GetLineCount()),
		}
		if f.PeerMedianPrice != nil {
			item.PeerMedianPrice = analyticsPrice(f.GetPeerMedianPrice(), f.GetUnitId())
			item.BelowPeerMedianFraction = f.BelowPeerMedianFraction
		}
		item.GrossMargin = f.GrossMargin
		presented = append(presented, item)
	}

	return &apiresource.AnalyzeRealizedMarginsResponse{
		Object:   constants.ObjectTypeAnalyzeRealizedMarginsResponse,
		Findings: apiresource.NewList(presented, apiresource.PageInfo{}),
		Summary: apiresource.RealizedMarginSummary{
			Object:                 constants.ObjectTypeRealizedMarginSummary,
			LinesAnalyzed:          int(resp.GetLinesAnalyzed()),
			RelationshipsAnalyzed:  int(resp.GetRelationshipsAnalyzed()),
			BelowPeerMedianCount:   int(resp.GetBelowPeerMedianCount()),
			BelowTargetMarginCount: int(resp.GetBelowTargetMarginCount()),
			MarginNotAssessedCount: int(resp.GetMarginNotAssessedCount()),
			Notes:                  []string{},
		},
	}
}
