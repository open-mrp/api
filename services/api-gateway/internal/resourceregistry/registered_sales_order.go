package resourceregistry

import (
	"context"
	"time"

	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
)

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeSalesOrder,
		Load:       resourceloaders.LoadSalesOrders,
		Subs: []resourcekit.SubField{
			{
				Key:         "customer",
				Target:      constants.ObjectTypeCustomer,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractCustomerIDFromSO,
				Populate:    populateCustomerOnSO,
			},
			{Key: "sales_rep", Populate: populateSalesRepOnSO},
			{
				Key:         "created_by",
				Target:      constants.ObjectTypeCreatedBy,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractSelfIDForCreatedBy,
				Populate:    populateCreatedByOnSO,
			},
			{Key: "bill_to_address", Populate: populateBillToAddressOnSO},
			{Key: "ship_to_address", Populate: populateShipToAddressOnSO},
			{Key: "freight", Populate: populateFreightOnSO},
			{Key: "payment_term", Populate: populatePaymentTermOnSO},
			{Key: "shipping_term", Populate: populateShippingTermOnSO},
			{Key: "order_discount", Populate: populateOrderDiscountOnSO},
			{Key: "totals", Populate: populateTotalsOnSO},
			{Key: "contacts", Populate: populateContactsOnSO},
			{
				Key:         "related.pick",
				Target:      constants.ObjectTypePick,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractPickIDFromSORelated,
				Populate:    populatePickOnSORelated,
			},
			{
				Key:         "related.production_run",
				Target:      constants.ObjectTypeProductionRun,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractProductionRunIDFromSORelated,
				Populate:    populateProductionRunOnSORelated,
			},
			{
				Key:         "related.shipments",
				Target:      constants.ObjectTypeShipment,
				Cardinality: resourcekit.CardinalityList,
				ExtractIDs:  extractShipmentIDsFromSORelated,
				Populate:    populateShipmentsOnSORelated,
			},
			{
				Key:         "related.invoices",
				Target:      constants.ObjectTypeInvoice,
				Cardinality: resourcekit.CardinalityList,
				ExtractIDs:  extractInvoiceIDsFromSORelated,
				Populate:    populateInvoicesOnSORelated,
			},
			{
				Key:         "lines",
				Target:      constants.ObjectTypeSalesOrderLine,
				ExtractRefs: extractLineRefsFromSO,
				Populate:    populateLinesOnSO,
			},
		},
	})
}

// created_by is resolved lazily from each order's create audit event (via platform-service), keyed by the order's own ID — so ExtractIDs returns the order ID and the loader registered for ObjectTypeCreatedBy fetches the creator.
func extractSelfIDForCreatedBy(_ context.Context, parent any) []string {
	so := parent.(*apiresource.SalesOrder)
	if so.ID == "" {
		return nil
	}
	return []string{so.ID}
}

func populateCreatedByOnSO(_ context.Context, parent any, loaded map[string]any) {
	so := parent.(*apiresource.SalesOrder)
	if v, ok := loaded[so.ID]; ok {
		so.CreatedBy = v.(*apiresource.CreatedBy)
	}
}

func populateSalesRepOnSO(ctx context.Context, parent any, _ map[string]any) {
	so := parent.(*apiresource.SalesOrder)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeSalesOrder, so.ID, "sales_rep")
	if !ok {
		return
	}
	so.SalesRep = v.(*apiresource.Actor)
}

func populateTotalsOnSO(ctx context.Context, parent any, _ map[string]any) {
	so := parent.(*apiresource.SalesOrder)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeSalesOrder, so.ID, "totals")
	if !ok {
		return
	}
	so.Totals = v.(*apiresource.SalesOrderTotals)
}

func populateContactsOnSO(ctx context.Context, parent any, _ map[string]any) {
	so := parent.(*apiresource.SalesOrder)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeSalesOrder, so.ID, "contacts")
	if !ok {
		return
	}
	so.Contacts = v.(*apiresource.OrderContact)
}

// ensureSORelated lazily creates the related group on first expanded member, so the group serializes to null when no related include (pick/production_run/shipments) was requested.
func ensureSORelated(so *apiresource.SalesOrder) *apiresource.SalesOrderRelated {
	if so.Related == nil {
		so.Related = &apiresource.SalesOrderRelated{Object: constants.ObjectTypeSalesOrderRelated}
	}
	return so.Related
}

// openClosedStatus maps a finished/completed timestamp to the open/closed status that picks and production runs expose (closed once done, open until then), matching their list-endpoint status filters.
func openClosedStatus(done *time.Time) string {
	if done != nil {
		return "closed"
	}
	return "open"
}

func extractPickIDFromSORelated(ctx context.Context, parent any) []string {
	so := parent.(*apiresource.SalesOrder)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeSalesOrder, so.ID, "related_pick_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populatePickOnSORelated(ctx context.Context, parent any, loaded map[string]any) {
	so := parent.(*apiresource.SalesOrder)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeSalesOrder, so.ID, "related_pick_id")
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
	ensureSORelated(so).Pick = rec
}

func extractProductionRunIDFromSORelated(ctx context.Context, parent any) []string {
	so := parent.(*apiresource.SalesOrder)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeSalesOrder, so.ID, "related_production_run_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateProductionRunOnSORelated(ctx context.Context, parent any, loaded map[string]any) {
	so := parent.(*apiresource.SalesOrder)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeSalesOrder, so.ID, "related_production_run_id")
	if id == "" {
		return
	}
	v, ok := loaded[id]
	if !ok {
		return
	}
	pr := v.(*apiresource.ProductionRun)
	rec := apiresource.NewRecord(id, constants.RecordTypeProductionRun)
	rec.Number = &pr.Number
	status := openClosedStatus(pr.CompletedAt)
	rec.Status = &status
	ensureSORelated(so).ProductionRun = rec
}

