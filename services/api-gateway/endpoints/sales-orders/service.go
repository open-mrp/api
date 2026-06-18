package salesorderep

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
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
	ListSalesOrders(ctx context.Context, req *ListSalesOrdersRequest) (*apiresource.List[apiresource.SalesOrder], *apierror.APIError)
	GetSalesOrder(ctx context.Context, req *RetrieveSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError)
	CreateSalesOrder(ctx context.Context, req *CreateSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError)
	UpdateSalesOrder(ctx context.Context, req *UpdateSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError)
	DeleteSalesOrder(ctx context.Context, req *DeleteSalesOrderRequest) (*apiresource.EmptyResource, *apierror.APIError)
	BulkDeleteSalesOrders(ctx context.Context, req *BulkDeleteSalesOrdersRequest) (*apiresource.EmptyResource, *apierror.APIError)
	IssueSalesOrder(ctx context.Context, req *IssueSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError)
	UnissueSalesOrder(ctx context.Context, req *UnissueSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError)
	CloseSalesOrder(ctx context.Context, req *CloseSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError)
	OpenSalesOrder(ctx context.Context, req *OpenSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError)
	CheckoutSalesOrder(ctx context.Context, req *CheckoutSalesOrderRequest) (*CheckoutSalesOrderResponse, *apierror.APIError)
	QuoteSalesOrderPrices(ctx context.Context, req *QuoteSalesOrderPricesRequest) (*QuoteSalesOrderPricesResponse, *apierror.APIError)
	CreateSalesOrderProductionRun(ctx context.Context, req *CreateProductionRunRequest) (*CreateProductionRunResponse, *apierror.APIError)
	CreateSalesOrderLine(ctx context.Context, req *CreateSalesOrderLineRequest) (*apiresource.SalesOrderLine, *apierror.APIError)
	UpdateSalesOrderLine(ctx context.Context, req *UpdateSalesOrderLineRequest) (*apiresource.SalesOrderLine, *apierror.APIError)
	DeleteSalesOrderLine(ctx context.Context, req *DeleteSalesOrderLineRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type SalesOrderSvcConfig struct {
	// CoreClient (required) is the core-service sales gRPC client.
	CoreClient pb.CoreSalesServiceClient
}

type salesOrderSvcImpl struct {
	coreClient pb.CoreSalesServiceClient
}

var salesOrderEpSvcTracer = tracing.GetTracer("api-gateway.endpoints.sales-orders.service")

var salesOrderIncludes = []string{"customer", "sales_rep", "bill_to_address", "ship_to_address", "freight", "payment_term", "shipping_term", "order_discount", "totals", "related.pick", "related.production_run", "related.shipments", "lines", "lines.product", "lines.quantity_ordered", "lines.unit_price", "lines.unit_cost", "lines.totals"}

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

func (m *salesOrderSvcImpl) ListSalesOrders(ctx context.Context, req *ListSalesOrdersRequest) (*apiresource.List[apiresource.SalesOrder], *apierror.APIError) {
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
		// The list returns the full sales-order resource; ask the backend to
		// expand only what the caller requested (inline fields always present).
		Includes: withLinesForTotals(resourcekit.FilterIncludes(ctx, salesOrderIncludes...)),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, salesOrderEpSvcTracer, "service.sales_orders.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListSalesOrdersResponse, error) {
			return m.coreClient.ListSalesOrders(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	orders := make([]apiresource.SalesOrder, len(resp.SalesOrders))
	for i, o := range resp.SalesOrders {
		orders[i] = salesOrderDetailFromProto(o)
		stashSalesOrderMeta(ctx, o, &orders[i])
	}

	return apiresource.NewList(orders, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *salesOrderSvcImpl) GetSalesOrder(ctx context.Context, req *RetrieveSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError) {
	pbReq := &pb.GetSalesOrderRequest{
		Id:       req.SalesOrderID,
		Includes: withLinesForTotals(resourcekit.FilterIncludes(ctx, salesOrderIncludes...)),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, salesOrderEpSvcTracer, "service.sales_orders.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetSalesOrderResponse, error) {
			return m.coreClient.GetSalesOrder(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := salesOrderDetailFromProto(resp.SalesOrder)
	stashSalesOrderMeta(ctx, resp.SalesOrder, &result)
	return &result, nil
}

func (m *salesOrderSvcImpl) CreateSalesOrder(ctx context.Context, req *CreateSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError) {
	lines := make([]*pb.CreateSalesOrderLineInput, len(req.Lines))
	for i, l := range req.Lines {
		line := &pb.CreateSalesOrderLineInput{
			ProductId:          l.ProductID,
			ProductSku:         l.ProductSKU.Ptr(),
			ProductDescription: l.ProductDescription.Ptr(),
			QuantityValue:      l.Quantity.Value,
			QuantityUnitId:     l.Quantity.UnitID,
			EdiLineItemId:      l.EdiLineItemID.Ptr(),
		}
		if up, ok := l.UnitPrice.Value(); ok {
			line.UnitPriceValue = &up.Value
			line.UnitPriceNumeratorUnitId = &up.NumeratorUnitID
			line.UnitPriceDenominatorUnitId = &up.DenominatorUnitID
		}
		lines[i] = line
	}

	var carrierBillingType *string
	if v, ok := req.CarrierBillingType.Value(); ok {
		s := string(v)
		carrierBillingType = &s
	}

	pbReq := &pb.CreateSalesOrderRequest{
		BuyerAccountId:               req.BuyerAccountID,
		CustomerPoNumber:             req.CustomerPurchaseOrderNumber.Ptr(),
		Note:                         req.Note.Ptr(),
		CarrierId:                    req.CarrierID.Ptr(),
		ServiceLevelId:               req.ServiceLevelID.Ptr(),
		CarrierBillingType:           carrierBillingType,
		CarrierBillingAccount:        req.CarrierBillingAccountNumber.Ptr(),
		PriorityCode:                 req.PriorityCode,
		SalesRepId:                   req.SalesRepID.Ptr(),
		ShippingTermId:               req.ShippingTermID.Ptr(),
		PaymentTermId:                req.PaymentTermID.Ptr(),
		OrderDiscountId:              req.OrderDiscountID.Ptr(),
		BillToAddressId:              req.BillToAddressID,
		ShipToAddressId:              req.ShipToAddressID,
		Lines:                        lines,
		AcknowledgementEmailContacts: toSalesOrderEmailContactInputs(req.AcknowledgementEmailContacts),
		InvoiceEmailContacts:         toSalesOrderEmailContactInputs(req.InvoiceEmailContacts),
		Includes:                     resourcekit.FilterIncludes(ctx, salesOrderIncludes...),
	}

	if v, ok := req.PromisedAt.Value(); ok {
		pbReq.PromisedAt = timestamppb.New(v)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, salesOrderEpSvcTracer, "service.sales_orders.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateSalesOrderResponse, error) {
			return m.coreClient.CreateSalesOrder(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := salesOrderDetailFromProto(resp.SalesOrder)
	stashSalesOrderMeta(ctx, resp.SalesOrder, &result)
	return &result, nil
}

func (m *salesOrderSvcImpl) UpdateSalesOrder(ctx context.Context, req *UpdateSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError) {
	var carrierBillingType *string
	if v, ok := req.CarrierBillingType.Value(); ok {
		s := string(v)
		carrierBillingType = &s
	}

	pbReq := &pb.UpdateSalesOrderRequest{
		Id:                           req.SalesOrderID,
		CustomerPoNumber:             req.CustomerPurchaseOrderNumber.Ptr(),
		Note:                         req.Note.Ptr(),
		CarrierId:                    req.CarrierID.Ptr(),
		ServiceLevelId:               req.ServiceLevelID.Ptr(),
		CarrierBillingType:           carrierBillingType,
		CarrierBillingAccount:        req.CarrierBillingAccountNumber.Ptr(),
		PriorityCode:                 req.PriorityCode.Ptr(),
		SalesRepId:                   req.SalesRepID.Ptr(),
		ShippingTermId:               req.ShippingTermID.Ptr(),
		PaymentTermId:                req.PaymentTermID.Ptr(),
		OrderDiscountId:              req.OrderDiscountID.Ptr(),
		BillingAddressId:             req.BillingAddressID.Ptr(),
		ShippingAddressId:            req.ShippingAddressID.Ptr(),
		AcknowledgementEmailContacts: toSalesOrderEmailContactList(req.AcknowledgementEmailContacts.Ptr()),
		InvoiceEmailContacts:         toSalesOrderEmailContactList(req.InvoiceEmailContacts.Ptr()),
		Includes:                     resourcekit.FilterIncludes(ctx, salesOrderIncludes...),
	}

	if v, ok := req.AcknowledgmentStatus.Value(); ok {
		sent := v == constants.AcknowledgmentStatusSent
		pbReq.IsAcknowledgmentSent = &sent
	}

	resp, apiErr := grpcutil.CallRPC(ctx, salesOrderEpSvcTracer, "service.sales_orders.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateSalesOrderResponse, error) {
			return m.coreClient.UpdateSalesOrder(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := salesOrderDetailFromProto(resp.SalesOrder)
	stashSalesOrderMeta(ctx, resp.SalesOrder, &result)
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

func (m *salesOrderSvcImpl) IssueSalesOrder(ctx context.Context, req *IssueSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError) {
	return m.changeSalesOrderStatus(ctx, req.SalesOrderID, constants.SalesOrderStatusChangeIssue, req.NotifyCustomer)
}

func (m *salesOrderSvcImpl) UnissueSalesOrder(ctx context.Context, req *UnissueSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError) {
	return m.changeSalesOrderStatus(ctx, req.SalesOrderID, constants.SalesOrderStatusChangeUnissue, req.NotifyCustomer)
}

func (m *salesOrderSvcImpl) CloseSalesOrder(ctx context.Context, req *CloseSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError) {
	return m.changeSalesOrderStatus(ctx, req.SalesOrderID, constants.SalesOrderStatusChangeClose, req.NotifyCustomer)
}

func (m *salesOrderSvcImpl) OpenSalesOrder(ctx context.Context, req *OpenSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError) {
	return m.changeSalesOrderStatus(ctx, req.SalesOrderID, constants.SalesOrderStatusChangeOpen, req.NotifyCustomer)
}

// changeSalesOrderStatus performs a sales order status transition via the core
// service. The four action endpoints (issue, unissue, close, open) funnel
// through here with their fixed action and the caller's notify_customer flag.
func (m *salesOrderSvcImpl) changeSalesOrderStatus(ctx context.Context, id string, action constants.SalesOrderStatusChange, notifyCustomer bool) (*apiresource.SalesOrder, *apierror.APIError) {
	pbReq := &pb.ChangeSalesOrderStatusRequest{
		Id:           id,
		StatusChange: string(action),
		SendEmail:    notifyCustomer,
		Includes:     resourcekit.FilterIncludes(ctx, salesOrderIncludes...),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, salesOrderEpSvcTracer, "service.sales_orders.change_status", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ChangeSalesOrderStatusResponse, error) {
			return m.coreClient.ChangeSalesOrderStatus(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := salesOrderDetailFromProto(resp.SalesOrder)
	stashSalesOrderMeta(ctx, resp.SalesOrder, &result)
	return &result, nil
}

func (m *salesOrderSvcImpl) CheckoutSalesOrder(ctx context.Context, req *CheckoutSalesOrderRequest) (*CheckoutSalesOrderResponse, *apierror.APIError) {
	pbReq := &pb.CheckoutSalesOrderRequest{
		Id:         req.SalesOrderID,
		Email:      req.Email,
		SuccessUrl: req.SuccessURL.Ptr(),
		CancelUrl:  req.CancelURL.Ptr(),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, salesOrderEpSvcTracer, "service.sales_orders.checkout", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CheckoutSalesOrderResponse, error) {
			return m.coreClient.CheckoutSalesOrder(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return &CheckoutSalesOrderResponse{
		Object:      constants.ObjectTypeCheckoutSalesOrderResponse,
		CheckoutURL: resp.CheckoutUrl,
	}, nil
}

func (m *salesOrderSvcImpl) QuoteSalesOrderPrices(ctx context.Context, req *QuoteSalesOrderPricesRequest) (*QuoteSalesOrderPricesResponse, *apierror.APIError) {
	lines := make([]*pb.QuoteSalesOrderLineInput, len(req.Lines))
	for i, l := range req.Lines {
		lines[i] = &pb.QuoteSalesOrderLineInput{
			ProductId:      l.ProductID,
			QuantityValue:  l.Quantity.Value,
			QuantityUnitId: l.Quantity.UnitID,
		}
	}

	pbReq := &pb.QuoteSalesOrderLinePricesRequest{
		BuyerAccountId: req.BuyerAccountID,
		Lines:          lines,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, salesOrderEpSvcTracer, "service.sales_orders.quote_prices", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.QuoteSalesOrderLinePricesResponse, error) {
			return m.coreClient.QuoteSalesOrderLinePrices(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	out := make([]QuotedSalesOrderLine, len(resp.Lines))
	for i, l := range resp.Lines {
		out[i] = QuotedSalesOrderLine{
			ProductID:                  l.ProductId,
			UnitPriceValue:             l.UnitPriceValue,
			UnitPriceNumeratorUnitID:   l.UnitPriceNumeratorUnitId,
			UnitPriceDenominatorUnitID: l.UnitPriceDenominatorUnitId,
		}
	}

	return &QuoteSalesOrderPricesResponse{
		Object: constants.ObjectTypeSalesOrderPriceQuote,
		Lines:  out,
	}, nil
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
		Object: constants.ObjectTypeCreateProductionRunResponse,
		ProductionRun: CreateProductionRunResponseRef{
			ID:     resp.ProductionRunId,
			Object: constants.ObjectTypeProductionRun,
		},
	}, nil
}

func (m *salesOrderSvcImpl) CreateSalesOrderLine(ctx context.Context, req *CreateSalesOrderLineRequest) (*apiresource.SalesOrderLine, *apierror.APIError) {
	pbReq := &pb.CreateSalesOrderLineRequest{
		SalesOrderId:               req.SalesOrderID,
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

	resp, apiErr := grpcutil.CallRPC(ctx, salesOrderEpSvcTracer, "service.sales_orders.create_line", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateSalesOrderLineResponse, error) {
			return m.coreClient.CreateSalesOrderLine(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := salesOrderLineDetailFromProto(resp.SalesOrderLine)
	stashSalesOrderLineMeta(resourcekit.GetLoadMeta(ctx), resp.SalesOrderLine, &result)
	return &result, nil
}

func (m *salesOrderSvcImpl) UpdateSalesOrderLine(ctx context.Context, req *UpdateSalesOrderLineRequest) (*apiresource.SalesOrderLine, *apierror.APIError) {
	pbReq := &pb.UpdateSalesOrderLineRequest{
		SalesOrderId:       req.SalesOrderID,
		Id:                 req.SalesOrderLineID,
		ItemId:             req.ItemID.Ptr(),
		ProductSku:         req.ProductSKU.Ptr(),
		ProductDescription: req.ProductDescription.Ptr(),
	}
	if v, ok := req.Quantity.Value(); ok {
		pbReq.QuantityValue = &v.Value
		pbReq.QuantityUnitId = &v.UnitID
	}
	if v, ok := req.UnitPrice.Value(); ok {
		pbReq.UnitPriceValue = &v.Value
		pbReq.UnitPriceNumeratorUnitId = &v.NumeratorUnitID
		pbReq.UnitPriceDenominatorUnitId = &v.DenominatorUnitID
	}
	if v, ok := req.UnitCost.Value(); ok {
		pbReq.UnitCostValue = &v.Value
		pbReq.UnitCostNumeratorUnitId = &v.NumeratorUnitID
		pbReq.UnitCostDenominatorUnitId = &v.DenominatorUnitID
	}

	resp, apiErr := grpcutil.CallRPC(ctx, salesOrderEpSvcTracer, "service.sales_orders.update_line", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateSalesOrderLineResponse, error) {
			return m.coreClient.UpdateSalesOrderLine(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := salesOrderLineDetailFromProto(resp.SalesOrderLine)
	stashSalesOrderLineMeta(resourcekit.GetLoadMeta(ctx), resp.SalesOrderLine, &result)
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

func salesOrderDetailFromProto(info *pb.SalesOrderInfo) apiresource.SalesOrder {
	d := apiresource.SalesOrder{
		ID:                          info.Id,
		Object:                      constants.ObjectTypeSalesOrder,
		Number:                      info.Number,
		CustomerPurchaseOrderNumber: info.CustomerPoNumber,
		Note:                        info.Note,
		AcknowledgmentStatus:        acknowledgmentStatusFromBool(info.IsAcknowledgmentSent),
		PaymentStatus:               salesOrderPaymentStatusFromProto(info.PaymentStatus),
		Status:                      constants.SalesOrderStatusCode(info.StatusCode),
		Priority:                    constants.PriorityCode(info.PriorityCode),
		LineCount:                   info.LineCount,
		CreatedAt:                   grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt:                   grpcutil.TimestampToTime(info.UpdatedAt),
	}

	// sales_rep, totals, and the related records (pick/production_run/shipments)
	// are all expandable — populated from stashed meta only when requested. The
	// related group is left nil here and created lazily when one of its members
	// is expanded, so it serializes to null when no related include is requested.

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

// withLinesForTotals ensures the backend returns line data whenever the
// `totals` include is requested, since order totals are derived from line
// values. Without this, ?include=totals alone would have no lines to sum and
// totals would resolve to null.
func withLinesForTotals(includes []string) []string {
	hasTotals, hasLines := false, false
	for _, inc := range includes {
		switch {
		case inc == "totals":
			hasTotals = true
		case inc == "lines" || strings.HasPrefix(inc, "lines."):
			hasLines = true
		}
	}
	if hasTotals && !hasLines {
		includes = append(includes, "lines")
	}
	return includes
}

// salesOrderTotalsFromLines derives the order's monetary totals and pick/
// fulfillment progress from its line data. Returns nil when no lines are present
// on the proto (i.e. the `lines` include was not requested). This mirrors the
// totals the legacy frontend computed client-side from the order's lines.
func salesOrderTotalsFromLines(lines []*pb.SalesOrderLineInfo) *apiresource.SalesOrderTotals {
	if len(lines) == 0 {
		return nil
	}

	var totalOrdered, totalPacked, totalInvoiced decimal.Decimal

	for _, l := range lines {
		price := parseDecimal(l.UnitPriceValue)
		totalOrdered = totalOrdered.Add(price.Mul(parseDecimal(l.QuantityValue)))
		if l.QuantityPackedValue != nil {
			totalPacked = totalPacked.Add(price.Mul(parseDecimal(*l.QuantityPackedValue)))
		}
		if l.QuantityInvoicedValue != nil {
			totalInvoiced = totalInvoiced.Add(price.Mul(parseDecimal(*l.QuantityInvoicedValue)))
		}
	}

	return &apiresource.SalesOrderTotals{
		Object:   constants.ObjectTypeSalesOrderTotals,
		Ordered:  totalOrdered.String(),
		Packed:   totalPacked.String(),
		Invoiced: totalInvoiced.String(),
	}
}

// acknowledgmentStatusFromBool maps the legacy boolean acknowledgment flag to
// the AcknowledgmentStatus enum.
func acknowledgmentStatusFromBool(sent bool) constants.AcknowledgmentStatus {
	if sent {
		return constants.AcknowledgmentStatusSent
	}
	return constants.AcknowledgmentStatusNotSent
}

// salesOrderPaymentStatusFromProto maps the proto payment status to the resource
// enum, defaulting to unpaid for any empty or unrecognized value.
func salesOrderPaymentStatusFromProto(status string) constants.SalesOrderPaymentStatus {
	s := constants.SalesOrderPaymentStatus(status)
	if !s.IsValid() {
		return constants.SalesOrderPaymentStatusUnpaid
	}
	return s
}

// parseDecimal parses a decimal string, treating empty/invalid input as zero.
func parseDecimal(s string) decimal.Decimal {
	if s == "" {
		return decimal.Zero
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero
	}
	return d
}

func stashSalesOrderMeta(ctx context.Context, info *pb.SalesOrderInfo, d *apiresource.SalesOrder) {
	if info == nil {
		return
	}

	meta := resourcekit.GetLoadMeta(ctx)

	// customer is an expandable reference: stash the FK id so LoadCustomers
	// fetches the real Customer on ?include=customer. Never fabricate.
	if info.CustomerId != "" {
		meta.Set(constants.ObjectTypeSalesOrder, d.ID, "customer_id", info.CustomerId)
	}

	// Sales rep (expandable)
	if info.SalesRepId != nil {
		meta.Set(constants.ObjectTypeSalesOrder, d.ID, "sales_rep",
			apiresource.NewActor(*info.SalesRepId, constants.ActorTypeUser, info.SalesRepName, nil))
	}

	// Totals (expandable) — derived from line data when lines are present.
	if totals := salesOrderTotalsFromLines(info.Lines); totals != nil {
		meta.Set(constants.ObjectTypeSalesOrder, d.ID, "totals", totals)
	}

	// Related records (expandable): pick / production_run / shipments.
	if info.PickId != nil {
		meta.Set(constants.ObjectTypeSalesOrder, d.ID, "related_pick",
			apiresource.NewRecord(*info.PickId, constants.RecordTypePick))
	}
	if info.ProductionRunId != nil {
		meta.Set(constants.ObjectTypeSalesOrder, d.ID, "related_production_run",
			apiresource.NewRecord(*info.ProductionRunId, constants.RecordTypeProductionRun))
	}
	// Linked shipments (populated by the backend only when related.shipments is
	// requested). Build the record list from the ids the order carries.
	if len(info.ShipmentIds) > 0 {
		records := make([]apiresource.Record, len(info.ShipmentIds))
		for i, sid := range info.ShipmentIds {
			records[i] = *apiresource.NewRecord(sid, constants.RecordTypeShipment)
		}
		meta.Set(constants.ObjectTypeSalesOrder, d.ID, "related_shipments",
			apiresource.NewList(records, apiresource.PageInfo{}))
	}

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
	meta.Set(constants.ObjectTypeSalesOrder, d.ID, "freight", freight)

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
	lines := make([]apiresource.SalesOrderLine, len(info.Lines))
	for i, l := range info.Lines {
		lines[i] = salesOrderLineDetailFromProto(l)
		stashSalesOrderLineMeta(meta, l, &lines[i])
	}
	meta.Set(constants.ObjectTypeSalesOrder, d.ID, "lines", apiresource.NewList(lines, apiresource.PageInfo{}))
}

func salesOrderLineDetailFromProto(info *pb.SalesOrderLineInfo) apiresource.SalesOrderLine {
	l := apiresource.SalesOrderLine{
		ID:                 info.Id,
		Object:             constants.ObjectTypeSalesOrderLine,
		LineItemNumber:     info.LineItemNumber,
		ProductSKU:         info.ProductSku,
		ProductDescription: info.ProductDescription,
		CreatedAt:          grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt:          grpcutil.TimestampToTime(info.UpdatedAt),
	}

	// Product — lightweight reference, fully loaded when lines.product is included.
	if info.ProductId != nil {
		l.Product = &apiresource.Product{
			ID:     *info.ProductId,
			Object: constants.ObjectTypeProduct,
		}
	}

	// quantity_ordered, unit_price, unit_cost, and totals are expandable —
	// populated from stashed meta only when requested.

	return l
}

// buildLineQuantityOrdered builds the quantity-ordered sub-resource for a line.
func buildLineQuantityOrdered(info *pb.SalesOrderLineInfo) *apiresource.Quantity {
	return &apiresource.Quantity{
		ID:           info.QuantityId,
		Object:       constants.ObjectTypeQuantity,
		Value:        info.QuantityValue,
		DisplayValue: apiresource.FormatDisplayValue(info.QuantityValue, info.QuantityUnitAbbreviation, info.QuantityUnitType),
	}
}

// buildLineUnitPrice builds the unit-price rate sub-resource for a line.
func buildLineUnitPrice(info *pb.SalesOrderLineInfo, createdAt, updatedAt time.Time) *apiresource.Rate {
	return &apiresource.Rate{
		ID:           info.UnitPriceId,
		Object:       constants.ObjectTypeRate,
		Value:        info.UnitPriceValue,
		DisplayValue: apiresource.FormatRateDisplayValue(info.UnitPriceValue, info.UnitPriceNumeratorUnitAbbreviation, "", info.UnitPriceDenominatorUnitAbbreviation),
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}
}

// buildLineUnitCost builds the unit-cost rate sub-resource for a line, or nil
// when the line has no unit cost.
func buildLineUnitCost(info *pb.SalesOrderLineInfo, createdAt, updatedAt time.Time) *apiresource.Rate {
	if info.UnitCostId == nil {
		return nil
	}
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
	return &apiresource.Rate{
		ID:           *info.UnitCostId,
		Object:       constants.ObjectTypeRate,
		Value:        unitCostValue,
		DisplayValue: apiresource.FormatRateDisplayValue(unitCostValue, unitCostNumeratorAbbr, "", unitCostDenominatorAbbr),
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}
}

// buildLineTotals derives the per-line monetary totals (ordered/packed/invoiced).
func buildLineTotals(info *pb.SalesOrderLineInfo) *apiresource.SalesOrderTotals {
	price := parseDecimal(info.UnitPriceValue)
	var packed, invoiced decimal.Decimal
	if info.QuantityPackedValue != nil {
		packed = parseDecimal(*info.QuantityPackedValue)
	}
	if info.QuantityInvoicedValue != nil {
		invoiced = parseDecimal(*info.QuantityInvoicedValue)
	}
	return &apiresource.SalesOrderTotals{
		Object:   constants.ObjectTypeSalesOrderTotals,
		Ordered:  price.Mul(parseDecimal(info.QuantityValue)).String(),
		Packed:   price.Mul(packed).String(),
		Invoiced: price.Mul(invoiced).String(),
	}
}

// stashSalesOrderLineMeta stashes a line's expandable sub-resources
// (quantity_ordered, unit_price, unit_cost, totals) so the include resolver can
// populate them when requested.
func stashSalesOrderLineMeta(meta *resourcekit.LoadMeta, info *pb.SalesOrderLineInfo, line *apiresource.SalesOrderLine) {
	meta.Set(constants.ObjectTypeSalesOrderLine, line.ID, "quantity_ordered", buildLineQuantityOrdered(info))
	meta.Set(constants.ObjectTypeSalesOrderLine, line.ID, "unit_price", buildLineUnitPrice(info, line.CreatedAt, line.UpdatedAt))
	if unitCost := buildLineUnitCost(info, line.CreatedAt, line.UpdatedAt); unitCost != nil {
		meta.Set(constants.ObjectTypeSalesOrderLine, line.ID, "unit_cost", unitCost)
	}
	meta.Set(constants.ObjectTypeSalesOrderLine, line.ID, "totals", buildLineTotals(info))
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
