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

-- name: GetQuarterlyOrderTotals :many
SELECT
    YEAR(so.issued_at) AS order_year,
    CAST(SUM(
        CASE WHEN QUARTER(so.issued_at) = 1 THEN
            (
                (CAST(q.value AS DECIMAL(65,30)) * (CAST(u_ord.ratio_numerator AS DECIMAL(65,30)) / CAST(u_ord.ratio_denominator AS DECIMAL(65,30))))
                + (CAST(u_ord.offset_numerator AS DECIMAL(65,30)) / CAST(u_ord.offset_denominator AS DECIMAL(65,30)))
            )
            *
            (
                (
                    (CAST(r_price.value AS DECIMAL(65,30)) * (CAST(u_price_num.ratio_numerator AS DECIMAL(65,30)) / CAST(u_price_num.ratio_denominator AS DECIMAL(65,30))))
                    + (CAST(u_price_num.offset_numerator AS DECIMAL(65,30)) / CAST(u_price_num.offset_denominator AS DECIMAL(65,30)))
                )
                / NULLIF(((CAST(u_price_den.ratio_numerator AS DECIMAL(65,30)) / CAST(u_price_den.ratio_denominator AS DECIMAL(65,30))) + (CAST(u_price_den.offset_numerator AS DECIMAL(65,30)) / CAST(u_price_den.offset_denominator AS DECIMAL(65,30)))), 0)
            )
        ELSE 0 END
    ) AS DECIMAL(65,30)) AS q1,
    CAST(SUM(
        CASE WHEN QUARTER(so.issued_at) = 2 THEN
            (
                (CAST(q.value AS DECIMAL(65,30)) * (CAST(u_ord.ratio_numerator AS DECIMAL(65,30)) / CAST(u_ord.ratio_denominator AS DECIMAL(65,30))))
                + (CAST(u_ord.offset_numerator AS DECIMAL(65,30)) / CAST(u_ord.offset_denominator AS DECIMAL(65,30)))
            )
            *
            (
                (
                    (CAST(r_price.value AS DECIMAL(65,30)) * (CAST(u_price_num.ratio_numerator AS DECIMAL(65,30)) / CAST(u_price_num.ratio_denominator AS DECIMAL(65,30))))
                    + (CAST(u_price_num.offset_numerator AS DECIMAL(65,30)) / CAST(u_price_num.offset_denominator AS DECIMAL(65,30)))
                )
                / NULLIF(((CAST(u_price_den.ratio_numerator AS DECIMAL(65,30)) / CAST(u_price_den.ratio_denominator AS DECIMAL(65,30))) + (CAST(u_price_den.offset_numerator AS DECIMAL(65,30)) / CAST(u_price_den.offset_denominator AS DECIMAL(65,30)))), 0)
            )
        ELSE 0 END
    ) AS DECIMAL(65,30)) AS q2,
    CAST(SUM(
        CASE WHEN QUARTER(so.issued_at) = 3 THEN
            (
                (CAST(q.value AS DECIMAL(65,30)) * (CAST(u_ord.ratio_numerator AS DECIMAL(65,30)) / CAST(u_ord.ratio_denominator AS DECIMAL(65,30))))
                + (CAST(u_ord.offset_numerator AS DECIMAL(65,30)) / CAST(u_ord.offset_denominator AS DECIMAL(65,30)))
            )
            *
            (
                (
                    (CAST(r_price.value AS DECIMAL(65,30)) * (CAST(u_price_num.ratio_numerator AS DECIMAL(65,30)) / CAST(u_price_num.ratio_denominator AS DECIMAL(65,30))))
                    + (CAST(u_price_num.offset_numerator AS DECIMAL(65,30)) / CAST(u_price_num.offset_denominator AS DECIMAL(65,30)))
                )
                / NULLIF(((CAST(u_price_den.ratio_numerator AS DECIMAL(65,30)) / CAST(u_price_den.ratio_denominator AS DECIMAL(65,30))) + (CAST(u_price_den.offset_numerator AS DECIMAL(65,30)) / CAST(u_price_den.offset_denominator AS DECIMAL(65,30)))), 0)
            )
        ELSE 0 END
    ) AS DECIMAL(65,30)) AS q3,
    CAST(SUM(
        CASE WHEN QUARTER(so.issued_at) = 4 THEN
            (
                (CAST(q.value AS DECIMAL(65,30)) * (CAST(u_ord.ratio_numerator AS DECIMAL(65,30)) / CAST(u_ord.ratio_denominator AS DECIMAL(65,30))))
                + (CAST(u_ord.offset_numerator AS DECIMAL(65,30)) / CAST(u_ord.offset_denominator AS DECIMAL(65,30)))
            )
            *
            (
                (
                    (CAST(r_price.value AS DECIMAL(65,30)) * (CAST(u_price_num.ratio_numerator AS DECIMAL(65,30)) / CAST(u_price_num.ratio_denominator AS DECIMAL(65,30))))
                    + (CAST(u_price_num.offset_numerator AS DECIMAL(65,30)) / CAST(u_price_num.offset_denominator AS DECIMAL(65,30)))
                )
                / NULLIF(((CAST(u_price_den.ratio_numerator AS DECIMAL(65,30)) / CAST(u_price_den.ratio_denominator AS DECIMAL(65,30))) + (CAST(u_price_den.offset_numerator AS DECIMAL(65,30)) / CAST(u_price_den.offset_denominator AS DECIMAL(65,30)))), 0)
            )
        ELSE 0 END
    ) AS DECIMAL(65,30)) AS q4,
    CAST(SUM(
        (
            (CAST(q.value AS DECIMAL(65,30)) * (CAST(u_ord.ratio_numerator AS DECIMAL(65,30)) / CAST(u_ord.ratio_denominator AS DECIMAL(65,30))))
            + (CAST(u_ord.offset_numerator AS DECIMAL(65,30)) / CAST(u_ord.offset_denominator AS DECIMAL(65,30)))
        )
        *
        (
            (
                (CAST(r_price.value AS DECIMAL(65,30)) * (CAST(u_price_num.ratio_numerator AS DECIMAL(65,30)) / CAST(u_price_num.ratio_denominator AS DECIMAL(65,30))))
                + (CAST(u_price_num.offset_numerator AS DECIMAL(65,30)) / CAST(u_price_num.offset_denominator AS DECIMAL(65,30)))
            )
            / NULLIF(((CAST(u_price_den.ratio_numerator AS DECIMAL(65,30)) / CAST(u_price_den.ratio_denominator AS DECIMAL(65,30))) + (CAST(u_price_den.offset_numerator AS DECIMAL(65,30)) / CAST(u_price_den.offset_denominator AS DECIMAL(65,30)))), 0)
        )
    ) AS DECIMAL(65,30)) AS total
