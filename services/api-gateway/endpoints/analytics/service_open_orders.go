package analyticsep

import (
	"context"

	"google.golang.org/grpc"

	jobep "github.com/open-mrp/api/services/api-gateway/endpoints/jobs"
	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
)

func (m *analyticsSvcImpl) AnalyzeOpenOrdersSummary(ctx context.Context, req *AnalyzeOpenOrdersSummaryRequest) (*apiresource.AnalyzeOpenOrdersSummaryResponse, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, analyticsSvcTracer, "service.analytics.analyze_open_orders_summary", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.AnalyzeOpenOrdersSummaryResponse, error) {
			return m.coreClient.AnalyzeOpenOrdersSummary(ctx, &pb.AnalyzeOpenOrdersSummaryRequest{Filter: req.OpenOrderFilters.toProto()}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return &apiresource.AnalyzeOpenOrdersSummaryResponse{
		Object:      constants.ObjectTypeOpenOrdersSummary,
		Ordered:     salesAmount(resp.GetOrdered(), nil),
		BackOrdered: salesAmount(resp.GetBackOrdered(), nil),
		Invoiced:    salesAmount(resp.GetInvoiced(), nil),
	}, nil
}

func (m *analyticsSvcImpl) AnalyzeOpenOrdersBreakdown(ctx context.Context, req *AnalyzeOpenOrdersBreakdownRequest) (*apiresource.List[apiresource.OpenOrderProduct], *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, analyticsSvcTracer, "service.analytics.analyze_open_order_products", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.AnalyzeOpenOrderProductsResponse, error) {
			return m.coreClient.AnalyzeOpenOrderProducts(ctx, &pb.AnalyzeOpenOrderProductsRequest{
				Filter: req.OpenOrderFilters.toProto(),
				Limit:  req.Limit,
				Cursor: req.Cursor,
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	products := make([]apiresource.OpenOrderProduct, len(resp.GetProducts()))
	for i, p := range resp.GetProducts() {
		unit := reportUnit(p.Unit)
		products[i] = apiresource.OpenOrderProduct{
			Object:              constants.ObjectTypeOpenOrderProduct,
			Item:                &apiresource.AnalyticsItem{ID: p.GetItemId(), Object: constants.ObjectTypeItem, Sku: p.GetSku(), Description: p.Description},
			Unit:                unit,
			QuantityOrdered:     openOrderQuantity(p.GetQuantityOrdered(), unit),
			QuantityBackOrdered: openOrderQuantity(p.GetQuantityBackOrdered(), unit),
			QuantityInvoiced:    openOrderQuantity(p.GetQuantityInvoiced(), unit),
		}
	}
	return apiresource.NewList(products, grpcutil.MapProtoPageInfo(ctx, resp.GetPageInfo())), nil
}

func (m *analyticsSvcImpl) ListOpenOrders(ctx context.Context, req *ListOpenOrdersRequest) (*apiresource.List[apiresource.OpenOrder], *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, analyticsSvcTracer, "service.analytics.list_open_orders", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListOpenOrdersResponse, error) {
			return m.coreClient.ListOpenOrders(ctx, &pb.ListOpenOrdersRequest{
				Filter: req.OpenOrderFilters.toProto(),
				Limit:  req.Limit,
				Cursor: req.Cursor,
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	orders := make([]apiresource.OpenOrder, len(resp.GetOrders()))
	for i, o := range resp.GetOrders() {
		orders[i] = apiresource.OpenOrder{
			Object:       constants.ObjectTypeOpenOrder,
			Order:        &apiresource.AnalyticsSalesOrder{ID: o.GetId(), Object: constants.ObjectTypeSalesOrder, Number: o.GetNumber()},
			Status:       constants.SalesOrderStatusCode(o.GetStatus()),
			IssuedAt:     grpcutil.TimestampToTime(o.GetIssuedAt()),
			Customer:     &apiresource.AnalyticsCustomer{ID: o.GetCustomerId(), Object: constants.ObjectTypeCustomer, Name: o.GetCustomerName(), Number: o.CustomerNumber},
			ShipTo:       &apiresource.AnalyticsShipTo{State: o.ShipToState, Country: o.ShipToCountry},
			LineCount:    o.GetLineCount(),
			TotalOrdered: salesAmount(o.GetTotalOrdered(), nil),
		}
	}
	return apiresource.NewList(orders, grpcutil.MapProtoPageInfo(ctx, resp.GetPageInfo())), nil
}

func (m *analyticsSvcImpl) ListOpenOrderLines(ctx context.Context, req *ListOpenOrderLinesRequest) (*apiresource.List[apiresource.OpenOrderLine], *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, analyticsSvcTracer, "service.analytics.list_open_order_lines", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListOpenOrderLinesResponse, error) {
			return m.coreClient.ListOpenOrderLines(ctx, &pb.ListOpenOrderLinesRequest{OrderId: req.SalesOrderID}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	lines := make([]apiresource.OpenOrderLine, len(resp.GetLines()))
	for i, l := range resp.GetLines() {
		unit := reportUnit(l.Unit)
		lines[i] = apiresource.OpenOrderLine{
			Object: constants.ObjectTypeOpenOrderLine,
			ID:     l.GetId(),
			Item:   &apiresource.AnalyticsItem{ID: l.GetItemId(), Object: constants.ObjectTypeItem, Sku: l.GetSku(), Description: l.Description},
			Unit:   unit,
			UnitPrice: &apiresource.ComputedRate{
				Object:       constants.ObjectTypeComputedRate,
				Value:        l.GetUnitPrice(),
				DisplayValue: apiresource.FormatRateDisplay(l.GetUnitPrice(), l.GetUnitPriceNumeratorAbbreviation(), l.GetUnitPriceDenominatorAbbreviation()),
			},
			QuantityBackOrdered: openOrderQuantity(l.GetQuantityBackOrdered(), unit),
			QuantityInvoiced:    openOrderQuantity(l.GetQuantityInvoiced(), unit),
			TotalOrdered:        salesAmount(l.GetTotalOrdered(), nil),
		}
	}
	return apiresource.NewList(lines, apiresource.PageInfo{}), nil
}

func (m *analyticsSvcImpl) ExportOpenOrderLines(ctx context.Context, req *ExportOpenOrderLinesRequest) (*apiresource.Job, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, analyticsSvcTracer, "service.analytics.export_open_order_lines", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ExportOpenOrderLinesResponse, error) {
			return m.coreClient.ExportOpenOrderLines(ctx, &pb.ExportOpenOrderLinesRequest{Filter: req.OpenOrderFilters.toProto()}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return jobep.JobFromProto(resp.GetJob()), nil
}

// openOrderQuantity renders a quantity in the unit it is counted in, which travels with it.
func openOrderQuantity(value string, unit *apiresource.Unit) *apiresource.ComputedQuantity {
	var abbr *string
	if unit != nil {
		abbr = &unit.Abbreviation
	}
	q := salesQuantity(value, abbr)
	q.Unit = unit
	return q
}

// reportUnit renders the unit a report's quantities are counted in, which core attaches to the report: a caller who may read the report need not also be allowed to browse units. Its owner is left to the include resolver.
func reportUnit(u *pb.UnitInfo) *apiresource.Unit {
	if u == nil {
		return nil
	}
	return &apiresource.Unit{
		ID:                u.GetId(),
		Object:            constants.ObjectTypeUnit,
		Name:              u.GetName(),
		Abbreviation:      u.GetAbbreviation(),
		Type:              constants.UnitType(u.GetType()),
		RatioNumerator:    db.TrimDecimal(u.GetRatioNumerator()),
		RatioDenominator:  db.TrimDecimal(u.GetRatioDenominator()),
		OffsetNumerator:   db.TrimDecimal(u.GetOffsetNumerator()),
		OffsetDenominator: db.TrimDecimal(u.GetOffsetDenominator()),
		IsBaseUnit:        u.GetIsBaseUnit(),
		CreatedAt:         grpcutil.TimestampToTime(u.GetCreatedAt()),
		UpdatedAt:         grpcutil.TimestampToTime(u.GetUpdatedAt()),
	}
}
