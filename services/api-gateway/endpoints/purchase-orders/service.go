package purchaseorderep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type PurchaseOrderSvc interface {
	ListPurchaseOrders(ctx context.Context, req *ListPurchaseOrdersRequest) (*apiresource.List[apiresource.PurchaseOrderSummary], *apierror.APIError)
	GetPurchaseOrder(ctx context.Context, req *RetrievePurchaseOrderRequest) (*apiresource.PurchaseOrderDetail, *apierror.APIError)
	CreatePurchaseOrder(ctx context.Context, req *CreatePurchaseOrderRequest) (*apiresource.PurchaseOrderDetail, *apierror.APIError)
	UpdatePurchaseOrder(ctx context.Context, req *UpdatePurchaseOrderRequest) (*apiresource.PurchaseOrderDetail, *apierror.APIError)
	DeletePurchaseOrder(ctx context.Context, req *DeletePurchaseOrderRequest) (*apiresource.EmptyResource, *apierror.APIError)
	BulkDeletePurchaseOrders(ctx context.Context, req *BulkDeletePurchaseOrdersRequest) (*apiresource.EmptyResource, *apierror.APIError)
	ChangePurchaseOrderStatus(ctx context.Context, req *ChangePurchaseOrderStatusRequest) (*apiresource.PurchaseOrderDetail, *apierror.APIError)
	CreatePurchaseOrderLine(ctx context.Context, req *CreatePurchaseOrderLineRequest) (*apiresource.PurchaseOrderLineDetail, *apierror.APIError)
	UpdatePurchaseOrderLine(ctx context.Context, req *UpdatePurchaseOrderLineRequest) (*apiresource.PurchaseOrderLineDetail, *apierror.APIError)
	DeletePurchaseOrderLine(ctx context.Context, req *DeletePurchaseOrderLineRequest) (*apiresource.EmptyResource, *apierror.APIError)
	ListPurchaseOrderStatuses(ctx context.Context, req *ListPurchaseOrderStatusesRequest) (*apiresource.List[apiresource.SalesOrderStatus], *apierror.APIError)
}

type PurchaseOrderSvcConfig struct {
	CoreClient  pb.CorePurchaseServiceClient
	SalesClient pb.CoreSalesServiceClient
}

type purchaseOrderSvcImpl struct {
	coreClient  pb.CorePurchaseServiceClient
	salesClient pb.CoreSalesServiceClient
}

var purchaseOrderEpSvcTracer = tracing.GetTracer("api-gateway.endpoints.purchase-orders.service")

var purchaseOrderIncludes = []string{"supplier", "bill_to_address", "ship_to_address", "carrier", "service_level", "payment_term", "shipping_term", "receiving_order", "lines", "contacts"}

func (c *PurchaseOrderSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("purchase order endpoint service: core client is required")
	}
	if c.SalesClient == nil {
		return fmt.Errorf("purchase order endpoint service: sales client is required")
	}
	return nil
}

func NewPurchaseOrderSvc(config *PurchaseOrderSvcConfig) PurchaseOrderSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &purchaseOrderSvcImpl{
		coreClient:  config.CoreClient,
		salesClient: config.SalesClient,
	}
}

