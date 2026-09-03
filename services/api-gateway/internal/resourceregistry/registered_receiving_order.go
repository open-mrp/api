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
		ObjectType: constants.ObjectTypeReceivingOrder,
		Load:       resourceloaders.LoadReceivingOrders,
		Subs: []resourcekit.SubField{
			{
				// Supplier is carried inline (prebuilt) rather than loaded: it is the seller account, which is cross-account and not resolvable via the account-scoped loader. Mirrors PurchaseOrder.
				Key:      "supplier",
				Populate: populateSupplierOnReceivingOrder,
			},
			{
				// Totals and related are computed alongside the order rather than loaded from another service, so they are carried inline and only revealed when asked for.
				Key:      "totals",
				Populate: populateTotalsOnReceivingOrder,
			},
			{
				Key:      "related",
				Populate: populateRelatedOnReceivingOrder,
			},
			{
				Key:         "lines",
				Target:      constants.ObjectTypeReceivingOrderLine,
				ExtractRefs: extractLineRefsFromReceivingOrder,
				Populate:    populateLinesOnReceivingOrder,
			},
		},
	})
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeReceivingOrderLine,
		Load:       resourceloaders.LoadReceivingOrderLines,
		Subs: []resourcekit.SubField{
			{
				Key:         "item",
				Target:      constants.ObjectTypeItem,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractItemIDFromReceivingOrderLine,
				Populate:    populateItemOnReceivingOrderLine,
			},
			{
				// Carried from the purchase order line alongside the receiving line, so there is nothing to fetch. Traversed rather than loaded so `quantity_ordered.unit` still resolves.
				Key:         "quantity_ordered",
				Target:      constants.ObjectTypeQuantity,
				Cardinality: resourcekit.CardinalityOnePtr,
				Populate:    populateQuantityOrderedOnReceivingOrderLine,
				ExtractRefs: extractQuantityOrderedRefFromReceivingOrderLine,
			},
			{
				// The quantity is already on the line — this exists so a caller can reach through it to the unit.
				Key:         "quantity",
				Target:      constants.ObjectTypeQuantity,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractRefs: extractQuantityRefFromReceivingOrderLine,
			},
		},
	})
}

// The resolver runs Populate before gathering refs, so the ordered quantity is already on the line by the time this is called.
func extractQuantityOrderedRefFromReceivingOrderLine(_ context.Context, parent any) []any {
	l := parent.(*apiresource.ReceivingOrderLine)
	if l.QuantityOrdered == nil {
		return nil
	}
	return []any{l.QuantityOrdered}
}

func extractQuantityRefFromReceivingOrderLine(_ context.Context, parent any) []any {
	l := parent.(*apiresource.ReceivingOrderLine)
	if l.Quantity == nil {
		return nil
	}
	return []any{l.Quantity}
}

func populateSupplierOnReceivingOrder(ctx context.Context, parent any, _ map[string]any) {
	ro := parent.(*apiresource.ReceivingOrder)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeReceivingOrder, ro.ID, "supplier")
	if !ok {
		return
	}
	ro.Supplier = v.(*apiresource.Supplier)
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
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeReceivingOrder, ro.ID, "lines")
	if !ok {
		return
	}
	ro.Lines = v.(*apiresource.List[apiresource.ReceivingOrderLine])
}

func populateTotalsOnReceivingOrder(ctx context.Context, parent any, _ map[string]any) {
	ro := parent.(*apiresource.ReceivingOrder)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeReceivingOrder, ro.ID, "totals")
	if !ok {
		return
	}
	ro.Totals = v.(*apiresource.ReceivingOrderTotals)
}

func populateRelatedOnReceivingOrder(ctx context.Context, parent any, _ map[string]any) {
	ro := parent.(*apiresource.ReceivingOrder)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeReceivingOrder, ro.ID, "related")
	if !ok {
		return
	}
	ro.Related = v.(*apiresource.ReceivingOrderRelated)
}

func extractItemIDFromReceivingOrderLine(ctx context.Context, parent any) []string {
	l := parent.(*apiresource.ReceivingOrderLine)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeReceivingOrderLine, l.ID, "item_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateItemOnReceivingOrderLine(ctx context.Context, parent any, loaded map[string]any) {
	l := parent.(*apiresource.ReceivingOrderLine)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeReceivingOrderLine, l.ID, "item_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		l.Item = v.(*apiresource.Item)
	}
}

func populateQuantityOrderedOnReceivingOrderLine(ctx context.Context, parent any, _ map[string]any) {
	l := parent.(*apiresource.ReceivingOrderLine)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeReceivingOrderLine, l.ID, "quantity_ordered")
	if !ok {
		return
	}
	l.QuantityOrdered = v.(*apiresource.Quantity)
}
