package resourceregistry

import (
	"context"
	"time"

	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeSalesOrder,
		Load:       resourceloaders.LoadSalesOrders,
		Subs: []include.SubField{
			salesOrderRef("customer", constants.ObjectTypeCustomer, resourceloaders.SalesOrderMetaCustomerID, func(so *apiresource.SalesOrder, v any) {
				so.Customer = v.(*apiresource.Customer)
			}),
			SiteSub(constants.ObjectTypeSalesOrder, salesOrderIDOf, func(parent any, site *apiresource.Location) {
				parent.(*apiresource.SalesOrder).Site = site
			}),
			salesOrderRef("sales_rep", constants.ObjectTypeAccountUser, resourceloaders.SalesOrderMetaSalesRepID, func(so *apiresource.SalesOrder, v any) {
				so.SalesRep = v.(*apiresource.AccountUser)
			}),
			{
				Key:         "created_by",
				Target:      constants.ObjectTypeCreatedBy,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractSelfIDForCreatedBy,
				Populate:    populateCreatedByOnSO,
			},
			AddressSnapshotSub(constants.ObjectTypeSalesOrder, "bill_to", func(parent any) (string, *apiresource.AddressSnapshot) {
				so := parent.(*apiresource.SalesOrder)
				return so.ID, so.BillTo
			}),
			AddressSnapshotSub(constants.ObjectTypeSalesOrder, "ship_to", func(parent any) (string, *apiresource.AddressSnapshot) {
				so := parent.(*apiresource.SalesOrder)
				return so.ID, so.ShipTo
			}),
			salesOrderRef("freight.carrier", constants.ObjectTypeCarrier, resourceloaders.SalesOrderMetaCarrierID, func(so *apiresource.SalesOrder, v any) {
				so.Freight.Carrier = v.(*apiresource.Carrier)
			}),
			salesOrderRef("freight.service_level", constants.ObjectTypeServiceLevel, resourceloaders.SalesOrderMetaServiceLevelID, func(so *apiresource.SalesOrder, v any) {
				so.Freight.ServiceLevel = v.(*apiresource.ServiceLevel)
			}),
			salesOrderRef("payment_term", constants.ObjectTypePaymentTerm, resourceloaders.SalesOrderMetaPaymentTermID, func(so *apiresource.SalesOrder, v any) {
				so.PaymentTerm = v.(*apiresource.PaymentTerm)
			}),
			salesOrderRef("shipping_term", constants.ObjectTypeShippingTerm, resourceloaders.SalesOrderMetaShippingTermID, func(so *apiresource.SalesOrder, v any) {
				so.ShippingTerm = v.(*apiresource.ShippingTerm)
			}),
			salesOrderRef("order_discount", constants.ObjectTypeOrderDiscount, resourceloaders.SalesOrderMetaOrderDiscountID, func(so *apiresource.SalesOrder, v any) {
				so.OrderDiscount = v.(*apiresource.OrderDiscount)
			}),
			salesOrderStashed("bill_to_address", resourceloaders.SalesOrderMetaBillToAddress, func(so *apiresource.SalesOrder, v any) {
				so.BillToAddress = v.(*apiresource.Address)
			}),
			salesOrderStashed("ship_to_address", resourceloaders.SalesOrderMetaShipToAddress, func(so *apiresource.SalesOrder, v any) {
				so.ShipToAddress = v.(*apiresource.Address)
			}),
			salesOrderStashed("contacts", resourceloaders.SalesOrderMetaContacts, func(so *apiresource.SalesOrder, v any) {
				so.Contacts = v.(*apiresource.OrderContact)
			}),
			{Key: "totals", Populate: populateTotalsOnSO},
			{Key: "email_recipients", Populate: populateRecipientsOnSO},
			{Key: "email_recipients.account_user", Populate: populateRecipientUsersOnSO},
			{
				Key:         "related.pick",
				Target:      constants.ObjectTypePick,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractPickIDFromSORelated,
				Populate:    populatePickOnSORelated,
			},
			{
				Key:         "related.jobs",
				Target:      constants.ObjectTypeProductionRun,
				Cardinality: include.CardinalityList,
				ExtractIDs:  extractJobIDsFromSORelated,
				Populate:    populateJobsOnSORelated,
			},
			{
				Key:         "related.shipments",
				Target:      constants.ObjectTypeShipment,
				Cardinality: include.CardinalityList,
				ExtractIDs:  extractShipmentIDsFromSORelated,
				Populate:    populateShipmentsOnSORelated,
			},
			{
				Key:         "related.invoices",
				Target:      constants.ObjectTypeInvoice,
				Cardinality: include.CardinalityList,
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

func salesOrderIDOf(parent any) string { return parent.(*apiresource.SalesOrder).ID }

// salesOrderRef declares an include that loads one record an order names by the ID its presenter stashed under
// metaKey, and sets it with set.
func salesOrderRef(key string, target constants.ObjectType, metaKey string, set func(so *apiresource.SalesOrder, v any)) include.SubField {
	idOf := func(ctx context.Context, parent any) string {
		id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeSalesOrder, salesOrderIDOf(parent), metaKey)
		return id
	}
	return include.SubField{
		Key:         key,
		Target:      target,
		Cardinality: include.CardinalityOnePtr,
		ExtractIDs: func(ctx context.Context, parent any) []string {
			if id := idOf(ctx, parent); id != "" {
				return []string{id}
			}
			return nil
		},
		Populate: func(ctx context.Context, parent any, loaded map[string]any) {
			so := parent.(*apiresource.SalesOrder)
			if so.Freight == nil {
				so.Freight = &apiresource.SalesOrderFreight{}
			}
			if v, ok := loaded[idOf(ctx, parent)]; ok && v != nil {
				set(so, v)
			}
		},
	}
}

// salesOrderStashed declares an include its presenter already built, stashed under metaKey.
func salesOrderStashed(key, metaKey string, set func(so *apiresource.SalesOrder, v any)) include.SubField {
	return include.SubField{Key: key, Populate: func(ctx context.Context, parent any, _ map[string]any) {
		so := parent.(*apiresource.SalesOrder)
		if v, ok := include.GetLoadMeta(ctx).Get(constants.ObjectTypeSalesOrder, so.ID, metaKey); ok {
			set(so, v)
		}
	}}
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

func populateTotalsOnSO(ctx context.Context, parent any, _ map[string]any) {
	so := parent.(*apiresource.SalesOrder)
	v, ok := include.GetLoadMeta(ctx).Get(constants.ObjectTypeSalesOrder, so.ID, "totals")
	if !ok {
		return
	}
	so.Totals = v.(*apiresource.SalesOrderTotals)
}

func populateRecipientsOnSO(ctx context.Context, parent any, _ map[string]any) {
	so := parent.(*apiresource.SalesOrder)
	if v, ok := include.GetLoadMeta(ctx).Get(constants.ObjectTypeSalesOrder, so.ID, resourceloaders.SalesOrderMetaRecipients); ok {
		so.EmailRecipients = v.(*apiresource.List[apiresource.EmailRecipient])
	}
}

// populateRecipientUsersOnSO sets each recipient's account user, which core-service loaded with the recipients:
// they belong to the customer's account, which the account-user loader can't reach.
func populateRecipientUsersOnSO(ctx context.Context, parent any, _ map[string]any) {
	so := parent.(*apiresource.SalesOrder)
	if so.EmailRecipients == nil {
		populateRecipientsOnSO(ctx, parent, nil)
	}
	if so.EmailRecipients == nil {
		return
	}
	for i := range so.EmailRecipients.Data {
		so.EmailRecipients.Data[i].AccountUser = resourceloaders.SalesOrderRecipientUser(ctx, so.ID, i)
	}
}

// ensureSORelated lazily creates the related group on first expanded member, so the group serializes to null when no related include was requested.
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
	id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeSalesOrder, so.ID, resourceloaders.SalesOrderMetaPickID)
	if id == "" {
		return nil
	}
	return []string{id}
}

func populatePickOnSORelated(ctx context.Context, parent any, loaded map[string]any) {
	so := parent.(*apiresource.SalesOrder)
	id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeSalesOrder, so.ID, resourceloaders.SalesOrderMetaPickID)
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

func extractJobIDsFromSORelated(ctx context.Context, parent any) []string {
	so := parent.(*apiresource.SalesOrder)
	ids, _ := include.GetLoadMeta(ctx).GetStrings(constants.ObjectTypeSalesOrder, so.ID, resourceloaders.SalesOrderMetaJobIDs)
	return ids
}

// populateJobsOnSORelated lists the production started for the order. Until jobs replace production runs, that is
// the order's production run.
func populateJobsOnSORelated(ctx context.Context, parent any, loaded map[string]any) {
	so := parent.(*apiresource.SalesOrder)
	records := make([]apiresource.Record, 0)
	for _, id := range extractJobIDsFromSORelated(ctx, parent) {
		v, ok := loaded[id]
		if !ok {
			continue
		}
		pr := v.(*apiresource.ProductionRun)
		rec := apiresource.NewRecord(id, constants.RecordTypeProductionRun)
		rec.Number = &pr.Number
		status := openClosedStatus(pr.CompletedAt)
		rec.Status = &status
		records = append(records, *rec)
	}
	ensureSORelated(so).Jobs = apiresource.NewList(records, apiresource.PageInfo{})
}

func extractShipmentIDsFromSORelated(ctx context.Context, parent any) []string {
	so := parent.(*apiresource.SalesOrder)
	ids, _ := include.GetLoadMeta(ctx).GetStrings(constants.ObjectTypeSalesOrder, so.ID, resourceloaders.SalesOrderMetaShipmentIDs)
	return ids
}

func populateShipmentsOnSORelated(ctx context.Context, parent any, loaded map[string]any) {
	so := parent.(*apiresource.SalesOrder)
	records := make([]apiresource.Record, 0)
	for _, id := range extractShipmentIDsFromSORelated(ctx, parent) {
		v, ok := loaded[id]
		if !ok {
			continue
		}
		s := v.(*apiresource.Shipment)
		rec := apiresource.NewRecord(id, constants.RecordTypeShipment)
		rec.Number = &s.Number
		status := string(s.Status)
		rec.Status = &status
		// Tracking number/URL, carrier, and ship date (stashed by the shipment loader) let an order preview each
		// shipment without expanding it.
		if m, ok := include.GetLoadMeta(ctx).Get(constants.ObjectTypeShipment, id, "record_metadata"); ok {
			if meta, ok := m.(map[string]string); ok && len(meta) > 0 {
				rec.Metadata = meta
			}
		}
		records = append(records, *rec)
	}
	ensureSORelated(so).Shipments = apiresource.NewList(records, apiresource.PageInfo{})
}

func extractInvoiceIDsFromSORelated(ctx context.Context, parent any) []string {
	so := parent.(*apiresource.SalesOrder)
	ids, _ := include.GetLoadMeta(ctx).GetStrings(constants.ObjectTypeSalesOrder, so.ID, resourceloaders.SalesOrderMetaInvoiceIDs)
	return ids
}

func populateInvoicesOnSORelated(ctx context.Context, parent any, loaded map[string]any) {
	so := parent.(*apiresource.SalesOrder)
	records := make([]apiresource.Record, 0)
	for _, id := range extractInvoiceIDsFromSORelated(ctx, parent) {
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
	ensureSORelated(so).Invoices = apiresource.NewList(records, apiresource.PageInfo{})
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
	v, ok := include.GetLoadMeta(ctx).Get(constants.ObjectTypeSalesOrder, so.ID, "lines")
	if !ok {
		return
	}
	so.Lines = v.(*apiresource.List[apiresource.SalesOrderLine])
}
