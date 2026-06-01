package salesorderep

import (
	"context"
	"fmt"
	"time"

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
	ListSalesOrders(ctx context.Context, req *ListSalesOrdersRequest) (*apiresource.List[apiresource.SalesOrderDetail], *apierror.APIError)
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

var salesOrderIncludes = []string{"customer", "bill_to_address", "ship_to_address", "carrier", "service_level", "payment_term", "shipping_term", "order_discount", "lines", "lines.item"}

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

func (m *salesOrderSvcImpl) ListSalesOrders(ctx context.Context, req *ListSalesOrdersRequest) (*apiresource.List[apiresource.SalesOrderDetail], *apierror.APIError) {
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

	orders := make([]apiresource.SalesOrderDetail, len(resp.SalesOrders))
	for i, o := range resp.SalesOrders {
		orders[i] = salesOrderSummaryToDetail(o)
		stashSalesOrderSummaryMeta(ctx, o, &orders[i])
	}

	return apiresource.NewList(orders, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *salesOrderSvcImpl) GetSalesOrder(ctx context.Context, req *RetrieveSalesOrderRequest) (*apiresource.SalesOrderDetail, *apierror.APIError) {
	pbReq := &pb.GetSalesOrderRequest{
		Id:       req.SalesOrderID,
		Includes: salesOrderIncludes,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, salesOrderEpSvcTracer, "service.sales_orders.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetSalesOrderResponse, error) {
			return m.coreClient.GetSalesOrder(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := salesOrderDetailFromProto(resp.SalesOrder)
	stashSalesOrderDetailMeta(ctx, resp.SalesOrder, &result)
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
		Includes:                     salesOrderIncludes,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, salesOrderEpSvcTracer, "service.sales_orders.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateSalesOrderResponse, error) {
			return m.coreClient.CreateSalesOrder(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := salesOrderDetailFromProto(resp.SalesOrder)
	stashSalesOrderDetailMeta(ctx, resp.SalesOrder, &result)
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
		Includes:                     salesOrderIncludes,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, salesOrderEpSvcTracer, "service.sales_orders.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateSalesOrderResponse, error) {
			return m.coreClient.UpdateSalesOrder(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := salesOrderDetailFromProto(resp.SalesOrder)
	stashSalesOrderDetailMeta(ctx, resp.SalesOrder, &result)
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
		Includes:     salesOrderIncludes,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, salesOrderEpSvcTracer, "service.sales_orders.change_status", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ChangeSalesOrderStatusResponse, error) {
			return m.coreClient.ChangeSalesOrderStatus(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := salesOrderDetailFromProto(resp.SalesOrder)
	stashSalesOrderDetailMeta(ctx, resp.SalesOrder, &result)
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
		ProductionRun: CreateProductionRunResponseRef{
			ID:     resp.ProductionRunId,
			Object: constants.ObjectTypeProductionRun,
		},
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

	result := salesOrderLineDetailFromProto(resp.SalesOrderLine)
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

	result := salesOrderLineDetailFromProto(resp.SalesOrderLine)
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

func salesOrderSummaryFromProto(info *pb.SalesOrderSummaryInfo) apiresource.SalesOrderSummary {
	customer := &apiresource.Customer{
		ID:               info.CustomerId,
		Object:           constants.ObjectTypeCustomer,
		Name:             info.CustomerName,
		Number:           info.CustomerNumber,
		EDIStatus:        constants.EDIStatusDisabled,
		RelationshipType: constants.CustomerRelationshipTypeStandalone,
	}
	if info.CustomerStatusCode != nil {
		customer.Status = constants.AccountStatusCode(*info.CustomerStatusCode)
	}
	if info.CustomerCommissionPolicy != nil {
		customer.CommissionPolicy = constants.CommissionPolicy(*info.CustomerCommissionPolicy)
	}

	s := apiresource.SalesOrderSummary{
		ID:       info.Id,
		Object:   constants.ObjectTypeSalesOrder,
		Number:   info.Number,
		Customer: customer,
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
		CustomerPO:           info.CustomerPoNumber,
		CreatedAt:            grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt:            grpcutil.TimestampToTime(info.UpdatedAt),
	}

	finalizeCustomerStubForInclude(customer, s.CreatedAt, s.UpdatedAt)

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

func salesOrderDetailFromProto(info *pb.SalesOrderInfo) apiresource.SalesOrderDetail {
	d := apiresource.SalesOrderDetail{
		ID:                    info.Id,
		Object:                constants.ObjectTypeSalesOrder,
		Number:                info.Number,
		CustomerPO:            info.CustomerPoNumber,
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

	// Sales rep (as Actor sub-resource) — always inline, not expandable
	if info.SalesRepId != nil {
		d.SalesRep = apiresource.NewActor(
			*info.SalesRepId,
			constants.ActorTypeUser,
			info.SalesRepName,
			nil,
		)
	}

	// Production run — always inline, not expandable
	if info.ProductionRunId != nil {
		d.ProductionRun = &apiresource.ProductionRun{
			ID:     *info.ProductionRunId,
			Object: constants.ObjectTypeProductionRun,
		}
	}

	// Pick — always inline, not expandable
	if info.PickId != nil {
		d.Pick = &apiresource.Pick{
			ID:     *info.PickId,
			Object: constants.ObjectTypePick,
		}
	}

	// Timestamps
	if info.IssuedAt != nil {
		t := grpcutil.TimestampToTime(info.IssuedAt)
		d.IssuedAt = &t
	}
	if info.CompletedAt != nil {
		t := grpcutil.TimestampToTime(info.CompletedAt)
		d.CompletedAt = &t
	}
	if info.FirstShipAt != nil {
		t := grpcutil.TimestampToTime(info.FirstShipAt)
		d.FirstShipAt = &t
	}
	if info.ExpiredAt != nil {
		t := grpcutil.TimestampToTime(info.ExpiredAt)
		d.ExpiredAt = &t
	}
	if info.PromisedAt != nil {
		t := grpcutil.TimestampToTime(info.PromisedAt)
		d.PromisedAt = &t
	}

	return d
}

func stashSalesOrderDetailMeta(ctx context.Context, info *pb.SalesOrderInfo, d *apiresource.SalesOrderDetail) {
	if info == nil {
		return
	}

	meta := resourcekit.GetLoadMeta(ctx)

	// Customer
	customer := &apiresource.Customer{
		ID:               info.CustomerId,
		Object:           constants.ObjectTypeCustomer,
		Name:             info.CustomerName,
		Number:           info.CustomerNumber,
		EDIStatus:        constants.EDIStatusDisabled,
		RelationshipType: constants.CustomerRelationshipTypeStandalone,
	}
	if info.CustomerStatusCode != nil {
		customer.Status = constants.AccountStatusCode(*info.CustomerStatusCode)
	}
	if info.CustomerCommissionPolicy != nil {
		customer.CommissionPolicy = constants.CommissionPolicy(*info.CustomerCommissionPolicy)
	}
	if info.CustomerCreatedAt != nil {
		customer.CreatedAt = info.CustomerCreatedAt.AsTime()
	}
	if info.CustomerUpdatedAt != nil {
		customer.UpdatedAt = info.CustomerUpdatedAt.AsTime()
	}
	finalizeCustomerStubForInclude(customer, d.CreatedAt, d.UpdatedAt)
	meta.Set(constants.ObjectTypeSalesOrder, d.ID, "customer", customer)

	// Bill-to address
	if info.BillingAddressId != "" {
		meta.Set(constants.ObjectTypeSalesOrder, d.ID, "bill_to_address",
			buildSOAddressFromProto(
				info.BillingAddressId, info.BillToName, info.BillToStreetLine_1, info.BillToStreetLine_2,
				info.BillToLocality, info.BillToState, info.BillToPostalCode, info.BillToCountry,
				info.BillToPhone, info.BillToEmail,
				info.BillToIsDropShip, info.BillToGeolocationId,
				grpcutil.TimestampToTime(info.BillToCreatedAt), grpcutil.TimestampToTime(info.BillToUpdatedAt),
			))
	}

	// Ship-to address
	if info.ShippingAddressId != "" {
		meta.Set(constants.ObjectTypeSalesOrder, d.ID, "ship_to_address",
			buildSOAddressFromProto(
				info.ShippingAddressId, info.ShipToName, info.ShipToStreetLine_1, info.ShipToStreetLine_2,
				info.ShipToLocality, info.ShipToState, info.ShipToPostalCode, info.ShipToCountry,
				info.ShipToPhone, info.ShipToEmail,
				info.ShipToIsDropShip, info.ShipToGeolocationId,
				grpcutil.TimestampToTime(info.ShipToCreatedAt), grpcutil.TimestampToTime(info.ShipToUpdatedAt),
			))
	}

	// Carrier
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
		meta.Set(constants.ObjectTypeSalesOrder, d.ID, "carrier", carrier)
	}

	// Service level
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
		meta.Set(constants.ObjectTypeSalesOrder, d.ID, "service_level", sl)
	}

	// Payment term
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
		meta.Set(constants.ObjectTypeSalesOrder, d.ID, "payment_term", pt)
	}

	// Shipping term
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
		meta.Set(constants.ObjectTypeSalesOrder, d.ID, "shipping_term", st)
	}

	// Order discount
	if info.OrderDiscountId != nil {
		od := &apiresource.OrderDiscount{
			ID:     *info.OrderDiscountId,
			Object: constants.ObjectTypeOrderDiscount,
		}
		if info.OrderDiscountName != nil {
			od.Name = *info.OrderDiscountName
		}
		if info.OrderDiscountCode != nil {
			od.Code = *info.OrderDiscountCode
		}
		if info.OrderDiscountPercentage != nil {
			od.Percentage = *info.OrderDiscountPercentage
		}
		if info.OrderDiscountAmount != nil {
			od.Amount = *info.OrderDiscountAmount
		}
		if info.OrderDiscountDiscountType != nil {
			od.DiscountType = constants.OrderDiscountType(*info.OrderDiscountDiscountType)
		}
		if info.OrderDiscountOrderCount != nil {
			od.OrderCount = *info.OrderDiscountOrderCount
		}
		if info.OrderDiscountCreatedAt != nil {
			od.CreatedAt = info.OrderDiscountCreatedAt.AsTime()
		}
		if info.OrderDiscountUpdatedAt != nil {
			od.UpdatedAt = info.OrderDiscountUpdatedAt.AsTime()
		}
		meta.Set(constants.ObjectTypeSalesOrder, d.ID, "order_discount", od)
	}

	// Lines
	lines := make([]apiresource.SalesOrderLineDetail, len(info.Lines))
	for i, l := range info.Lines {
		lines[i] = salesOrderLineDetailFromProto(l)
		stashSalesOrderLineMeta(meta, l, &lines[i])
	}
	meta.Set(constants.ObjectTypeSalesOrder, d.ID, "lines", apiresource.NewList(lines, apiresource.PageInfo{}))
}

// salesOrderSummaryToDetail maps a list-view SalesOrderSummaryInfo to SalesOrderDetail.
// Expandable sub-resources (customer, addresses, carrier, service level, etc.) are left nil
// since they are populated via the V2 include resolver from stashed meta.
func salesOrderSummaryToDetail(info *pb.SalesOrderSummaryInfo) apiresource.SalesOrderDetail {
	d := apiresource.SalesOrderDetail{
		ID:                   info.Id,
		Object:               constants.ObjectTypeSalesOrder,
		Number:               info.Number,
		CustomerPO:           info.CustomerPoNumber,
		IsAcknowledgmentSent: info.IsAcknowledgmentSent,
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
		LineCount: info.LineCount,
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

	return d
}

func stashSalesOrderSummaryMeta(ctx context.Context, info *pb.SalesOrderSummaryInfo, d *apiresource.SalesOrderDetail) {
	if info == nil {
		return
	}

	customer := &apiresource.Customer{
		ID:               info.CustomerId,
		Object:           constants.ObjectTypeCustomer,
		Name:             info.CustomerName,
		Number:           info.CustomerNumber,
		EDIStatus:        constants.EDIStatusDisabled,
		RelationshipType: constants.CustomerRelationshipTypeStandalone,
	}
	if info.CustomerStatusCode != nil {
		customer.Status = constants.AccountStatusCode(*info.CustomerStatusCode)
	}
	if info.CustomerCommissionPolicy != nil {
		customer.CommissionPolicy = constants.CommissionPolicy(*info.CustomerCommissionPolicy)
	}
	finalizeCustomerStubForInclude(customer, d.CreatedAt, d.UpdatedAt)

	resourcekit.GetLoadMeta(ctx).Set(constants.ObjectTypeSalesOrder, d.ID, "customer", customer)
}

func salesOrderLineDetailFromProto(info *pb.SalesOrderLineInfo) apiresource.SalesOrderLineDetail {
	l := apiresource.SalesOrderLineDetail{
		ID:                 info.Id,
		Object:             constants.ObjectTypeSalesOrderLine,
		LineItemNumber:     info.LineItemNumber,
		ProductSKU:         info.ProductSku,
		ProductDescription: info.ProductDescription,
		EdiLineItemID:      info.EdiLineItemId,
		CreatedAt:          grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt:          grpcutil.TimestampToTime(info.UpdatedAt),
	}

	// Item
	if info.ItemId != nil {
		item := &apiresource.Item{
			ID:           *info.ItemId,
			Object:       constants.ObjectTypeItem,
			ItemTypeCode: constants.ItemTypeCodeProduct,
			CreatedAt:    grpcutil.TimestampToTime(info.CreatedAt),
			UpdatedAt:    grpcutil.TimestampToTime(info.UpdatedAt),
		}
		if info.ItemSku != nil && *info.ItemSku != "" {
			item.SKU = *info.ItemSku
		} else {
			item.SKU = info.ProductSku
		}
		l.Item = item
	}

	unitAbbr := info.QuantityUnitAbbreviation
	unitType := info.QuantityUnitType

	// Quantity ordered
	l.QuantityOrdered = &apiresource.Quantity{
		ID:           info.QuantityId,
		Object:       constants.ObjectTypeQuantity,
		Value:        info.QuantityValue,
		DisplayValue: apiresource.FormatDisplayValue(info.QuantityValue, unitAbbr, unitType),
	}

	// Unit price
	l.UnitPrice = &apiresource.Rate{
		ID:           info.UnitPriceId,
		Object:       constants.ObjectTypeRate,
		Value:        info.UnitPriceValue,
		DisplayValue: apiresource.FormatRateDisplayValue(info.UnitPriceValue, info.UnitPriceNumeratorUnitAbbreviation, "", info.UnitPriceDenominatorUnitAbbreviation),
		CreatedAt:    l.CreatedAt,
		UpdatedAt:    l.UpdatedAt,
	}

	// Unit cost
	if info.UnitCostId != nil {
		var unitCostValue, unitCostNumeratorAbbr, unitCostDenominatorAbbr string
		if info.UnitCostValue != nil {
			unitCostValue = *info.UnitCostValue
		}
		if info.UnitCostNumeratorUnitAbbreviation != nil {
			unitCostNumeratorAbbr = *info.UnitCostNumeratorUnitAbbreviation
		}
		if info.UnitCostDenominatorUnitAbbreviation != nil {
			unitCostDenominatorAbbr = *info.UnitCostDenominatorUnitAbbreviation
		}
		l.UnitCost = &apiresource.Rate{
			ID:           *info.UnitCostId,
			Object:       constants.ObjectTypeRate,
			Value:        unitCostValue,
			DisplayValue: apiresource.FormatRateDisplayValue(unitCostValue, unitCostNumeratorAbbr, "", unitCostDenominatorAbbr),
			CreatedAt:    l.CreatedAt,
			UpdatedAt:    l.UpdatedAt,
		}
	}

	// Quantity picked
	if info.QuantityPickedValue != nil {
		l.QuantityPicked = &apiresource.Quantity{
			ID:           info.Id + ":picked",
			Object:       constants.ObjectTypeQuantity,
			Value:        *info.QuantityPickedValue,
			DisplayValue: apiresource.FormatDisplayValue(*info.QuantityPickedValue, unitAbbr, unitType),
		}
	}

	// Quantity packed
	if info.QuantityPackedValue != nil {
		l.QuantityPacked = &apiresource.Quantity{
			ID:           info.Id + ":packed",
			Object:       constants.ObjectTypeQuantity,
			Value:        *info.QuantityPackedValue,
			DisplayValue: apiresource.FormatDisplayValue(*info.QuantityPackedValue, unitAbbr, unitType),
		}
	}

	// Quantity invoiced
	if info.QuantityInvoicedValue != nil {
		l.QuantityInvoiced = &apiresource.Quantity{
			ID:           info.Id + ":invoiced",
			Object:       constants.ObjectTypeQuantity,
			Value:        *info.QuantityInvoicedValue,
			DisplayValue: apiresource.FormatDisplayValue(*info.QuantityInvoicedValue, unitAbbr, unitType),
		}
	}

	// Completed at
	if info.CompletedAt != nil {
		t := grpcutil.TimestampToTime(info.CompletedAt)
		l.CompletedAt = &t
	}

	return l
}

func stashSalesOrderLineMeta(meta *resourcekit.LoadMeta, info *pb.SalesOrderLineInfo, line *apiresource.SalesOrderLineDetail) {
	if info.ItemId != nil {
		meta.Set(constants.ObjectTypeSalesOrderLine, line.ID, "item_id", *info.ItemId)
	}
	meta.Set(constants.ObjectTypeQuantity, info.QuantityId, "unit_id", info.QuantityUnitId)
	meta.Set(constants.ObjectTypeRate, info.UnitPriceId, "numerator_unit_id", info.UnitPriceNumeratorUnitId)
	meta.Set(constants.ObjectTypeRate, info.UnitPriceId, "denominator_unit_id", info.UnitPriceDenominatorUnitId)
	if info.UnitCostId != nil {
		if info.UnitCostNumeratorUnitId != nil {
			meta.Set(constants.ObjectTypeRate, *info.UnitCostId, "numerator_unit_id", *info.UnitCostNumeratorUnitId)
		}
		if info.UnitCostDenominatorUnitId != nil {
			meta.Set(constants.ObjectTypeRate, *info.UnitCostId, "denominator_unit_id", *info.UnitCostDenominatorUnitId)
		}
	}
}

func finalizeCustomerStubForInclude(c *apiresource.Customer, fallbackCreated, fallbackUpdated time.Time) {
	if c == nil {
		return
	}
	if c.Status == "" {
		c.Status = constants.AccountStatusCodeNormal
	}
	if c.CommissionPolicy == "" {
		c.CommissionPolicy = constants.CommissionPolicyApplied
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = fallbackCreated
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = fallbackUpdated
	}
}

func buildSOAddressFromProto(
	id string, name, line1, line2, locality, state, postalCode, country, phone, email *string,
	isDropShip *bool, geolocationID *string,
	createdAt time.Time, updatedAt time.Time,
) *apiresource.Address {
	addr := &apiresource.Address{
		ID:        id,
		Object:    constants.ObjectTypeAddress,
		Phone:     phone,
		Email:     email,
		Type:      constants.AddressTypeStandard,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	if name != nil {
		addr.Name = *name
	}

	if isDropShip != nil && *isDropShip {
		addr.Type = constants.AddressTypeDropShip
	}

	countryStr := ""
	if country != nil {
		countryStr = *country
	}

	geo := &apiresource.Geolocation{
		Object:      constants.ObjectTypeGeolocation,
		StreetLine1: line1,
		StreetLine2: line2,
		Locality:    locality,
		State:       state,
		PostalCode:  postalCode,
		Country:     countryStr,
	}
	if geolocationID != nil {
		geo.ID = *geolocationID
	}
	addr.Geolocation = geo

	return addr
}
