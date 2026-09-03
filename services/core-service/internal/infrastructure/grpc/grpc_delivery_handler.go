package grpc

import (
	"context"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/contracts"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/ptrutil"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (h *gRPCHandler) ListDeliveries(ctx context.Context, req *pb.ListDeliveriesRequest) (*pb.ListDeliveriesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListDeliveriesParams{
		Cursor:      req.Cursor,
		Limit:       req.Limit,
		Query:       req.Query,
		Status:      req.Status,
		ItemIDs:     req.ItemIds,
		SupplierIDs: req.SupplierIds,
		Includes:    req.Includes,
	}

	if req.StartDate != nil {
		t := req.StartDate.AsTime()
		params.StartDate = &t
	}
	if req.EndDate != nil {
		t := req.EndDate.AsTime()
		params.EndDate = &t
	}

	result, apiErr := h.deliverySvc.ListDeliveries(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	deliveries := make([]*pb.DeliverySummaryInfo, len(result.Deliveries))
	for i, d := range result.Deliveries {
		deliveries[i] = deliverySummaryToProto(d)
	}

	return &pb.ListDeliveriesResponse{
		Deliveries: deliveries,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) GetDelivery(ctx context.Context, req *pb.GetDeliveryRequest) (*pb.GetDeliveryResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	delivery, apiErr := h.deliverySvc.GetDelivery(ctx, domain.GetDeliveryParams{
		DeliveryID: req.Id,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetDeliveryResponse{
		Delivery: deliveryToProto(delivery),
	}, nil
}

func deliverySummaryToProto(d *domain.DeliverySummary) *pb.DeliverySummaryInfo {
	if d == nil {
		return nil
	}

	info := &pb.DeliverySummaryInfo{
		Id:                   d.ID,
		Number:               d.Number,
		PurchaseOrderId:      d.PurchaseOrderID,
		PurchaseOrderNumber:  d.PurchaseOrderNumber,
		ReceivingOrderId:     ptrutil.NonEmptyPtr(d.ReceivingOrderID),
		ReceivingOrderNumber: ptrutil.NonEmptyPtr(d.ReceivingOrderNumber),
		Status:               d.Status,
		LineCount:            d.LineCount,
		CreatedAt:            timestamppb.New(d.CreatedAt),
		UpdatedAt:            timestamppb.New(d.UpdatedAt),
	}

	if d.AcceptedAt != nil {
		info.AcceptedAt = timestamppb.New(*d.AcceptedAt)
	}
	if d.RejectedAt != nil {
		info.RejectedAt = timestamppb.New(*d.RejectedAt)
	}

	for _, l := range d.Lines {
		info.Lines = append(info.Lines, deliveryLineToProto(l))
	}

	return info
}

func deliveryToProto(d *domain.Delivery) *pb.DeliveryInfo {
	if d == nil {
		return nil
	}

	lines := make([]*pb.DeliveryLineInfo, len(d.Lines))
	for i, l := range d.Lines {
		lines[i] = deliveryLineToProto(l)
	}

	info := &pb.DeliveryInfo{
		Id:                   d.ID,
		Number:               d.Number,
		PurchaseOrderId:      d.PurchaseOrderID,
		PurchaseOrderNumber:  d.PurchaseOrderNumber,
		ReceivingOrderId:     ptrutil.NonEmptyPtr(d.ReceivingOrderID),
		ReceivingOrderNumber: ptrutil.NonEmptyPtr(d.ReceivingOrderNumber),
		Status:               d.Status,
		Lines:                lines,
		CreatedAt:            timestamppb.New(d.CreatedAt),
		UpdatedAt:            timestamppb.New(d.UpdatedAt),
	}

	if d.AcceptedAt != nil {
		info.AcceptedAt = timestamppb.New(*d.AcceptedAt)
	}
	if d.RejectedAt != nil {
		info.RejectedAt = timestamppb.New(*d.RejectedAt)
	}

	return info
}

func deliveryLineToProto(l *domain.DeliveryLine) *pb.DeliveryLineInfo {
	if l == nil {
		return nil
	}

	info := &pb.DeliveryLineInfo{
		Id:                                  l.ID,
		QuantityId:                          l.QuantityID,
		QuantityValue:                       l.QuantityValue,
		QuantityUnitId:                      l.QuantityUnitID,
		QuantityUnitAbbreviation:            l.QuantityUnitAbbreviation,
		OrderLineId:                         l.OrderLineID,
		UnitCostId:                          l.UnitCostID,
		UnitCostValue:                       l.UnitCostValue,
		UnitCostNumeratorUnitId:             l.UnitCostNumeratorUnitID,
		UnitCostDenominatorUnitId:           l.UnitCostDenominatorUnitID,
		UnitCostNumeratorUnitAbbreviation:   l.UnitCostNumeratorUnitAbbreviation,
		UnitCostDenominatorUnitAbbreviation: l.UnitCostDenominatorUnitAbbreviation,
		UnitCostCreatedAt:                   timestamppb.New(l.UnitCostCreatedAt),
		UnitCostUpdatedAt:                   timestamppb.New(l.UnitCostUpdatedAt),
		CreatedAt:                           timestamppb.New(l.CreatedAt),
		UpdatedAt:                           timestamppb.New(l.UpdatedAt),
	}

	if l.ItemID != nil {
		info.ItemId = l.ItemID
	}
	if l.ItemSKU != nil {
		info.ItemSku = l.ItemSKU
	}
	if l.ItemDescription != nil {
		info.ItemDescription = l.ItemDescription
	}
	if l.LocationID != nil {
		info.LocationId = l.LocationID
	}
	if l.LocationName != nil {
		info.LocationName = l.LocationName
	}
	if l.LotID != nil {
		info.LotId = l.LotID
	}
	if l.LotNumber != nil {
		info.LotNumber = l.LotNumber
	}
	if l.AcceptedAt != nil {
		info.AcceptedAt = timestamppb.New(*l.AcceptedAt)
	}
	if l.RejectedAt != nil {
		info.RejectedAt = timestamppb.New(*l.RejectedAt)
	}

	return info
}