func extractShipmentIDsFromSORelated(ctx context.Context, parent any) []string {
	so := parent.(*apiresource.SalesOrder)
	ids, _ := resourcekit.GetLoadMeta(ctx).GetStrings(constants.ObjectTypeSalesOrder, so.ID, "related_shipment_ids")
	return ids
}

func populateShipmentsOnSORelated(ctx context.Context, parent any, loaded map[string]any) {
	so := parent.(*apiresource.SalesOrder)
	ids, _ := resourcekit.GetLoadMeta(ctx).GetStrings(constants.ObjectTypeSalesOrder, so.ID, "related_shipment_ids")
	if len(ids) == 0 {
		return
	}
	records := make([]apiresource.Record, 0, len(ids))
	for _, id := range ids {
		v, ok := loaded[id]
		if !ok {
			continue
		}
		s := v.(*apiresource.Shipment)
		rec := apiresource.NewRecord(id, constants.RecordTypeShipment)
		rec.Number = &s.Number
		status := string(s.Status)
		rec.Status = &status
		// Surface tracking number/URL, carrier, and ship date (stashed by the shipment
		// loader) so the sales-order detail page can preview each linked shipment without
		// expanding the full shipment resource.
		if m, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeShipment, id, "record_metadata"); ok {
			if meta, ok := m.(map[string]string); ok && len(meta) > 0 {
				rec.Metadata = meta
			}
		}
		records = append(records, *rec)
	}
	if len(records) == 0 {
		return
	}
	ensureSORelated(so).Shipments = apiresource.NewList(records, apiresource.PageInfo{})
}

func extractInvoiceIDsFromSORelated(ctx context.Context, parent any) []string {
	so := parent.(*apiresource.SalesOrder)
	ids, _ := resourcekit.GetLoadMeta(ctx).GetStrings(constants.ObjectTypeSalesOrder, so.ID, "related_invoice_ids")
	return ids
}

func populateInvoicesOnSORelated(ctx context.Context, parent any, loaded map[string]any) {
	so := parent.(*apiresource.SalesOrder)
	ids, _ := resourcekit.GetLoadMeta(ctx).GetStrings(constants.ObjectTypeSalesOrder, so.ID, "related_invoice_ids")
	if len(ids) == 0 {
		return
	}
	records := make([]apiresource.Record, 0, len(ids))
	for _, id := range ids {
		v, ok := loaded[id]
		if !ok {
			continue
		}
		inv := v.(*apiresource.Invoice)
		rec := apiresource.NewRecord(id, constants.RecordTypeInvoice)
		rec.Number = &inv.Number
		status := string(inv.PaymentStatus)
		rec.Status = &status
		records = append(records, *rec)
	}
	if len(records) == 0 {
		return
	}
	ensureSORelated(so).Invoices = apiresource.NewList(records, apiresource.PageInfo{})
}

func extractCustomerIDFromSO(ctx context.Context, parent any) []string {
	so := parent.(*apiresource.SalesOrder)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeSalesOrder, so.ID, "customer_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateCustomerOnSO(ctx context.Context, parent any, loaded map[string]any) {
	so := parent.(*apiresource.SalesOrder)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeSalesOrder, so.ID, "customer_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		so.Customer = v.(*apiresource.Customer)
	}
}

func populateBillToAddressOnSO(ctx context.Context, parent any, _ map[string]any) {
	so := parent.(*apiresource.SalesOrder)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeSalesOrder, so.ID, "bill_to_address")
	if !ok {
		return
	}
	so.BillToAddress = v.(*apiresource.Address)
}

func populateShipToAddressOnSO(ctx context.Context, parent any, _ map[string]any) {
	so := parent.(*apiresource.SalesOrder)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeSalesOrder, so.ID, "ship_to_address")
	if !ok {
		return
	}
	so.ShipToAddress = v.(*apiresource.Address)
}

func populateFreightOnSO(ctx context.Context, parent any, _ map[string]any) {
	so := parent.(*apiresource.SalesOrder)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeSalesOrder, so.ID, "freight")
	if !ok {
		return
	}
	so.Freight = v.(*apiresource.Freight)
}

func populatePaymentTermOnSO(ctx context.Context, parent any, _ map[string]any) {
	so := parent.(*apiresource.SalesOrder)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeSalesOrder, so.ID, "payment_term")
	if !ok {
		return
	}
	so.PaymentTerm = v.(*apiresource.PaymentTerm)
}

func populateShippingTermOnSO(ctx context.Context, parent any, _ map[string]any) {
	so := parent.(*apiresource.SalesOrder)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeSalesOrder, so.ID, "shipping_term")
	if !ok {
		return
	}
	so.ShippingTerm = v.(*apiresource.ShippingTerm)
}

func populateOrderDiscountOnSO(ctx context.Context, parent any, _ map[string]any) {
	so := parent.(*apiresource.SalesOrder)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeSalesOrder, so.ID, "order_discount")
	if !ok {
		return
	}
	so.OrderDiscount = v.(*apiresource.OrderDiscount)
}

func extractLineRefsFromSO(_ context.Context, parent any) []any {
	so := parent.(*apiresource.SalesOrder)
	if so.Lines == nil {
		return nil
	}
	refs := make([]any, len(so.Lines.Data))
	for i := range so.Lines.Data {
		refs[i] = &so.Lines.Data[i]
	}
	return refs
}

func populateLinesOnSO(ctx context.Context, parent any, _ map[string]any) {
	so := parent.(*apiresource.SalesOrder)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeSalesOrder, so.ID, "lines")
	if !ok {
		return
	}
	so.Lines = v.(*apiresource.List[apiresource.SalesOrderLine])
}
