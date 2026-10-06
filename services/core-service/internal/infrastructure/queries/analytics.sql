-- name: GetNewCustomerEntries :many
SELECT ar.created_at
FROM account_relation ar
WHERE ar.owner_account_id = sqlc.arg('owner_account_id')
  AND ar.account_relation_role_code = 'customer'
  AND ar.created_at >= sqlc.arg('start_date')
  AND ar.created_at <= sqlc.arg('end_date')
  AND (
    sqlc.arg('include_customer_group_filter') = false
    OR ar.account_group_id IN (sqlc.slice('customer_group_ids'))
    OR EXISTS (
      SELECT 1 FROM account_relation_price_group arpg
      WHERE arpg.account_relation_id = ar.id
        AND arpg.account_group_id IN (sqlc.slice('price_group_ids'))
    )
  )
  AND (
    sqlc.arg('include_sales_rep_filter') = false
    OR ar.default_sales_rep_id IN (sqlc.slice('sales_rep_ids'))
  )
ORDER BY ar.created_at ASC;

-- GetSalesEntries reads the window's invoices first: left to choose, a customer-group or product-line filter made the optimizer start from that filter's every order or line ever invoiced.
-- The customer and customer-group filters arrive resolved to the buyers they admit (resolveCustomerBuyers).
-- name: GetSalesEntries :many
SELECT
    il.id AS id,
    so.issued_at AS issued_at,
    so.completed_at AS completed_at,
    so.first_ship_at AS first_ship_at,
    so.promised_at AS promised_at,
    inv.created_at AS invoice_date,
    inv.id AS invoice_id,
    inv.number AS invoice_number,
    so.customer_po_number AS customer_po,
    so.number AS sales_order_number,
    so.id AS sales_order_id,
    so.sales_rep_id AS sales_rep_id,
    sr_user.username AS sales_rep_username,
    so.buyer_account_id AS customer_id,
    parent_ar.counterparty_account_id AS parent_customer_id,
    buyer.name AS customer_name,
    ar.external_number AS customer_number,
    buyer.created_at AS customer_created_at,
    ar.account_group_id AS customer_type_group_id,
    ag.name AS customer_group_name,
    fg.product_line_id AS product_line_id,
    fg.product_type_code AS product_type_code,
    pb.id AS item_id,
    pb.sku AS product_sku,
    pb.description AS product_description,
    ic.name AS category_name,
    pl.name AS product_line,
    bu_unit.abbreviation AS unit,
    -- quantityInvoiced: convert invoice line quantity to base unit
    CAST(
        (
            (
                (CAST(q_in.value AS DECIMAL(65,30)) * (CAST(u_in.ratio_numerator AS DECIMAL(65,30)) / CAST(u_in.ratio_denominator AS DECIMAL(65,30))))
                + (CAST(u_in.offset_numerator AS DECIMAL(65,30)) / CAST(u_in.offset_denominator AS DECIMAL(65,30)))
            )
            - (CAST(bu_unit.offset_numerator AS DECIMAL(65,30)) / CAST(bu_unit.offset_denominator AS DECIMAL(65,30)))
        )
        / NULLIF((CAST(bu_unit.ratio_numerator AS DECIMAL(65,30)) / CAST(bu_unit.ratio_denominator AS DECIMAL(65,30))), 0)
        AS DECIMAL(65,30)
    ) AS quantity_invoiced,
    -- totalInvoiced: quantity_in_base * converted_price
    CAST(
        (
            (CAST(q_in.value AS DECIMAL(65,30)) * (CAST(u_in.ratio_numerator AS DECIMAL(65,30)) / CAST(u_in.ratio_denominator AS DECIMAL(65,30))))
            + (CAST(u_in.offset_numerator AS DECIMAL(65,30)) / CAST(u_in.offset_denominator AS DECIMAL(65,30)))
        )
        *
        (
            (
                (CAST(r_price.value AS DECIMAL(65,30)) * (CAST(u_price_num.ratio_numerator AS DECIMAL(65,30)) / CAST(u_price_num.ratio_denominator AS DECIMAL(65,30))))
                + (CAST(u_price_num.offset_numerator AS DECIMAL(65,30)) / CAST(u_price_num.offset_denominator AS DECIMAL(65,30)))
            )
            / NULLIF(((CAST(u_price_den.ratio_numerator AS DECIMAL(65,30)) / CAST(u_price_den.ratio_denominator AS DECIMAL(65,30))) + (CAST(u_price_den.offset_numerator AS DECIMAL(65,30)) / CAST(u_price_den.offset_denominator AS DECIMAL(65,30)))), 0)
        )
        AS DECIMAL(65,30)
    ) AS total_invoiced,
    -- totalCost: quantity_in_base * converted_cost
    CAST(
        (
            (CAST(q_in.value AS DECIMAL(65,30)) * (CAST(u_in.ratio_numerator AS DECIMAL(65,30)) / CAST(u_in.ratio_denominator AS DECIMAL(65,30))))
            + (CAST(u_in.offset_numerator AS DECIMAL(65,30)) / CAST(u_in.offset_denominator AS DECIMAL(65,30)))
        )
        *
        (
            (
                (COALESCE(CAST(r_cost.value AS DECIMAL(65,30)), 0) * (COALESCE(CAST(u_cost_num.ratio_numerator AS DECIMAL(65,30)), 1) / COALESCE(CAST(u_cost_num.ratio_denominator AS DECIMAL(65,30)), 1)))
                + (COALESCE(CAST(u_cost_num.offset_numerator AS DECIMAL(65,30)), 0) / COALESCE(CAST(u_cost_num.offset_denominator AS DECIMAL(65,30)), 1))
            )
            / NULLIF(((COALESCE(CAST(u_cost_den.ratio_numerator AS DECIMAL(65,30)), 1) / COALESCE(CAST(u_cost_den.ratio_denominator AS DECIMAL(65,30)), 1)) + (COALESCE(CAST(u_cost_den.offset_numerator AS DECIMAL(65,30)), 0) / COALESCE(CAST(u_cost_den.offset_denominator AS DECIMAL(65,30)), 1))), 0)
        )
        AS DECIMAL(65,30)
    ) AS total_cost,
    -- totalProfit: totalInvoiced - totalCost
    CAST(
        (
            (CAST(q_in.value AS DECIMAL(65,30)) * (CAST(u_in.ratio_numerator AS DECIMAL(65,30)) / CAST(u_in.ratio_denominator AS DECIMAL(65,30))))
            + (CAST(u_in.offset_numerator AS DECIMAL(65,30)) / CAST(u_in.offset_denominator AS DECIMAL(65,30)))
        )
        *
        (
            (
                (CAST(r_price.value AS DECIMAL(65,30)) * (CAST(u_price_num.ratio_numerator AS DECIMAL(65,30)) / CAST(u_price_num.ratio_denominator AS DECIMAL(65,30))))
                + (CAST(u_price_num.offset_numerator AS DECIMAL(65,30)) / CAST(u_price_num.offset_denominator AS DECIMAL(65,30)))
            )
            / NULLIF(((CAST(u_price_den.ratio_numerator AS DECIMAL(65,30)) / CAST(u_price_den.ratio_denominator AS DECIMAL(65,30))) + (CAST(u_price_den.offset_numerator AS DECIMAL(65,30)) / CAST(u_price_den.offset_denominator AS DECIMAL(65,30)))), 0)
        )
        -
        (
            (CAST(q_in.value AS DECIMAL(65,30)) * (CAST(u_in.ratio_numerator AS DECIMAL(65,30)) / CAST(u_in.ratio_denominator AS DECIMAL(65,30))))
            + (CAST(u_in.offset_numerator AS DECIMAL(65,30)) / CAST(u_in.offset_denominator AS DECIMAL(65,30)))
        )
        *
        (
            (
                (COALESCE(CAST(r_cost.value AS DECIMAL(65,30)), 0) * (COALESCE(CAST(u_cost_num.ratio_numerator AS DECIMAL(65,30)), 1) / COALESCE(CAST(u_cost_num.ratio_denominator AS DECIMAL(65,30)), 1)))
                + (COALESCE(CAST(u_cost_num.offset_numerator AS DECIMAL(65,30)), 0) / COALESCE(CAST(u_cost_num.offset_denominator AS DECIMAL(65,30)), 1))
            )
            / NULLIF(((COALESCE(CAST(u_cost_den.ratio_numerator AS DECIMAL(65,30)), 1) / COALESCE(CAST(u_cost_den.ratio_denominator AS DECIMAL(65,30)), 1)) + (COALESCE(CAST(u_cost_den.offset_numerator AS DECIMAL(65,30)), 0) / COALESCE(CAST(u_cost_den.offset_denominator AS DECIMAL(65,30)), 1))), 0)
        )
        AS DECIMAL(65,30)
    ) AS total_profit,
    -- unitPrice: totalInvoiced / quantityInvoiced
    CAST(
        (
            (CAST(q_in.value AS DECIMAL(65,30)) * (CAST(u_in.ratio_numerator AS DECIMAL(65,30)) / CAST(u_in.ratio_denominator AS DECIMAL(65,30))))
            + (CAST(u_in.offset_numerator AS DECIMAL(65,30)) / CAST(u_in.offset_denominator AS DECIMAL(65,30)))
        )
        *
        (
            (
                (CAST(r_price.value AS DECIMAL(65,30)) * (CAST(u_price_num.ratio_numerator AS DECIMAL(65,30)) / CAST(u_price_num.ratio_denominator AS DECIMAL(65,30))))
                + (CAST(u_price_num.offset_numerator AS DECIMAL(65,30)) / CAST(u_price_num.offset_denominator AS DECIMAL(65,30)))
            )
            / NULLIF(((CAST(u_price_den.ratio_numerator AS DECIMAL(65,30)) / CAST(u_price_den.ratio_denominator AS DECIMAL(65,30))) + (CAST(u_price_den.offset_numerator AS DECIMAL(65,30)) / CAST(u_price_den.offset_denominator AS DECIMAL(65,30)))), 0)
        )
        / NULLIF(
            (
                (
                    (CAST(q_in.value AS DECIMAL(65,30)) * (CAST(u_in.ratio_numerator AS DECIMAL(65,30)) / CAST(u_in.ratio_denominator AS DECIMAL(65,30))))
                    + (CAST(u_in.offset_numerator AS DECIMAL(65,30)) / CAST(u_in.offset_denominator AS DECIMAL(65,30)))
                )
                - (CAST(bu_unit.offset_numerator AS DECIMAL(65,30)) / CAST(bu_unit.offset_denominator AS DECIMAL(65,30)))
            )
            / NULLIF((CAST(bu_unit.ratio_numerator AS DECIMAL(65,30)) / CAST(bu_unit.ratio_denominator AS DECIMAL(65,30))), 0),
            0
        )
        AS DECIMAL(65,30)
    ) AS unit_price,
    -- unitCost: totalCost / quantityInvoiced
    CAST(
        (
            (CAST(q_in.value AS DECIMAL(65,30)) * (CAST(u_in.ratio_numerator AS DECIMAL(65,30)) / CAST(u_in.ratio_denominator AS DECIMAL(65,30))))
            + (CAST(u_in.offset_numerator AS DECIMAL(65,30)) / CAST(u_in.offset_denominator AS DECIMAL(65,30)))
        )
        *
        (
            (
                (COALESCE(CAST(r_cost.value AS DECIMAL(65,30)), 0) * (COALESCE(CAST(u_cost_num.ratio_numerator AS DECIMAL(65,30)), 1) / COALESCE(CAST(u_cost_num.ratio_denominator AS DECIMAL(65,30)), 1)))
                + (COALESCE(CAST(u_cost_num.offset_numerator AS DECIMAL(65,30)), 0) / COALESCE(CAST(u_cost_num.offset_denominator AS DECIMAL(65,30)), 1))
            )
            / NULLIF(((COALESCE(CAST(u_cost_den.ratio_numerator AS DECIMAL(65,30)), 1) / COALESCE(CAST(u_cost_den.ratio_denominator AS DECIMAL(65,30)), 1)) + (COALESCE(CAST(u_cost_den.offset_numerator AS DECIMAL(65,30)), 0) / COALESCE(CAST(u_cost_den.offset_denominator AS DECIMAL(65,30)), 1))), 0)
        )
        / NULLIF(
            (
                (
                    (CAST(q_in.value AS DECIMAL(65,30)) * (CAST(u_in.ratio_numerator AS DECIMAL(65,30)) / CAST(u_in.ratio_denominator AS DECIMAL(65,30))))
                    + (CAST(u_in.offset_numerator AS DECIMAL(65,30)) / CAST(u_in.offset_denominator AS DECIMAL(65,30)))
                )
                - (CAST(bu_unit.offset_numerator AS DECIMAL(65,30)) / CAST(bu_unit.offset_denominator AS DECIMAL(65,30)))
            )
            / NULLIF((CAST(bu_unit.ratio_numerator AS DECIMAL(65,30)) / CAST(bu_unit.ratio_denominator AS DECIMAL(65,30))), 0),
            0
        )
        AS DECIMAL(65,30)
    ) AS unit_cost,
    -- unitProfit: unitPrice - unitCost
    CAST(
        (
            (
                (CAST(q_in.value AS DECIMAL(65,30)) * (CAST(u_in.ratio_numerator AS DECIMAL(65,30)) / CAST(u_in.ratio_denominator AS DECIMAL(65,30))))
                + (CAST(u_in.offset_numerator AS DECIMAL(65,30)) / CAST(u_in.offset_denominator AS DECIMAL(65,30)))
            )
            *
            (
                (
                    (CAST(r_price.value AS DECIMAL(65,30)) * (CAST(u_price_num.ratio_numerator AS DECIMAL(65,30)) / CAST(u_price_num.ratio_denominator AS DECIMAL(65,30))))
                    + (CAST(u_price_num.offset_numerator AS DECIMAL(65,30)) / CAST(u_price_num.offset_denominator AS DECIMAL(65,30)))
                )
                / NULLIF(((CAST(u_price_den.ratio_numerator AS DECIMAL(65,30)) / CAST(u_price_den.ratio_denominator AS DECIMAL(65,30))) + (CAST(u_price_den.offset_numerator AS DECIMAL(65,30)) / CAST(u_price_den.offset_denominator AS DECIMAL(65,30)))), 0)
            )
            -
            (
                (CAST(q_in.value AS DECIMAL(65,30)) * (CAST(u_in.ratio_numerator AS DECIMAL(65,30)) / CAST(u_in.ratio_denominator AS DECIMAL(65,30))))
                + (CAST(u_in.offset_numerator AS DECIMAL(65,30)) / CAST(u_in.offset_denominator AS DECIMAL(65,30)))
            )
            *
            (
                (
                    (COALESCE(CAST(r_cost.value AS DECIMAL(65,30)), 0) * (COALESCE(CAST(u_cost_num.ratio_numerator AS DECIMAL(65,30)), 1) / COALESCE(CAST(u_cost_num.ratio_denominator AS DECIMAL(65,30)), 1)))
                    + (COALESCE(CAST(u_cost_num.offset_numerator AS DECIMAL(65,30)), 0) / COALESCE(CAST(u_cost_num.offset_denominator AS DECIMAL(65,30)), 1))
                )
                / NULLIF(((COALESCE(CAST(u_cost_den.ratio_numerator AS DECIMAL(65,30)), 1) / COALESCE(CAST(u_cost_den.ratio_denominator AS DECIMAL(65,30)), 1)) + (COALESCE(CAST(u_cost_den.offset_numerator AS DECIMAL(65,30)), 0) / COALESCE(CAST(u_cost_den.offset_denominator AS DECIMAL(65,30)), 1))), 0)
            )
        )
        / NULLIF(
            (
                (
                    (CAST(q_in.value AS DECIMAL(65,30)) * (CAST(u_in.ratio_numerator AS DECIMAL(65,30)) / CAST(u_in.ratio_denominator AS DECIMAL(65,30))))
                    + (CAST(u_in.offset_numerator AS DECIMAL(65,30)) / CAST(u_in.offset_denominator AS DECIMAL(65,30)))
                )
                - (CAST(bu_unit.offset_numerator AS DECIMAL(65,30)) / CAST(bu_unit.offset_denominator AS DECIMAL(65,30)))
            )
            / NULLIF((CAST(bu_unit.ratio_numerator AS DECIMAL(65,30)) / CAST(bu_unit.ratio_denominator AS DECIMAL(65,30))), 0),
            0
        )
        AS DECIMAL(65,30)
    ) AS unit_profit,
    geo.state AS ship_to_state,
    geo.locality AS ship_to_city,
    geo.postal_code AS ship_to_postal_code,
    geo.country AS ship_to_country,
    od.code AS order_discount_code
