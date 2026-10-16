package service

import (
	"context"
	"fmt"
	"reflect"
	"slices"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/id"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/ptrutil"
)

// orderRecipientTypes are the notification types an order's email recipients can receive.
var orderRecipientTypes = []string{
	string(constants.AccountRelationNotificationTypeOrderAcknowledgement),
	string(constants.AccountRelationNotificationTypeInvoice),
	string(constants.AccountRelationNotificationTypeShipment),
}

// resolveOrderSnapshot is the copy an order keeps of an address it names (input.AddressID) or is given inline.
//
//  1. A saved address must be in the buyer's or the owner's address book; anything else is not found on
//     `<param>.address_id`.
//  2. An inline address needs a name and a country.
func (s *salesOrderSvcImpl) resolveOrderSnapshot(ctx context.Context, ownerAccountID, buyerAccountID string, input *domain.AddressSnapshot, param string) (*domain.AddressSnapshot, *apierror.APIError) {
	if input == nil {
		return nil, nil
	}
	accountID := buyerAccountID
	if input.AddressID != nil {
		holder, apiErr := orderAddressAccount(ctx, s.repos.NewAddressRepo(), ownerAccountID, buyerAccountID, *input.AddressID)
		if apiErr != nil {
			return nil, apiErr
		}
		if holder == "" {
			return nil, apierror.NewNotFoundError("No address found with the provided ID.").WithParam(param + ".address_id")
		}
		accountID = holder
	}
	return s.mediators().Address.Snapshot(ctx, accountID, input, param)
}

// orderAddressID is the address row an order's legacy address column points at for snap: the saved address it was
// copied from, or, for an address given inline, a row of the order's own that no address book lists. Picks,
// shipments and invoices still read the order's address through that column.
func (s *salesOrderSvcImpl) orderAddressID(ctx context.Context, snap *domain.AddressSnapshot) (string, *apierror.APIError) {
	if snap.AddressID != nil {
		return *snap.AddressID, nil
	}
	addressID, apiErr := id.GenID(id.AddressIDPrefix, nil)
	if apiErr != nil {
		return "", apiErr
	}
	geolocationID, apiErr := id.GenID(id.GeolocationIDPrefix, nil)
	if apiErr != nil {
		return "", apiErr
	}
	if apiErr := s.repos.NewAddressRepo().CreateOrderAddress(ctx, addressID, geolocationID, domain.CreateAddressParams{
		Name:        snap.Name,
		Phone:       snap.Phone,
		Email:       snap.Email,
		StreetLine1: snap.Line1,
		StreetLine2: snap.Line2,
		Locality:    snap.City,
		State:       snap.State,
		PostalCode:  snap.PostalCode,
		Country:     snap.Country,
	}); apiErr != nil {
		return "", apiErr
	}
	return addressID, nil
}

// sameSnapshot reports whether two address copies are identical, so one given inline for both sides is stored once.
func sameSnapshot(a, b *domain.AddressSnapshot) bool {
	return reflect.DeepEqual(a, b)
}

// shippingAddressFromSnapshot is an order's address copy in the form the shipping-rate and sales-rep logic read.
func shippingAddressFromSnapshot(snap *domain.AddressSnapshot) domain.ShippingAddress {
	if snap == nil {
		return domain.ShippingAddress{}
	}
	return domain.ShippingAddress{
		Name:    snap.Name,
		Street1: ptrutil.Deref(snap.Line1),
		Street2: snap.Line2,
		City:    ptrutil.Deref(snap.City),
		State:   ptrutil.Deref(snap.State),
		Zip:     ptrutil.Deref(snap.PostalCode),
		Country: snap.Country,
		Phone:   snap.Phone,
		Email:   snap.Email,
	}
}

// checkOrderRecipients requires each recipient to be one of the buyer's account users and every document one an
// order sends. Errors name `<param>[i]`.
func (s *salesOrderSvcImpl) checkOrderRecipients(ctx context.Context, buyerAccountID string, recipients []domain.NotificationRecipientRef, param string) *apierror.APIError {
	for i, r := range recipients {
		at := fmt.Sprintf("%s[%d]", param, i)
		if r.AccountUserID == "" {
			return apierror.NewMissingFieldError(at+".account_user_id", "Each email recipient must reference an account user.")
		}
		if len(r.NotificationTypeCodes) == 0 {
			return apierror.NewMissingFieldError(at+".documents", "Each email recipient must receive at least one document.")
		}
		for _, code := range r.NotificationTypeCodes {
			if !slices.Contains(orderRecipientTypes, code) {
				return apierror.NewInvalidFormatError(at+".documents", "An order's recipients receive acknowledgments, invoices and shipments only.")
			}
		}
		if _, apiErr := s.repos.NewAccountUserRepo().GetDetailByAccountAndID(ctx, buyerAccountID, r.AccountUserID, nil); apiErr != nil {
			if apiErr.Code == apierror.CodeResourceNotFound {
				return apierror.NewNotFoundError("No account user of this customer has the provided ID.").WithParam(at + ".account_user_id")
			}
			return apiErr
		}
	}
	return nil
}

