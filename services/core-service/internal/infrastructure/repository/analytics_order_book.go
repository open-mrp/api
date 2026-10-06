package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/pagination"
	"github.com/open-mrp/api/shared/textutil"
	"github.com/open-mrp/api/shared/tracing"
)

// The order-book reports (order data, open orders) price and count each sale line exactly as the dashboard's analytics did: quantities in the item's base unit, money as the line's quantity in its dimension's base unit times the price per that unit, unrounded.
const (
	obQtyOrderedNorm   = "((q_ord.value * (u_ord.ratio_numerator / u_ord.ratio_denominator)) + (u_ord.offset_numerator / u_ord.offset_denominator))"
	obQtyInvoicedNorm  = "COALESCE(inv.qty_inv_norm, 0)"
	obBaseOffset       = "(bu_unit.offset_numerator / bu_unit.offset_denominator)"
	obBaseRatio        = "NULLIF((bu_unit.ratio_numerator / bu_unit.ratio_denominator), 0)"
	obQtyOrdered       = "((" + obQtyOrderedNorm + " - " + obBaseOffset + ") / " + obBaseRatio + ")"
	obQtyInvoiced      = "((" + obQtyInvoicedNorm + " - " + obBaseOffset + ") / " + obBaseRatio + ")"
	obQtyBackOrdered   = "(" + obQtyOrdered + " - " + obQtyInvoiced + ")"
	obPricePerBase     = "(((r_price.value * (u_price_num.ratio_numerator / u_price_num.ratio_denominator)) + (u_price_num.offset_numerator / u_price_num.offset_denominator)) / NULLIF(((u_price_den.ratio_numerator / u_price_den.ratio_denominator) + (u_price_den.offset_numerator / u_price_den.offset_denominator)), 0))"
	obCostPerBase      = "(((COALESCE(r_cost.value, 0) * (COALESCE(u_cost_num.ratio_numerator, 1) / COALESCE(u_cost_num.ratio_denominator, 1))) + (COALESCE(u_cost_num.offset_numerator, 0) / COALESCE(u_cost_num.offset_denominator, 1))) / NULLIF(((COALESCE(u_cost_den.ratio_numerator, 1) / COALESCE(u_cost_den.ratio_denominator, 1)) + (COALESCE(u_cost_den.offset_numerator, 0) / COALESCE(u_cost_den.offset_denominator, 1))), 0))"
	obTotalOrdered     = "(" + obQtyOrderedNorm + " * " + obPricePerBase + ")"
	obTotalInvoiced    = "(" + obQtyInvoicedNorm + " * " + obPricePerBase + ")"
	obTotalBackOrdered = "((" + obQtyOrderedNorm + " - " + obQtyInvoicedNorm + ") * " + obPricePerBase + ")"
	obTotalCost        = "(" + obQtyInvoicedNorm + " * " + obCostPerBase + ")"
)

// orderBookScope is the sale lines of an account's issued sales orders that one order-book report reads. Only the filters set emit a predicate.
type orderBookScope struct {
	accountID string
	// anyStatus reads estimates and fulfilled orders too, not only issued ones.
	anyStatus bool
	// openOnly keeps orders not yet completed.
	openOnly bool
	// orderIDs, when set, reads only these orders.
	orderIDs []string
	// buyers is the customer and customer-group filters resolved to the buyers they admit (resolveCustomerBuyers); applied when buyersFiltered.
	buyers         []string
	buyersFiltered bool
	salesRepIDs    []string
	productLineIDs []string
	itemIDs        []string
	// orderIndex is the sales_order key the orders are read through; empty leaves it to the planner.
	orderIndex string
}

// orderPredicates scope the orders read through alias: the account's issued sales orders, narrowed by the order-level filters.
func (s orderBookScope) orderPredicates(alias string) ([]string, []any) {
	preds := []string{
		alias + ".owner_account_id = ?",
		alias + ".sales_order_type_code = 'sales_order'",
	}
	args := []any{s.accountID}
	if !s.anyStatus {
		preds = append(preds, alias+".sales_order_status_code = 'issued'")
	}
	if s.openOnly {
		preds = append(preds, alias+".completed_at IS NULL")
	}
	if len(s.orderIDs) > 0 {
		preds = append(preds, alias+".id IN ("+placeholders(len(s.orderIDs))+")")
		args = append(args, stringsToAny(s.orderIDs)...)
	}
	if s.buyersFiltered {
		preds = append(preds, alias+".buyer_account_id IN ("+placeholders(len(s.buyers))+")")
		args = append(args, stringsToAny(s.buyers)...)
	}
	if len(s.salesRepIDs) > 0 {
		preds = append(preds, alias+".sales_rep_id IN ("+placeholders(len(s.salesRepIDs))+")")
		args = append(args, stringsToAny(s.salesRepIDs)...)
	}
	return preds, args
}

