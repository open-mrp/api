package repository

import (
	"context"
	"strconv"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/id"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/db"
	"github.com/open-mrp/apikit/metadata"
	"github.com/open-mrp/apikit/tracing"
)

// documentPartyID is the key of accountID's party row on a document: derived, so a write and the backfill that copies
// the legacy columns land on the same row.
func documentPartyID(documentID, accountID string) string {
	return id.DerivedID(id.DocumentPartyIDPrefix, documentID+":"+accountID)
}

// attachForgeFields fills in the columns the detail queries don't read: the address snapshots, site, firmness and
// promised lead time, and viewerAccountID's own instructions and metadata.
//
//  1. Read every order's columns and the viewer's party row in one query.
//  2. Without a snapshot yet (rows the backfill hasn't reached), copy the live saved address the order names.
//  3. With a party row, take its instructions and metadata. Without one, the owner keeps the legacy columns it already
//     read, and anyone else reads none.
func (r *salesOrderRepoImpl) attachForgeFields(ctx context.Context, viewerAccountID string, orders []*domain.SalesOrder) *apierror.APIError {
	ctx, span := salesOrderRepoTracer.Start(ctx, "repository.sales_order.attach_forge_fields")
	defer span.End()

	if len(orders) == 0 {
		return nil
	}
	ids := make([]string, len(orders))
	for i, o := range orders {
		ids[i] = o.ID
	}
	rows, err := r.queries.GetSalesOrderForgeFields(ctx, sqlc.GetSalesOrderForgeFieldsParams{ViewerAccountID: viewerAccountID, Ids: ids})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	byID := make(map[string]sqlc.GetSalesOrderForgeFieldsRow, len(rows))
	for _, row := range rows {
		byID[row.ID] = row
	}
	for _, o := range orders {
		row, ok := byID[o.ID]
		if !ok {
			continue
		}
		o.BillTo = AddressSnapshotColumns{
			AddressID: row.BillToAddressID, Name: row.BillToName, Line1: row.BillToLine1, Line2: row.BillToLine2,
			City: row.BillToCity, State: row.BillToState, PostalCode: row.BillToPostalCode, Country: row.BillToCountry,
			Phone: row.BillToPhone, Email: row.BillToEmail,
		}.Snapshot()
		if o.BillTo == nil {
			o.BillTo = liveOrderAddress(o.BillingAddressID, o.BillToName, o.BillToStreetLine1, o.BillToStreetLine2, o.BillToLocality,
				o.BillToState, o.BillToPostalCode, o.BillToCountry, o.BillToPhone, o.BillToEmail)
		}
		o.ShipTo = AddressSnapshotColumns{
			AddressID: row.ShipToAddressID, Name: row.ShipToName, Line1: row.ShipToLine1, Line2: row.ShipToLine2,
			City: row.ShipToCity, State: row.ShipToState, PostalCode: row.ShipToPostalCode, Country: row.ShipToCountry,
			Phone: row.ShipToPhone, Email: row.ShipToEmail,
		}.Snapshot()
		if o.ShipTo == nil {
			o.ShipTo = liveOrderAddress(o.ShippingAddressID, o.ShipToName, o.ShipToStreetLine1, o.ShipToStreetLine2, o.ShipToLocality,
				o.ShipToState, o.ShipToPostalCode, o.ShipToCountry, o.ShipToPhone, o.ShipToEmail)
		}
		o.SiteID = nullStringToPtr(row.SiteID)
		o.CommitmentFirmness = nullStringToPtr(row.CommitmentFirmness)
		if row.PromisedLeadTimeDays.Valid {
			days := row.PromisedLeadTimeDays.Int32
			o.PromisedLeadTimeDays = &days
		}
		o.PromisedLeadTimeSource = nullStringToPtr(row.PromisedLeadTimeSource)

		switch {
		case row.PartyID.Valid:
			o.Instructions = nullStringToPtr(row.PartyInstructions)
			o.Metadata = metadata.Decode(row.PartyMetadata)
		case viewerAccountID != o.OwnerAccountID:
			o.Instructions = nil
			o.Metadata = map[string]string{}
		}
	}
	return nil
}

