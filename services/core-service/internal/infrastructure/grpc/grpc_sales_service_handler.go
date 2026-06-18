package grpc

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/core"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type salesGRPCHandler struct {
	pb.UnimplementedCoreSalesServiceServer

	salesOrderStatusSvc domain.SalesOrderStatusSvc
	orderDiscountSvc    domain.OrderDiscountSvc
	volumeDiscountSvc   domain.VolumeDiscountSvc
	salesOrderSvc       domain.SalesOrderSvc
	salesOrderLineSvc   domain.SalesOrderLineSvc
}

func salesOrderStatusToProto(s *domain.SalesOrderStatus) *pb.SalesOrderStatusInfo {
	if s == nil {
		return nil
	}

	return &pb.SalesOrderStatusInfo{
		Id:        s.ID,
		Code:      s.Code,
		Name:      s.Name,
		CreatedAt: timestamppb.New(s.CreatedAt),
		UpdatedAt: timestamppb.New(s.UpdatedAt),
	}
}

func orderDiscountToProto(d *domain.OrderDiscount) *pb.OrderDiscountInfo {
	if d == nil {
		return nil
	}

	return &pb.OrderDiscountInfo{
		Id:           d.ID,
		Name:         d.Name,
		Code:         d.Code,
		Percentage:   d.Percentage,
		Amount:       d.Amount,
		DiscountType: d.DiscountTypeCode,
		OrderCount:   d.OrderCount,
		CreatedAt:    timestamppb.New(d.CreatedAt),
		UpdatedAt:    timestamppb.New(d.UpdatedAt),
	}
}

