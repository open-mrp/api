-- name: ListPicksForward :many
SELECT
    p.id,
    p.number,
    p.sales_order_id,
    so.number AS sales_order_number,
    ar.counterparty_account_id AS customer_id,
    ba.name AS customer_name,
    ar.external_number AS customer_number,
    so.priority_code,
    pr.id AS priority_id,
    pr.name AS priority_name,
    p.finished_at,
    p.created_at,
    p.updated_at,
    (SELECT COUNT(*) FROM pick_line plc WHERE plc.pick_id = p.id) AS line_count,
    -- Latest ship date across the order's shipments; drives the date in the pick header.
    (SELECT MAX(sh.shipped_at) FROM shipment sh WHERE sh.sales_order_id = so.id) AS last_shipped_at,
    so.promised_at,
    -- The order's cross-reference and instructions, carried so the floor works the pick without opening the order.
    so.customer_po_number,
    so.note,
    -- Freight is the order's, carried so a pick shows the carrier it ships on.
    so.carrier_id,
    cr.name AS carrier_name,
    cr.is_portal_enabled AS carrier_is_portal_enabled,
    cr.created_at AS carrier_created_at,
    cr.updated_at AS carrier_updated_at,
    so.carrier_option_id AS service_level_id,
    co.name AS service_level_name,
    co.is_portal_enabled AS service_level_is_portal_enabled,
    co.service_level_token,
    co.created_at AS service_level_created_at,
    co.updated_at AS service_level_updated_at,
    so.carrier_billing_type,
    so.carrier_billing_account,
    -- The order's delivery commitment and how it was derived, so a pick can explain its dates.
    so.ship_by_date,
    so.ship_by_cutoff_at,
    so.lead_time_days,
    so.lead_time_source_code,
    so.transit_days,
    so.transit_source_code,
    -- Ship-to is the order's, denormalized so a pick header needs no second fetch.
    so.shipping_address_id,
    addr.name AS shipping_address_name,
    addr.phone AS shipping_address_phone,
    addr.email AS shipping_address_email,
    addr.is_drop_ship AS shipping_address_is_drop_ship,
    ship_geo.id AS shipping_address_geolocation_id,
    ship_geo.street_line_1 AS shipping_address_street_line_1,
    ship_geo.street_line_2 AS shipping_address_street_line_2,
    ship_geo.locality AS shipping_address_locality,
    ship_geo.state AS shipping_address_state,
    ship_geo.postal_code AS shipping_address_postal_code,
    ship_geo.country AS shipping_address_country,
    addr.created_at AS shipping_address_created_at,
    addr.updated_at AS shipping_address_updated_at
FROM pick p
JOIN sales_order so ON so.id = p.sales_order_id
JOIN account_relation ar ON ar.owner_account_id = so.owner_account_id
    AND ar.counterparty_account_id = so.buyer_account_id
JOIN account ba ON ba.id = so.buyer_account_id
JOIN priority pr ON pr.code = so.priority_code
LEFT JOIN address addr ON addr.id = so.shipping_address_id
LEFT JOIN geolocation ship_geo ON ship_geo.id = addr.geolocation_id
LEFT JOIN carrier cr ON cr.id = so.carrier_id
LEFT JOIN carrier_option co ON co.id = so.carrier_option_id
WHERE p.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR p.number LIKE sqlc.narg('search_query')
    OR so.number LIKE sqlc.narg('search_query')
    OR so.customer_po_number LIKE sqlc.narg('search_query')
    OR ba.name LIKE sqlc.narg('search_query')
    OR ar.external_number LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('status') IS NULL
    OR (sqlc.narg('status') = 'open' AND p.finished_at IS NULL)
    OR (sqlc.narg('status') = 'closed' AND p.finished_at IS NOT NULL)
)
AND (
    sqlc.arg('include_customer_filter') = false
    OR so.buyer_account_id IN (sqlc.slice('customer_ids'))
)
AND (
    sqlc.arg('include_customer_group_filter') = false
    OR ar.account_group_id IN (sqlc.slice('customer_group_ids'))
)
AND (
    sqlc.arg('include_product_line_filter') = false
    OR EXISTS (
        SELECT 1 FROM pick_line pl2
        JOIN sales_order_line sol2 ON sol2.id = pl2.sales_order_line_id
        JOIN product prod ON prod.id = sol2.product_id
        WHERE pl2.pick_id = p.id
        AND prod.product_line_id IN (sqlc.slice('product_line_ids'))
    )
)
AND (
    sqlc.narg('start_date') IS NULL
    OR p.created_at >= sqlc.narg('start_date')
)
AND (
    sqlc.narg('end_date') IS NULL
    OR p.created_at <= sqlc.narg('end_date')
)
AND (
    sqlc.arg('sort_by_ship_by') = true
    OR sqlc.narg('cursor_created_at') IS NULL
    OR p.created_at < sqlc.narg('cursor_created_at')
    OR (p.created_at = sqlc.narg('cursor_created_at') AND p.id < sqlc.narg('cursor_id'))
)
-- The sentinel keeps a pick whose order has no ship-by date sortable and last; the repository's cursor must use the same value.
AND (
    sqlc.arg('sort_by_ship_by') = false
    OR sqlc.narg('cursor_ship_by_date') IS NULL
    OR COALESCE(so.ship_by_date, '9999-12-31') > CAST(sqlc.narg('cursor_ship_by_date') AS DATE)
    OR (
        COALESCE(so.ship_by_date, '9999-12-31') = CAST(sqlc.narg('cursor_ship_by_date') AS DATE)
        AND p.id > sqlc.narg('cursor_id')
    )
)
ORDER BY
    CASE WHEN sqlc.arg('sort_by_ship_by') = true THEN COALESCE(so.ship_by_date, '9999-12-31') END ASC,
    CASE WHEN sqlc.arg('sort_by_ship_by') = true THEN p.id END ASC,
    p.created_at DESC,
    p.id DESC
