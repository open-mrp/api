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
		ObjectType: constants.ObjectTypeDelivery,
		Load:       resourceloaders.LoadDeliveries,
		Subs: []resourcekit.SubField{
			{
				Key:         "purchase_order",
				Target:      constants.ObjectTypePurchaseOrder,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractPurchaseOrderIDFromDelivery,
				Populate:    populatePurchaseOrderOnDelivery,
			},
			{
				Key:      "lines",
				Populate: populateLinesOnDelivery,
			},
		},
	})
}

func extractPurchaseOrderIDFromDelivery(ctx context.Context, parent any) []string {
	d := parent.(*apiresource.Delivery)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeDelivery, d.ID, "purchase_order_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populatePurchaseOrderOnDelivery(ctx context.Context, parent any, loaded map[string]any) {
	d := parent.(*apiresource.Delivery)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeDelivery, d.ID, "purchase_order_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		d.PurchaseOrder = v.(*apiresource.PurchaseOrder)
	}
}

func populateLinesOnDelivery(ctx context.Context, parent any, _ map[string]any) {
	d := parent.(*apiresource.Delivery)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeDelivery, d.ID, "lines")
	if !ok {
		return
	}
	d.Lines = v.(*apiresource.List[apiresource.DeliveryLine])
}