FROM sales_order_line sol
JOIN sales_order so ON so.id = sol.sales_order_id
JOIN product fg ON fg.id = sol.product_id
JOIN quantity q ON q.id = sol.quantity_id
JOIN unit u_ord ON u_ord.id = q.unit_id
LEFT JOIN rate r_price ON r_price.id = sol.unit_price_id
LEFT JOIN unit u_price_num ON u_price_num.id = r_price.numerator_unit_id
LEFT JOIN unit u_price_den ON u_price_den.id = r_price.denominator_unit_id
LEFT JOIN account_relation ar ON ar.owner_account_id = so.owner_account_id
    AND ar.counterparty_account_id = so.buyer_account_id
    AND ar.account_relation_role_code = 'customer'
WHERE so.owner_account_id = sqlc.arg('owner_account_id')
  AND fg.product_type_code = 'sale'
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
  AND (sqlc.arg('include_sales_rep_filter') = false OR so.sales_rep_id IN (sqlc.slice('sales_rep_ids')))
  AND (sqlc.arg('include_product_line_filter') = false OR fg.product_line_id IN (sqlc.slice('product_line_ids')))
  AND (sqlc.arg('include_item_filter') = false OR fg.item_id IN (sqlc.slice('item_ids')))
  AND (sqlc.arg('include_customer_group_filter') = false OR ar.account_group_id IN (sqlc.slice('customer_group_ids')))
GROUP BY order_year ORDER BY order_year ASC;

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
JOIN invoice inv ON inv.id = il.invoice_id
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
ORDER BY inv.created_at ASC;