LIMIT ?;

-- name: ListPicksBackward :many
SELECT
    p.id,
    p.number,
    p.sales_order_id,
    so.number AS sales_order_number,
    ar.counterparty_account_id AS customer_id,
    ba.name AS customer_name,
    ar.external_number AS customer_number,
    so.priority_code,
    pr.id AS priority_id,
    pr.name AS priority_name,
    p.finished_at,
    p.created_at,
    p.updated_at,
    (SELECT COUNT(*) FROM pick_line plc WHERE plc.pick_id = p.id) AS line_count,
    -- Latest ship date across the order's shipments; drives the date in the pick header.
    (SELECT MAX(sh.shipped_at) FROM shipment sh WHERE sh.sales_order_id = so.id) AS last_shipped_at,
    so.promised_at,
    -- The order's cross-reference and instructions, carried so the floor works the pick without opening the order.
    so.customer_po_number,
    so.note,
    -- Freight is the order's, carried so a pick shows the carrier it ships on.
    so.carrier_id,
    cr.name AS carrier_name,
    cr.is_portal_enabled AS carrier_is_portal_enabled,
    cr.created_at AS carrier_created_at,
    cr.updated_at AS carrier_updated_at,
    so.carrier_option_id AS service_level_id,
    co.name AS service_level_name,
    co.is_portal_enabled AS service_level_is_portal_enabled,
    co.service_level_token,
    co.created_at AS service_level_created_at,
    co.updated_at AS service_level_updated_at,
    so.carrier_billing_type,
    so.carrier_billing_account,
    -- The order's delivery commitment and how it was derived, so a pick can explain its dates.
    so.ship_by_date,
    so.ship_by_cutoff_at,
    so.lead_time_days,
    so.lead_time_source_code,
    so.transit_days,
    so.transit_source_code,
    -- Ship-to is the order's, denormalized so a pick header needs no second fetch.
    so.shipping_address_id,
    addr.name AS shipping_address_name,
    addr.phone AS shipping_address_phone,
    addr.email AS shipping_address_email,
    addr.is_drop_ship AS shipping_address_is_drop_ship,
    ship_geo.id AS shipping_address_geolocation_id,
    ship_geo.street_line_1 AS shipping_address_street_line_1,
    ship_geo.street_line_2 AS shipping_address_street_line_2,
    ship_geo.locality AS shipping_address_locality,
    ship_geo.state AS shipping_address_state,
    ship_geo.postal_code AS shipping_address_postal_code,
    ship_geo.country AS shipping_address_country,
    addr.created_at AS shipping_address_created_at,
    addr.updated_at AS shipping_address_updated_at
FROM pick p
JOIN sales_order so ON so.id = p.sales_order_id
JOIN account_relation ar ON ar.owner_account_id = so.owner_account_id
    AND ar.counterparty_account_id = so.buyer_account_id