func (h *salesGRPCHandler) ListSalesOrderStatuses(ctx context.Context, req *pb.ListSalesOrderStatusesRequest) (*pb.ListSalesOrderStatusesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListSalesOrderStatusesParams{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	result, apiErr := h.salesOrderStatusSvc.ListSalesOrderStatuses(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	statuses := make([]*pb.SalesOrderStatusInfo, len(result.SalesOrderStatuses))
	for i, s := range result.SalesOrderStatuses {
		statuses[i] = salesOrderStatusToProto(s)
	}

	return &pb.ListSalesOrderStatusesResponse{
		SalesOrderStatuses: statuses,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *salesGRPCHandler) ListOrderDiscounts(ctx context.Context, req *pb.ListOrderDiscountsRequest) (*pb.ListOrderDiscountsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListOrderDiscountsParams{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	result, apiErr := h.orderDiscountSvc.ListOrderDiscounts(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	discounts := make([]*pb.OrderDiscountInfo, len(result.OrderDiscounts))
	for i, d := range result.OrderDiscounts {
		discounts[i] = orderDiscountToProto(d)
	}

	return &pb.ListOrderDiscountsResponse{
		OrderDiscounts: discounts,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *salesGRPCHandler) GetOrderDiscount(ctx context.Context, req *pb.GetOrderDiscountRequest) (*pb.GetOrderDiscountResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	discount, apiErr := h.orderDiscountSvc.GetOrderDiscount(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetOrderDiscountResponse{
		OrderDiscount: orderDiscountToProto(discount),
	}, nil
}

func (h *salesGRPCHandler) BatchGetOrderDiscountsByIDs(ctx context.Context, req *pb.BatchGetOrderDiscountsByIDsRequest) (*pb.BatchGetOrderDiscountsByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	discounts, apiErr := h.orderDiscountSvc.BatchGetOrderDiscountsByIDs(ctx, req.Ids)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	pbDiscounts := make([]*pb.OrderDiscountInfo, len(discounts))
	for i, d := range discounts {
		pbDiscounts[i] = orderDiscountToProto(d)
	}
	return &pb.BatchGetOrderDiscountsByIDsResponse{OrderDiscounts: pbDiscounts}, nil
}

func (h *salesGRPCHandler) CreateOrderDiscount(ctx context.Context, req *pb.CreateOrderDiscountRequest) (*pb.CreateOrderDiscountResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.CreateOrderDiscountParams{
		Name:         req.Name,
		Code:         req.Code,
		Percentage:   req.Percentage,
		Amount:       req.Amount,
		DiscountType: req.DiscountType,
	}

	discount, apiErr := h.orderDiscountSvc.CreateOrderDiscount(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateOrderDiscountResponse{
		OrderDiscount: orderDiscountToProto(discount),
	}, nil
}

func (h *salesGRPCHandler) UpdateOrderDiscount(ctx context.Context, req *pb.UpdateOrderDiscountRequest) (*pb.UpdateOrderDiscountResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateOrderDiscountParams{
		OrderDiscountID: req.Id,
		Name:            req.Name,
		Code:            req.Code,
		Percentage:      req.Percentage,
		Amount:          req.Amount,
		DiscountType:    req.DiscountType,
	}

	discount, apiErr := h.orderDiscountSvc.UpdateOrderDiscount(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateOrderDiscountResponse{
		OrderDiscount: orderDiscountToProto(discount),
	}, nil
}

func (h *salesGRPCHandler) DeleteOrderDiscount(ctx context.Context, req *pb.DeleteOrderDiscountRequest) (*pb.DeleteOrderDiscountResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	discount, apiErr := h.orderDiscountSvc.DeleteOrderDiscount(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.DeleteOrderDiscountResponse{
		OrderDiscount: orderDiscountToProto(discount),
	}, nil
}

func salesOrderToProto(o *domain.SalesOrder) *pb.SalesOrderInfo {
	if o == nil {
		return nil
	}

	info := &pb.SalesOrderInfo{
		Id:                    o.ID,
		Number:                o.Number,
		CustomerPoNumber:      o.CustomerPONumber,
		Note:                  o.Note,
		IsAcknowledgmentSent:  o.IsAcknowledgmentSent,
		BillingAddressId:      o.BillingAddressID,
		ShippingAddressId:     o.ShippingAddressID,
		CarrierId:             o.CarrierID,
		ServiceLevelId:        o.ServiceLevelID,
		CarrierBillingType:    o.CarrierBillingType,
		CarrierBillingAccount: o.CarrierBillingAccount,
		CustomerId:            o.BuyerAccountID,
		CustomerName:          o.CustomerName,
		CustomerNumber:        o.CustomerNumber,
		SalesRepId:            o.SalesRepID,
		SalesRepName:          o.SalesRepName,
		StatusCode:            o.SalesOrderStatusCode,
		StatusName:            o.StatusName,
		TypeCode:              o.SalesOrderTypeCode,
		TypeName:              o.TypeName,
		PriorityCode:          string(o.PriorityCode),
		PriorityName:          o.PriorityName,
		PaymentTermId:         o.PaymentTermID,
		PaymentTermName:       o.PaymentTermName,
		ShippingTermId:        o.ShippingTermID,
		ShippingTermName:      o.ShippingTermName,
		OrderDiscountId:       o.OrderDiscountID,
		OrderDiscountName:     o.OrderDiscountName,
		ProductionRunId:       o.ProductionRunID,
		PickId:                o.PickID,
		BillToName:            o.BillToName,
		BillToStreetLine_1:    o.BillToStreetLine1,
		BillToStreetLine_2:    o.BillToStreetLine2,
		BillToLocality:        o.BillToLocality,
		BillToState:           o.BillToState,
		BillToPostalCode:      o.BillToPostalCode,
		BillToCountry:         o.BillToCountry,
		BillToPhone:           o.BillToPhone,
		BillToEmail:           o.BillToEmail,
		ShipToName:            o.ShipToName,
		ShipToStreetLine_1:    o.ShipToStreetLine1,
		ShipToStreetLine_2:    o.ShipToStreetLine2,
		ShipToLocality:        o.ShipToLocality,
		ShipToState:           o.ShipToState,
		ShipToPostalCode:      o.ShipToPostalCode,
		ShipToCountry:         o.ShipToCountry,
		ShipToPhone:           o.ShipToPhone,
		ShipToEmail:           o.ShipToEmail,
		CarrierName:           o.CarrierName,
		ServiceLevelName:      o.ServiceLevelName,
		LineCount:             o.LineCount,
		ShipmentIds:           o.ShipmentIDs,
		InvoiceEmails:         o.InvoiceEmails,
		AcknowledgementEmails: o.AcknowledgementEmails,
		PaymentStatus:         string(o.PaymentStatus),
		CreatedAt:             timestamppb.New(o.CreatedAt),
		UpdatedAt:             timestamppb.New(o.UpdatedAt),
	}

	if o.ServiceLevelToken != nil {
		info.ServiceLevelToken = o.ServiceLevelToken
	}
	if o.ServiceLevelCreatedAt != nil {
		info.ServiceLevelCreatedAt = timestamppb.New(*o.ServiceLevelCreatedAt)
	}
	if o.ServiceLevelUpdatedAt != nil {
		info.ServiceLevelUpdatedAt = timestamppb.New(*o.ServiceLevelUpdatedAt)
	}
	if o.CustomerStatusCode != nil {
		info.CustomerStatusCode = o.CustomerStatusCode
	}
	if o.CustomerCommissionPolicy != nil {
		info.CustomerCommissionPolicy = o.CustomerCommissionPolicy
	}
	if o.CustomerCreatedAt != nil {
		info.CustomerCreatedAt = timestamppb.New(*o.CustomerCreatedAt)
	}
	if o.CustomerUpdatedAt != nil {
		info.CustomerUpdatedAt = timestamppb.New(*o.CustomerUpdatedAt)
	}
	if o.BillToIsDropShip != nil {
		info.BillToIsDropShip = o.BillToIsDropShip
	}
	if o.BillToGeolocationID != nil {
		info.BillToGeolocationId = o.BillToGeolocationID
	}
	if o.BillToCreatedAt != nil {
		info.BillToCreatedAt = timestamppb.New(*o.BillToCreatedAt)
	}
	if o.BillToUpdatedAt != nil {
		info.BillToUpdatedAt = timestamppb.New(*o.BillToUpdatedAt)
	}
	if o.ShipToIsDropShip != nil {
		info.ShipToIsDropShip = o.ShipToIsDropShip
	}
	if o.ShipToGeolocationID != nil {
		info.ShipToGeolocationId = o.ShipToGeolocationID
	}
	if o.ShipToCreatedAt != nil {
		info.ShipToCreatedAt = timestamppb.New(*o.ShipToCreatedAt)
	}
	if o.ShipToUpdatedAt != nil {
		info.ShipToUpdatedAt = timestamppb.New(*o.ShipToUpdatedAt)
	}
	if o.PaymentTermCreatedAt != nil {
		info.PaymentTermCreatedAt = timestamppb.New(*o.PaymentTermCreatedAt)
	}
	if o.PaymentTermUpdatedAt != nil {
		info.PaymentTermUpdatedAt = timestamppb.New(*o.PaymentTermUpdatedAt)
	}
	if o.ShippingTermCreatedAt != nil {
		info.ShippingTermCreatedAt = timestamppb.New(*o.ShippingTermCreatedAt)
	}
	if o.ShippingTermUpdatedAt != nil {
		info.ShippingTermUpdatedAt = timestamppb.New(*o.ShippingTermUpdatedAt)
	}
	if o.OrderDiscountCode != nil {
		info.OrderDiscountCode = o.OrderDiscountCode
	}
	if o.OrderDiscountPercentage != nil {
		info.OrderDiscountPercentage = o.OrderDiscountPercentage
	}
	if o.OrderDiscountAmount != nil {
		info.OrderDiscountAmount = o.OrderDiscountAmount
	}
	if o.OrderDiscountDiscountType != nil {
		info.OrderDiscountDiscountType = o.OrderDiscountDiscountType
	}
	if o.OrderDiscountOrderCount != nil {
		info.OrderDiscountOrderCount = o.OrderDiscountOrderCount
	}
	if o.OrderDiscountCreatedAt != nil {
		info.OrderDiscountCreatedAt = timestamppb.New(*o.OrderDiscountCreatedAt)
	}
	if o.OrderDiscountUpdatedAt != nil {
		info.OrderDiscountUpdatedAt = timestamppb.New(*o.OrderDiscountUpdatedAt)
	}

	if o.IssuedAt != nil {
		info.IssuedAt = timestamppb.New(*o.IssuedAt)
	}
	if o.CompletedAt != nil {
		info.CompletedAt = timestamppb.New(*o.CompletedAt)
	}
	if o.FirstShipAt != nil {
		info.FirstShipAt = timestamppb.New(*o.FirstShipAt)
	}
	if o.ExpiredAt != nil {
		info.ExpiredAt = timestamppb.New(*o.ExpiredAt)
	}
	if o.PromisedAt != nil {
		info.PromisedAt = timestamppb.New(*o.PromisedAt)
	}

	if o.CarrierIsPortalEnabled != nil {
		info.CarrierIsPortalEnabled = o.CarrierIsPortalEnabled
	}
	if o.CarrierCreatedAt != nil {
		info.CarrierCreatedAt = timestamppb.New(*o.CarrierCreatedAt)
	}
	if o.CarrierUpdatedAt != nil {
		info.CarrierUpdatedAt = timestamppb.New(*o.CarrierUpdatedAt)
	}
	if o.ServiceLevelIsPortalEnabled != nil {
		info.ServiceLevelIsPortalEnabled = o.ServiceLevelIsPortalEnabled
	}
	if o.PaymentTermIsActive != nil {
		info.PaymentTermIsActive = o.PaymentTermIsActive
	}
	if o.ShippingTermIsFreightExempt != nil || o.ShippingTermIsCarrierRate != nil {
		isFreightExempt := o.ShippingTermIsFreightExempt != nil && *o.ShippingTermIsFreightExempt
		isCarrierRate := o.ShippingTermIsCarrierRate != nil && *o.ShippingTermIsCarrierRate
		var stType string
		if isFreightExempt {
			stType = "free_freight"
		} else if isCarrierRate {
			stType = "carrier_rate_freight"
		} else {
			stType = "flat_rate_freight"
		}
		info.ShippingTermType = &stType
	}
	if o.PriorityID != nil {
		info.PriorityId = o.PriorityID
	}

	if o.Lines != nil {
		lines := make([]*pb.SalesOrderLineInfo, len(o.Lines))
		for i, l := range o.Lines {
			lines[i] = salesOrderLineToProto(l)
		}
		info.Lines = lines
	}

	return info
}

func salesOrderLineToProto(l *domain.SalesOrderLine) *pb.SalesOrderLineInfo {
	if l == nil {
		return nil
	}

	info := &pb.SalesOrderLineInfo{
		Id:                                   l.ID,
		LineItemNumber:                       l.LineItemNumber,
		ProductSku:                           l.ProductSKU,
		ProductDescription:                   l.ProductDescription,
		ProductId:                            l.ProductID,
		ItemId:                               l.ItemID,
		ItemSku:                              l.ItemSKU,
		EdiLineItemId:                        l.EdiLineItemID,
		QuantityId:                           l.QuantityID,
		QuantityValue:                        l.QuantityValue,
		QuantityUnitId:                       l.QuantityUnitID,
		QuantityUnitName:                     l.QuantityUnitName,
		QuantityUnitAbbreviation:             l.QuantityUnitAbbreviation,
		QuantityUnitType:                     l.QuantityUnitType,
		QuantityPickedValue:                  l.QuantityPickedValue,
		QuantityPackedValue:                  l.QuantityPackedValue,
		QuantityInvoicedValue:                l.QuantityInvoicedValue,
		UnitPriceId:                          l.UnitPriceID,
		UnitPriceValue:                       l.UnitPriceValue,
		UnitPriceNumeratorUnitId:             l.UnitPriceNumeratorUnitID,
		UnitPriceNumeratorUnitAbbreviation:   l.UnitPriceNumeratorUnitAbbr,
		UnitPriceDenominatorUnitId:           l.UnitPriceDenominatorUnitID,
		UnitPriceDenominatorUnitAbbreviation: l.UnitPriceDenominatorUnitAbbr,
		UnitCostId:                           l.UnitCostID,
		UnitCostValue:                        l.UnitCostValue,
		UnitCostNumeratorUnitId:              l.UnitCostNumeratorUnitID,
		UnitCostNumeratorUnitAbbreviation:    l.UnitCostNumeratorUnitAbbr,
		UnitCostDenominatorUnitId:            l.UnitCostDenominatorUnitID,
		UnitCostDenominatorUnitAbbreviation:  l.UnitCostDenominatorUnitAbbr,
		CreatedAt:                            timestamppb.New(l.CreatedAt),
		UpdatedAt:                            timestamppb.New(l.UpdatedAt),
	}

	return info
}

func (h *salesGRPCHandler) ListSalesOrders(ctx context.Context, req *pb.ListSalesOrdersRequest) (*pb.ListSalesOrdersResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListSalesOrdersParams{
		Cursor:                req.Cursor,
		Limit:                 req.Limit,
		Query:                 req.Query,
		StatusCodes:           req.StatusCodes,
		ItemIDs:               req.ItemIds,
		ProductLineIDs:        req.ProductLineIds,
		CustomerIDs:           req.CustomerIds,
		CustomerGroupIDs:      req.CustomerGroupIds,
		SalesRepIDs:           req.SalesRepIds,
		StartDate:             req.StartDate,
		EndDate:               req.EndDate,
		ExcludeInternalOrders: req.ExcludeInternalOrders,
		BuyerAccountID:        req.BuyerAccountId,
		Includes:              req.Includes,
	}

	result, apiErr := h.salesOrderSvc.ListSalesOrders(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	orders := make([]*pb.SalesOrderInfo, len(result.SalesOrders))
	for i, o := range result.SalesOrders {
		orders[i] = salesOrderToProto(o)
	}

	return &pb.ListSalesOrdersResponse{
		SalesOrders: orders,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *salesGRPCHandler) GetSalesOrder(ctx context.Context, req *pb.GetSalesOrderRequest) (*pb.GetSalesOrderResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.GetSalesOrderParams{
		SalesOrderID:   req.Id,
		Includes:       req.Includes,
		BuyerAccountID: req.BuyerAccountId,
	}

	order, apiErr := h.salesOrderSvc.GetSalesOrder(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetSalesOrderResponse{
		SalesOrder: salesOrderToProto(order),
	}, nil
}

// BatchGetSalesOrdersByIDs returns full sales orders by ID for the api-gateway include resolver. It reuses the authorized single-get path per id; ids the caller cannot access or that no longer exist are omitted from the response so the resolver simply leaves those references null.
func (h *salesGRPCHandler) BatchGetSalesOrdersByIDs(ctx context.Context, req *pb.BatchGetSalesOrdersByIDsRequest) (*pb.BatchGetSalesOrdersByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	orders := make([]*pb.SalesOrderInfo, 0, len(req.Ids))
	for _, soID := range req.Ids {
		if soID == "" {
			continue
		}
		order, apiErr := h.salesOrderSvc.GetSalesOrder(ctx, domain.GetSalesOrderParams{SalesOrderID: soID})
		if apiErr != nil {
			continue
		}
		orders = append(orders, salesOrderToProto(order))
	}

	return &pb.BatchGetSalesOrdersByIDsResponse{SalesOrders: orders}, nil
}

func (h *salesGRPCHandler) FindOrderDiscountByCode(ctx context.Context, req *pb.FindOrderDiscountByCodeRequest) (*pb.FindOrderDiscountByCodeResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.FindOrderDiscountByCodeParams{
		Code:           req.Code,
		BuyerAccountID: req.BuyerAccountId,
		SalesOrderID:   req.SalesOrderId,
	}

	discount, apiErr := h.orderDiscountSvc.FindOrderDiscountByCode(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.FindOrderDiscountByCodeResponse{
		OrderDiscount: orderDiscountToProto(discount),
	}, nil
}

func (h *salesGRPCHandler) CreateSalesOrder(ctx context.Context, req *pb.CreateSalesOrderRequest) (*pb.CreateSalesOrderResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	lines := make([]domain.CreateSalesOrderLineInput, len(req.Lines))
	for i, l := range req.Lines {
		line := domain.CreateSalesOrderLineInput{
			ProductID:          l.ProductId,
			ProductSKU:         l.ProductSku,
			ProductDescription: l.ProductDescription,
			QuantityValue:      l.QuantityValue,
			QuantityUnitID:     l.QuantityUnitId,
			EdiLineItemID:      l.EdiLineItemId,
		}
		if l.UnitPriceValue != nil {
			line.UnitPrice = &domain.RateValue{
				Value:             l.GetUnitPriceValue(),
				NumeratorUnitID:   l.GetUnitPriceNumeratorUnitId(),
				DenominatorUnitID: l.GetUnitPriceDenominatorUnitId(),
			}
		}
		lines[i] = line
	}

	params := domain.CreateSalesOrderParams{
		BuyerAccountID:               req.BuyerAccountId,
		CustomerPONumber:             req.CustomerPoNumber,
		Note:                         req.Note,
		CarrierID:                    req.CarrierId,
		ServiceLevelID:               req.ServiceLevelId,
		CarrierBillingType:           req.CarrierBillingType,
		CarrierBillingAccount:        req.CarrierBillingAccount,
		PriorityCode:                 req.PriorityCode,
		SalesRepID:                   req.SalesRepId,
		ShippingTermID:               req.ShippingTermId,
		PaymentTermID:                req.PaymentTermId,
		OrderDiscountID:              req.OrderDiscountId,
		BillToAddressID:              req.BillToAddressId,
		ShipToAddressID:              req.ShipToAddressId,
		Lines:                        lines,
		AcknowledgementEmailContacts: protoToEmailContactInputs(req.AcknowledgementEmailContacts),
		InvoiceEmailContacts:         protoToEmailContactInputs(req.InvoiceEmailContacts),
		Includes:                     req.Includes,
	}

	if req.PromisedAt != nil {
		t := req.PromisedAt.AsTime()
		params.PromisedAt = &t
	}

	order, apiErr := h.salesOrderSvc.CreateSalesOrder(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateSalesOrderResponse{
		SalesOrder: salesOrderToProto(order),
	}, nil
}

func (h *salesGRPCHandler) UpdateSalesOrder(ctx context.Context, req *pb.UpdateSalesOrderRequest) (*pb.UpdateSalesOrderResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateSalesOrderParams{
		SalesOrderID:          req.Id,
		Number:                req.Number,
		CustomerPONumber:      req.CustomerPoNumber,
		Note:                  req.Note,
		CarrierID:             req.CarrierId,
		ServiceLevelID:        req.ServiceLevelId,
		CarrierBillingType:    req.CarrierBillingType,
		CarrierBillingAccount: req.CarrierBillingAccount,
		PriorityCode:          req.PriorityCode,
		SalesRepID:            req.SalesRepId,
		ShippingTermID:        req.ShippingTermId,
		PaymentTermID:         req.PaymentTermId,
		OrderDiscountID:       req.OrderDiscountId,
		IsAcknowledgmentSent:  req.IsAcknowledgmentSent,
		BuyerAccountID:        req.CustomerId,
		BillingAddressID:      req.BillingAddressId,
		ShippingAddressID:     req.ShippingAddressId,
		Includes:              req.Includes,
	}

	if req.PromisedAt != nil {
		t := req.PromisedAt.AsTime()
		params.PromisedAt = &t
	}

	if list := req.AcknowledgementEmailContacts; list != nil {
		contacts := protoToEmailContactInputs(list.Contacts)
		params.AcknowledgementEmailContacts = &contacts
	}
	if list := req.InvoiceEmailContacts; list != nil {
		contacts := protoToEmailContactInputs(list.Contacts)
		params.InvoiceEmailContacts = &contacts
	}

	order, apiErr := h.salesOrderSvc.UpdateSalesOrder(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateSalesOrderResponse{
		SalesOrder: salesOrderToProto(order),
	}, nil
}

func (h *salesGRPCHandler) DeleteSalesOrder(ctx context.Context, req *pb.DeleteSalesOrderRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.salesOrderSvc.DeleteSalesOrder(ctx, domain.DeleteSalesOrderParams{
		SalesOrderID: req.Id,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *salesGRPCHandler) BulkDeleteSalesOrders(ctx context.Context, req *pb.BulkDeleteSalesOrdersRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	apiErr := h.salesOrderSvc.BulkDeleteSalesOrders(ctx, domain.BulkDeleteSalesOrdersParams{
		SalesOrderIDs: req.Ids,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *salesGRPCHandler) ChangeSalesOrderStatus(ctx context.Context, req *pb.ChangeSalesOrderStatusRequest) (*pb.ChangeSalesOrderStatusResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	order, apiErr := h.salesOrderSvc.ChangeSalesOrderStatus(ctx, domain.ChangeSalesOrderStatusParams{
		SalesOrderID: req.Id,
		StatusChange: req.StatusChange,
		SendEmail:    req.SendEmail,
		Includes:     req.Includes,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.ChangeSalesOrderStatusResponse{
		SalesOrder: salesOrderToProto(order),
	}, nil
}

func (h *salesGRPCHandler) CheckoutSalesOrder(ctx context.Context, req *pb.CheckoutSalesOrderRequest) (*pb.CheckoutSalesOrderResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	result, apiErr := h.salesOrderSvc.CheckoutSalesOrder(ctx, domain.CheckoutSalesOrderParams{
		SalesOrderID: req.Id,
		Email:        req.Email,
		SuccessURL:   req.SuccessUrl,
		CancelURL:    req.CancelUrl,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CheckoutSalesOrderResponse{
		CheckoutUrl: result.CheckoutURL,
	}, nil
}

func (h *salesGRPCHandler) QuoteSalesOrderLinePrices(ctx context.Context, req *pb.QuoteSalesOrderLinePricesRequest) (*pb.QuoteSalesOrderLinePricesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	lines := make([]domain.SalesOrderPriceLineInput, len(req.Lines))
	for i, l := range req.Lines {
		lines[i] = domain.SalesOrderPriceLineInput{
			ProductID:      l.ProductId,
			QuantityValue:  l.QuantityValue,
			QuantityUnitID: l.QuantityUnitId,
		}
	}

	quotes, apiErr := h.salesOrderSvc.QuoteSalesOrderLinePrices(ctx, domain.QuoteSalesOrderLinePricesParams{
		BuyerAccountID: req.BuyerAccountId,
		Lines:          lines,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	respLines := make([]*pb.SalesOrderLineQuote, len(quotes))
	for i, q := range quotes {
		respLines[i] = &pb.SalesOrderLineQuote{
			ProductId:                  q.ProductID,
			UnitPriceValue:             q.UnitPrice.Value,
			UnitPriceNumeratorUnitId:   q.UnitPrice.NumeratorUnitID,
			UnitPriceDenominatorUnitId: q.UnitPrice.DenominatorUnitID,
		}
	}

	return &pb.QuoteSalesOrderLinePricesResponse{Lines: respLines}, nil
}

func (h *salesGRPCHandler) CreateSalesOrderProductionRun(ctx context.Context, req *pb.CreateSalesOrderProductionRunRequest) (*pb.CreateSalesOrderProductionRunResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	result, apiErr := h.salesOrderSvc.CreateSalesOrderProductionRun(ctx, domain.CreateSalesOrderProductionRunParams{
		SalesOrderID: req.Id,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateSalesOrderProductionRunResponse{
		ProductionRunId: result.ProductionRunID,
	}, nil
}

func (h *salesGRPCHandler) CreateSalesOrderLine(ctx context.Context, req *pb.CreateSalesOrderLineRequest) (*pb.CreateSalesOrderLineResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.CreateSalesOrderLineParams{
		SalesOrderID:               req.SalesOrderId,
		ProductID:                  req.ProductId,
		ItemID:                     req.ItemId,
		ProductSKU:                 req.ProductSku,
		ProductDescription:         req.ProductDescription,
		QuantityValue:              req.QuantityValue,
		QuantityUnitID:             req.QuantityUnitId,
		UnitPriceValue:             req.UnitPriceValue,
		UnitPriceNumeratorUnitID:   req.UnitPriceNumeratorUnitId,
		UnitPriceDenominatorUnitID: req.UnitPriceDenominatorUnitId,
		UnitCostValue:              req.UnitCostValue,
		UnitCostNumeratorUnitID:    req.UnitCostNumeratorUnitId,
		UnitCostDenominatorUnitID:  req.UnitCostDenominatorUnitId,
		EdiLineItemID:              req.EdiLineItemId,
	}

	line, apiErr := h.salesOrderLineSvc.CreateSalesOrderLine(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateSalesOrderLineResponse{
		SalesOrderLine: salesOrderLineToProto(line),
	}, nil
}

func (h *salesGRPCHandler) UpdateSalesOrderLine(ctx context.Context, req *pb.UpdateSalesOrderLineRequest) (*pb.UpdateSalesOrderLineResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateSalesOrderLineParams{
		SalesOrderLineID:           req.Id,
		SalesOrderID:               req.SalesOrderId,
		ProductID:                  req.ProductId,
		ItemID:                     req.ItemId,
		ProductSKU:                 req.ProductSku,
		ProductDescription:         req.ProductDescription,
		QuantityValue:              req.QuantityValue,
		QuantityUnitID:             req.QuantityUnitId,
		UnitPriceValue:             req.UnitPriceValue,
		UnitPriceNumeratorUnitID:   req.UnitPriceNumeratorUnitId,
		UnitPriceDenominatorUnitID: req.UnitPriceDenominatorUnitId,
		UnitCostValue:              req.UnitCostValue,
		UnitCostNumeratorUnitID:    req.UnitCostNumeratorUnitId,
		UnitCostDenominatorUnitID:  req.UnitCostDenominatorUnitId,
		EdiLineItemID:              req.EdiLineItemId,
	}

	line, apiErr := h.salesOrderLineSvc.UpdateSalesOrderLine(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateSalesOrderLineResponse{
		SalesOrderLine: salesOrderLineToProto(line),
	}, nil
}

func (h *salesGRPCHandler) DeleteSalesOrderLine(ctx context.Context, req *pb.DeleteSalesOrderLineRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.salesOrderLineSvc.DeleteSalesOrderLine(ctx, domain.DeleteSalesOrderLineParams{
		SalesOrderLineID: req.Id,
		SalesOrderID:     req.SalesOrderId,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *salesGRPCHandler) CreateCustomerCheckoutSession(ctx context.Context, req *pb.CreateCustomerCheckoutSessionRequest) (*pb.CreateCustomerCheckoutSessionResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.CreateCustomerCheckoutSessionParams{
		OrderID:         req.OrderId,
		OrderNumber:     req.OrderNumber,
		OrderTotalCents: req.OrderTotalCents,
		CustomerPO:      req.CustomerPo,
	}

	result, apiErr := h.salesOrderSvc.CreateCustomerCheckoutSession(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateCustomerCheckoutSessionResponse{
		ClientSecret: result.ClientSecret,
	}, nil
}

func (h *salesGRPCHandler) RecordOrderPayment(ctx context.Context, req *pb.RecordOrderPaymentRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.salesOrderSvc.RecordOrderPayment(ctx, req.SalesOrderId, req.PaymentIntentId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

// protoToEmailContactInputs maps proto contact inputs to the domain layer.
func protoToEmailContactInputs(in []*pb.SalesOrderEmailContactInput) []domain.SalesOrderEmailContactInput {
	if len(in) == 0 {
		return nil
	}
	out := make([]domain.SalesOrderEmailContactInput, len(in))
	for i, c := range in {
		out[i] = domain.SalesOrderEmailContactInput{AccountUserID: c.AccountUserId}
	}
	return out
}

func (h *salesGRPCHandler) BatchGetSalesOrderStatusesByIDs(ctx context.Context, req *pb.BatchGetSalesOrderStatusesByIDsRequest) (*pb.BatchGetSalesOrderStatusesByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	statuses, apiErr := h.salesOrderStatusSvc.BatchGetSalesOrderStatusesByIDs(ctx, req.Ids)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	pbStatuses := make([]*pb.SalesOrderStatusInfo, len(statuses))
	for i, s := range statuses {
		pbStatuses[i] = salesOrderStatusToProto(s)
	}
	return &pb.BatchGetSalesOrderStatusesByIDsResponse{SalesOrderStatuses: pbStatuses}, nil
}
