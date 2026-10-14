package repository

import (
	"context"
	"strings"

	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/db"
)

// deliveryOrders reads the window's measured orders from the quantities their lines carry once every line
// carries them (lineCopiesComplete), and from the quantity rows until then.
func (r *productionScheduleInputRepoImpl) deliveryOrders(ctx context.Context, params sqlc.ListDeliveryPerformanceOrdersParams, filters deliveryFilterArgs) ([]sqlc.ListDeliveryPerformanceOrdersRow, *apierror.APIError) {
	copies, err := lineCopiesComplete(ctx, r.queries.DB())
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	if !copies {
		rows, err := r.queries.ListDeliveryPerformanceOrders(ctx, params)
		return rows, db.MapSQLError(err)
	}
	rows, err := r.deliveryOrdersFromCopies(ctx, params, filters)
	return rows, db.MapSQLError(err)
}

// deliveryOrdersFromCopies is ListDeliveryPerformanceOrders over the quantities the lines carry: an order's
// lines, and each line's pick lines, are each one range of a key holding the quantity, so no line or
// quantity row is visited. A line whose quantity row is missing has no copied unit and is left out, as the
// inner join to the row left it out. Only the filters set emit a predicate.
func (r *productionScheduleInputRepoImpl) deliveryOrdersFromCopies(ctx context.Context, p sqlc.ListDeliveryPerformanceOrdersParams, f deliveryFilterArgs) ([]sqlc.ListDeliveryPerformanceOrdersRow, error) {
	where := []string{
		"so.owner_account_id = ?",
		"so.sales_order_type_code = 'sales_order'",
		// An IN-list over the closed status enum keeps the status key usable past its first column.
		"so.sales_order_status_code IN ('issued', 'fulfilled')",
		"sol.quantity_unit_id IS NOT NULL",
		"p.product_type_code = 'sale'",
		"so.ship_by_date IS NOT NULL",
		"so.ship_by_date >= ?",
		"so.ship_by_date <= ?",
	}
	args := []any{p.AccountID, p.WindowStart, p.WindowEnd}
	if f.includeSalesRep {
		where = append(where, "so.sales_rep_id IN ("+placeholders(len(f.salesRepIDs))+")")
		for _, id := range f.salesRepIDs {
			args = append(args, id)
		}
	}
	if f.includeProductLine {
		where = append(where, "p.product_line_id IN ("+placeholders(len(f.productLineIDs))+")")
		for _, id := range f.productLineIDs {
			args = append(args, id)
		}
	}
	if f.includeBuyer {
		where = append(where, "so.buyer_account_id IN ("+placeholders(len(f.buyerIDs))+")")
		args = append(args, stringsToAny(f.buyerIDs)...)
	}
	rows, err := r.queries.DB().QueryContext(ctx, `SELECT
    so.id, so.number, so.buyer_account_id, buyer.name, ar.account_group_id, ag.name, so.sales_rep_id,
    so.ship_by_date, so.issued_at, so.first_ship_at, so.completed_at, so.sales_order_status_code,
    so.lead_time_days, so.lead_time_source_code,
    CAST(COALESCE(SUM(sol.quantity_value), 0) AS DECIMAL(65,30)),
    CAST(COALESCE(SUM(
        (SELECT COALESCE(SUM(pl.quantity_value), 0) FROM pick_line pl FORCE INDEX (pick_line_sol_packed_copies_idx)
            WHERE pl.sales_order_line_id = sol.id AND pl.packed_at IS NOT NULL)
    ), 0) AS DECIMAL(65,30))
FROM sales_order so FORCE INDEX (sales_order_owner_status_ship_by_idx, sales_order_owner_buyer_ship_by_idx, sales_order_owner_rep_ship_by_idx)
JOIN sales_order_line sol FORCE INDEX (sales_order_line_order_copies_idx) ON sol.sales_order_id = so.id
JOIN product p ON p.id = sol.product_id
LEFT JOIN account buyer ON buyer.id = so.buyer_account_id
LEFT JOIN account_relation ar ON ar.owner_account_id = so.owner_account_id
    AND ar.counterparty_account_id = so.buyer_account_id
    AND ar.account_relation_role_code = 'customer'
LEFT JOIN account_group ag ON ag.id = ar.account_group_id
WHERE `+strings.Join(where, "\n  AND ")+`
GROUP BY so.id, so.number, so.buyer_account_id, buyer.name, ar.account_group_id, ag.name,
    so.sales_rep_id, so.ship_by_date, so.issued_at, so.first_ship_at, so.completed_at, so.sales_order_status_code,
    so.lead_time_days, so.lead_time_source_code
ORDER BY so.ship_by_date, so.id`, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []sqlc.ListDeliveryPerformanceOrdersRow
	for rows.Next() {
		var i sqlc.ListDeliveryPerformanceOrdersRow
		if err := rows.Scan(&i.SalesOrderID, &i.SalesOrderNumber, &i.BuyerAccountID, &i.CustomerName, &i.CustomerGroupID,
			&i.CustomerGroupName, &i.SalesRepID, &i.ShipByDate, &i.IssuedAt, &i.FirstShipAt, &i.CompletedAt,
			&i.SalesOrderStatusCode, &i.LeadTimeDays, &i.LeadTimeSourceCode, &i.QuantityOrdered, &i.QuantityPacked); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}
