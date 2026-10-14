package analyticsep

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	jobep "github.com/open-mrp/api/services/api-gateway/endpoints/jobs"
	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
)

func salesReportFilterToProto(startsAt, endsAt *timestamppb.Timestamp, cmp *SalesComparisonPeriod, f SalesReportFilters) (*pb.SalesReportFilterProto, *apierror.APIError) {
	if startsAt != nil && endsAt != nil && endsAt.AsTime().Before(startsAt.AsTime()) {
		return nil, apierror.NewFieldError("ends_at", apierror.CodeInvalidFormat, "Must not be before starts_at.")
	}
	out := &pb.SalesReportFilterProto{
		StartsAt:         startsAt,
		EndsAt:           endsAt,
		CustomerIds:      f.CustomerIDs,
		CustomerGroupIds: f.CustomerGroupIDs,
		ProductLineIds:   f.ProductLineIDs,
		SalesRepIds:      f.SalesRepIDs,
		ItemIds:          f.ItemIDs,
	}
	if cmp != nil {
		start, hasStart := cmp.ComparisonStartDate.Value()
		end, hasEnd := cmp.ComparisonEndDate.Value()
		if hasStart != hasEnd {
			return nil, apierror.NewParameterInvalidError("comparison_starts_at", "comparison_starts_at and comparison_ends_at must be set together.")
		}
		if hasStart && end.Before(start) {
			return nil, apierror.NewFieldError("comparison_ends_at", apierror.CodeInvalidFormat, "Must not be before comparison_starts_at.")
		}
		if hasStart {
			out.ComparisonStartsAt = timestamppb.New(start)
			out.ComparisonEndsAt = timestamppb.New(end)
		}
	}
	return out, nil
}

