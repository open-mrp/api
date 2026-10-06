package service

import (
	"context"
	"fmt"
	"slices"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	apierror "github.com/open-mrp/api/shared/errors"
)

// customerRefKind is a record a customer's account_relation row points at. The row has no foreign keys, so this check is all that stops it naming another account's record.
type customerRefKind string

const (
	customerRefCarrier         customerRefKind = "carrier"
	customerRefServiceLevel    customerRefKind = "service level"
	customerRefPaymentTerm     customerRefKind = "payment term"
	customerRefShippingTerm    customerRefKind = "shipping term"
	customerRefReceiveCalendar customerRefKind = "receive calendar"
	customerRefSalesRep        customerRefKind = "sales rep"
	customerRefAccountGroup    customerRefKind = "account group"
)

// customerRef is one id a customer write would store, with the request field it came from.
type customerRef struct {
	kind  customerRefKind
	id    string
	param string
}

type customerRefs []customerRef

// add names id unless it is absent or is already the record the customer holds.
func (r *customerRefs) add(kind customerRefKind, id, held *string, param string) {
	if id == nil || *id == "" || (held != nil && *held == *id) {
		return
	}
	*r = append(*r, customerRef{kind: kind, id: *id, param: param})
}

// newCustomerRefs lists every reference a customer create stores.
func newCustomerRefs(params domain.CreateCustomerParams) customerRefs {
	var refs customerRefs
	refs.add(customerRefCarrier, params.DefaultCarrierID, nil, "default_carrier_id")
	refs.add(customerRefServiceLevel, params.DefaultServiceLevelID, nil, "default_service_level_id")
	refs.add(customerRefPaymentTerm, params.DefaultPaymentTermID, nil, "default_payment_term_id")
	refs.add(customerRefShippingTerm, params.DefaultShippingTermID, nil, "default_shipping_term_id")
	refs.add(customerRefReceiveCalendar, params.ReceiveCalendarID, nil, "receive_calendar_id")
	refs.add(customerRefSalesRep, params.DefaultSalesRepID, nil, "default_sales_rep_id")
	refs.add(customerRefAccountGroup, params.CustomerTypeGroupID, nil, "customer_type_group_id")
	for _, groupID := range params.CustomerPriceGroupIDs {
		refs.add(customerRefAccountGroup, &groupID, nil, "customer_price_group_ids")
	}
	return refs
}

// changedCustomerRefs lists the references an update moves to a different record. One the customer already holds is left alone, so an edit is not refused over a reference it did not change.
func changedCustomerRefs(params domain.UpdateCustomerParams, old *domain.Customer) customerRefs {
	var refs customerRefs
	refs.add(customerRefCarrier, params.DefaultCarrierID, old.DefaultCarrierID, "default_carrier_id")
	refs.add(customerRefServiceLevel, params.DefaultServiceLevelID.ValuePtr(), old.DefaultServiceLevelID, "default_service_level_id")
	refs.add(customerRefPaymentTerm, params.DefaultPaymentTermID, old.DefaultPaymentTermID, "default_payment_term_id")
	refs.add(customerRefShippingTerm, params.DefaultShippingTermID, old.DefaultShippingTermID, "default_shipping_term_id")
	refs.add(customerRefReceiveCalendar, params.ReceiveCalendarID.ValuePtr(), old.ReceiveCalendarID, "receive_calendar_id")
	refs.add(customerRefSalesRep, params.DefaultSalesRepID.ValuePtr(), old.DefaultSalesRepID, "default_sales_rep_id")
	refs.add(customerRefAccountGroup, params.CustomerTypeGroupID, old.TypeGroupID, "customer_type_group_id")
	if params.HasCustomerPriceGroupIDs {
		for _, groupID := range params.CustomerPriceGroupIDs {
			held := slices.ContainsFunc(old.PriceGroups, func(g domain.CustomerAccountGroup) bool { return g.ID == groupID })
			if !held {
				refs.add(customerRefAccountGroup, &groupID, nil, "customer_price_group_ids")
			}
		}
	}
	return refs
}

// checkCustomerRefs refuses, as not found on the field that named it, any reference that is neither the account's own record nor a system record of a kind that has them. Each kind is read once however many ids name it.
func checkCustomerRefs(ctx context.Context, repos domain.RepoFactory, accountID string, refs customerRefs) *apierror.APIError {
	var kinds []customerRefKind
	idsByKind := make(map[customerRefKind][]string)
	for _, ref := range refs {
		if _, seen := idsByKind[ref.kind]; !seen {
			kinds = append(kinds, ref.kind)
		}
		idsByKind[ref.kind] = append(idsByKind[ref.kind], ref.id)
	}

	for _, kind := range kinds {
		found, apiErr := findCustomerRefs(ctx, repos, accountID, kind, idsByKind[kind])
		if apiErr != nil {
			return apiErr
		}
		for _, ref := range refs {
			if ref.kind == kind && !found[ref.id] {
				return apierror.NewResourceNotFoundError(fmt.Sprintf("No %s found with the provided ID.", kind)).WithParam(ref.param)
			}
		}
	}
	return nil
}

// findCustomerRefs returns which of ids the account may use as a record of kind.
func findCustomerRefs(ctx context.Context, repos domain.RepoFactory, accountID string, kind customerRefKind, ids []string) (map[string]bool, *apierror.APIError) {
	found := make(map[string]bool, len(ids))
	switch kind {
	case customerRefCarrier:
		carriers, apiErr := repos.NewCarrierRepo().GetByIDs(ctx, accountID, ids)
		if apiErr != nil {
			return nil, apiErr
		}
		for _, carrier := range carriers {
			found[carrier.ID] = true
		}
	case customerRefServiceLevel:
		levels, apiErr := repos.NewCarrierRepo().GetOptionsByIDs(ctx, accountID, ids)
		if apiErr != nil {
			return nil, apiErr
		}
		for _, level := range levels {
			found[level.ID] = true
		}
	case customerRefPaymentTerm:
		terms, apiErr := repos.NewPaymentTermRepo().GetByIDs(ctx, accountID, ids)
		if apiErr != nil {
			return nil, apiErr
		}
		for _, term := range terms {
			found[term.ID] = true
		}
	case customerRefShippingTerm:
		terms, apiErr := repos.NewShippingTermRepo().GetByIDs(ctx, accountID, ids)
		if apiErr != nil {
			return nil, apiErr
		}
		for _, term := range terms {
			found[term.ID] = true
		}
	case customerRefAccountGroup:
		groups, apiErr := repos.NewAccountGroupRepo().GetByIDs(ctx, accountID, ids)
		if apiErr != nil {
			return nil, apiErr
		}
		for _, group := range groups {
			found[group.ID] = true
		}
	case customerRefReceiveCalendar:
		for _, calendarID := range ids {
			_, apiErr := repos.NewOperatingCalendarRepo().Get(ctx, accountID, calendarID)
			if apiErr != nil && !apierror.IsNotFound(apiErr) {
				return nil, apiErr
			}
			found[calendarID] = apiErr == nil
		}
	case customerRefSalesRep:
		for _, accountUserID := range ids {
			_, apiErr := repos.NewAccountUserRepo().GetDetailByAccountAndID(ctx, accountID, accountUserID, nil)
			if apiErr != nil && !apierror.IsNotFound(apiErr) {
				return nil, apiErr
			}
			found[accountUserID] = apiErr == nil
		}
	default:
		return nil, apierror.NewInvariantViolationError(fmt.Sprintf("Unknown customer reference kind %q.", kind))
	}
	return found, nil
}