-- name: GetOrderEntries :many
SELECT
    sol.id AS id,
    so.issued_at AS issued_at,
    so.completed_at AS completed_at,
    so.first_ship_at AS first_ship_at,
    so.promised_at AS promised_at,
    so.customer_po_number AS customer_po,
    so.number AS order_number,
    so.id AS order_id,
    so.sales_rep_id AS sales_rep_id,
    bu.username AS sales_rep_username,
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
    CAST(
        (
            (
                (q_ord.value * (u_ord.ratio_numerator / u_ord.ratio_denominator))
                + (u_ord.offset_numerator / u_ord.offset_denominator)
            )
            - (bu_unit.offset_numerator / bu_unit.offset_denominator)
        )
        / NULLIF((bu_unit.ratio_numerator / bu_unit.ratio_denominator), 0)
        AS DECIMAL(65,30)
    ) AS quantity_ordered,
    CAST(
        (
            COALESCE(inv.qty_inv_norm, 0)
            - (bu_unit.offset_numerator / bu_unit.offset_denominator)
        )
        / NULLIF((bu_unit.ratio_numerator / bu_unit.ratio_denominator), 0)
        AS DECIMAL(65,30)
    ) AS quantity_invoiced,
    CAST(
        (
            (
                (
                    (q_ord.value * (u_ord.ratio_numerator / u_ord.ratio_denominator))
                    + (u_ord.offset_numerator / u_ord.offset_denominator)
                )
                - (bu_unit.offset_numerator / bu_unit.offset_denominator)
            )
            / NULLIF((bu_unit.ratio_numerator / bu_unit.ratio_denominator), 0)
            -
            (
                COALESCE(inv.qty_inv_norm, 0)
                - (bu_unit.offset_numerator / bu_unit.offset_denominator)
            )
            / NULLIF((bu_unit.ratio_numerator / bu_unit.ratio_denominator), 0)
        )
        AS DECIMAL(65,30)
    ) AS quantity_back_ordered,
    bu_unit.abbreviation AS unit,
    CAST(
        (
            (
                (q_ord.value * (u_ord.ratio_numerator / u_ord.ratio_denominator))
                + (u_ord.offset_numerator / u_ord.offset_denominator)
            )
            *
            (
                (
                    (r_price.value * (u_price_num.ratio_numerator / u_price_num.ratio_denominator))
                    + (u_price_num.offset_numerator / u_price_num.offset_denominator)
                )
                / NULLIF(((u_price_den.ratio_numerator / u_price_den.ratio_denominator) + (u_price_den.offset_numerator / u_price_den.offset_denominator)), 0)
            )
        ) AS DECIMAL(65,30)
    ) AS total_ordered,
    CAST(
        COALESCE(inv.qty_inv_norm, 0)
        *
        (
            (
                (r_price.value * (u_price_num.ratio_numerator / u_price_num.ratio_denominator))
                + (u_price_num.offset_numerator / u_price_num.offset_denominator)
            )
            / NULLIF(((u_price_den.ratio_numerator / u_price_den.ratio_denominator) + (u_price_den.offset_numerator / u_price_den.offset_denominator)), 0)
        )
        AS DECIMAL(65,30)
    ) AS total_invoiced,
    CAST(
        (
            (
                (q_ord.value * (u_ord.ratio_numerator / u_ord.ratio_denominator))
                + (u_ord.offset_numerator / u_ord.offset_denominator)
            )
            - COALESCE(inv.qty_inv_norm, 0)
        )
        *
        (
            (
                (r_price.value * (u_price_num.ratio_numerator / u_price_num.ratio_denominator))
                + (u_price_num.offset_numerator / u_price_num.offset_denominator)
            )
            / NULLIF(((u_price_den.ratio_numerator / u_price_den.ratio_denominator) + (u_price_den.offset_numerator / u_price_den.offset_denominator)), 0)
        )
        AS DECIMAL(65,30)
    ) AS total_back_ordered,
    CAST(
        COALESCE(inv.qty_inv_norm, 0)
        *
        (
            (
                (COALESCE(r_cost.value, 0) * (COALESCE(u_cost_num.ratio_numerator, 1) / COALESCE(u_cost_num.ratio_denominator, 1)))
                + (COALESCE(u_cost_num.offset_numerator, 0) / COALESCE(u_cost_num.offset_denominator, 1))
            )
            / NULLIF(((COALESCE(u_cost_den.ratio_numerator, 1) / COALESCE(u_cost_den.ratio_denominator, 1)) + (COALESCE(u_cost_den.offset_numerator, 0) / COALESCE(u_cost_den.offset_denominator, 1))), 0)
        )
        AS DECIMAL(65,30)
    ) AS total_cost,
    CAST(
        (
            COALESCE(inv.qty_inv_norm, 0)
            *
            (
                (
                    (r_price.value * (u_price_num.ratio_numerator / u_price_num.ratio_denominator))
                    + (u_price_num.offset_numerator / u_price_num.offset_denominator)
                )
                / NULLIF(((u_price_den.ratio_numerator / u_price_den.ratio_denominator) + (u_price_den.offset_numerator / u_price_den.offset_denominator)), 0)
            )
        )
        -
        (
            COALESCE(inv.qty_inv_norm, 0)
            *
            (
                (
                    (COALESCE(r_cost.value, 0) * (COALESCE(u_cost_num.ratio_numerator, 1) / COALESCE(u_cost_num.ratio_denominator, 1)))
                    + (COALESCE(u_cost_num.offset_numerator, 0) / COALESCE(u_cost_num.offset_denominator, 1))
                )
                / NULLIF(((COALESCE(u_cost_den.ratio_numerator, 1) / COALESCE(u_cost_den.ratio_denominator, 1)) + (COALESCE(u_cost_den.offset_numerator, 0) / COALESCE(u_cost_den.offset_denominator, 1))), 0)
            )
        )
        AS DECIMAL(65,30)
    ) AS total_profit,
    CAST(
        (
            COALESCE(inv.qty_inv_norm, 0)
            *
            (
                (
                    (r_price.value * (u_price_num.ratio_numerator / u_price_num.ratio_denominator))
                    + (u_price_num.offset_numerator / u_price_num.offset_denominator)
                )
                / NULLIF(((u_price_den.ratio_numerator / u_price_den.ratio_denominator) + (u_price_den.offset_numerator / u_price_den.offset_denominator)), 0)
            )
        )
        / NULLIF(
            (
                COALESCE(inv.qty_inv_norm, 0)
                - (bu_unit.offset_numerator / bu_unit.offset_denominator)
            )
            / NULLIF((bu_unit.ratio_numerator / bu_unit.ratio_denominator), 0),
            0
        )
        AS DECIMAL(65,30)
    ) AS unit_price,
    CAST(
        (
            COALESCE(inv.qty_inv_norm, 0)
            *
            (
                (
                    (COALESCE(r_cost.value, 0) * (COALESCE(u_cost_num.ratio_numerator, 1) / COALESCE(u_cost_num.ratio_denominator, 1)))
                    + (COALESCE(u_cost_num.offset_numerator, 0) / COALESCE(u_cost_num.offset_denominator, 1))
                )
                / NULLIF(((COALESCE(u_cost_den.ratio_numerator, 1) / COALESCE(u_cost_den.ratio_denominator, 1)) + (COALESCE(u_cost_den.offset_numerator, 0) / COALESCE(u_cost_den.offset_denominator, 1))), 0)
            )
        )
        / NULLIF(
            (
                COALESCE(inv.qty_inv_norm, 0)
                - (bu_unit.offset_numerator / bu_unit.offset_denominator)
            )
            / NULLIF((bu_unit.ratio_numerator / bu_unit.ratio_denominator), 0),
            0
        )
        AS DECIMAL(65,30)
    ) AS unit_cost,
    CAST(
        (
            (
                COALESCE(inv.qty_inv_norm, 0)
                *
                (
                    (
                        (r_price.value * (u_price_num.ratio_numerator / u_price_num.ratio_denominator))
                        + (u_price_num.offset_numerator / u_price_num.offset_denominator)
                    )
                    / NULLIF(((u_price_den.ratio_numerator / u_price_den.ratio_denominator) + (u_price_den.offset_numerator / u_price_den.offset_denominator)), 0)
                )
            )
            -
            (
                COALESCE(inv.qty_inv_norm, 0)
                *
                (
                    (
                        (COALESCE(r_cost.value, 0) * (COALESCE(u_cost_num.ratio_numerator, 1) / COALESCE(u_cost_num.ratio_denominator, 1)))
                        + (COALESCE(u_cost_num.offset_numerator, 0) / COALESCE(u_cost_num.offset_denominator, 1))
                    )
                    / NULLIF(((COALESCE(u_cost_den.ratio_numerator, 1) / COALESCE(u_cost_den.ratio_denominator, 1)) + (COALESCE(u_cost_den.offset_numerator, 0) / COALESCE(u_cost_den.offset_denominator, 1))), 0)
                )
            )
        )
        / NULLIF(
            (
                COALESCE(inv.qty_inv_norm, 0)
                - (bu_unit.offset_numerator / bu_unit.offset_denominator)
            )
            / NULLIF((bu_unit.ratio_numerator / bu_unit.ratio_denominator), 0),
            0
        )
        AS DECIMAL(65,30)
    ) AS unit_profit,
    shipping_geolocation.state AS ship_to_state,
    shipping_geolocation.locality AS ship_to_city,
    shipping_geolocation.postal_code AS ship_to_zipcode,
    shipping_geolocation.country AS ship_to_country,
    od.code AS order_discount_code
