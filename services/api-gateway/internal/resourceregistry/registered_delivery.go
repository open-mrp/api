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
		ObjectType: constants.ObjectTypeDelivery,
		Load:       resourceloaders.LoadDeliveries,
		Subs: []include.SubField{
			// `related` is expandable and so is each reference on it. The bare key reveals the object;
			// the child keys fill in one reference each. A single sub attaching every reference it holds
			// would make the children unconditional, which is the same as not having made them
			// expandable at all.
			//
			// All of it is carried inline from the delivery query, which already returns each document's
			// id, number and status, so none of it needs a Target to fetch through.
			{Key: "related", Populate: populateRelatedOnDelivery},
			{Key: "related.purchase_order", Populate: populatePurchaseOrderOnDeliveryRelated},
			{Key: "related.receiving_order", Populate: populateReceivingOrderOnDeliveryRelated},
			{
				Key:         "lines",
				Target:      constants.ObjectTypeDeliveryLine,
				ExtractRefs: extractLineRefsFromDelivery,
				Populate:    populateLinesOnDelivery,
			},
		},
	})
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeDeliveryLine,
		Load:       resourceloaders.LoadDeliveryLines,
		Subs: []include.SubField{
			{
				Key:         "order_line",
				Target:      constants.ObjectTypePurchaseOrderLine,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractOrderLineIDFromDeliveryLine,
				Populate:    populateOrderLineOnDeliveryLine,
			},
			{
				Key:         "item",
				Target:      constants.ObjectTypeItem,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractItemIDFromDeliveryLine,
				Populate:    populateItemOnDeliveryLine,
			},
			{
				Key:         "location",
				Target:      constants.ObjectTypeLocation,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractLocationIDFromDeliveryLine,
				Populate:    populateLocationOnDeliveryLine,
			},
			{
				// Carried inline: the delivery query returns the lot's number alongside its id.
				Key:      "lot",
				Populate: populateLotOnDeliveryLine,
			},
			valueShapeSub(constants.ObjectTypeDeliveryLine, "unit_cost", func(p any) string { return p.(*apiresource.DeliveryLine).ID }, func(p, v any) {
				p.(*apiresource.DeliveryLine).UnitCost = v.(*apiresource.UnitPrice)
			}),
		},
	})
}

// populateRelatedOnDelivery reveals the object itself, leaving every reference on it to its own key.
func populateRelatedOnDelivery(ctx context.Context, parent any, _ map[string]any) {
	d := parent.(*apiresource.Delivery)
	if stashedDeliveryRelated(ctx, d.ID) != nil {
		deliveryRelated(d)
	}
}

func populatePurchaseOrderOnDeliveryRelated(ctx context.Context, parent any, _ map[string]any) {
	d := parent.(*apiresource.Delivery)
	stashed := stashedDeliveryRelated(ctx, d.ID)
	if stashed == nil {
		return
	}
	deliveryRelated(d).PurchaseOrder = stashed.PurchaseOrder
}

func populateReceivingOrderOnDeliveryRelated(ctx context.Context, parent any, _ map[string]any) {
	d := parent.(*apiresource.Delivery)
	stashed := stashedDeliveryRelated(ctx, d.ID)
	if stashed == nil {
		return
	}
	deliveryRelated(d).ReceivingOrder = stashed.ReceivingOrder
}

func stashedDeliveryRelated(ctx context.Context, deliveryID string) *apiresource.DeliveryRelated {
	v, ok := include.GetLoadMeta(ctx).Get(constants.ObjectTypeDelivery, deliveryID, "related")
	if !ok {
		return nil
	}
	return v.(*apiresource.DeliveryRelated)
}

// deliveryRelated returns the delivery's related object, creating it on first use so two independently
// requested children populate into the same one.
func deliveryRelated(d *apiresource.Delivery) *apiresource.DeliveryRelated {
	if d.Related == nil {
		d.Related = &apiresource.DeliveryRelated{Object: constants.ObjectTypeDeliveryRelated}
	}
	return d.Related
}

func populateLinesOnDelivery(ctx context.Context, parent any, _ map[string]any) {
	d := parent.(*apiresource.Delivery)
	v, ok := include.GetLoadMeta(ctx).Get(constants.ObjectTypeDelivery, d.ID, "lines")
	if !ok {
		return
	}
	d.Lines = v.(*apiresource.List[apiresource.DeliveryLine])
}

func extractLineRefsFromDelivery(_ context.Context, parent any) []any {
	d := parent.(*apiresource.Delivery)
	if d.Lines == nil {
		return nil
	}
	refs := make([]any, len(d.Lines.Data))
	for i := range d.Lines.Data {
		refs[i] = &d.Lines.Data[i]
	}
	return refs
}

func extractItemIDFromDeliveryLine(ctx context.Context, parent any) []string {
	return deliveryLineMetaID(ctx, parent, "item_id")
}

func populateItemOnDeliveryLine(ctx context.Context, parent any, loaded map[string]any) {
	l := parent.(*apiresource.DeliveryLine)
	if v, ok := loadedByMetaID(ctx, l.ID, "item_id", loaded); ok {
		l.Item = v.(*apiresource.Item)
	}
}

func extractLocationIDFromDeliveryLine(ctx context.Context, parent any) []string {
	return deliveryLineMetaID(ctx, parent, "location_id")
}

func populateLocationOnDeliveryLine(ctx context.Context, parent any, loaded map[string]any) {
	l := parent.(*apiresource.DeliveryLine)
	if v, ok := loadedByMetaID(ctx, l.ID, "location_id", loaded); ok {
		l.Location = v.(*apiresource.Location)
	}
}

func populateLotOnDeliveryLine(ctx context.Context, parent any, _ map[string]any) {
	l := parent.(*apiresource.DeliveryLine)
	v, ok := include.GetLoadMeta(ctx).Get(constants.ObjectTypeDeliveryLine, l.ID, "lot")
	if !ok {
		return
	}
	l.Lot = v.(*apiresource.Lot)
}

func deliveryLineMetaID(ctx context.Context, parent any, key string) []string {
	l := parent.(*apiresource.DeliveryLine)
	id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeDeliveryLine, l.ID, key)
	if id == "" {
		return nil
	}
	return []string{id}
}

func loadedByMetaID(ctx context.Context, lineID, key string, loaded map[string]any) (any, bool) {
	id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeDeliveryLine, lineID, key)
	if id == "" {
		return nil, false
	}
	v, ok := loaded[id]
	return v, ok
}

func extractOrderLineIDFromDeliveryLine(ctx context.Context, parent any) []string {
	l := parent.(*apiresource.DeliveryLine)
	id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeDeliveryLine, l.ID, "order_line_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateOrderLineOnDeliveryLine(ctx context.Context, parent any, loaded map[string]any) {
	l := parent.(*apiresource.DeliveryLine)
	id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeDeliveryLine, l.ID, "order_line_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		l.OrderLine = v.(*apiresource.PurchaseOrderLine)
	}
}
