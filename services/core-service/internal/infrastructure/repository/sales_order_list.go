package repository

import (
	"context"
	"strings"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/pagination"
)

// The sales order list's keys. Each leads with the owner and ends in list order, so a list on its
// filter reads a page and stops.
const (
	salesOrderCreatedIndex  = "sales_order_owner_created_idx"
	salesOrderStatusIndex   = "sales_order_owner_status_created_idx"
	salesOrderBuyerIndex    = "sales_order_owner_buyer_created_idx"
	salesOrderSalesRepIndex = "sales_order_owner_sales_rep_created_idx"
	// The ship-by keys range a ship-by window, which no list-order key can stop at.
	salesOrderShipByIndex       = "sales_order_owner_ship_by_idx"
	salesOrderStatusShipByIndex = "sales_order_owner_status_ship_by_idx"
)

// salesOrderListColumns is the GetSalesOrder projection, less the order discount's order count.
const salesOrderListColumns = `
    so.id, so.number, so.customer_po_number, so.note, so.is_acknowledgment_sent, so.billing_address_id,
    so.shipping_address_id, so.carrier_id, so.carrier_option_id, so.carrier_billing_type, so.carrier_billing_account,
    so.priority_code, so.sales_rep_id, so.shipping_term_id, so.sales_order_status_code, so.sales_order_type_code,
    so.payment_term_id, so.production_run_id, so.order_discount_id, so.buyer_account_id, so.seller_account_id,
    so.owner_account_id, so.issued_at, so.completed_at, so.first_ship_at, so.expired_at, so.promised_at,
    so.ship_by_date, so.lead_time_days, so.lead_time_source_code, so.transit_days, so.transit_source_code,
    so.lead_time_override_days, so.ship_by_override_date, so.ship_by_cutoff_at, so.calendar_adjustment_days,
    so.created_at, so.updated_at,
    ba.name, ar.external_number, ar.account_status_code, ar.commission_status_code, ar.created_at, ar.updated_at,
    sos.name, sot.name, pr.name, pr.id,
    bill_addr.name, bill_addr.is_drop_ship, bill_geo.id, bill_geo.street_line_1, bill_geo.street_line_2,
    bill_geo.locality, bill_geo.state, bill_geo.postal_code, bill_geo.country, bill_addr.phone, bill_addr.email,
    bill_addr.created_at, bill_addr.updated_at,
    ship_addr.name, ship_addr.is_drop_ship, ship_geo.id, ship_geo.street_line_1, ship_geo.street_line_2,
    ship_geo.locality, ship_geo.state, ship_geo.postal_code, ship_geo.country, ship_addr.phone, ship_addr.email,
    ship_addr.created_at, ship_addr.updated_at,
    cr.name, cr.is_portal_enabled, cr.created_at, cr.updated_at,
    co.name, co.is_portal_enabled, co.service_level_token, co.created_at, co.updated_at,
    sr_user.name,
    pt.name, pt.is_active, pt.created_at, pt.updated_at,
    st.name, st.is_freight_exempt, st.is_carrier_rate, st.created_at, st.updated_at,
    od.name, od.code, od.percentage, od.value, od.discount_type_code, od.created_at, od.updated_at,
    pk.id`

