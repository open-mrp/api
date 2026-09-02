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
				Key:         "lot",
				Target:      constants.ObjectTypeLot,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractLotIDFromDeliveryLine,
				Populate:    populateLotOnDeliveryLine,
			},
			{
				// Carried inline: the delivery query already returns the cost the goods were stocked at.
				Key:      "unit_cost",
				Populate: populateUnitCostOnDeliveryLine,
			},
		},
	})
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

func extractLotIDFromDeliveryLine(ctx context.Context, parent any) []string {
	return deliveryLineMetaID(ctx, parent, "lot_id")
}

func populateLotOnDeliveryLine(ctx context.Context, parent any, loaded map[string]any) {
	l := parent.(*apiresource.DeliveryLine)
	if v, ok := loadedByMetaID(ctx, l.ID, "lot_id", loaded); ok {
		l.Lot = v.(*apiresource.Lot)
	}
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
