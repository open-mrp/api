package invoiceep

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
	"google.golang.org/protobuf/types/known/timestamppb"
)

type InvoiceSvc interface {
	ListInvoices(ctx context.Context, req *ListInvoicesRequest) (*apiresource.List[apiresource.Invoice], *apierror.APIError)
	GetInvoice(ctx context.Context, req *RetrieveInvoiceRequest) (*apiresource.Invoice, *apierror.APIError)
	UpdateInvoice(ctx context.Context, req *UpdateInvoiceRequest) (*apiresource.Invoice, *apierror.APIError)
	ListCustomerInvoices(ctx context.Context, req *ListCustomerInvoicesRequest) (*apiresource.List[apiresource.InvoiceForPayment], *apierror.APIError)
}

type InvoiceSvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient pb.CoreServiceClient
}

type invoiceSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var invoiceSvcTracer = tracing.GetTracer("api-gateway.endpoints.invoices.service")

var invoiceIncludes = []string{"lines", "allocations"}

func (c *InvoiceSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("invoice endpoint service: core client is required")
	}
	return nil
}

func NewInvoiceSvc(config *InvoiceSvcConfig) InvoiceSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &invoiceSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *invoiceSvcImpl) ListInvoices(ctx context.Context, req *ListInvoicesRequest) (*apiresource.List[apiresource.Invoice], *apierror.APIError) {
	pbReq := &pb.ListInvoicesRequest{
		Cursor:           req.Cursor,
		Limit:            req.Limit,
		Query:            req.Query,
		Status:           req.Status.StringPtr(),
		ItemIds:          req.ItemIDs,
		CustomerIds:      req.CustomerIDs,
		ProductLineIds:   req.ProductLineIDs,
		CustomerGroupIds: req.CustomerGroupIDs,
		SalesRepIds:      req.SalesRepIDs,
		// Ask the backend to expand lines when requested (the rest of the includes
		// are resolved gateway-side).
		Includes: resourcekit.FilterIncludes(ctx, "lines"),
	}

	if req.StartDate != nil {
		t, err := grpcutil.ParseDateString(*req.StartDate)
		if err == nil {
			pbReq.StartDate = timestamppb.New(t)
		}
	}
	if req.EndDate != nil {
		t, err := grpcutil.ParseDateString(*req.EndDate)
		if err == nil {
			pbReq.EndDate = timestamppb.New(t)
		}
	}

	resp, apiErr := grpcutil.CallRPC(ctx, invoiceSvcTracer, "service.invoices.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListInvoicesResponse, error) {
			return m.coreClient.ListInvoices(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	if resp == nil {
		return apiresource.NewList[apiresource.Invoice](nil, apiresource.PageInfo{}), nil
	}

	invoices := make([]apiresource.Invoice, len(resp.Invoices))
	for i, d := range resp.Invoices {
		invoices[i] = invoiceFromSummaryProto(ctx, d)
	}

	return apiresource.NewList(invoices, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *invoiceSvcImpl) GetInvoice(ctx context.Context, req *RetrieveInvoiceRequest) (*apiresource.Invoice, *apierror.APIError) {
	pbReq := &pb.GetInvoiceRequest{
		Id:       req.InvoiceID,
		Includes: resourcekit.FilterIncludes(ctx, invoiceIncludes...),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, invoiceSvcTracer, "service.invoices.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetInvoiceResponse, error) {
			return m.coreClient.GetInvoice(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := invoiceFromProto(ctx, resp.Invoice)
	return &result, nil
}

func (m *invoiceSvcImpl) UpdateInvoice(ctx context.Context, req *UpdateInvoiceRequest) (*apiresource.Invoice, *apierror.APIError) {
	pbReq := &pb.UpdateInvoiceRequest{
		Id:           req.InvoiceID,
		Note:         req.Note.Ptr(),
		HasBeenSent:  req.HasBeenSent.Ptr(),
		IsEdiSent:    req.IsEdiSent.Ptr(),
		IsPaidInFull: req.IsPaidInFull.Ptr(),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, invoiceSvcTracer, "service.invoices.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateInvoiceResponse, error) {
			return m.coreClient.UpdateInvoice(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := invoiceFromSummaryProto(ctx, resp.Invoice)
	return &result, nil
}

func (m *invoiceSvcImpl) ListCustomerInvoices(ctx context.Context, req *ListCustomerInvoicesRequest) (*apiresource.List[apiresource.InvoiceForPayment], *apierror.APIError) {
	pbReq := &pb.ListCustomerInvoicesRequest{
		CustomerAccountId:    req.CustomerAccountID,
		Cursor:               req.Cursor,
		Limit:                req.Limit,
		Query:                req.Query,
		IncludeChildAccounts: req.IncludeChildAccounts,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, invoiceSvcTracer, "service.invoices.list_customer", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListCustomerInvoicesResponse, error) {
			return m.coreClient.ListCustomerInvoices(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	if resp == nil {
		return apiresource.NewList[apiresource.InvoiceForPayment](nil, apiresource.PageInfo{}), nil
	}

	invoices := make([]apiresource.InvoiceForPayment, len(resp.Invoices))
	for i, d := range resp.Invoices {
		invoices[i] = invoiceForPaymentFromProto(ctx, d)
	}

	return apiresource.NewList(invoices, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

// invoicePaymentStatus derives the InvoicePaymentStatus enum from the legacy
// is_over_paid / is_paid_in_full booleans carried by core. The partially_paid
// state is reserved for a future core signal and is never produced here.
func invoicePaymentStatus(isPaidInFull, isOverPaid bool) constants.InvoicePaymentStatus {
	switch {
	case isOverPaid:
		return constants.InvoicePaymentStatusOverpaid
	case isPaidInFull:
		return constants.InvoicePaymentStatusPaid
	default:
		return constants.InvoicePaymentStatusUnpaid
	}
}

// invoiceFromSummaryProto builds an Invoice with base fields only from the list
// projection. Expandable relations (customer, order, shipment, billing_address,
// payment_term) are left nil and populated by registered loaders on ?include=;
// only the FK ids are stashed here. Never fabricate.
func invoiceFromSummaryProto(ctx context.Context, d *pb.InvoiceSummaryInfo) apiresource.Invoice {
	if d == nil {
		return apiresource.Invoice{}
	}

	inv := apiresource.Invoice{
		ID:                   d.Id,
		Object:               constants.ObjectTypeInvoice,
		Number:               d.Number,
		Note:                 d.Note,
		LineCount:            d.LineCount,
		PriorityCode:         constants.PriorityCode(d.PriorityCode),
		PaymentStatus:        invoicePaymentStatus(d.IsPaidInFull, false),
		IsEdiSent:            d.IsEdiSent,
		HasBeenSent:          d.HasBeenSent,
		TotalInvoiced:        d.TotalInvoiced,
		AcceptsInvoiceEmails: d.AcceptsInvoiceEmails,
		CustomerIsEdiEnabled: d.CustomerIsEdiEnabled,
		CreatedAt:            grpcutil.TimestampToTime(d.CreatedAt),
		UpdatedAt:            grpcutil.TimestampToTime(d.UpdatedAt),
	}

	meta := resourcekit.GetLoadMeta(ctx)
	if d.CustomerId != "" {
		meta.Set(constants.ObjectTypeInvoice, inv.ID, "customer_id", d.CustomerId)
	}
	if d.OrderId != "" {
		meta.Set(constants.ObjectTypeInvoice, inv.ID, "order_id", d.OrderId)
	}
	if d.ShipmentId != nil && *d.ShipmentId != "" {
		meta.Set(constants.ObjectTypeInvoice, inv.ID, "shipment_id", *d.ShipmentId)
	}
	if d.BillingAddressId != "" {
		meta.Set(constants.ObjectTypeInvoice, inv.ID, "billing_address_id", d.BillingAddressId)
	}
	if d.PaymentTermId != nil && *d.PaymentTermId != "" {
		meta.Set(constants.ObjectTypeInvoice, inv.ID, "payment_term_id", *d.PaymentTermId)
	}
	// Lines are populated on the summary only when the list includes them.
	if len(d.Lines) > 0 {
		lines := make([]apiresource.InvoiceLine, len(d.Lines))
		for i, l := range d.Lines {
			lines[i] = invoiceLineFromProto(ctx, l)
		}
		meta.Set(constants.ObjectTypeInvoice, inv.ID, "lines", apiresource.NewList(lines, apiresource.PageInfo{}))
	}
	return inv
}

// invoiceFromProto builds an Invoice from the full document projection and stashes
// expandable relations (FK ids for customer/order/shipment/billing_address, and the
// pre-built lines/allocations lists). Relations are left nil on the returned struct
// and populated by registered loaders on ?include=. Never fabricate.
func invoiceFromProto(ctx context.Context, d *pb.InvoiceInfo) apiresource.Invoice {
	if d == nil {
		return apiresource.Invoice{}
	}

	inv := apiresource.Invoice{
		ID:                   d.Id,
		Object:               constants.ObjectTypeInvoice,
		Number:               d.Number,
		Note:                 d.Note,
		PaymentStatus:        invoicePaymentStatus(d.IsPaidInFull, d.IsOverPaid),
		IsEdiSent:            d.IsEdiSent,
		HasBeenSent:          d.HasBeenSent,
		AcceptsInvoiceEmails: d.AcceptsInvoiceEmails,
		CreatedAt:            grpcutil.TimestampToTime(d.CreatedAt),
		UpdatedAt:            grpcutil.TimestampToTime(d.UpdatedAt),
	}

	stashInvoiceMeta(ctx, d, &inv)
	return inv
}

func stashInvoiceMeta(ctx context.Context, d *pb.InvoiceInfo, inv *apiresource.Invoice) {
	if d == nil {
		return
	}

	meta := resourcekit.GetLoadMeta(ctx)

	// order / shipment / billing_address are expandable references: stash the FK id
	// so the registered loader fetches the real resource on ?include=. Never fabricate.
	if d.OrderId != "" {
		meta.Set(constants.ObjectTypeInvoice, inv.ID, "order_id", d.OrderId)
	}
	if d.ShipmentId != nil && *d.ShipmentId != "" {
		meta.Set(constants.ObjectTypeInvoice, inv.ID, "shipment_id", *d.ShipmentId)
	}
	if d.BillingAddressId != "" {
		meta.Set(constants.ObjectTypeInvoice, inv.ID, "billing_address_id", d.BillingAddressId)
	}
	if d.CustomerId != "" {
		meta.Set(constants.ObjectTypeInvoice, inv.ID, "customer_id", d.CustomerId)
	}
	if d.PaymentTermId != nil && *d.PaymentTermId != "" {
		meta.Set(constants.ObjectTypeInvoice, inv.ID, "payment_term_id", *d.PaymentTermId)
	}

	lines := make([]apiresource.InvoiceLine, len(d.Lines))
	for i, l := range d.Lines {
		lines[i] = invoiceLineFromProto(ctx, l)
	}
	meta.Set(constants.ObjectTypeInvoice, inv.ID, "lines", apiresource.NewList(lines, apiresource.PageInfo{}))

	allocations := make([]apiresource.InvoiceAllocation, len(d.Allocations))
	for i, a := range d.Allocations {
		allocations[i] = invoiceAllocationFromProto(a)
	}
	meta.Set(constants.ObjectTypeInvoice, inv.ID, "allocations", apiresource.NewList(allocations, apiresource.PageInfo{}))
}

func invoiceLineFromProto(ctx context.Context, l *pb.InvoiceLineInfo) apiresource.InvoiceLine {
	if l == nil {
		return apiresource.InvoiceLine{}
	}

	ts := grpcutil.TimestampToTime(l.CreatedAt)

	line := apiresource.InvoiceLine{
		ID:     l.Id,
		Object: constants.ObjectTypeInvoiceLine,
		Quantity: &apiresource.Quantity{
			ID:           l.QuantityId,
			Object:       constants.ObjectTypeQuantity,
			Value:        l.QuantityValue,
			DisplayValue: apiresource.FormatDisplayValue(l.QuantityValue, l.QuantityUnitAbbreviation, ""),
			// Unit left nil: expandable, loaded with real data via ?include=; never fabricated.
		},
		UnitPrice: &apiresource.Rate{
			ID:     l.UnitPriceId,
			Object: constants.ObjectTypeRate,
			Value:  l.UnitPriceValue,
			// numerator_unit / denominator_unit left nil: expandable, loaded real via ?include=.
			DisplayValue: apiresource.FormatRateDisplayValue(l.UnitPriceValue, "$", string(constants.UnitTypeCurrency), l.QuantityUnitAbbreviation),
			CreatedAt:    ts,
			UpdatedAt:    ts,
		},
		// OrderLine left nil: expandable, populated from the stashed pre-built object via ?include=lines.order_line.
		CreatedAt: ts,
		UpdatedAt: grpcutil.TimestampToTime(l.UpdatedAt),
	}

	// order_line is a line-level expandable reference. There is no standalone
	// sales-order-line loader, so build a pre-built, new-shape SalesOrderLine from
	// the line's identifying proto fields and stash it for populate on ?include=.
	if l.OrderLineId != "" {
		resourcekit.GetLoadMeta(ctx).Set(constants.ObjectTypeInvoiceLine, line.ID, "order_line", buildSalesOrderLineForInvoice(l))
	}
	// Item carried inline (the order line's item) so lines.item.id resolves.
	if l.OrderLineItemId != nil && *l.OrderLineItemId != "" {
		item := &apiresource.Item{ID: *l.OrderLineItemId, Object: constants.ObjectTypeItem}
		if l.OrderLineItemSku != nil {
			item.SKU = *l.OrderLineItemSku
		}
		line.Item = item
	}
	return line
}

// buildSalesOrderLineForInvoice builds a pre-built, new-shape SalesOrderLine
// reference from the invoice line's identifying proto fields. Only the required
// base fields the proto carries are set; the expandable money/quantity fields
// (Product, QuantityOrdered, UnitPrice, UnitCost, Totals) are left nil and are
// never fabricated.
func buildSalesOrderLineForInvoice(l *pb.InvoiceLineInfo) *apiresource.SalesOrderLine {
	now := grpcutil.TimestampToTime(l.CreatedAt)
	sku := "—"
	if l.OrderLineItemSku != nil && *l.OrderLineItemSku != "" {
		sku = *l.OrderLineItemSku
	}

	return &apiresource.SalesOrderLine{
		ID:         l.OrderLineId,
		Object:     constants.ObjectTypeSalesOrderLine,
		ProductSKU: sku,
		CreatedAt:  now,
		UpdatedAt:  grpcutil.TimestampToTime(l.UpdatedAt),
	}
}

func invoiceAllocationFromProto(a *pb.InvoiceAllocationInfo) apiresource.InvoiceAllocation {
	if a == nil {
		return apiresource.InvoiceAllocation{}
	}

	return apiresource.InvoiceAllocation{
		ID:     a.Id,
		Object: constants.ObjectTypeInvoiceAllocation,
		Amount: &apiresource.Quantity{
			ID:           a.AmountId,
			Object:       constants.ObjectTypeQuantity,
			Value:        a.AmountValue,
			DisplayValue: apiresource.FormatDisplayValue(a.AmountValue, a.AmountUnitAbbreviation, string(constants.UnitTypeCurrency)),
			// Unit left nil: expandable, loaded with real data via ?include=; never fabricated.
		},
		Note:      a.Note,
		CreatedAt: grpcutil.TimestampToTime(a.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(a.UpdatedAt),
	}
}

func invoiceForPaymentFromProto(ctx context.Context, d *pb.InvoiceForPaymentInfo) apiresource.InvoiceForPayment {
	if d == nil {
		return apiresource.InvoiceForPayment{}
	}

	createdAt := grpcutil.TimestampToTime(d.CreatedAt)

	allocations := make([]apiresource.InvoiceAllocation, len(d.Allocations))
	for i, a := range d.Allocations {
		allocations[i] = invoiceAllocationFromProto(a)
	}

	inv := apiresource.InvoiceForPayment{
		ID:              d.Id,
		Object:          constants.ObjectTypeInvoiceForPayment,
		Number:          d.Number,
		CustomerPO:      d.CustomerPo,
		IsParentAccount: d.IsParentAccount,
		IsPrepaid:       d.IsPrepaid,
		InvoiceTotal:    d.InvoiceTotal,
		IsPaidInFull:    d.IsPaidInFull,
		Allocations:     apiresource.NewList(allocations, apiresource.PageInfo{}),
		CreatedAt:       createdAt,
		UpdatedAt:       grpcutil.TimestampToTime(d.UpdatedAt),
	}

	// customer, parent_account, and billing_address are expandable references:
	// left nil (null) and populated with real data by registered loaders on
	// ?include=. Never fabricate.
	meta := resourcekit.GetLoadMeta(ctx)
	if d.CustomerId != "" {
		meta.Set(constants.ObjectTypeInvoiceForPayment, inv.ID, "customer_id", d.CustomerId)
	}
	if d.ParentAccountId != nil && *d.ParentAccountId != "" {
		meta.Set(constants.ObjectTypeInvoiceForPayment, inv.ID, "parent_account_id", *d.ParentAccountId)
	}
	return inv
}
