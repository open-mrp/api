package resourceregistry

import (
	"context"

	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
)

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeSalesOrder,
		Load:       resourceloaders.LoadSalesOrders,
		Subs: []resourcekit.SubField{
			{Key: "customer", Populate: populateCustomerOnSO},
			{Key: "bill_to_address", Populate: populateBillToAddressOnSO},
			{Key: "ship_to_address", Populate: populateShipToAddressOnSO},
			{Key: "carrier", Populate: populateCarrierOnSO},
			{Key: "service_level", Populate: populateServiceLevelOnSO},
			{Key: "payment_term", Populate: populatePaymentTermOnSO},
			{Key: "shipping_term", Populate: populateShippingTermOnSO},
			{Key: "order_discount", Populate: populateOrderDiscountOnSO},
			{
				Key:         "lines",
				Target:      constants.ObjectTypeSalesOrderLine,
				ExtractRefs: extractLineRefsFromSO,
				Populate:    populateLinesOnSO,
			},
		},
	})
}

func populateCustomerOnSO(ctx context.Context, parent any, _ map[string]any) {
	so := parent.(*apiresource.SalesOrderDetail)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeSalesOrder, so.ID, "customer")
	if !ok {
		return
	}
	so.Customer = v.(*apiresource.Customer)
}

func populateBillToAddressOnSO(ctx context.Context, parent any, _ map[string]any) {
	so := parent.(*apiresource.SalesOrderDetail)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeSalesOrder, so.ID, "bill_to_address")
	if !ok {
		return
	}
	so.BillToAddress = v.(*apiresource.Address)
}

func populateShipToAddressOnSO(ctx context.Context, parent any, _ map[string]any) {
	so := parent.(*apiresource.SalesOrderDetail)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeSalesOrder, so.ID, "ship_to_address")
	if !ok {
		return
	}
	so.ShipToAddress = v.(*apiresource.Address)
}

func populateCarrierOnSO(ctx context.Context, parent any, _ map[string]any) {
	so := parent.(*apiresource.SalesOrderDetail)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeSalesOrder, so.ID, "carrier")
	if !ok {
		return
	}
	so.Carrier = v.(*apiresource.Carrier)
}

func populateServiceLevelOnSO(ctx context.Context, parent any, _ map[string]any) {
	so := parent.(*apiresource.SalesOrderDetail)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeSalesOrder, so.ID, "service_level")
	if !ok {
		return
	}
	so.ServiceLevel = v.(*apiresource.ServiceLevel)
}

func populatePaymentTermOnSO(ctx context.Context, parent any, _ map[string]any) {
	so := parent.(*apiresource.SalesOrderDetail)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeSalesOrder, so.ID, "payment_term")
	if !ok {
		return
	}
	so.PaymentTerm = v.(*apiresource.PaymentTerm)
}

func populateShippingTermOnSO(ctx context.Context, parent any, _ map[string]any) {
	so := parent.(*apiresource.SalesOrderDetail)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeSalesOrder, so.ID, "shipping_term")
	if !ok {
		return
	}
	so.ShippingTerm = v.(*apiresource.ShippingTerm)
}

func populateOrderDiscountOnSO(ctx context.Context, parent any, _ map[string]any) {
	so := parent.(*apiresource.SalesOrderDetail)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeSalesOrder, so.ID, "order_discount")
	if !ok {
		return
	}
	so.OrderDiscount = v.(*apiresource.OrderDiscount)
}

func extractLineRefsFromSO(_ context.Context, parent any) []any {
	so := parent.(*apiresource.SalesOrderDetail)
	if so.Lines == nil {
		return nil
	}
	refs := make([]any, len(so.Lines.Data))
	for i := range so.Lines.Data {
		refs[i] = &so.Lines.Data[i]
	}
	return refs
}

func populateLinesOnSO(ctx context.Context, parent any, _ map[string]any) {
	so := parent.(*apiresource.SalesOrderDetail)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeSalesOrder, so.ID, "lines")
	if !ok {
		return
	}
	so.Lines = v.(*apiresource.List[apiresource.SalesOrderLineDetail])
}