JOIN account ba ON ba.id = so.buyer_account_id
JOIN priority pr ON pr.code = so.priority_code
LEFT JOIN address addr ON addr.id = so.shipping_address_id
LEFT JOIN geolocation ship_geo ON ship_geo.id = addr.geolocation_id
LEFT JOIN carrier cr ON cr.id = so.carrier_id
LEFT JOIN carrier_option co ON co.id = so.carrier_option_id
WHERE p.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR p.number LIKE sqlc.narg('search_query')
    OR so.number LIKE sqlc.narg('search_query')
    OR so.customer_po_number LIKE sqlc.narg('search_query')
    OR ba.name LIKE sqlc.narg('search_query')
    OR ar.external_number LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('status') IS NULL
    OR (sqlc.narg('status') = 'open' AND p.finished_at IS NULL)
    OR (sqlc.narg('status') = 'closed' AND p.finished_at IS NOT NULL)
)
AND (
    sqlc.arg('include_customer_filter') = false
    OR so.buyer_account_id IN (sqlc.slice('customer_ids'))
)
AND (
    sqlc.arg('include_customer_group_filter') = false
    OR ar.account_group_id IN (sqlc.slice('customer_group_ids'))
)
AND (
    sqlc.arg('include_product_line_filter') = false
    OR EXISTS (
        SELECT 1 FROM pick_line pl2
        JOIN sales_order_line sol2 ON sol2.id = pl2.sales_order_line_id
        JOIN product prod ON prod.id = sol2.product_id
        WHERE pl2.pick_id = p.id
        AND prod.product_line_id IN (sqlc.slice('product_line_ids'))
    )
)
AND (
    sqlc.narg('start_date') IS NULL
    OR p.created_at >= sqlc.narg('start_date')
)
AND (
    sqlc.narg('end_date') IS NULL
    OR p.created_at <= sqlc.narg('end_date')
)
AND (
    sqlc.arg('sort_by_ship_by') = true
    OR p.created_at > sqlc.arg('cursor_created_at')
    OR (p.created_at = sqlc.arg('cursor_created_at') AND p.id > sqlc.arg('cursor_id'))
)
-- The sentinel keeps a pick whose order has no ship-by date sortable and last; the repository's cursor must use the same value.
AND (
    sqlc.arg('sort_by_ship_by') = false
    OR COALESCE(so.ship_by_date, '9999-12-31') < CAST(sqlc.arg('cursor_ship_by_date') AS DATE)
    OR (
        COALESCE(so.ship_by_date, '9999-12-31') = CAST(sqlc.arg('cursor_ship_by_date') AS DATE)
        AND p.id < sqlc.arg('cursor_id')
    )
)
ORDER BY
    CASE WHEN sqlc.arg('sort_by_ship_by') = true THEN COALESCE(so.ship_by_date, '9999-12-31') END DESC,
    CASE WHEN sqlc.arg('sort_by_ship_by') = true THEN p.id END DESC,
    p.created_at ASC,
    p.id ASC
LIMIT ?;

-- name: CountPicks :one
SELECT COUNT(DISTINCT p.id) AS total
FROM pick p
JOIN sales_order so ON so.id = p.sales_order_id
JOIN account_relation ar ON ar.owner_account_id = so.owner_account_id
    AND ar.counterparty_account_id = so.buyer_account_id
JOIN account ba ON ba.id = so.buyer_account_id
WHERE p.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR p.number LIKE sqlc.narg('search_query')
    OR so.number LIKE sqlc.narg('search_query')
    OR so.customer_po_number LIKE sqlc.narg('search_query')
    OR ba.name LIKE sqlc.narg('search_query')
    OR ar.external_number LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('status') IS NULL
    OR (sqlc.narg('status') = 'open' AND p.finished_at IS NULL)
    OR (sqlc.narg('status') = 'closed' AND p.finished_at IS NOT NULL)
)
AND (
    sqlc.arg('include_customer_filter') = false
    OR so.buyer_account_id IN (sqlc.slice('customer_ids'))
)
AND (
    sqlc.arg('include_customer_group_filter') = false
    OR ar.account_group_id IN (sqlc.slice('customer_group_ids'))
)
AND (
    sqlc.arg('include_product_line_filter') = false
    OR EXISTS (
        SELECT 1 FROM pick_line pl2
        JOIN sales_order_line sol2 ON sol2.id = pl2.sales_order_line_id
        JOIN product prod ON prod.id = sol2.product_id
        WHERE pl2.pick_id = p.id
        AND prod.product_line_id IN (sqlc.slice('product_line_ids'))
    )
)
AND (
    sqlc.narg('start_date') IS NULL
    OR p.created_at >= sqlc.narg('start_date')
)
AND (
    sqlc.narg('end_date') IS NULL
    OR p.created_at <= sqlc.narg('end_date')
);