// salesOrderListJoins hydrate a chosen page. The customer relation and account are joined in the page
// query as well, where they decide which orders list at all.
const salesOrderListJoins = `
JOIN sales_order so ON so.id = page.id
JOIN account_relation ar ON ar.id = page.relation_id
JOIN account ba ON ba.id = so.buyer_account_id
JOIN sales_order_status sos ON sos.code = so.sales_order_status_code
JOIN sales_order_type sot ON sot.code = so.sales_order_type_code
JOIN priority pr ON pr.code = so.priority_code
LEFT JOIN address bill_addr ON bill_addr.id = so.billing_address_id
LEFT JOIN geolocation bill_geo ON bill_geo.id = bill_addr.geolocation_id
LEFT JOIN address ship_addr ON ship_addr.id = so.shipping_address_id
LEFT JOIN geolocation ship_geo ON ship_geo.id = ship_addr.geolocation_id
LEFT JOIN carrier cr ON cr.id = so.carrier_id
LEFT JOIN carrier_option co ON co.id = so.carrier_option_id
LEFT JOIN account_user sr_au ON sr_au.id = so.sales_rep_id
LEFT JOIN user sr_user ON sr_user.id = sr_au.user_id
LEFT JOIN payment_term pt ON pt.id = so.payment_term_id
LEFT JOIN shipping_term st ON st.id = so.shipping_term_id
LEFT JOIN order_discount od ON od.id = so.order_discount_id
LEFT JOIN pick pk ON pk.sales_order_id = so.id`

func (r *listSalesOrderRow) dest() []any {
	return []any{&r.ID, &r.Number, &r.CustomerPoNumber, &r.Note, &r.IsAcknowledgmentSent, &r.BillingAddressID,
		&r.ShippingAddressID, &r.CarrierID, &r.CarrierOptionID, &r.CarrierBillingType, &r.CarrierBillingAccount,
		&r.PriorityCode, &r.SalesRepID, &r.ShippingTermID, &r.SalesOrderStatusCode, &r.SalesOrderTypeCode,
		&r.PaymentTermID, &r.ProductionRunID, &r.OrderDiscountID, &r.BuyerAccountID, &r.SellerAccountID,
		&r.OwnerAccountID, &r.IssuedAt, &r.CompletedAt, &r.FirstShipAt, &r.ExpiredAt, &r.PromisedAt,
		&r.ShipByDate, &r.LeadTimeDays, &r.LeadTimeSourceCode, &r.TransitDays, &r.TransitSourceCode,
		&r.LeadTimeOverrideDays, &r.ShipByOverrideDate, &r.ShipByCutoffAt, &r.CalendarAdjustmentDays,
		&r.CreatedAt, &r.UpdatedAt,
		&r.CustomerName, &r.CustomerNumber, &r.CustomerStatusCode, &r.CustomerCommissionPolicy, &r.CustomerCreatedAt, &r.CustomerUpdatedAt,
		&r.StatusName, &r.TypeName, &r.PriorityName, &r.PriorityID,
		&r.BillToName, &r.BillToIsDropShip, &r.BillToGeolocationID, &r.BillToStreetLine1, &r.BillToStreetLine2,
		&r.BillToLocality, &r.BillToState, &r.BillToPostalCode, &r.BillToCountry, &r.BillToPhone, &r.BillToEmail,
		&r.BillToCreatedAt, &r.BillToUpdatedAt,
		&r.ShipToName, &r.ShipToIsDropShip, &r.ShipToGeolocationID, &r.ShipToStreetLine1, &r.ShipToStreetLine2,
		&r.ShipToLocality, &r.ShipToState, &r.ShipToPostalCode, &r.ShipToCountry, &r.ShipToPhone, &r.ShipToEmail,
		&r.ShipToCreatedAt, &r.ShipToUpdatedAt,
		&r.CarrierName, &r.CarrierIsPortalEnabled, &r.CarrierCreatedAt, &r.CarrierUpdatedAt,
		&r.CarrierOptionName, &r.ServiceLevelIsPortalEnabled, &r.ServiceLevelToken, &r.ServiceLevelCreatedAt, &r.ServiceLevelUpdatedAt,
		&r.SalesRepName,
		&r.PaymentTermName, &r.PaymentTermIsActive, &r.PaymentTermCreatedAt, &r.PaymentTermUpdatedAt,
		&r.ShippingTermName, &r.ShippingTermIsFreightExempt, &r.ShippingTermIsCarrierRate, &r.ShippingTermCreatedAt, &r.ShippingTermUpdatedAt,
		&r.OrderDiscountName, &r.OrderDiscountCode, &r.OrderDiscountPercentage, &r.OrderDiscountAmount,
		&r.OrderDiscountDiscountType, &r.OrderDiscountCreatedAt, &r.OrderDiscountUpdatedAt,
		&r.PickID}
}

