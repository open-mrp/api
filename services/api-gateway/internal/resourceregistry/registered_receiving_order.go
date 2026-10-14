package resourceregistry

import (
	"context"
	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeReceivingOrder,
		Load:       resourceloaders.LoadReceivingOrders,
		Subs: []include.SubField{
			{
				Key:         "supplier",
				Target:      constants.ObjectTypeSupplier,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractSupplierIDFromReceivingOrder,
				Populate:    populateSupplierOnReceivingOrder,
			},
			{
				// Totals and related are computed alongside the order rather than loaded from another service, so they are carried inline and only revealed when asked for.
				Key:      "totals",
				Populate: populateTotalsOnReceivingOrder,
			},
			// `related` is expandable and so is each reference on it: the bare key reveals the object,
			// the child keys fill in one reference each. All carried inline from the order's own query.
			{Key: "related", Populate: populateRelatedOnReceivingOrder},
			{Key: "related.purchase_order", Populate: populatePurchaseOrderOnRORelated},
			{Key: "related.deliveries", Populate: populateDeliveriesOnRORelated},
			{
				Key:         "lines",
				Target:      constants.ObjectTypeReceivingOrderLine,
				ExtractRefs: extractLineRefsFromReceivingOrder,
				Populate:    populateLinesOnReceivingOrder,
			},
		},
	})
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeReceivingOrderLine,
		Load:       resourceloaders.LoadReceivingOrderLines,
		Subs: []include.SubField{
			{
				Key:         "item",
				Target:      constants.ObjectTypeItem,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractItemIDFromReceivingOrderLine,
				Populate:    populateItemOnReceivingOrderLine,
			},
			valueShapeSub(constants.ObjectTypeReceivingOrderLine, "quantity_ordered", func(p any) string { return p.(*apiresource.ReceivingOrderLine).ID }, func(p, v any) {
				p.(*apiresource.ReceivingOrderLine).QuantityOrdered = v.(*apiresource.Quantity)
			}),
			{
				Key:         "order_line",
				Target:      constants.ObjectTypePurchaseOrderLine,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractOrderLineIDFromReceivingOrderLine,
				Populate:    populateOrderLineOnReceivingOrderLine,
			},
		},
	})
}

func extractSupplierIDFromReceivingOrder(ctx context.Context, parent any) []string {
	ro := parent.(*apiresource.ReceivingOrder)
	id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeReceivingOrder, ro.ID, "supplier_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateSupplierOnReceivingOrder(ctx context.Context, parent any, loaded map[string]any) {
	ro := parent.(*apiresource.ReceivingOrder)
	id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeReceivingOrder, ro.ID, "supplier_id")
	if v, ok := loaded[id]; ok {
		ro.Supplier = v.(*apiresource.Supplier)
	}
}

func extractLineRefsFromReceivingOrder(_ context.Context, parent any) []any {
	ro := parent.(*apiresource.ReceivingOrder)
	if ro.Lines == nil {
		return nil
	}
	refs := make([]any, len(ro.Lines.Data))
	for i := range ro.Lines.Data {
		refs[i] = &ro.Lines.Data[i]
	}
	return refs
}

func populateLinesOnReceivingOrder(ctx context.Context, parent any, _ map[string]any) {
	ro := parent.(*apiresource.ReceivingOrder)
	v, ok := include.GetLoadMeta(ctx).
		Get(constants.ObjectTypeReceivingOrder, ro.ID, "lines")
	if !ok {
		return
	}
	ro.Lines = v.(*apiresource.List[apiresource.ReceivingOrderLine])
}

func populateTotalsOnReceivingOrder(ctx context.Context, parent any, _ map[string]any) {
	ro := parent.(*apiresource.ReceivingOrder)
	v, ok := include.GetLoadMeta(ctx).Get(constants.ObjectTypeReceivingOrder, ro.ID, "totals")
	if !ok {
		return
	}
	ro.Totals = v.(*apiresource.ReceivingOrderTotals)
}

// populateRelatedOnReceivingOrder reveals the object itself, leaving every reference on it to its own key.
func populateRelatedOnReceivingOrder(ctx context.Context, parent any, _ map[string]any) {
	ro := parent.(*apiresource.ReceivingOrder)
	if stashedRORelated(ctx, ro.ID) != nil {
		roRelated(ro)
	}
}

func populatePurchaseOrderOnRORelated(ctx context.Context, parent any, _ map[string]any) {
	ro := parent.(*apiresource.ReceivingOrder)
	if stashed := stashedRORelated(ctx, ro.ID); stashed != nil {
		roRelated(ro).PurchaseOrder = stashed.PurchaseOrder
	}
}

func populateDeliveriesOnRORelated(ctx context.Context, parent any, _ map[string]any) {
	ro := parent.(*apiresource.ReceivingOrder)
	if stashed := stashedRORelated(ctx, ro.ID); stashed != nil {
		roRelated(ro).Deliveries = stashed.Deliveries
	}
}

func stashedRORelated(ctx context.Context, orderID string) *apiresource.ReceivingOrderRelated {
	v, ok := include.GetLoadMeta(ctx).Get(constants.ObjectTypeReceivingOrder, orderID, "related")
	if !ok {
		return nil
	}
	return v.(*apiresource.ReceivingOrderRelated)
}

// roRelated returns the order's related object, creating it on first use so two independently
// requested children populate into the same one.
func roRelated(ro *apiresource.ReceivingOrder) *apiresource.ReceivingOrderRelated {
	if ro.Related == nil {
		ro.Related = &apiresource.ReceivingOrderRelated{Object: constants.ObjectTypeReceivingOrderRelated}
	}
	return ro.Related
}

func extractItemIDFromReceivingOrderLine(ctx context.Context, parent any) []string {
	l := parent.(*apiresource.ReceivingOrderLine)
	id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeReceivingOrderLine, l.ID, "item_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateItemOnReceivingOrderLine(ctx context.Context, parent any, loaded map[string]any) {
	l := parent.(*apiresource.ReceivingOrderLine)
	id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeReceivingOrderLine, l.ID, "item_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		l.Item = v.(*apiresource.Item)
	}
}

func extractOrderLineIDFromReceivingOrderLine(ctx context.Context, parent any) []string {
	l := parent.(*apiresource.ReceivingOrderLine)
	id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeReceivingOrderLine, l.ID, "order_line_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateOrderLineOnReceivingOrderLine(ctx context.Context, parent any, loaded map[string]any) {
	l := parent.(*apiresource.ReceivingOrderLine)
	id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeReceivingOrderLine, l.ID, "order_line_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		l.OrderLine = v.(*apiresource.PurchaseOrderLine)
	}
}
