-- Delivery performance: did we ship what we promised, when we promised it.
--
-- Measured against ship_by_date, which is the commitment stamped on the order at issue rather than anything recomputed later. An order whose customer's lead time has since been renegotiated is still judged against what it was actually promised.

-- ListDeliveryPerformanceOrders returns every order carrying a commitment that came due inside the window, with what actually happened to it.
--
-- Only orders with a ship_by_date participate: an order with no commitment cannot be late, and counting it as on time would inflate the rate with orders nobody promised anything about. The count of unstamped orders is reported separately so that exclusion is visible rather than silent.
--
-- quantity_ordered and quantity_packed are aggregated over sale-type lines only, matching the fulfillment-progress math everywhere else — freight and credit lines are not shipped.
-- The customer, customer-group, product-line and sales-rep filters are spelled exactly as GetSalesEntries spells them, including the parent/child customer expansion. Two analytics pages that disagree about what "this customer" selects are worse than either being wrong on its own.
-- name: ListDeliveryPerformanceOrders :many
SELECT
    so.id AS sales_order_id,
    so.number AS sales_order_number,
    so.buyer_account_id,
    buyer.name AS customer_name,
    ar.account_group_id AS customer_group_id,
    ag.name AS customer_group_name,
    so.sales_rep_id,
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
LEFT JOIN account buyer ON buyer.id = so.buyer_account_id
LEFT JOIN account_relation ar ON ar.owner_account_id = so.owner_account_id
    AND ar.counterparty_account_id = so.buyer_account_id
    AND ar.account_relation_role_code = 'customer'
LEFT JOIN account_group ag ON ag.id = ar.account_group_id
WHERE so.owner_account_id = sqlc.arg('account_id')
  AND so.sales_order_type_code = 'sales_order'
  AND so.sales_order_status_code <> 'estimate'
  AND p.product_type_code = 'sale'
  AND so.ship_by_date IS NOT NULL
  AND so.ship_by_date >= sqlc.arg('window_start')
  AND so.ship_by_date <= sqlc.arg('window_end')
  AND (sqlc.arg('include_sales_rep_filter') = false OR so.sales_rep_id IN (sqlc.slice('sales_rep_ids')))
  AND (sqlc.arg('include_product_line_filter') = false OR p.product_line_id IN (sqlc.slice('product_line_ids')))
  AND (sqlc.arg('include_customer_group_filter') = false OR ar.account_group_id IN (sqlc.slice('customer_group_ids')))
  AND (sqlc.arg('include_customer_filter') = false OR (
      so.buyer_account_id IN (sqlc.slice('customer_ids'))
      OR EXISTS (
          SELECT 1
          FROM account_relation ar_child
          WHERE ar_child.owner_account_id = so.owner_account_id
            AND ar_child.account_relation_role_code = 'customer'
            AND ar_child.counterparty_account_id = so.buyer_account_id
            AND ar_child.parent_account_relation_id IN (
                SELECT ar_parent.id
                FROM account_relation ar_parent
                WHERE ar_parent.owner_account_id = so.owner_account_id
                  AND ar_parent.account_relation_role_code = 'customer'
                  AND ar_parent.counterparty_account_id IN (sqlc.slice('customer_ids'))
            )
      )
  ))
GROUP BY so.id, so.number, so.buyer_account_id, buyer.name, ar.account_group_id, ag.name,
         so.sales_rep_id, so.ship_by_date, so.issued_at,
         so.first_ship_at, so.completed_at, so.sales_order_status_code,
         so.lead_time_days, so.lead_time_source_code
ORDER BY so.ship_by_date, so.id;

-- ListDeliveryOrderProductLines maps each order in the window onto the product lines it contains, so delivery can be broken down by line.
--
-- Separate from the query above rather than folded into it: that one aggregates one row per order, and joining the product line in would multiply the order across its lines and double-count every quantity. An order spanning two product lines appears here twice, which is correct — a late order is late for every line on it.
-- name: ListDeliveryOrderProductLines :many
SELECT DISTINCT
    so.id AS sales_order_id,
    p.product_line_id,
    pl.name AS product_line_name
FROM sales_order so
JOIN sales_order_line sol ON sol.sales_order_id = so.id
JOIN product p ON p.id = sol.product_id
JOIN product_line pl ON pl.id = p.product_line_id
LEFT JOIN account_relation ar ON ar.owner_account_id = so.owner_account_id
    AND ar.counterparty_account_id = so.buyer_account_id
    AND ar.account_relation_role_code = 'customer'
