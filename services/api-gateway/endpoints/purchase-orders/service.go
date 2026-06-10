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
	"github.com/augno/api/shared/safeconv"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type PurchaseOrderSvc interface {
	ListPurchaseOrders(ctx context.Context, req *ListPurchaseOrdersRequest) (*apiresource.List[apiresource.PurchaseOrder], *apierror.APIError)
	GetPurchaseOrder(ctx context.Context, req *RetrievePurchaseOrderRequest) (*apiresource.PurchaseOrder, *apierror.APIError)
	CreatePurchaseOrder(ctx context.Context, req *CreatePurchaseOrderRequest) (*apiresource.PurchaseOrder, *apierror.APIError)
	UpdatePurchaseOrder(ctx context.Context, req *UpdatePurchaseOrderRequest) (*apiresource.PurchaseOrder, *apierror.APIError)
	DeletePurchaseOrder(ctx context.Context, req *DeletePurchaseOrderRequest) (*apiresource.EmptyResource, *apierror.APIError)
	BulkDeletePurchaseOrders(ctx context.Context, req *BulkDeletePurchaseOrdersRequest) (*apiresource.EmptyResource, *apierror.APIError)
	ChangePurchaseOrderStatus(ctx context.Context, req *ChangePurchaseOrderStatusRequest) (*apiresource.PurchaseOrder, *apierror.APIError)
	CreatePurchaseOrderLine(ctx context.Context, req *CreatePurchaseOrderLineRequest) (*apiresource.PurchaseOrderLine, *apierror.APIError)
	UpdatePurchaseOrderLine(ctx context.Context, req *UpdatePurchaseOrderLineRequest) (*apiresource.PurchaseOrderLine, *apierror.APIError)
	DeletePurchaseOrderLine(ctx context.Context, req *DeletePurchaseOrderLineRequest) (*apiresource.EmptyResource, *apierror.APIError)
	ListPurchaseOrderStatuses(ctx context.Context, req *ListPurchaseOrderStatusesRequest) (*apiresource.List[apiresource.SalesOrderStatus], *apierror.APIError)
}

type PurchaseOrderSvcConfig struct {
	// CoreClient (required) is the core-service purchasing gRPC client.
	CoreClient pb.CorePurchaseServiceClient

	// SalesClient (required) is the core-service sales gRPC client.
	SalesClient pb.CoreSalesServiceClient
}

type purchaseOrderSvcImpl struct {
	coreClient  pb.CorePurchaseServiceClient
	salesClient pb.CoreSalesServiceClient
}

var purchaseOrderEpSvcTracer = tracing.GetTracer("api-gateway.endpoints.purchase-orders.service")

var purchaseOrderIncludes = []string{"supplier", "bill_to_address", "ship_to_address", "freight", "payment_term", "shipping_term", "receiving_order", "lines", "contacts"}

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

