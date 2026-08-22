package resourceloaders

import (
	"context"
	"strings"
	"time"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
)

var salesOrderLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.sales_order")

// Fetches orders by ID as expandable references. Lines and customer stay unpopulated; the
// relations a nested order can expand are stashed alongside — see stashNestedSalesOrderMeta.
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
		out[so.Id] = salesOrderReferenceFromProto(so)
		stashNestedSalesOrderMeta(ctx, so)
	}
	return out, nil
}

func salesOrderReferenceFromProto(info *pb.SalesOrderInfo) *apiresource.SalesOrder {
	ackStatus := constants.AcknowledgmentStatusNotSent
	if info.IsAcknowledgmentSent {
		ackStatus = constants.AcknowledgmentStatusSent
	}
	ref := &apiresource.SalesOrder{
		ID:                          info.Id,
		Object:                      constants.ObjectTypeSalesOrder,
		Number:                      info.Number,
		CustomerPurchaseOrderNumber: info.CustomerPoNumber,
		Note:                        info.Note,
		Status:                      constants.SalesOrderStatusCode(info.StatusCode),
		Priority:                    constants.PriorityCode(info.PriorityCode),
		PaymentStatus:               salesOrderPaymentStatusFromProto(info.PaymentStatus),
		AcknowledgmentStatus:        ackStatus,
		CreatedAt:                   grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt:                   grpcutil.TimestampToTime(info.UpdatedAt),
	}
	// The pick-ticket PDF prints the promised date off the order it hangs from.
	if info.PromisedAt != nil {
		t := grpcutil.TimestampToTime(info.PromisedAt)
		ref.PromisedAt = &t
	}
	return ref
}

// Translates the caller's `sales_order.*` include keys into the names core understands, so a
// nested order fetches the relations its detail projection does not already carry.
func nestedSalesOrderIncludes(ctx context.Context) []string {
	requested := resourcekit.FilterIncludes(ctx, nestedSalesOrderIncludeKeys...)
	out := make([]string, 0, len(requested))
	for _, key := range requested {
		out = append(out, strings.TrimPrefix(key, "sales_order."))
	}
	return out
}

// Only relations gated behind an `includes` check in core belong here; everything else already
// rides the detail projection.
var nestedSalesOrderIncludeKeys = []string{"sales_order.related.shipments"}

// Defaults to unpaid for an empty or unrecognized value, so a nested order never claims a
// payment state the backend did not assert.
func salesOrderPaymentStatusFromProto(status string) constants.SalesOrderPaymentStatus {
	s := constants.SalesOrderPaymentStatus(status)
	if !s.IsValid() {
		return constants.SalesOrderPaymentStatusUnpaid
	}
	return s
}