// linePredicates choose which of an order's lines count: sale products, narrowed by the product filters.
func (s orderBookScope) linePredicates() ([]string, []any) {
	preds := []string{"fg.product_type_code = 'sale'"}
	var args []any
	if len(s.productLineIDs) > 0 {
		preds = append(preds, "fg.product_line_id IN ("+placeholders(len(s.productLineIDs))+")")
		args = append(args, stringsToAny(s.productLineIDs)...)
	}
	if len(s.itemIDs) > 0 {
		preds = append(preds, "fg.item_id IN ("+placeholders(len(s.itemIDs))+")")
		args = append(args, stringsToAny(s.itemIDs)...)
	}
	return preds, args
}

// fromWhere is the FROM and WHERE every order-book read shares. Selects over it are STRAIGHT_JOIN so the orders drive: left to choose, the planner starts from a product line's every line ever ordered.
//
// The invoiced quantities are totalled in a derived table the outer filters cannot reach, so it is held to the same orders; left alone it groups every invoice line in the database.
func (s orderBookScope) fromWhere(extraJoins string) (string, []any) {
	invPreds, invArgs := s.orderPredicates("inv_so")
	orderPreds, orderArgs := s.orderPredicates("so")
	linePreds, lineArgs := s.linePredicates()

	from := `FROM sales_order so` + s.orderIndexHint() + `
JOIN sales_order_line sol ON sol.sales_order_id = so.id
JOIN product fg ON fg.id = sol.product_id
LEFT JOIN item pb ON pb.id = fg.item_id
LEFT JOIN item_category ic ON ic.id = pb.item_category_id
LEFT JOIN unit_group ug ON ug.id = ic.unit_group_id
LEFT JOIN unit bu_unit ON bu_unit.id = ug.base_unit_id
LEFT JOIN quantity q_ord ON q_ord.id = sol.quantity_id
LEFT JOIN unit u_ord ON u_ord.id = q_ord.unit_id
LEFT JOIN rate r_price ON r_price.id = sol.unit_price_id
LEFT JOIN unit u_price_num ON u_price_num.id = r_price.numerator_unit_id
LEFT JOIN unit u_price_den ON u_price_den.id = r_price.denominator_unit_id
LEFT JOIN (
    SELECT il.sales_order_line_id AS line_id,
        SUM((q_in.value * (u_in.ratio_numerator / u_in.ratio_denominator)) + (u_in.offset_numerator / u_in.offset_denominator)) AS qty_inv_norm
    FROM invoice_line il
    JOIN quantity q_in ON q_in.id = il.quantity_id
    JOIN unit u_in ON u_in.id = q_in.unit_id
    WHERE il.sales_order_line_id IN (
        SELECT inv_sol.id
        FROM sales_order inv_so
        JOIN sales_order_line inv_sol ON inv_sol.sales_order_id = inv_so.id
        WHERE ` + strings.Join(invPreds, " AND ") + `
    )
    GROUP BY il.sales_order_line_id
) inv ON inv.line_id = sol.id
` + extraJoins + `
WHERE ` + strings.Join(append(orderPreds, linePreds...), " AND ")

	args := append(append(invArgs, orderArgs...), lineArgs...)
	return from, args
}

func (s orderBookScope) orderIndexHint() string {
	if s.orderIndex == "" {
		return ""
	}
	return " FORCE INDEX (" + s.orderIndex + ")"
}

// keyRange is one key a report could read its rows through, and the predicate it ranges, written against the alias kso.
type keyRange struct {
	index string
	where string
	args  []any
}

// keyCountCap bounds the counts cheapestKey compares, so sizing a key never reads more than the largest report would.
const keyCountCap = 200_000

// cheapestKey picks whichever of table's keys reaches the fewest rows. Left to choose, the planner reads a buyer's orders with every seller, or the account's every row, and no one key suits both a narrow window and a narrow filter. Each count reads index entries only and stops one past the best so far.
func cheapestKey(ctx context.Context, q sqlc.DBTX, table string, keys []keyRange) (string, *apierror.APIError) {
	best, bestCount := "", keyCountCap+1
	for _, k := range keys {
		var n int
		err := q.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM (SELECT 1 FROM "+table+" kso FORCE INDEX ("+k.index+") WHERE "+k.where+" LIMIT ?) n",
			append(append([]any{}, k.args...), bestCount)...).Scan(&n)
		if err != nil {
			return "", db.MapSQLError(err)
		}
		if best == "" || n < bestCount {
			best, bestCount = k.index, n
		}
	}
	return best, nil
}

func (r *analyticsRepoImpl) cheapestOrderKey(ctx context.Context, keys []keyRange) (string, *apierror.APIError) {
	return cheapestKey(ctx, r.queries.DB(), "sales_order", keys)
}

