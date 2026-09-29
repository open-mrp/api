package grpc

import (
	"context"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/pagination"
	pb "github.com/open-mrp/api/shared/proto/core"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func salesReportFilterFromProto(p *pb.SalesReportFilterProto) domain.SalesReportFilter {
	if p == nil {
		return domain.SalesReportFilter{}
	}
	f := domain.SalesReportFilter{
		CustomerIDs:      p.CustomerIds,
		CustomerGroupIDs: p.CustomerGroupIds,
		ProductLineIDs:   p.ProductLineIds,
		SalesRepIDs:      p.SalesRepIds,
		ItemIDs:          p.ItemIds,
	}
	if p.StartsAt != nil {
		f.StartsAt = p.StartsAt.AsTime()
	}
	if p.EndsAt != nil {
		f.EndsAt = p.EndsAt.AsTime()
	}
	if p.ComparisonStartsAt != nil && p.ComparisonEndsAt != nil {
		start, end := p.ComparisonStartsAt.AsTime(), p.ComparisonEndsAt.AsTime()
		f.ComparisonStartsAt, f.ComparisonEndsAt = &start, &end
	}
	return f
}

func pageInfoToProto(pi pagination.PageInfo) *pb.PageInfo {
	return &pb.PageInfo{
		NextCursor:  pi.NextCursor,
		PrevCursor:  pi.PrevCursor,
		HasNextPage: pi.HasNextPage,
		HasPrevPage: pi.HasPrevPage,
	}
}

func salesTotalsToProto(t domain.SalesTotals) *pb.SalesTotalsProto {
	out := &pb.SalesTotalsProto{
		Invoiced:     t.Invoiced,
		Cost:         t.Cost,
		Quantity:     t.Quantity,
		InvoiceCount: t.InvoiceCount,
		LineCount:    t.LineCount,
	}
	if t.PeriodStart != nil {
		out.PeriodStart = timestamppb.New(*t.PeriodStart)
	}
	return out
}

func salesTotalsListToProto(ts []domain.SalesTotals) []*pb.SalesTotalsProto {
	out := make([]*pb.SalesTotalsProto, len(ts))
	for i, t := range ts {
		out[i] = salesTotalsToProto(t)
	}
	return out
}

func (h *gRPCHandler) AnalyzeSalesSummary(ctx context.Context, req *pb.AnalyzeSalesSummaryRequest) (*pb.AnalyzeSalesSummaryResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	summary, apiErr := h.analyticsSvc.AnalyzeSalesSummary(ctx, domain.AnalyzeSalesSummaryParams{
		SalesReportFilter: salesReportFilterFromProto(req.Filter),
		TZOffsetMinutes:   req.TzOffsetMinutes,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	resp := &pb.AnalyzeSalesSummaryResponse{
		Current:         salesTotalsToProto(summary.Current),
		Daily:           salesTotalsListToProto(summary.Daily),
		ComparisonDaily: salesTotalsListToProto(summary.ComparisonDaily),
	}
	if summary.Comparison != nil {
		resp.Comparison = salesTotalsToProto(*summary.Comparison)
	}
	return resp, nil
}

func (h *gRPCHandler) AnalyzeSalesBreakdown(ctx context.Context, req *pb.AnalyzeSalesBreakdownRequest) (*pb.AnalyzeSalesBreakdownResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	breakdown, apiErr := h.analyticsSvc.AnalyzeSalesBreakdown(ctx, domain.AnalyzeSalesBreakdownParams{
		SalesReportFilter: salesReportFilterFromProto(req.Filter),
		GroupBy:           constants.SalesBreakdownGroupBy(req.GroupBy),
		Limit:             req.Limit,
		Cursor:            req.Cursor,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	resp := &pb.AnalyzeSalesBreakdownResponse{
		Groups:   make([]*pb.SalesBreakdownGroupProto, len(breakdown.Groups)),
		PageInfo: pageInfoToProto(breakdown.PageInfo),
	}
	for i, g := range breakdown.Groups {
		group := &pb.SalesBreakdownGroupProto{
			Key:              g.Key,
			Label:            g.Label,
			Description:      g.Description,
			UnitAbbreviation: g.UnitAbbreviation,
			Totals:           salesTotalsToProto(g.Totals),
		}
		if g.Comparison != nil {
			group.Comparison = salesTotalsToProto(*g.Comparison)
		}
		resp.Groups[i] = group
	}
	return resp, nil
}

func (h *gRPCHandler) AnalyzeSalesInvoices(ctx context.Context, req *pb.AnalyzeSalesInvoicesRequest) (*pb.AnalyzeSalesInvoicesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	page, apiErr := h.analyticsSvc.AnalyzeSalesInvoices(ctx, domain.AnalyzeSalesInvoicesParams{
		SalesReportFilter: salesReportFilterFromProto(req.Filter),
		Limit:             req.Limit,
		Cursor:            req.Cursor,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	resp := &pb.AnalyzeSalesInvoicesResponse{
		Invoices: make([]*pb.SalesInvoiceSummaryProto, len(page.Invoices)),
		PageInfo: pageInfoToProto(page.PageInfo),
	}
	for i, inv := range page.Invoices {
		resp.Invoices[i] = &pb.SalesInvoiceSummaryProto{
			InvoiceId:     inv.InvoiceID,
			InvoiceNumber: inv.InvoiceNumber,
			CustomerId:    inv.CustomerID,
			CustomerName:  inv.CustomerName,
			InvoicedAt:    timestamppb.New(inv.InvoicedAt),
			ItemCount:     inv.ItemCount,
			Invoiced:      inv.Invoiced,
		}
	}
	return resp, nil
}

func (h *gRPCHandler) ListSalesLines(ctx context.Context, req *pb.ListSalesLinesRequest) (*pb.ListSalesLinesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	page, apiErr := h.analyticsSvc.ListSalesLines(ctx, domain.ListSalesLinesParams{
		SalesReportFilter: salesReportFilterFromProto(req.Filter),
		HasWindow:         req.HasWindow,
		Limit:             req.Limit,
		Cursor:            req.Cursor,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	resp := &pb.ListSalesLinesResponse{
		Lines:    make([]*pb.SalesEntryProto, len(page.Lines)),
		PageInfo: pageInfoToProto(page.PageInfo),
	}
	for i, e := range page.Lines {
		resp.Lines[i] = salesEntryToProto(e)
	}
	return resp, nil
}

func (h *gRPCHandler) ExportSalesLines(ctx context.Context, req *pb.ExportSalesLinesRequest) (*pb.ExportSalesLinesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	job, apiErr := h.analyticsSvc.ExportSalesLines(ctx, domain.ExportSalesLinesParams{
		SalesReportFilter: salesReportFilterFromProto(req.Filter),
		HasWindow:         req.HasWindow,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.ExportSalesLinesResponse{Job: jobToProto(job)}, nil
}