FROM invoice_line il
JOIN invoice inv FORCE INDEX (invoice_account_created_idx) ON inv.id = il.invoice_id
JOIN sales_order_line sol ON sol.id = il.sales_order_line_id
JOIN sales_order so ON so.id = inv.sales_order_id
JOIN product fg ON fg.id = sol.product_id
JOIN item pb ON pb.id = fg.item_id
JOIN item_category ic ON ic.id = pb.item_category_id
JOIN product_line pl ON pl.id = fg.product_line_id
JOIN quantity q_in ON q_in.id = il.quantity_id
JOIN unit u_in ON u_in.id = q_in.unit_id
LEFT JOIN unit_group ug ON ug.id = ic.unit_group_id
LEFT JOIN unit bu_unit ON bu_unit.id = ug.base_unit_id
LEFT JOIN rate r_price ON r_price.id = sol.unit_price_id
LEFT JOIN unit u_price_num ON u_price_num.id = r_price.numerator_unit_id
LEFT JOIN unit u_price_den ON u_price_den.id = r_price.denominator_unit_id
LEFT JOIN rate r_cost ON r_cost.id = sol.unit_cost_id
LEFT JOIN unit u_cost_num ON u_cost_num.id = r_cost.numerator_unit_id
LEFT JOIN unit u_cost_den ON u_cost_den.id = r_cost.denominator_unit_id
LEFT JOIN account buyer ON so.buyer_account_id = buyer.id
LEFT JOIN account_relation ar ON ar.owner_account_id = so.owner_account_id
    AND ar.counterparty_account_id = so.buyer_account_id
    AND ar.account_relation_role_code = 'customer'
LEFT JOIN account_relation parent_ar ON parent_ar.id = ar.parent_account_relation_id
    AND parent_ar.owner_account_id = ar.owner_account_id