// orderKeys are the keys an order-book scope can read its orders through: its status, and its buyers or reps.
func (s orderBookScope) orderKeys() []keyRange {
	keys := []keyRange{{
		index: "sales_order_owner_type_status_created_idx",
		where: "kso.owner_account_id = ? AND kso.sales_order_type_code = 'sales_order' AND kso.sales_order_status_code = 'issued'",
		args:  []any{s.accountID},
	}}
	if s.anyStatus {
		keys[0] = keyRange{index: "sales_order_owner_type_created_idx", where: "kso.owner_account_id = ? AND kso.sales_order_type_code = 'sales_order'", args: []any{s.accountID}}
	}
	return append(keys, filterOrderKeys(s.accountID, s.buyers, s.buyersFiltered, s.salesRepIDs)...)
}

// filterOrderKeys are the buyer and rep keys a filtered report can read its orders through.
func filterOrderKeys(accountID string, buyers []string, buyersFiltered bool, salesRepIDs []string) []keyRange {
	var keys []keyRange
	if buyersFiltered && len(buyers) > 0 {
		keys = append(keys, keyRange{
			index: "sales_order_owner_buyer_created_idx",
			where: "kso.owner_account_id = ? AND kso.buyer_account_id IN (" + placeholders(len(buyers)) + ")",
			args:  append([]any{accountID}, stringsToAny(buyers)...),
		})
	}
	if len(salesRepIDs) > 0 {
		keys = append(keys, keyRange{
			index: "sales_order_owner_sales_rep_created_idx",
			where: "kso.owner_account_id = ? AND kso.sales_rep_id IN (" + placeholders(len(salesRepIDs)) + ")",
			args:  append([]any{accountID}, stringsToAny(salesRepIDs)...),
		})
	}
	return keys
}

// newOrderBookScope resolves the customer filters to buyers. empty reports a filter that admits no buyer, which no order can match.
func (r *analyticsRepoImpl) newOrderBookScope(ctx context.Context, f domain.OpenOrderFilter, openOnly bool) (scope orderBookScope, empty bool, apiErr *apierror.APIError) {
	buyers, filtered, apiErr := resolveCustomerBuyers(ctx, r.queries.DB(), f.AccountID, f.CustomerIDs, f.CustomerGroupIDs)
	if apiErr != nil {
		return orderBookScope{}, false, apiErr
	}
	scope = orderBookScope{
		accountID:      f.AccountID,
		openOnly:       openOnly,
		buyers:         buyers,
		buyersFiltered: filtered,
		salesRepIDs:    f.SalesRepIDs,
		productLineIDs: f.ProductLineIDs,
		itemIDs:        f.ItemIDs,
	}
	if filtered && len(buyers) == 0 {
		return scope, true, nil
	}
	if scope.orderIndex, apiErr = r.cheapestOrderKey(ctx, scope.orderKeys()); apiErr != nil {
		return orderBookScope{}, false, apiErr
	}
	return scope, false, nil
}

// --- Order entries (order data, and the open-order-lines export) ---

// orderEntryJoins are the order, customer and cost details an order entry carries beyond the shared fragment.
const orderEntryJoins = `LEFT JOIN product_line pl ON pl.id = fg.product_line_id
LEFT JOIN account buyer ON buyer.id = so.buyer_account_id
LEFT JOIN account_relation ar ON ar.owner_account_id = so.owner_account_id AND ar.counterparty_account_id = so.buyer_account_id AND ar.account_relation_role_code = 'customer'
LEFT JOIN account_relation parent_ar ON parent_ar.id = ar.parent_account_relation_id AND parent_ar.owner_account_id = ar.owner_account_id
LEFT JOIN account_group ag ON ag.id = ar.account_group_id
LEFT JOIN address shipping_address ON shipping_address.id = so.shipping_address_id
LEFT JOIN geolocation shipping_geolocation ON shipping_geolocation.id = shipping_address.geolocation_id
LEFT JOIN account_user ou ON ou.id = so.sales_rep_id
LEFT JOIN ` + "`user`" + ` bu ON bu.id = ou.user_id
LEFT JOIN order_discount od ON od.id = so.order_discount_id
LEFT JOIN rate r_cost ON r_cost.id = sol.unit_cost_id
LEFT JOIN unit u_cost_num ON u_cost_num.id = r_cost.numerator_unit_id
LEFT JOIN unit u_cost_den ON u_cost_den.id = r_cost.denominator_unit_id`