-- name: GetPick :one
SELECT
    p.id,
    p.number,
    p.sales_order_id,
    so.number AS sales_order_number,
    ar.counterparty_account_id AS customer_id,
    ba.name AS customer_name,
    ar.external_number AS customer_number,
    so.priority_code,
    pr.id AS priority_id,
    pr.name AS priority_name,
    p.finished_at,
    p.created_at,
    p.updated_at,
    (SELECT COUNT(*) FROM pick_line plc WHERE plc.pick_id = p.id) AS line_count,
    -- Latest ship date across the order's shipments; drives the date in the pick header.
    (SELECT MAX(sh.shipped_at) FROM shipment sh WHERE sh.sales_order_id = so.id) AS last_shipped_at,
    so.promised_at,
    -- The order's cross-reference and instructions, carried so the floor works the pick without opening the order.
    so.customer_po_number,
    so.note,
    -- Freight is the order's, carried so a pick shows the carrier it ships on.
    so.carrier_id,
    cr.name AS carrier_name,
    cr.is_portal_enabled AS carrier_is_portal_enabled,
    cr.created_at AS carrier_created_at,
    cr.updated_at AS carrier_updated_at,
    so.carrier_option_id AS service_level_id,
    co.name AS service_level_name,
    co.is_portal_enabled AS service_level_is_portal_enabled,
    co.service_level_token,
    co.created_at AS service_level_created_at,
    co.updated_at AS service_level_updated_at,
    so.carrier_billing_type,
    so.carrier_billing_account,
    -- The order's delivery commitment and how it was derived, so a pick can explain its dates.
    so.ship_by_date,
    so.ship_by_cutoff_at,
    so.lead_time_days,
    so.lead_time_source_code,
    so.transit_days,
    so.transit_source_code,
    -- Ship-to is the order's, denormalized so a pick header needs no second fetch.
    so.shipping_address_id,
    addr.name AS shipping_address_name,
    addr.phone AS shipping_address_phone,
    addr.email AS shipping_address_email,
    addr.is_drop_ship AS shipping_address_is_drop_ship,
    ship_geo.id AS shipping_address_geolocation_id,
    ship_geo.street_line_1 AS shipping_address_street_line_1,
    ship_geo.street_line_2 AS shipping_address_street_line_2,
    ship_geo.locality AS shipping_address_locality,
    ship_geo.state AS shipping_address_state,
    ship_geo.postal_code AS shipping_address_postal_code,
    ship_geo.country AS shipping_address_country,
    addr.created_at AS shipping_address_created_at,
    addr.updated_at AS shipping_address_updated_at
FROM pick p
JOIN sales_order so ON so.id = p.sales_order_id
JOIN account_relation ar ON ar.owner_account_id = so.owner_account_id
    AND ar.counterparty_account_id = so.buyer_account_id
JOIN account ba ON ba.id = so.buyer_account_id
JOIN priority pr ON pr.code = so.priority_code
LEFT JOIN address addr ON addr.id = so.shipping_address_id
LEFT JOIN geolocation ship_geo ON ship_geo.id = addr.geolocation_id
LEFT JOIN carrier cr ON cr.id = so.carrier_id
LEFT JOIN carrier_option co ON co.id = so.carrier_option_id
WHERE p.id = sqlc.arg('pick_id')
AND p.account_id = sqlc.arg('account_id');

-- name: GetPickLines :many
SELECT
    pl.id,
    pl.pick_id,
    pl.sales_order_line_id,
    pl.packed_at,
    pl.created_at,
    pl.updated_at,
    -- Pick line quantity
    q.id AS quantity_id,
    q.value AS quantity_value,
    u.id AS quantity_unit_id,
    u.name AS quantity_unit_name,
    u.abbreviation AS quantity_unit_abbreviation,
    -- Sales order line info
    sol.line_item_number,
    sol.product_sku,
    sol.product_description,
    sol.product_id,
    sol.item_id AS order_line_item_id,
    -- Ordered quantity
    sol_q.id AS ordered_quantity_id,
    sol_q.value AS ordered_quantity_value,
    sol_u.id AS ordered_quantity_unit_id,
    sol_u.name AS ordered_quantity_unit_name,
    sol_u.abbreviation AS ordered_quantity_unit_abbreviation,
    -- Unit price (sales order line)
    up.id AS unit_price_id,
    up.value AS unit_price_value,
    up_nu.id AS unit_price_numerator_unit_id,
    up_nu.abbreviation AS unit_price_numerator_unit_abbreviation,
    up_du.id AS unit_price_denominator_unit_id,
    up_du.abbreviation AS unit_price_denominator_unit_abbreviation
FROM pick_line pl
JOIN quantity q ON q.id = pl.quantity_id
JOIN unit u ON u.id = q.unit_id
JOIN sales_order_line sol ON sol.id = pl.sales_order_line_id
JOIN quantity sol_q ON sol_q.id = sol.quantity_id
JOIN unit sol_u ON sol_u.id = sol_q.unit_id
JOIN rate up ON up.id = sol.unit_price_id
JOIN unit up_nu ON up_nu.id = up.numerator_unit_id
JOIN unit up_du ON up_du.id = up.denominator_unit_id
WHERE pl.pick_id = sqlc.arg('pick_id')
ORDER BY sol.line_item_number ASC;

-- name: GetPickProgress :many
-- Aggregates ordered/picked/packed quantities per pick in one batched pass, keyed by
-- pick ID. Backs the pick-level picked/packed completion fractions on both the list and
-- detail endpoints without loading each pick's lines. Only product_type_code = 'sale'
-- lines are counted, matching the frontend's completion math. Picks whose lines are all
-- non-sale are absent from the result and read as zero progress.
SELECT
    pl.pick_id,
    COALESCE(SUM(solq.value), 0) AS quantity_ordered,
    COALESCE(SUM(plq.value), 0) AS quantity_picked,
    COALESCE(SUM(CASE WHEN pl.packed_at IS NOT NULL THEN plq.value ELSE 0 END), 0) AS quantity_packed
