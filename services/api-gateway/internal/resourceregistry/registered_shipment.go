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
		ObjectType: constants.ObjectTypeShipment,
		Load:       resourceloaders.LoadShipments,
		Subs: []resourcekit.SubField{
			{Key: "lines", Populate: populateLinesOnShipment},
			{Key: "shipping_cases", Populate: populateShippingCasesOnShipment},
			{Key: "sales_order", Populate: populateSalesOrderOnShipment},
			{Key: "customer", Populate: populateCustomerOnShipment},
			{Key: "carrier", Populate: populateCarrierOnShipment},
			{Key: "service_level", Populate: populateServiceLevelOnShipment},
			{Key: "shipping_address", Populate: populateShippingAddressOnShipment},
			{Key: "shipped_by", Populate: populateShippedByOnShipment},
			{Key: "invoice", Populate: populateInvoiceOnShipment},
			{Key: "pick", Populate: populatePickOnShipment},
		},
	})
}

func populateLinesOnShipment(ctx context.Context, parent any, _ map[string]any) {
	s := parent.(*apiresource.ShipmentDetail)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeShipment, s.ID, "lines")
	if !ok {
		return
	}
	s.Lines = v.(*apiresource.List[apiresource.ShipmentLine])
}

func populateShippingCasesOnShipment(ctx context.Context, parent any, _ map[string]any) {
	s := parent.(*apiresource.ShipmentDetail)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeShipment, s.ID, "shipping_cases")
	if !ok {
		return
	}
	s.ShippingCases = v.(*apiresource.List[apiresource.ShippingCaseDetail])
}

func populateSalesOrderOnShipment(ctx context.Context, parent any, _ map[string]any) {
	s := parent.(*apiresource.ShipmentDetail)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeShipment, s.ID, "sales_order")
	if !ok {
		return
	}
	s.SalesOrder = v.(*apiresource.SalesOrderDetail)
}

func populateCustomerOnShipment(ctx context.Context, parent any, _ map[string]any) {
	s := parent.(*apiresource.ShipmentDetail)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeShipment, s.ID, "customer")
	if !ok {
		return
	}
	s.Customer = v.(*apiresource.Customer)
}

func populateCarrierOnShipment(ctx context.Context, parent any, _ map[string]any) {
	s := parent.(*apiresource.ShipmentDetail)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeShipment, s.ID, "carrier")
	if !ok {
		return
	}
	s.Carrier = v.(*apiresource.Carrier)
}

func populateServiceLevelOnShipment(ctx context.Context, parent any, _ map[string]any) {
	s := parent.(*apiresource.ShipmentDetail)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeShipment, s.ID, "service_level")
	if !ok {
		return
	}
	s.ServiceLevel = v.(*apiresource.ServiceLevel)
}

func populateShippingAddressOnShipment(ctx context.Context, parent any, _ map[string]any) {
	s := parent.(*apiresource.ShipmentDetail)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeShipment, s.ID, "shipping_address")
	if !ok {
		return
	}
	s.ShippingAddress = v.(*apiresource.Address)
}

func populateShippedByOnShipment(ctx context.Context, parent any, _ map[string]any) {
	s := parent.(*apiresource.ShipmentDetail)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeShipment, s.ID, "shipped_by")
	if !ok {
		return
	}
	s.ShippedBy = v.(*apiresource.AccountUser)
}

func populateInvoiceOnShipment(ctx context.Context, parent any, _ map[string]any) {
	s := parent.(*apiresource.ShipmentDetail)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeShipment, s.ID, "invoice")
	if !ok {
		return
	}
	s.Invoice = v.(*apiresource.Invoice)
}

func populatePickOnShipment(ctx context.Context, parent any, _ map[string]any) {
	s := parent.(*apiresource.ShipmentDetail)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeShipment, s.ID, "pick")
	if !ok {
		return
	}
	s.Pick = v.(*apiresource.PickDetail)
}