LEFT JOIN account_group ag ON ag.id = ar.account_group_id
LEFT JOIN account_user sr ON so.sales_rep_id = sr.id
LEFT JOIN `user` sr_user ON sr_user.id = sr.user_id
LEFT JOIN address ship_addr ON ship_addr.id = so.shipping_address_id
LEFT JOIN geolocation geo ON geo.id = ship_addr.geolocation_id
LEFT JOIN order_discount od ON so.order_discount_id = od.id
WHERE inv.account_id = sqlc.arg('owner_account_id')
  AND inv.created_at >= sqlc.arg('start_date')
  AND inv.created_at <= sqlc.arg('end_date')
  AND (sqlc.arg('include_sales_rep_filter') = false OR so.sales_rep_id IN (sqlc.slice('sales_rep_ids')))
  AND (sqlc.arg('include_product_line_filter') = false OR fg.product_line_id IN (sqlc.slice('product_line_ids')))
  AND (sqlc.arg('include_buyer_filter') = false OR so.buyer_account_id IN (sqlc.slice('buyer_ids')))
ORDER BY inv.created_at ASC;

-- name: GetProductionCostEntries :many
SELECT
    it.id AS item_id,
    it.sku AS product_sku,
    it.description AS product_description,
    pl.name AS product_line,
    COALESCE(SUM(CAST(b_q.value AS DECIMAL(65,30))), 0) AS total_quantity,
    0 AS total_cost,
    0 AS cost_per_unit,
    b_u.abbreviation AS unit
FROM batch b
JOIN item it ON it.id = b.item_id
JOIN quantity b_q ON b_q.id = b.quantity_id
JOIN unit b_u ON b_u.id = b_q.unit_id
LEFT JOIN product p ON p.item_id = it.id AND p.product_type_code = 'sale'
LEFT JOIN product_line pl ON pl.id = p.product_line_id
WHERE b.account_id = sqlc.arg('owner_account_id')
  AND b.closed_at IS NOT NULL
GROUP BY it.id, it.sku, it.description, pl.name, b_u.abbreviation;

-- name: GetManufacturingProduction :one
SELECT COALESCE(SUM(CAST(b_q.value AS DECIMAL(65,30))), 0) AS total_production
FROM batch b
JOIN quantity b_q ON b_q.id = b.quantity_id
WHERE b.account_id = sqlc.arg('owner_account_id')
  AND b.scanned_at >= sqlc.arg('start_date')
  AND b.scanned_at <= sqlc.arg('end_date');

-- name: GetManufacturingQuality :one
SELECT
    CASE
        WHEN COALESCE(total_qty, 0) = 0 THEN 0
        ELSE good_qty / total_qty
    END AS quality
FROM (
    SELECT
        COALESCE(SUM(CAST(b_q.value AS DECIMAL(65,30))), 0) AS good_qty,
        COALESCE(SUM(CAST(b_q.value AS DECIMAL(65,30))), 0)
            + COALESCE(SUM(CAST(COALESCE(wq.value, 0) AS DECIMAL(65,30))), 0)
            + COALESCE(SUM(CAST(COALESCE(sq.value, 0) AS DECIMAL(65,30))), 0) AS total_qty
    FROM batch b
    JOIN quantity b_q ON b_q.id = b.quantity_id
    LEFT JOIN quantity wq ON wq.id = b.waste_quantity_id
    LEFT JOIN quantity sq ON sq.id = b.seconds_quantity_id
    WHERE b.account_id = sqlc.arg('owner_account_id')
      AND b.scanned_at >= sqlc.arg('start_date')
      AND b.scanned_at <= sqlc.arg('end_date')
) sub;

-- name: GetManufacturingCostsPerUnit :one
SELECT
    COALESCE(SUM(
        (
            (q_in.value * (u_in.ratio_numerator / u_in.ratio_denominator))
            + (u_in.offset_numerator / u_in.offset_denominator)
        )
        *
        (
            (
                (COALESCE(r_cost.value, 0) * (COALESCE(u_cost_num.ratio_numerator, 1) / COALESCE(u_cost_num.ratio_denominator, 1)))
                + (COALESCE(u_cost_num.offset_numerator, 0) / COALESCE(u_cost_num.offset_denominator, 1))
            )
            / NULLIF(((COALESCE(u_cost_den.ratio_numerator, 1) / COALESCE(u_cost_den.ratio_denominator, 1)) + (COALESCE(u_cost_den.offset_numerator, 0) / COALESCE(u_cost_den.offset_denominator, 1))), 0)
        )
    ), 0) AS total_cost,
    COALESCE(SUM(
        (q_in.value * (u_in.ratio_numerator / u_in.ratio_denominator))
        + (u_in.offset_numerator / u_in.offset_denominator)
    ), 0) AS total_quantity
FROM invoice_line il
JOIN invoice i ON i.id = il.invoice_id
JOIN sales_order_line sol ON sol.id = il.sales_order_line_id
JOIN quantity q_in ON q_in.id = il.quantity_id
JOIN unit u_in ON u_in.id = q_in.unit_id
LEFT JOIN rate r_cost ON r_cost.id = sol.unit_cost_id
LEFT JOIN unit u_cost_num ON u_cost_num.id = r_cost.numerator_unit_id
LEFT JOIN unit u_cost_den ON u_cost_den.id = r_cost.denominator_unit_id
WHERE i.account_id = sqlc.arg('owner_account_id')
  AND i.created_at >= sqlc.arg('start_date')
  AND i.created_at <= sqlc.arg('end_date');

-- name: GetManufacturingMargin :one
SELECT
    COALESCE(SUM(
        (
            (q_in.value * (u_in.ratio_numerator / u_in.ratio_denominator))
            + (u_in.offset_numerator / u_in.offset_denominator)
        )
        *
        (
            (
                (r_price.value * (u_price_num.ratio_numerator / u_price_num.ratio_denominator))
                + (u_price_num.offset_numerator / u_price_num.offset_denominator)
            )
            / NULLIF(((u_price_den.ratio_numerator / u_price_den.ratio_denominator) + (u_price_den.offset_numerator / u_price_den.offset_denominator)), 0)
        )
        -
        (
            (q_in.value * (u_in.ratio_numerator / u_in.ratio_denominator))
            + (u_in.offset_numerator / u_in.offset_denominator)
        )
        *
        (
            (
                (COALESCE(r_cost.value, 0) * (COALESCE(u_cost_num.ratio_numerator, 1) / COALESCE(u_cost_num.ratio_denominator, 1)))
                + (COALESCE(u_cost_num.offset_numerator, 0) / COALESCE(u_cost_num.offset_denominator, 1))
            )
            / NULLIF(((COALESCE(u_cost_den.ratio_numerator, 1) / COALESCE(u_cost_den.ratio_denominator, 1)) + (COALESCE(u_cost_den.offset_numerator, 0) / COALESCE(u_cost_den.offset_denominator, 1))), 0)
        )
    ), 0) AS total_profit,
    COALESCE(SUM(
        (
            (q_in.value * (u_in.ratio_numerator / u_in.ratio_denominator))
            + (u_in.offset_numerator / u_in.offset_denominator)
        )
        *
        (
            (
                (r_price.value * (u_price_num.ratio_numerator / u_price_num.ratio_denominator))
                + (u_price_num.offset_numerator / u_price_num.offset_denominator)
            )
            / NULLIF(((u_price_den.ratio_numerator / u_price_den.ratio_denominator) + (u_price_den.offset_numerator / u_price_den.offset_denominator)), 0)
        )
    ), 0) AS total_invoiced
FROM invoice_line il
JOIN invoice i ON i.id = il.invoice_id
JOIN sales_order_line sol ON sol.id = il.sales_order_line_id
JOIN quantity q_in ON q_in.id = il.quantity_id
JOIN unit u_in ON u_in.id = q_in.unit_id
LEFT JOIN rate r_price ON r_price.id = sol.unit_price_id
LEFT JOIN unit u_price_num ON u_price_num.id = r_price.numerator_unit_id
LEFT JOIN unit u_price_den ON u_price_den.id = r_price.denominator_unit_id
LEFT JOIN rate r_cost ON r_cost.id = sol.unit_cost_id
LEFT JOIN unit u_cost_num ON u_cost_num.id = r_cost.numerator_unit_id
LEFT JOIN unit u_cost_den ON u_cost_den.id = r_cost.denominator_unit_id
WHERE i.account_id = sqlc.arg('owner_account_id')
  AND i.created_at >= sqlc.arg('start_date')
  AND i.created_at <= sqlc.arg('end_date');

