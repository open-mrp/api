package salesorderep

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

// toSalesOrderEmailContactInputs converts the endpoint input slice to proto messages.
func toSalesOrderEmailContactInputs(inputs []SalesOrderEmailContactInput) []*pb.SalesOrderEmailContactInput {
	if len(inputs) == 0 {
		return nil
	}
	out := make([]*pb.SalesOrderEmailContactInput, len(inputs))
	for i, c := range inputs {
		out[i] = &pb.SalesOrderEmailContactInput{AccountUserId: c.AccountUserID}
	}
	return out
}

// toSalesOrderEmailContactList wraps an optional contact list for update requests.
// nil → leave existing contacts untouched; non-nil (even empty) → replace existing contacts.
func toSalesOrderEmailContactList(inputs *[]SalesOrderEmailContactInput) *pb.SalesOrderEmailContactList {
	if inputs == nil {
		return nil
	}
	return &pb.SalesOrderEmailContactList{Contacts: toSalesOrderEmailContactInputs(*inputs)}
}

type SalesOrderSvc interface {
	ListSalesOrders(ctx context.Context, req *ListSalesOrdersRequest) (*apiresource.List[apiresource.SalesOrderSummary], *apierror.APIError)
	GetSalesOrder(ctx context.Context, req *RetrieveSalesOrderRequest) (*apiresource.SalesOrderDetail, *apierror.APIError)
	CreateSalesOrder(ctx context.Context, req *CreateSalesOrderRequest) (*apiresource.SalesOrderDetail, *apierror.APIError)
	UpdateSalesOrder(ctx context.Context, req *UpdateSalesOrderRequest) (*apiresource.SalesOrderDetail, *apierror.APIError)
	DeleteSalesOrder(ctx context.Context, req *DeleteSalesOrderRequest) (*apiresource.EmptyResource, *apierror.APIError)
	BulkDeleteSalesOrders(ctx context.Context, req *BulkDeleteSalesOrdersRequest) (*apiresource.EmptyResource, *apierror.APIError)
	ChangeSalesOrderStatus(ctx context.Context, req *ChangeSalesOrderStatusRequest) (*apiresource.SalesOrderDetail, *apierror.APIError)
	CheckoutSalesOrder(ctx context.Context, req *CheckoutSalesOrderRequest) (*CheckoutSalesOrderResponse, *apierror.APIError)
	CreateSalesOrderProductionRun(ctx context.Context, req *CreateProductionRunRequest) (*CreateProductionRunResponse, *apierror.APIError)
	CreateSalesOrderLine(ctx context.Context, req *CreateSalesOrderLineRequest) (*apiresource.SalesOrderLineDetail, *apierror.APIError)
	UpdateSalesOrderLine(ctx context.Context, req *UpdateSalesOrderLineRequest) (*apiresource.SalesOrderLineDetail, *apierror.APIError)
	DeleteSalesOrderLine(ctx context.Context, req *DeleteSalesOrderLineRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type SalesOrderSvcConfig struct {
	CoreClient pb.CoreSalesServiceClient
}

type salesOrderSvcImpl struct {
	coreClient pb.CoreSalesServiceClient
}

var salesOrderEpSvcTracer = tracing.GetTracer("api-gateway.endpoints.sales-orders.service")

func (c *SalesOrderSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("sales order endpoint service: core client is required")
	}
	return nil
}

func NewSalesOrderSvc(config *SalesOrderSvcConfig) SalesOrderSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &salesOrderSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *salesOrderSvcImpl) ListSalesOrders(ctx context.Context, req *ListSalesOrdersRequest) (*apiresource.List[apiresource.SalesOrderSummary], *apierror.APIError) {
	pbReq := &pb.ListSalesOrdersRequest{
		Cursor:                req.Cursor,
		Limit:                 req.Limit,
		Query:                 req.Query,
		StatusCodes:           req.StatusCodes,
		ItemIds:               req.ItemIDs,
		ProductLineIds:        req.ProductLineIDs,
		CustomerIds:           req.CustomerIDs,
		CustomerGroupIds:      req.CustomerGroupIDs,
		SalesRepIds:           req.SalesRepIDs,
		StartDate:             req.StartDate,
		EndDate:               req.EndDate,
		ExcludeInternalOrders: req.ExcludeInternalOrders,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, salesOrderEpSvcTracer, "service.sales_orders.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListSalesOrdersResponse, error) {
			return m.coreClient.ListSalesOrders(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return SalesOrderListPresenter(resp), nil
}

func (m *salesOrderSvcImpl) GetSalesOrder(ctx context.Context, req *RetrieveSalesOrderRequest) (*apiresource.SalesOrderDetail, *apierror.APIError) {
	pbReq := &pb.GetSalesOrderRequest{
		Id:       req.SalesOrderID,
		Includes: req.Includes,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, salesOrderEpSvcTracer, "service.sales_orders.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetSalesOrderResponse, error) {
			return m.coreClient.GetSalesOrder(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := SalesOrderDetailPresenter(resp.SalesOrder)
	return &result, nil
}

func (m *salesOrderSvcImpl) CreateSalesOrder(ctx context.Context, req *CreateSalesOrderRequest) (*apiresource.SalesOrderDetail, *apierror.APIError) {
	lines := make([]*pb.CreateSalesOrderLineInput, len(req.Lines))
	for i, l := range req.Lines {
		lines[i] = &pb.CreateSalesOrderLineInput{
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
			EdiLineItemId:              l.EdiLineItemID,
		}
	}

	pbReq := &pb.CreateSalesOrderRequest{
		BuyerAccountId:               req.BuyerAccountID,
		CustomerPoNumber:             req.CustomerPONumber,
		Note:                         req.Note,
		CarrierId:                    req.CarrierID,
		ServiceLevelId:               req.ServiceLevelID,
		CarrierBillingType:           req.CarrierBillingType,
		CarrierBillingAccount:        req.CarrierBillingAccount,
		PriorityCode:                 req.PriorityCode,
		SalesRepId:                   req.SalesRepID,
		ShippingTermId:               req.ShippingTermID,
		SalesOrderTypeCode:           req.SalesOrderTypeCode,
		PaymentTermId:                req.PaymentTermID,
		OrderDiscountId:              req.OrderDiscountID,
		BillToName:                   req.BillToName,
		BillToStreetLine_1:           req.BillToStreetLine1,
		BillToStreetLine_2:           req.BillToStreetLine2,
		BillToLocality:               req.BillToLocality,
		BillToState:                  req.BillToState,
		BillToPostalCode:             req.BillToPostalCode,
		BillToCountry:                req.BillToCountry,
		ShipToName:                   req.ShipToName,
		ShipToStreetLine_1:           req.ShipToStreetLine1,
		ShipToStreetLine_2:           req.ShipToStreetLine2,
		ShipToLocality:               req.ShipToLocality,
		ShipToState:                  req.ShipToState,
		ShipToPostalCode:             req.ShipToPostalCode,
		ShipToCountry:                req.ShipToCountry,
		Lines:                        lines,
		AcknowledgementEmailContacts: toSalesOrderEmailContactInputs(req.AcknowledgementEmailContacts),
		InvoiceEmailContacts:         toSalesOrderEmailContactInputs(req.InvoiceEmailContacts),
		Includes:                     appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, salesOrderEpSvcTracer, "service.sales_orders.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateSalesOrderResponse, error) {
			return m.coreClient.CreateSalesOrder(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := SalesOrderDetailPresenter(resp.SalesOrder)
	return &result, nil
}

func (m *salesOrderSvcImpl) UpdateSalesOrder(ctx context.Context, req *UpdateSalesOrderRequest) (*apiresource.SalesOrderDetail, *apierror.APIError) {
	pbReq := &pb.UpdateSalesOrderRequest{
		Id:                           req.SalesOrderID,
		CustomerPoNumber:             req.CustomerPONumber,
		Note:                         req.Note,
		CarrierId:                    req.CarrierID,
		ServiceLevelId:               req.ServiceLevelID,
		CarrierBillingType:           req.CarrierBillingType,
		CarrierBillingAccount:        req.CarrierBillingAccount,
		PriorityCode:                 req.PriorityCode,
		SalesRepId:                   req.SalesRepID,
		ShippingTermId:               req.ShippingTermID,
		PaymentTermId:                req.PaymentTermID,
		OrderDiscountId:              req.OrderDiscountID,
		BillToName:                   req.BillToName,
		BillToStreetLine_1:           req.BillToStreetLine1,
		BillToStreetLine_2:           req.BillToStreetLine2,
		BillToLocality:               req.BillToLocality,
		BillToState:                  req.BillToState,
		BillToPostalCode:             req.BillToPostalCode,
		BillToCountry:                req.BillToCountry,
		ShipToName:                   req.ShipToName,
		ShipToStreetLine_1:           req.ShipToStreetLine1,
		ShipToStreetLine_2:           req.ShipToStreetLine2,
		ShipToLocality:               req.ShipToLocality,
		ShipToState:                  req.ShipToState,
		ShipToPostalCode:             req.ShipToPostalCode,
		ShipToCountry:                req.ShipToCountry,
		AcknowledgementEmailContacts: toSalesOrderEmailContactList(req.AcknowledgementEmailContacts),
		InvoiceEmailContacts:         toSalesOrderEmailContactList(req.InvoiceEmailContacts),
		Includes:                     appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, salesOrderEpSvcTracer, "service.sales_orders.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateSalesOrderResponse, error) {
			return m.coreClient.UpdateSalesOrder(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := SalesOrderDetailPresenter(resp.SalesOrder)
	return &result, nil
}

func (m *salesOrderSvcImpl) DeleteSalesOrder(ctx context.Context, req *DeleteSalesOrderRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteSalesOrderRequest{Id: req.SalesOrderID}

	_, apiErr := grpcutil.CallRPC(ctx, salesOrderEpSvcTracer, "service.sales_orders.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteSalesOrder(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func (m *salesOrderSvcImpl) BulkDeleteSalesOrders(ctx context.Context, req *BulkDeleteSalesOrdersRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.BulkDeleteSalesOrdersRequest{Ids: req.SalesOrderIDs}

	_, apiErr := grpcutil.CallRPC(ctx, salesOrderEpSvcTracer, "service.sales_orders.bulk_delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.BulkDeleteSalesOrders(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func (m *salesOrderSvcImpl) ChangeSalesOrderStatus(ctx context.Context, req *ChangeSalesOrderStatusRequest) (*apiresource.SalesOrderDetail, *apierror.APIError) {
	pbReq := &pb.ChangeSalesOrderStatusRequest{
		Id:           req.SalesOrderID,
		StatusChange: req.StatusChange,
		SendEmail:    req.SendEmail,
		Includes:     appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, salesOrderEpSvcTracer, "service.sales_orders.change_status", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ChangeSalesOrderStatusResponse, error) {
			return m.coreClient.ChangeSalesOrderStatus(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := SalesOrderDetailPresenter(resp.SalesOrder)
	return &result, nil
}

func (m *salesOrderSvcImpl) CheckoutSalesOrder(ctx context.Context, req *CheckoutSalesOrderRequest) (*CheckoutSalesOrderResponse, *apierror.APIError) {
	pbReq := &pb.CheckoutSalesOrderRequest{
		Id:         req.SalesOrderID,
		Email:      req.Email,
		SuccessUrl: req.SuccessURL,
		CancelUrl:  req.CancelURL,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, salesOrderEpSvcTracer, "service.sales_orders.checkout", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CheckoutSalesOrderResponse, error) {
			return m.coreClient.CheckoutSalesOrder(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return &CheckoutSalesOrderResponse{CheckoutURL: resp.CheckoutUrl}, nil
}

func (m *salesOrderSvcImpl) CreateSalesOrderProductionRun(ctx context.Context, req *CreateProductionRunRequest) (*CreateProductionRunResponse, *apierror.APIError) {
	pbReq := &pb.CreateSalesOrderProductionRunRequest{Id: req.SalesOrderID}

	resp, apiErr := grpcutil.CallRPC(ctx, salesOrderEpSvcTracer, "service.sales_orders.create_production_run", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateSalesOrderProductionRunResponse, error) {
			return m.coreClient.CreateSalesOrderProductionRun(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return &CreateProductionRunResponse{
		ProductionRun: CreateProductionRunResponseRef{ID: resp.ProductionRunId},
	}, nil
}

func (m *salesOrderSvcImpl) CreateSalesOrderLine(ctx context.Context, req *CreateSalesOrderLineRequest) (*apiresource.SalesOrderLineDetail, *apierror.APIError) {
	pbReq := &pb.CreateSalesOrderLineRequest{
		SalesOrderId:               req.SalesOrderID,
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
		EdiLineItemId:              req.EdiLineItemID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, salesOrderEpSvcTracer, "service.sales_orders.create_line", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateSalesOrderLineResponse, error) {
			return m.coreClient.CreateSalesOrderLine(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := SalesOrderLineDetailPresenter(resp.SalesOrderLine)
	return &result, nil
}

func (m *salesOrderSvcImpl) UpdateSalesOrderLine(ctx context.Context, req *UpdateSalesOrderLineRequest) (*apiresource.SalesOrderLineDetail, *apierror.APIError) {
	pbReq := &pb.UpdateSalesOrderLineRequest{
		SalesOrderId:               req.SalesOrderID,
		Id:                         req.SalesOrderLineID,
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
		EdiLineItemId:              req.EdiLineItemID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, salesOrderEpSvcTracer, "service.sales_orders.update_line", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateSalesOrderLineResponse, error) {
			return m.coreClient.UpdateSalesOrderLine(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := SalesOrderLineDetailPresenter(resp.SalesOrderLine)
	return &result, nil
}

func (m *salesOrderSvcImpl) DeleteSalesOrderLine(ctx context.Context, req *DeleteSalesOrderLineRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteSalesOrderLineRequest{
		SalesOrderId: req.SalesOrderID,
		Id:           req.SalesOrderLineID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, salesOrderEpSvcTracer, "service.sales_orders.delete_line", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteSalesOrderLine(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}
