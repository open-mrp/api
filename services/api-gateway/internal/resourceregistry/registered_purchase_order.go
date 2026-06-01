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
		ObjectType: constants.ObjectTypePurchaseOrder,
		Load:       resourceloaders.LoadPurchaseOrders,
		Subs: []resourcekit.SubField{
			{Key: "supplier", Populate: populateSupplierOnPO},
			{Key: "bill_to_address", Populate: populateBillToAddressOnPO},
			{Key: "ship_to_address", Populate: populateShipToAddressOnPO},
			{Key: "carrier", Populate: populateCarrierOnPO},
			{Key: "service_level", Populate: populateServiceLevelOnPO},
			{Key: "payment_term", Populate: populatePaymentTermOnPO},
			{Key: "shipping_term", Populate: populateShippingTermOnPO},
			{Key: "receiving_order", Populate: populateReceivingOrderOnPO},
			{Key: "lines", Populate: populateLinesOnPO},
			{Key: "contacts", Populate: populateContactsOnPO},
		},
	})
}

func populateSupplierOnPO(ctx context.Context, parent any, _ map[string]any) {
	po := parent.(*apiresource.PurchaseOrderDetail)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypePurchaseOrder, po.ID, "supplier")
	if !ok {
		return
	}
	po.Supplier = v.(*apiresource.Supplier)
}

func populateBillToAddressOnPO(ctx context.Context, parent any, _ map[string]any) {
	po := parent.(*apiresource.PurchaseOrderDetail)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypePurchaseOrder, po.ID, "bill_to_address")
	if !ok {
		return
	}
	po.BillToAddress = v.(*apiresource.Address)
}

func populateShipToAddressOnPO(ctx context.Context, parent any, _ map[string]any) {
	po := parent.(*apiresource.PurchaseOrderDetail)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypePurchaseOrder, po.ID, "ship_to_address")
	if !ok {
		return
	}
	po.ShipToAddress = v.(*apiresource.Address)
}

func populateCarrierOnPO(ctx context.Context, parent any, _ map[string]any) {
	po := parent.(*apiresource.PurchaseOrderDetail)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypePurchaseOrder, po.ID, "carrier")
	if !ok {
		return
	}
	po.Carrier = v.(*apiresource.Carrier)
}

func populateServiceLevelOnPO(ctx context.Context, parent any, _ map[string]any) {
	po := parent.(*apiresource.PurchaseOrderDetail)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypePurchaseOrder, po.ID, "service_level")
	if !ok {
		return
	}
	po.ServiceLevel = v.(*apiresource.ServiceLevel)
}

func populatePaymentTermOnPO(ctx context.Context, parent any, _ map[string]any) {
	po := parent.(*apiresource.PurchaseOrderDetail)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypePurchaseOrder, po.ID, "payment_term")
	if !ok {
		return
	}
	po.PaymentTerm = v.(*apiresource.PaymentTerm)
}

func populateShippingTermOnPO(ctx context.Context, parent any, _ map[string]any) {
	po := parent.(*apiresource.PurchaseOrderDetail)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypePurchaseOrder, po.ID, "shipping_term")
	if !ok {
		return
	}
	po.ShippingTerm = v.(*apiresource.ShippingTerm)
}

func populateReceivingOrderOnPO(ctx context.Context, parent any, _ map[string]any) {
	po := parent.(*apiresource.PurchaseOrderDetail)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypePurchaseOrder, po.ID, "receiving_order")
	if !ok {
		return
	}
	po.ReceivingOrder = v.(*apiresource.ReceivingOrder)
}

func populateLinesOnPO(ctx context.Context, parent any, _ map[string]any) {
	po := parent.(*apiresource.PurchaseOrderDetail)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypePurchaseOrder, po.ID, "lines")
	if !ok {
		return
	}
	po.Lines = v.(*apiresource.List[apiresource.PurchaseOrderLineDetail])
}

func populateContactsOnPO(ctx context.Context, parent any, _ map[string]any) {
	po := parent.(*apiresource.PurchaseOrderDetail)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypePurchaseOrder, po.ID, "contacts")
	if !ok {
		return
	}
	po.Contacts = v.(*apiresource.List[apiresource.EmailContact])
}