// liveOrderAddress is the saved address an order names, as a snapshot, for an order that has no copy of its own yet.
// nil when the order names none or it no longer exists.
func liveOrderAddress(addressID string, name, line1, line2, city, state, postalCode, country, phone, email *string) *domain.AddressSnapshot {
	if addressID == "" || country == nil {
		return nil
	}
	snap := &domain.AddressSnapshot{
		AddressID: &addressID, Line1: line1, Line2: line2, City: city, State: state, PostalCode: postalCode,
		Country: *country, Phone: phone, Email: email,
	}
	if name != nil {
		snap.Name = *name
	}
	return snap
}

// writeOwnerParty stores the owner's instructions and metadata on its party row, which mirrors the order's legacy
// note and metadata columns until those are dropped.
func (r *salesOrderRepoImpl) writeOwnerParty(ctx context.Context, order *domain.SalesOrder) *apierror.APIError {
	err := r.queries.UpsertDocumentParty(ctx, sqlc.UpsertDocumentPartyParams{
		ID:           documentPartyID(order.ID, order.OwnerAccountID),
		DocumentID:   order.ID,
		AccountID:    order.OwnerAccountID,
		Instructions: toNullString(order.Instructions),
		Metadata:     metadata.Encode(order.Metadata),
	})
	return db.MapSQLError(err)
}

// GetItemLeadTimeDays is the longest sales lead time of the items on the order's lines, in whole days rounded up; nil
// when none of them has one.
func (r *salesOrderRepoImpl) GetItemLeadTimeDays(ctx context.Context, salesOrderID string) (*int, *apierror.APIError) {
	ctx, span := salesOrderRepoTracer.Start(ctx, "repository.sales_order.get_item_lead_time_days")
	defer span.End()

	v, err := r.queries.GetSalesOrderItemLeadTimeDays(ctx, salesOrderID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	var days int64
	switch t := v.(type) {
	case int64:
		days = t
	case []byte:
		parsed, err := strconv.ParseInt(string(t), 10, 64)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Unreadable item lead time."))
		}
		days = parsed
	}
	if days < 0 {
		return nil, nil
	}
	out := int(days)
	return &out, nil
}

// GetEmailRecipients is each order's email recipients and the notification types they receive, in the order they
// were added. Orders with none are absent.
func (r *salesOrderRepoImpl) GetEmailRecipients(ctx context.Context, salesOrderIDs []string) (map[string][]domain.NotificationRecipientRef, *apierror.APIError) {
	ctx, span := salesOrderRepoTracer.Start(ctx, "repository.sales_order.get_email_recipients")
	defer span.End()

	out := make(map[string][]domain.NotificationRecipientRef)
	if len(salesOrderIDs) == 0 {
		return out, nil
	}
	rows, err := r.queries.GetOrderEmailRecipients(ctx, salesOrderIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	for _, row := range rows {
		refs := out[row.SalesOrderID]
		found := false
		for i := range refs {
			if refs[i].AccountUserID == row.AccountUserID {
				refs[i].NotificationTypeCodes = append(refs[i].NotificationTypeCodes, row.NotificationTypeCode)
				found = true
				break
			}
		}
		if !found {
			refs = append(refs, domain.NotificationRecipientRef{AccountUserID: row.AccountUserID, NotificationTypeCodes: []string{row.NotificationTypeCode}})
		}
		out[row.SalesOrderID] = refs
	}
	return out, nil
}

// ReplaceEmailRecipients makes recipients the order's email recipients.
func (r *salesOrderRepoImpl) ReplaceEmailRecipients(ctx context.Context, salesOrderID string, recipients []domain.NotificationRecipientRef) *apierror.APIError {
	ctx, span := salesOrderRepoTracer.Start(ctx, "repository.sales_order.replace_email_recipients")
	defer span.End()

	if apiErr := db.MapSQLError(r.queries.DeleteOrderEmailContacts(ctx, salesOrderID)); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	for _, rcpt := range recipients {
		for _, code := range rcpt.NotificationTypeCodes {
			contactID, apiErr := id.GenID(id.OrderEmailIDPrefix, nil)
			if apiErr != nil {
				return tracing.Trace(span, apiErr)
			}
			if apiErr := r.CreateEmailContact(ctx, contactID, salesOrderID, rcpt.AccountUserID, code); apiErr != nil {
				return tracing.Trace(span, apiErr)
			}
		}
	}
	return nil
}