FROM sales_order_line sol
LEFT JOIN product fg ON sol.product_id = fg.id
LEFT JOIN item pb ON fg.item_id = pb.id
LEFT JOIN item_category ic ON pb.item_category_id = ic.id
LEFT JOIN unit_group ug ON ug.id = ic.unit_group_id
LEFT JOIN unit bu_unit ON bu_unit.id = ug.base_unit_id
LEFT JOIN product_line pl ON fg.product_line_id = pl.id
LEFT JOIN sales_order so ON sol.sales_order_id = so.id
LEFT JOIN account buyer ON so.buyer_account_id = buyer.id
LEFT JOIN account_relation ar ON ar.owner_account_id = so.owner_account_id AND ar.counterparty_account_id = so.buyer_account_id AND ar.account_relation_role_code = 'customer'
LEFT JOIN account_relation parent_ar ON parent_ar.id = ar.parent_account_relation_id AND parent_ar.owner_account_id = ar.owner_account_id
LEFT JOIN account_group ag ON ar.account_group_id = ag.id
LEFT JOIN address shipping_address ON so.shipping_address_id = shipping_address.id
LEFT JOIN geolocation shipping_geolocation ON shipping_address.geolocation_id = shipping_geolocation.id
LEFT JOIN account_user ou ON so.sales_rep_id = ou.id
LEFT JOIN `user` bu ON ou.user_id = bu.id
LEFT JOIN order_discount od ON so.order_discount_id = od.id
-- ordered quantity and unit
LEFT JOIN quantity q_ord ON q_ord.id = sol.quantity_id
LEFT JOIN unit u_ord ON u_ord.id = q_ord.unit_id
-- aggregated invoice quantities (normalized to base)
LEFT JOIN (
    SELECT
        il.sales_order_line_id AS line_id,
        SUM(
            (q_in.value * (u_in.ratio_numerator / u_in.ratio_denominator))
            + (u_in.offset_numerator / u_in.offset_denominator)
        ) AS qty_inv_norm
    FROM invoice_line il
    JOIN quantity q_in ON q_in.id = il.quantity_id
    JOIN unit u_in ON u_in.id = q_in.unit_id
    GROUP BY il.sales_order_line_id
) inv ON inv.line_id = sol.id
-- prices and units
LEFT JOIN rate r_price ON r_price.id = sol.unit_price_id
LEFT JOIN unit u_price_num ON u_price_num.id = r_price.numerator_unit_id
LEFT JOIN unit u_price_den ON u_price_den.id = r_price.denominator_unit_id
-- costs and units
LEFT JOIN rate r_cost ON r_cost.id = sol.unit_cost_id
LEFT JOIN unit u_cost_num ON u_cost_num.id = r_cost.numerator_unit_id
LEFT JOIN unit u_cost_den ON u_cost_den.id = r_cost.denominator_unit_id
WHERE so.owner_account_id = sqlc.arg('owner_account_id')
  AND so.sales_order_status_code = 'issued'
  AND fg.product_type_code = 'sale'
  AND (sqlc.arg('include_sales_rep_filter') = false OR so.sales_rep_id IN (sqlc.slice('sales_rep_ids')))
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
  AND (sqlc.arg('include_customer_group_filter') = false OR ar.account_group_id IN (sqlc.slice('customer_group_ids')))
  AND (sqlc.arg('include_product_line_filter') = false OR fg.product_line_id IN (sqlc.slice('product_line_ids')))
