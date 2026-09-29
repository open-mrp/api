package analyticsep

import (
	"context"

	"github.com/shopspring/decimal"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	jobep "github.com/open-mrp/api/services/api-gateway/endpoints/jobs"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
)

func salesReportFilterToProto(startsAt, endsAt *timestamppb.Timestamp, cmp *SalesComparisonPeriod, f SalesReportFilters) (*pb.SalesReportFilterProto, *apierror.APIError) {
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
			return nil, apierror.NewParameterInvalidError("comparison_starts_at and comparison_ends_at must be set together.", "comparison_starts_at")
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
		return nil, apierror.NewParameterInvalidError("group_by is not a supported dimension.", "group_by")
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
			Revenue:      salesAmount(inv.GetInvoiced(), nil),
		}
	}
	return apiresource.NewList(invoices, grpcutil.MapProtoPageInfo(ctx, resp.GetPageInfo())), nil
}

func (m *analyticsSvcImpl) ListSalesLines(ctx context.Context, req *ListSalesLinesRequest) (*apiresource.List[apiresource.SalesEntry], *apierror.APIError) {
	start, hasStart := req.StartDate.Value()
	end, hasEnd := req.EndDate.Value()
	if hasStart != hasEnd {
		return nil, apierror.NewParameterInvalidError("starts_at and ends_at must be set together.", "starts_at")
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

func (m *analyticsSvcImpl) ExportSalesLines(ctx context.Context, req *ExportSalesLinesRequest) (*apiresource.Job, *apierror.APIError) {
	start, hasStart := req.StartDate.Value()
	end, hasEnd := req.EndDate.Value()
	if hasStart != hasEnd {
		return nil, apierror.NewParameterInvalidError("starts_at and ends_at must be set together.", "starts_at")
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
		Revenue:          salesAmount(t.GetInvoiced(), nil),
		QuantityInvoiced: salesQuantity(t.GetQuantity(), unitAbbr),
		InvoiceCount:     t.GetInvoiceCount(),
		LineCount:        t.GetLineCount(),
	}
	if t.Cost != nil {
		out.Cost = salesAmount(*t.Cost, nil)
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

// salesAmount renders an exact money decimal as a computed quantity. value keeps every digit core-service summed, so a caller adding or comparing figures sees no rounding; display_value is rounded to cents with thousands separators.
func salesAmount(value string, unitAbbr *string) *apiresource.ComputedQuantity {
	return computedDecimal(value, unitAbbr, func(d decimal.Decimal) string { return d.StringFixed(2) })
}

// salesQuantity renders a quantity the same way, but displays only its significant digits: 17 pairs reads "17 pr", not "17.00 pr".
func salesQuantity(value string, unitAbbr *string) *apiresource.ComputedQuantity {
	return computedDecimal(value, unitAbbr, func(d decimal.Decimal) string { return d.Round(4).String() })
}

func computedDecimal(value string, unitAbbr *string, format func(decimal.Decimal) string) *apiresource.ComputedQuantity {
	amount, err := decimal.NewFromString(value)
	if err != nil {
		amount, value = decimal.Zero, "0"
	}
	display := groupThousands(format(amount))
	if unitAbbr != nil && *unitAbbr != "" {
		display += " " + *unitAbbr
	}
	return &apiresource.ComputedQuantity{
		Object:       constants.ObjectTypeComputedQuantity,
		Value:        value,
		DisplayValue: display,
	}
}

// groupThousands inserts thousands separators into a plain fixed-point number.
func groupThousands(fixed string) string {
	sign := ""
	if len(fixed) > 0 && fixed[0] == '-' {
		sign, fixed = "-", fixed[1:]
	}
	intPart, frac := fixed, ""
	for i := range len(fixed) {
		if fixed[i] == '.' {
			intPart, frac = fixed[:i], fixed[i:]
			break
		}
	}
	var b []byte
	for i := range len(intPart) {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b = append(b, ',')
		}
		b = append(b, intPart[i])
	}
	return sign + string(b) + frac
}