// customerOrderRecipients is the customer's notification recipients as a new order copies them: only the documents
// an order sends, and only recipients left with one.
func (s *salesOrderSvcImpl) customerOrderRecipients(ctx context.Context, ownerAccountID, buyerAccountID string) ([]domain.NotificationRecipientRef, *apierror.APIError) {
	relationRepo := s.repos.NewAccountRelationRepo()
	relationID, apiErr := relationRepo.FindRelationByOwnerAndCounterparty(ctx, ownerAccountID, buyerAccountID)
	if apiErr != nil {
		if apierror.HasCode(apiErr, apierror.CodeResourceNotFound) {
			return nil, nil
		}
		return nil, apiErr
	}
	refs, apiErr := relationRepo.ListNotificationRecipients(ctx, relationID)
	if apiErr != nil {
		return nil, apiErr
	}
	out := make([]domain.NotificationRecipientRef, 0, len(refs))
	for _, ref := range refs {
		var codes []string
		for _, code := range ref.NotificationTypeCodes {
			if slices.Contains(orderRecipientTypes, code) {
				codes = append(codes, code)
			}
		}
		if len(codes) > 0 {
			out = append(out, domain.NotificationRecipientRef{AccountUserID: ref.AccountUserID, NotificationTypeCodes: codes})
		}
	}
	return out, nil
}

// attachEmailRecipients fills in each order's email recipients with their account users on the buyer's account, as an
// empty list when it has none. A recipient whose account user no longer exists is left out.
func attachEmailRecipients(ctx context.Context, repos domain.RepoFactory, orders []*domain.SalesOrder) *apierror.APIError {
	if len(orders) == 0 {
		return nil
	}
	ids := make([]string, len(orders))
	for i, o := range orders {
		ids[i] = o.ID
	}
	byOrder, apiErr := repos.NewSalesOrderRepo().GetEmailRecipients(ctx, ids)
	if apiErr != nil {
		return apiErr
	}
	usersByBuyer := map[string][]string{}
	for _, o := range orders {
		for _, ref := range byOrder[o.ID] {
			usersByBuyer[o.BuyerAccountID] = append(usersByBuyer[o.BuyerAccountID], ref.AccountUserID)
		}
	}
	details := map[string]*domain.AccountUserDetail{}
	for buyer, userIDs := range usersByBuyer {
		found, apiErr := repos.NewAccountUserRepo().GetByIDs(ctx, buyer, userIDs)
		if apiErr != nil {
			return apiErr
		}
		for _, d := range found {
			details[buyer+"\x00"+d.ID] = d
		}
	}
	for _, o := range orders {
		o.EmailRecipients = []domain.NotificationRecipient{}
		for _, ref := range byOrder[o.ID] {
			if d, ok := details[o.BuyerAccountID+"\x00"+ref.AccountUserID]; ok {
				o.EmailRecipients = append(o.EmailRecipients, domain.NotificationRecipient{AccountUser: d, NotificationTypeCodes: ref.NotificationTypeCodes})
			}
		}
	}
	return nil
}

// includesSalesOrderEmailRecipients reports whether the email_recipients include (or one under it) was requested.
func includesSalesOrderEmailRecipients(includes []string) bool {
	for _, inc := range includes {
		if inc == "email_recipients" || inc == "email_recipients.account_user" {
			return true
		}
	}
	return false
}

// appendRecipientsChange records an order's email recipients moving from before to after, if they did.
func appendRecipientsChange(changes []audit.FieldChange, before, after []domain.NotificationRecipientRef) []audit.FieldChange {
	if change, ok := recipientsAuditChange(before, after); ok {
		change.Field = "email_recipients"
		return append(changes, change)
	}
	return changes
}

// cancelOutcome is the status an issued order is canceled to: canceled when nothing has shipped, else fulfilled with
// the unshipped remainder closed.
func cancelOutcome(order *domain.SalesOrder) constants.SalesOrderStatusCode {
	if order.FirstShipAt != nil {
		return constants.SalesOrderStatusCodeFulfilled
	}
	return constants.SalesOrderStatusCodeCanceled
}
