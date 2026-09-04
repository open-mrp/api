package grpc

import (
	"context"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/contracts"
	pb "github.com/open-mrp/api/shared/proto/core"

	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type receivingGRPCHandler struct {
	pb.UnimplementedCoreReceivingServiceServer

	receivingOrderSvc     domain.ReceivingOrderSvc
	receivingOrderLineSvc domain.ReceivingOrderLineSvc
}

func receivingOrderSummaryToProto(s *domain.ReceivingOrderSummary) *pb.ReceivingOrderSummaryInfo {
	if s == nil {
		return nil
	}

	info := &pb.ReceivingOrderSummaryInfo{
		Id:                  s.ID,
		Number:              s.Number,
		PurchaseOrderId:     s.PurchaseOrderID,
		PurchaseOrderNumber: s.PurchaseOrderNumber,
		PurchaseOrderStatus: s.PurchaseOrderStatus,
		LineCount:           s.LineCount,
		CreatedAt:           timestamppb.New(s.CreatedAt),
		UpdatedAt:           timestamppb.New(s.UpdatedAt),
		Totals:              receivingOrderTotalsToProto(s.Totals),
		Deliveries:          documentRefsToProto(s.Deliveries),
	}

	if s.SupplierID != nil {
		info.SupplierId = s.SupplierID
	}
	if s.SupplierName != nil {
		info.SupplierName = s.SupplierName
	}
	if s.SupplierNumber != nil {
		info.SupplierNumber = s.SupplierNumber
	}
	if s.CompletedAt != nil {
		info.CompletedAt = timestamppb.New(*s.CompletedAt)
	}

	for _, l := range s.Lines {
		info.Lines = append(info.Lines, receivingOrderLineToProto(l))
	}

	return info
}

func receivingOrderToProto(o *domain.ReceivingOrder) *pb.ReceivingOrderInfo {
	if o == nil {
		return nil
	}

	info := &pb.ReceivingOrderInfo{
		Id:                  o.ID,
		Number:              o.Number,
		PurchaseOrderId:     o.PurchaseOrderID,
		PurchaseOrderNumber: o.PurchaseOrderNumber,
		PurchaseOrderStatus: o.PurchaseOrderStatus,
		Totals:              receivingOrderTotalsToProto(o.Totals),
		Deliveries:          documentRefsToProto(o.Deliveries),
		CreatedAt:           timestamppb.New(o.CreatedAt),
		UpdatedAt:           timestamppb.New(o.UpdatedAt),
	}

	if o.SupplierID != nil {
		info.SupplierId = o.SupplierID
	}
	if o.SupplierName != nil {
		info.SupplierName = o.SupplierName
	}
	if o.SupplierNumber != nil {
		info.SupplierNumber = o.SupplierNumber
	}
	if o.Note != nil {
		info.Note = o.Note
	}
	if o.CompletedAt != nil {
		info.CompletedAt = timestamppb.New(*o.CompletedAt)
	}

	if o.Lines != nil {
		lines := make([]*pb.ReceivingOrderLineInfo, len(o.Lines))
		for i, l := range o.Lines {
			lines[i] = receivingOrderLineToProto(l)
		}
		info.Lines = lines
	}

	return info
}

func receivingOrderLineToProto(l *domain.ReceivingOrderLine) *pb.ReceivingOrderLineInfo {
	if l == nil {
		return nil
	}

	info := &pb.ReceivingOrderLineInfo{
		Id:                        l.ID,
		QuantityId:                l.QuantityID,
		QuantityValue:             l.QuantityValue,
		QuantityUnitId:            l.QuantityUnitID,
		QuantityUnitAbbreviation:  l.QuantityUnitAbbreviation,
		OrderLineId:               l.OrderLineID,
		OrderLineQuantityId:       l.OrderLineQuantityID,
		OrderLineQuantityOrdered:  l.OrderLineQuantityOrdered,
		OrderLineUnitId:           l.OrderLineUnitID,
		OrderLineUnitAbbreviation: l.OrderLineUnitAbbreviation,
		CreatedAt:                 timestamppb.New(l.CreatedAt),
		UpdatedAt:                 timestamppb.New(l.UpdatedAt),
	}

	if l.RejectedQuantityValue != nil {
		info.RejectedQuantityValue = l.RejectedQuantityValue
	}
	if l.OrderLineProductID != nil {
		info.OrderLineProductId = l.OrderLineProductID
	}
	info.OrderLineItemNumber = l.OrderLineItemNumber
	if l.OrderLineItemID != nil {
		info.OrderLineItemId = l.OrderLineItemID
	}
	if l.OrderLineItemSKU != nil {
		info.OrderLineItemSku = l.OrderLineItemSKU
	}
	if l.OrderLineItemDescription != nil {
		info.OrderLineItemDescription = l.OrderLineItemDescription
	}
	if l.StockedAt != nil {
		info.StockedAt = timestamppb.New(*l.StockedAt)
	}

	return info
}

func stockingDataFromProto(d *pb.StockingDataInfo) domain.StockingData {
	if d == nil {
		return domain.StockingData{}
	}

	lineItems := make([]domain.StockingLineItem, len(d.LineItems))
	for i, li := range d.LineItems {
		item := domain.StockingLineItem{
			ReceivingOrderLineID: li.ReceivingOrderLineId,
			LotNumber:            li.LotNumber,
		}

		if li.RejectedQuantity != nil {
			rq, _ := decimal.NewFromString(*li.RejectedQuantity)
			item.RejectedQuantity = &rq
		}

		allocations := make([]domain.StorageAllocation, len(li.Allocations))
		for j, a := range li.Allocations {
			qty, _ := decimal.NewFromString(a.Quantity)
			allocations[j] = domain.StorageAllocation{
				LocationID: a.LocationId,
				Quantity:   qty,
			}
		}
		item.Allocations = allocations

		lineItems[i] = item
	}

	return domain.StockingData{
		LineItems: lineItems,
	}
}

// ListReceivingOrders returns a paginated list of receiving orders.
func (h *receivingGRPCHandler) ListReceivingOrders(ctx context.Context, req *pb.ListReceivingOrdersRequest) (*pb.ListReceivingOrdersResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListReceivingOrdersParams{
		Limit:    req.Limit,
		Includes: req.Includes,
	}

	if req.Cursor != nil {
		params.Cursor = req.Cursor
	}
	if req.Query != nil {
		params.Query = req.Query
	}
	if req.Status != nil {
		params.Status = req.Status
	}
	if len(req.ItemIds) > 0 {
		params.ItemIDs = req.ItemIds
	}
	if len(req.SupplierIds) > 0 {
		params.SupplierIDs = req.SupplierIds
	}
	if req.StartDate != nil {
		t := req.StartDate.AsTime()
		params.StartDate = &t
	}
	if req.EndDate != nil {
		t := req.EndDate.AsTime()
		params.EndDate = &t
	}

	result, apiErr := h.receivingOrderSvc.ListReceivingOrders(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	orders := make([]*pb.ReceivingOrderSummaryInfo, len(result.ReceivingOrders))
	for i, o := range result.ReceivingOrders {
		orders[i] = receivingOrderSummaryToProto(o)
	}

	return &pb.ListReceivingOrdersResponse{
		ReceivingOrders: orders,
		PageInfo: &pb.PageInfo{
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
		},
	}, nil
}

// GetReceivingOrder returns a single receiving order by ID with lines.
func (h *receivingGRPCHandler) GetReceivingOrder(ctx context.Context, req *pb.GetReceivingOrderRequest) (*pb.GetReceivingOrderResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	order, apiErr := h.receivingOrderSvc.GetReceivingOrder(ctx, domain.GetReceivingOrderParams{
		ReceivingOrderID: req.Id,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetReceivingOrderResponse{
		ReceivingOrder: receivingOrderToProto(order),
	}, nil
}

// StockReceivingOrder stocks a receiving order, creating deliveries and inventory records.
func (h *receivingGRPCHandler) StockReceivingOrder(ctx context.Context, req *pb.StockReceivingOrderRequest) (*pb.StockReceivingOrderResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	order, apiErr := h.receivingOrderSvc.StockReceivingOrder(ctx, domain.StockReceivingOrderParams{
		ReceivingOrderID: req.Id,
		Data:             stockingDataFromProto(req.Data),
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.StockReceivingOrderResponse{
		ReceivingOrder: receivingOrderToProto(order),
	}, nil
}

// ReceiveReceivingOrder receives all unstocked lines, setting their quantities to remaining.
func (h *receivingGRPCHandler) ReceiveReceivingOrder(ctx context.Context, req *pb.ReceiveReceivingOrderRequest) (*pb.ReceiveReceivingOrderResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	order, apiErr := h.receivingOrderSvc.ReceiveReceivingOrder(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.ReceiveReceivingOrderResponse{
		ReceivingOrder: receivingOrderToProto(order),
	}, nil
}

// VoidReceivingOrder voids all lines in a receiving order.
func (h *receivingGRPCHandler) VoidReceivingOrder(ctx context.Context, req *pb.VoidReceivingOrderRequest) (*pb.VoidReceivingOrderResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	order, apiErr := h.receivingOrderSvc.VoidReceivingOrder(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.VoidReceivingOrderResponse{
		ReceivingOrder: receivingOrderToProto(order),
	}, nil
}

// UpdateReceivingOrderLine updates a receiving order line's quantity.
func (h *receivingGRPCHandler) UpdateReceivingOrderLine(ctx context.Context, req *pb.UpdateReceivingOrderLineRequest) (*pb.UpdateReceivingOrderLineResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateReceivingOrderLineParams{
		ReceivingOrderID: req.ReceivingOrderId,
		LineID:           req.Id,
	}

	if req.QuantityValue != nil {
		params.QuantityValue = req.QuantityValue
	}

	line, apiErr := h.receivingOrderLineSvc.UpdateReceivingOrderLine(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateReceivingOrderLineResponse{
		Line: receivingOrderLineToProto(line),
	}, nil
}

// VoidReceivingOrderLine voids a single receiving order line.
func (h *receivingGRPCHandler) VoidReceivingOrderLine(ctx context.Context, req *pb.VoidReceivingOrderLineRequest) (*pb.VoidReceivingOrderLineResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	line, apiErr := h.receivingOrderLineSvc.VoidReceivingOrderLine(ctx, req.ReceivingOrderId, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.VoidReceivingOrderLineResponse{
		Line: receivingOrderLineToProto(line),
	}, nil
}

// ReceiveReceivingOrderLine receives a single receiving order line.
func (h *receivingGRPCHandler) ReceiveReceivingOrderLine(ctx context.Context, req *pb.ReceiveReceivingOrderLineRequest) (*pb.ReceiveReceivingOrderLineResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	line, apiErr := h.receivingOrderLineSvc.ReceiveReceivingOrderLine(ctx, req.ReceivingOrderId, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.ReceiveReceivingOrderLineResponse{
		Line: receivingOrderLineToProto(line),
	}, nil
}

func receivingOrderTotalsToProto(t *domain.ReceivingOrderTotals) *pb.ReceivingOrderTotalsInfo {
	if t == nil {
		return nil
	}
	return &pb.ReceivingOrderTotalsInfo{
		OrderedAmount:  t.OrderedAmount,
		StockedAmount:  t.StockedAmount,
		RejectedAmount: t.RejectedAmount,
	}
}

func documentRefsToProto(refs []domain.DocumentRef) []*pb.DocumentRefInfo {
	if len(refs) == 0 {
		return nil
	}
	out := make([]*pb.DocumentRefInfo, len(refs))
	for i, r := range refs {
		out[i] = &pb.DocumentRefInfo{Id: r.ID, Number: r.Number, Status: r.Status}
	}
	return out
}