FROM pick_line pl
JOIN quantity plq ON plq.id = pl.quantity_id
JOIN sales_order_line sol ON sol.id = pl.sales_order_line_id
JOIN quantity solq ON solq.id = sol.quantity_id
JOIN product p ON p.id = sol.product_id
WHERE pl.pick_id IN (sqlc.slice('pick_ids'))
AND p.product_type_code = 'sale'
GROUP BY pl.pick_id;

-- name: UpdatePickFinishedAt :exec
UPDATE pick SET
    finished_at = sqlc.arg('finished_at'),
    updated_at = NOW(3)
WHERE id = sqlc.arg('pick_id')
AND account_id = sqlc.arg('account_id');

-- name: ClearPickFinishedAt :exec
UPDATE pick SET
    finished_at = NULL,
    updated_at = NOW(3)
WHERE id = sqlc.arg('pick_id')
AND account_id = sqlc.arg('account_id');

-- name: HasShippedItems :one
SELECT EXISTS(
    SELECT 1 FROM shipment s
    WHERE s.sales_order_id = (
        SELECT pk.sales_order_id FROM pick pk
        WHERE pk.id = sqlc.arg('pick_id')
        AND pk.account_id = sqlc.arg('account_id')
    )
) AS has_shipped;

-- name: VoidAllPickLines :exec
-- Skips a finished pick's lines: voiding one clears finished_at but must not rewrite work that
-- was already completed (Dashboard filters the same way on pick.finishedAt).
UPDATE quantity SET
    value = 0,
    updated_at = NOW(3)
WHERE id IN (
    SELECT pl.quantity_id FROM pick_line pl
    JOIN pick p ON p.id = pl.pick_id
    WHERE pl.pick_id = sqlc.arg('pick_id')
    AND pl.packed_at IS NULL
    AND p.finished_at IS NULL
);

-- name: DeleteDuplicatePickLines :exec
DELETE FROM pick_line WHERE id IN (
    SELECT id FROM (
        SELECT pl.id
        FROM pick_line pl
        INNER JOIN (
            SELECT pl_inner.sales_order_line_id, MIN(pl_inner.id) AS min_id
            FROM pick_line pl_inner
            WHERE pl_inner.sales_order_line_id IN (
                SELECT pl_filter.sales_order_line_id FROM pick_line pl_filter WHERE pl_filter.pick_id = sqlc.arg('pick_id')
            )
            GROUP BY pl_inner.sales_order_line_id
            HAVING COUNT(*) > 1
        ) dup ON pl.sales_order_line_id = dup.sales_order_line_id AND pl.id != dup.min_id
        INNER JOIN sales_order_line sol ON sol.id = pl.sales_order_line_id
        INNER JOIN sales_order so ON so.id = sol.sales_order_id
        WHERE so.owner_account_id = sqlc.arg('account_id')
    ) AS to_delete
);

-- name: PickAllLines :exec
UPDATE quantity q
JOIN pick_line pl ON pl.quantity_id = q.id
JOIN sales_order_line sol ON sol.id = pl.sales_order_line_id
JOIN quantity sol_q ON sol_q.id = sol.quantity_id
LEFT JOIN (
    SELECT
        pl_sum.sales_order_line_id,
        SUM(q_sum.value) AS total_picked_value
    FROM pick_line pl_sum
    JOIN quantity q_sum ON q_sum.id = pl_sum.quantity_id
    -- Restrict the aggregate to this pick's own lines: without this the derived table groups every pick_line in the database while the update holds its locks. It stays a derived table rather than a correlated subquery because `quantity` is the table being updated.
    WHERE pl_sum.sales_order_line_id IN (
        SELECT pl_scope.sales_order_line_id
        FROM pick_line pl_scope
        WHERE pl_scope.pick_id = sqlc.arg('pick_id')
    )
    GROUP BY pl_sum.sales_order_line_id
) picked ON picked.sales_order_line_id = pl.sales_order_line_id
JOIN pick p ON p.id = pl.pick_id
SET q.value = GREATEST(0, sol_q.value - GREATEST(COALESCE(picked.total_picked_value, 0) - q.value, 0)),
    q.updated_at = NOW(3)
WHERE pl.pick_id = sqlc.arg('pick_id')
AND pl.packed_at IS NULL
-- A finished pick is completed work; Dashboard filters its pick-all on pick.finishedAt too.
AND p.finished_at IS NULL;