func (m *purchaseOrderSvcImpl) ListPurchaseOrders(ctx context.Context, req *ListPurchaseOrdersRequest) (*apiresource.List[apiresource.PurchaseOrderSummary], *apierror.APIError) {
	pbReq := &pb.ListPurchaseOrdersRequest{
		Cursor:      req.Cursor,
		Limit:       req.Limit,
		Query:       req.Query,
		StatusCodes: req.StatusCodes,
		ItemIds:     req.ItemIDs,
		SupplierIds: req.SupplierIDs,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, purchaseOrderEpSvcTracer, "service.purchase_orders.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListPurchaseOrdersResponse, error) {
			return m.coreClient.ListPurchaseOrders(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	orders := make([]apiresource.PurchaseOrderSummary, len(resp.PurchaseOrders))
	for i, o := range resp.PurchaseOrders {
		orders[i] = purchaseOrderSummaryFromProto(o)
	}

	return apiresource.NewList(orders, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *purchaseOrderSvcImpl) GetPurchaseOrder(ctx context.Context, req *RetrievePurchaseOrderRequest) (*apiresource.PurchaseOrderDetail, *apierror.APIError) {
	pbReq := &pb.GetPurchaseOrderRequest{
		Id:       req.PurchaseOrderID,
		Includes: purchaseOrderIncludes,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, purchaseOrderEpSvcTracer, "service.purchase_orders.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetPurchaseOrderResponse, error) {
			return m.coreClient.GetPurchaseOrder(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := purchaseOrderDetailFromProto(resp.PurchaseOrder)
	stashPurchaseOrderDetailMeta(ctx, resp.PurchaseOrder, &result)
	return &result, nil
}

func (m *purchaseOrderSvcImpl) CreatePurchaseOrder(ctx context.Context, req *CreatePurchaseOrderRequest) (*apiresource.PurchaseOrderDetail, *apierror.APIError) {
	lines := make([]*pb.CreatePurchaseOrderLineInput, len(req.Lines))
	for i, l := range req.Lines {
		lines[i] = &pb.CreatePurchaseOrderLineInput{
			ProductId:                  l.ProductID,
			ItemId:                     l.ItemID,
			ProductSku:                 l.ProductSKU,
			ProductDescription:         l.ProductDescription,
			QuantityValue:              l.QuantityValue,
			QuantityUnitId:             l.QuantityUnitID,
			UnitPriceValue:             l.UnitPriceValue,
			UnitPriceNumeratorUnitId:   l.UnitPriceNumeratorUnitID,
			UnitPriceDenominatorUnitId: l.UnitPriceDenominatorUnitID,
			UnitCostValue:              l.UnitCostValue,
			UnitCostNumeratorUnitId:    l.UnitCostNumeratorUnitID,
			UnitCostDenominatorUnitId:  l.UnitCostDenominatorUnitID,
		}
	}

	pbReq := &pb.CreatePurchaseOrderRequest{
		SupplierAccountId:     req.SupplierAccountID,
		Note:                  req.Note,
		CarrierId:             req.CarrierID,
		ServiceLevelId:        req.ServiceLevelID,
		CarrierBillingType:    req.CarrierBillingType,
		CarrierBillingAccount: req.CarrierBillingAccount,
		PriorityCode:          req.PriorityCode,
		ShippingTermId:        req.ShippingTermID,
		PaymentTermId:         req.PaymentTermID,
		BillToName:            req.BillToName,
		BillToStreetLine_1:    req.BillToStreetLine1,
		BillToStreetLine_2:    req.BillToStreetLine2,
		BillToLocality:        req.BillToLocality,
		BillToState:           req.BillToState,
		BillToPostalCode:      req.BillToPostalCode,
		BillToCountry:         req.BillToCountry,
		ShipToName:            req.ShipToName,
		ShipToStreetLine_1:    req.ShipToStreetLine1,
		ShipToStreetLine_2:    req.ShipToStreetLine2,
		ShipToLocality:        req.ShipToLocality,
		ShipToState:           req.ShipToState,
		ShipToPostalCode:      req.ShipToPostalCode,
		ShipToCountry:         req.ShipToCountry,
		Lines:                 lines,
		ContactAccountUserIds: req.ContactAccountUserIDs,
		PromisedAt:            req.PromisedAt,
		Includes:              purchaseOrderIncludes,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, purchaseOrderEpSvcTracer, "service.purchase_orders.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreatePurchaseOrderResponse, error) {
			return m.coreClient.CreatePurchaseOrder(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := purchaseOrderDetailFromProto(resp.PurchaseOrder)
	stashPurchaseOrderDetailMeta(ctx, resp.PurchaseOrder, &result)
	return &result, nil
}

func (m *purchaseOrderSvcImpl) UpdatePurchaseOrder(ctx context.Context, req *UpdatePurchaseOrderRequest) (*apiresource.PurchaseOrderDetail, *apierror.APIError) {
	pbReq := &pb.UpdatePurchaseOrderRequest{
		Id:                    req.PurchaseOrderID,
		Note:                  req.Note,
		Number:                req.Number,
		PriorityCode:          req.PriorityCode,
		BillingAddressId:      req.BillingAddressID,
		ShippingAddressId:     req.ShippingAddressID,
		PromisedAt:            req.PromisedAt,
		ContactAccountUserIds: req.ContactAccountUserIDs,
		Includes:              purchaseOrderIncludes,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, purchaseOrderEpSvcTracer, "service.purchase_orders.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdatePurchaseOrderResponse, error) {
			return m.coreClient.UpdatePurchaseOrder(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := purchaseOrderDetailFromProto(resp.PurchaseOrder)
	stashPurchaseOrderDetailMeta(ctx, resp.PurchaseOrder, &result)
	return &result, nil
}

func (m *purchaseOrderSvcImpl) DeletePurchaseOrder(ctx context.Context, req *DeletePurchaseOrderRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeletePurchaseOrderRequest{Id: req.PurchaseOrderID}

	_, apiErr := grpcutil.CallRPC(ctx, purchaseOrderEpSvcTracer, "service.purchase_orders.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeletePurchaseOrder(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func (m *purchaseOrderSvcImpl) BulkDeletePurchaseOrders(ctx context.Context, req *BulkDeletePurchaseOrdersRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.BulkDeletePurchaseOrdersRequest{Ids: req.PurchaseOrderIDs}

	_, apiErr := grpcutil.CallRPC(ctx, purchaseOrderEpSvcTracer, "service.purchase_orders.bulk_delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.BulkDeletePurchaseOrders(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func (m *purchaseOrderSvcImpl) ChangePurchaseOrderStatus(ctx context.Context, req *ChangePurchaseOrderStatusRequest) (*apiresource.PurchaseOrderDetail, *apierror.APIError) {
	pbReq := &pb.ChangePurchaseOrderStatusRequest{
		Id:           req.PurchaseOrderID,
		StatusChange: req.StatusChange,
		SendEmail:    req.SendEmail,
		Includes:     purchaseOrderIncludes,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, purchaseOrderEpSvcTracer, "service.purchase_orders.change_status", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ChangePurchaseOrderStatusResponse, error) {
			return m.coreClient.ChangePurchaseOrderStatus(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := purchaseOrderDetailFromProto(resp.PurchaseOrder)
	stashPurchaseOrderDetailMeta(ctx, resp.PurchaseOrder, &result)
	return &result, nil
}

func (m *purchaseOrderSvcImpl) CreatePurchaseOrderLine(ctx context.Context, req *CreatePurchaseOrderLineRequest) (*apiresource.PurchaseOrderLineDetail, *apierror.APIError) {
	pbReq := &pb.CreatePurchaseOrderLineRequest{
		PurchaseOrderId:            req.PurchaseOrderID,
		ProductId:                  req.ProductID,
		ItemId:                     req.ItemID,
		ProductSku:                 req.ProductSKU,
		ProductDescription:         req.ProductDescription,
		QuantityValue:              req.QuantityValue,
		QuantityUnitId:             req.QuantityUnitID,
		UnitPriceValue:             req.UnitPriceValue,
		UnitPriceNumeratorUnitId:   req.UnitPriceNumeratorUnitID,
		UnitPriceDenominatorUnitId: req.UnitPriceDenominatorUnitID,
		UnitCostValue:              req.UnitCostValue,
		UnitCostNumeratorUnitId:    req.UnitCostNumeratorUnitID,
		UnitCostDenominatorUnitId:  req.UnitCostDenominatorUnitID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, purchaseOrderEpSvcTracer, "service.purchase_orders.create_line", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreatePurchaseOrderLineResponse, error) {
			return m.coreClient.CreatePurchaseOrderLine(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := purchaseOrderLineDetailFromProto(resp.PurchaseOrderLine)
	return &result, nil
}

func (m *purchaseOrderSvcImpl) UpdatePurchaseOrderLine(ctx context.Context, req *UpdatePurchaseOrderLineRequest) (*apiresource.PurchaseOrderLineDetail, *apierror.APIError) {
	pbReq := &pb.UpdatePurchaseOrderLineRequest{
		PurchaseOrderId:            req.PurchaseOrderID,
		Id:                         req.PurchaseOrderLineID,
		ProductId:                  req.ProductID,
		ItemId:                     req.ItemID,
		ProductSku:                 req.ProductSKU,
		ProductDescription:         req.ProductDescription,
		QuantityValue:              req.QuantityValue,
		QuantityUnitId:             req.QuantityUnitID,
		UnitPriceValue:             req.UnitPriceValue,
		UnitPriceNumeratorUnitId:   req.UnitPriceNumeratorUnitID,
		UnitPriceDenominatorUnitId: req.UnitPriceDenominatorUnitID,
		UnitCostValue:              req.UnitCostValue,
		UnitCostNumeratorUnitId:    req.UnitCostNumeratorUnitID,
		UnitCostDenominatorUnitId:  req.UnitCostDenominatorUnitID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, purchaseOrderEpSvcTracer, "service.purchase_orders.update_line", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdatePurchaseOrderLineResponse, error) {
			return m.coreClient.UpdatePurchaseOrderLine(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := purchaseOrderLineDetailFromProto(resp.PurchaseOrderLine)
	return &result, nil
}

func (m *purchaseOrderSvcImpl) DeletePurchaseOrderLine(ctx context.Context, req *DeletePurchaseOrderLineRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeletePurchaseOrderLineRequest{
		PurchaseOrderId: req.PurchaseOrderID,
		Id:              req.PurchaseOrderLineID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, purchaseOrderEpSvcTracer, "service.purchase_orders.delete_line", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeletePurchaseOrderLine(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func (m *purchaseOrderSvcImpl) ListPurchaseOrderStatuses(ctx context.Context, req *ListPurchaseOrderStatusesRequest) (*apiresource.List[apiresource.SalesOrderStatus], *apierror.APIError) {
	pbReq := &pb.ListSalesOrderStatusesRequest{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, purchaseOrderEpSvcTracer, "service.purchase_orders.list_statuses", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListSalesOrderStatusesResponse, error) {
			return m.salesClient.ListSalesOrderStatuses(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	if resp == nil {
		return apiresource.NewList[apiresource.SalesOrderStatus](nil, apiresource.PageInfo{}), nil
	}

	statuses := make([]apiresource.SalesOrderStatus, len(resp.SalesOrderStatuses))
	for i, s := range resp.SalesOrderStatuses {
		statuses[i] = apiresource.SalesOrderStatus{
			ID:        s.Id,
			Object:    constants.ObjectTypeSalesOrderStatus,
			Code:      constants.SalesOrderStatusCode(s.Code),
			Name:      s.Name,
			CreatedAt: grpcutil.TimestampToTime(s.CreatedAt),
			UpdatedAt: grpcutil.TimestampToTime(s.UpdatedAt),
		}
	}

	return apiresource.NewList(statuses, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func purchaseOrderSummaryFromProto(info *pb.PurchaseOrderSummaryInfo) apiresource.PurchaseOrderSummary {
	s := apiresource.PurchaseOrderSummary{
		ID:     info.Id,
		Object: constants.ObjectTypePurchaseOrder,
		Number: info.Number,
		Supplier: &apiresource.Supplier{
			ID:     info.SupplierId,
			Object: constants.ObjectTypeSupplier,
			Name:   info.SupplierName,
			Number: info.SupplierNumber,
		},
		Status: &apiresource.SalesOrderStatusDetail{
			Code:   info.StatusCode,
			Object: constants.ObjectTypeSalesOrderStatus,
			Name:   info.StatusName,
		},
		Type: &apiresource.SalesOrderType{
			Code:   info.TypeCode,
			Object: constants.ObjectTypeSalesOrderType,
			Name:   info.TypeName,
		},
		Priority:             apiresource.ExpandablePriorityStub("", constants.PriorityCode(info.PriorityCode), info.PriorityName, grpcutil.TimestampToTime(info.CreatedAt)),
		LineCount:            info.LineCount,
		IsAcknowledgmentSent: info.IsAcknowledgmentSent,
		CreatedAt:            grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt:            grpcutil.TimestampToTime(info.UpdatedAt),
	}

	if info.PriorityId != nil {
		s.Priority.ID = *info.PriorityId
	}

	if info.IssuedAt != nil {
		t := grpcutil.TimestampToTime(info.IssuedAt)
		s.IssuedAt = &t
	}
	if info.CompletedAt != nil {
		t := grpcutil.TimestampToTime(info.CompletedAt)
		s.CompletedAt = &t
	}

	return s
}

func purchaseOrderDetailFromProto(info *pb.PurchaseOrderInfo) apiresource.PurchaseOrderDetail {
	d := apiresource.PurchaseOrderDetail{
		ID:                    info.Id,
		Object:                constants.ObjectTypePurchaseOrder,
		Number:                info.Number,
		Note:                  info.Note,
		IsAcknowledgmentSent:  info.IsAcknowledgmentSent,
		CarrierBillingType:    info.CarrierBillingType,
		CarrierBillingAccount: info.CarrierBillingAccount,
		Status: &apiresource.SalesOrderStatusDetail{
			Code:   info.StatusCode,
			Object: constants.ObjectTypeSalesOrderStatus,
			Name:   info.StatusName,
		},
		Type: &apiresource.SalesOrderType{
			Code:   info.TypeCode,
			Object: constants.ObjectTypeSalesOrderType,
			Name:   info.TypeName,
		},
		Priority:  apiresource.ExpandablePriorityStub("", constants.PriorityCode(info.PriorityCode), info.PriorityName, grpcutil.TimestampToTime(info.CreatedAt)),
		CreatedAt: grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(info.UpdatedAt),
	}

	if info.PriorityId != nil {
		d.Priority.ID = *info.PriorityId
	}

	if info.IssuedAt != nil {
		t := grpcutil.TimestampToTime(info.IssuedAt)
		d.IssuedAt = &t
	}
	if info.CompletedAt != nil {
		t := grpcutil.TimestampToTime(info.CompletedAt)
		d.CompletedAt = &t
	}
	if info.PromisedAt != nil {
		t := grpcutil.TimestampToTime(info.PromisedAt)
		d.ScheduledAt = &t
	}

	return d
}

func stashPurchaseOrderDetailMeta(ctx context.Context, info *pb.PurchaseOrderInfo, d *apiresource.PurchaseOrderDetail) {
	if info == nil {
		return
	}

	meta := resourcekit.GetLoadMeta(ctx)

	meta.Set(constants.ObjectTypePurchaseOrder, d.ID, "supplier", &apiresource.Supplier{
		ID:     info.SupplierId,
		Object: constants.ObjectTypeSupplier,
		Name:   info.SupplierName,
		Number: info.SupplierNumber,
	})

	if info.BillingAddressId != "" {
		meta.Set(constants.ObjectTypePurchaseOrder, d.ID, "bill_to_address",
			buildAddressFromProto(
				info.BillingAddressId, info.BillToAddressType,
				info.BillToName, info.BillToStreetLine_1, info.BillToStreetLine_2,
				info.BillToLocality, info.BillToState, info.BillToPostalCode, info.BillToCountry,
				info.BillToPhone, info.BillToEmail,
				info.BillToAddressCreatedAt, info.BillToAddressUpdatedAt,
			))
	}

	if info.ShippingAddressId != "" {
		meta.Set(constants.ObjectTypePurchaseOrder, d.ID, "ship_to_address",
			buildAddressFromProto(
				info.ShippingAddressId, info.ShipToAddressType,
				info.ShipToName, info.ShipToStreetLine_1, info.ShipToStreetLine_2,
				info.ShipToLocality, info.ShipToState, info.ShipToPostalCode, info.ShipToCountry,
				info.ShipToPhone, info.ShipToEmail,
				info.ShipToAddressCreatedAt, info.ShipToAddressUpdatedAt,
			))
	}

	if info.CarrierId != nil {
		carrier := &apiresource.Carrier{
			ID:     *info.CarrierId,
			Object: constants.ObjectTypeCarrier,
		}
		if info.CarrierName != nil {
			carrier.Name = *info.CarrierName
		}
		if info.CarrierIsPortalEnabled != nil && *info.CarrierIsPortalEnabled {
			carrier.CustomerPortalVisibility = constants.CustomerPortalVisibilityVisible
		} else {
			carrier.CustomerPortalVisibility = constants.CustomerPortalVisibilityHidden
		}
		if info.CarrierCreatedAt != nil {
			carrier.CreatedAt = info.CarrierCreatedAt.AsTime()
		}
		if info.CarrierUpdatedAt != nil {
			carrier.UpdatedAt = info.CarrierUpdatedAt.AsTime()
		}
		meta.Set(constants.ObjectTypePurchaseOrder, d.ID, "carrier", carrier)
	}

	if info.ServiceLevelId != nil {
		sl := &apiresource.ServiceLevel{
			ID:     *info.ServiceLevelId,
			Object: constants.ObjectTypeServiceLevel,
		}
		if info.ServiceLevelName != nil {
			sl.Name = *info.ServiceLevelName
		}
		if info.ServiceLevelIsPortalEnabled != nil && *info.ServiceLevelIsPortalEnabled {
			sl.CustomerPortalVisibility = constants.CustomerPortalVisibilityVisible
		} else {
			sl.CustomerPortalVisibility = constants.CustomerPortalVisibilityHidden
		}
		if info.ServiceLevelToken != nil {
			sl.ServiceLevelToken = constants.ServiceLevelCode(*info.ServiceLevelToken)
		}
		if info.ServiceLevelCreatedAt != nil {
			sl.CreatedAt = info.ServiceLevelCreatedAt.AsTime()
		}
		if info.ServiceLevelUpdatedAt != nil {
			sl.UpdatedAt = info.ServiceLevelUpdatedAt.AsTime()
		}
		meta.Set(constants.ObjectTypePurchaseOrder, d.ID, "service_level", sl)
	}

	if info.PaymentTermId != nil {
		pt := &apiresource.PaymentTerm{
			ID:     *info.PaymentTermId,
			Object: constants.ObjectTypePaymentTerm,
		}
		if info.PaymentTermName != nil {
			pt.Name = *info.PaymentTermName
		}
		if info.PaymentTermIsActive != nil && *info.PaymentTermIsActive {
			pt.Status = constants.PaymentTermStatusActive
		} else {
			pt.Status = constants.PaymentTermStatusInactive
		}
		if info.PaymentTermCreatedAt != nil {
			pt.CreatedAt = info.PaymentTermCreatedAt.AsTime()
		}
		if info.PaymentTermUpdatedAt != nil {
			pt.UpdatedAt = info.PaymentTermUpdatedAt.AsTime()
		}
		meta.Set(constants.ObjectTypePurchaseOrder, d.ID, "payment_term", pt)
	}

	if info.ShippingTermId != nil {
		st := &apiresource.ShippingTerm{
			ID:     *info.ShippingTermId,
			Object: constants.ObjectTypeShippingTerm,
		}
		if info.ShippingTermName != nil {
			st.Name = *info.ShippingTermName
		}
		if info.ShippingTermType != nil {
			st.Type = constants.ShippingTermType(*info.ShippingTermType)
		}
		if info.ShippingTermCreatedAt != nil {
			st.CreatedAt = info.ShippingTermCreatedAt.AsTime()
		}
		if info.ShippingTermUpdatedAt != nil {
			st.UpdatedAt = info.ShippingTermUpdatedAt.AsTime()
		}
		meta.Set(constants.ObjectTypePurchaseOrder, d.ID, "shipping_term", st)
	}

	if info.ReceivingOrderId != nil {
		ro := &apiresource.ReceivingOrder{
			ID:     *info.ReceivingOrderId,
			Object: constants.ObjectTypeReceivingOrder,
		}
		if info.ReceivingOrder != nil {
			roInfo := info.ReceivingOrder
			ro.Number = roInfo.Number
			ro.CreatedAt = grpcutil.TimestampToTime(roInfo.CreatedAt)
			ro.UpdatedAt = grpcutil.TimestampToTime(roInfo.UpdatedAt)
			ro.Note = roInfo.Note
			if roInfo.CompletedAt != nil {
				t := grpcutil.TimestampToTime(roInfo.CompletedAt)
				ro.CompletedAt = &t
			}
			ro.PurchaseOrder = &apiresource.SalesOrderDetail{
				ID:     roInfo.PurchaseOrderId,
				Object: constants.ObjectTypeSalesOrder,
				Number: roInfo.PurchaseOrderNumber,
			}
		}
		meta.Set(constants.ObjectTypePurchaseOrder, d.ID, "receiving_order", ro)
	}

	lines := make([]apiresource.PurchaseOrderLineDetail, len(info.Lines))
	for i, l := range info.Lines {
		lines[i] = purchaseOrderLineDetailFromProto(l)
	}
	meta.Set(constants.ObjectTypePurchaseOrder, d.ID, "lines", apiresource.NewList(lines, apiresource.PageInfo{}))

	contactItems := make([]apiresource.EmailContact, len(info.Contacts))
	for i, c := range info.Contacts {
		contactItems[i] = apiresource.EmailContact{
			ID:     c.Id,
			Object: constants.ObjectTypeEmailContact,
			AccountUser: &apiresource.AccountUser{
				ID:     c.AccountUserId,
				Object: constants.ObjectTypeAccountUser,
			},
		}
	}
	meta.Set(constants.ObjectTypePurchaseOrder, d.ID, "contacts", apiresource.NewList(contactItems, apiresource.PageInfo{}))
}

func purchaseOrderLineDetailFromProto(info *pb.PurchaseOrderLineInfo) apiresource.PurchaseOrderLineDetail {
	l := apiresource.PurchaseOrderLineDetail{
		ID:                 info.Id,
		Object:             constants.ObjectTypePurchaseOrderLine,
		LineItemNumber:     info.LineItemNumber,
		ProductSKU:         info.ProductSku,
		ProductDescription: info.ProductDescription,
		CreatedAt:          grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt:          grpcutil.TimestampToTime(info.UpdatedAt),
	}

	if info.ItemId != nil {
		item := &apiresource.Item{
			ID:     *info.ItemId,
			Object: constants.ObjectTypeItem,
		}
		if info.ItemSku != nil {
			item.SKU = *info.ItemSku
		}
		l.Item = item
	}

	l.QuantityOrdered = &apiresource.Quantity{
		ID:     info.QuantityId,
		Object: constants.ObjectTypeQuantity,
		Value:  info.QuantityValue,
		Unit: &apiresource.Unit{
			ID:           info.QuantityUnitId,
			Object:       constants.ObjectTypeUnit,
			Name:         info.QuantityUnitName,
			Abbreviation: info.QuantityUnitAbbreviation,
		},
	}

	if info.QuantityReceivedValue != nil {
		l.QuantityReceived = &apiresource.Quantity{
			Object: constants.ObjectTypeQuantity,
			Value:  *info.QuantityReceivedValue,
			Unit: &apiresource.Unit{
				ID:           info.QuantityUnitId,
				Object:       constants.ObjectTypeUnit,
				Name:         info.QuantityUnitName,
				Abbreviation: info.QuantityUnitAbbreviation,
			},
		}
	}

	l.UnitPrice = &apiresource.Rate{
		ID:     info.UnitPriceId,
		Object: constants.ObjectTypeRate,
		Value:  info.UnitPriceValue,
		NumeratorUnit: &apiresource.Unit{
			ID:           info.UnitPriceNumeratorUnitId,
			Object:       constants.ObjectTypeUnit,
			Abbreviation: info.UnitPriceNumeratorUnitAbbreviation,
		},
		DenominatorUnit: &apiresource.Unit{
			ID:           info.UnitPriceDenominatorUnitId,
			Object:       constants.ObjectTypeUnit,
			Abbreviation: info.UnitPriceDenominatorUnitAbbreviation,
		},
		DisplayValue: apiresource.FormatRateDisplayValue(info.UnitPriceValue, info.UnitPriceNumeratorUnitAbbreviation, "", info.UnitPriceDenominatorUnitAbbreviation),
	}

	if info.UnitCostId != nil {
		l.UnitCost = &apiresource.Rate{
			ID:     *info.UnitCostId,
			Object: constants.ObjectTypeRate,
		}
		var unitCostValue, unitCostNumeratorAbbr, unitCostDenominatorAbbr string
		if info.UnitCostValue != nil {
			l.UnitCost.Value = *info.UnitCostValue
			unitCostValue = *info.UnitCostValue
		}
		if info.UnitCostNumeratorUnitId != nil {
			l.UnitCost.NumeratorUnit = &apiresource.Unit{
				ID:     *info.UnitCostNumeratorUnitId,
				Object: constants.ObjectTypeUnit,
			}
			if info.UnitCostNumeratorUnitAbbreviation != nil {
				l.UnitCost.NumeratorUnit.Abbreviation = *info.UnitCostNumeratorUnitAbbreviation
				unitCostNumeratorAbbr = *info.UnitCostNumeratorUnitAbbreviation
			}
		}
		if info.UnitCostDenominatorUnitId != nil {
			l.UnitCost.DenominatorUnit = &apiresource.Unit{
				ID:     *info.UnitCostDenominatorUnitId,
				Object: constants.ObjectTypeUnit,
			}
			if info.UnitCostDenominatorUnitAbbreviation != nil {
				l.UnitCost.DenominatorUnit.Abbreviation = *info.UnitCostDenominatorUnitAbbreviation
				unitCostDenominatorAbbr = *info.UnitCostDenominatorUnitAbbreviation
			}
		}
		l.UnitCost.DisplayValue = apiresource.FormatRateDisplayValue(unitCostValue, unitCostNumeratorAbbr, "", unitCostDenominatorAbbr)
	}

	return l
}

func buildAddressFromProto(
	id string, addrType *string,
	name, line1, line2, locality, state, postalCode, country, phone, email *string,
	createdAt, updatedAt *timestamppb.Timestamp,
) *apiresource.Address {
	addr := &apiresource.Address{
		ID:     id,
		Object: constants.ObjectTypeAddress,
		Type:   constants.AddressTypeStandard,
		Phone:  phone,
		Email:  email,
	}

	if addrType != nil {
		addr.Type = constants.AddressType(*addrType)
	}

	if name != nil {
		addr.Name = *name
	}

	if createdAt != nil {
		addr.CreatedAt = createdAt.AsTime()
	}
	if updatedAt != nil {
		addr.UpdatedAt = updatedAt.AsTime()
	}

	countryStr := ""
	if country != nil {
		countryStr = *country
	}

	addr.Geolocation = &apiresource.Geolocation{
		Object:      constants.ObjectTypeGeolocation,
		StreetLine1: line1,
		StreetLine2: line2,
		Locality:    locality,
		State:       state,
		PostalCode:  postalCode,
		Country:     countryStr,
	}

	return addr
}