-- name: GetManufacturingLaborEfficiency :one
-- Labor efficiency weighs each batch's good, waste and seconds counts by the standard labor they stand for:
-- the step's labor time in hours per base unit, times the count in base units. A labor time quantified in
-- another dimension than the count has no conversion, so it is read as per the count's own unit.
SELECT
    CAST(COALESCE(SUM(COALESCE(CAST(qf.value AS DECIMAL(65,30)), 0) * CAST(lt.value AS DECIMAL(65,30)) * (ltn.ratio_numerator / ltn.ratio_denominator) * CASE WHEN ltd.unit_dimension_code = qfu.unit_dimension_code THEN (qfu.ratio_numerator / qfu.ratio_denominator) / (ltd.ratio_numerator / ltd.ratio_denominator) ELSE 1 END), 0) AS DECIMAL(65,30)) AS labor_quantity,
    CAST(COALESCE(SUM(COALESCE(CAST(qw.value AS DECIMAL(65,30)), 0) * CAST(lt.value AS DECIMAL(65,30)) * (ltn.ratio_numerator / ltn.ratio_denominator) * CASE WHEN ltd.unit_dimension_code = qwu.unit_dimension_code THEN (qwu.ratio_numerator / qwu.ratio_denominator) / (ltd.ratio_numerator / ltd.ratio_denominator) ELSE 1 END), 0) AS DECIMAL(65,30)) AS labor_waste,
    CAST(COALESCE(SUM(COALESCE(CAST(qs.value AS DECIMAL(65,30)), 0) * CAST(lt.value AS DECIMAL(65,30)) * (ltn.ratio_numerator / ltn.ratio_denominator) * CASE WHEN ltd.unit_dimension_code = qsu.unit_dimension_code THEN (qsu.ratio_numerator / qsu.ratio_denominator) / (ltd.ratio_numerator / ltd.ratio_denominator) ELSE 1 END), 0) AS DECIMAL(65,30)) AS labor_seconds
FROM batch b
LEFT JOIN quantity qf ON qf.id = b.quantity_id
LEFT JOIN unit qfu ON qfu.id = qf.unit_id
LEFT JOIN quantity qw ON qw.id = b.waste_quantity_id
LEFT JOIN unit qwu ON qwu.id = qw.unit_id
LEFT JOIN quantity qs ON qs.id = b.seconds_quantity_id
LEFT JOIN unit qsu ON qsu.id = qs.unit_id
LEFT JOIN production_step ps ON ps.id = b.production_step_id
LEFT JOIN rate lt ON lt.id = ps.labor_time_id
LEFT JOIN unit ltn ON ltn.id = lt.numerator_unit_id
LEFT JOIN unit ltd ON ltd.id = lt.denominator_unit_id
WHERE b.account_id = sqlc.arg('owner_account_id')
  AND b.scanned_at >= sqlc.arg('start_date')
  AND b.scanned_at <= sqlc.arg('end_date');

-- name: GetManufacturingBatchBatchMetrics :one
-- Combined batch-table query for production, quality, and labor efficiency. Uses scanned_at for date filtering to match legacy dashboard behavior.
-- Labor efficiency weighs each batch's good, waste and seconds counts by the standard labor they stand for:
-- the step's labor time in hours per base unit, times the count in base units. A labor time quantified in
-- another dimension than the count has no conversion, so it is read as per the count's own unit.
SELECT
    CAST(COALESCE(SUM(CAST(qf.value AS DECIMAL(65,30))), 0) AS DECIMAL(65,30)) AS total_quantity,
    CAST(COALESCE(SUM(CAST(COALESCE(qw.value, 0) AS DECIMAL(65,30))), 0) AS DECIMAL(65,30)) AS total_waste,
    CAST(COALESCE(SUM(CAST(COALESCE(qs.value, 0) AS DECIMAL(65,30))), 0) AS DECIMAL(65,30)) AS total_seconds,
    CAST(COALESCE(SUM(COALESCE(CAST(qf.value AS DECIMAL(65,30)), 0) * CAST(lt.value AS DECIMAL(65,30)) * (ltn.ratio_numerator / ltn.ratio_denominator) * CASE WHEN ltd.unit_dimension_code = qfu.unit_dimension_code THEN (qfu.ratio_numerator / qfu.ratio_denominator) / (ltd.ratio_numerator / ltd.ratio_denominator) ELSE 1 END), 0) AS DECIMAL(65,30)) AS labor_quantity,
    CAST(COALESCE(SUM(COALESCE(CAST(qw.value AS DECIMAL(65,30)), 0) * CAST(lt.value AS DECIMAL(65,30)) * (ltn.ratio_numerator / ltn.ratio_denominator) * CASE WHEN ltd.unit_dimension_code = qwu.unit_dimension_code THEN (qwu.ratio_numerator / qwu.ratio_denominator) / (ltd.ratio_numerator / ltd.ratio_denominator) ELSE 1 END), 0) AS DECIMAL(65,30)) AS labor_waste,
    CAST(COALESCE(SUM(COALESCE(CAST(qs.value AS DECIMAL(65,30)), 0) * CAST(lt.value AS DECIMAL(65,30)) * (ltn.ratio_numerator / ltn.ratio_denominator) * CASE WHEN ltd.unit_dimension_code = qsu.unit_dimension_code THEN (qsu.ratio_numerator / qsu.ratio_denominator) / (ltd.ratio_numerator / ltd.ratio_denominator) ELSE 1 END), 0) AS DECIMAL(65,30)) AS labor_seconds
FROM batch b
LEFT JOIN quantity qf ON qf.id = b.quantity_id
LEFT JOIN unit qfu ON qfu.id = qf.unit_id
LEFT JOIN quantity qw ON qw.id = b.waste_quantity_id
LEFT JOIN unit qwu ON qwu.id = qw.unit_id
LEFT JOIN quantity qs ON qs.id = b.seconds_quantity_id
LEFT JOIN unit qsu ON qsu.id = qs.unit_id
LEFT JOIN production_step ps ON ps.id = b.production_step_id
LEFT JOIN rate lt ON lt.id = ps.labor_time_id
LEFT JOIN unit ltn ON ltn.id = lt.numerator_unit_id
LEFT JOIN unit ltd ON ltd.id = lt.denominator_unit_id
WHERE b.account_id = sqlc.arg('owner_account_id')
  AND b.scanned_at >= sqlc.arg('start_date')
  AND b.scanned_at <= sqlc.arg('end_date');

-- name: GetManufacturingBatchInvoiceMetrics :one
-- Combined invoice-table query for costs per unit and margin. Uses unit conversion logic to normalize quantities, costs, and prices.
SELECT
    COALESCE(SUM(
        (
            (CAST(q_in.value AS DECIMAL(65,30)) * (CAST(u_in.ratio_numerator AS DECIMAL(65,30)) / CAST(u_in.ratio_denominator AS DECIMAL(65,30))))
            + (CAST(u_in.offset_numerator AS DECIMAL(65,30)) / CAST(u_in.offset_denominator AS DECIMAL(65,30)))
        )
        *
        (
            (
                (COALESCE(CAST(r_cost.value AS DECIMAL(65,30)), 0) * (COALESCE(CAST(u_cost_num.ratio_numerator AS DECIMAL(65,30)), 1) / COALESCE(CAST(u_cost_num.ratio_denominator AS DECIMAL(65,30)), 1)))
                + (COALESCE(CAST(u_cost_num.offset_numerator AS DECIMAL(65,30)), 0) / COALESCE(CAST(u_cost_num.offset_denominator AS DECIMAL(65,30)), 1))
            )
            / NULLIF(((COALESCE(CAST(u_cost_den.ratio_numerator AS DECIMAL(65,30)), 1) / COALESCE(CAST(u_cost_den.ratio_denominator AS DECIMAL(65,30)), 1)) + (COALESCE(CAST(u_cost_den.offset_numerator AS DECIMAL(65,30)), 0) / COALESCE(CAST(u_cost_den.offset_denominator AS DECIMAL(65,30)), 1))), 0)
        )
    ), 0) AS total_cost,
    COALESCE(SUM(
        (CAST(q_in.value AS DECIMAL(65,30)) * (CAST(u_in.ratio_numerator AS DECIMAL(65,30)) / CAST(u_in.ratio_denominator AS DECIMAL(65,30))))
        + (CAST(u_in.offset_numerator AS DECIMAL(65,30)) / CAST(u_in.offset_denominator AS DECIMAL(65,30)))
    ), 0) AS total_quantity,
    COALESCE(SUM(
        (
            (CAST(q_in.value AS DECIMAL(65,30)) * (CAST(u_in.ratio_numerator AS DECIMAL(65,30)) / CAST(u_in.ratio_denominator AS DECIMAL(65,30))))
            + (CAST(u_in.offset_numerator AS DECIMAL(65,30)) / CAST(u_in.offset_denominator AS DECIMAL(65,30)))
        )
        *
        (
            (
                (CAST(r_price.value AS DECIMAL(65,30)) * (CAST(u_price_num.ratio_numerator AS DECIMAL(65,30)) / CAST(u_price_num.ratio_denominator AS DECIMAL(65,30))))
                + (CAST(u_price_num.offset_numerator AS DECIMAL(65,30)) / CAST(u_price_num.offset_denominator AS DECIMAL(65,30)))
            )
            / NULLIF(((CAST(u_price_den.ratio_numerator AS DECIMAL(65,30)) / CAST(u_price_den.ratio_denominator AS DECIMAL(65,30))) + (CAST(u_price_den.offset_numerator AS DECIMAL(65,30)) / CAST(u_price_den.offset_denominator AS DECIMAL(65,30)))), 0)
        )
    ), 0) AS total_revenue,
    COALESCE(SUM(
        (
            (CAST(q_in.value AS DECIMAL(65,30)) * (CAST(u_in.ratio_numerator AS DECIMAL(65,30)) / CAST(u_in.ratio_denominator AS DECIMAL(65,30))))
            + (CAST(u_in.offset_numerator AS DECIMAL(65,30)) / CAST(u_in.offset_denominator AS DECIMAL(65,30)))
        )
        *
        (
            (
                (CAST(r_price.value AS DECIMAL(65,30)) * (CAST(u_price_num.ratio_numerator AS DECIMAL(65,30)) / CAST(u_price_num.ratio_denominator AS DECIMAL(65,30))))
                + (CAST(u_price_num.offset_numerator AS DECIMAL(65,30)) / CAST(u_price_num.offset_denominator AS DECIMAL(65,30)))
            )
            / NULLIF(((CAST(u_price_den.ratio_numerator AS DECIMAL(65,30)) / CAST(u_price_den.ratio_denominator AS DECIMAL(65,30))) + (CAST(u_price_den.offset_numerator AS DECIMAL(65,30)) / CAST(u_price_den.offset_denominator AS DECIMAL(65,30)))), 0)
        )
        -
        (
            (CAST(q_in.value AS DECIMAL(65,30)) * (CAST(u_in.ratio_numerator AS DECIMAL(65,30)) / CAST(u_in.ratio_denominator AS DECIMAL(65,30))))
            + (CAST(u_in.offset_numerator AS DECIMAL(65,30)) / CAST(u_in.offset_denominator AS DECIMAL(65,30)))
        )
        *
        (
            (
                (COALESCE(CAST(r_cost.value AS DECIMAL(65,30)), 0) * (COALESCE(CAST(u_cost_num.ratio_numerator AS DECIMAL(65,30)), 1) / COALESCE(CAST(u_cost_num.ratio_denominator AS DECIMAL(65,30)), 1)))
                + (COALESCE(CAST(u_cost_num.offset_numerator AS DECIMAL(65,30)), 0) / COALESCE(CAST(u_cost_num.offset_denominator AS DECIMAL(65,30)), 1))
            )
            / NULLIF(((COALESCE(CAST(u_cost_den.ratio_numerator AS DECIMAL(65,30)), 1) / COALESCE(CAST(u_cost_den.ratio_denominator AS DECIMAL(65,30)), 1)) + (COALESCE(CAST(u_cost_den.offset_numerator AS DECIMAL(65,30)), 0) / COALESCE(CAST(u_cost_den.offset_denominator AS DECIMAL(65,30)), 1))), 0)
        )
    ), 0) AS total_profit