// orderEntryColumns is an order entry's select list. Nothing invoiced yet leaves the per-unit price, cost and profit undefined (x / 0): they read 0, as the dashboard's rows did.
var orderEntryColumns = strings.Join([]string{
	"sol.id", "so.issued_at", "so.completed_at", "so.first_ship_at", "so.promised_at", "so.customer_po_number", "so.number", "so.id",
	"so.sales_rep_id", "bu.username", "so.buyer_account_id", "parent_ar.counterparty_account_id", "buyer.name", "ar.external_number",
	"buyer.created_at", "ar.account_group_id", "ag.name", "fg.product_line_id", "fg.product_type_code", "pb.id", "pb.sku", "pb.description",
	"ic.name", "pl.name",
	obDecimal(obQtyOrdered), obDecimal(obQtyInvoiced), obDecimal(obQtyBackOrdered), "bu_unit.abbreviation",
	obDecimal(obTotalOrdered), obDecimal(obTotalInvoiced), obDecimal(obTotalBackOrdered), obDecimal(obTotalCost),
	obDecimal(obTotalInvoiced + " - " + obTotalCost),
	"COALESCE(" + obDecimal(obTotalInvoiced+" / NULLIF("+obQtyInvoiced+", 0)") + ", 0)",
	"COALESCE(" + obDecimal(obTotalCost+" / NULLIF("+obQtyInvoiced+", 0)") + ", 0)",
	"COALESCE(" + obDecimal("("+obTotalInvoiced+" - "+obTotalCost+") / NULLIF("+obQtyInvoiced+", 0)") + ", 0)",
	"shipping_geolocation.state", "shipping_geolocation.locality", "shipping_geolocation.postal_code", "shipping_geolocation.country", "od.code",
}, ",\n    ")

func obDecimal(expr string) string {
	return "CAST(" + expr + " AS DECIMAL(65,30))"
}

// orderEntriesQuery reads every scoped line, oldest order first.
func orderEntriesQuery(s orderBookScope, limit int) (string, []any) {
	from, args := s.fromWhere(orderEntryJoins)
	query := "SELECT STRAIGHT_JOIN\n    " + orderEntryColumns + "\n" + from + "\nORDER BY so.issued_at ASC, so.id ASC, sol.id ASC"
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}
	return query, args
}

func (r *analyticsRepoImpl) GetOrderEntries(ctx context.Context, params domain.AnalyzeOrdersParams) ([]domain.OrderEntry, *apierror.APIError) {
	ctx, span := analyticsRepoTracer.Start(ctx, "repository.analytics.get_order_entries")
	defer span.End()

	scope, empty, apiErr := r.newOrderBookScope(ctx, domain.OpenOrderFilter{
		AccountID:        params.AccountID,
		CustomerIDs:      params.CustomerIDs,
		CustomerGroupIDs: params.CustomerGroupIDs,
		SalesRepIDs:      params.SalesRepIDs,
		ProductLineIDs:   params.ProductLineIDs,
	}, false)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if empty {
		return []domain.OrderEntry{}, nil
	}
	entries, apiErr := r.queryOrderEntries(ctx, scope, 0)
	return entries, tracing.Trace(span, apiErr)
}

// GetOpenOrderLineEntries reads the open lines as order entries, oldest order first, at most limit of them.
func (r *analyticsRepoImpl) GetOpenOrderLineEntries(ctx context.Context, f domain.OpenOrderFilter, limit int) ([]domain.OrderEntry, *apierror.APIError) {
	ctx, span := analyticsRepoTracer.Start(ctx, "repository.analytics.get_open_order_line_entries")
	defer span.End()

	scope, empty, apiErr := r.newOrderBookScope(ctx, f, true)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if empty {
		return []domain.OrderEntry{}, nil
	}
	entries, apiErr := r.queryOrderEntries(ctx, scope, limit)
	return entries, tracing.Trace(span, apiErr)
}