func (m *purchaseOrderSvcImpl) ListPurchaseOrders(ctx context.Context, req *ListPurchaseOrdersRequest) (*apiresource.List[apiresource.PurchaseOrder], *apierror.APIError) {
	pbReq := &pb.ListPurchaseOrdersRequest{
		Cursor:      req.Cursor,
		Limit:       req.Limit,
		Query:       req.Query,
		StatusCodes: req.StatusCodes,
		ItemIds:     req.ItemIDs,
		SupplierIds: req.SupplierIDs,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		// Ask the backend to expand lines when the caller requested them (the rest
		// of the includes are resolved gateway-side from the summary stash).
		Includes: resourcekit.FilterIncludes(ctx, purchaseOrderIncludes...),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, purchaseOrderEpSvcTracer, "service.purchase_orders.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListPurchaseOrdersResponse, error) {
			return m.coreClient.ListPurchaseOrders(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	orders := make([]apiresource.PurchaseOrder, len(resp.PurchaseOrders))
	for i, o := range resp.PurchaseOrders {
		orders[i] = purchaseOrderSummaryFromProto(o)
		stashPurchaseOrderSummaryMeta(ctx, o, &orders[i])
	}

	return apiresource.NewList(orders, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *purchaseOrderSvcImpl) GetPurchaseOrder(ctx context.Context, req *RetrievePurchaseOrderRequest) (*apiresource.PurchaseOrder, *apierror.APIError) {
	pbReq := &pb.GetPurchaseOrderRequest{
		Id:       req.PurchaseOrderID,
		Includes: resourcekit.FilterIncludes(ctx, purchaseOrderIncludes...),
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

func (m *purchaseOrderSvcImpl) CreatePurchaseOrder(ctx context.Context, req *CreatePurchaseOrderRequest) (*apiresource.PurchaseOrder, *apierror.APIError) {
	lines := make([]*pb.CreatePurchaseOrderLineInput, len(req.Lines))
	for i, l := range req.Lines {
		lines[i] = &pb.CreatePurchaseOrderLineInput{
			ProductId:                  l.ProductID,
			ItemId:                     l.ItemID.Ptr(),
			ProductSku:                 l.ProductSKU,
			ProductDescription:         l.ProductDescription.Ptr(),
			QuantityValue:              l.QuantityValue,
			QuantityUnitId:             l.QuantityUnitID,
			UnitPriceValue:             l.UnitPriceValue,
			UnitPriceNumeratorUnitId:   l.UnitPriceNumeratorUnitID,
			UnitPriceDenominatorUnitId: l.UnitPriceDenominatorUnitID,
			UnitCostValue:              l.UnitCostValue.Ptr(),
			UnitCostNumeratorUnitId:    l.UnitCostNumeratorUnitID.Ptr(),
			UnitCostDenominatorUnitId:  l.UnitCostDenominatorUnitID.Ptr(),
		}
	}

	pbReq := &pb.CreatePurchaseOrderRequest{
		SupplierAccountId:     req.SupplierAccountID,
		Note:                  req.Note.Ptr(),
		CarrierId:             req.CarrierID.Ptr(),
		ServiceLevelId:        req.ServiceLevelID.Ptr(),
		CarrierBillingType:    req.CarrierBillingType.Ptr(),
		CarrierBillingAccount: req.CarrierBillingAccount.Ptr(),
		PriorityCode:          req.PriorityCode,
		ShippingTermId:        req.ShippingTermID.Ptr(),
		PaymentTermId:         req.PaymentTermID.Ptr(),
		BillToName:            req.BillToName.Ptr(),
		BillToStreetLine_1:    req.BillToStreetLine1.Ptr(),
		BillToStreetLine_2:    req.BillToStreetLine2.Ptr(),
		BillToLocality:        req.BillToLocality.Ptr(),
		BillToState:           req.BillToState.Ptr(),
		BillToPostalCode:      req.BillToPostalCode.Ptr(),
		BillToCountry:         req.BillToCountry.Ptr(),
		ShipToName:            req.ShipToName.Ptr(),
		ShipToStreetLine_1:    req.ShipToStreetLine1.Ptr(),
		ShipToStreetLine_2:    req.ShipToStreetLine2.Ptr(),
		ShipToLocality:        req.ShipToLocality.Ptr(),
		ShipToState:           req.ShipToState.Ptr(),
		ShipToPostalCode:      req.ShipToPostalCode.Ptr(),
		ShipToCountry:         req.ShipToCountry.Ptr(),
		Lines:                 lines,
		ContactAccountUserIds: req.ContactAccountUserIDs,
		PromisedAt:            req.PromisedAt.Ptr(),
		Includes:              resourcekit.FilterIncludes(ctx, purchaseOrderIncludes...),
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

func (m *purchaseOrderSvcImpl) UpdatePurchaseOrder(ctx context.Context, req *UpdatePurchaseOrderRequest) (*apiresource.PurchaseOrder, *apierror.APIError) {
	contactAccountUserIDs, _ := req.ContactAccountUserIDs.Value()
	pbReq := &pb.UpdatePurchaseOrderRequest{
		Id:                    req.PurchaseOrderID,
		Note:                  req.Note.Ptr(),
		Number:                req.Number.Ptr(),
		PriorityCode:          req.PriorityCode.Ptr(),
		BillingAddressId:      req.BillingAddressID.Ptr(),
		ShippingAddressId:     req.ShippingAddressID.Ptr(),
		PromisedAt:            req.PromisedAt.Ptr(),
		ContactAccountUserIds: contactAccountUserIDs,
		Includes:              resourcekit.FilterIncludes(ctx, purchaseOrderIncludes...),
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

func (m *purchaseOrderSvcImpl) ChangePurchaseOrderStatus(ctx context.Context, req *ChangePurchaseOrderStatusRequest) (*apiresource.PurchaseOrder, *apierror.APIError) {
	pbReq := &pb.ChangePurchaseOrderStatusRequest{
		Id:           req.PurchaseOrderID,
		StatusChange: req.StatusChange,
		SendEmail:    req.SendEmail,
		Includes:     resourcekit.FilterIncludes(ctx, purchaseOrderIncludes...),
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

func (m *purchaseOrderSvcImpl) CreatePurchaseOrderLine(ctx context.Context, req *CreatePurchaseOrderLineRequest) (*apiresource.PurchaseOrderLine, *apierror.APIError) {
	pbReq := &pb.CreatePurchaseOrderLineRequest{
		PurchaseOrderId:            req.PurchaseOrderID,
		ProductId:                  req.ProductID,
		ItemId:                     req.ItemID.Ptr(),
		ProductSku:                 req.ProductSKU,
		ProductDescription:         req.ProductDescription.Ptr(),
		QuantityValue:              req.QuantityValue,
		QuantityUnitId:             req.QuantityUnitID,
		UnitPriceValue:             req.UnitPriceValue,
		UnitPriceNumeratorUnitId:   req.UnitPriceNumeratorUnitID,
		UnitPriceDenominatorUnitId: req.UnitPriceDenominatorUnitID,
		UnitCostValue:              req.UnitCostValue.Ptr(),
		UnitCostNumeratorUnitId:    req.UnitCostNumeratorUnitID.Ptr(),
		UnitCostDenominatorUnitId:  req.UnitCostDenominatorUnitID.Ptr(),
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

func (m *purchaseOrderSvcImpl) UpdatePurchaseOrderLine(ctx context.Context, req *UpdatePurchaseOrderLineRequest) (*apiresource.PurchaseOrderLine, *apierror.APIError) {
	pbReq := &pb.UpdatePurchaseOrderLineRequest{
		PurchaseOrderId:            req.PurchaseOrderID,
		Id:                         req.PurchaseOrderLineID,
		ProductId:                  req.ProductID.Ptr(),
		ItemId:                     req.ItemID.Ptr(),
		ProductSku:                 req.ProductSKU.Ptr(),
		ProductDescription:         req.ProductDescription.Ptr(),
		QuantityValue:              req.QuantityValue.Ptr(),
		QuantityUnitId:             req.QuantityUnitID.Ptr(),
		UnitPriceValue:             req.UnitPriceValue.Ptr(),
		UnitPriceNumeratorUnitId:   req.UnitPriceNumeratorUnitID.Ptr(),
		UnitPriceDenominatorUnitId: req.UnitPriceDenominatorUnitID.Ptr(),
		UnitCostValue:              req.UnitCostValue.Ptr(),
		UnitCostNumeratorUnitId:    req.UnitCostNumeratorUnitID.Ptr(),
		UnitCostDenominatorUnitId:  req.UnitCostDenominatorUnitID.Ptr(),
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

// purchaseOrderSummaryFromProto maps a list-view PurchaseOrderSummaryInfo to
// PurchaseOrder. Expandable sub-resources (supplier, addresses, freight, etc.)
// are left nil since they are populated via the include resolver from stashed
// meta.
func purchaseOrderSummaryFromProto(info *pb.PurchaseOrderSummaryInfo) apiresource.PurchaseOrder {
	s := apiresource.PurchaseOrder{
		ID:                   info.Id,
		Object:               constants.ObjectTypePurchaseOrder,
		Number:               info.Number,
		Status:               constants.SalesOrderStatusCode(info.StatusCode),
		Priority:             constants.PriorityCode(info.PriorityCode),
		AcknowledgmentStatus: acknowledgmentStatusFromBool(info.IsAcknowledgmentSent),
		LineCount:            info.LineCount,
		CreatedAt:            grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt:            grpcutil.TimestampToTime(info.UpdatedAt),
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

func purchaseOrderDetailFromProto(info *pb.PurchaseOrderInfo) apiresource.PurchaseOrder {
	d := apiresource.PurchaseOrder{
		ID:                   info.Id,
		Object:               constants.ObjectTypePurchaseOrder,
		Number:               info.Number,
		Note:                 info.Note,
		Status:               constants.SalesOrderStatusCode(info.StatusCode),
		Priority:             constants.PriorityCode(info.PriorityCode),
		AcknowledgmentStatus: acknowledgmentStatusFromBool(info.IsAcknowledgmentSent),
		LineCount:            safeconv.IntToInt32(len(info.Lines)),
		CreatedAt:            grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt:            grpcutil.TimestampToTime(info.UpdatedAt),
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

// acknowledgmentStatusFromBool maps the legacy boolean acknowledgment flag to
// the AcknowledgmentStatus enum.
func acknowledgmentStatusFromBool(sent bool) constants.AcknowledgmentStatus {
	if sent {
		return constants.AcknowledgmentStatusSent
	}
	return constants.AcknowledgmentStatusNotSent
}

// stashPurchaseOrderSummaryMeta stashes the FK ids exposed by a list-view
// summary so the include resolver can fetch the real expandable resources.
func stashPurchaseOrderSummaryMeta(ctx context.Context, info *pb.PurchaseOrderSummaryInfo, d *apiresource.PurchaseOrder) {
	if info == nil {
		return
	}

	if info.SupplierId != "" {
		resourcekit.GetLoadMeta(ctx).Set(constants.ObjectTypePurchaseOrder, d.ID, "supplier", &apiresource.Supplier{
			ID:     info.SupplierId,
			Object: constants.ObjectTypeSupplier,
			Name:   info.SupplierName,
			Number: info.SupplierNumber,
		})
	}

	// Lines are populated on the summary only when the list request includes them.
	if len(info.Lines) > 0 {
		lines := make([]apiresource.PurchaseOrderLine, len(info.Lines))
		for i, l := range info.Lines {
			lines[i] = purchaseOrderLineDetailFromProto(l)
		}
		resourcekit.GetLoadMeta(ctx).Set(constants.ObjectTypePurchaseOrder, d.ID, "lines",
			apiresource.NewList(lines, apiresource.PageInfo{}))
	}
}

func stashPurchaseOrderDetailMeta(ctx context.Context, info *pb.PurchaseOrderInfo, d *apiresource.PurchaseOrder) {
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

	// Freight (carrier selection + freight billing). Expanded as a whole via
	// include[]=freight; carries the full carrier and service level inline.
	freight := &apiresource.Freight{Object: constants.ObjectTypeFreight}
	if info.CarrierBillingType != nil {
		bt := constants.CarrierBillingType(*info.CarrierBillingType)
		freight.BillingType = &bt
	}
	freight.BillingAccountNumber = info.CarrierBillingAccount
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
		freight.Carrier = carrier
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
		freight.ServiceLevel = sl
	}
	meta.Set(constants.ObjectTypePurchaseOrder, d.ID, "freight", freight)

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

	// receiving_order is a document-level cross-reference: stash only the FK id
	// so the loader fetches the real ReceivingOrder on ?include=receiving_order.
	// Never fabricate an inline stub.
	if info.ReceivingOrderId != nil {
		meta.Set(constants.ObjectTypePurchaseOrder, d.ID, "receiving_order_id", *info.ReceivingOrderId)
	}

	lines := make([]apiresource.PurchaseOrderLine, len(info.Lines))
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

func purchaseOrderLineDetailFromProto(info *pb.PurchaseOrderLineInfo) apiresource.PurchaseOrderLine {
	l := apiresource.PurchaseOrderLine{
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