// salesOrderListFilter is the WHERE of a sales order list page, on so (sales_order) and ar (the
// buyer's relation to the owner). productIDs are the products of params.ProductLineIDs
// (productsInLines), and groupBuyers the buyers related to the owner in params.CustomerGroupIDs.
func salesOrderListFilter(params domain.ListSalesOrdersParams, productIDs, groupBuyers []string) *listFilter {
	f := &listFilter{}
	f.add("so.owner_account_id = ?", params.AccountID)
	f.add("so.seller_account_id = so.owner_account_id")
	if params.BuyerAccountID != nil {
		f.add("so.buyer_account_id = ?", *params.BuyerAccountID)
	}
	if includeStatus, statusCodes, _, _, _, _, _, _, _, _, _, _ := buildSalesOrderListFilters(params); includeStatus {
		f.in("so.sales_order_status_code", statusCodes)
	}
	lineQueries, lineArgs := orderLineSubqueries(params.ItemIDs, productIDs)
	for i, q := range lineQueries {
		f.add("so.id IN ("+q+")", lineArgs[i]...)
	}
	f.in("so.buyer_account_id", params.CustomerIDs)
	f.in("ar.account_group_id", params.CustomerGroupIDs)
	// The group again, as its buyers, so the buyer key can read it.
	f.in("so.buyer_account_id", groupBuyers)
	f.in("so.sales_rep_id", params.SalesRepIDs)
	if d := parseDateString(params.StartDate); d.Valid {
		f.add("so.created_at >= ?", d.Time)
	}
	if d := parseDateString(params.EndDate); d.Valid {
		f.add("so.created_at <= ?", d.Time)
	}
	if d := parseDateString(params.ShipByAfter); d.Valid {
		f.add("so.ship_by_date >= ?", d.Time)
	}
	if d := parseDateString(params.ShipByBefore); d.Valid {
		f.add("so.ship_by_date <= ?", d.Time)
	}
	// Past due is a fact about work still owed, so it is scoped to issued orders. A fulfilled order that
	// shipped late is a delivery-performance question, not a backlog one.
	const pastDue = "so.ship_by_date IS NOT NULL AND so.ship_by_date < CURDATE() AND so.sales_order_status_code = 'issued'"
	if params.PastDue != nil {
		if *params.PastDue {
			f.add(pastDue)
		} else {
			f.add("NOT (" + pastDue + ")")
		}
	}
	return f
}

// salesOrderKeyFilter is a filter a key can read: an equality filter on column with a list-order key
// of its own, or (column "") an order line filter, read from its lines and looked up by primary key.
type salesOrderKeyFilter struct {
	index, column string
	values        []string
	// resolved is set on a filter whose values were looked up (a group's buyers): however few, the
	// planner cannot see their count.
	resolved bool
	// lineQuery selects the ids of the orders the line filter admits (args lineArgs).
	lineQuery string
	lineArgs  []any
}