-- name: PickRemainingQuantityForLine :exec
UPDATE quantity q
JOIN pick_line pl ON pl.quantity_id = q.id
JOIN sales_order_line sol ON sol.id = pl.sales_order_line_id
JOIN quantity sol_q ON sol_q.id = sol.quantity_id
LEFT JOIN (
    SELECT
        pl_sum.sales_order_line_id,
        SUM(q_sum.value) AS total_picked_value
    FROM pick_line pl_sum
    JOIN quantity q_sum ON q_sum.id = pl_sum.quantity_id
    -- Restrict the aggregate to the line being picked: without this the derived table groups every pick_line in the database while the update holds its locks. It stays a derived table rather than a correlated subquery because `quantity` is the table being updated.
    WHERE pl_sum.sales_order_line_id IN (
        SELECT pl_scope.sales_order_line_id
        FROM pick_line pl_scope
        WHERE pl_scope.id = sqlc.arg('pick_line_id')
    )
    -- The line being picked is excluded from its own outstanding calculation. Counting it would subtract what it already holds, so picking an already-picked line set it back to zero — a second click wiping a picker's work rather than doing nothing.
    AND pl_sum.id != sqlc.arg('pick_line_id')
    GROUP BY pl_sum.sales_order_line_id
) picked ON picked.sales_order_line_id = pl.sales_order_line_id
-- Remaining excludes this line's own quantity, matching PickAllLines. Dashboard subtracts the
-- total including self, so picking a line already holding 3 of 10 leaves it at 7 rather than
-- filling it to 10 — a deliberate divergence, and the two picking paths must agree.
SET q.value = GREATEST(0, sol_q.value - GREATEST(COALESCE(picked.total_picked_value, 0) - q.value, 0)),
    q.updated_at = NOW(3)
WHERE pl.id = sqlc.arg('pick_line_id')
AND pl.packed_at IS NULL;

-- name: VoidPickLine :exec
UPDATE quantity SET
    value = 0,
    updated_at = NOW(3)
WHERE id = (
    SELECT pl.quantity_id FROM pick_line pl
    WHERE pl.id = sqlc.arg('pick_line_id')
);

-- name: UpdatePickLineQuantity :exec
UPDATE quantity SET
    value = COALESCE(sqlc.narg('value'), value),
    unit_id = COALESCE(sqlc.narg('unit_id'), unit_id),
    updated_at = NOW(3)
WHERE id = (
    SELECT pl.quantity_id FROM pick_line pl
    WHERE pl.id = sqlc.arg('pick_line_id')
);

-- name: IsPickInAccount :one
SELECT EXISTS(
    SELECT 1 FROM pick
    WHERE id = sqlc.arg('pick_id')
    AND account_id = sqlc.arg('account_id')
) AS is_in_account;

-- name: FindLinesToPack :many
SELECT
    pl.id,
    pl.pick_id,
    pl.sales_order_line_id,
    pl.packed_at,
    pl.created_at,
    pl.updated_at,
    -- Pick line quantity
    q.id AS quantity_id,
    q.value AS quantity_value,
    u.id AS quantity_unit_id,
    u.name AS quantity_unit_name,
    u.abbreviation AS quantity_unit_abbreviation,
    -- Sales order line info
    sol.line_item_number,
    sol.product_sku,
    sol.product_description,
    sol.product_id,
    sol.item_id AS order_line_item_id,
    -- Ordered quantity
    sol_q.id AS ordered_quantity_id,
    sol_q.value AS ordered_quantity_value,
    sol_u.id AS ordered_quantity_unit_id,
    sol_u.name AS ordered_quantity_unit_name,
    sol_u.abbreviation AS ordered_quantity_unit_abbreviation
FROM pick_line pl
JOIN quantity q ON q.id = pl.quantity_id
JOIN unit u ON u.id = q.unit_id
JOIN sales_order_line sol ON sol.id = pl.sales_order_line_id
JOIN quantity sol_q ON sol_q.id = sol.quantity_id
JOIN unit sol_u ON sol_u.id = sol_q.unit_id
WHERE pl.pick_id = sqlc.arg('pick_id')
AND q.value > 0
AND pl.packed_at IS NULL;

-- name: PackPickLines :exec
UPDATE pick_line SET
    packed_at = NOW(3),
    updated_at = NOW(3)
WHERE pick_id = sqlc.arg('pick_id')
AND packed_at IS NULL
AND quantity_id IN (
    SELECT q.id FROM quantity q WHERE q.value > 0
);

-- name: MarkPickFinishedIfAllPacked :exec
-- Finish a pick only when every one of its lines is packed. An unpacked line is
-- outstanding work regardless of its picked quantity: a partial pack leaves a
-- zero-quantity remainder pick line for the not-yet-picked balance, and that line
-- must keep the pick open (and editable) until it too is picked and packed. The
-- earlier `q.value > 0` guard ignored those remainder lines, so it finished picks
-- that still had quantity left to pick — freezing the remainder line in the UI.
UPDATE pick pk SET
    pk.finished_at = NOW(3),
    pk.updated_at = NOW(3)
WHERE pk.id = sqlc.arg('pick_id')
AND NOT EXISTS (
    SELECT 1 FROM pick_line pl
    WHERE pl.pick_id = pk.id
    AND pl.packed_at IS NULL
);

