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
			{Key: "related.pick", Populate: populatePickOnSORelated},
			{Key: "related.production_run", Populate: populateProductionRunOnSORelated},
			{Key: "related.shipments", Populate: populateShipmentsOnSORelated},
			{
				Key:         "lines",
				Target:      constants.ObjectTypeSalesOrderLine,
				ExtractRefs: extractLineRefsFromSO,
				Populate:    populateLinesOnSO,
			},
		},
	})
}

// created_by is resolved lazily from each order's create audit event (via
// platform-service), keyed by the order's own ID — so ExtractIDs returns the
// order ID and the loader registered for ObjectTypeCreatedBy fetches the creator.
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

// ensureSORelated lazily creates the related group on first expanded member, so
// the group serializes to null when no related include (pick/production_run/
// shipments) was requested.
func ensureSORelated(so *apiresource.SalesOrder) *apiresource.SalesOrderRelated {
	if so.Related == nil {
		so.Related = &apiresource.SalesOrderRelated{Object: constants.ObjectTypeSalesOrderRelated}
	}
	return so.Related
}

func populatePickOnSORelated(ctx context.Context, parent any, _ map[string]any) {
	so := parent.(*apiresource.SalesOrder)
	related := ensureSORelated(so)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeSalesOrder, so.ID, "related_pick")
	if !ok {
		return
	}
	related.Pick = v.(*apiresource.Record)
}

func populateProductionRunOnSORelated(ctx context.Context, parent any, _ map[string]any) {
	so := parent.(*apiresource.SalesOrder)
	related := ensureSORelated(so)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeSalesOrder, so.ID, "related_production_run")
	if !ok {
		return
	}
	related.ProductionRun = v.(*apiresource.Record)
}

func populateShipmentsOnSORelated(ctx context.Context, parent any, _ map[string]any) {
	so := parent.(*apiresource.SalesOrder)
	related := ensureSORelated(so)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeSalesOrder, so.ID, "related_shipments")
	if !ok {
		return
	}
	related.Shipments = v.(*apiresource.List[apiresource.Record])
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
