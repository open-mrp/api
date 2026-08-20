package resourceregistry

import (
	"context"

	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeInvoice,
		Load:       resourceloaders.LoadInvoices,
		Subs: []resourcekit.SubField{
			{
				Key:         "customer",
				Target:      constants.ObjectTypeCustomer,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractCustomerIDFromInvoice,
				Populate:    populateCustomerOnInvoice,
			},
			{
				Key:         "order",
				Target:      constants.ObjectTypeSalesOrder,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractOrderIDFromInvoice,
				Populate:    populateOrderOnInvoice,
			},
			{
				Key:         "shipment",
				Target:      constants.ObjectTypeShipment,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractShipmentIDFromInvoice,
				Populate:    populateShipmentOnInvoice,
			},
			{
				Key:         "billing_address",
				Target:      constants.ObjectTypeAddress,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractBillingAddressIDFromInvoice,
				Populate:    populateBillingAddressOnInvoice,
			},
			{
				Key:         "payment_term",
				Target:      constants.ObjectTypePaymentTerm,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractPaymentTermIDFromInvoice,
				Populate:    populatePaymentTermOnInvoice,
			},
			{
				Key:         "related.sales_order",
				Target:      constants.ObjectTypeSalesOrder,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractOrderIDFromInvoice,
				Populate:    populateOrderOnInvoiceRelated,
			},
			{
				Key:         "related.shipment",
				Target:      constants.ObjectTypeShipment,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractShipmentIDFromInvoice,
				Populate:    populateShipmentOnInvoiceRelated,
			},
			{Key: "lines", Target: constants.ObjectTypeInvoiceLine, ExtractRefs: extractLineRefsFromInvoice, Populate: populateLinesOnInvoice},
			{Key: "allocations", Populate: populateAllocationsOnInvoice},
		},
	})

	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeInvoiceLine,
		Load:       stubLoadInvoiceLines,
		Subs: []resourcekit.SubField{
			// Target + ExtractRefs (not a loader) because the line is already stashed by the invoice
			// presenter; they exist so the resolver descends into it for lines.order_line.product.
			{
				Key:         "order_line",
				Target:      constants.ObjectTypeSalesOrderLine,
				ExtractRefs: extractOrderLineRefFromInvoiceLine,
				Populate:    populateOrderLineOnInvoiceLine,
			},
		},
	})

	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeInvoiceForPayment,
		Load:       resourceloaders.LoadInvoices,
		Subs: []resourcekit.SubField{
			{
				Key:         "customer",
				Target:      constants.ObjectTypeCustomer,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractCustomerIDFromInvoiceForPayment,
				Populate:    populateCustomerOnInvoiceForPayment,
			},
			{
				Key:         "parent_account",
				Target:      constants.ObjectTypeAccount,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractParentAccountIDFromInvoiceForPayment,
				Populate:    populateParentAccountOnInvoiceForPayment,
			},
			{Key: "allocations", Populate: populateAllocationsOnInvoiceForPayment},
		},
	})
}

func extractCustomerIDFromInvoice(ctx context.Context, parent any) []string {
	inv := parent.(*apiresource.Invoice)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeInvoice, inv.ID, "customer_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateCustomerOnInvoice(ctx context.Context, parent any, loaded map[string]any) {
	inv := parent.(*apiresource.Invoice)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeInvoice, inv.ID, "customer_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		inv.Customer = v.(*apiresource.Customer)
	}
}

func extractOrderIDFromInvoice(ctx context.Context, parent any) []string {
	inv := parent.(*apiresource.Invoice)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeInvoice, inv.ID, "order_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateOrderOnInvoice(ctx context.Context, parent any, loaded map[string]any) {
	inv := parent.(*apiresource.Invoice)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeInvoice, inv.ID, "order_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		inv.Order = v.(*apiresource.SalesOrder)
	}
}

func extractShipmentIDFromInvoice(ctx context.Context, parent any) []string {
	inv := parent.(*apiresource.Invoice)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeInvoice, inv.ID, "shipment_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateShipmentOnInvoice(ctx context.Context, parent any, loaded map[string]any) {
	inv := parent.(*apiresource.Invoice)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeInvoice, inv.ID, "shipment_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		inv.Shipment = v.(*apiresource.Shipment)
	}
}

func extractBillingAddressIDFromInvoice(ctx context.Context, parent any) []string {
	inv := parent.(*apiresource.Invoice)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeInvoice, inv.ID, "billing_address_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateBillingAddressOnInvoice(ctx context.Context, parent any, loaded map[string]any) {
	inv := parent.(*apiresource.Invoice)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeInvoice, inv.ID, "billing_address_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		inv.BillingAddress = v.(*apiresource.Address)
	}
}