-- name: CloseOpenPickLines :exec
-- Pack every still-open (unpacked) pick line — including zero-quantity remainder lines.
-- Used when an order is closed (fulfilled): the pick's open lines are marked done so the
-- pick reads as complete alongside the order.
UPDATE pick_line SET
    packed_at = NOW(3),
    updated_at = NOW(3)
WHERE pick_id = sqlc.arg('pick_id')
AND packed_at IS NULL;

-- name: ReopenIncompletePickLines :exec
-- Reopen (unpack) pick lines that are not complete — the pick line's picked quantity is
-- less than its order line's ordered quantity. Fully-picked lines stay packed. Used when a
-- fulfilled order is reopened so outstanding lines can be worked again.
UPDATE pick_line pl
JOIN quantity q ON q.id = pl.quantity_id
JOIN sales_order_line sol ON sol.id = pl.sales_order_line_id
JOIN quantity sol_q ON sol_q.id = sol.quantity_id
SET pl.packed_at = NULL,
    pl.updated_at = NOW(3)
WHERE pl.pick_id = sqlc.arg('pick_id')
AND q.value < sol_q.value;

-- name: CountShipmentsByOrder :one
SELECT COUNT(*) AS total FROM shipment
WHERE sales_order_id = sqlc.arg('sales_order_id');

-- name: GetSalesOrderForPick :one
SELECT
    so.id,
    so.number,
    so.carrier_id,
    so.carrier_option_id,
    so.shipping_address_id
FROM sales_order so
WHERE so.id = (
    SELECT pk.sales_order_id FROM pick pk
    WHERE pk.id = sqlc.arg('pick_id')
    AND pk.account_id = sqlc.arg('account_id')
);

-- name: CreateShipment :exec
INSERT INTO shipment (
    id, number, sales_order_id, carrier_id, carrier_option_id,
    shipping_address_id, shipment_status_code, account_id,
    created_at, updated_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('number'), sqlc.arg('sales_order_id'),
    sqlc.narg('carrier_id'), sqlc.narg('carrier_option_id'),
    sqlc.narg('shipping_address_id'), sqlc.arg('shipment_status_code'),
    sqlc.arg('account_id'), NOW(3), NOW(3)
);

-- name: CreateShipmentLineFromPick :exec
INSERT INTO shipment_line (
    id, shipment_id, sales_order_line_id, quantity_id,
    created_at, updated_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('shipment_id'), sqlc.arg('sales_order_line_id'),
    sqlc.arg('quantity_id'), NOW(3), NOW(3)
);

-- name: CreateShippingCase :exec
INSERT INTO shipping_case (
    id, number, freight_amount_id, freight_weight_id,
    shipment_id, carrier_id, account_id,
    created_at, updated_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('number'), sqlc.arg('freight_amount_id'),
    sqlc.arg('freight_weight_id'), sqlc.arg('shipment_id'),
    sqlc.narg('carrier_id'), sqlc.arg('account_id'), NOW(3), NOW(3)
);

-- name: CreateQuantity :exec
INSERT INTO quantity (id, value, unit_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('value'), sqlc.arg('unit_id'), NOW(3), NOW(3));

-- name: GetPickLine :one
SELECT
    pl.id,
    pl.pick_id,
    pl.sales_order_line_id,
    pl.packed_at,
    pl.created_at,
    pl.updated_at,
    -- Pick line quantity
    q.id AS quantity_id,
    q.value AS quantity_value,
    u.id AS quantity_unit_id,
    u.name AS quantity_unit_name,
    u.abbreviation AS quantity_unit_abbreviation,
    -- Sales order line info
    sol.line_item_number,
    sol.product_sku,
    sol.product_description,
    sol.product_id,
    sol.item_id AS order_line_item_id,
    -- Ordered quantity
    sol_q.id AS ordered_quantity_id,
    sol_q.value AS ordered_quantity_value,
    sol_u.id AS ordered_quantity_unit_id,
    sol_u.name AS ordered_quantity_unit_name,
    sol_u.abbreviation AS ordered_quantity_unit_abbreviation
FROM pick_line pl
JOIN quantity q ON q.id = pl.quantity_id
JOIN unit u ON u.id = q.unit_id
JOIN sales_order_line sol ON sol.id = pl.sales_order_line_id
JOIN quantity sol_q ON sol_q.id = sol.quantity_id
JOIN unit sol_u ON sol_u.id = sol_q.unit_id
WHERE pl.id = sqlc.arg('pick_line_id');

-- name: IsPickLineInPick :one
SELECT EXISTS(
    SELECT 1 FROM pick_line
    WHERE id = sqlc.arg('pick_line_id')
    AND pick_id = sqlc.arg('pick_id')
) AS is_in_pick;

