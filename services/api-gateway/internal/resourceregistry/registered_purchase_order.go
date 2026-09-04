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
			// `related` is expandable and so is each reference on it: the bare key reveals the object,
			// the child keys fill in one reference each. All carried inline — the order already knows
			// its receiving order and its deliveries, and a record reference is all a caller needs to
			// follow either.
			{Key: "related", Populate: populateRelatedOnPO},
			{Key: "related.receiving_order", Populate: populateReceivingOrderOnPORelated},
			{Key: "related.deliveries", Populate: populateDeliveriesOnPORelated},
			{
				Key:         "lines",
				Target:      constants.ObjectTypePurchaseOrderLine,
				Cardinality: resourcekit.CardinalityList,
				Populate:    populateLinesOnPO,
				ExtractRefs: extractLineRefsFromPO,
			},
			{Key: "contacts", Populate: populateContactsOnPO},
		},
	})
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypePurchaseOrderLine,
		Load:       resourceloaders.LoadPurchaseOrderLines,
		Subs: []resourcekit.SubField{
			{
				Key:         "item",
				Target:      constants.ObjectTypeItem,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractItemIDFromPOLine,
				Populate:    populateItemOnPOLine,
			},
			// The quantity and the two rates already ride on the line; these exist so a caller can
			// reach through them to the units they are counted in.
			{
				Key:         "quantity_ordered",
				Target:      constants.ObjectTypeQuantity,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractRefs: extractQuantityOrderedRefFromPOLine,
			},
			{
				Key:         "unit_price",
				Target:      constants.ObjectTypeRate,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractRefs: extractUnitPriceRefFromPOLine,
			},
		},
	})
}

func extractLineRefsFromPO(_ context.Context, parent any) []any {
	po := parent.(*apiresource.PurchaseOrder)
	if po.Lines == nil {
		return nil
	}
	refs := make([]any, len(po.Lines.Data))
	for i := range po.Lines.Data {
		refs[i] = &po.Lines.Data[i]
	}
	return refs
}

func extractItemIDFromPOLine(ctx context.Context, parent any) []string {
	l := parent.(*apiresource.PurchaseOrderLine)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypePurchaseOrderLine, l.ID, "item_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateItemOnPOLine(ctx context.Context, parent any, loaded map[string]any) {
	l := parent.(*apiresource.PurchaseOrderLine)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypePurchaseOrderLine, l.ID, "item_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		l.Item = v.(*apiresource.Item)
	}
}

func extractQuantityOrderedRefFromPOLine(_ context.Context, parent any) []any {
	l := parent.(*apiresource.PurchaseOrderLine)
	if l.QuantityOrdered == nil {
		return nil
	}
	return []any{l.QuantityOrdered}
}

func extractUnitPriceRefFromPOLine(_ context.Context, parent any) []any {
	l := parent.(*apiresource.PurchaseOrderLine)
	if l.UnitPrice == nil {
		return nil
	}
	return []any{l.UnitPrice}
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

// populateRelatedOnPO reveals the object itself, leaving every reference on it to its own key.
func populateRelatedOnPO(ctx context.Context, parent any, _ map[string]any) {
	po := parent.(*apiresource.PurchaseOrder)
	if stashedPORelated(ctx, po.ID) != nil {
		poRelated(po)
	}
}

func populateReceivingOrderOnPORelated(ctx context.Context, parent any, _ map[string]any) {
	po := parent.(*apiresource.PurchaseOrder)
	if stashed := stashedPORelated(ctx, po.ID); stashed != nil {
		poRelated(po).ReceivingOrder = stashed.ReceivingOrder
	}
}

func populateDeliveriesOnPORelated(ctx context.Context, parent any, _ map[string]any) {
	po := parent.(*apiresource.PurchaseOrder)
	if stashed := stashedPORelated(ctx, po.ID); stashed != nil {
		poRelated(po).Deliveries = stashed.Deliveries
	}
}

func stashedPORelated(ctx context.Context, orderID string) *apiresource.PurchaseOrderRelated {
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypePurchaseOrder, orderID, "related")
	if !ok {
		return nil
	}
	return v.(*apiresource.PurchaseOrderRelated)
}

// poRelated returns the order's related object, creating it on first use so two independently
// requested children populate into the same one.
func poRelated(po *apiresource.PurchaseOrder) *apiresource.PurchaseOrderRelated {
	if po.Related == nil {
		po.Related = &apiresource.PurchaseOrderRelated{Object: constants.ObjectTypePurchaseOrderRelated}
	}
	return po.Related
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