func (r *analyticsRepoImpl) queryOrderEntries(ctx context.Context, scope orderBookScope, limit int) ([]domain.OrderEntry, *apierror.APIError) {
	query, args := orderEntriesQuery(scope, limit)
	rows, err := r.queries.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, db.MapSQLError(err)
	}
	defer rows.Close()

	entries := []domain.OrderEntry{}
	for rows.Next() {
		var (
			e                                                             domain.OrderEntry
			issuedAt, completedAt, firstShipAt, promisedAt, customerSince sql.NullTime
			customerPO, salesRepID, salesRepUsername, parentCustomerID    sql.NullString
			customerName, customerNumber, groupID, groupName              sql.NullString
			productLineID, description, categoryName, productLine, unit   sql.NullString
			itemID, sku                                                   sql.NullString
			qtyOrdered, qtyInvoiced, qtyBackOrdered                       sql.NullString
			totalOrdered, totalInvoiced, totalBackOrdered, totalCost      sql.NullString
			totalProfit, unitPrice, unitCost, unitProfit                  sql.NullString
			state, city, postal, country, discount                        sql.NullString
		)
		if err := rows.Scan(&e.ID, &issuedAt, &completedAt, &firstShipAt, &promisedAt, &customerPO, &e.OrderNumber, &e.OrderID,
			&salesRepID, &salesRepUsername, &e.CustomerID, &parentCustomerID, &customerName, &customerNumber,
			&customerSince, &groupID, &groupName, &productLineID, &e.ProductTypeCode, &itemID, &sku, &description,
			&categoryName, &productLine,
			&qtyOrdered, &qtyInvoiced, &qtyBackOrdered, &unit,
			&totalOrdered, &totalInvoiced, &totalBackOrdered, &totalCost, &totalProfit,
			&unitPrice, &unitCost, &unitProfit,
			&state, &city, &postal, &country, &discount); err != nil {
			return nil, db.MapSQLError(err)
		}
		e.IssuedAt, e.CompletedAt, e.FirstShipAt, e.PromisedAt = nullTimePtr(issuedAt), nullTimePtr(completedAt), nullTimePtr(firstShipAt), nullTimePtr(promisedAt)
		e.OrderNumber = textutil.FormatRecordNumber(e.OrderNumber)
		e.CustomerPO, e.SalesRepID, e.SalesRepUsername, e.ParentCustomerID = nullStringPtr(customerPO), nullStringPtr(salesRepID), nullStringPtr(salesRepUsername), nullStringPtr(parentCustomerID)
		e.CustomerName, e.CustomerNumber = customerName.String, customerNumber.String
		if customerSince.Valid {
			e.CustomerCreatedAt = customerSince.Time
		}
		e.CustomerTypeGroupID, e.CustomerGroupName = nullStringPtr(groupID), nullStringPtr(groupName)
		e.ProductLineID, e.ItemID, e.ProductSku = nullStringPtr(productLineID), itemID.String, sku.String
		e.ProductDescription, e.CategoryName, e.ProductLine, e.Unit = nullStringPtr(description), categoryName.String, nullStringPtr(productLine), unit.String
		e.QuantityOrdered, e.QuantityInvoiced, e.QuantityBackOrdered = decimalToFloat(qtyOrdered), decimalToFloat(qtyInvoiced), decimalToFloat(qtyBackOrdered)
		e.TotalOrdered, e.TotalInvoiced, e.TotalBackOrdered = decimalToFloat(totalOrdered), decimalToFloat(totalInvoiced), decimalToFloat(totalBackOrdered)
		e.TotalCost, e.TotalProfit = decimalToFloat(totalCost), decimalToFloat(totalProfit)
		e.UnitPrice, e.UnitCost, e.UnitProfit = decimalToFloat(unitPrice), decimalToFloat(unitCost), decimalToFloat(unitProfit)
		e.ShipToState, e.ShipToCity, e.ShipToZipcode, e.ShipToCountry = nullStringPtr(state), nullStringPtr(city), nullStringPtr(postal), nullStringPtr(country)
		e.OrderDiscountCode = nullStringPtr(discount)
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, db.MapSQLError(err)
	}
	return entries, nil
}

// --- Open orders ---

func openOrdersSummaryQuery(s orderBookScope) (string, []any) {
	from, args := s.fromWhere("")
	return fmt.Sprintf(`SELECT STRAIGHT_JOIN
    %s,
    %s,
    %s
%s`, obDecimal("COALESCE(SUM("+obTotalOrdered+"), 0)"), obDecimal("COALESCE(SUM("+obTotalBackOrdered+"), 0)"),
		obDecimal("COALESCE(SUM("+obTotalInvoiced+"), 0)"), from), args
}

func (r *analyticsRepoImpl) GetOpenOrdersSummary(ctx context.Context, f domain.OpenOrderFilter) (*domain.OpenOrdersSummary, *apierror.APIError) {
	ctx, span := analyticsRepoTracer.Start(ctx, "repository.analytics.get_open_orders_summary")
	defer span.End()

	scope, empty, apiErr := r.newOrderBookScope(ctx, f, true)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if empty {
		return &domain.OpenOrdersSummary{Ordered: "0", BackOrdered: "0", Invoiced: "0"}, nil
	}
	query, args := openOrdersSummaryQuery(scope)
	var ordered, backOrdered, invoiced sql.NullString
	if err := r.queries.DB().QueryRowContext(ctx, query, args...).Scan(&ordered, &backOrdered, &invoiced); err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	return &domain.OpenOrdersSummary{Ordered: exactDecimal(ordered), BackOrdered: exactDecimal(backOrdered), Invoiced: exactDecimal(invoiced)}, nil
}

// openOrderProductsQuery totals the open lines per item and pages the items with any quantity ordered, most back-ordered first, then by item. A page is a keyset over that ranking compared as exact decimals; a backward page reads it in reverse.
func openOrderProductsQuery(s orderBookScope, cur *pagination.ValueCursor, limit int32) (string, []any) {
	from, args := s.fromWhere("")
	having := "g.qty_ordered > 0"
	order := "g.qty_back_ordered DESC, g.item_id ASC"
	if cur != nil {
		if cur.Direction == pagination.DirectionBackward {
			having += " AND (g.qty_back_ordered > CAST(? AS DECIMAL(65,30)) OR (g.qty_back_ordered = CAST(? AS DECIMAL(65,30)) AND g.item_id < ?))"
			order = "g.qty_back_ordered ASC, g.item_id DESC"
		} else {
			having += " AND (g.qty_back_ordered < CAST(? AS DECIMAL(65,30)) OR (g.qty_back_ordered = CAST(? AS DECIMAL(65,30)) AND g.item_id > ?))"
		}
		args = append(args, cur.Value, cur.Value, cur.ID)
	}
	query := fmt.Sprintf(`SELECT g.item_id, g.sku, g.description, g.unit_id, g.qty_ordered, g.qty_back_ordered, g.qty_invoiced FROM (
SELECT STRAIGHT_JOIN
    pb.id AS item_id,
    pb.sku AS sku,
    pb.description AS description,
    bu_unit.id AS unit_id,
    %s AS qty_ordered,
    %s AS qty_back_ordered,
    %s AS qty_invoiced
%s
GROUP BY pb.id, pb.sku, pb.description, bu_unit.id
) g
WHERE %s
ORDER BY %s
LIMIT ?`, obDecimal("COALESCE(SUM("+obQtyOrdered+"), 0)"), obDecimal("COALESCE(SUM("+obQtyBackOrdered+"), 0)"),
		obDecimal("COALESCE(SUM("+obQtyInvoiced+"), 0)"), from, having, order)
	return query, append(args, limit+1)
}