func (m *analyticsSvcImpl) AnalyzeSalesSummary(ctx context.Context, req *AnalyzeSalesSummaryRequest) (*apiresource.AnalyzeSalesSummaryResponse, *apierror.APIError) {
	filter, apiErr := salesReportFilterToProto(timestamppb.New(req.StartDate), timestamppb.New(req.EndDate), &req.SalesComparisonPeriod, req.SalesReportFilters)
	if apiErr != nil {
		return nil, apiErr
	}
	tz, _ := req.TZOffsetMinutes.Value()
	resp, apiErr := grpcutil.CallRPC(ctx, analyticsSvcTracer, "service.analytics.analyze_sales_summary", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.AnalyzeSalesSummaryResponse, error) {
			return m.coreClient.AnalyzeSalesSummary(ctx, &pb.AnalyzeSalesSummaryRequest{Filter: filter, TzOffsetMinutes: tz}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	out := &apiresource.AnalyzeSalesSummaryResponse{
		Object:  constants.ObjectTypeAnalyzeSalesSummaryResponse,
		Overall: salesTotalsFromProto(resp.GetCurrent(), nil),
		Periods: salesTotalsListFromProto(resp.GetDaily()),
	}
	if resp.Comparison != nil {
		out.Comparison = salesTotalsFromProto(resp.Comparison, nil)
		out.ComparisonPeriods = salesTotalsListFromProto(resp.GetComparisonDaily())
	}
	return out, nil
}

func (m *analyticsSvcImpl) AnalyzeSalesBreakdown(ctx context.Context, req *AnalyzeSalesBreakdownRequest) (*apiresource.List[apiresource.SalesBreakdown], *apierror.APIError) {
	if !req.GroupBy.IsValid() {
		return nil, apierror.NewParameterInvalidError("group_by", "group_by is not a supported dimension.")
	}
	filter, apiErr := salesReportFilterToProto(timestamppb.New(req.StartDate), timestamppb.New(req.EndDate), &req.SalesComparisonPeriod, req.SalesReportFilters)
	if apiErr != nil {
		return nil, apiErr
	}
	resp, apiErr := grpcutil.CallRPC(ctx, analyticsSvcTracer, "service.analytics.analyze_sales_breakdown", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.AnalyzeSalesBreakdownResponse, error) {
			return m.coreClient.AnalyzeSalesBreakdown(ctx, &pb.AnalyzeSalesBreakdownRequest{
				Filter:  filter,
				GroupBy: string(req.GroupBy),
				Limit:   req.Limit,
				Cursor:  req.Cursor,
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	groups := make([]apiresource.SalesBreakdown, len(resp.GetGroups()))
	for i, g := range resp.GetGroups() {
		unit := ""
		if g.UnitAbbreviation != nil {
			unit = *g.UnitAbbreviation
		}
		groups[i] = apiresource.SalesBreakdown{
			Object:           constants.ObjectTypeSalesBreakdown,
			Key:              g.GetKey(),
			Label:            g.GetLabel(),
			Description:      g.Description,
			UnitAbbreviation: g.UnitAbbreviation,
			Totals:           salesTotalsFromProto(g.GetTotals(), &unit),
		}
		if g.Comparison != nil {
			groups[i].ComparisonTotals = salesTotalsFromProto(g.Comparison, &unit)
		}
	}
	return apiresource.NewList(groups, grpcutil.MapProtoPageInfo(ctx, resp.GetPageInfo())), nil
}

func (m *analyticsSvcImpl) AnalyzeSalesInvoices(ctx context.Context, req *AnalyzeSalesInvoicesRequest) (*apiresource.List[apiresource.SalesInvoice], *apierror.APIError) {
	filter, apiErr := salesReportFilterToProto(timestamppb.New(req.StartDate), timestamppb.New(req.EndDate), nil, req.SalesReportFilters)
	if apiErr != nil {
		return nil, apiErr
	}
	resp, apiErr := grpcutil.CallRPC(ctx, analyticsSvcTracer, "service.analytics.analyze_sales_invoices", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.AnalyzeSalesInvoicesResponse, error) {
			return m.coreClient.AnalyzeSalesInvoices(ctx, &pb.AnalyzeSalesInvoicesRequest{Filter: filter, Limit: req.Limit, Cursor: req.Cursor}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	invoices := make([]apiresource.SalesInvoice, len(resp.GetInvoices()))
	for i, inv := range resp.GetInvoices() {
		invoices[i] = apiresource.SalesInvoice{
			Object:       constants.ObjectTypeSalesInvoice,
			ID:           inv.GetInvoiceId(),
			Number:       inv.GetInvoiceNumber(),
			CustomerID:   inv.GetCustomerId(),
			CustomerName: inv.GetCustomerName(),
			InvoicedAt:   grpcutil.TimestampToTime(inv.GetInvoicedAt()),
			ItemCount:    inv.GetItemCount(),
			Revenue:      apiresource.NewMoney(inv.GetInvoiced()),
		}
	}
	return apiresource.NewList(invoices, grpcutil.MapProtoPageInfo(ctx, resp.GetPageInfo())), nil
}

func (m *analyticsSvcImpl) ListNewCustomers(ctx context.Context, req *ListNewCustomersRequest) (*apiresource.List[apiresource.NewCustomer], *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, analyticsSvcTracer, "service.analytics.list_new_customers", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListNewCustomersResponse, error) {
			return m.coreClient.ListNewCustomers(ctx, &pb.ListNewCustomersRequest{
				StartsAt:         timestamppb.New(req.StartDate),
				EndsAt:           timestamppb.New(req.EndDate),
				CustomerGroupIds: req.CustomerGroupIDs,
				SalesRepIds:      req.SalesRepIDs,
				Limit:            req.Limit,
				Cursor:           req.Cursor,
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	customers := make([]apiresource.NewCustomer, len(resp.GetCustomers()))
	for i, c := range resp.GetCustomers() {
		customers[i] = apiresource.NewCustomer{
			Object:            constants.ObjectTypeNewCustomer,
			ID:                c.GetCustomerId(),
			Number:            c.GetCustomerNumber(),
			Name:              c.GetCustomerName(),
			CustomerGroupName: c.CustomerGroupName,
			Location:          c.Location,
			SalesRepName:      c.SalesRepName,
			LifetimeRevenue:   apiresource.NewMoney(c.GetTotalInvoiced()),
			FirstOrderedAt:    grpcutil.TimestampToTime(c.GetFirstOrderedAt()),
			AddedAt:           grpcutil.TimestampToTime(c.GetCustomerCreatedAt()),
		}
	}
	return apiresource.NewList(customers, grpcutil.MapProtoPageInfo(ctx, resp.GetPageInfo())), nil
}

func (m *analyticsSvcImpl) ListSalesLines(ctx context.Context, req *ListSalesLinesRequest) (*apiresource.List[apiresource.SalesEntry], *apierror.APIError) {
	start, hasStart := req.StartDate.Value()
	end, hasEnd := req.EndDate.Value()
	if hasStart != hasEnd {
		return nil, apierror.NewParameterInvalidError("starts_at", "starts_at and ends_at must be set together.")
	}
	var startsAt, endsAt *timestamppb.Timestamp
	if hasStart {
		startsAt, endsAt = timestamppb.New(start), timestamppb.New(end)
	}
	filter, apiErr := salesReportFilterToProto(startsAt, endsAt, nil, req.SalesReportFilters)
	if apiErr != nil {
		return nil, apiErr
	}
	resp, apiErr := grpcutil.CallRPC(ctx, analyticsSvcTracer, "service.analytics.list_sales_lines", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListSalesLinesResponse, error) {
			return m.coreClient.ListSalesLines(ctx, &pb.ListSalesLinesRequest{Filter: filter, HasWindow: hasStart, Limit: req.Limit, Cursor: req.Cursor}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	lines := make([]apiresource.SalesEntry, len(resp.GetLines()))
	for i, l := range resp.GetLines() {
		lines[i] = salesEntryFromProto(l)
	}
	return apiresource.NewList(lines, grpcutil.MapProtoPageInfo(ctx, resp.GetPageInfo())), nil
}

func (m *analyticsSvcImpl) ExportSalesLines(ctx context.Context, req *ExportSalesLinesRequest) (*apiresource.AsyncJob, *apierror.APIError) {
	start, hasStart := req.StartDate.Value()
	end, hasEnd := req.EndDate.Value()
	if hasStart != hasEnd {
		return nil, apierror.NewParameterInvalidError("starts_at", "starts_at and ends_at must be set together.")
	}
	var startsAt, endsAt *timestamppb.Timestamp
	if hasStart {
		startsAt, endsAt = timestamppb.New(start), timestamppb.New(end)
	}
	filter, apiErr := salesReportFilterToProto(startsAt, endsAt, nil, req.SalesReportFilters)
	if apiErr != nil {
		return nil, apiErr
	}
	resp, apiErr := grpcutil.CallRPC(ctx, analyticsSvcTracer, "service.analytics.export_sales_lines", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ExportSalesLinesResponse, error) {
			return m.coreClient.ExportSalesLines(ctx, &pb.ExportSalesLinesRequest{Filter: filter, HasWindow: hasStart}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return jobep.JobFromProto(resp.GetJob()), nil
}

// salesTotalsFromProto renders totals; unitAbbr labels the quantity when every line in it shares one base unit.
func salesTotalsFromProto(t *pb.SalesTotalsProto, unitAbbr *string) *apiresource.SalesTotals {
	if t == nil {
		t = &pb.SalesTotalsProto{}
	}
	out := &apiresource.SalesTotals{
		Object:           constants.ObjectTypeSalesTotals,
		Revenue:          apiresource.NewMoney(t.GetInvoiced()),
		QuantityInvoiced: t.GetQuantity(),
		InvoiceCount:     t.GetInvoiceCount(),
		LineCount:        t.GetLineCount(),
	}
	if t.Cost != nil {
		out.Cost = apiresource.NewMoney(*t.Cost)
	}
	if t.PeriodStart != nil {
		ps := grpcutil.TimestampToTime(t.PeriodStart)
		out.PeriodStart = &ps
	}
	return out
}

func salesTotalsListFromProto(ts []*pb.SalesTotalsProto) *apiresource.List[apiresource.SalesTotals] {
	out := make([]apiresource.SalesTotals, len(ts))
	for i, t := range ts {
		out[i] = *salesTotalsFromProto(t, nil)
	}
	return apiresource.NewList(out, apiresource.PageInfo{})
}