// Records what a nested order exposes so `?include=sales_order.ship_to_address` resolves on a
// parent; narrower than the endpoint's stash, which stays separate because it serves live traffic.
func stashNestedSalesOrderMeta(ctx context.Context, info *pb.SalesOrderInfo) {
	if info == nil {
		return
	}
	meta := resourcekit.GetLoadMeta(ctx)

	// The customer is resolved by id, so stash the FK: without it order.customer stays nil on every
	// order reached as a nested include.
	if info.CustomerId != "" {
		meta.Set(constants.ObjectTypeSalesOrder, info.Id, "customer_id", info.CustomerId)
	}

	// Populated only when the caller asked for it — core skips the lookup otherwise.
	if len(info.ShipmentIds) > 0 {
		meta.Set(constants.ObjectTypeSalesOrder, info.Id, "related_shipment_ids", info.ShipmentIds)
	}

	if info.BillingAddressId != "" {
		meta.Set(constants.ObjectTypeSalesOrder, info.Id, "bill_to_address", nestedOrderAddress(
			info.BillingAddressId, info.BillToName, info.BillToStreetLine_1, info.BillToStreetLine_2,
			info.BillToLocality, info.BillToState, info.BillToPostalCode, info.BillToCountry,
			info.BillToPhone, info.BillToEmail, info.BillToIsDropShip, info.BillToGeolocationId,
			grpcutil.TimestampToTime(info.BillToCreatedAt), grpcutil.TimestampToTime(info.BillToUpdatedAt),
		))
	}
	if info.ShippingAddressId != "" {
		meta.Set(constants.ObjectTypeSalesOrder, info.Id, "ship_to_address", nestedOrderAddress(
			info.ShippingAddressId, info.ShipToName, info.ShipToStreetLine_1, info.ShipToStreetLine_2,
			info.ShipToLocality, info.ShipToState, info.ShipToPostalCode, info.ShipToCountry,
			info.ShipToPhone, info.ShipToEmail, info.ShipToIsDropShip, info.ShipToGeolocationId,
			grpcutil.TimestampToTime(info.ShipToCreatedAt), grpcutil.TimestampToTime(info.ShipToUpdatedAt),
		))
	}

	// Carrier and service level have no standalone include; both reach the client through freight.
	freight := &apiresource.Freight{Object: constants.ObjectTypeFreight}
	if info.CarrierBillingType != nil {
		bt := constants.CarrierBillingType(*info.CarrierBillingType)
		freight.BillingType = &bt
	}
	freight.BillingAccountNumber = info.CarrierBillingAccount
	if info.CarrierId != nil {
		carrier := &apiresource.Carrier{ID: *info.CarrierId, Object: constants.ObjectTypeCarrier}
		if info.CarrierName != nil {
			carrier.Name = *info.CarrierName
		}
		carrier.CustomerPortalVisibility = portalVisibility(info.CarrierIsPortalEnabled)
		if info.CarrierCreatedAt != nil {
			carrier.CreatedAt = info.CarrierCreatedAt.AsTime()
		}
		if info.CarrierUpdatedAt != nil {
			carrier.UpdatedAt = info.CarrierUpdatedAt.AsTime()
		}
		freight.Carrier = carrier
	}
	if info.ServiceLevelId != nil {
		sl := &apiresource.ServiceLevel{ID: *info.ServiceLevelId, Object: constants.ObjectTypeServiceLevel}
		if info.ServiceLevelName != nil {
			sl.Name = *info.ServiceLevelName
		}
		sl.CustomerPortalVisibility = portalVisibility(info.ServiceLevelIsPortalEnabled)
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
	meta.Set(constants.ObjectTypeSalesOrder, info.Id, "freight", freight)

	if info.PaymentTermId != nil {
		pt := &apiresource.PaymentTerm{ID: *info.PaymentTermId, Object: constants.ObjectTypePaymentTerm}
		if info.PaymentTermName != nil {
			pt.Name = *info.PaymentTermName
		}
		pt.Status = constants.PaymentTermStatusInactive
		if info.PaymentTermIsActive != nil && *info.PaymentTermIsActive {
			pt.Status = constants.PaymentTermStatusActive
		}
		if info.PaymentTermCreatedAt != nil {
			pt.CreatedAt = info.PaymentTermCreatedAt.AsTime()
		}
		if info.PaymentTermUpdatedAt != nil {
			pt.UpdatedAt = info.PaymentTermUpdatedAt.AsTime()
		}
		meta.Set(constants.ObjectTypeSalesOrder, info.Id, "payment_term", pt)
	}

	if info.ShippingTermId != nil {
		st := &apiresource.ShippingTerm{ID: *info.ShippingTermId, Object: constants.ObjectTypeShippingTerm}
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
		meta.Set(constants.ObjectTypeSalesOrder, info.Id, "shipping_term", st)
	}
}

func portalVisibility(enabled *bool) constants.CustomerPortalVisibility {
	if enabled != nil && *enabled {
		return constants.CustomerPortalVisibilityVisible
	}
	return constants.CustomerPortalVisibilityHidden
}

func nestedOrderAddress(
	id string, name, line1, line2, locality, state, postalCode, country, phone, email *string,
	isDropShip *bool, geolocationID *string, createdAt, updatedAt time.Time,
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
