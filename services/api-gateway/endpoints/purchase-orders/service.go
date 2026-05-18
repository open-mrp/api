package purchaseorderep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/appctx"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
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

	return PurchaseOrderListPresenter(ctx, resp), nil
}

func (m *purchaseOrderSvcImpl) GetPurchaseOrder(ctx context.Context, req *RetrievePurchaseOrderRequest) (*apiresource.PurchaseOrderDetail, *apierror.APIError) {
	pbReq := &pb.GetPurchaseOrderRequest{
		Id:       req.PurchaseOrderID,
		Includes: req.Includes,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, purchaseOrderEpSvcTracer, "service.purchase_orders.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetPurchaseOrderResponse, error) {
			return m.coreClient.GetPurchaseOrder(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := PurchaseOrderDetailPresenter(resp.PurchaseOrder)
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
		Includes:              appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, purchaseOrderEpSvcTracer, "service.purchase_orders.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreatePurchaseOrderResponse, error) {
			return m.coreClient.CreatePurchaseOrder(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := PurchaseOrderDetailPresenter(resp.PurchaseOrder)
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
		Includes:              appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, purchaseOrderEpSvcTracer, "service.purchase_orders.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdatePurchaseOrderResponse, error) {
			return m.coreClient.UpdatePurchaseOrder(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := PurchaseOrderDetailPresenter(resp.PurchaseOrder)
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
		Includes:     appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, purchaseOrderEpSvcTracer, "service.purchase_orders.change_status", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ChangePurchaseOrderStatusResponse, error) {
			return m.coreClient.ChangePurchaseOrderStatus(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := PurchaseOrderDetailPresenter(resp.PurchaseOrder)
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

	result := PurchaseOrderLineDetailPresenter(resp.PurchaseOrderLine)
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

	result := PurchaseOrderLineDetailPresenter(resp.PurchaseOrderLine)
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

	return PurchaseOrderStatusListPresenter(ctx, resp), nil
}
