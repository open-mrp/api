package resourceloaders

import (
	"context"
	"strings"
	"time"

	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/timeutil"
	"github.com/open-mrp/apikit/tracing"
	"google.golang.org/grpc"
)

var salesOrderLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.sales_order")

// Load-meta keys a sales order's presenter stashes for its includes.
const (
	SalesOrderMetaCustomerID      = "customer_id"
	SalesOrderMetaSalesRepID      = "sales_rep_id"
	SalesOrderMetaCarrierID       = "carrier_id"
	SalesOrderMetaServiceLevelID  = "service_level_id"
	SalesOrderMetaPaymentTermID   = "payment_term_id"
	SalesOrderMetaShippingTermID  = "shipping_term_id"
	SalesOrderMetaOrderDiscountID = "order_discount_id"
	SalesOrderMetaRecipients      = "email_recipients"
	SalesOrderMetaRecipientUsers  = "email_recipient_users"
	SalesOrderMetaPickID          = "related_pick_id"
	SalesOrderMetaJobIDs          = "related_job_ids"
	SalesOrderMetaShipmentIDs     = "related_shipment_ids"
	SalesOrderMetaInvoiceIDs      = "related_invoice_ids"
	SalesOrderMetaBillToAddress   = "bill_to_address"
	SalesOrderMetaShipToAddress   = "ship_to_address"
	SalesOrderMetaContacts        = "contacts"
)