FROM invoice_line il
JOIN invoice i ON i.id = il.invoice_id
JOIN sales_order_line sol ON sol.id = il.sales_order_line_id
JOIN quantity q_in ON q_in.id = il.quantity_id
JOIN unit u_in ON u_in.id = q_in.unit_id
LEFT JOIN rate r_cost ON r_cost.id = sol.unit_cost_id
LEFT JOIN unit u_cost_num ON u_cost_num.id = r_cost.numerator_unit_id
LEFT JOIN unit u_cost_den ON u_cost_den.id = r_cost.denominator_unit_id
LEFT JOIN rate r_price ON r_price.id = sol.unit_price_id
LEFT JOIN unit u_price_num ON u_price_num.id = r_price.numerator_unit_id
LEFT JOIN unit u_price_den ON u_price_den.id = r_price.denominator_unit_id
WHERE i.account_id = sqlc.arg('owner_account_id')
  AND i.created_at >= sqlc.arg('start_date')
  AND i.created_at <= sqlc.arg('end_date');

-- GetOeeDepartmentData returns the unit counts and the standard time earned per department.
--
-- standard_seconds_earned is the numerator of OEE Performance: the time the work *should* have taken at the production step's own labor rate. The rate is a Rate whose numerator unit decides its scale, converted to seconds via that unit's ratio_numerator/ratio_denominator (its size in the time base, the hour) rather than a hardcoded abbreviation table — the same conversion the solver applies in SecondsPerUnitFromLaborTime, which now reads the same columns, so the two never disagree about what a run rate means and both handle day and account-defined time units.
--
-- Seconds-grade units count toward output but not toward good: they are sellable, but they are not first-pass quality, and leaving them out of the denominator would report a plant that produces nothing but irregulars as 100% quality.
-- name: GetOeeDepartmentData :many
SELECT
    COALESCE(d.id, 'unassigned') AS department_id,
    COALESCE(d.name, 'Unassigned') AS department_name,
    CAST(COALESCE(SUM(COALESCE(qf.value * (u_qf.ratio_numerator / u_qf.ratio_denominator), 0)), 0) AS DECIMAL(65,30)) AS good_units,
    CAST(COALESCE(SUM(COALESCE(qw.value * (u_qw.ratio_numerator / u_qw.ratio_denominator), 0)), 0) AS DECIMAL(65,30)) AS waste_units,
    CAST(COALESCE(SUM(COALESCE(qs.value * (u_qs.ratio_numerator / u_qs.ratio_denominator), 0)), 0) AS DECIMAL(65,30)) AS seconds_units,
    CAST(COALESCE(SUM(
        (
            COALESCE(qf.value * (u_qf.ratio_numerator / u_qf.ratio_denominator), 0)
            + COALESCE(qw.value * (u_qw.ratio_numerator / u_qw.ratio_denominator), 0)
            + COALESCE(qs.value * (u_qs.ratio_numerator / u_qs.ratio_denominator), 0)
        ) * COALESCE(
            -- Labor time to seconds per BASE quantity unit, via both of the rate's unit ratios, not a
            -- fixed abbreviation table. The numerator unit's ratio is its size in the time dimension's
            -- base (the hour), so value * (ratio) * 3600 gives seconds; handles day and any
            -- account-defined time unit. The denominator unit's ratio is its size in the quantity base,
            -- and the rate is DIVIDED by it because the quantities above are already normalized to that
            -- base: a step rated 554 sec/pr is 277 sec per each, and skipping this division counts every
            -- pair-rated step twice. Stays in step with the solver's SecondsPerUnitFromLaborTime, which
            -- reads the same four columns. A missing numerator unit leaves the product NULL, so the
            -- COALESCE books no standard time rather than guessing a scale; a missing denominator unit
            -- divides by one, leaving a rate already quoted per base unit alone.
            labor_time.value * (labor_time_unit.ratio_numerator / labor_time_unit.ratio_denominator) * 3600
                / COALESCE(NULLIF(labor_time_qty_unit.ratio_numerator / labor_time_qty_unit.ratio_denominator, 0), 1),
            0
        )
    ), 0) AS DECIMAL(65,30)) AS standard_seconds_earned
-- FORCE INDEX pins the (account_id, scanned_at) range scan. With a year window on a
-- plant with several years of scans the optimizer otherwise picks the wider
-- (account_id, scanning_station_id, scanned_at, id) composite, which cannot use scanned_at
-- as a range bound and so reads every batch the account ever scanned — the 14s read that
-- times the trend out.
FROM batch b FORCE INDEX (batch_account_id_scanned_at_idx)
LEFT JOIN quantity qf ON qf.id = b.quantity_id
LEFT JOIN unit u_qf ON u_qf.id = qf.unit_id
LEFT JOIN quantity qw ON qw.id = b.waste_quantity_id
LEFT JOIN unit u_qw ON u_qw.id = qw.unit_id
LEFT JOIN quantity qs ON qs.id = b.seconds_quantity_id
LEFT JOIN unit u_qs ON u_qs.id = qs.unit_id
LEFT JOIN scanning_station ss ON ss.id = b.scanning_station_id
LEFT JOIN department d ON d.id = ss.department_id
LEFT JOIN production_step ps ON ps.id = b.production_step_id
LEFT JOIN rate labor_time ON labor_time.id = ps.labor_time_id
LEFT JOIN unit labor_time_unit ON labor_time_unit.id = labor_time.numerator_unit_id
LEFT JOIN unit labor_time_qty_unit ON labor_time_qty_unit.id = labor_time.denominator_unit_id
WHERE b.account_id = sqlc.arg('owner_account_id')
  AND b.scanned_at >= sqlc.arg('start_date')
  AND b.scanned_at <= sqlc.arg('end_date')
GROUP BY d.id, d.name;