WHERE so.owner_account_id = sqlc.arg('account_id')
  AND so.sales_order_type_code = 'sales_order'
  AND so.sales_order_status_code <> 'estimate'
  AND p.product_type_code = 'sale'
  AND so.ship_by_date IS NOT NULL
  AND so.ship_by_date >= sqlc.arg('window_start')
  AND so.ship_by_date <= sqlc.arg('window_end')
  AND (sqlc.arg('include_sales_rep_filter') = false OR so.sales_rep_id IN (sqlc.slice('sales_rep_ids')))
  AND (sqlc.arg('include_product_line_filter') = false OR p.product_line_id IN (sqlc.slice('product_line_ids')))
  AND (sqlc.arg('include_customer_group_filter') = false OR ar.account_group_id IN (sqlc.slice('customer_group_ids')))
  AND (sqlc.arg('include_customer_filter') = false OR (
      so.buyer_account_id IN (sqlc.slice('customer_ids'))
      OR EXISTS (
          SELECT 1
          FROM account_relation ar_child
          WHERE ar_child.owner_account_id = so.owner_account_id
            AND ar_child.account_relation_role_code = 'customer'
            AND ar_child.counterparty_account_id = so.buyer_account_id
            AND ar_child.parent_account_relation_id IN (
                SELECT ar_parent.id
                FROM account_relation ar_parent
                WHERE ar_parent.owner_account_id = so.owner_account_id
                  AND ar_parent.account_relation_role_code = 'customer'
                  AND ar_parent.counterparty_account_id IN (sqlc.slice('customer_ids'))
            )
      )
  ));

-- CountUncommittedOrders counts issued orders in the window that carry no ship-by date.
--
-- These are excluded from every rate above. Reported so the exclusion is visible: a delivery score computed over half the order book, silently, is worse than one that says which half.
-- The same filters as the measured set, so the excluded count describes the same slice of the order book the rates do. An unfiltered count beside a filtered rate would read as "this customer has 40 uncommitted orders" when the 40 belong to the whole account.
--
-- The product-line filter is an EXISTS rather than a join: this is a COUNT of orders, and joining the lines in would count an order once per line on it.
-- name: CountUncommittedOrders :one
SELECT COUNT(*) AS uncommitted_count
FROM sales_order so
LEFT JOIN account_relation ar ON ar.owner_account_id = so.owner_account_id
    AND ar.counterparty_account_id = so.buyer_account_id
    AND ar.account_relation_role_code = 'customer'
WHERE so.owner_account_id = sqlc.arg('account_id')
  AND so.sales_order_type_code = 'sales_order'
  AND so.sales_order_status_code <> 'estimate'
  AND so.ship_by_date IS NULL
  AND so.issued_at IS NOT NULL
  AND so.issued_at >= sqlc.arg('window_start')
  AND so.issued_at <= sqlc.arg('window_end')
  AND (sqlc.arg('include_sales_rep_filter') = false OR so.sales_rep_id IN (sqlc.slice('sales_rep_ids')))
  AND (sqlc.arg('include_customer_group_filter') = false OR ar.account_group_id IN (sqlc.slice('customer_group_ids')))
  AND (sqlc.arg('include_product_line_filter') = false OR EXISTS (
      SELECT 1
      FROM sales_order_line sol
      JOIN product p ON p.id = sol.product_id
      WHERE sol.sales_order_id = so.id
        AND p.product_type_code = 'sale'
        AND p.product_line_id IN (sqlc.slice('product_line_ids'))
  ))
  AND (sqlc.arg('include_customer_filter') = false OR (
      so.buyer_account_id IN (sqlc.slice('customer_ids'))
      OR EXISTS (
          SELECT 1
          FROM account_relation ar_child
          WHERE ar_child.owner_account_id = so.owner_account_id
            AND ar_child.account_relation_role_code = 'customer'
            AND ar_child.counterparty_account_id = so.buyer_account_id
            AND ar_child.parent_account_relation_id IN (
                SELECT ar_parent.id
                FROM account_relation ar_parent
                WHERE ar_parent.owner_account_id = so.owner_account_id
                  AND ar_parent.account_relation_role_code = 'customer'
                  AND ar_parent.counterparty_account_id IN (sqlc.slice('customer_ids'))
            )
      )
  ));
