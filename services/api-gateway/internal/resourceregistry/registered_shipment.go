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
			{Key: "freight", Populate: populateFreightOnShipment},
			{Key: "sales_order", Target: constants.ObjectTypeSalesOrder, Cardinality: resourcekit.CardinalityOnePtr, ExtractIDs: extractSalesOrderIDFromShipment, Populate: populateSalesOrderOnShipment},
			{Key: "customer", Target: constants.ObjectTypeCustomer, Cardinality: resourcekit.CardinalityOnePtr, ExtractIDs: extractCustomerIDFromShipment, Populate: populateCustomerOnShipment},
			{Key: "shipping_address", Populate: populateShippingAddressOnShipment},
			{Key: "shipped_by", Target: constants.ObjectTypeAccountUser, Cardinality: resourcekit.CardinalityOnePtr, ExtractIDs: extractShippedByIDFromShipment, Populate: populateShippedByOnShipment},
			{Key: "invoice", Target: constants.ObjectTypeInvoice, Cardinality: resourcekit.CardinalityOnePtr, ExtractIDs: extractInvoiceIDFromShipment, Populate: populateInvoiceOnShipment},
			{Key: "pick", Target: constants.ObjectTypePick, Cardinality: resourcekit.CardinalityOnePtr, ExtractIDs: extractPickIDFromShipment, Populate: populatePickOnShipment},
		},
	})
}

func populateLinesOnShipment(ctx context.Context, parent any, _ map[string]any) {
	s := parent.(*apiresource.Shipment)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeShipment, s.ID, "lines")
	if !ok {
		return
	}
	s.Lines = v.(*apiresource.List[apiresource.ShipmentLine])
}

func populateShippingCasesOnShipment(ctx context.Context, parent any, _ map[string]any) {
	s := parent.(*apiresource.Shipment)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeShipment, s.ID, "shipping_cases")
	if !ok {
		return
	}
	s.ShippingCases = v.(*apiresource.List[apiresource.ShippingCaseDetail])
}

func populateFreightOnShipment(ctx context.Context, parent any, _ map[string]any) {
	s := parent.(*apiresource.Shipment)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeShipment, s.ID, "freight")
	if !ok {
		return
	}
	s.Freight = v.(*apiresource.Freight)
}

func extractSalesOrderIDFromShipment(ctx context.Context, parent any) []string {
	s := parent.(*apiresource.Shipment)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeShipment, s.ID, "sales_order_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateSalesOrderOnShipment(ctx context.Context, parent any, loaded map[string]any) {
	s := parent.(*apiresource.Shipment)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeShipment, s.ID, "sales_order_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		s.SalesOrder = v.(*apiresource.SalesOrder)
	}
}

func extractCustomerIDFromShipment(ctx context.Context, parent any) []string {
	s := parent.(*apiresource.Shipment)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeShipment, s.ID, "customer_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateCustomerOnShipment(ctx context.Context, parent any, loaded map[string]any) {
	s := parent.(*apiresource.Shipment)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeShipment, s.ID, "customer_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		s.Customer = v.(*apiresource.Customer)
	}
}

func populateShippingAddressOnShipment(ctx context.Context, parent any, _ map[string]any) {
	s := parent.(*apiresource.Shipment)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeShipment, s.ID, "shipping_address")
	if !ok {
		return
	}
	s.ShippingAddress = v.(*apiresource.Address)
}

func extractShippedByIDFromShipment(ctx context.Context, parent any) []string {
	s := parent.(*apiresource.Shipment)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeShipment, s.ID, "shipped_by_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateShippedByOnShipment(ctx context.Context, parent any, loaded map[string]any) {
	s := parent.(*apiresource.Shipment)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeShipment, s.ID, "shipped_by_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		s.ShippedBy = v.(*apiresource.AccountUser)
	}
}

func extractInvoiceIDFromShipment(ctx context.Context, parent any) []string {
	s := parent.(*apiresource.Shipment)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeShipment, s.ID, "invoice_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateInvoiceOnShipment(ctx context.Context, parent any, loaded map[string]any) {
	s := parent.(*apiresource.Shipment)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeShipment, s.ID, "invoice_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		s.Invoice = v.(*apiresource.Invoice)
	}
}

func extractPickIDFromShipment(ctx context.Context, parent any) []string {
	s := parent.(*apiresource.Shipment)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeShipment, s.ID, "pick_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populatePickOnShipment(ctx context.Context, parent any, loaded map[string]any) {
	s := parent.(*apiresource.Shipment)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeShipment, s.ID, "pick_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		s.Pick = v.(*apiresource.Pick)
	}
}