-- name: CreatePickLine :exec
INSERT INTO pick_line (id, pick_id, quantity_id, sales_order_line_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('pick_id'), sqlc.arg('quantity_id'), sqlc.arg('sales_order_line_id'), NOW(3), NOW(3));

-- name: CalculateRemainingForOrderLine :one
SELECT
    GREATEST(0, sol_q.value - COALESCE(SUM(pl_q.value), 0)) AS remaining_value,
    sol_q.unit_id
FROM sales_order_line sol
JOIN quantity sol_q ON sol_q.id = sol.quantity_id
LEFT JOIN pick_line pl ON pl.sales_order_line_id = sol.id
LEFT JOIN quantity pl_q ON pl_q.id = pl.quantity_id
WHERE sol.id = sqlc.arg('sales_order_line_id')
GROUP BY sol.id, sol_q.value, sol_q.unit_id;

-- name: HasUnpackedPickLineForOrderLine :one
SELECT EXISTS(
    SELECT 1 FROM pick_line
    WHERE sales_order_line_id = sqlc.arg('sales_order_line_id')
    AND packed_at IS NULL
) AS has_unpacked;

-- name: GetOrderLinePackProgress :one
-- The ordered quantity and the total already packed (shipped-committed) for an
-- order line. outstanding = ordered - packed drives pick reconciliation on a
-- quantity change: > 0 means an open pick line is still needed; <= 0 means the
-- packed lines already cover the order, so any open pick line is surplus.
SELECT
    sol_q.value AS ordered_value,
    sol_q.unit_id AS unit_id,
    COALESCE(SUM(CASE WHEN pl.packed_at IS NOT NULL THEN pl_q.value ELSE 0 END), 0) AS packed_value
FROM sales_order_line sol
JOIN quantity sol_q ON sol_q.id = sol.quantity_id
LEFT JOIN pick_line pl ON pl.sales_order_line_id = sol.id
LEFT JOIN quantity pl_q ON pl_q.id = pl.quantity_id
WHERE sol.id = sqlc.arg('sales_order_line_id')
GROUP BY sol.id, sol_q.value, sol_q.unit_id;

-- name: DeleteQuantitiesByUnpackedPickLinesForLine :exec
DELETE q FROM quantity q
JOIN pick_line pl ON pl.quantity_id = q.id
WHERE pl.sales_order_line_id = sqlc.arg('sales_order_line_id')
AND pl.packed_at IS NULL;

-- name: DeleteUnpackedPickLinesForLine :exec
DELETE FROM pick_line
WHERE sales_order_line_id = sqlc.arg('sales_order_line_id')
AND packed_at IS NULL;

-- name: CountPickLinesByPick :one
SELECT COUNT(*) AS total FROM pick_line WHERE pick_id = sqlc.arg('pick_id');

-- name: ListShippedOrderLineQuantitiesByShipment :many
-- The order line and shipped quantity behind each of a shipment's lines. Deleting the
-- shipment hands these back to the pick, so the reopened pick line can be matched to
-- the quantity this shipment took.
SELECT sl.sales_order_line_id, q.value AS shipped_value
FROM shipment_line sl
JOIN quantity q ON q.id = sl.quantity_id
WHERE sl.shipment_id = sqlc.arg('shipment_id');

-- name: ListPickLinesForOrderLine :many
SELECT pl.id, pl.quantity_id, pl.packed_at, q.value AS quantity_value
FROM pick_line pl
JOIN quantity q ON q.id = pl.quantity_id
WHERE pl.sales_order_line_id = sqlc.arg('sales_order_line_id')
ORDER BY pl.created_at, pl.id;

-- name: ReopenPickLine :exec
-- Reopens a packed pick line so the goods it committed become pickable again. The caller
-- restores its quantity separately (UpdatePickLineQuantity).
UPDATE pick_line SET
    packed_at = NULL,
    updated_at = NOW(3)
WHERE id = sqlc.arg('pick_line_id');

-- name: DeleteQuantitiesByPickLineIDs :exec
DELETE q FROM quantity q
JOIN pick_line pl ON pl.quantity_id = q.id
WHERE pl.id IN (sqlc.slice('ids'));

-- name: DeletePickLinesByIDs :exec
DELETE FROM pick_line WHERE id IN (sqlc.slice('ids'));

-- name: FindPickIDByShipmentOrder :one
SELECT pk.id FROM pick pk
WHERE pk.account_id = sqlc.arg('account_id')
AND pk.sales_order_id = (
    SELECT s.sales_order_id FROM shipment s
    WHERE s.id = sqlc.arg('shipment_id')
)
LIMIT 1;

-- name: GetShipmentIDsByPick :many
-- Shipments raised against the pick's sales order, oldest first. Backs related.shipments.
SELECT s.id
FROM shipment s
JOIN pick pk ON pk.sales_order_id = s.sales_order_id
WHERE pk.id = sqlc.arg('pick_id')
AND pk.account_id = sqlc.arg('account_id')
ORDER BY s.created_at, s.id;