ORDER BY so.issued_at ASC;

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
SELECT
    CAST(COALESCE(SUM(CASE WHEN COALESCE(pt.prod_total, 0) > 0 THEN COALESCE(qf.value, 0) * (lt.value / pt.prod_total) ELSE 0 END), 0) AS DECIMAL(65,30)) AS labor_quantity,
    CAST(COALESCE(SUM(CASE WHEN COALESCE(pt.prod_total, 0) > 0 THEN COALESCE(qw.value, 0) * (lt.value / pt.prod_total) ELSE 0 END), 0) AS DECIMAL(65,30)) AS labor_waste,
    CAST(COALESCE(SUM(CASE WHEN COALESCE(pt.prod_total, 0) > 0 THEN COALESCE(qs.value, 0) * (lt.value / pt.prod_total) ELSE 0 END), 0) AS DECIMAL(65,30)) AS labor_seconds
FROM batch b
LEFT JOIN quantity qf ON qf.id = b.quantity_id
LEFT JOIN quantity qw ON qw.id = b.waste_quantity_id
LEFT JOIN quantity qs ON qs.id = b.seconds_quantity_id
LEFT JOIN production_step ps ON ps.id = b.production_step_id
LEFT JOIN rate lt ON lt.id = ps.labor_time_id
LEFT JOIN (
    SELECT pr.production_step_id, SUM(qp.value) AS prod_total
    FROM production pr
    JOIN quantity qp ON qp.id = pr.quantity_id
    WHERE pr.production_step_id IN (
        SELECT DISTINCT b2.production_step_id FROM batch b2
        WHERE b2.account_id = sqlc.arg('owner_account_id')
          AND b2.scanned_at >= sqlc.arg('start_date')
          AND b2.scanned_at <= sqlc.arg('end_date')
    )
    GROUP BY pr.production_step_id
) pt ON pt.production_step_id = b.production_step_id
WHERE b.account_id = sqlc.arg('owner_account_id')
  AND b.scanned_at >= sqlc.arg('start_date')
  AND b.scanned_at <= sqlc.arg('end_date');

-- name: GetManufacturingBatchBatchMetrics :one
-- Combined batch-table query for production, quality, and labor efficiency.
-- Uses scanned_at for date filtering to match legacy dashboard behavior.
WITH active_steps AS (
    SELECT pr.production_step_id, SUM(CAST(qp.value AS DECIMAL(65,30))) AS prod_total
    FROM production pr
    JOIN quantity qp ON qp.id = pr.quantity_id
    WHERE pr.production_step_id IN (
        SELECT DISTINCT b2.production_step_id FROM batch b2
        WHERE b2.account_id = sqlc.arg('owner_account_id')
          AND b2.scanned_at >= sqlc.arg('start_date')
          AND b2.scanned_at <= sqlc.arg('end_date')
    )
    GROUP BY pr.production_step_id
)
SELECT
    CAST(COALESCE(SUM(CAST(qf.value AS DECIMAL(65,30))), 0) AS DECIMAL(65,30)) AS total_quantity,
    CAST(COALESCE(SUM(CAST(COALESCE(qw.value, 0) AS DECIMAL(65,30))), 0) AS DECIMAL(65,30)) AS total_waste,
    CAST(COALESCE(SUM(CAST(COALESCE(qs.value, 0) AS DECIMAL(65,30))), 0) AS DECIMAL(65,30)) AS total_seconds,
    CAST(COALESCE(SUM(CASE WHEN COALESCE(pt.prod_total, 0) > 0 THEN COALESCE(CAST(qf.value AS DECIMAL(65,30)), 0) * (CAST(lt.value AS DECIMAL(65,30)) / pt.prod_total) ELSE 0 END), 0) AS DECIMAL(65,30)) AS labor_quantity,
    CAST(COALESCE(SUM(CASE WHEN COALESCE(pt.prod_total, 0) > 0 THEN COALESCE(CAST(COALESCE(qw.value, 0) AS DECIMAL(65,30)), 0) * (CAST(lt.value AS DECIMAL(65,30)) / pt.prod_total) ELSE 0 END), 0) AS DECIMAL(65,30)) AS labor_waste,
    CAST(COALESCE(SUM(CASE WHEN COALESCE(pt.prod_total, 0) > 0 THEN COALESCE(CAST(COALESCE(qs.value, 0) AS DECIMAL(65,30)), 0) * (CAST(lt.value AS DECIMAL(65,30)) / pt.prod_total) ELSE 0 END), 0) AS DECIMAL(65,30)) AS labor_seconds
FROM batch b
LEFT JOIN quantity qf ON qf.id = b.quantity_id
LEFT JOIN quantity qw ON qw.id = b.waste_quantity_id
LEFT JOIN quantity qs ON qs.id = b.seconds_quantity_id
LEFT JOIN production_step ps ON ps.id = b.production_step_id
LEFT JOIN rate lt ON lt.id = ps.labor_time_id
LEFT JOIN active_steps pt ON pt.production_step_id = b.production_step_id
WHERE b.account_id = sqlc.arg('owner_account_id')
  AND b.scanned_at >= sqlc.arg('start_date')
  AND b.scanned_at <= sqlc.arg('end_date');

-- name: GetManufacturingBatchInvoiceMetrics :one
-- Combined invoice-table query for costs per unit and margin.
-- Uses unit conversion logic to normalize quantities, costs, and prices.
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

