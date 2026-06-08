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
		ObjectType: constants.ObjectTypeReceivingOrder,
		Load:       resourceloaders.LoadReceivingOrders,
		Subs: []resourcekit.SubField{
			{
				Key:         "supplier",
				Target:      constants.ObjectTypeAccount,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractSupplierIDFromReceivingOrder,
				Populate:    populateSupplierOnReceivingOrder,
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
			{Key: "order_line", Populate: populateOrderLineOnReceivingOrderLine},
		},
	})
}

func extractSupplierIDFromReceivingOrder(ctx context.Context, parent any) []string {
	ro := parent.(*apiresource.ReceivingOrder)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeReceivingOrder, ro.ID, "supplier_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateSupplierOnReceivingOrder(ctx context.Context, parent any, loaded map[string]any) {
	ro := parent.(*apiresource.ReceivingOrder)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeReceivingOrder, ro.ID, "supplier_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		ro.Supplier = v.(*apiresource.Account)
	}
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
