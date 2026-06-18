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
			{Key: "lines", Target: constants.ObjectTypeInvoiceLine, ExtractRefs: extractLineRefsFromInvoice, Populate: populateLinesOnInvoice},
			{Key: "allocations", Populate: populateAllocationsOnInvoice},
		},
	})

	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeInvoiceLine,
		Load:       stubLoadInvoiceLines,
		Subs: []resourcekit.SubField{
			{Key: "order_line", Populate: populateOrderLineOnInvoiceLine},
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