-- name: GetInventoryReceiptEntries :many
SELECT
    it.id AS item_id,
    it.sku AS product_sku,
    it.description AS product_description,
    sl.id AS storage_location_id,
    sl.name AS storage_location_name,
    l.id AS lot_id,
    l.lot_number AS lot_number,
    ir.owner_account_id AS owner_account_id,
    oa.name AS owner_account_name,
    ir.holder_account_id AS holder_account_id,
    ha.name AS holder_account_name,
    SUM(GREATEST(CAST(ir_q.value AS DECIMAL(65,30)) - COALESCE(alloc_sum.total_allocated, 0), 0)) AS remaining_quantity,
    CASE
        WHEN SUM(GREATEST(CAST(ir_q.value AS DECIMAL(65,30)) - COALESCE(alloc_sum.total_allocated, 0), 0)) > 0
        THEN SUM(GREATEST(CAST(ir_q.value AS DECIMAL(65,30)) - COALESCE(alloc_sum.total_allocated, 0), 0) * COALESCE(CAST(r_cost.value AS DECIMAL(65,30)), 0))
             / SUM(GREATEST(CAST(ir_q.value AS DECIMAL(65,30)) - COALESCE(alloc_sum.total_allocated, 0), 0))
        ELSE 0
    END AS weighted_average_unit_cost,
    SUM(GREATEST(CAST(ir_q.value AS DECIMAL(65,30)) - COALESCE(alloc_sum.total_allocated, 0), 0) * COALESCE(CAST(r_cost.value AS DECIMAL(65,30)), 0)) AS inventory_value,
    MIN(ir.received_at) AS oldest_receipt_at,
    MAX(ir.received_at) AS newest_receipt_at,
    ir_u.abbreviation AS unit,
    ir_u.name AS unit_name,
    COALESCE(r_cost_nu.abbreviation, '') AS cost_numerator_unit_abbreviation,
    COALESCE(r_cost_nu.name, '') AS cost_numerator_unit_name,
    COALESCE(r_cost_du.abbreviation, '') AS cost_denominator_unit_abbreviation,
    COALESCE(r_cost_du.name, '') AS cost_denominator_unit_name
FROM inventory_receipt ir
JOIN item it ON it.id = ir.item_id
JOIN quantity ir_q ON ir_q.id = ir.quantity_id
JOIN unit ir_u ON ir_u.id = ir_q.unit_id
LEFT JOIN rate r_cost ON r_cost.id = ir.unit_cost_id
LEFT JOIN unit r_cost_nu ON r_cost_nu.id = r_cost.numerator_unit_id
LEFT JOIN unit r_cost_du ON r_cost_du.id = r_cost.denominator_unit_id
LEFT JOIN storage_location sl ON sl.id = ir.storage_location_id
LEFT JOIN lot l ON l.id = ir.lot_id
JOIN account oa ON oa.id = ir.owner_account_id
JOIN account ha ON ha.id = ir.holder_account_id
LEFT JOIN (
    SELECT ia.inventory_receipt_id, SUM(CAST(aq.value AS DECIMAL(65,30))) AS total_allocated
    FROM inventory_allocation ia
    JOIN quantity aq ON aq.id = ia.quantity_id
    GROUP BY ia.inventory_receipt_id
) alloc_sum ON alloc_sum.inventory_receipt_id = ir.id
WHERE (ir.owner_account_id = sqlc.arg('requesting_account_id') OR ir.holder_account_id = sqlc.arg('requesting_account_id'))
  AND ir.status_code = 'available'
GROUP BY it.id, it.sku, it.description, sl.id, sl.name, l.id, l.lot_number,
         ir.owner_account_id, oa.name, ir.holder_account_id, ha.name,
         ir_u.abbreviation, ir_u.name,
         r_cost_nu.abbreviation, r_cost_nu.name,
         r_cost_du.abbreviation, r_cost_du.name;

-- name: GetOeeDepartmentData :many
SELECT
    COALESCE(d.id, 'unassigned') AS department_id,
    COALESCE(d.name, 'Unassigned') AS department_name,
    CAST(COALESCE(SUM(COALESCE(qf.value * (u_qf.ratio_numerator / u_qf.ratio_denominator), 0)), 0) AS DECIMAL(65,30)) AS good_units,
    CAST(COALESCE(SUM(COALESCE(qw.value * (u_qw.ratio_numerator / u_qw.ratio_denominator), 0)), 0) AS DECIMAL(65,30)) AS waste_units,
    CAST(COALESCE(SUM(COALESCE(qs.value * (u_qs.ratio_numerator / u_qs.ratio_denominator), 0)), 0) AS DECIMAL(65,30)) AS seconds_units
FROM batch b
LEFT JOIN quantity qf ON qf.id = b.quantity_id
LEFT JOIN unit u_qf ON u_qf.id = qf.unit_id
LEFT JOIN quantity qw ON qw.id = b.waste_quantity_id
LEFT JOIN unit u_qw ON u_qw.id = qw.unit_id
LEFT JOIN quantity qs ON qs.id = b.seconds_quantity_id
LEFT JOIN unit u_qs ON u_qs.id = qs.unit_id
LEFT JOIN scanning_station ss ON ss.id = b.scanning_station_id
LEFT JOIN department d ON d.id = ss.department_id
WHERE b.account_id = sqlc.arg('owner_account_id')
  AND b.scanned_at >= sqlc.arg('start_date')
  AND b.scanned_at <= sqlc.arg('end_date')
