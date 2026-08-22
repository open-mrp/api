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
				Key:         "purchase_order",
				Target:      constants.ObjectTypePurchaseOrder,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractPurchaseOrderIDFromReceivingOrder,
				Populate:    populatePurchaseOrderOnReceivingOrder,
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
				// Target + ExtractRefs so the resolver recurses into the order_line (a SalesOrderLine) and resolves its nested product/item includes.
				Key:         "order_line",
				Target:      constants.ObjectTypeSalesOrderLine,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractRefs: extractOrderLineRefFromReceivingOrderLine,
				Populate:    populateOrderLineOnReceivingOrderLine,
			},
		},
	})
}

func populateSupplierOnReceivingOrder(ctx context.Context, parent any, _ map[string]any) {
	ro := parent.(*apiresource.ReceivingOrder)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeReceivingOrder, ro.ID, "supplier")
	if !ok {
		return
	}
	ro.Supplier = v.(*apiresource.Supplier)
}

func extractPurchaseOrderIDFromReceivingOrder(ctx context.Context, parent any) []string {
	ro := parent.(*apiresource.ReceivingOrder)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeReceivingOrder, ro.ID, "purchase_order_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populatePurchaseOrderOnReceivingOrder(ctx context.Context, parent any, loaded map[string]any) {
	ro := parent.(*apiresource.ReceivingOrder)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeReceivingOrder, ro.ID, "purchase_order_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		ro.PurchaseOrder = v.(*apiresource.PurchaseOrder)
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
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeReceivingOrder, ro.ID, "lines")
	if !ok {
		return
	}
	ro.Lines = v.(*apiresource.List[apiresource.ReceivingOrderLine])
}

func populateOrderLineOnReceivingOrderLine(ctx context.Context, parent any, _ map[string]any) {
	l := parent.(*apiresource.ReceivingOrderLine)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeReceivingOrderLine, l.ID, "order_line")
	if !ok {
		return
	}
	l.OrderLine = v.(*apiresource.SalesOrderLine)
}

// extractOrderLineRefFromReceivingOrderLine returns the populated order_line so the resolver recurses into it (resolving order_line.product[.item]).
func extractOrderLineRefFromReceivingOrderLine(_ context.Context, parent any) []any {
	l := parent.(*apiresource.ReceivingOrderLine)
	if l.OrderLine == nil {
		return nil
	}
	return []any{l.OrderLine}
}