func extractPaymentTermIDFromInvoice(ctx context.Context, parent any) []string {
	inv := parent.(*apiresource.Invoice)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeInvoice, inv.ID, "payment_term_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populatePaymentTermOnInvoice(ctx context.Context, parent any, loaded map[string]any) {
	inv := parent.(*apiresource.Invoice)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeInvoice, inv.ID, "payment_term_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		inv.PaymentTerm = v.(*apiresource.PaymentTerm)
	}
}

func extractLineRefsFromInvoice(_ context.Context, parent any) []any {
	inv := parent.(*apiresource.Invoice)
	if inv.Lines == nil {
		return nil
	}
	refs := make([]any, len(inv.Lines.Data))
	for i := range inv.Lines.Data {
		refs[i] = &inv.Lines.Data[i]
	}
	return refs
}

func populateLinesOnInvoice(ctx context.Context, parent any, _ map[string]any) {
	inv := parent.(*apiresource.Invoice)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeInvoice, inv.ID, "lines")
	if !ok {
		return
	}
	inv.Lines = v.(*apiresource.List[apiresource.InvoiceLine])
}

func populateAllocationsOnInvoice(ctx context.Context, parent any, _ map[string]any) {
	inv := parent.(*apiresource.Invoice)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeInvoice, inv.ID, "allocations")
	if !ok {
		return
	}
	inv.Allocations = v.(*apiresource.List[apiresource.InvoiceAllocation])
}

// stubLoadInvoiceLines is a no-op loader: invoice lines are always carried inline on the parent invoice; the line definition exists only so the line-level expandable (order_line) can be processed.
func stubLoadInvoiceLines(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, nil
}

// Hands the resolver the stashed order line so nested includes below it resolve.
func extractOrderLineRefFromInvoiceLine(ctx context.Context, parent any) []any {
	l := parent.(*apiresource.InvoiceLine)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeInvoiceLine, l.ID, "order_line")
	if !ok {
		return nil
	}
	return []any{v.(*apiresource.SalesOrderLine)}
}

func populateOrderLineOnInvoiceLine(ctx context.Context, parent any, _ map[string]any) {
	l := parent.(*apiresource.InvoiceLine)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeInvoiceLine, l.ID, "order_line")
	if !ok {
		return
	}
	l.OrderLine = v.(*apiresource.SalesOrderLine)
}

func extractCustomerIDFromInvoiceForPayment(ctx context.Context, parent any) []string {
	inv := parent.(*apiresource.InvoiceForPayment)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeInvoiceForPayment, inv.ID, "customer_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateCustomerOnInvoiceForPayment(ctx context.Context, parent any, loaded map[string]any) {
	inv := parent.(*apiresource.InvoiceForPayment)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeInvoiceForPayment, inv.ID, "customer_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		inv.Customer = v.(*apiresource.Customer)
	}
}

func extractParentAccountIDFromInvoiceForPayment(ctx context.Context, parent any) []string {
	inv := parent.(*apiresource.InvoiceForPayment)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeInvoiceForPayment, inv.ID, "parent_account_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateParentAccountOnInvoiceForPayment(ctx context.Context, parent any, loaded map[string]any) {
	inv := parent.(*apiresource.InvoiceForPayment)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeInvoiceForPayment, inv.ID, "parent_account_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		inv.ParentAccount = v.(*apiresource.Account)
	}
}

// Populates the payment list's allocations from the stash, so the settle flow can work out each
// invoice's paid amount without a second request per invoice.
func populateAllocationsOnInvoiceForPayment(ctx context.Context, parent any, _ map[string]any) {
	inv := parent.(*apiresource.InvoiceForPayment)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeInvoiceForPayment, inv.ID, "allocations")
	if !ok {
		return
	}
	inv.Allocations = v.(*apiresource.List[apiresource.InvoiceAllocation])
}

// Lazily creates the related group on first expanded member, so it serializes to null when no
// related include was requested.
func ensureInvoiceRelated(inv *apiresource.Invoice) *apiresource.InvoiceRelated {
	if inv.Related == nil {
		inv.Related = &apiresource.InvoiceRelated{Object: constants.ObjectTypeInvoiceRelated}
	}
	return inv.Related
}

func populateOrderOnInvoiceRelated(ctx context.Context, parent any, loaded map[string]any) {
	inv := parent.(*apiresource.Invoice)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeInvoice, inv.ID, "order_id")
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
	ensureInvoiceRelated(inv).SalesOrder = rec
}

func populateShipmentOnInvoiceRelated(ctx context.Context, parent any, loaded map[string]any) {
	inv := parent.(*apiresource.Invoice)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeInvoice, inv.ID, "shipment_id")
	if id == "" {
		return
	}
	v, ok := loaded[id]
	if !ok {
		return
	}
	s := v.(*apiresource.Shipment)
	rec := apiresource.NewRecord(id, constants.RecordTypeShipment)
	rec.Number = &s.Number
	status := string(s.Status)
	rec.Status = &status
	ensureInvoiceRelated(inv).Shipment = rec
}