GROUP BY d.id, d.name;

-- name: GetOeeEstimatedRuntime :many
SELECT
    department_id,
    SUM(TIMESTAMPDIFF(SECOND, day_first, day_last)) AS runtime_seconds
FROM (
    SELECT
        COALESCE(ss.department_id, 'unassigned') AS department_id,
        DATE(b.scanned_at) AS scan_date,
        MIN(b.scanned_at) AS day_first,
        MAX(b.scanned_at) AS day_last
    FROM batch b
    LEFT JOIN scanning_station ss ON ss.id = b.scanning_station_id
    WHERE b.account_id = sqlc.arg('owner_account_id')
      AND b.scanned_at >= sqlc.arg('start_date')
      AND b.scanned_at <= sqlc.arg('end_date')
    GROUP BY COALESCE(ss.department_id, 'unassigned'), DATE(b.scanned_at)
) daily
GROUP BY department_id;

-- name: GetDemandForecastMonthlyDemand :many
SELECT
    it.id AS item_id,
    it.sku AS product_sku,
    it.description AS product_description,
    pl.id AS product_line_id,
    sol_u.abbreviation AS unit,
    COALESCE(u_price_num.abbreviation, '$') AS currency,
    YEAR(so.created_at) AS demand_year,
    MONTH(so.created_at) AS demand_month,
    COALESCE(SUM(CAST(sol_q.value AS DECIMAL(65,30))), 0) AS monthly_demand,
    COALESCE(SUM(CAST(sol_q.value AS DECIMAL(65,30)) * COALESCE(CAST(r_sell.value AS DECIMAL(65,30)), 0)), 0) AS monthly_revenue
FROM sales_order so
JOIN sales_order_line sol ON sol.sales_order_id = so.id
JOIN product p ON p.id = sol.product_id
JOIN item it ON it.id = p.item_id
JOIN quantity sol_q ON sol_q.id = sol.quantity_id
JOIN unit sol_u ON sol_u.id = sol_q.unit_id
LEFT JOIN product_line pl ON pl.id = p.product_line_id
LEFT JOIN rate r_sell ON r_sell.id = sol.unit_price_id
LEFT JOIN unit u_price_num ON u_price_num.id = r_sell.numerator_unit_id
WHERE so.owner_account_id = sqlc.arg('owner_account_id')
  AND so.created_at >= sqlc.arg('start_date')
  AND so.created_at < sqlc.arg('end_date')
  AND so.sales_order_status_code != 'cancelled'
  AND p.product_type_code = 'sale'
GROUP BY it.id, it.sku, it.description, pl.id, sol_u.abbreviation, u_price_num.abbreviation, YEAR(so.created_at), MONTH(so.created_at)
ORDER BY it.id, demand_year, demand_month;

-- name: GetDemandForecastMonthlyRevenue :many
SELECT
    it.id AS item_id,
    YEAR(inv.created_at) AS revenue_year,
    MONTH(inv.created_at) AS revenue_month,
    COALESCE(SUM(CAST(il_q.value AS DECIMAL(65,30)) * COALESCE(CAST(r_sell.value AS DECIMAL(65,30)), 0)), 0) AS monthly_revenue
FROM invoice inv
JOIN invoice_line il ON il.invoice_id = inv.id
JOIN sales_order_line sol ON sol.id = il.sales_order_line_id
JOIN sales_order so ON so.id = sol.sales_order_id
JOIN product p ON p.id = sol.product_id
JOIN item it ON it.id = p.item_id
JOIN quantity il_q ON il_q.id = il.quantity_id
LEFT JOIN rate r_sell ON r_sell.id = sol.unit_price_id
WHERE so.owner_account_id = sqlc.arg('owner_account_id')
  AND inv.created_at >= sqlc.arg('start_date')
  AND inv.created_at < sqlc.arg('end_date')
  AND p.product_type_code = 'sale'
GROUP BY it.id, YEAR(inv.created_at), MONTH(inv.created_at)
ORDER BY it.id, revenue_year, revenue_month;

-- name: GetSaleProductItemIDs :many
SELECT
    p.item_id,
    p.product_line_id
FROM product p
JOIN item i ON i.id = p.item_id
WHERE i.account_id = sqlc.arg('owner_account_id')
  AND p.product_type_code = 'sale'
  AND i.deleted_at IS NULL;

-- name: GetOrderQuantityByProductLine :one
SELECT
    COALESCE(SUM(CAST(sol_q.value AS DECIMAL(65,30))), 0) AS total_quantity,
    COALESCE(
        (SELECT bu.abbreviation FROM product_line pl2
         JOIN unit_group ug ON pl2.unit_group_id = ug.id
         JOIN unit bu ON ug.base_unit_id = bu.id
         WHERE pl2.id COLLATE utf8mb4_general_ci = CAST(sqlc.arg('target_product_line_id') AS CHAR(191)) LIMIT 1), ''
    ) AS unit_abbreviation,
    COALESCE(
        (SELECT ug.unit_type_code FROM product_line pl3
         JOIN unit_group ug ON pl3.unit_group_id = ug.id
         WHERE pl3.id COLLATE utf8mb4_general_ci = CAST(sqlc.arg('target_product_line_id') AS CHAR(191)) LIMIT 1), ''
    ) AS unit_type