func (r *analyticsRepoImpl) GetOpenOrderProducts(ctx context.Context, params domain.AnalyzeOpenOrderProductsParams) (*domain.OpenOrderProductPage, *apierror.APIError) {
	ctx, span := analyticsRepoTracer.Start(ctx, "repository.analytics.get_open_order_products")
	defer span.End()

	cur, apiErr := decodeBreakdownCursor(params.Cursor)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	scope, empty, apiErr := r.newOrderBookScope(ctx, params.OpenOrderFilter, true)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if empty {
		return &domain.OpenOrderProductPage{Products: []domain.OpenOrderProduct{}}, nil
	}

	query, args := openOrderProductsQuery(scope, cur, params.Limit)
	rows, err := r.queries.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	defer rows.Close()
	products := []domain.OpenOrderProduct{}
	var ranks []string
	for rows.Next() {
		var (
			p                                  domain.OpenOrderProduct
			itemID, sku, description, unitID   sql.NullString
			qtyOrdered, qtyBackOrdered, qtyInv sql.NullString
		)
		if err := rows.Scan(&itemID, &sku, &description, &unitID, &qtyOrdered, &qtyBackOrdered, &qtyInv); err != nil {
			return nil, tracing.Trace(span, db.MapSQLError(err))
		}
		p.ItemID, p.Sku, p.Description, p.UnitID = itemID.String, sku.String, nullStringPtr(description), unitID.String
		p.QuantityOrdered, p.QuantityBackOrdered, p.QuantityInvoiced = exactDecimal(qtyOrdered), exactDecimal(qtyBackOrdered), exactDecimal(qtyInv)
		products = append(products, p)
		ranks = append(ranks, qtyBackOrdered.String)
	}
	if err := rows.Err(); err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	page := &domain.OpenOrderProductPage{}
	page.Products, page.PageInfo = valueKeysetPage(products, ranks, params.Limit, cur, func(p domain.OpenOrderProduct) string { return p.ItemID })
	return page, nil
}

// valueKeysetPage trims a page read one past limit (in reverse for a backward page) and cursors its ends. ranks holds each row's ranking value as stored, in the order read.
func valueKeysetPage[T any](rows []T, ranks []string, limit int32, cur *pagination.ValueCursor, id func(T) string) ([]T, pagination.PageInfo) {
	more := len(rows) > int(limit)
	if more {
		rows, ranks = rows[:limit], ranks[:limit]
	}
	backward := cur != nil && cur.Direction == pagination.DirectionBackward
	if backward {
		slices.Reverse(rows)
		slices.Reverse(ranks)
	}
	var info pagination.PageInfo
	// Forward: more rows lie ahead, and a cursor means rows lie behind. Backward, the reverse.
	info.HasNextPage, info.HasPrevPage = more, cur != nil
	if backward {
		info.HasNextPage, info.HasPrevPage = true, more
	}
	if len(rows) == 0 {
		return rows, pagination.PageInfo{}
	}
	if info.HasNextPage {
		last := len(rows) - 1
		next := pagination.EncodeValueCursor(pagination.ValueCursor{Value: ranks[last], ID: id(rows[last]), Direction: pagination.DirectionForward})
		info.NextCursor = &next
	}
	if info.HasPrevPage {
		prev := pagination.EncodeValueCursor(pagination.ValueCursor{Value: ranks[0], ID: id(rows[0]), Direction: pagination.DirectionBackward})
		info.PrevCursor = &prev
	}
	return rows, info
}

