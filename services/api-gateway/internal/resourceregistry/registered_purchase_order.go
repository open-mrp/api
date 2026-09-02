package resourceregistry

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypePurchaseOrder,
		Load:       resourceloaders.LoadPurchaseOrders,
		Subs: []resourcekit.SubField{
			{Key: "supplier", Populate: populateSupplierOnPO},
			{Key: "bill_to_address", Populate: populateBillToAddressOnPO},
			{Key: "ship_to_address", Populate: populateShipToAddressOnPO},
			{Key: "freight", Populate: populateFreightOnPO},
			{Key: "payment_term", Populate: populatePaymentTermOnPO},
			{Key: "shipping_term", Populate: populateShippingTermOnPO},
			{
				// Carried inline: the order already knows its receiving order's id, and a record reference is all a caller needs to follow it.
				Key:      "related",
				Populate: populateRelatedOnPO,
			},
			{Key: "lines", Populate: populateLinesOnPO},
			{Key: "contacts", Populate: populateContactsOnPO},
		},
	})
}

func populateSupplierOnPO(ctx context.Context, parent any, _ map[string]any) {
	po := parent.(*apiresource.PurchaseOrder)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypePurchaseOrder, po.ID, "supplier")
	if !ok {
		return
	}
	po.Supplier = v.(*apiresource.Supplier)
}

func populateBillToAddressOnPO(ctx context.Context, parent any, _ map[string]any) {
	po := parent.(*apiresource.PurchaseOrder)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypePurchaseOrder, po.ID, "bill_to_address")
	if !ok {
		return
	}
	po.BillToAddress = v.(*apiresource.Address)
}

func populateShipToAddressOnPO(ctx context.Context, parent any, _ map[string]any) {
	po := parent.(*apiresource.PurchaseOrder)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypePurchaseOrder, po.ID, "ship_to_address")
	if !ok {
		return
	}
	po.ShipToAddress = v.(*apiresource.Address)
}

func populateFreightOnPO(ctx context.Context, parent any, _ map[string]any) {
	po := parent.(*apiresource.PurchaseOrder)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypePurchaseOrder, po.ID, "freight")
	if !ok {
		return
	}
	po.Freight = v.(*apiresource.Freight)
}

func populatePaymentTermOnPO(ctx context.Context, parent any, _ map[string]any) {
	po := parent.(*apiresource.PurchaseOrder)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypePurchaseOrder, po.ID, "payment_term")
	if !ok {
		return
	}
	po.PaymentTerm = v.(*apiresource.PaymentTerm)
}

func populateShippingTermOnPO(ctx context.Context, parent any, _ map[string]any) {
	po := parent.(*apiresource.PurchaseOrder)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypePurchaseOrder, po.ID, "shipping_term")
	if !ok {
		return
	}
	po.ShippingTerm = v.(*apiresource.ShippingTerm)
}

func populateRelatedOnPO(ctx context.Context, parent any, _ map[string]any) {
	po := parent.(*apiresource.PurchaseOrder)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypePurchaseOrder, po.ID, "related")
	if !ok {
		return
	}
	po.Related = v.(*apiresource.PurchaseOrderRelated)
}

func populateLinesOnPO(ctx context.Context, parent any, _ map[string]any) {
	po := parent.(*apiresource.PurchaseOrder)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypePurchaseOrder, po.ID, "lines")
	if !ok {
		return
	}
	po.Lines = v.(*apiresource.List[apiresource.PurchaseOrderLine])
}

func populateContactsOnPO(ctx context.Context, parent any, _ map[string]any) {
	po := parent.(*apiresource.PurchaseOrder)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypePurchaseOrder, po.ID, "contacts")
	if !ok {
		return
	}
	po.Contacts = v.(*apiresource.List[apiresource.EmailContact])
}
