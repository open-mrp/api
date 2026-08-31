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
		ObjectType: constants.ObjectTypeShippingCase,
		Load:       resourceloaders.LoadShippingCases,
		Subs: []resourcekit.SubField{
			{
				Key:         "carrier",
				Target:      constants.ObjectTypeCarrier,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractCarrierIDFromShippingCase,
				Populate:    populateCarrierOnShippingCase,
			},
			{
				Key:         "shipment",
				Target:      constants.ObjectTypeShipment,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractShipmentIDFromShippingCase,
				Populate:    populateShipmentOnShippingCase,
			},
			{Key: "freight_amount", Populate: populateFreightAmountOnShippingCase},
			{Key: "freight_weight", Populate: populateFreightWeightOnShippingCase},
		},
	})
}

func extractCarrierIDFromShippingCase(ctx context.Context, parent any) []string {
	sc := parent.(*apiresource.ShippingCase)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeShippingCase, sc.ID, "carrier_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateCarrierOnShippingCase(ctx context.Context, parent any, loaded map[string]any) {
	sc := parent.(*apiresource.ShippingCase)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeShippingCase, sc.ID, "carrier_id")
	if v, ok := loaded[id]; ok {
		sc.Carrier = v.(*apiresource.Carrier)
	}
}

func extractShipmentIDFromShippingCase(ctx context.Context, parent any) []string {
	sc := parent.(*apiresource.ShippingCase)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeShippingCase, sc.ID, "shipment_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateShipmentOnShippingCase(ctx context.Context, parent any, loaded map[string]any) {
	sc := parent.(*apiresource.ShippingCase)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeShippingCase, sc.ID, "shipment_id")
	if v, ok := loaded[id]; ok {
		sc.Shipment = v.(*apiresource.Shipment)
	}
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