// Fetches orders by ID as expandable references. Lines stay unpopulated; the relations a nested order can expand
// are stashed alongside.
func LoadSalesOrders(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, salesOrderLoaderTracer, "loader.sales_orders.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetSalesOrdersByIDsResponse, error) {
			return coreSalesClient.BatchGetSalesOrdersByIDs(ctx, &pb.BatchGetSalesOrdersByIDsRequest{
				Ids:      ids,
				Includes: nestedSalesOrderIncludes(ctx),
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := make(map[string]any, len(resp.SalesOrders))
	for _, so := range resp.SalesOrders {
		out[so.Id] = new(SalesOrderFromProto(ctx, so))
	}
	return out, nil
}

// Translates the caller's `sales_order.*` include keys into the names core understands, so a
// nested order fetches the relations its detail projection does not already carry.
func nestedSalesOrderIncludes(ctx context.Context) []string {
	requested := include.FilterIncludes(ctx, nestedSalesOrderIncludeKeys...)
	out := make([]string, 0, len(requested))
	for _, key := range requested {
		out = append(out, strings.TrimPrefix(key, "sales_order."))
	}
	return out
}

// Only relations gated behind an `includes` check in core belong here; everything else already
// rides the detail projection.
var nestedSalesOrderIncludeKeys = []string{"sales_order.related.shipments"}

// Defaults to unpaid for an empty or unrecognized value, so an order never claims a payment state the backend did
// not assert.
func salesOrderPaymentStatusFromProto(status string) constants.SalesOrderPaymentStatus {
	s := constants.SalesOrderPaymentStatus(status)
	if !s.IsValid() {
		return constants.SalesOrderPaymentStatusUnpaid
	}
	return s
}

// SalesOrderFromProto is an order as the API returns it, less its lines and totals, with what its includes load
// stashed beside it.
func SalesOrderFromProto(ctx context.Context, info *pb.SalesOrderInfo) apiresource.SalesOrder {
	ackStatus := constants.AcknowledgmentStatusNotSent
	if info.IsAcknowledgmentSent {
		ackStatus = constants.AcknowledgmentStatusSent
	}
	so := apiresource.SalesOrder{
		ID:                          info.Id,
		Object:                      constants.ObjectTypeSalesOrder,
		Number:                      info.Number,
		Status:                      constants.SalesOrderStatusCode(info.StatusCode).Status(),
		Currency:                    constants.CurrencyUSD,
		CustomerPurchaseOrderNumber: info.CustomerPoNumber,
		Instructions:                info.Instructions,
		PaymentStatus:               salesOrderPaymentStatusFromProto(info.PaymentStatus),
		AcknowledgmentStatus:        ackStatus,
		Freight:                     &apiresource.SalesOrderFreight{},
		Commitment:                  salesOrderCommitmentFromProto(info),
		LineCount:                   info.LineCount,
		IssuedAt:                    grpcutil.TimestampToTimePtr(info.IssuedAt),
		FulfilledAt:                 grpcutil.TimestampToTimePtr(info.CompletedAt),
		FirstShippedAt:              grpcutil.TimestampToTimePtr(info.FirstShipAt),
		Metadata:                    apiresource.MetadataFromProto(info.Metadata),
		CreatedAt:                   grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt:                   grpcutil.TimestampToTime(info.UpdatedAt),
	}
	if info.ExpiredAt != nil {
		so.ExpiresOn = new(timeutil.DateOf(info.ExpiredAt.AsTime()))
	}
	if info.CarrierBillingType != nil {
		so.Freight.CarrierBilling = &apiresource.CarrierBilling{
			Type:          constants.CarrierBillingType(*info.CarrierBillingType),
			AccountNumber: info.CarrierBillingAccount,
		}
	}
	so.BillTo = StashAddressSnapshot(ctx, constants.ObjectTypeSalesOrder, so.ID, "bill_to", info.BillTo)
	so.ShipTo = StashAddressSnapshot(ctx, constants.ObjectTypeSalesOrder, so.ID, "ship_to", info.ShipTo)
	stashSalesOrderRefs(ctx, info)
	return so
}

// salesOrderCommitmentFromProto is the commitment stamped on an order, with the bases it was asked for. A record
// carries the commitment it was made on, not a projection forward from it.
func salesOrderCommitmentFromProto(info *pb.SalesOrderInfo) *apiresource.SalesOrderCommitment {
	c := &apiresource.SalesOrderCommitment{
		LeadTimeOverride:       DaysQuantity(info.LeadTimeOverrideDays),
		PromisedLeadTime:       DaysQuantity(info.PromisedLeadTimeDays),
		TransitDays:            info.TransitDays,
		CalendarAdjustmentDays: info.CalendarAdjustmentDays,
		ShipByCutoffAt:         grpcutil.TimestampToTimePtr(info.ShipByCutoffAt),
		PromisedOn:             dateOf(info.PromisedAt.AsTime(), info.PromisedAt != nil),
		ShipByOverrideOn:       dateOf(info.ShipByOverrideDate.AsTime(), info.ShipByOverrideDate != nil),
		ShipByOn:               dateOf(info.ShipByDate.AsTime(), info.ShipByDate != nil),
	}
	if info.PromisedLeadTimeSource != nil {
		c.PromisedLeadTimeSource = new(constants.PromisedLeadTimeSource(*info.PromisedLeadTimeSource))
	}
	if info.CommitmentFirmness != nil {
		c.Firmness = new(constants.CommitmentFirmness(*info.CommitmentFirmness))
	}
	if info.LeadTimeSourceCode != nil {
		c.LeadTimeSource = new(constants.LeadTimeSource(*info.LeadTimeSourceCode))
	}
	if info.TransitSourceCode != nil {
		c.TransitSource = new(constants.TransitSource(*info.TransitSourceCode))
	}
	return c
}

// dateOf is t's UTC day when ok, else nil: business days travel as midnight UTC.
func dateOf(t time.Time, ok bool) *timeutil.Date {
	if !ok {
		return nil
	}
	return new(timeutil.DateOf(t.UTC()))
}

// stashSalesOrderRefs stashes the IDs of the records an order names, its email recipients and its related records,
// for the includes that load them.
func stashSalesOrderRefs(ctx context.Context, info *pb.SalesOrderInfo) {
	meta := include.GetLoadMeta(ctx)
	set := func(key string, id *string) {
		if id != nil && *id != "" {
			meta.Set(constants.ObjectTypeSalesOrder, info.Id, key, *id)
		}
	}
	set(SalesOrderMetaCustomerID, &info.CustomerId)
	set(SalesOrderMetaSalesRepID, info.SalesRepId)
	set(SalesOrderMetaCarrierID, info.CarrierId)
	set(SalesOrderMetaServiceLevelID, info.ServiceLevelId)
	set(SalesOrderMetaPaymentTermID, info.PaymentTermId)
	set(SalesOrderMetaShippingTermID, info.ShippingTermId)
	set(SalesOrderMetaOrderDiscountID, info.OrderDiscountId)
	set(SalesOrderMetaPickID, info.PickId)
	if info.SiteId != nil {
		StashSiteID(ctx, constants.ObjectTypeSalesOrder, info.Id, *info.SiteId)
	}
	if info.ProductionRunId != nil {
		meta.Set(constants.ObjectTypeSalesOrder, info.Id, SalesOrderMetaJobIDs, []string{*info.ProductionRunId})
	}
	if len(info.ShipmentIds) > 0 {
		meta.Set(constants.ObjectTypeSalesOrder, info.Id, SalesOrderMetaShipmentIDs, info.ShipmentIds)
	}
	if len(info.InvoiceIds) > 0 {
		meta.Set(constants.ObjectTypeSalesOrder, info.Id, SalesOrderMetaInvoiceIDs, info.InvoiceIds)
	}
	// The live saved addresses and the recipients' addresses by document, which preview.9 callers read.
	if info.BillingAddressId != "" {
		meta.Set(constants.ObjectTypeSalesOrder, info.Id, SalesOrderMetaBillToAddress, OrderAddress(
			info.BillingAddressId, info.BillToName, info.BillToStreetLine_1, info.BillToStreetLine_2,
			info.BillToLocality, info.BillToState, info.BillToPostalCode, info.BillToCountry,
			info.BillToPhone, info.BillToEmail, info.BillToIsDropShip,
			grpcutil.TimestampToTime(info.BillToCreatedAt), grpcutil.TimestampToTime(info.BillToUpdatedAt),
		))
	}
	if info.ShippingAddressId != "" {
		meta.Set(constants.ObjectTypeSalesOrder, info.Id, SalesOrderMetaShipToAddress, OrderAddress(
			info.ShippingAddressId, info.ShipToName, info.ShipToStreetLine_1, info.ShipToStreetLine_2,
			info.ShipToLocality, info.ShipToState, info.ShipToPostalCode, info.ShipToCountry,
			info.ShipToPhone, info.ShipToEmail, info.ShipToIsDropShip,
			grpcutil.TimestampToTime(info.ShipToCreatedAt), grpcutil.TimestampToTime(info.ShipToUpdatedAt),
		))
	}
	meta.Set(constants.ObjectTypeSalesOrder, info.Id, SalesOrderMetaContacts, &apiresource.OrderContact{
		Object:          constants.ObjectTypeOrderContact,
		Invoice:         nonNilStrings(info.InvoiceEmails),
		Acknowledgement: nonNilStrings(info.AcknowledgementEmails),
	})

	if info.HasEmailRecipients {
		recipients := make([]apiresource.EmailRecipient, len(info.EmailRecipients))
		users := make([]*apiresource.AccountUser, len(info.EmailRecipients))
		for i, r := range info.EmailRecipients {
			docs := make([]constants.RecipientDocument, len(r.Documents))
			for j, d := range r.Documents {
				docs[j] = constants.RecipientDocument(d)
			}
			recipients[i] = apiresource.EmailRecipient{Documents: docs}
			if r.AccountUser != nil {
				users[i] = AccountUserFromDetail(ctx, r.AccountUser)
			}
		}
		meta.Set(constants.ObjectTypeSalesOrder, info.Id, SalesOrderMetaRecipients, apiresource.NewList(recipients, apiresource.PageInfo{}))
		meta.Set(constants.ObjectTypeSalesOrder, info.Id, SalesOrderMetaRecipientUsers, users)
	}
}

// SalesOrderRecipientUser is the account user of an order's i-th email recipient, when they were loaded.
func SalesOrderRecipientUser(ctx context.Context, orderID string, i int) *apiresource.AccountUser {
	v, ok := include.GetLoadMeta(ctx).Get(constants.ObjectTypeSalesOrder, orderID, SalesOrderMetaRecipientUsers)
	if !ok {
		return nil
	}
	users := v.([]*apiresource.AccountUser)
	if i < 0 || i >= len(users) {
		return nil
	}
	return users[i]
}

// OrderAddress is an order's address as the order query joins it.
func OrderAddress(
	id string, name, line1, line2, locality, state, postalCode, country, phone, email *string,
	isDropShip *bool, createdAt, updatedAt time.Time,
) *apiresource.Address {
	parts := AddressParts{
		ID:         id,
		IsDropShip: isDropShip != nil && *isDropShip,
		Line1:      line1,
		Line2:      line2,
		City:       locality,
		State:      state,
		PostalCode: postalCode,
		Phone:      phone,
		Email:      email,
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
	}
	if name != nil {
		parts.Name = *name
	}
	if country != nil {
		parts.Country = *country
	}
	return NewAddress(parts)
}
