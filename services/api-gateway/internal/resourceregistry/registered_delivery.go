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
		ObjectType: constants.ObjectTypeDelivery,
		Load:       resourceloaders.LoadDeliveries,
		Subs: []resourcekit.SubField{
			{
				// Carried inline from the delivery query, which already returns the order's id and number.
				Key:      "related",
				Populate: populateRelatedOnDelivery,
			},
			{
				Key:         "lines",
				Target:      constants.ObjectTypeDeliveryLine,
				ExtractRefs: extractLineRefsFromDelivery,
				Populate:    populateLinesOnDelivery,
			},
		},
	})
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeDeliveryLine,
		Load:       resourceloaders.LoadDeliveryLines,
		Subs: []resourcekit.SubField{
			{
				Key:         "order_line",
				Target:      constants.ObjectTypePurchaseOrderLine,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractOrderLineIDFromDeliveryLine,
				Populate:    populateOrderLineOnDeliveryLine,
			},
			{
				Key:         "item",
				Target:      constants.ObjectTypeItem,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractItemIDFromDeliveryLine,
				Populate:    populateItemOnDeliveryLine,
			},
			{
				Key:         "location",
				Target:      constants.ObjectTypeLocation,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractLocationIDFromDeliveryLine,
				Populate:    populateLocationOnDeliveryLine,
			},
			{
				// Carried inline: the delivery query returns the lot's number alongside its id.
				Key:      "lot",
				Populate: populateLotOnDeliveryLine,
			},
			{
				// Carried inline: the delivery query already returns the cost the goods were stocked at, in full. Traversed rather than loaded so `unit_cost.numerator_unit` still resolves.
				Key:         "unit_cost",
				Target:      constants.ObjectTypeRate,
				Cardinality: resourcekit.CardinalityOnePtr,
				Populate:    populateUnitCostOnDeliveryLine,
				ExtractRefs: extractUnitCostRefFromDeliveryLine,
			},
			{
				// The quantity is already on the line — this exists so a caller can reach through it to the unit.
				Key:         "quantity",
				Target:      constants.ObjectTypeQuantity,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractRefs: extractQuantityRefFromDeliveryLine,
			},
		},
	})
}

// The resolver runs Populate before gathering refs, so the rate is already on the line by the time this is called.
func extractUnitCostRefFromDeliveryLine(_ context.Context, parent any) []any {
	l := parent.(*apiresource.DeliveryLine)
	if l.UnitCost == nil {
		return nil
	}
	return []any{l.UnitCost}
}

func extractQuantityRefFromDeliveryLine(_ context.Context, parent any) []any {
	l := parent.(*apiresource.DeliveryLine)
	if l.Quantity == nil {
		return nil
	}
	return []any{l.Quantity}
}

func populateRelatedOnDelivery(ctx context.Context, parent any, _ map[string]any) {
	d := parent.(*apiresource.Delivery)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeDelivery, d.ID, "related")
	if !ok {
		return
	}
	d.Related = v.(*apiresource.DeliveryRelated)
}

func populateLinesOnDelivery(ctx context.Context, parent any, _ map[string]any) {
	d := parent.(*apiresource.Delivery)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeDelivery, d.ID, "lines")
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
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeDeliveryLine, l.ID, "lot")
	if !ok {
		return
	}
	l.Lot = v.(*apiresource.Lot)
}

func populateUnitCostOnDeliveryLine(ctx context.Context, parent any, _ map[string]any) {
	l := parent.(*apiresource.DeliveryLine)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeDeliveryLine, l.ID, "unit_cost")
	if !ok {
		return
	}
	l.UnitCost = v.(*apiresource.Rate)
}

func deliveryLineMetaID(ctx context.Context, parent any, key string) []string {
	l := parent.(*apiresource.DeliveryLine)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeDeliveryLine, l.ID, key)
	if id == "" {
		return nil
	}
	return []string{id}
}

func loadedByMetaID(ctx context.Context, lineID, key string, loaded map[string]any) (any, bool) {
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeDeliveryLine, lineID, key)
	if id == "" {
		return nil, false
	}
	v, ok := loaded[id]
	return v, ok
}

func extractOrderLineIDFromDeliveryLine(ctx context.Context, parent any) []string {
	l := parent.(*apiresource.DeliveryLine)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeDeliveryLine, l.ID, "order_line_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateOrderLineOnDeliveryLine(ctx context.Context, parent any, loaded map[string]any) {
	l := parent.(*apiresource.DeliveryLine)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeDeliveryLine, l.ID, "order_line_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		l.OrderLine = v.(*apiresource.PurchaseOrderLine)
	}
}
