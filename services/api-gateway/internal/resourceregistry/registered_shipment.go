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
		ObjectType: constants.ObjectTypeShipment,
		Load:       resourceloaders.LoadShipments,
		Subs: []resourcekit.SubField{
			// Target + ExtractRefs (not a loader) because the lines are already stashed by the
			// shipment presenter; they exist so the resolver descends for lines.sales_order_line.
			{
				Key:         "lines",
				Target:      constants.ObjectTypeShipmentLine,
				ExtractRefs: extractLineRefsFromShipment,
				Populate:    populateLinesOnShipment,
			},
			{Key: "shipping_cases", Populate: populateShippingCasesOnShipment},
			{Key: "freight", Populate: populateFreightOnShipment},
			{Key: "related.sales_order", Target: constants.ObjectTypeSalesOrder, Cardinality: resourcekit.CardinalityOnePtr, ExtractIDs: extractSalesOrderIDFromShipment, Populate: populateSalesOrderOnShipmentRelated},
			{Key: "customer", Target: constants.ObjectTypeCustomer, Cardinality: resourcekit.CardinalityOnePtr, ExtractIDs: extractCustomerIDFromShipment, Populate: populateCustomerOnShipment},
			{Key: "shipping_address", Populate: populateShippingAddressOnShipment},
			{Key: "shipped_by", Target: constants.ObjectTypeCreatedBy, Cardinality: resourcekit.CardinalityOnePtr, ExtractIDs: extractShippedByIDFromShipment, Populate: populateShippedByOnShipment},
			{Key: "related.invoice", Target: constants.ObjectTypeInvoice, Cardinality: resourcekit.CardinalityOnePtr, ExtractIDs: extractInvoiceIDFromShipment, Populate: populateInvoiceOnShipmentRelated},
			{Key: "related.pick", Target: constants.ObjectTypePick, Cardinality: resourcekit.CardinalityOnePtr, ExtractIDs: extractPickIDFromShipment, Populate: populatePickOnShipmentRelated},
		},
	})
}

func extractLineRefsFromShipment(_ context.Context, parent any) []any {
	s := parent.(*apiresource.Shipment)
	if s.Lines == nil {
		return nil
	}
	refs := make([]any, len(s.Lines.Data))
	for i := range s.Lines.Data {
		refs[i] = &s.Lines.Data[i]
	}
	return refs
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
		s.ShippedBy = v.(*apiresource.CreatedBy)
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

func extractPickIDFromShipment(ctx context.Context, parent any) []string {
	s := parent.(*apiresource.Shipment)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeShipment, s.ID, "pick_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

// Lazily creates the related group on first expanded member, so it serializes to null when no
// related include was requested.
func ensureShipmentRelated(s *apiresource.Shipment) *apiresource.ShipmentRelated {
	if s.Related == nil {
		s.Related = &apiresource.ShipmentRelated{Object: constants.ObjectTypeShipmentRelated}
	}
	return s.Related
}

func populateSalesOrderOnShipmentRelated(ctx context.Context, parent any, loaded map[string]any) {
	s := parent.(*apiresource.Shipment)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeShipment, s.ID, "sales_order_id")
	if id == "" {
		return
	}
	v, ok := loaded[id]
	if !ok {
		return
	}
	so := v.(*apiresource.SalesOrder)
	rec := apiresource.NewRecord(id, constants.RecordTypeSalesOrder)
	rec.Number = &so.Number
	status := string(so.Status)
	rec.Status = &status
	ensureShipmentRelated(s).SalesOrder = rec
}

func populatePickOnShipmentRelated(ctx context.Context, parent any, loaded map[string]any) {
	s := parent.(*apiresource.Shipment)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeShipment, s.ID, "pick_id")
	if id == "" {
		return
	}
	v, ok := loaded[id]
	if !ok {
		return
	}
	p := v.(*apiresource.Pick)
	rec := apiresource.NewRecord(id, constants.RecordTypePick)
	rec.Number = &p.Number
	status := openClosedStatus(p.FinishedAt)
	rec.Status = &status
	ensureShipmentRelated(s).Pick = rec
}

func populateInvoiceOnShipmentRelated(ctx context.Context, parent any, loaded map[string]any) {
	s := parent.(*apiresource.Shipment)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeShipment, s.ID, "invoice_id")
	if id == "" {
		return
	}
	v, ok := loaded[id]
	if !ok {
		return
	}
	inv := v.(*apiresource.Invoice)
	rec := apiresource.NewRecord(id, constants.RecordTypeInvoice)
	rec.Number = &inv.Number
	status := string(inv.PaymentStatus)
	rec.Status = &status
	ensureShipmentRelated(s).Invoice = rec
}