-- GetOeeDepartmentDataForMachines is GetOeeDepartmentData restricted to production on a given set of machines — the machines the plan scheduled.
--
-- Performance divides the standard time earned by the scheduled machines' run time, so counting output from machines that were never scheduled would report a department running many times faster than the plant it was measured against. The service calls this variant for scheduled departments and the unrestricted query above for everything else; the two SELECT lists (including the standard_seconds_earned conversion) must stay identical so the scoped and whole-floor reads can never disagree about what a run rate means.
-- name: GetOeeDepartmentDataForMachines :many
SELECT
    COALESCE(d.id, 'unassigned') AS department_id,
    COALESCE(d.name, 'Unassigned') AS department_name,
    CAST(COALESCE(SUM(COALESCE(qf.value * (u_qf.ratio_numerator / u_qf.ratio_denominator), 0)), 0) AS DECIMAL(65,30)) AS good_units,
    CAST(COALESCE(SUM(COALESCE(qw.value * (u_qw.ratio_numerator / u_qw.ratio_denominator), 0)), 0) AS DECIMAL(65,30)) AS waste_units,
    CAST(COALESCE(SUM(COALESCE(qs.value * (u_qs.ratio_numerator / u_qs.ratio_denominator), 0)), 0) AS DECIMAL(65,30)) AS seconds_units,
    CAST(COALESCE(SUM(
        (
            COALESCE(qf.value * (u_qf.ratio_numerator / u_qf.ratio_denominator), 0)
            + COALESCE(qw.value * (u_qw.ratio_numerator / u_qw.ratio_denominator), 0)
            + COALESCE(qs.value * (u_qs.ratio_numerator / u_qs.ratio_denominator), 0)
        ) * COALESCE(
            -- Labor time to seconds per BASE quantity unit, via both of the rate's unit ratios, not a
            -- fixed abbreviation table. The numerator unit's ratio is its size in the time dimension's
            -- base (the hour), so value * (ratio) * 3600 gives seconds; handles day and any
            -- account-defined time unit. The denominator unit's ratio is its size in the quantity base,
            -- and the rate is DIVIDED by it because the quantities above are already normalized to that
            -- base: a step rated 554 sec/pr is 277 sec per each, and skipping this division counts every
            -- pair-rated step twice. Stays in step with the solver's SecondsPerUnitFromLaborTime, which
            -- reads the same four columns. A missing numerator unit leaves the product NULL, so the
            -- COALESCE books no standard time rather than guessing a scale; a missing denominator unit
            -- divides by one, leaving a rate already quoted per base unit alone.
            labor_time.value * (labor_time_unit.ratio_numerator / labor_time_unit.ratio_denominator) * 3600
                / COALESCE(NULLIF(labor_time_qty_unit.ratio_numerator / labor_time_qty_unit.ratio_denominator, 0), 1),
            0
        )
    ), 0) AS DECIMAL(65,30)) AS standard_seconds_earned
-- FORCE INDEX pins the (account_id, scanned_at) range scan; see GetOeeDepartmentData.
FROM batch b FORCE INDEX (batch_account_id_scanned_at_idx)
LEFT JOIN quantity qf ON qf.id = b.quantity_id
LEFT JOIN unit u_qf ON u_qf.id = qf.unit_id
LEFT JOIN quantity qw ON qw.id = b.waste_quantity_id
LEFT JOIN unit u_qw ON u_qw.id = qw.unit_id
LEFT JOIN quantity qs ON qs.id = b.seconds_quantity_id
LEFT JOIN unit u_qs ON u_qs.id = qs.unit_id
LEFT JOIN scanning_station ss ON ss.id = b.scanning_station_id
LEFT JOIN department d ON d.id = ss.department_id
LEFT JOIN production_step ps ON ps.id = b.production_step_id
LEFT JOIN rate labor_time ON labor_time.id = ps.labor_time_id
LEFT JOIN unit labor_time_unit ON labor_time_unit.id = labor_time.numerator_unit_id
LEFT JOIN unit labor_time_qty_unit ON labor_time_qty_unit.id = labor_time.denominator_unit_id
WHERE b.account_id = sqlc.arg('owner_account_id')
  AND b.scanned_at >= sqlc.arg('start_date')
  AND b.scanned_at <= sqlc.arg('end_date')
  AND EXISTS (
      SELECT 1 FROM _batches_machines bm
      WHERE bm.A = b.id AND bm.B IN (sqlc.slice('machine_ids'))
  )
GROUP BY d.id, d.name;

-- GetSaleProductItemIDs returns every sale product's item and product line, deleted items included: their stock is still on hand.
-- The account's items drive: left to choose, the planner reads every tenant's sale products by type.
-- name: GetSaleProductItemIDs :many
SELECT STRAIGHT_JOIN
    p.item_id,
    p.product_line_id
FROM item i FORCE INDEX (item_account_created_idx)
JOIN product p ON p.item_id = i.id
WHERE i.account_id = sqlc.arg('owner_account_id')
  AND p.product_type_code = 'sale';

-- GetWeeksOfSalesOnHand is each item's available receipts net of what has been drawn against them, in the base unit of the item's dimension, deleted items included. Nothing is clamped: a receipt drawn on for more than it holds nets negative, as the dashboard's ledger did.
-- name: GetWeeksOfSalesOnHand :many
SELECT
    i.id AS item_id,
    CAST((
        COALESCE(
            (SELECT SUM((q.value * (u.ratio_numerator / u.ratio_denominator)) + (u.offset_numerator / u.offset_denominator))
             FROM inventory_receipt ir
             JOIN quantity q ON q.id = ir.quantity_id
             JOIN unit u ON u.id = q.unit_id
             WHERE ir.item_id = i.id
             AND (ir.owner_account_id = sqlc.arg('account_id') OR ir.holder_account_id = sqlc.arg('account_id'))
             AND ir.status_code = 'available'), 0
        ) - COALESCE(
            (SELECT SUM((aq.value * (au.ratio_numerator / au.ratio_denominator)) + (au.offset_numerator / au.offset_denominator))
             FROM inventory_receipt ir2
             JOIN inventory_allocation ia2 ON ia2.inventory_receipt_id = ir2.id
             JOIN quantity aq ON aq.id = ia2.quantity_id
             JOIN unit au ON au.id = aq.unit_id
             WHERE ir2.item_id = i.id
             AND (ir2.owner_account_id = sqlc.arg('account_id') OR ir2.holder_account_id = sqlc.arg('account_id'))
             AND ir2.status_code = 'available'), 0
        )
    ) AS DECIMAL(65,30)) AS on_hand
FROM item i
WHERE i.id IN (sqlc.slice('item_ids'))
  AND i.account_id = sqlc.arg('account_id');

-- GetOrderQuantitiesByProductLines returns, for each requested product line, the quantity ordered on sales orders issued in the window in the line's base unit, and that unit. A line with no orders still returns a row with zero demand.
-- base_ratio is the base unit's size in its dimension's base unit, for bringing on-hand stock to it.
-- The orders drive, ranged by issue date: left to choose, the planner starts from the product lines' every line ever ordered.
-- name: GetOrderQuantitiesByProductLines :many
SELECT
    pl.id AS product_line_id,
    CAST(COALESCE(
        (demand.total_quantity - (bu.offset_numerator / bu.offset_denominator)) / NULLIF(bu.ratio_numerator / bu.ratio_denominator, 0),
        0
    ) AS DECIMAL(65,30)) AS total_quantity,
    COALESCE(bu.abbreviation, '') AS unit_abbreviation,
    COALESCE(ug.unit_type_code, '') AS unit_type,
    CAST(COALESCE(bu.ratio_numerator / bu.ratio_denominator, 1) AS DECIMAL(65,30)) AS base_ratio
FROM product_line pl
LEFT JOIN unit_group ug ON ug.id = pl.unit_group_id
LEFT JOIN unit bu ON bu.id = ug.base_unit_id
LEFT JOIN (
    SELECT STRAIGHT_JOIN
        p.product_line_id,
        SUM((sol_q.value * (sol_u.ratio_numerator / sol_u.ratio_denominator)) + (sol_u.offset_numerator / sol_u.offset_denominator)) AS total_quantity
    FROM sales_order so FORCE INDEX (sales_order_owner_type_issued_idx)
    JOIN sales_order_line sol ON sol.sales_order_id = so.id
    JOIN product p ON p.id = sol.product_id
    JOIN quantity sol_q ON sol_q.id = sol.quantity_id
    JOIN unit sol_u ON sol_u.id = sol_q.unit_id
    WHERE so.owner_account_id = sqlc.arg('owner_account_id')
      AND so.sales_order_type_code = 'sales_order'
      AND p.product_line_id IN (sqlc.slice('demand_product_line_ids'))
      AND so.issued_at >= sqlc.arg('start_date')
      AND so.issued_at <= sqlc.arg('end_date')
    GROUP BY p.product_line_id
) demand ON demand.product_line_id = pl.id
WHERE pl.id IN (sqlc.slice('product_line_ids'));

-- name: GetProductLineInfo :many
SELECT
    pl.id,
    pl.name
FROM product_line pl
WHERE pl.id IN (sqlc.slice('product_line_ids'))
  AND (pl.account_id = sqlc.arg('owner_account_id') OR pl.account_id IS NULL)
ORDER BY pl.name ASC, pl.id ASC;

-- name: GetDeliveryEntries :many
SELECT
    inv.number AS invoice_number,
    inv.created_at AS invoiced_at,
    so.issued_at AS issued_at,
    so.completed_at AS completed_at,
    so.first_ship_at AS first_ship_at,
    so.promised_at AS promised_at
FROM invoice inv
JOIN sales_order so ON so.id = inv.sales_order_id
WHERE inv.account_id = sqlc.arg('owner_account_id')
  AND inv.created_at >= sqlc.arg('start_date')
  AND inv.created_at <= sqlc.arg('end_date')
ORDER BY inv.created_at ASC;

