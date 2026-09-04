package grpc

import (
	"context"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	pb "github.com/open-mrp/api/shared/proto/core"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type purchaseGRPCHandler struct {
	pb.UnimplementedCorePurchaseServiceServer

	purchaseOrderSvc     domain.PurchaseOrderSvc
	purchaseOrderLineSvc domain.PurchaseOrderLineSvc
}

func purchaseOrderSummaryToProto(s *domain.PurchaseOrderSummary) *pb.PurchaseOrderSummaryInfo {
	if s == nil {
		return nil
	}

	info := &pb.PurchaseOrderSummaryInfo{
		Id:                   s.ID,
		Number:               s.Number,
		StatusCode:           s.StatusCode,
		StatusName:           s.StatusName,
		TypeCode:             s.TypeCode,
		TypeName:             s.TypeName,
		SupplierId:           s.SupplierID,
		SupplierName:         s.SupplierName,
		SupplierNumber:       s.SupplierNumber,
		LineCount:            s.LineCount,
		IsAcknowledgmentSent: s.IsAcknowledgmentSent,
		PriorityCode:         string(s.PriorityCode),
		PriorityName:         s.PriorityName,
		CreatedAt:            timestamppb.New(s.CreatedAt),
		UpdatedAt:            timestamppb.New(s.UpdatedAt),
	}

	if s.PriorityID != nil {
		info.PriorityId = s.PriorityID
	}
	if s.IssuedAt != nil {
		info.IssuedAt = timestamppb.New(*s.IssuedAt)
	}
	if s.CompletedAt != nil {
		info.CompletedAt = timestamppb.New(*s.CompletedAt)
	}

	for _, l := range s.Lines {
		info.Lines = append(info.Lines, purchaseOrderLineToProto(l))
	}

	return info
}

func purchaseOrderToProto(o *domain.PurchaseOrder) *pb.PurchaseOrderInfo {
	if o == nil {
		return nil
	}

	info := &pb.PurchaseOrderInfo{
		Id:                    o.ID,
		Number:                o.Number,
		Note:                  o.Note,
		IsAcknowledgmentSent:  o.IsAcknowledgmentSent,
		BillingAddressId:      o.BillingAddressID,
		ShippingAddressId:     o.ShippingAddressID,
		CarrierId:             o.CarrierID,
		ServiceLevelId:        o.ServiceLevelID,
		CarrierBillingType:    o.CarrierBillingType,
		CarrierBillingAccount: o.CarrierBillingAccount,
		SupplierId:            o.SellerAccountID,
		SupplierName:          o.SupplierName,
		SupplierNumber:        o.SupplierNumber,
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
		ReceivingOrderId:      o.ReceivingOrderID,
		ReceivingOrderNumber:  o.ReceivingOrderNumber,
		ReceivingOrderStatus:  o.ReceivingOrderStatus,
		Deliveries:            deliveryRefsToProto(o.Deliveries),
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

	if o.IssuedAt != nil {
		info.IssuedAt = timestamppb.New(*o.IssuedAt)
	}
	if o.CompletedAt != nil {
		info.CompletedAt = timestamppb.New(*o.CompletedAt)
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

	if o.BillToIsDropShip != nil {
		addrType := string(constants.AddressTypeStandard)
		if *o.BillToIsDropShip {
			addrType = string(constants.AddressTypeDropShip)
		}
		info.BillToAddressType = &addrType
	}
	if o.BillToCreatedAt != nil {
		info.BillToAddressCreatedAt = timestamppb.New(*o.BillToCreatedAt)
	}
	if o.BillToUpdatedAt != nil {
		info.BillToAddressUpdatedAt = timestamppb.New(*o.BillToUpdatedAt)
	}

	if o.ShipToIsDropShip != nil {
		addrType := string(constants.AddressTypeStandard)
		if *o.ShipToIsDropShip {
			addrType = string(constants.AddressTypeDropShip)
		}
		info.ShipToAddressType = &addrType
	}
	if o.ShipToCreatedAt != nil {
		info.ShipToAddressCreatedAt = timestamppb.New(*o.ShipToCreatedAt)
	}
	if o.ShipToUpdatedAt != nil {
		info.ShipToAddressUpdatedAt = timestamppb.New(*o.ShipToUpdatedAt)
	}

	if o.Lines != nil {
		lines := make([]*pb.PurchaseOrderLineInfo, len(o.Lines))
		for i, l := range o.Lines {
			lines[i] = purchaseOrderLineToProto(l)
		}
		info.Lines = lines
	}

	if o.Contacts != nil {
		contacts := make([]*pb.EmailContactInfo, len(o.Contacts))
		for i, c := range o.Contacts {
			contacts[i] = emailContactToProto(c)
		}
		info.Contacts = contacts
	}

	if o.ReceivingOrder != nil {
		info.ReceivingOrder = receivingOrderToProto(o.ReceivingOrder)
	}

	return info
}

func purchaseOrderLineToProto(l *domain.PurchaseOrderLine) *pb.PurchaseOrderLineInfo {
	if l == nil {
		return nil
	}

	info := &pb.PurchaseOrderLineInfo{
		Id:                                   l.ID,
		LineItemNumber:                       l.LineItemNumber,
		ProductSku:                           l.ProductSKU,
		ProductDescription:                   l.ProductDescription,
		ProductId:                            l.ProductID,
		ItemId:                               l.ItemID,
		ItemSku:                              l.ItemSKU,
		QuantityId:                           l.QuantityID,
		QuantityValue:                        l.QuantityValue,
		QuantityUnitId:                       l.QuantityUnitID,
		QuantityUnitName:                     l.QuantityUnitName,
		QuantityUnitAbbreviation:             l.QuantityUnitAbbreviation,
		QuantityUnitType:                     l.QuantityUnitType,
		QuantityReceivedValue:                l.QuantityReceivedValue,
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
		UnitPriceCreatedAt:                   timestamppb.New(l.UnitPriceCreatedAt),
		UnitPriceUpdatedAt:                   timestamppb.New(l.UnitPriceUpdatedAt),
		CreatedAt:                            timestamppb.New(l.CreatedAt),
		UpdatedAt:                            timestamppb.New(l.UpdatedAt),
	}

	return info
}

func emailContactToProto(c *domain.PurchaseOrderEmailContact) *pb.EmailContactInfo {
	if c == nil {
		return nil
	}

	return &pb.EmailContactInfo{
		Id:            c.ID,
		AccountUserId: c.AccountUserID,
	}
}

func (h *purchaseGRPCHandler) ListPurchaseOrders(ctx context.Context, req *pb.ListPurchaseOrdersRequest) (*pb.ListPurchaseOrdersResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListPurchaseOrdersParams{
		Cursor:      req.Cursor,
		Limit:       req.Limit,
		Query:       req.Query,
		StatusCodes: req.StatusCodes,
		ItemIDs:     req.ItemIds,
		SupplierIDs: req.SupplierIds,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		Includes:    req.Includes,
	}

	result, apiErr := h.purchaseOrderSvc.ListPurchaseOrders(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	orders := make([]*pb.PurchaseOrderSummaryInfo, len(result.PurchaseOrders))
	for i, o := range result.PurchaseOrders {
		orders[i] = purchaseOrderSummaryToProto(o)
	}

	return &pb.ListPurchaseOrdersResponse{
		PurchaseOrders: orders,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *purchaseGRPCHandler) GetPurchaseOrder(ctx context.Context, req *pb.GetPurchaseOrderRequest) (*pb.GetPurchaseOrderResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.GetPurchaseOrderParams{
		PurchaseOrderID: req.Id,
		Includes:        req.Includes,
	}

	order, apiErr := h.purchaseOrderSvc.GetPurchaseOrder(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetPurchaseOrderResponse{
		PurchaseOrder: purchaseOrderToProto(order),
	}, nil
}

// BatchGetPurchaseOrdersByIDs returns purchase orders by ID for the api-gateway include resolver. It reuses the authorized single-get path per id; ids the caller cannot access or that no longer exist are omitted so the resolver leaves those references null.
func (h *purchaseGRPCHandler) BatchGetPurchaseOrdersByIDs(ctx context.Context, req *pb.BatchGetPurchaseOrdersByIDsRequest) (*pb.BatchGetPurchaseOrdersByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	orders := make([]*pb.PurchaseOrderInfo, 0, len(req.Ids))
	for _, poID := range req.Ids {
		if poID == "" {
			continue
		}
		order, apiErr := h.purchaseOrderSvc.GetPurchaseOrder(ctx, domain.GetPurchaseOrderParams{PurchaseOrderID: poID})
		if apiErr != nil {
			continue
		}
		orders = append(orders, purchaseOrderToProto(order))
	}

	return &pb.BatchGetPurchaseOrdersByIDsResponse{PurchaseOrders: orders}, nil
}

// BatchGetPurchaseOrderLinesByIDs returns purchase order lines by ID for the api-gateway include
// resolver, so a receiving or delivery line can name the line it was raised from. The service scopes
// them through the order they belong to; a line the caller cannot reach is omitted rather than an
// error, so the resolver leaves that reference null.
func (h *purchaseGRPCHandler) BatchGetPurchaseOrderLinesByIDs(ctx context.Context, req *pb.BatchGetPurchaseOrderLinesByIDsRequest) (*pb.BatchGetPurchaseOrderLinesByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	lines, apiErr := h.purchaseOrderSvc.BatchGetPurchaseOrderLinesByIDs(ctx, req.Ids)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	infos := make([]*pb.PurchaseOrderLineInfo, 0, len(lines))
	for _, l := range lines {
		infos = append(infos, purchaseOrderLineToProto(l))
	}

	return &pb.BatchGetPurchaseOrderLinesByIDsResponse{Lines: infos}, nil
}

func (h *purchaseGRPCHandler) CreatePurchaseOrder(ctx context.Context, req *pb.CreatePurchaseOrderRequest) (*pb.CreatePurchaseOrderResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	lines := make([]domain.CreatePurchaseOrderLineInput, len(req.Lines))
	for i, l := range req.Lines {
		lines[i] = domain.CreatePurchaseOrderLineInput{
			ProductID:                  l.ProductId,
			ItemID:                     l.ItemId,
			ProductSKU:                 l.ProductSku,
			ProductDescription:         l.ProductDescription,
			QuantityValue:              l.QuantityValue,
			QuantityUnitID:             l.QuantityUnitId,
			UnitPriceValue:             l.UnitPriceValue,
			UnitPriceNumeratorUnitID:   l.UnitPriceNumeratorUnitId,
			UnitPriceDenominatorUnitID: l.UnitPriceDenominatorUnitId,
			UnitCostValue:              l.UnitCostValue,
			UnitCostNumeratorUnitID:    l.UnitCostNumeratorUnitId,
			UnitCostDenominatorUnitID:  l.UnitCostDenominatorUnitId,
		}
	}

	params := domain.CreatePurchaseOrderParams{
		SupplierAccountID:     req.SupplierAccountId,
		Note:                  req.Note,
		CarrierID:             req.CarrierId,
		ServiceLevelID:        req.ServiceLevelId,
		CarrierBillingType:    req.CarrierBillingType,
		CarrierBillingAccount: req.CarrierBillingAccount,
		PriorityCode:          req.PriorityCode,
		ShippingTermID:        req.ShippingTermId,
		PaymentTermID:         req.PaymentTermId,
		PromisedAt:            req.PromisedAt,
		BillToName:            req.BillToName,
		BillToStreetLine1:     req.BillToStreetLine_1,
		BillToStreetLine2:     req.BillToStreetLine_2,
		BillToLocality:        req.BillToLocality,
		BillToState:           req.BillToState,
		BillToPostalCode:      req.BillToPostalCode,
		BillToCountry:         req.BillToCountry,
		ShipToName:            req.ShipToName,
		ShipToStreetLine1:     req.ShipToStreetLine_1,
		ShipToStreetLine2:     req.ShipToStreetLine_2,
		ShipToLocality:        req.ShipToLocality,
		ShipToState:           req.ShipToState,
		ShipToPostalCode:      req.ShipToPostalCode,
		ShipToCountry:         req.ShipToCountry,
		Lines:                 lines,
		ContactAccountUserIDs: req.ContactAccountUserIds,
		Includes:              req.Includes,
	}

	order, apiErr := h.purchaseOrderSvc.CreatePurchaseOrder(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreatePurchaseOrderResponse{
		PurchaseOrder: purchaseOrderToProto(order),
	}, nil
}

func (h *purchaseGRPCHandler) UpdatePurchaseOrder(ctx context.Context, req *pb.UpdatePurchaseOrderRequest) (*pb.UpdatePurchaseOrderResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdatePurchaseOrderParams{
		PurchaseOrderID:       req.Id,
		Note:                  req.Note,
		Number:                req.Number,
		PriorityCode:          req.PriorityCode,
		BillingAddressID:      req.BillingAddressId,
		ShippingAddressID:     req.ShippingAddressId,
		PromisedAt:            req.PromisedAt,
		ContactAccountUserIDs: req.ContactAccountUserIds,
		Includes:              req.Includes,
	}

	order, apiErr := h.purchaseOrderSvc.UpdatePurchaseOrder(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdatePurchaseOrderResponse{
		PurchaseOrder: purchaseOrderToProto(order),
	}, nil
}

func (h *purchaseGRPCHandler) DeletePurchaseOrder(ctx context.Context, req *pb.DeletePurchaseOrderRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.purchaseOrderSvc.DeletePurchaseOrder(ctx, domain.DeletePurchaseOrderParams{
		PurchaseOrderID: req.Id,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *purchaseGRPCHandler) BulkDeletePurchaseOrders(ctx context.Context, req *pb.BulkDeletePurchaseOrdersRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.purchaseOrderSvc.BulkDeletePurchaseOrders(ctx, domain.BulkDeletePurchaseOrdersParams{
		PurchaseOrderIDs: req.Ids,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *purchaseGRPCHandler) ChangePurchaseOrderStatus(ctx context.Context, req *pb.ChangePurchaseOrderStatusRequest) (*pb.ChangePurchaseOrderStatusResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	order, apiErr := h.purchaseOrderSvc.ChangePurchaseOrderStatus(ctx, domain.ChangePurchaseOrderStatusParams{
		PurchaseOrderID: req.Id,
		StatusChange:    req.StatusChange,
		SendEmail:       req.SendEmail,
		Includes:        req.Includes,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.ChangePurchaseOrderStatusResponse{
		PurchaseOrder: purchaseOrderToProto(order),
	}, nil
}

func (h *purchaseGRPCHandler) CreatePurchaseOrderLine(ctx context.Context, req *pb.CreatePurchaseOrderLineRequest) (*pb.CreatePurchaseOrderLineResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.CreatePurchaseOrderLineParams{
		SalesOrderID:               req.PurchaseOrderId,
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
	}

	line, apiErr := h.purchaseOrderLineSvc.CreatePurchaseOrderLine(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreatePurchaseOrderLineResponse{
		PurchaseOrderLine: purchaseOrderLineToProto(line),
	}, nil
}

func (h *purchaseGRPCHandler) UpdatePurchaseOrderLine(ctx context.Context, req *pb.UpdatePurchaseOrderLineRequest) (*pb.UpdatePurchaseOrderLineResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdatePurchaseOrderLineParams{
		PurchaseOrderLineID:        req.Id,
		SalesOrderID:               req.PurchaseOrderId,
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
	}

	line, apiErr := h.purchaseOrderLineSvc.UpdatePurchaseOrderLine(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdatePurchaseOrderLineResponse{
		PurchaseOrderLine: purchaseOrderLineToProto(line),
	}, nil
}

func (h *purchaseGRPCHandler) DeletePurchaseOrderLine(ctx context.Context, req *pb.DeletePurchaseOrderLineRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.purchaseOrderLineSvc.DeletePurchaseOrderLine(ctx, domain.DeletePurchaseOrderLineParams{
		PurchaseOrderLineID: req.Id,
		SalesOrderID:        req.PurchaseOrderId,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func deliveryRefsToProto(refs []domain.DocumentRef) []*pb.DocumentRefInfo {
	if len(refs) == 0 {
		return nil
	}
	out := make([]*pb.DocumentRefInfo, len(refs))
	for i, r := range refs {
		out[i] = &pb.DocumentRefInfo{Id: r.ID, Number: r.Number, Status: r.Status}
	}
	return out
}