// openOrderPageQuery picks one page of open orders, newest issue first, from the orders alone: an order is open for the report while it has a sale line the line filters count.
func openOrderPageQuery(s orderBookScope, cur *pagination.StringCursor, limit int32) (string, []any) {
	preds, args := s.orderPredicates("so")
	linePreds, lineArgs := s.linePredicates()
	preds = append(preds, "EXISTS (SELECT 1 FROM sales_order_line sol JOIN product fg ON fg.id = sol.product_id WHERE sol.sales_order_id = so.id AND "+strings.Join(linePreds, " AND ")+")")
	args = append(args, lineArgs...)
	seek, seekArgs, order := keysetPredicate(cur, "so.issued_at", "so.id")
	if seek != "" {
		preds = append(preds, seek)
		args = append(args, seekArgs...)
	}
	query := "SELECT so.id FROM sales_order so" + s.orderIndexHint() + " WHERE " + strings.Join(preds, " AND ") +
		fmt.Sprintf(" ORDER BY so.issued_at %[1]s, so.id %[1]s LIMIT ?", order)
	return query, append(args, limit+1)
}

// openOrderTotalsQuery totals the counted lines of the given orders.
func openOrderTotalsQuery(s orderBookScope) (string, []any) {
	from, args := s.fromWhere(`LEFT JOIN account buyer ON buyer.id = so.buyer_account_id
LEFT JOIN account_relation ar ON ar.owner_account_id = so.owner_account_id AND ar.counterparty_account_id = so.buyer_account_id AND ar.account_relation_role_code = 'customer'
LEFT JOIN address shipping_address ON shipping_address.id = so.shipping_address_id
LEFT JOIN geolocation shipping_geolocation ON shipping_geolocation.id = shipping_address.geolocation_id`)
	return `SELECT STRAIGHT_JOIN
    so.id, so.number, so.sales_order_status_code, so.issued_at, so.buyer_account_id, buyer.name, ar.external_number,
    shipping_geolocation.state, shipping_geolocation.country,
    COUNT(DISTINCT sol.id),
    ` + obDecimal("COALESCE(SUM("+obTotalOrdered+"), 0)") + `
` + from + `
GROUP BY so.id, so.number, so.sales_order_status_code, so.issued_at, so.buyer_account_id, buyer.name, ar.external_number,
    shipping_geolocation.state, shipping_geolocation.country`, args
}

func (r *analyticsRepoImpl) ListOpenOrders(ctx context.Context, params domain.ListOpenOrdersParams) (*domain.OpenOrderPage, *apierror.APIError) {
	ctx, span := analyticsRepoTracer.Start(ctx, "repository.analytics.list_open_orders")
	defer span.End()

	cur, apiErr := keysetCursor(params.Cursor)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	scope, empty, apiErr := r.newOrderBookScope(ctx, params.OpenOrderFilter, true)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if empty {
		return &domain.OpenOrderPage{Orders: []domain.OpenOrder{}}, nil
	}

	pageQuery, pageArgs := openOrderPageQuery(scope, cur, params.Limit)
	ids, apiErr := queryStringColumn(ctx, r.queries.DB(), pageQuery, pageArgs...)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	page := &domain.OpenOrderPage{Orders: []domain.OpenOrder{}}
	if len(ids) == 0 {
		page.Orders, page.PageInfo = pagination.BuildPageString(page.Orders, params.Limit, cursorDirection(cur),
			func(o domain.OpenOrder) time.Time { return o.IssuedAt }, func(o domain.OpenOrder) string { return o.ID })
		return page, nil
	}

	scope.orderIDs, scope.orderIndex = ids, ""
	query, args := openOrderTotalsQuery(scope)
	rows, err := r.queries.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	defer rows.Close()
	byID := make(map[string]domain.OpenOrder, len(ids))
	for rows.Next() {
		var (
			o                                          domain.OpenOrder
			issuedAt                                   sql.NullTime
			customerName, customerNumber, state, cntry sql.NullString
			total                                      sql.NullString
		)
		if err := rows.Scan(&o.ID, &o.Number, &o.Status, &issuedAt, &o.CustomerID, &customerName, &customerNumber, &state, &cntry, &o.LineCount, &total); err != nil {
			return nil, tracing.Trace(span, db.MapSQLError(err))
		}
		o.Number = textutil.FormatRecordNumber(o.Number)
		if issuedAt.Valid {
			o.IssuedAt = issuedAt.Time
		}
		o.CustomerName, o.CustomerNumber = customerName.String, nullStringPtr(customerNumber)
		o.ShipToState, o.ShipToCountry, o.TotalOrdered = nullStringPtr(state), nullStringPtr(cntry), exactDecimal(total)
		byID[o.ID] = o
	}
	if err := rows.Err(); err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	// The page query read the orders in page order; an order is in it only because a counted line exists, so each has its totals.
	for _, id := range ids {
		if o, ok := byID[id]; ok {
			page.Orders = append(page.Orders, o)
		}
	}
	page.Orders, page.PageInfo = pagination.BuildPageString(page.Orders, params.Limit, cursorDirection(cur),
		func(o domain.OpenOrder) time.Time { return o.IssuedAt }, func(o domain.OpenOrder) string { return o.ID })
	return page, nil
}

