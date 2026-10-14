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
		ObjectType: constants.ObjectTypeShippingCase,
		Load:       resourceloaders.LoadShippingCases,
		Subs: []include.SubField{
			{
				Key:         "carrier",
				Target:      constants.ObjectTypeCarrier,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractCarrierIDFromShippingCase,
				Populate:    populateCarrierOnShippingCase,
			},
			{
				Key:         "shipment",
				Target:      constants.ObjectTypeShipment,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractShipmentIDFromShippingCase,
				Populate:    populateShipmentOnShippingCase,
			},
			valueShapeSub(constants.ObjectTypeShippingCase, "freight_amount", func(p any) string { return p.(*apiresource.ShippingCase).ID }, func(p, v any) {
				p.(*apiresource.ShippingCase).FreightAmount = v.(*apiresource.Money)
			}),
			valueShapeSub(constants.ObjectTypeShippingCase, "freight_weight", func(p any) string { return p.(*apiresource.ShippingCase).ID }, func(p, v any) {
				p.(*apiresource.ShippingCase).FreightWeight = v.(*apiresource.Quantity)
			}),
		},
	})
}

func extractCarrierIDFromShippingCase(ctx context.Context, parent any) []string {
	sc := parent.(*apiresource.ShippingCase)
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeShippingCase, sc.ID, "carrier_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateCarrierOnShippingCase(ctx context.Context, parent any, loaded map[string]any) {
	sc := parent.(*apiresource.ShippingCase)
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeShippingCase, sc.ID, "carrier_id")
	if v, ok := loaded[id]; ok {
		sc.Carrier = v.(*apiresource.Carrier)
	}
}

func extractShipmentIDFromShippingCase(ctx context.Context, parent any) []string {
	sc := parent.(*apiresource.ShippingCase)
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeShippingCase, sc.ID, "shipment_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateShipmentOnShippingCase(ctx context.Context, parent any, loaded map[string]any) {
	sc := parent.(*apiresource.ShippingCase)
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeShippingCase, sc.ID, "shipment_id")
	if v, ok := loaded[id]; ok {
		sc.Shipment = v.(*apiresource.Shipment)
	}
}
