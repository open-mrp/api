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
		ObjectType: constants.ObjectTypeShippingCase,
		Load:       resourceloaders.LoadShippingCases,
		Subs: []resourcekit.SubField{
			{Key: "carrier", Populate: populateCarrierOnShippingCase},
			{Key: "shipment", Populate: populateShipmentOnShippingCase},
			{Key: "freight_amount", Populate: populateFreightAmountOnShippingCase},
			{Key: "freight_weight", Populate: populateFreightWeightOnShippingCase},
		},
	})
}

func populateCarrierOnShippingCase(ctx context.Context, parent any, _ map[string]any) {
	sc := parent.(*apiresource.ShippingCase)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeShippingCase, sc.ID, "carrier")
	if !ok {
		return
	}
	sc.Carrier = v.(*apiresource.Carrier)
}

func populateShipmentOnShippingCase(ctx context.Context, parent any, _ map[string]any) {
	sc := parent.(*apiresource.ShippingCase)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeShippingCase, sc.ID, "shipment")
	if !ok {
		return
	}
	sc.Shipment = v.(*apiresource.ShipmentDetail)
}

func populateFreightAmountOnShippingCase(ctx context.Context, parent any, _ map[string]any) {
	sc := parent.(*apiresource.ShippingCase)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeShippingCase, sc.ID, "freight_amount")
	if !ok {
		return
	}
	sc.FreightAmount = v.(*apiresource.Quantity)
}

func populateFreightWeightOnShippingCase(ctx context.Context, parent any, _ map[string]any) {
	sc := parent.(*apiresource.ShippingCase)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeShippingCase, sc.ID, "freight_weight")
	if !ok {
		return
	}
	sc.FreightWeight = v.(*apiresource.Quantity)
}
