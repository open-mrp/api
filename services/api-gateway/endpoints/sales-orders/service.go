package salesorderep

import (
	"context"
	"fmt"
	"strings"
	"time"

	productionrunep "github.com/augno/api/services/api-gateway/endpoints/production-runs"
	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
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
	QuoteSalesOrderFreight(ctx context.Context, req *QuoteSalesOrderFreightRequest) (*QuoteSalesOrderFreightResponse, *apierror.APIError)
	CreateSalesOrderProductionRun(ctx context.Context, req *CreateProductionRunRequest) (*apiresource.ProductionRun, *apierror.APIError)
	CreateSalesOrderLine(ctx context.Context, req *CreateSalesOrderLineRequest) (*apiresource.SalesOrderLine, *apierror.APIError)
	UpdateSalesOrderLine(ctx context.Context, req *UpdateSalesOrderLineRequest) (*apiresource.SalesOrderLine, *apierror.APIError)
	DeleteSalesOrderLine(ctx context.Context, req *DeleteSalesOrderLineRequest) (*apiresource.EmptyResource, *apierror.APIError)
	ReorderSalesOrderLines(ctx context.Context, req *ReorderSalesOrderLinesRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type SalesOrderSvcConfig struct {
	// CoreClient (required) is the core-service sales gRPC client.
	CoreClient pb.CoreSalesServiceClient
}

type salesOrderSvcImpl struct {
	coreClient pb.CoreSalesServiceClient
}

var salesOrderEpSvcTracer = tracing.GetTracer("api-gateway.endpoints.sales-orders.service")

var salesOrderIncludes = []string{"customer", "sales_rep", "bill_to_address", "ship_to_address", "freight", "payment_term", "shipping_term", "order_discount", "totals", "contacts", "related.pick", "related.production_run", "related.shipments", "related.invoices", "lines", "lines.product", "lines.quantity_ordered", "lines.quantity_ordered.unit", "lines.unit_price", "lines.unit_price.numerator_unit", "lines.unit_price.denominator_unit", "lines.unit_cost", "lines.unit_cost.numerator_unit", "lines.unit_cost.denominator_unit", "lines.totals"}

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
		Cursor:           req.Cursor,
		Limit:            req.Limit,
		Query:            req.Query,
		StatusCodes:      req.StatusCodes,
		ItemIds:          req.ItemIDs,
		ProductLineIds:   req.ProductLineIDs,
		CustomerIds:      req.CustomerIDs,
		CustomerGroupIds: req.CustomerGroupIDs,
		SalesRepIds:      req.SalesRepIDs,
		StartDate:        req.StartDate,
		EndDate:          req.EndDate,
		ShipByAfter:      req.ShipByAfter,
		ShipByBefore:     req.ShipByBefore,
		PastDue:          req.PastDue,
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
	orderIDs := make([]string, len(resp.SalesOrders))
	for i, o := range resp.SalesOrders {
		orders[i] = salesOrderDetailFromProto(o)
		stashSalesOrderMeta(ctx, o, &orders[i])
		orderIDs[i] = orders[i].ID
	}
	hydrateSalesReps(ctx, orderIDs)

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
	hydrateSalesReps(ctx, []string{result.ID})
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
	hydrateSalesReps(ctx, []string{result.ID})
	return &result, nil
}

func (m *salesOrderSvcImpl) UpdateSalesOrder(ctx context.Context, req *UpdateSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError) {
	// Clearable enum: map Clearable[CarrierBillingType] to a StringPatch (clear vs set vs leave).
	var carrierBillingTypePatch *pb.StringPatch
	switch {
	case req.CarrierBillingType.IsClear():
		carrierBillingTypePatch = &pb.StringPatch{Clear: true}
	case req.CarrierBillingType.IsSet():
		v, _ := req.CarrierBillingType.Value()
		s := string(v)
		carrierBillingTypePatch = &pb.StringPatch{Value: &s}
	}

	pbReq := &pb.UpdateSalesOrderRequest{
		Id: req.SalesOrderID,
		// Clearable nullable fields → *Patch (clear / set / leave).
		CustomerPoNumber:      field.StringClearableToProto(req.CustomerPurchaseOrderNumber),
		Note:                  field.StringClearableToProto(req.Note),
		ServiceLevelId:        field.StringClearableToProto(req.ServiceLevelID),
		CarrierBillingType:    carrierBillingTypePatch,
		CarrierBillingAccount: field.StringClearableToProto(req.CarrierBillingAccountNumber),
		SalesRepId:            field.StringClearableToProto(req.SalesRepID),
		OrderDiscountId:       field.StringClearableToProto(req.OrderDiscountID),
		PromisedAt:            field.TimestampClearableToProto(req.PromisedAt),
		// Non-nullable optional fields → *string (set or leave; not clearable).
		CarrierId:                    req.CarrierID.Ptr(),
		PriorityCode:                 req.PriorityCode.Ptr(),
		ShippingTermId:               req.ShippingTermID.Ptr(),
		PaymentTermId:                req.PaymentTermID.Ptr(),
		BillingAddressId:             req.BillingAddressID.Ptr(),
		ShippingAddressId:            req.ShippingAddressID.Ptr(),
		CustomerId:                   req.CustomerID.Ptr(),
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
	hydrateSalesReps(ctx, []string{result.ID})
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
	return m.changeSalesOrderStatus(ctx, req.SalesOrderID, constants.SalesOrderStatusChangeUnissue, false)
}

func (m *salesOrderSvcImpl) CloseSalesOrder(ctx context.Context, req *CloseSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError) {
	return m.changeSalesOrderStatus(ctx, req.SalesOrderID, constants.SalesOrderStatusChangeClose, false)
}

func (m *salesOrderSvcImpl) OpenSalesOrder(ctx context.Context, req *OpenSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError) {
	return m.changeSalesOrderStatus(ctx, req.SalesOrderID, constants.SalesOrderStatusChangeOpen, false)
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
	hydrateSalesReps(ctx, []string{result.ID})
	return &result, nil
}

func (m *salesOrderSvcImpl) CheckoutSalesOrder(ctx context.Context, req *CheckoutSalesOrderRequest) (*CheckoutSalesOrderResponse, *apierror.APIError) {
	pbReq := &pb.CheckoutSalesOrderRequest{
		Id:    req.SalesOrderID,
		Email: req.Email,
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

	unitIDs := make([]string, 0, len(resp.Lines)*2)
	productIDs := make([]string, 0, len(resp.Lines))
	for _, l := range resp.Lines {
		unitIDs = append(unitIDs, l.UnitPriceNumeratorUnitId, l.UnitPriceDenominatorUnitId)
		productIDs = append(productIDs, l.ProductId)
	}
	units, apiErr := m.hydrateQuoteUnits(ctx, unitIDs...)
	if apiErr != nil {
		return nil, apiErr
	}
	products, apiErr := m.hydrateQuoteProducts(ctx, productIDs...)
	if apiErr != nil {
		return nil, apiErr
	}

	out := make([]QuotedSalesOrderLine, len(resp.Lines))
	for i, l := range resp.Lines {
		out[i] = QuotedSalesOrderLine{
			Object:    constants.ObjectTypeSalesOrderPriceQuoteLine,
			Product:   products[l.ProductId],
			UnitPrice: newSalesOrderQuoteRate(l.UnitPriceValue, units[l.UnitPriceNumeratorUnitId], units[l.UnitPriceDenominatorUnitId]),
		}
	}

	return &QuoteSalesOrderPricesResponse{
		Object: constants.ObjectTypeSalesOrderPriceQuote,
		Lines:  apiresource.NewList(out, apiresource.PageInfo{}),
	}, nil
}

// hydrateQuoteUnits batch-loads fully presented Unit resources for the given unit IDs so quote rates present their units the same way persisted resources do. The core quote RPCs return only unit IDs, so the gateway resolves them here. Returns a lookup keyed by unit ID; a missing ID simply maps to nil.
func (m *salesOrderSvcImpl) hydrateQuoteUnits(ctx context.Context, ids ...string) (map[string]*apiresource.Unit, *apierror.APIError) {
	unique := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	if len(unique) == 0 {
		return map[string]*apiresource.Unit{}, nil
	}

	loaded, apiErr := resourceloaders.LoadUnits(ctx, unique)
	if apiErr != nil {
		return nil, apiErr
	}

	out := make(map[string]*apiresource.Unit, len(loaded))
	for id, v := range loaded {
		if u, ok := v.(*apiresource.Unit); ok {
			out[id] = u
		}
	}
	return out, nil
}

// hydrateQuoteProducts batch-loads fully presented Product resources for the given product IDs so quote lines present the priced product the same way persisted resources do. The core quote RPC returns only product IDs, so the gateway resolves them here. Returns a lookup keyed by product ID; a missing ID simply maps to nil.
func (m *salesOrderSvcImpl) hydrateQuoteProducts(ctx context.Context, ids ...string) (map[string]*apiresource.Product, *apierror.APIError) {
	unique := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	if len(unique) == 0 {
		return map[string]*apiresource.Product{}, nil
	}

	loaded, apiErr := resourceloaders.LoadProducts(ctx, unique)
	if apiErr != nil {
		return nil, apiErr
	}

	out := make(map[string]*apiresource.Product, len(loaded))
	for id, v := range loaded {
		if p, ok := v.(*apiresource.Product); ok {
			out[id] = p
		}
	}
	return out, nil
}

func (m *salesOrderSvcImpl) QuoteSalesOrderFreight(ctx context.Context, req *QuoteSalesOrderFreightRequest) (*QuoteSalesOrderFreightResponse, *apierror.APIError) {
	pbReq := &pb.QuoteSalesOrderFreightRequest{Id: req.SalesOrderID}

	resp, apiErr := grpcutil.CallRPC(ctx, salesOrderEpSvcTracer, "service.sales_orders.quote_freight", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.QuoteSalesOrderFreightResponse, error) {
			return m.coreClient.QuoteSalesOrderFreight(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	units, apiErr := m.hydrateQuoteUnits(ctx, resp.UnitPriceNumeratorUnitId, resp.UnitPriceDenominatorUnitId)
	if apiErr != nil {
		return nil, apiErr
	}

	return &QuoteSalesOrderFreightResponse{
		Object:    constants.ObjectTypeSalesOrderFreightQuote,
		UnitPrice: newSalesOrderQuoteRate(resp.UnitPriceValue, units[resp.UnitPriceNumeratorUnitId], units[resp.UnitPriceDenominatorUnitId]),
	}, nil
}

func (m *salesOrderSvcImpl) CreateSalesOrderProductionRun(ctx context.Context, req *CreateProductionRunRequest) (*apiresource.ProductionRun, *apierror.APIError) {
	pbReq := &pb.CreateSalesOrderProductionRunRequest{Id: req.SalesOrderID}

	resp, apiErr := grpcutil.CallRPC(ctx, salesOrderEpSvcTracer, "service.sales_orders.create_production_run", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateSalesOrderProductionRunResponse, error) {
			return m.coreClient.CreateSalesOrderProductionRun(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	run := productionrunep.ProductionRunFromProto(resp.ProductionRun)
	productionrunep.StashProductionRunMeta(meta, resp.ProductionRun)
	return &run, nil
}

func (m *salesOrderSvcImpl) CreateSalesOrderLine(ctx context.Context, req *CreateSalesOrderLineRequest) (*apiresource.SalesOrderLine, *apierror.APIError) {
	pbReq := &pb.CreateSalesOrderLineRequest{
		SalesOrderId:       req.SalesOrderID,
		ProductId:          req.ProductID,
		ProductSku:         req.ProductSKU,
		ProductDescription: req.ProductDescription.Ptr(),
		QuantityValue:      req.Quantity.Value,
		QuantityUnitId:     req.Quantity.UnitID,
	}
	// Unit price is an optional override. Leave it empty when omitted so the core
	// service prices the line from the product (unit cost is always server-derived).
	if up, ok := req.UnitPrice.Value(); ok {
		pbReq.UnitPriceValue = up.Value
		pbReq.UnitPriceNumeratorUnitId = up.NumeratorUnitID
		pbReq.UnitPriceDenominatorUnitId = up.DenominatorUnitID
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

func (m *salesOrderSvcImpl) ReorderSalesOrderLines(ctx context.Context, req *ReorderSalesOrderLinesRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.ReorderSalesOrderLinesRequest{
		SalesOrderId: req.SalesOrderID,
		LineIds:      req.LineIDs,
	}

	_, apiErr := grpcutil.CallRPC(ctx, salesOrderEpSvcTracer, "service.sales_orders.reorder_lines", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ReorderSalesOrderLinesResponse, error) {
			return m.coreClient.ReorderSalesOrderLines(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func (m *salesOrderSvcImpl) UpdateSalesOrderLine(ctx context.Context, req *UpdateSalesOrderLineRequest) (*apiresource.SalesOrderLine, *apierror.APIError) {
	pbReq := &pb.UpdateSalesOrderLineRequest{
		SalesOrderId:       req.SalesOrderID,
		Id:                 req.SalesOrderLineID,
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
		PaymentIntentIDs:            paymentIntentIDsFromProto(info.PaymentIntentIds),
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
	if info.ShipByDate != nil {
		t := grpcutil.TimestampToTime(info.ShipByDate)
		d.ShipByDate = &t
	}
	d.LeadTimeDays = info.LeadTimeDays
	if info.LeadTimeSourceCode != nil {
		source := constants.LeadTimeSource(*info.LeadTimeSourceCode)
		d.LeadTimeSource = &source
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

// salesOrderTotalsFromOrder derives the order's monetary totals — with per-stage
// completion progress — from its line data and the order-level fulfillment
// aggregate. The stage amounts are summed from the lines; the completion
// fractions come from the order's PickedCompletion/PackedCompletion/
// InvoicedCompletion, computed server-side over all line items. Returns nil when
// no lines are present on the proto (i.e. the `lines` include was not requested),
// since the amounts cannot be derived without them.
func salesOrderTotalsFromOrder(info *pb.SalesOrderInfo) *apiresource.SalesOrderTotals {
	if len(info.Lines) == 0 {
		return nil
	}

	var totalOrdered, totalPicked, totalPacked, totalInvoiced decimal.Decimal

	for _, l := range info.Lines {
		price := parseDecimal(l.UnitPriceValue)
		totalOrdered = totalOrdered.Add(price.Mul(parseDecimal(l.QuantityValue)))
		if l.QuantityPickedValue != nil {
			totalPicked = totalPicked.Add(price.Mul(parseDecimal(*l.QuantityPickedValue)))
		}
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
		Picked:   apiresource.SalesOrderStageTotal{Object: constants.ObjectTypeSalesOrderStageTotal, Amount: totalPicked.String(), Completion: info.PickedCompletion},
		Packed:   apiresource.SalesOrderStageTotal{Object: constants.ObjectTypeSalesOrderStageTotal, Amount: totalPacked.String(), Completion: info.PackedCompletion},
		Invoiced: apiresource.SalesOrderStageTotal{Object: constants.ObjectTypeSalesOrderStageTotal, Amount: totalInvoiced.String(), Completion: info.InvoicedCompletion},
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

// paymentIntentIDsFromProto guarantees a non-nil slice so the field serializes as an empty JSON array rather than null.
func paymentIntentIDsFromProto(ids []string) []string {
	if ids == nil {
		return []string{}
	}
	return ids
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

// orEmptyStrings returns a non-nil slice so the value serializes as a JSON array
// (`[]`) rather than `null`.
func orEmptyStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
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

	// Totals (expandable) — monetary amounts from line data plus order-level
	// completion progress; populated only when lines are present.
	if totals := salesOrderTotalsFromOrder(info); totals != nil {
		meta.Set(constants.ObjectTypeSalesOrder, d.ID, "totals", totals)
	}

	// Contacts (expandable): email recipients grouped by notification purpose. The
	// backend only populates the email lists when contacts is included; always stash
	// the object so the populate step (which runs only for requested includes) finds it.
	// Coalesce nil to empty slices so each group always serializes as a JSON array.
	meta.Set(constants.ObjectTypeSalesOrder, d.ID, "contacts", &apiresource.OrderContact{
		Object:          constants.ObjectTypeOrderContact,
		Invoice:         orEmptyStrings(info.InvoiceEmails),
		Acknowledgement: orEmptyStrings(info.AcknowledgementEmails),
	})

	// Related records (expandable): stash the referenced ids so the include resolver can fetch each record's number and status from its owning service when related.pick / related.production_run / related.shipments is requested.
	if info.PickId != nil {
		meta.Set(constants.ObjectTypeSalesOrder, d.ID, "related_pick_id", *info.PickId)
	}
	if info.ProductionRunId != nil {
		meta.Set(constants.ObjectTypeSalesOrder, d.ID, "related_production_run_id", *info.ProductionRunId)
	}
	if len(info.ShipmentIds) > 0 {
		meta.Set(constants.ObjectTypeSalesOrder, d.ID, "related_shipment_ids", info.ShipmentIds)
	}
	if len(info.InvoiceIds) > 0 {
		meta.Set(constants.ObjectTypeSalesOrder, d.ID, "related_invoice_ids", info.InvoiceIds)
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

// hydrateSalesReps batch-fills each order's sales-rep actor with the rep's display
// name, handle (email), and avatar URL. The backend only ships the rep's id and name
// on SalesOrderInfo, so handle/avatar_url would otherwise always be null. The actors
// are stashed as pointers in the load meta, so mutating them here also updates the
// object the sales_rep populate step later attaches to each order. Runs only when the
// caller requested ?include=sales_rep (skips the extra core lookup otherwise) and
// batches all reps into a single account-user fetch. Best-effort: on failure each rep
// keeps its id + name.
func hydrateSalesReps(ctx context.Context, orderIDs []string) {
	if !resourcekit.RequestedIncludeSet(ctx)["sales_rep"] {
		return
	}
	meta := resourcekit.GetLoadMeta(ctx)
	actors := make([]*apiresource.Actor, 0, len(orderIDs))
	for _, id := range orderIDs {
		if v, ok := meta.Get(constants.ObjectTypeSalesOrder, id, "sales_rep"); ok {
			if a, ok := v.(*apiresource.Actor); ok {
				actors = append(actors, a)
			}
		}
	}
	resourceloaders.HydrateActorNames(ctx, actors)
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

	// product, quantity_ordered, unit_price, unit_cost, and totals are expandable — populated from stashed meta only when requested.

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

// buildLineTotals derives the per-line monetary totals, pairing each downstream
// stage's amount with its completion (stage quantity / ordered quantity).
func buildLineTotals(info *pb.SalesOrderLineInfo) *apiresource.SalesOrderTotals {
	price := parseDecimal(info.UnitPriceValue)
	ordered := parseDecimal(info.QuantityValue)
	var picked, packed, invoiced decimal.Decimal
	if info.QuantityPickedValue != nil {
		picked = parseDecimal(*info.QuantityPickedValue)
	}
	if info.QuantityPackedValue != nil {
		packed = parseDecimal(*info.QuantityPackedValue)
	}
	if info.QuantityInvoicedValue != nil {
		invoiced = parseDecimal(*info.QuantityInvoicedValue)
	}
	return &apiresource.SalesOrderTotals{
		Object:   constants.ObjectTypeSalesOrderTotals,
		Ordered:  price.Mul(ordered).String(),
		Picked:   apiresource.SalesOrderStageTotal{Object: constants.ObjectTypeSalesOrderStageTotal, Amount: price.Mul(picked).String(), Completion: completionFraction(picked, ordered)},
		Packed:   apiresource.SalesOrderStageTotal{Object: constants.ObjectTypeSalesOrderStageTotal, Amount: price.Mul(packed).String(), Completion: completionFraction(packed, ordered)},
		Invoiced: apiresource.SalesOrderStageTotal{Object: constants.ObjectTypeSalesOrderStageTotal, Amount: price.Mul(invoiced).String(), Completion: completionFraction(invoiced, ordered)},
	}
}

// completionFraction returns part/whole as a float in [0,1], or 0 when whole is
// zero. It mirrors the order-level completion aggregate for a single line.
func completionFraction(part, whole decimal.Decimal) float64 {
	if whole.IsZero() {
		return 0
	}
	f := part.Div(whole).InexactFloat64()
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}

// stashSalesOrderLineMeta stashes a line's expandable sub-resources
// (quantity_ordered, unit_price, unit_cost, totals) so the include resolver can
// populate them when requested.
func stashSalesOrderLineMeta(meta *resourcekit.LoadMeta, info *pb.SalesOrderLineInfo, line *apiresource.SalesOrderLine) {
	if info.ProductId != nil {
		meta.Set(constants.ObjectTypeSalesOrderLine, line.ID, "product_id", *info.ProductId)
	}
	// Quantity + its unit. quantity_ordered.unit is expandable: the proto carries only the
	// unit FK, so stash it on the Quantity for LoadUnits to hydrate on ?include=...unit.
	qty := buildLineQuantityOrdered(info)
	meta.Set(constants.ObjectTypeSalesOrderLine, line.ID, "quantity_ordered", qty)
	if info.QuantityUnitId != "" {
		meta.Set(constants.ObjectTypeQuantity, qty.ID, "unit_id", info.QuantityUnitId)
	}

	// Unit price + its numerator/denominator units (unit_price.numerator_unit, ...).
	unitPrice := buildLineUnitPrice(info, line.CreatedAt, line.UpdatedAt)
	meta.Set(constants.ObjectTypeSalesOrderLine, line.ID, "unit_price", unitPrice)
	if info.UnitPriceNumeratorUnitId != "" {
		meta.Set(constants.ObjectTypeRate, unitPrice.ID, "numerator_unit_id", info.UnitPriceNumeratorUnitId)
	}
	if info.UnitPriceDenominatorUnitId != "" {
		meta.Set(constants.ObjectTypeRate, unitPrice.ID, "denominator_unit_id", info.UnitPriceDenominatorUnitId)
	}

	if unitCost := buildLineUnitCost(info, line.CreatedAt, line.UpdatedAt); unitCost != nil {
		meta.Set(constants.ObjectTypeSalesOrderLine, line.ID, "unit_cost", unitCost)
		if info.UnitCostNumeratorUnitId != nil && *info.UnitCostNumeratorUnitId != "" {
			meta.Set(constants.ObjectTypeRate, unitCost.ID, "numerator_unit_id", *info.UnitCostNumeratorUnitId)
		}
		if info.UnitCostDenominatorUnitId != nil && *info.UnitCostDenominatorUnitId != "" {
			meta.Set(constants.ObjectTypeRate, unitCost.ID, "denominator_unit_id", *info.UnitCostDenominatorUnitId)
		}
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