-- GetMaterialsWithDetails lists the account's materials with what converting their stock needs: the order point's unit, and the item's base unit for a material whose order point row is gone.
-- The account's items drive: material carries no account, and read first it is every tenant's.
-- name: GetMaterialsWithDetails :many
SELECT STRAIGHT_JOIN
    m.id AS material_id,
    it.id AS item_id,
    it.sku AS item_sku,
    it.description AS item_description,
    op_q.value AS order_point_value,
    op_u.name AS order_point_unit_name,
    op_u.abbreviation AS order_point_unit_abbreviation,
    op_u.unit_dimension_code AS order_point_unit_type,
    op_u.ratio_numerator AS order_point_unit_ratio_numerator,
    op_u.ratio_denominator AS order_point_unit_ratio_denominator,
    op_u.offset_numerator AS order_point_unit_offset_numerator,
    op_u.offset_denominator AS order_point_unit_offset_denominator,
    lt_q.value AS lead_time_value,
    lt_u.name AS lead_time_unit_name,
    lt_u.abbreviation AS lead_time_unit_abbreviation,
    lt_u.unit_dimension_code AS lead_time_unit_type,
    ug.id AS unit_group_id,
    ug.name AS unit_group_name,
    bu.name AS base_unit_name,
    bu.abbreviation AS base_unit_abbreviation,
    bu.unit_dimension_code AS base_unit_type,
    bu.ratio_numerator AS base_unit_ratio_numerator,
    bu.ratio_denominator AS base_unit_ratio_denominator,
    bu.offset_numerator AS base_unit_offset_numerator,
    bu.offset_denominator AS base_unit_offset_denominator
FROM item it FORCE INDEX (item_account_created_idx)
JOIN material m ON m.item_id = it.id
LEFT JOIN quantity op_q ON op_q.id = m.order_point_id
LEFT JOIN unit op_u ON op_u.id = op_q.unit_id
LEFT JOIN quantity lt_q ON lt_q.id = m.lead_time_id
LEFT JOIN unit lt_u ON lt_u.id = lt_q.unit_id
JOIN item_category ic ON ic.id = it.item_category_id
JOIN unit_group ug ON ug.id = ic.unit_group_id
JOIN unit bu ON bu.id = ug.base_unit_id
WHERE it.account_id = sqlc.arg('owner_account_id')
  AND it.deleted_at IS NULL;

-- name: GetMaterialUnitGroupUnits :many
SELECT
    ugu.unit_group_id,
    u.id AS unit_id,
    u.name AS unit_name,
    u.abbreviation AS unit_abbreviation,
    CAST(u.ratio_numerator AS DECIMAL(65,30)) / CAST(u.ratio_denominator AS DECIMAL(65,30)) AS conversion_factor,
    u.is_base_unit
FROM unit_group_unit ugu
JOIN unit u ON u.id = ugu.unit_id
WHERE ugu.unit_group_id IN (sqlc.slice('unit_group_ids'));

-- GetMaterialOnHandByItem nets each item's available receipts against what has been drawn on them, in the base unit of the item's dimension. Nothing is clamped: a row drawn on for more than it holds nets negative and carries, as the dashboard's ledger did.
-- The receipts are read by item: left to choose, the planner reads every tenant's available receipts by status.
-- name: GetMaterialOnHandByItem :many
SELECT
    ir.item_id,
    CAST(SUM(
        (ir_q.value * (ir_u.ratio_numerator / ir_u.ratio_denominator)) + (ir_u.offset_numerator / ir_u.offset_denominator)
        - COALESCE((
            -- Correlated per row: a grouped derived table cannot take the account and item filters and so aggregates all of inventory_allocation on every call.
            SELECT SUM((aq.value * (au.ratio_numerator / au.ratio_denominator)) + (au.offset_numerator / au.offset_denominator))
            FROM inventory_allocation ia
            JOIN quantity aq ON aq.id = ia.quantity_id
            JOIN unit au ON au.id = aq.unit_id
            WHERE ia.inventory_receipt_id = ir.id
        ), 0)
    ) AS DECIMAL(65,30)) AS remaining_quantity
FROM inventory_receipt ir FORCE INDEX (inventory_receipt_item_id_status_code_idx)
JOIN quantity ir_q ON ir_q.id = ir.quantity_id
JOIN unit ir_u ON ir_u.id = ir_q.unit_id
WHERE (ir.owner_account_id = sqlc.arg('account_id') OR ir.holder_account_id = sqlc.arg('account_id'))
  AND ir.item_id IN (sqlc.slice('item_ids'))
  AND ir.status_code = 'available'
GROUP BY ir.item_id;

-- GetMaterialReservedByItem nets each item's reserved issues against what has been drawn on them, in the base unit of the item's dimension. Nothing is clamped: a row drawn on for more than it holds nets negative and carries, as the dashboard's ledger did.
-- The issues are read by account and item: left to choose, the planner reads every tenant's issues by status.
-- name: GetMaterialReservedByItem :many
SELECT
    ii.item_id,
    CAST(SUM(
        (ii_q.value * (ii_u.ratio_numerator / ii_u.ratio_denominator)) + (ii_u.offset_numerator / ii_u.offset_denominator)
        - COALESCE((
            -- Correlated per row: a grouped derived table cannot take the account and item filters and so aggregates all of inventory_allocation on every call.
            SELECT SUM((aq.value * (au.ratio_numerator / au.ratio_denominator)) + (au.offset_numerator / au.offset_denominator))
            FROM inventory_allocation ia
            JOIN quantity aq ON aq.id = ia.quantity_id
            JOIN unit au ON au.id = aq.unit_id
            WHERE ia.inventory_issue_id = ii.id
        ), 0)
    ) AS DECIMAL(65,30)) AS remaining_quantity
FROM inventory_issue ii FORCE INDEX (inventory_issue_open_paging_idx)
JOIN quantity ii_q ON ii_q.id = ii.quantity_id
JOIN unit ii_u ON ii_u.id = ii_q.unit_id
WHERE ii.account_id = sqlc.arg('account_id')
  AND ii.item_id IN (sqlc.slice('item_ids'))
  AND ii.status_code = 'reserved'
GROUP BY ii.item_id;

-- GetMaterialOpenByItem nets each item's open issues against what has been drawn on them, in the base unit of the item's dimension. Nothing is clamped: a row drawn on for more than it holds nets negative and carries, as the dashboard's ledger did.
-- The issues are read by account and item: left to choose, the planner reads every tenant's issues by status.
-- name: GetMaterialOpenByItem :many
SELECT
    ii.item_id,
    CAST(SUM(
        (ii_q.value * (ii_u.ratio_numerator / ii_u.ratio_denominator)) + (ii_u.offset_numerator / ii_u.offset_denominator)
        - COALESCE((
            -- Correlated per row: a grouped derived table cannot take the account and item filters and so aggregates all of inventory_allocation on every call.
            SELECT SUM((aq.value * (au.ratio_numerator / au.ratio_denominator)) + (au.offset_numerator / au.offset_denominator))
            FROM inventory_allocation ia
            JOIN quantity aq ON aq.id = ia.quantity_id
            JOIN unit au ON au.id = aq.unit_id
            WHERE ia.inventory_issue_id = ii.id
        ), 0)
    ) AS DECIMAL(65,30)) AS remaining_quantity
FROM inventory_issue ii FORCE INDEX (inventory_issue_open_paging_idx)
JOIN quantity ii_q ON ii_q.id = ii.quantity_id
JOIN unit ii_u ON ii_u.id = ii_q.unit_id
WHERE ii.account_id = sqlc.arg('account_id')
  AND ii.item_id IN (sqlc.slice('item_ids'))
  AND ii.status_code = 'open'
GROUP BY ii.item_id;

-- name: GetMaterialSupplierInfo :many
SELECT
    m.item_id,
    sm.supplier_part_number,
    COALESCE(NULLIF(ar.alias, ''), a.name) AS supplier_name
FROM supplier_material sm
JOIN material m ON m.id = sm.material_id
JOIN account a ON a.id = sm.supplier_account_id
LEFT JOIN account_relation ar ON ar.owner_account_id = sm.owner_account_id
    AND ar.counterparty_account_id = sm.supplier_account_id
    AND ar.account_relation_role_code = 'supplier'
WHERE sm.owner_account_id = sqlc.arg('owner_account_id')
  AND sm.supplier_account_id IN (sqlc.slice('supplier_ids'));

-- CountMachinesByDepartment gives the scheduled-time denominator for OEE.
--
-- Availability is machine-hours run over machine-hours scheduled, and downtime is logged per machine, so a department's scheduled time has to scale with how many machines it has or a three-machine room would be measured against one machine's shift.
-- name: CountMachinesByDepartment :many
SELECT
    m.department_id,
    COUNT(*) AS machine_count
FROM machine m
WHERE m.account_id = sqlc.arg('account_id')
AND m.department_id != ''
GROUP BY m.department_id;