// salesOrderListIndexHint is the keys a sales order page may be read from. Left to itself, the planner
// picks a single-column key and sorts every match, or walks created_at past every row a rare filter
// rejects.
//   - A ship-by window filters a column the list does not sort by, so no key can stop at a page: the
//     ship-by keys read just the window, and a customer or sales rep key is offered for one narrower
//     than it, and the primary key for a line filter's orders to be looked up by.
//   - Otherwise each single-valued filter's key both narrows and orders, and the planner picks among
//     them. The created_at key is offered only alone: forced beside another, the planner may swap to
//     it for the order and walk it from the account's far end, past a deep page's cursor.
//   - A multi-valued filter (several values, past due's issued orders beside a status list, a group's
//     buyers) has no key that yields it in list order, the planner cannot tell a rare one from a common
//     one, and neither can it for an item or product line. Their presence returns every filter as
//     counted, for salesOrderCountedHint to settle by counting; indexes is then the fallback for when
//     all are common.
func salesOrderListIndexHint(params domain.ListSalesOrdersParams, productIDs, groupBuyers []string) (indexes []string, counted []salesOrderKeyFilter) {
	lineQueries, lineArgs := orderLineSubqueries(params.ItemIDs, productIDs)
	if parseDateString(params.ShipByAfter).Valid || parseDateString(params.ShipByBefore).Valid {
		indexes = []string{salesOrderShipByIndex, salesOrderStatusShipByIndex}
		if len(params.CustomerIDs) > 0 || params.BuyerAccountID != nil || len(groupBuyers) > 0 {
			indexes = append(indexes, salesOrderBuyerIndex)
		}
		if len(params.SalesRepIDs) > 0 {
			indexes = append(indexes, salesOrderSalesRepIndex)
		}
		if len(lineQueries) > 0 {
			indexes = append(indexes, "PRIMARY")
		}
		return indexes, nil
	}

	var filters []salesOrderKeyFilter
	includeStatus, statusCodes, _, _, _, _, _, _, _, _, _, _ := buildSalesOrderListFilters(params)
	if params.PastDue != nil && *params.PastDue {
		// Past due admits issued orders only, whatever the status list.
		includeStatus, statusCodes = true, []string{"issued"}
	}
	if includeStatus {
		filters = append(filters, salesOrderKeyFilter{index: salesOrderStatusIndex, column: "sales_order_status_code", values: statusCodes})
	}
	if params.BuyerAccountID != nil {
		filters = append(filters, salesOrderKeyFilter{index: salesOrderBuyerIndex, column: "buyer_account_id", values: []string{*params.BuyerAccountID}})
	}
	for _, kf := range []salesOrderKeyFilter{
		{index: salesOrderBuyerIndex, column: "buyer_account_id", values: params.CustomerIDs},
		{index: salesOrderBuyerIndex, column: "buyer_account_id", values: groupBuyers, resolved: true},
		{index: salesOrderSalesRepIndex, column: "sales_rep_id", values: params.SalesRepIDs},
	} {
		if len(kf.values) > 0 {
			filters = append(filters, kf)
		}
	}
	for i, q := range lineQueries {
		filters = append(filters, salesOrderKeyFilter{index: "PRIMARY", lineQuery: q, lineArgs: lineArgs[i]})
	}

	countNeeded := false
	for _, kf := range filters {
		switch {
		case kf.lineQuery != "" || kf.resolved || len(kf.values) > 1:
			countNeeded = true
		case !containsString(indexes, kf.index):
			indexes = append(indexes, kf.index)
		}
	}
	if len(indexes) == 0 {
		indexes = []string{salesOrderCreatedIndex}
	}
	if countNeeded {
		return indexes, filters
	}
	return indexes, nil
}

// salesOrderCountedRatio sets how many matches, in pages, make a filter common.
const salesOrderCountedRatio = 40

// salesOrderCountedHint picks the key of the filter matching the fewest orders: a single-valued one's
// key stops at the page, a multi-valued one's (or a line filter's, through its orders' primary keys) is
// read whole and sorted, and either reads no more than the filter matches. When every filter matches
// many orders, they are common enough that fallback finds a page quickly in list order. Each count is
// capped, reading at most that many index entries.
func (r *salesOrderRepoImpl) salesOrderCountedHint(ctx context.Context, accountID string, limit int32, filters []salesOrderKeyFilter, fallback []string) ([]string, error) {
	capped := int64(salesOrderCountedRatio * (limit + 1))
	best, bestCount := "", capped
	for _, kf := range filters {
		var query string
		var args []any
		if kf.lineQuery != "" {
			// Lines, not orders: an order with several matching lines counts more than once, which only
			// makes a line filter look commoner than it is.
			query, args = "SELECT COUNT(*) FROM ("+kf.lineQuery+" LIMIT ?) matches", append(append([]any{}, kf.lineArgs...), capped)
		} else {
			query = "SELECT COUNT(*) FROM (SELECT 1 FROM sales_order FORCE INDEX (" + kf.index + ") WHERE owner_account_id = ? AND " +
				kf.column + " IN (" + placeholders(len(kf.values)) + ") LIMIT ?) matches"
			args = append(append([]any{accountID}, stringArgs(kf.values)...), capped)
		}
		var n int64
		if err := r.queries.DB().QueryRowContext(ctx, query, args...).Scan(&n); err != nil {
			return nil, err
		}
		if n < bestCount {
			best, bestCount = kf.index, n
		}
	}
	if best == "" {
		return fallback, nil
	}
	return []string{best}, nil
}

