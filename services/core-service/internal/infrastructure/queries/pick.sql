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
    p.updated_at
FROM pick p
JOIN sales_order so ON so.id = p.sales_order_id
JOIN account_relation ar ON ar.owner_account_id = so.owner_account_id
    AND ar.counterparty_account_id = so.buyer_account_id
JOIN account ba ON ba.id = so.buyer_account_id
JOIN priority pr ON pr.code = so.priority_code
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
    sqlc.arg('include_department_filter') = false
    OR EXISTS (
        SELECT 1 FROM _departments_picks dp
        WHERE dp.B = p.id
        AND dp.A IN (sqlc.slice('department_ids'))
    )
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
    sqlc.narg('cursor_created_at') IS NULL
    OR p.created_at < sqlc.narg('cursor_created_at')
    OR (p.created_at = sqlc.narg('cursor_created_at') AND p.id < sqlc.narg('cursor_id'))
)
ORDER BY p.created_at DESC, p.id DESC
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
    p.updated_at
FROM pick p
JOIN sales_order so ON so.id = p.sales_order_id
JOIN account_relation ar ON ar.owner_account_id = so.owner_account_id
    AND ar.counterparty_account_id = so.buyer_account_id
JOIN account ba ON ba.id = so.buyer_account_id
JOIN priority pr ON pr.code = so.priority_code
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
    sqlc.arg('include_department_filter') = false
    OR EXISTS (
        SELECT 1 FROM _departments_picks dp
        WHERE dp.B = p.id
        AND dp.A IN (sqlc.slice('department_ids'))
    )
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
    p.created_at > sqlc.arg('cursor_created_at')
    OR (p.created_at = sqlc.arg('cursor_created_at') AND p.id > sqlc.arg('cursor_id'))
)
ORDER BY p.created_at ASC, p.id ASC
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
    sqlc.arg('include_department_filter') = false
    OR EXISTS (
        SELECT 1 FROM _departments_picks dp
        WHERE dp.B = p.id
        AND dp.A IN (sqlc.slice('department_ids'))
    )
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
    p.updated_at
FROM pick p
JOIN sales_order so ON so.id = p.sales_order_id
JOIN account_relation ar ON ar.owner_account_id = so.owner_account_id
    AND ar.counterparty_account_id = so.buyer_account_id
JOIN account ba ON ba.id = so.buyer_account_id
JOIN priority pr ON pr.code = so.priority_code
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

-- name: GetPickDepartments :many
SELECT
    d.id,
    d.name
FROM _departments_picks dp
JOIN department d ON d.id = dp.A
WHERE dp.B = sqlc.arg('pick_id');

-- name: UpdatePickNumber :exec
UPDATE pick SET
    number = sqlc.arg('number'),
    updated_at = NOW(3)
WHERE id = sqlc.arg('pick_id')
AND account_id = sqlc.arg('account_id');

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
UPDATE quantity SET
    value = 0,
    updated_at = NOW(3)
WHERE id IN (
    SELECT quantity_id FROM pick_line
    WHERE pick_id = sqlc.arg('pick_id')
    AND packed_at IS NULL
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
SET q.value = GREATEST(0, sol_q.value - GREATEST(COALESCE(picked.total_picked_value, 0) - q.value, 0)),
    q.updated_at = NOW(3)
WHERE pl.pick_id = sqlc.arg('pick_id')
AND pl.packed_at IS NULL;

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
    GROUP BY pl_sum.sales_order_line_id
) picked ON picked.sales_order_line_id = pl.sales_order_line_id
SET q.value = GREATEST(0, sol_q.value - COALESCE(picked.total_picked_value, 0)),
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
    value = sqlc.arg('value'),
    updated_at = NOW(3)
WHERE id = (
    SELECT pl.quantity_id FROM pick_line pl
    WHERE pl.id = sqlc.arg('pick_line_id')
);

-- name: GetPickShipmentNumbers :many
SELECT s.number FROM shipment s
WHERE s.account_id = sqlc.arg('account_id')
AND s.sales_order_id = (
    SELECT pk.sales_order_id FROM pick pk
    WHERE pk.id = sqlc.arg('pick_id')
    AND pk.account_id = sqlc.arg('account_id')
)
AND (
    sqlc.narg('search_query') IS NULL
    OR s.number LIKE sqlc.narg('search_query')
)
ORDER BY s.created_at ASC
LIMIT ? OFFSET ?;

-- name: CountPickShipmentNumbers :one
SELECT COUNT(*) AS count FROM shipment s
WHERE s.account_id = sqlc.arg('account_id')
AND s.sales_order_id = (
    SELECT pk.sales_order_id FROM pick pk
    WHERE pk.id = sqlc.arg('pick_id')
    AND pk.account_id = sqlc.arg('account_id')
)
AND (
    sqlc.narg('search_query') IS NULL
    OR s.number LIKE sqlc.narg('search_query')
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

-- name: UnpackPickLinesByShipment :exec
UPDATE pick_line SET
    packed_at = NULL,
    updated_at = NOW(3)
WHERE sales_order_line_id IN (
    SELECT sl.sales_order_line_id FROM shipment_line sl
    WHERE sl.shipment_id = sqlc.arg('shipment_id')
);

-- name: FindPickIDByShipmentOrder :one
SELECT pk.id FROM pick pk
WHERE pk.account_id = sqlc.arg('account_id')
AND pk.sales_order_id = (
    SELECT s.sales_order_id FROM shipment s
    WHERE s.id = sqlc.arg('shipment_id')
)
LIMIT 1;
