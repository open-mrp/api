package salesorderep

import (
	"context"
	"fmt"

	"github.com/open-mrp/apikit/include"

	productionrunep "github.com/open-mrp/api/services/api-gateway/endpoints/production-runs"
	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/fieldpb"
	"github.com/open-mrp/api/shared/metadatapb"
	"github.com/open-mrp/api/shared/pricing"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/metadata"
	"github.com/open-mrp/apikit/tracing"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type SalesOrderSvc interface {
	ListSalesOrders(ctx context.Context, req *ListSalesOrdersRequest) (*apiresource.List[apiresource.SalesOrder], *apierror.APIError)
	GetSalesOrder(ctx context.Context, req *RetrieveSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError)
	CreateSalesOrder(ctx context.Context, req *CreateSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError)
	UpdateSalesOrder(ctx context.Context, req *UpdateSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError)
	DeleteSalesOrder(ctx context.Context, req *DeleteSalesOrderRequest) (*apiresource.EmptyResource, *apierror.APIError)
	BulkDeleteSalesOrders(ctx context.Context, req *BulkDeleteSalesOrdersRequest) (*apiresource.EmptyResource, *apierror.APIError)
	IssueSalesOrder(ctx context.Context, req *IssueSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError)
	UnissueSalesOrder(ctx context.Context, req *UnissueSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError)
	CancelSalesOrder(ctx context.Context, req *CancelSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError)
	Preview9IssueSalesOrder(ctx context.Context, req *IssueSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError)
	Preview9UnissueSalesOrder(ctx context.Context, req *UnissueSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError)
	Preview9CloseSalesOrder(ctx context.Context, req *CancelSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError)
	Preview9OpenSalesOrder(ctx context.Context, req *CancelSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError)
	CheckoutSalesOrder(ctx context.Context, req *CheckoutSalesOrderRequest) (*CheckoutSalesOrderResponse, *apierror.APIError)
	QuoteSalesOrderPrices(ctx context.Context, req *QuoteSalesOrderPricesRequest) (*QuoteSalesOrderPricesResponse, *apierror.APIError)
	QuoteSalesOrderFreight(ctx context.Context, req *QuoteSalesOrderFreightRequest) (*QuoteSalesOrderFreightResponse, *apierror.APIError)
	QuoteSalesOrderCommitment(ctx context.Context, req *QuoteSalesOrderCommitmentRequest) (*QuoteSalesOrderCommitmentResponse, *apierror.APIError)
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

// enumStrings narrows typed enum filters to the plain strings the proto layer carries.
func enumStrings[T ~string](values []T) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = string(v)
	}
	return out
}

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

	productIDs := make([]string, 0, len(resp.Lines))
	for _, l := range resp.Lines {
		productIDs = append(productIDs, l.ProductId)
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
			UnitPrice: resourceloaders.ExactUnitPriceFromProto(l.UnitPrice),
		}
	}

	return &QuoteSalesOrderPricesResponse{
		Object: constants.ObjectTypeSalesOrderPriceQuote,
		Lines:  apiresource.NewList(out, apiresource.PageInfo{}),
	}, nil
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

	return &QuoteSalesOrderFreightResponse{
		Object:    constants.ObjectTypeSalesOrderFreightQuote,
		UnitPrice: resourceloaders.ExactMoneyFromProto(resp.UnitPrice),
	}, nil
}