// listPage reads one page of sales orders. The page is chosen from sales_order and the buyer's
// relation alone and joined after, so a filter no key serves in list order reads only its matches
// rather than its matches through every join.
func (r *salesOrderRepoImpl) listPage(ctx context.Context, params domain.ListSalesOrdersParams, cursor *pagination.StringCursor) ([]*domain.SalesOrder, error) {
	var productIDs []string
	if len(params.ProductLineIDs) > 0 {
		var err error
		if productIDs, err = productsInLines(ctx, r.queries.DB(), params.ProductLineIDs); err != nil || len(productIDs) == 0 {
			return nil, err
		}
	}
	var groupBuyers []string
	if len(params.CustomerGroupIDs) > 0 {
		var err error
		groupBuyers, err = selectStrings(ctx, r.queries.DB(), "SELECT counterparty_account_id FROM account_relation WHERE owner_account_id = ? AND account_group_id IN ("+
			placeholders(len(params.CustomerGroupIDs))+")", append([]any{params.AccountID}, stringArgs(params.CustomerGroupIDs)...)...)
		if err != nil || len(groupBuyers) == 0 {
			return nil, err
		}
	}
	indexes, counted := salesOrderListIndexHint(params, productIDs, groupBuyers)
	if len(counted) > 0 {
		var err error
		if indexes, err = r.salesOrderCountedHint(ctx, params.AccountID, params.Limit, counted, indexes); err != nil {
			return nil, err
		}
	}
	f := salesOrderListFilter(params, productIDs, groupBuyers)
	orderBy := f.keyset(cursor, "so.created_at", "so.id")

	var sb strings.Builder
	sb.WriteString("SELECT")
	sb.WriteString(salesOrderListColumns)
	// JOIN_SUFFIX keeps the relation and account after the order: driven from the relations, a group
	// reads every order of its customers to sort them.
	sb.WriteString("\nFROM (SELECT /*+ JOIN_SUFFIX(ar, ba) */ so.id, ar.id AS relation_id FROM sales_order so FORCE INDEX (")
	sb.WriteString(strings.Join(indexes, ", "))
	sb.WriteString(")\nJOIN account_relation ar ON ar.owner_account_id = so.owner_account_id AND ar.counterparty_account_id = so.buyer_account_id")
	sb.WriteString("\nJOIN account ba ON ba.id = so.buyer_account_id")
	sb.WriteString(f.whereSQL())
	sb.WriteString("\nORDER BY " + orderBy + "\nLIMIT ?) page")
	sb.WriteString(salesOrderListJoins)
	sb.WriteString("\nORDER BY " + orderBy)
	args := append(f.args, params.Limit+1)

	rows, err := r.queries.DB().QueryContext(ctx, sb.String(), args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*domain.SalesOrder
	for rows.Next() {
		var row listSalesOrderRow
		if err := rows.Scan(row.dest()...); err != nil {
			return nil, err
		}
		out = append(out, mapListSalesOrderRow(row))
	}
	return out, rows.Err()
}

func containsString(values []string, v string) bool {
	for _, s := range values {
		if s == v {
			return true
		}
	}
	return false
}
