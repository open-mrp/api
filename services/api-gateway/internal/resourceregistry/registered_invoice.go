package resourceregistry

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
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
			{Key: "allocations", Target: constants.ObjectTypeInvoiceAllocation, ExtractRefs: extractAllocationRefsFromInvoice, Populate: populateAllocationsOnInvoice},
		},
	})

	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeInvoiceAllocation,
		Load:       stubLoadInvoiceAllocations,
		Subs: []resourcekit.SubField{
			{
				// The amount is already on the allocation — this exists so a caller can reach through
				// it to the currency it is counted in.
				Key:         "amount",
				Target:      constants.ObjectTypeQuantity,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractRefs: extractAmountRefFromInvoiceAllocation,
			},
			{
				Key:         "transaction",
				Target:      constants.ObjectTypeTransaction,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractTransactionIDFromInvoiceAllocation,
				Populate:    populateTransactionOnInvoiceAllocation,
			},
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
			{
				Key:         "item",
				Target:      constants.ObjectTypeItem,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractItemIDFromInvoiceLine,
				Populate:    populateItemOnInvoiceLine,
			},
			{
				// The quantity and the price are already on the line — these exist so a caller can
				// reach through them to the units they are counted in.
				Key:         "quantity",
				Target:      constants.ObjectTypeQuantity,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractRefs: extractQuantityRefFromInvoiceLine,
			},
			{
				Key:         "unit_price",
				Target:      constants.ObjectTypeRate,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractRefs: extractUnitPriceRefFromInvoiceLine,
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
			{Key: "allocations", Target: constants.ObjectTypeInvoiceAllocation, ExtractRefs: extractAllocationRefsFromInvoiceForPayment, Populate: populateAllocationsOnInvoiceForPayment},
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

func stubLoadInvoiceAllocations(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, nil
}

// The resolver runs Populate before gathering refs, so the allocations are already on the invoice.
func extractAllocationRefsFromInvoice(_ context.Context, parent any) []any {
	inv := parent.(*apiresource.Invoice)
	if inv.Allocations == nil {
		return nil
	}
	refs := make([]any, len(inv.Allocations.Data))
	for i := range inv.Allocations.Data {
		refs[i] = &inv.Allocations.Data[i]
	}
	return refs
}

func extractTransactionIDFromInvoiceAllocation(ctx context.Context, parent any) []string {
	a := parent.(*apiresource.InvoiceAllocation)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeInvoiceAllocation, a.ID, "transaction_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateTransactionOnInvoiceAllocation(ctx context.Context, parent any, loaded map[string]any) {
	a := parent.(*apiresource.InvoiceAllocation)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeInvoiceAllocation, a.ID, "transaction_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		a.Transaction = v.(*apiresource.TransactionDetail)
	}
}

func extractAmountRefFromInvoiceAllocation(_ context.Context, parent any) []any {
	a := parent.(*apiresource.InvoiceAllocation)
	if a.Amount == nil {
		return nil
	}
	return []any{a.Amount}
}

func extractItemIDFromInvoiceLine(ctx context.Context, parent any) []string {
	l := parent.(*apiresource.InvoiceLine)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeInvoiceLine, l.ID, "item_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateItemOnInvoiceLine(ctx context.Context, parent any, loaded map[string]any) {
	l := parent.(*apiresource.InvoiceLine)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeInvoiceLine, l.ID, "item_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		l.Item = v.(*apiresource.Item)
	}
}

func extractQuantityRefFromInvoiceLine(_ context.Context, parent any) []any {
	l := parent.(*apiresource.InvoiceLine)
	if l.Quantity == nil {
		return nil
	}
	return []any{l.Quantity}
}

func extractUnitPriceRefFromInvoiceLine(_ context.Context, parent any) []any {
	l := parent.(*apiresource.InvoiceLine)
	if l.UnitPrice == nil {
		return nil
	}
	return []any{l.UnitPrice}
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

func extractAllocationRefsFromInvoiceForPayment(_ context.Context, parent any) []any {
	inv := parent.(*apiresource.InvoiceForPayment)
	if inv.Allocations == nil {
		return nil
	}
	refs := make([]any, len(inv.Allocations.Data))
	for i := range inv.Allocations.Data {
		refs[i] = &inv.Allocations.Data[i]
	}
	return refs
}
