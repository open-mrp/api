package repository

import (
	"context"
	"database/sql"
	"strings"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

// demandForecastFilters are the product-line and item filters both forecast reads take, on fg (product) and pb (item).
func demandForecastFilters(params domain.GetDemandForecastWindowParams) ([]string, []any) {
	var preds []string
	var args []any
	if len(params.ProductLineIDs) > 0 {
		preds = append(preds, "fg.product_line_id IN ("+placeholders(len(params.ProductLineIDs))+")")
		args = append(args, stringsToAny(params.ProductLineIDs)...)
	}
	if len(params.ItemIDs) > 0 {
		preds = append(preds, "pb.id IN ("+placeholders(len(params.ItemIDs))+")")
		args = append(args, stringsToAny(params.ItemIDs)...)
	}
	return preds, args
}

// demandForecastDemandQuery totals each item's ordered quantity (in its base unit) and value per month of order creation. Every sales order counts, estimates included, as the dashboard counted them. The orders drive, ranged by creation date: left to choose, the planner starts from the lines and reads each order once per line.
//
// With copies, each order's lines are one range of a key holding their quantity and price, so no line, quantity or rate row is visited: from those rows, two years of a large account's orders took 3 s warm and 28 s cold.
func demandForecastDemandQuery(params domain.GetDemandForecastWindowParams, copies bool) (string, []any) {
	preds := []string{
		"so.owner_account_id = ?",
		"fg.product_type_code = 'sale'",
		"so.sales_order_type_code = 'sales_order'",
		"so.created_at >= ?",
		"so.created_at < ?",
	}
	args := []any{params.AccountID, params.StartDate, params.EndDate}
	filters, filterArgs := demandForecastFilters(params)
	preds = append(preds, filters...)
	args = append(args, filterArgs...)

	query := `SELECT STRAIGHT_JOIN
    pb.id,
    pb.sku,
    pb.description,
    fg.product_line_id,
    bu_unit.abbreviation,
    COALESCE(u_price_num.abbreviation, '$'),
    YEAR(so.created_at) AS demand_year,
    MONTH(so.created_at) AS demand_month,
    ` + obDecimal("SUM("+obQtyOrdered+")") + `,
    ` + obDecimal("SUM("+obTotalOrdered+")") + `
FROM sales_order so FORCE INDEX (sales_order_owner_type_created_idx)
` + orderedLineValues(copies, true) + `
WHERE ` + strings.Join(preds, " AND ") + `
GROUP BY pb.id, pb.sku, pb.description, fg.product_line_id, bu_unit.abbreviation, u_price_num.abbreviation, demand_year, demand_month
ORDER BY pb.id, demand_year, demand_month`
	if copies {
		query = fromCopies.Replace(query)
	}
	return query, args
}

// orderedLineValues joins an order's lines (sol) and their product (fg), with withBaseUnit the item's base unit
// (bu_unit), and the quantity (u_ord) and price (u_price_num, u_price_den) units obTotalOrdered prices them with.
// A line without its quantity row is left out; one without its price row is not.
//
// With copies, the lines are read from sales_order_line_order_copies_idx and their values from the copies, so
// the expressions read through fromCopies. The copied unit ids are ASCII and unit.id is not, so left to choose,
// the planner scans the unit table once per line for each unit joined; the lookups are forced instead.
func orderedLineValues(copies, withBaseUnit bool) string {
	lines, values := `JOIN sales_order_line sol ON sol.sales_order_id = so.id`, `JOIN quantity q_ord ON q_ord.id = sol.quantity_id
JOIN unit u_ord ON u_ord.id = q_ord.unit_id
LEFT JOIN rate r_price ON r_price.id = sol.unit_price_id
LEFT JOIN unit u_price_num ON u_price_num.id = r_price.numerator_unit_id
LEFT JOIN unit u_price_den ON u_price_den.id = r_price.denominator_unit_id`
	if copies {
		lines = `JOIN sales_order_line sol FORCE INDEX (sales_order_line_order_copies_idx) ON sol.sales_order_id = so.id`
		values = `JOIN unit u_ord FORCE INDEX (PRIMARY) ON u_ord.id = sol.quantity_unit_id
LEFT JOIN unit u_price_num FORCE INDEX (PRIMARY) ON u_price_num.id = sol.unit_price_numerator_unit_id
LEFT JOIN unit u_price_den FORCE INDEX (PRIMARY) ON u_price_den.id = sol.unit_price_denominator_unit_id`
	}
	product := `JOIN product fg ON fg.id = sol.product_id`
	if withBaseUnit {
		product += `
JOIN item pb ON pb.id = fg.item_id
JOIN item_category ic ON ic.id = pb.item_category_id
JOIN unit_group ug ON ug.id = ic.unit_group_id
JOIN unit bu_unit FORCE INDEX (PRIMARY) ON bu_unit.id = ug.base_unit_id`
	}
	return lines + "\n" + product + "\n" + values
}

// GetDemandForecastMonthlyDemand returns order-based monthly demand and revenue rows per item within the given window.
func (r *analyticsRepoImpl) GetDemandForecastMonthlyDemand(ctx context.Context, params domain.GetDemandForecastWindowParams) ([]domain.DemandForecastMonthlyDemandRow, *apierror.APIError) {
	ctx, span := analyticsRepoTracer.Start(ctx, "repository.analytics.get_demand_forecast_monthly_demand")
	defer span.End()

	copies, apiErr := r.lineCopies(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	query, args := demandForecastDemandQuery(params, copies)
	rows, err := r.queries.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	defer rows.Close()

	out := []domain.DemandForecastMonthlyDemandRow{}
	for rows.Next() {
		var (
			row                              domain.DemandForecastMonthlyDemandRow
			description, productLineID, unit sql.NullString
			demand, revenue                  sql.NullString
		)
		if err := rows.Scan(&row.ItemID, &row.ProductSku, &description, &productLineID, &unit, &row.Currency,
			&row.DemandYear, &row.DemandMonth, &demand, &revenue); err != nil {
			return nil, tracing.Trace(span, db.MapSQLError(err))
		}
		row.ProductDescription, row.ProductLineID, row.Unit = nullStringPtr(description), nullStringPtr(productLineID), unit.String
		row.MonthlyDemand, row.MonthlyRevenue = decimalToFloat(demand), decimalToFloat(revenue)
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	return out, nil
}

// demandForecastRevenueQuery totals each item's invoiced value per month of invoice. It reads the account's invoices in the window first: left to choose, the planner starts from a product-line or item filter's every invoiced line.
//
// With copies, an invoice's lines are one range of invoice_line_invoice_copies_idx and the lines of the order
// they invoice one range of sales_order_line_order_copies_idx, so no line, quantity or rate row is visited.
// The order line is found within the invoice's order: an invoice only invoices its own order's lines.
func demandForecastRevenueQuery(params domain.GetDemandForecastWindowParams, copies bool) (string, []any) {
	preds := []string{
		"inv.account_id = ?",
		"inv.created_at >= ?",
		"inv.created_at < ?",
		"fg.product_type_code = 'sale'",
		"so.sales_order_type_code = 'sales_order'",
	}
	args := []any{params.AccountID, params.StartDate, params.EndDate}
	filters, filterArgs := demandForecastFilters(params)
	preds = append(preds, filters...)
	args = append(args, filterArgs...)

	const invoicedNorm = "((q_in.value * (u_in.ratio_numerator / u_in.ratio_denominator)) + (u_in.offset_numerator / u_in.offset_denominator))"
	from := `FROM invoice inv FORCE INDEX (invoice_account_created_idx)
JOIN invoice_line il ON il.invoice_id = inv.id
JOIN sales_order_line sol ON sol.id = il.sales_order_line_id
JOIN sales_order so ON so.id = sol.sales_order_id
JOIN product fg ON fg.id = sol.product_id
JOIN item pb ON pb.id = fg.item_id
JOIN item_category ic ON ic.id = pb.item_category_id
JOIN unit_group ug ON ug.id = ic.unit_group_id
JOIN unit bu_unit ON bu_unit.id = ug.base_unit_id
JOIN quantity q_in ON q_in.id = il.quantity_id
JOIN unit u_in ON u_in.id = q_in.unit_id
LEFT JOIN rate r_price ON r_price.id = sol.unit_price_id
LEFT JOIN unit u_price_num ON u_price_num.id = r_price.numerator_unit_id
LEFT JOIN unit u_price_den ON u_price_den.id = r_price.denominator_unit_id`
	if copies {
		from = `FROM invoice inv FORCE INDEX (invoice_account_created_idx)
JOIN invoice_line il FORCE INDEX (invoice_line_invoice_copies_idx) ON il.invoice_id = inv.id
JOIN sales_order_line sol FORCE INDEX (sales_order_line_order_copies_idx) ON sol.sales_order_id = inv.sales_order_id AND sol.id = il.sales_order_line_id
JOIN sales_order so ON so.id = sol.sales_order_id
JOIN product fg ON fg.id = sol.product_id
JOIN item pb ON pb.id = fg.item_id
JOIN item_category ic ON ic.id = pb.item_category_id
JOIN unit_group ug ON ug.id = ic.unit_group_id
JOIN unit bu_unit FORCE INDEX (PRIMARY) ON bu_unit.id = ug.base_unit_id
JOIN unit u_in FORCE INDEX (PRIMARY) ON u_in.id = il.quantity_unit_id
LEFT JOIN unit u_price_num FORCE INDEX (PRIMARY) ON u_price_num.id = sol.unit_price_numerator_unit_id
LEFT JOIN unit u_price_den FORCE INDEX (PRIMARY) ON u_price_den.id = sol.unit_price_denominator_unit_id`
	}
	query := `SELECT STRAIGHT_JOIN
    pb.id,
    YEAR(inv.created_at) AS revenue_year,
    MONTH(inv.created_at) AS revenue_month,
    ` + obDecimal("SUM("+invoicedNorm+" * "+obPricePerBase+")") + `
` + from + `
WHERE ` + strings.Join(preds, " AND ") + `
GROUP BY pb.id, revenue_year, revenue_month
ORDER BY pb.id, revenue_year, revenue_month`
	if copies {
		query = fromCopies.Replace(query)
	}
	return query, args
}

// GetDemandForecastMonthlyRevenue returns invoice-based monthly revenue rows per item within the given window.
func (r *analyticsRepoImpl) GetDemandForecastMonthlyRevenue(ctx context.Context, params domain.GetDemandForecastWindowParams) ([]domain.DemandForecastMonthlyRevenueRow, *apierror.APIError) {
	ctx, span := analyticsRepoTracer.Start(ctx, "repository.analytics.get_demand_forecast_monthly_revenue")
	defer span.End()

	copies, apiErr := r.lineCopies(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	query, args := demandForecastRevenueQuery(params, copies)
	rows, err := r.queries.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	defer rows.Close()

	out := []domain.DemandForecastMonthlyRevenueRow{}
	for rows.Next() {
		var (
			row     domain.DemandForecastMonthlyRevenueRow
			revenue sql.NullString
		)
		if err := rows.Scan(&row.ItemID, &row.RevenueYear, &row.RevenueMonth, &revenue); err != nil {
			return nil, tracing.Trace(span, db.MapSQLError(err))
		}
		row.MonthlyRevenue = decimalToFloat(revenue)
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	return out, nil
}
