package grpc

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/contracts"
	pb "github.com/open-mrp/api/shared/proto/core"
)

func openOrderFilterFromProto(p *pb.OpenOrderFilterProto) domain.OpenOrderFilter {
	if p == nil {
		return domain.OpenOrderFilter{}
	}
	return domain.OpenOrderFilter{
		CustomerIDs:      p.CustomerIds,
		CustomerGroupIDs: p.CustomerGroupIds,
		ProductLineIDs:   p.ProductLineIds,
		SalesRepIDs:      p.SalesRepIds,
		ItemIDs:          p.ItemIds,
	}
}

func (h *gRPCHandler) AnalyzeOpenOrdersSummary(ctx context.Context, req *pb.AnalyzeOpenOrdersSummaryRequest) (*pb.AnalyzeOpenOrdersSummaryResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	summary, apiErr := h.analyticsSvc.AnalyzeOpenOrdersSummary(ctx, openOrderFilterFromProto(req.Filter))
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.AnalyzeOpenOrdersSummaryResponse{
		Ordered:     summary.Ordered,
		BackOrdered: summary.BackOrdered,
		Invoiced:    summary.Invoiced,
	}, nil
}

func (h *gRPCHandler) AnalyzeOpenOrderProducts(ctx context.Context, req *pb.AnalyzeOpenOrderProductsRequest) (*pb.AnalyzeOpenOrderProductsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	page, apiErr := h.analyticsSvc.AnalyzeOpenOrderProducts(ctx, domain.AnalyzeOpenOrderProductsParams{
		OpenOrderFilter: openOrderFilterFromProto(req.Filter),
		Limit:           req.Limit,
		Cursor:          req.Cursor,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	resp := &pb.AnalyzeOpenOrderProductsResponse{
		Products: make([]*pb.OpenOrderProductProto, len(page.Products)),
		PageInfo: pageInfoToProto(page.PageInfo),
	}
	for i, p := range page.Products {
		resp.Products[i] = &pb.OpenOrderProductProto{
			ItemId:              p.ItemID,
			Sku:                 p.Sku,
			Description:         p.Description,
			UnitId:              p.UnitID,
			QuantityOrdered:     p.QuantityOrdered,
			QuantityBackOrdered: p.QuantityBackOrdered,
			QuantityInvoiced:    p.QuantityInvoiced,
		}
		if p.Unit != nil {
			resp.Products[i].Unit = unitToProto(p.Unit)
		}
	}
	return resp, nil
}

func (h *gRPCHandler) ListOpenOrders(ctx context.Context, req *pb.ListOpenOrdersRequest) (*pb.ListOpenOrdersResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	page, apiErr := h.analyticsSvc.ListOpenOrders(ctx, domain.ListOpenOrdersParams{
		OpenOrderFilter: openOrderFilterFromProto(req.Filter),
		Limit:           req.Limit,
		Cursor:          req.Cursor,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	resp := &pb.ListOpenOrdersResponse{
		Orders:   make([]*pb.OpenOrderProto, len(page.Orders)),
		PageInfo: pageInfoToProto(page.PageInfo),
	}
	for i, o := range page.Orders {
		resp.Orders[i] = &pb.OpenOrderProto{
			Id:             o.ID,
			Number:         o.Number,
			Status:         o.Status,
			IssuedAt:       timestamppb.New(o.IssuedAt),
			CustomerId:     o.CustomerID,
			CustomerName:   o.CustomerName,
			CustomerNumber: o.CustomerNumber,
			ShipToState:    o.ShipToState,
			ShipToCountry:  o.ShipToCountry,
			LineCount:      o.LineCount,
			TotalOrdered:   o.TotalOrdered,
		}
	}
	return resp, nil
}

func (h *gRPCHandler) ListOpenOrderLines(ctx context.Context, req *pb.ListOpenOrderLinesRequest) (*pb.ListOpenOrderLinesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	lines, apiErr := h.analyticsSvc.ListOpenOrderLines(ctx, req.OrderId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	resp := &pb.ListOpenOrderLinesResponse{Lines: make([]*pb.OpenOrderLineProto, len(lines))}
	for i, l := range lines {
		resp.Lines[i] = &pb.OpenOrderLineProto{
			Id:                               l.ID,
			ItemId:                           l.ItemID,
			Sku:                              l.Sku,
			Description:                      l.Description,
			UnitId:                           l.UnitID,
			UnitPrice:                        l.UnitPrice,
			UnitPriceNumeratorUnitId:         l.UnitPriceNumeratorUnitID,
			UnitPriceNumeratorAbbreviation:   l.UnitPriceNumeratorAbbr,
			UnitPriceDenominatorUnitId:       l.UnitPriceDenominatorUnitID,
			UnitPriceDenominatorAbbreviation: l.UnitPriceDenominatorAbbr,
			QuantityBackOrdered:              l.QuantityBackOrdered,
			QuantityInvoiced:                 l.QuantityInvoiced,
			TotalOrdered:                     l.TotalOrdered,
		}
		if l.Unit != nil {
			resp.Lines[i].Unit = unitToProto(l.Unit)
		}
	}
	return resp, nil
}

func (h *gRPCHandler) ExportOpenOrderLines(ctx context.Context, req *pb.ExportOpenOrderLinesRequest) (*pb.ExportOpenOrderLinesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	job, apiErr := h.analyticsSvc.ExportOpenOrderLines(ctx, domain.ExportOpenOrderLinesParams{OpenOrderFilter: openOrderFilterFromProto(req.Filter)})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.ExportOpenOrderLinesResponse{Job: jobToProto(job)}, nil
}
