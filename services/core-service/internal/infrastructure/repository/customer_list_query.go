package repository

import (
	"strings"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/pagination"
)

// customerListSearch is a list's search: the LIKE pattern, and the accounts it matches when they were few enough
// to find from each searched column's own key (customerSearchMatches).
type customerListSearch struct {
	like    string
	matches []string
}

// customerListKeys are the keys that yield an owner's customers in list order: the plain one when no filter narrows
// the set, a filter's own when one does. Left to itself the planner reads the owner/counterparty key, which pins the
// owner but not the order, and sorts every customer. PRIMARY is offered too, so a price group's relations (resolved
// by ListPriceGroupRelationIDs) are read by id rather than found by walking.
const customerListKeys = "account_relation_owner_role_created_idx, " +
	"account_relation_owner_role_group_created_idx, " +
	"account_relation_owner_role_rep_created_idx, " +
	"account_relation_owner_role_status_created_idx, " +
	"account_relation_owner_role_carrier_created_idx, " +
	"account_relation_owner_role_carrier_option_created_idx, " +
	"account_relation_owner_role_payment_term_created_idx, " +
	"account_relation_owner_role_shipping_term_created_idx, " +
	"PRIMARY"

// customerChildExists is true for a relation that some other relation names as its parent.
const customerChildExists = "EXISTS (SELECT 1 FROM account_relation car WHERE car.parent_account_relation_id = ar.id)"

// customerListPageQuery selects one page of customer account IDs in list order, emitting only the predicates the
// request uses. GetCustomersByIDs reads the page's rows.
//
// A search matches the displayed name (the alias, falling back to the account's name) or the number. A rare search
// reads its matches by id; a dense one is tested on each customer the walk in list order meets.
func customerListPageQuery(params domain.ListCustomersParams, search customerListSearch, pricingRelationIDs []string, cursor *pagination.StringCursor, limit int32) (string, []any) {
	var f listFilter
	f.add("ar.owner_account_id = ?", params.AccountID)
	f.add("ar.account_relation_role_code = 'customer'")

	join := ""
	if search.like != "" {
		join = "\nINNER JOIN account a ON a.id = ar.counterparty_account_id"
		f.in("ar.counterparty_account_id", search.matches)
		f.add("(COALESCE(NULLIF(ar.alias, ''), a.name) LIKE ? OR ar.external_number LIKE ?)", search.like, search.like)
	}
	f.in("ar.account_group_id", params.CustomerGroupIDs)
	f.in("ar.id", pricingRelationIDs)
	f.in("ar.default_sales_rep_id", params.SalesRepIDs)
	f.in("ar.account_status_code", params.StatusCodes)
	f.in("ar.shipping_term_id", params.ShippingTermIDs)
	f.in("ar.payment_term_id", params.PaymentTermIDs)
	f.in("ar.commission_status_code", commissionStatusCodes(params.CommissionPolicyCodes))
	f.in("ar.freight_status_code", freightStatusCodes(params.FreightPolicyCodes))
	f.in("ar.default_carrier_id", params.CarrierIDs)
	f.in("ar.default_carrier_option_id", params.ServiceLevelIDs)
	if params.IsActive != nil {
		f.add("ar.is_active = ?", *params.IsActive)
	}
	if params.IsParentAccount != nil {
		if *params.IsParentAccount {
			f.add(customerChildExists)
		} else {
			f.add("NOT " + customerChildExists)
		}
	}
	if clause := customerRelationshipClause(params.RelationshipTypes); clause != "" {
		f.add(clause)
	}
	if params.City != nil || params.State != nil || params.PostalCode != nil {
		// A probe of each customer's few addresses; as a semi-join the planner scans geolocation for each.
		var place []string
		var args []any
		for _, c := range []struct {
			column string
			value  *string
		}{{"g.locality", params.City}, {"g.state", params.State}, {"g.postal_code", params.PostalCode}} {
			if c.value != nil {
				place = append(place, c.column+" = ?")
				args = append(args, *c.value)
			}
		}
		f.add("(SELECT STRAIGHT_JOIN 1 FROM account_address aa"+
			" JOIN address addr ON addr.id = aa.address_id"+
			" JOIN geolocation g ON g.id = addr.geolocation_id"+
			" WHERE aa.account_id = ar.counterparty_account_id AND "+strings.Join(place, " AND ")+" LIMIT 1) IS NOT NULL", args...)
	}
	if params.StartDate != nil {
		f.add("ar.created_at >= ?", *params.StartDate)
	}
	if params.EndDate != nil {
		f.add("ar.created_at <= ?", *params.EndDate)
	}
	f.createdBetween("ar.created_at", params.Created)

	order := f.keyset(cursor, "ar.created_at", "ar.counterparty_account_id")
	return "SELECT STRAIGHT_JOIN ar.counterparty_account_id FROM account_relation ar FORCE INDEX (" + customerListKeys + ")" +
		join + f.whereSQL() + "\nORDER BY " + order + "\nLIMIT ?", append(f.args, limit)
}

// customerRelationshipClause matches customers of any of types: standalone (no parent, no children), parent (has
// children) or child (has a parent and no children of its own). Empty when types names none.
func customerRelationshipClause(types []constants.CustomerRelationshipType) string {
	var alts []string
	seen := map[constants.CustomerRelationshipType]bool{}
	for _, t := range types {
		if seen[t] {
			continue
		}
		seen[t] = true
		switch t {
		case constants.CustomerRelationshipTypeStandalone:
			alts = append(alts, "(ar.parent_account_relation_id IS NULL AND NOT "+customerChildExists+")")
		case constants.CustomerRelationshipTypeParent:
			alts = append(alts, customerChildExists)
		case constants.CustomerRelationshipTypeChild:
			alts = append(alts, "(ar.parent_account_relation_id IS NOT NULL AND NOT "+customerChildExists+")")
		}
	}
	if len(alts) == 0 {
		return ""
	}
	return "(" + strings.Join(alts, " OR ") + ")"
}