FROM sales_order so
JOIN sales_order_line sol ON sol.sales_order_id = so.id
JOIN product p ON p.id = sol.product_id
JOIN item i ON i.id = p.item_id
JOIN quantity sol_q ON sol_q.id = sol.quantity_id
WHERE so.owner_account_id = sqlc.arg('owner_account_id')
  AND p.product_line_id COLLATE utf8mb4_general_ci = CAST(sqlc.arg('target_product_line_id') AS CHAR(191))
  AND so.issued_at >= sqlc.arg('start_date')
  AND so.issued_at <= sqlc.arg('end_date');

-- name: GetProductLineInfo :many
SELECT
    pl.id,
    pl.name
FROM product_line pl
WHERE pl.id IN (sqlc.slice('product_line_ids'))
  AND (pl.account_id = sqlc.arg('owner_account_id') OR pl.account_id IS NULL);

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

-- name: GetMaterialsWithDetails :many
SELECT
    m.id AS material_id,
    it.id AS item_id,
    it.sku AS item_sku,
    it.description AS item_description,
    CAST(op_q.value AS DECIMAL(65,30)) AS order_point_value,
    op_u.name AS order_point_unit_name,
    op_u.abbreviation AS order_point_unit_abbreviation,
    op_u.unit_dimension_code AS order_point_unit_type,
    CAST(lt_q.value AS DECIMAL(65,30)) AS lead_time_value,
    lt_u.name AS lead_time_unit_name,
    lt_u.abbreviation AS lead_time_unit_abbreviation,
    lt_u.unit_dimension_code AS lead_time_unit_type,
    ug.id AS unit_group_id,
    ug.name AS unit_group_name
FROM material m
JOIN item it ON it.id = m.item_id
JOIN quantity op_q ON op_q.id = m.order_point_id
JOIN unit op_u ON op_u.id = op_q.unit_id
JOIN quantity lt_q ON lt_q.id = m.lead_time_id
JOIN unit lt_u ON lt_u.id = lt_q.unit_id
JOIN item_category ic ON ic.id = it.item_category_id
JOIN unit_group ug ON ug.id = ic.unit_group_id
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

-- name: GetMaterialOnHandByItem :many
SELECT
    ir.item_id,
    SUM(GREATEST(
        CAST(ir_q.value AS DECIMAL(65,30)) - COALESCE(alloc_sum.total_allocated, 0),
        0
    )) AS remaining_quantity
FROM inventory_receipt ir
JOIN quantity ir_q ON ir_q.id = ir.quantity_id
LEFT JOIN (
    SELECT ia.inventory_receipt_id, SUM(CAST(aq.value AS DECIMAL(65,30))) AS total_allocated
    FROM inventory_allocation ia
    JOIN quantity aq ON aq.id = ia.quantity_id
    GROUP BY ia.inventory_receipt_id
) alloc_sum ON alloc_sum.inventory_receipt_id = ir.id
WHERE (ir.owner_account_id = sqlc.arg('account_id') OR ir.holder_account_id = sqlc.arg('account_id'))
  AND ir.item_id IN (sqlc.slice('item_ids'))
  AND ir.status_code = 'available'
GROUP BY ir.item_id;

-- name: GetMaterialReservedByItem :many
SELECT
    ii.item_id,
    SUM(GREATEST(
        CAST(ii_q.value AS DECIMAL(65,30)) - COALESCE(alloc_sum.total_allocated, 0),
        0
    )) AS remaining_quantity
FROM inventory_issue ii
JOIN quantity ii_q ON ii_q.id = ii.quantity_id
LEFT JOIN (
    SELECT ia.inventory_issue_id, SUM(CAST(aq.value AS DECIMAL(65,30))) AS total_allocated
    FROM inventory_allocation ia
    JOIN quantity aq ON aq.id = ia.quantity_id
    GROUP BY ia.inventory_issue_id
) alloc_sum ON alloc_sum.inventory_issue_id = ii.id
WHERE ii.account_id = sqlc.arg('account_id')
  AND ii.item_id IN (sqlc.slice('item_ids'))
  AND ii.status_code = 'reserved'
GROUP BY ii.item_id;

-- name: GetMaterialOpenByItem :many
SELECT
    ii.item_id,
    SUM(GREATEST(
        CAST(ii_q.value AS DECIMAL(65,30)) - COALESCE(alloc_sum.total_allocated, 0),
        0
    )) AS remaining_quantity
FROM inventory_issue ii
JOIN quantity ii_q ON ii_q.id = ii.quantity_id
LEFT JOIN (
    SELECT ia.inventory_issue_id, SUM(CAST(aq.value AS DECIMAL(65,30))) AS total_allocated
    FROM inventory_allocation ia
    JOIN quantity aq ON aq.id = ia.quantity_id
    GROUP BY ia.inventory_issue_id
) alloc_sum ON alloc_sum.inventory_issue_id = ii.id
WHERE ii.account_id = sqlc.arg('account_id')
  AND ii.item_id IN (sqlc.slice('item_ids'))
  AND ii.status_code = 'open'
GROUP BY ii.item_id;

-- name: GetMaterialSupplierInfo :many
SELECT
    m.item_id,
    sm.supplier_part_number,
    a.name AS supplier_name
FROM supplier_material sm
JOIN material m ON m.id = sm.material_id
JOIN account a ON a.id = sm.supplier_account_id
WHERE sm.owner_account_id = sqlc.arg('owner_account_id')
  AND sm.supplier_account_id IN (sqlc.slice('supplier_ids'));
