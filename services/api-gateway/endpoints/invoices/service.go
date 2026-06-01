package invoiceep

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
	"google.golang.org/protobuf/types/known/timestamppb"
)

type InvoiceSvc interface {
	ListInvoices(ctx context.Context, req *ListInvoicesRequest) (*apiresource.List[apiresource.InvoiceSummary], *apierror.APIError)
	GetInvoice(ctx context.Context, req *RetrieveInvoiceRequest) (*apiresource.Invoice, *apierror.APIError)
	UpdateInvoice(ctx context.Context, req *UpdateInvoiceRequest) (*apiresource.InvoiceSummary, *apierror.APIError)
	ListCustomerInvoices(ctx context.Context, req *ListCustomerInvoicesRequest) (*apiresource.List[apiresource.InvoiceForPayment], *apierror.APIError)
}

type InvoiceSvcConfig struct {
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

func (m *invoiceSvcImpl) ListInvoices(ctx context.Context, req *ListInvoicesRequest) (*apiresource.List[apiresource.InvoiceSummary], *apierror.APIError) {
	pbReq := &pb.ListInvoicesRequest{
		Cursor:           req.Cursor,
		Limit:            req.Limit,
		Query:            req.Query,
		Status:           req.Status,
		ItemIds:          req.ItemIDs,
		CustomerIds:      req.CustomerIDs,
		ProductLineIds:   req.ProductLineIDs,
		CustomerGroupIds: req.CustomerGroupIDs,
		SalesRepIds:      req.SalesRepIDs,
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
		return apiresource.NewList[apiresource.InvoiceSummary](nil, apiresource.PageInfo{}), nil
	}

	invoices := make([]apiresource.InvoiceSummary, len(resp.Invoices))
	for i, d := range resp.Invoices {
		invoices[i] = invoiceSummaryFromProto(d)
	}

	return apiresource.NewList(invoices, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *invoiceSvcImpl) GetInvoice(ctx context.Context, req *RetrieveInvoiceRequest) (*apiresource.Invoice, *apierror.APIError) {
	pbReq := &pb.GetInvoiceRequest{
		Id:       req.InvoiceID,
		Includes: invoiceIncludes,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, invoiceSvcTracer, "service.invoices.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetInvoiceResponse, error) {
			return m.coreClient.GetInvoice(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := invoiceFromProto(resp.Invoice)
	stashInvoiceMeta(ctx, resp.Invoice, &result)
	return &result, nil
}

func (m *invoiceSvcImpl) UpdateInvoice(ctx context.Context, req *UpdateInvoiceRequest) (*apiresource.InvoiceSummary, *apierror.APIError) {
	pbReq := &pb.UpdateInvoiceRequest{
		Id:           req.InvoiceID,
		Note:         req.Note,
		HasBeenSent:  req.HasBeenSent,
		IsEdiSent:    req.IsEdiSent,
		IsPaidInFull: req.IsPaidInFull,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, invoiceSvcTracer, "service.invoices.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateInvoiceResponse, error) {
			return m.coreClient.UpdateInvoice(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := invoiceSummaryFromProto(resp.Invoice)
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
		invoices[i] = invoiceForPaymentFromProto(d)
	}

	return apiresource.NewList(invoices, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func invoiceSummaryFromProto(d *pb.InvoiceSummaryInfo) apiresource.InvoiceSummary {
	if d == nil {
		return apiresource.InvoiceSummary{}
	}

	createdAt := grpcutil.TimestampToTime(d.CreatedAt)
	updatedAt := grpcutil.TimestampToTime(d.UpdatedAt)

	customer := &apiresource.Customer{
		ID:               d.CustomerId,
		Object:           constants.ObjectTypeCustomer,
		Name:             d.CustomerName,
		Number:           d.CustomerNumber,
		EDIStatus:        constants.EDIStatusDisabled,
		RelationshipType: constants.CustomerRelationshipTypeStandalone,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
	}
	if d.CustomerStatusCode != nil {
		customer.Status = constants.AccountStatusCode(*d.CustomerStatusCode)
	}
	if d.CustomerCommissionPolicy != nil {
		customer.CommissionPolicy = constants.CommissionPolicy(*d.CustomerCommissionPolicy)
	}

	summary := apiresource.InvoiceSummary{
		ID:                   d.Id,
		Object:               constants.ObjectTypeInvoiceSummary,
		Number:               d.Number,
		Note:                 d.Note,
		Customer:             customer,
		Order:                apiresource.ExpandableSalesOrderStub(d.OrderId, d.OrderNumber, createdAt),
		LineCount:            d.LineCount,
		BillingAddress:       invoiceBillingAddress(d.BillingAddressId, d.BillingAddressName, d.BillingAddressLine1, d.BillingAddressLine2, d.BillingAddressCity, d.BillingAddressState, d.BillingAddressZip, d.BillingAddressCountry, createdAt),
		PriorityCode:         constants.PriorityCode(d.PriorityCode),
		IsPaidInFull:         d.IsPaidInFull,
		IsEdiSent:            d.IsEdiSent,
		HasBeenSent:          d.HasBeenSent,
		TotalInvoiced:        d.TotalInvoiced,
		AcceptsInvoiceEmails: d.AcceptsInvoiceEmails,
		CustomerIsEdiEnabled: d.CustomerIsEdiEnabled,
		CreatedAt:            createdAt,
		UpdatedAt:            updatedAt,
	}

	if d.ShipmentId != nil {
		summary.Shipment = apiresource.ExpandableShipmentStub(*d.ShipmentId, "", createdAt)
	}

	if d.PaymentTermId != nil {
		status := constants.PaymentTermStatusInactive
		if d.PaymentTermIsActive != nil && *d.PaymentTermIsActive {
			status = constants.PaymentTermStatusActive
		}
		pt := apiresource.ExpandablePaymentTermStub(*d.PaymentTermId, "", status, createdAt)
		if d.PaymentTermName != nil {
			pt.Name = *d.PaymentTermName
		}
		summary.PaymentTerm = pt
	}

	return summary
}

func invoiceFromProto(d *pb.InvoiceInfo) apiresource.Invoice {
	if d == nil {
		return apiresource.Invoice{}
	}

	createdAt := grpcutil.TimestampToTime(d.CreatedAt)

	inv := apiresource.Invoice{
		ID:                   d.Id,
		Object:               constants.ObjectTypeInvoice,
		Number:               d.Number,
		Note:                 d.Note,
		Order:                apiresource.ExpandableSalesOrderStub(d.OrderId, d.OrderNumber, createdAt),
		BillingAddress:       invoiceBillingAddress(d.BillingAddressId, d.BillingAddressName, d.BillingAddressLine1, d.BillingAddressLine2, d.BillingAddressCity, d.BillingAddressState, d.BillingAddressZip, d.BillingAddressCountry, createdAt),
		IsPaidInFull:         d.IsPaidInFull,
		IsOverPaid:           d.IsOverPaid,
		IsEdiSent:            d.IsEdiSent,
		HasBeenSent:          d.HasBeenSent,
		AcceptsInvoiceEmails: d.AcceptsInvoiceEmails,
		CreatedAt:            createdAt,
		UpdatedAt:            grpcutil.TimestampToTime(d.UpdatedAt),
	}

	if d.ShipmentId != nil {
		shipment := apiresource.ExpandableShipmentStub(*d.ShipmentId, "", createdAt)
		if d.ShipmentNumber != nil {
			shipment.Number = *d.ShipmentNumber
		}
		inv.Shipment = shipment
	}

	return inv
}

func stashInvoiceMeta(ctx context.Context, d *pb.InvoiceInfo, inv *apiresource.Invoice) {
	if d == nil {
		return
	}

	meta := resourcekit.GetLoadMeta(ctx)

	lines := make([]apiresource.InvoiceLine, len(d.Lines))
	for i, l := range d.Lines {
		lines[i] = invoiceLineFromProto(l)
	}
	meta.Set(constants.ObjectTypeInvoice, inv.ID, "lines", apiresource.NewList(lines, apiresource.PageInfo{}))

	allocations := make([]apiresource.InvoiceAllocation, len(d.Allocations))
	for i, a := range d.Allocations {
		allocations[i] = invoiceAllocationFromProto(a)
	}
	meta.Set(constants.ObjectTypeInvoice, inv.ID, "allocations", apiresource.NewList(allocations, apiresource.PageInfo{}))
}

func invoiceLineFromProto(l *pb.InvoiceLineInfo) apiresource.InvoiceLine {
	if l == nil {
		return apiresource.InvoiceLine{}
	}

	ts := grpcutil.TimestampToTime(l.CreatedAt)

	return apiresource.InvoiceLine{
		ID:     l.Id,
		Object: constants.ObjectTypeInvoiceLine,
		Quantity: &apiresource.Quantity{
			ID:           l.QuantityId,
			Object:       constants.ObjectTypeQuantity,
			Value:        l.QuantityValue,
			DisplayValue: apiresource.FormatDisplayValue(l.QuantityValue, l.QuantityUnitAbbreviation, ""),
			Unit:         apiresource.ExpandableUnitStub(l.QuantityUnitId, "", l.QuantityUnitAbbreviation, string(constants.UnitTypeQuantity), ts),
		},
		UnitPrice: &apiresource.Rate{
			ID:              l.UnitPriceId,
			Object:          constants.ObjectTypeRate,
			Value:           l.UnitPriceValue,
			NumeratorUnit:   apiresource.ExpandableUnitStub(l.UnitPriceNumeratorUnitId, "US Dollar", "$", string(constants.UnitTypeCurrency), ts),
			DenominatorUnit: apiresource.ExpandableUnitStub(l.UnitPriceDenominatorUnitId, "", l.QuantityUnitAbbreviation, string(constants.UnitTypeQuantity), ts),
			DisplayValue:    apiresource.FormatRateDisplayValue(l.UnitPriceValue, "$", string(constants.UnitTypeCurrency), l.QuantityUnitAbbreviation),
			CreatedAt:       ts,
			UpdatedAt:       ts,
		},
		CreatedAt: ts,
		UpdatedAt: grpcutil.TimestampToTime(l.UpdatedAt),
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
			Unit:         apiresource.ExpandableUnitStub(a.AmountUnitId, "", a.AmountUnitAbbreviation, string(constants.UnitTypeCurrency), grpcutil.TimestampToTime(a.CreatedAt)),
		},
		Note:      a.Note,
		CreatedAt: grpcutil.TimestampToTime(a.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(a.UpdatedAt),
	}
}

func invoiceForPaymentFromProto(d *pb.InvoiceForPaymentInfo) apiresource.InvoiceForPayment {
	if d == nil {
		return apiresource.InvoiceForPayment{}
	}

	createdAt := grpcutil.TimestampToTime(d.CreatedAt)

	allocations := make([]apiresource.InvoiceAllocation, len(d.Allocations))
	for i, a := range d.Allocations {
		allocations[i] = invoiceAllocationFromProto(a)
	}

	inv := apiresource.InvoiceForPayment{
		ID:         d.Id,
		Object:     constants.ObjectTypeInvoiceForPayment,
		Number:     d.Number,
		CustomerPO: d.CustomerPo,
		Customer: &apiresource.Customer{
			ID:               d.CustomerId,
			Object:           constants.ObjectTypeCustomer,
			Name:             d.CustomerName,
			Number:           d.CustomerNumber,
			Status:           constants.AccountStatusCodeNormal,
			EDIStatus:        constants.EDIStatusDisabled,
			RelationshipType: constants.CustomerRelationshipTypeStandalone,
			CommissionPolicy: constants.CommissionPolicyApplied,
			CreatedAt:        createdAt,
			UpdatedAt:        grpcutil.TimestampToTime(d.UpdatedAt),
		},
		IsParentAccount: d.IsParentAccount,
		IsPrepaid:       d.IsPrepaid,
		InvoiceTotal:    d.InvoiceTotal,
		IsPaidInFull:    d.IsPaidInFull,
		Allocations:     apiresource.NewList(allocations, apiresource.PageInfo{}),
		CreatedAt:       createdAt,
		UpdatedAt:       grpcutil.TimestampToTime(d.UpdatedAt),
	}

	if d.ParentAccountId != nil {
		inv.ParentAccount = apiresource.ExpandableAccountStub(*d.ParentAccountId, "", createdAt)
	}

	if d.BillingAddressId != nil {
		inv.BillingAddress = invoiceBillingAddress(*d.BillingAddressId, d.BillingAddressName, nil, nil, nil, nil, nil, "US", createdAt)
	}

	return inv
}

func invoiceBillingAddress(id string, name, line1, line2, city, state, zip *string, country string, ts time.Time) *apiresource.Address {
	addrName := "Billing Address"
	if name != nil && *name != "" {
		addrName = *name
	}
	addr := apiresource.ExpandableAddressStub(id, addrName, country, ts)
	addr.Geolocation.StreetLine1 = line1
	addr.Geolocation.StreetLine2 = line2
	addr.Geolocation.Locality = city
	addr.Geolocation.State = state
	addr.Geolocation.PostalCode = zip
	return addr
}