// openOrderLinesQuery reads one order's sale lines, by SKU. The unit price is per the base unit of its denominator's dimension.
func openOrderLinesQuery(s orderBookScope) (string, []any) {
	from, args := s.fromWhere("")
	return `SELECT STRAIGHT_JOIN
    sol.id, pb.id, pb.sku, pb.description, bu_unit.id,
    ` + obDecimal(obPricePerBase) + `,
    u_price_num.id, u_price_num.abbreviation, u_price_den.unit_dimension_code,
    ` + obDecimal(obQtyBackOrdered) + `,
    ` + obDecimal(obQtyInvoiced) + `,
    ` + obDecimal(obTotalOrdered) + `
` + from + `
ORDER BY pb.sku ASC, sol.id ASC`, args
}

// GetOpenOrderLines returns one sales order's sale lines, whether or not it is still open. found is false when the account has no such sales order, or salesRepID is set and the order is not theirs.
func (r *analyticsRepoImpl) GetOpenOrderLines(ctx context.Context, accountID, orderID string, salesRepID *string) (lines []domain.OpenOrderLine, found bool, apiErr *apierror.APIError) {
	ctx, span := analyticsRepoTracer.Start(ctx, "repository.analytics.get_open_order_lines")
	defer span.End()

	existsQuery := "SELECT 1 FROM sales_order WHERE id = ? AND owner_account_id = ? AND sales_order_type_code = 'sales_order'"
	existsArgs := []any{orderID, accountID}
	if salesRepID != nil {
		existsQuery += " AND sales_rep_id = ?"
		existsArgs = append(existsArgs, *salesRepID)
	}
	var exists int
	err := r.queries.DB().QueryRowContext(ctx, existsQuery, existsArgs...).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, tracing.Trace(span, db.MapSQLError(err))
	}

	// The dashboard's order detail read every sale line of the order, whatever its status.
	query, args := openOrderLinesQuery(orderBookScope{accountID: accountID, anyStatus: true, orderIDs: []string{orderID}})
	rows, err := r.queries.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, true, tracing.Trace(span, db.MapSQLError(err))
	}
	defer rows.Close()
	lines = []domain.OpenOrderLine{}
	var dimensions []string
	for rows.Next() {
		var (
			l                                         domain.OpenOrderLine
			itemID, sku, description, unitID          sql.NullString
			price, numID, numAbbr, dimension          sql.NullString
			qtyBackOrdered, qtyInvoiced, totalOrdered sql.NullString
		)
		if err := rows.Scan(&l.ID, &itemID, &sku, &description, &unitID, &price, &numID, &numAbbr, &dimension,
			&qtyBackOrdered, &qtyInvoiced, &totalOrdered); err != nil {
			return nil, true, tracing.Trace(span, db.MapSQLError(err))
		}
		l.ItemID, l.Sku, l.Description, l.UnitID = itemID.String, sku.String, nullStringPtr(description), unitID.String
		l.UnitPrice, l.UnitPriceNumeratorUnitID, l.UnitPriceNumeratorAbbr = exactDecimal(price), numID.String, numAbbr.String
		l.QuantityBackOrdered, l.QuantityInvoiced, l.TotalOrdered = exactDecimal(qtyBackOrdered), exactDecimal(qtyInvoiced), exactDecimal(totalOrdered)
		lines = append(lines, l)
		dimensions = append(dimensions, dimension.String)
	}
	if err := rows.Err(); err != nil {
		return nil, true, tracing.Trace(span, db.MapSQLError(err))
	}
	if len(lines) == 0 {
		return lines, true, nil
	}

	bases, apiErr := r.dimensionBaseUnits(ctx)
	if apiErr != nil {
		return nil, true, tracing.Trace(span, apiErr)
	}
	for i := range lines {
		base := bases[dimensions[i]]
		lines[i].UnitPriceDenominatorUnitID, lines[i].UnitPriceDenominatorAbbr = base.id, base.abbreviation
	}
	return lines, true, nil
}

type dimensionBaseUnit struct{ id, abbreviation string }

// dimensionBaseUnits is each dimension's base unit, the one every other unit's ratio is relative to. Base units are platform-defined.
func (r *analyticsRepoImpl) dimensionBaseUnits(ctx context.Context) (map[string]dimensionBaseUnit, *apierror.APIError) {
	rows, err := r.queries.DB().QueryContext(ctx,
		"SELECT unit_dimension_code, id, abbreviation FROM unit WHERE account_id IS NULL AND is_base_unit = 1 ORDER BY unit_dimension_code, id")
	if err != nil {
		return nil, db.MapSQLError(err)
	}
	defer rows.Close()
	out := map[string]dimensionBaseUnit{}
	for rows.Next() {
		var dimension string
		var u dimensionBaseUnit
		if err := rows.Scan(&dimension, &u.id, &u.abbreviation); err != nil {
			return nil, db.MapSQLError(err)
		}
		if _, ok := out[dimension]; !ok {
			out[dimension] = u
		}
	}
	if err := rows.Err(); err != nil {
		return nil, db.MapSQLError(err)
	}
	return out, nil
}