func (m *salesOrderSvcImpl) QuoteSalesOrderCommitment(ctx context.Context, req *QuoteSalesOrderCommitmentRequest) (*QuoteSalesOrderCommitmentResponse, *apierror.APIError) {
	pbReq := &pb.QuoteSalesOrderCommitmentRequest{
		Id:                   req.SalesOrderID.Ptr(),
		BuyerAccountId:       req.BuyerAccountID.Ptr(),
		ShipToAddressId:      req.ShipToAddressID.Ptr(),
		CarrierId:            req.CarrierID.Ptr(),
		ServiceLevelId:       req.ServiceLevelID.Ptr(),
		LeadTimeOverrideDays: req.LeadTimeOverrideDays.Ptr(),
	}
	if v, ok := req.IssuedAt.Value(); ok {
		pbReq.IssuedAt = timestamppb.New(v)
	}
	if v, ok := req.PromisedAt.Value(); ok {
		pbReq.PromisedAt = timestamppb.New(v)
	}
	if v, ok := req.ShipByOverrideDate.Value(); ok {
		pbReq.ShipByOverrideDate = timestamppb.New(v)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, salesOrderEpSvcTracer, "service.sales_orders.quote_commitment", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.QuoteSalesOrderCommitmentResponse, error) {
			return m.coreClient.QuoteSalesOrderCommitment(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	commitment := &apiresource.Commitment{
		Object:                 constants.ObjectTypeCommitment,
		PromisedAt:             grpcutil.TimestampToTimePtr(pbReq.PromisedAt),
		LeadTimeOverrideDays:   pbReq.LeadTimeOverrideDays,
		ShipByOverrideDate:     grpcutil.TimestampToTimePtr(pbReq.ShipByOverrideDate),
		ShipByDate:             apiresource.ShipBy(grpcutil.TimestampToTimePtr(resp.ShipByDate), grpcutil.TimestampToTimePtr(resp.ShipByCutoffAt)),
		LeadTimeDays:           resp.LeadTimeDays,
		TransitDays:            resp.TransitDays,
		CalendarAdjustmentDays: &resp.CalendarAdjustmentDays,
		EstimatedDeliveryDate:  grpcutil.TimestampToTimePtr(resp.EstimatedDeliveryDate),
	}
	if resp.LeadTimeSourceCode != nil {
		source := constants.LeadTimeSource(*resp.LeadTimeSourceCode)
		commitment.LeadTimeSource = &source
	}
	if resp.TransitSourceCode != nil {
		source := constants.TransitSource(*resp.TransitSourceCode)
		commitment.TransitSource = &source
	}

	out := &QuoteSalesOrderCommitmentResponse{
		Object:     constants.ObjectTypeSalesOrderCommitmentQuote,
		Commitment: commitment,
		Steps:      []apiresource.CommitmentQuoteStep{},
	}
	for _, step := range resp.Steps {
		out.Steps = append(out.Steps, apiresource.CommitmentQuoteStep{
			Code:      constants.CommitmentStep(step.Code),
			Date:      grpcutil.TimestampToTime(step.Date),
			DaysMoved: step.DaysMoved,
			Detail:    stepDetailOrNil(step.Detail),
		})
	}
	return out, nil
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

	meta := include.GetLoadMeta(ctx)
	run := productionrunep.ProductionRunFromProto(resp.ProductionRun)
	productionrunep.StashProductionRunMeta(meta, resp.ProductionRun)
	return &run, nil
}

func (m *salesOrderSvcImpl) CreateSalesOrderLine(ctx context.Context, req *CreateSalesOrderLineRequest) (*apiresource.SalesOrderLine, *apierror.APIError) {
	pbReq := &pb.CreateSalesOrderLineRequest{
		Metadata:           metadata.FromCreateRequest(req.Metadata),
		SalesOrderId:       req.SalesOrderID,
		ProductId:          req.ProductID,
		ProductSku:         req.ProductSKU,
		ProductDescription: req.ProductDescription.Ptr(),
		QuantityValue:      req.Quantity.Value,
		QuantityUnitId:     req.Quantity.UnitID,
	}
	// Unit price is an optional override. Leave it empty when omitted so the core
	// service prices the line from the product (unit cost is always server-derived).
	pbReq.UnitPrice = apirequest.OptionalUnitPriceToProto(req.UnitPrice)

	resp, apiErr := grpcutil.CallRPC(ctx, salesOrderEpSvcTracer, "service.sales_orders.create_line", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateSalesOrderLineResponse, error) {
			return m.coreClient.CreateSalesOrderLine(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := salesOrderLineDetailFromProto(resp.SalesOrderLine)
	stashSalesOrderLineMeta(include.GetLoadMeta(ctx), resp.SalesOrderLine, &result)
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
		Metadata:           metadatapb.PatchToProto(req.Metadata),
		SalesOrderId:       req.SalesOrderID,
		Id:                 req.SalesOrderLineID,
		ProductSku:         req.ProductSKU.Ptr(),
		ProductDescription: fieldpb.StringClearableToProto(req.ProductDescription),
	}
	if v, ok := req.Quantity.Value(); ok {
		pbReq.QuantityValue = &v.Value
		pbReq.QuantityUnitId = &v.UnitID
	}
	pbReq.UnitPrice = apirequest.OptionalUnitPriceToProto(req.UnitPrice)
	pbReq.UnitCost = apirequest.OptionalUnitPriceToProto(req.UnitCost)

	resp, apiErr := grpcutil.CallRPC(ctx, salesOrderEpSvcTracer, "service.sales_orders.update_line", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateSalesOrderLineResponse, error) {
			return m.coreClient.UpdateSalesOrderLine(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := salesOrderLineDetailFromProto(resp.SalesOrderLine)
	stashSalesOrderLineMeta(include.GetLoadMeta(ctx), resp.SalesOrderLine, &result)
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

// salesOrderTotalsFromProto maps the order totals core computed onto the resource,
// pairing each stage amount with the order-level completion fraction. handled is
// false when the proto carries no totals (an older core that predates the field),
// telling the caller to fall back to summing the shipped lines; when handled is
// true but the totals are unavailable (a line core could not price), it returns a
// nil total so the field serializes null rather than a wrong number.
func salesOrderTotalsFromProto(info *pb.SalesOrderInfo) (*apiresource.SalesOrderTotals, bool) {
	if info.Totals == nil {
		return nil, false
	}
	if !info.Totals.Available {
		return nil, true
	}
	return &apiresource.SalesOrderTotals{
		Object:   constants.ObjectTypeSalesOrderTotals,
		Ordered:  info.Totals.Ordered,
		Picked:   apiresource.SalesOrderStageTotal{Object: constants.ObjectTypeSalesOrderStageTotal, Amount: info.Totals.Picked, Completion: info.PickedCompletion},
		Packed:   apiresource.SalesOrderStageTotal{Object: constants.ObjectTypeSalesOrderStageTotal, Amount: info.Totals.Packed, Completion: info.PackedCompletion},
		Invoiced: apiresource.SalesOrderStageTotal{Object: constants.ObjectTypeSalesOrderStageTotal, Amount: info.Totals.Invoiced, Completion: info.InvoicedCompletion},
	}, true
}

// salesOrderTotalsFromOrder derives the order's monetary totals — with per-stage
// completion progress — from its line data and the order-level fulfillment
// aggregate. The stage amounts are summed from the lines; the completion
// fractions come from the order's PickedCompletion/PackedCompletion/
// InvoicedCompletion, computed server-side over all line items. Returns nil when
// no lines are present on the proto (i.e. the `lines` include was not requested),
// since the amounts cannot be derived without them, and when a line cannot be
// priced, since a null total is better than a wrong one. Each line is rounded to
// the cent before summing, as the dashboard's calculateTotalOrdered does.
func salesOrderTotalsFromOrder(info *pb.SalesOrderInfo) *apiresource.SalesOrderTotals {
	if len(info.Lines) == 0 {
		return nil
	}

	var totalOrdered, totalPicked, totalPacked, totalInvoiced decimal.Decimal

	for _, l := range info.Lines {
		price := parseDecimal(l.UnitPriceValue)
		conv, ok := linePriceUnitConversion(l)
		if !ok {
			return nil
		}
		totalOrdered = totalOrdered.Add(pricing.LineTotal(parseDecimal(l.QuantityValue), price, conv))
		if l.QuantityPickedValue != nil {
			totalPicked = totalPicked.Add(pricing.LineTotal(parseDecimal(*l.QuantityPickedValue), price, conv))
		}
		if l.QuantityPackedValue != nil {
			totalPacked = totalPacked.Add(pricing.LineTotal(parseDecimal(*l.QuantityPackedValue), price, conv))
		}
		if l.QuantityInvoicedValue != nil {
			totalInvoiced = totalInvoiced.Add(pricing.LineTotal(parseDecimal(*l.QuantityInvoicedValue), price, conv))
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

func salesOrderLineDetailFromProto(info *pb.SalesOrderLineInfo) apiresource.SalesOrderLine {
	l := apiresource.SalesOrderLine{
		Metadata:           apiresource.MetadataFromProto(info.Metadata),
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
	return apiresource.NewQuantity(info.QuantityValue, info.QuantityUnitId)
}

// linePriceUnitConversion reads the factor that restates the line's quantities in
// its price's unit. ok is false when the line carries a conversion that cannot be
// used, in which case its money cannot be stated.
func linePriceUnitConversion(info *pb.SalesOrderLineInfo) (pricing.UnitConversion, bool) {
	if info.PricingUnavailable {
		return pricing.UnitConversion{}, false
	}
	if info.PricingQuantityRatioNumerator == "" {
		// A core-service that predates the fields; its lines price as they always did.
		return pricing.Identity, true
	}
	conv, err := pricing.ParseUnitConversion(info.PricingQuantityRatioNumerator, info.PricingQuantityRatioDenominator,
		info.PricingPriceRatioNumerator, info.PricingPriceRatioDenominator)
	return conv, err == nil
}

// buildLineTotals derives the per-line monetary totals, pairing each downstream
// stage's amount with its completion (stage quantity / ordered quantity). Each
// amount is rounded to the cent, as the dashboard's OrderLineUtils totals are.
// Returns nil when the line cannot be priced.
func buildLineTotals(info *pb.SalesOrderLineInfo) *apiresource.SalesOrderTotals {
	conv, ok := linePriceUnitConversion(info)
	if !ok {
		return nil
	}
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
		Ordered:  pricing.LineTotal(ordered, price, conv).String(),
		Picked:   apiresource.SalesOrderStageTotal{Object: constants.ObjectTypeSalesOrderStageTotal, Amount: pricing.LineTotal(picked, price, conv).String(), Completion: completionFraction(picked, ordered)},
		Packed:   apiresource.SalesOrderStageTotal{Object: constants.ObjectTypeSalesOrderStageTotal, Amount: pricing.LineTotal(packed, price, conv).String(), Completion: completionFraction(packed, ordered)},
		Invoiced: apiresource.SalesOrderStageTotal{Object: constants.ObjectTypeSalesOrderStageTotal, Amount: pricing.LineTotal(invoiced, price, conv).String(), Completion: completionFraction(invoiced, ordered)},
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
func stashSalesOrderLineMeta(meta *include.LoadMeta, info *pb.SalesOrderLineInfo, line *apiresource.SalesOrderLine) {
	if info.ProductId != nil {
		meta.Set(constants.ObjectTypeSalesOrderLine, line.ID, "product_id", *info.ProductId)
	}
	qty := buildLineQuantityOrdered(info)
	meta.Set(constants.ObjectTypeSalesOrderLine, line.ID, "quantity_ordered", qty)

	unitPrice := resourceloaders.UnitPriceFromProto(info.UnitPrice)
	meta.Set(constants.ObjectTypeSalesOrderLine, line.ID, "unit_price", unitPrice)

	if unitCost := resourceloaders.UnitPriceFromProto(info.UnitCost); unitCost != nil {
		meta.Set(constants.ObjectTypeSalesOrderLine, line.ID, "unit_cost", unitCost)
	}
	if totals := buildLineTotals(info); totals != nil {
		meta.Set(constants.ObjectTypeSalesOrderLine, line.ID, "totals", totals)
	}
}

// stepDetailOrNil keeps a rule that takes no parameter distinguishable from one whose parameter happens to be empty.
func stepDetailOrNil(detail string) *string {
	if detail == "" {
		return nil
	}
	return &detail
}
