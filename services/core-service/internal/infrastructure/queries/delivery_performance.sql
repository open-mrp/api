-- Delivery performance: did we ship what we promised, when we promised it.
--
-- Measured against ship_by_date, which is the commitment stamped on the order at issue rather than anything recomputed later. An order whose customer's lead time has since been renegotiated is still judged against what it was actually promised.

-- ListDeliveryPerformanceOrders returns every order carrying a commitment that came due inside the window, with what actually happened to it.
--
-- Only orders with a ship_by_date participate: an order with no commitment cannot be late, and counting it as on time would inflate the rate with orders nobody promised anything about. The count of unstamped orders is reported separately so that exclusion is visible rather than silent.
--
-- quantity_ordered and quantity_packed are aggregated over sale-type lines only, matching the fulfillment-progress math everywhere else — freight and credit lines are not shipped.
-- name: ListDeliveryPerformanceOrders :many
SELECT
    so.id AS sales_order_id,
    so.number AS sales_order_number,
    so.buyer_account_id,
    so.ship_by_date,
    so.issued_at,
    so.first_ship_at,
    so.completed_at,
    so.sales_order_status_code,
    so.lead_time_days,
    so.lead_time_source_code,
    CAST(COALESCE(SUM(q.value), 0) AS DECIMAL(65,30)) AS quantity_ordered,
    CAST(COALESCE(SUM(
        (SELECT COALESCE(SUM(plq.value), 0) FROM pick_line pl
            JOIN quantity plq ON plq.id = pl.quantity_id
            WHERE pl.sales_order_line_id = sol.id AND pl.packed_at IS NOT NULL)
    ), 0) AS DECIMAL(65,30)) AS quantity_packed
FROM sales_order so
JOIN sales_order_line sol ON sol.sales_order_id = so.id
JOIN quantity q ON q.id = sol.quantity_id
JOIN product p ON p.id = sol.product_id
WHERE so.owner_account_id = sqlc.arg('account_id')
  AND so.sales_order_type_code = 'sales_order'
  AND so.sales_order_status_code <> 'estimate'
  AND p.product_type_code = 'sale'
  AND so.ship_by_date IS NOT NULL
  AND so.ship_by_date >= sqlc.arg('window_start')
  AND so.ship_by_date <= sqlc.arg('window_end')
GROUP BY so.id, so.number, so.buyer_account_id, so.ship_by_date, so.issued_at,
         so.first_ship_at, so.completed_at, so.sales_order_status_code,
         so.lead_time_days, so.lead_time_source_code
ORDER BY so.ship_by_date, so.id;

-- CountUncommittedOrders counts issued orders in the window that carry no ship-by date.
--
-- These are excluded from every rate above. Reported so the exclusion is visible: a delivery score computed over half the order book, silently, is worse than one that says which half.
-- name: CountUncommittedOrders :one
SELECT COUNT(*) AS uncommitted_count
FROM sales_order so
WHERE so.owner_account_id = sqlc.arg('account_id')
  AND so.sales_order_type_code = 'sales_order'
  AND so.sales_order_status_code <> 'estimate'
  AND so.ship_by_date IS NULL
  AND so.issued_at IS NOT NULL
  AND so.issued_at >= sqlc.arg('window_start')
  AND so.issued_at <= sqlc.arg('window_end');