-- GetOeeTrendDepartmentDataByWeek is GetOeeDepartmentData bucketed into production weeks, so one read covers a whole trend window instead of one round trip per week.
--
-- The week key is the start of the scan's production week, following the account's configured week_start_day, exactly as SumActualsByWeek buckets it: a trend that bucketed on a different day would disagree with schedule attainment about which week a batch belongs to.
-- name: GetOeeTrendDepartmentDataByWeek :many
SELECT
    DATE(DATE_SUB(b.scanned_at, INTERVAL ((DAYOFWEEK(b.scanned_at) + 6 - CAST(sqlc.arg('week_start_day') AS SIGNED)) % 7) DAY)) AS week_start_date,
    COALESCE(d.id, 'unassigned') AS department_id,
    COALESCE(d.name, 'Unassigned') AS department_name,
    CAST(COALESCE(SUM(COALESCE(qf.value * (u_qf.ratio_numerator / u_qf.ratio_denominator), 0)), 0) AS DECIMAL(65,30)) AS good_units,
    CAST(COALESCE(SUM(COALESCE(qw.value * (u_qw.ratio_numerator / u_qw.ratio_denominator), 0)), 0) AS DECIMAL(65,30)) AS waste_units,
    CAST(COALESCE(SUM(COALESCE(qs.value * (u_qs.ratio_numerator / u_qs.ratio_denominator), 0)), 0) AS DECIMAL(65,30)) AS seconds_units,
    CAST(COALESCE(SUM(
        (
            COALESCE(qf.value * (u_qf.ratio_numerator / u_qf.ratio_denominator), 0)
            + COALESCE(qw.value * (u_qw.ratio_numerator / u_qw.ratio_denominator), 0)
            + COALESCE(qs.value * (u_qs.ratio_numerator / u_qs.ratio_denominator), 0)
        ) * COALESCE(
            -- Labor time to seconds per BASE quantity unit, via both of the rate's unit ratios, not a
            -- fixed abbreviation table. The numerator unit's ratio is its size in the time dimension's
            -- base (the hour), so value * (ratio) * 3600 gives seconds; handles day and any
            -- account-defined time unit. The denominator unit's ratio is its size in the quantity base,
            -- and the rate is DIVIDED by it because the quantities above are already normalized to that
            -- base: a step rated 554 sec/pr is 277 sec per each, and skipping this division counts every
            -- pair-rated step twice. Stays in step with the solver's SecondsPerUnitFromLaborTime, which
            -- reads the same four columns. A missing numerator unit leaves the product NULL, so the
            -- COALESCE books no standard time rather than guessing a scale; a missing denominator unit
            -- divides by one, leaving a rate already quoted per base unit alone.
            labor_time.value * (labor_time_unit.ratio_numerator / labor_time_unit.ratio_denominator) * 3600
                / COALESCE(NULLIF(labor_time_qty_unit.ratio_numerator / labor_time_qty_unit.ratio_denominator, 0), 1),
            0
        )
    ), 0) AS DECIMAL(65,30)) AS standard_seconds_earned
-- FORCE INDEX pins the (account_id, scanned_at) range scan; see GetOeeDepartmentData.
FROM batch b FORCE INDEX (batch_account_id_scanned_at_idx)
LEFT JOIN quantity qf ON qf.id = b.quantity_id
LEFT JOIN unit u_qf ON u_qf.id = qf.unit_id
LEFT JOIN quantity qw ON qw.id = b.waste_quantity_id
LEFT JOIN unit u_qw ON u_qw.id = qw.unit_id
LEFT JOIN quantity qs ON qs.id = b.seconds_quantity_id
LEFT JOIN unit u_qs ON u_qs.id = qs.unit_id
LEFT JOIN scanning_station ss ON ss.id = b.scanning_station_id
LEFT JOIN department d ON d.id = ss.department_id
LEFT JOIN production_step ps ON ps.id = b.production_step_id
LEFT JOIN rate labor_time ON labor_time.id = ps.labor_time_id
LEFT JOIN unit labor_time_unit ON labor_time_unit.id = labor_time.numerator_unit_id
LEFT JOIN unit labor_time_qty_unit ON labor_time_qty_unit.id = labor_time.denominator_unit_id
WHERE b.account_id = sqlc.arg('owner_account_id')
  AND b.scanned_at >= sqlc.arg('start_date')
  AND b.scanned_at <= sqlc.arg('end_date')
GROUP BY week_start_date, d.id, d.name;

-- GetOeeTrendDepartmentDataByWeekForMachines is GetOeeTrendDepartmentDataByWeek restricted to production on a given set of machines, mirroring GetOeeDepartmentDataForMachines so a trend point measures the same machines as the table beside it. The SELECT list must stay identical to GetOeeTrendDepartmentDataByWeek.
-- name: GetOeeTrendDepartmentDataByWeekForMachines :many
SELECT
    DATE(DATE_SUB(b.scanned_at, INTERVAL ((DAYOFWEEK(b.scanned_at) + 6 - CAST(sqlc.arg('week_start_day') AS SIGNED)) % 7) DAY)) AS week_start_date,
    COALESCE(d.id, 'unassigned') AS department_id,
    COALESCE(d.name, 'Unassigned') AS department_name,
    CAST(COALESCE(SUM(COALESCE(qf.value * (u_qf.ratio_numerator / u_qf.ratio_denominator), 0)), 0) AS DECIMAL(65,30)) AS good_units,
    CAST(COALESCE(SUM(COALESCE(qw.value * (u_qw.ratio_numerator / u_qw.ratio_denominator), 0)), 0) AS DECIMAL(65,30)) AS waste_units,
    CAST(COALESCE(SUM(COALESCE(qs.value * (u_qs.ratio_numerator / u_qs.ratio_denominator), 0)), 0) AS DECIMAL(65,30)) AS seconds_units,
    CAST(COALESCE(SUM(
        (
            COALESCE(qf.value * (u_qf.ratio_numerator / u_qf.ratio_denominator), 0)
            + COALESCE(qw.value * (u_qw.ratio_numerator / u_qw.ratio_denominator), 0)
            + COALESCE(qs.value * (u_qs.ratio_numerator / u_qs.ratio_denominator), 0)
        ) * COALESCE(
            -- Labor time to seconds per BASE quantity unit, via both of the rate's unit ratios, not a
            -- fixed abbreviation table. The numerator unit's ratio is its size in the time dimension's
            -- base (the hour), so value * (ratio) * 3600 gives seconds; handles day and any
            -- account-defined time unit. The denominator unit's ratio is its size in the quantity base,
            -- and the rate is DIVIDED by it because the quantities above are already normalized to that
            -- base: a step rated 554 sec/pr is 277 sec per each, and skipping this division counts every
            -- pair-rated step twice. Stays in step with the solver's SecondsPerUnitFromLaborTime, which
            -- reads the same four columns. A missing numerator unit leaves the product NULL, so the
            -- COALESCE books no standard time rather than guessing a scale; a missing denominator unit
            -- divides by one, leaving a rate already quoted per base unit alone.
            labor_time.value * (labor_time_unit.ratio_numerator / labor_time_unit.ratio_denominator) * 3600
                / COALESCE(NULLIF(labor_time_qty_unit.ratio_numerator / labor_time_qty_unit.ratio_denominator, 0), 1),
            0
        )
    ), 0) AS DECIMAL(65,30)) AS standard_seconds_earned
-- FORCE INDEX pins the (account_id, scanned_at) range scan; see GetOeeDepartmentData.
FROM batch b FORCE INDEX (batch_account_id_scanned_at_idx)
LEFT JOIN quantity qf ON qf.id = b.quantity_id
LEFT JOIN unit u_qf ON u_qf.id = qf.unit_id
LEFT JOIN quantity qw ON qw.id = b.waste_quantity_id
LEFT JOIN unit u_qw ON u_qw.id = qw.unit_id
LEFT JOIN quantity qs ON qs.id = b.seconds_quantity_id
LEFT JOIN unit u_qs ON u_qs.id = qs.unit_id
LEFT JOIN scanning_station ss ON ss.id = b.scanning_station_id
LEFT JOIN department d ON d.id = ss.department_id
LEFT JOIN production_step ps ON ps.id = b.production_step_id
LEFT JOIN rate labor_time ON labor_time.id = ps.labor_time_id
LEFT JOIN unit labor_time_unit ON labor_time_unit.id = labor_time.numerator_unit_id
LEFT JOIN unit labor_time_qty_unit ON labor_time_qty_unit.id = labor_time.denominator_unit_id
WHERE b.account_id = sqlc.arg('owner_account_id')
  AND b.scanned_at >= sqlc.arg('start_date')
  AND b.scanned_at <= sqlc.arg('end_date')
  AND EXISTS (
      SELECT 1 FROM _batches_machines bm
      WHERE bm.A = b.id AND bm.B IN (sqlc.slice('machine_ids'))
  )
GROUP BY week_start_date, d.id, d.name;

-- GetOeeDowntimeIntervals lists logged downtime per department and reason as raw intervals, unclipped (open events coalesce to now). Both OEE reads use it: the per-department table and the trend.
--
-- Nothing is totalled here because a logged span is not the same as lost capacity, and neither clip can be expressed in one SQL sum. An event that crosses a week boundary belongs partly to each week, and an event that runs overnight belongs to the plant's shift window only for the part the plant was open. Both are exact interval arithmetic in Go (see oeeShiftWindow) and need no calendar table.
-- name: GetOeeDowntimeIntervals :many
SELECT
    COALESCE(e.department_id, 'unassigned') AS department_id,
    e.reason_code,
    r.oee_bucket,
    e.started_at,
    COALESCE(e.ended_at, NOW(3)) AS ended_at
FROM machine_downtime_event e FORCE INDEX (machine_downtime_account_started_idx, machine_downtime_account_ended_started_idx)
JOIN machine_downtime_reason r ON r.code = e.reason_code
WHERE e.account_id = sqlc.arg('account_id')
  -- Overlap test rather than containment: an event that started before the window and is still running must still contribute its in-window seconds.
  -- COALESCE(ended_at, NOW(3)) >= start_date, spelled so each side can be read from a key: the forced keys read either the events started before the window ends or those ending (or open) after it starts, whichever is fewer.
  AND e.started_at <= sqlc.arg('end_date')
  AND (e.ended_at >= sqlc.arg('start_date') OR (e.ended_at IS NULL AND NOW(3) >= sqlc.arg('start_date')))
ORDER BY e.started_at;
