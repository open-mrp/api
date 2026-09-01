-- name: CreateReceivingOrder :exec
INSERT INTO receiving_order (id, number, order_id, account_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('number'), sqlc.arg('order_id'), sqlc.arg('account_id'), NOW(3), NOW(3));

-- name: CreateReceivingOrderLine :exec
INSERT INTO receiving_order_line (id, receiving_order_id, quantity_id, sales_order_line_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('receiving_order_id'), sqlc.arg('quantity_id'), sqlc.arg('sales_order_line_id'), NOW(3), NOW(3));

-- name: GetReceivingOrderByOrderID :one
SELECT ro.id
FROM receiving_order ro
WHERE ro.order_id = sqlc.arg('order_id');

-- name: DeleteReceivingOrderLinesByOrderID :exec
DELETE rol FROM receiving_order_line rol
JOIN receiving_order ro ON ro.id = rol.receiving_order_id
WHERE ro.order_id = sqlc.arg('order_id');

-- name: DeleteReceivingOrderByOrderID :exec
DELETE FROM receiving_order WHERE order_id = sqlc.arg('order_id');

-- name: MarkReceivingOrderComplete :exec
UPDATE receiving_order SET
    completed_at = NOW(3),
    updated_at = NOW(3)
WHERE order_id = sqlc.arg('order_id');

-- name: MarkReceivingOrderIncomplete :exec
UPDATE receiving_order SET
    completed_at = NULL,
    updated_at = NOW(3)
WHERE order_id = sqlc.arg('order_id');

-- name: DeleteReceivingOrderLinesByOrderLineID :exec
DELETE FROM receiving_order_line WHERE sales_order_line_id = sqlc.arg('sales_order_line_id');

-- name: ListReceivingOrdersForward :many
SELECT
    ro.id,
    ro.number,
    ro.completed_at,
    ro.created_at,
    ro.updated_at,
    so.id AS purchase_order_id,
    so.number AS purchase_order_number,
    a.id AS supplier_id,
    a.name AS supplier_name,
    ar.external_number AS supplier_number,
    COUNT(rol.id) AS line_count,
    CASE
        WHEN COUNT(rol.id) = 0 THEN 0
        ELSE ROUND(COUNT(CASE WHEN rol.stocked_at IS NOT NULL THEN 1 END) * 100.0 / COUNT(rol.id), 2)
    END AS completion_percentage
FROM receiving_order ro
JOIN sales_order so ON ro.order_id = so.id
LEFT JOIN account_relation ar ON so.seller_account_id = ar.counterparty_account_id AND ar.owner_account_id = ro.account_id
LEFT JOIN account a ON ar.counterparty_account_id = a.id
LEFT JOIN receiving_order_line rol ON rol.receiving_order_id = ro.id
WHERE ro.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR ro.number LIKE sqlc.narg('search_query')
    OR so.number LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('status') IS NULL
    OR (sqlc.narg('status') = 'open' AND ro.completed_at IS NULL)
    OR (sqlc.narg('status') = 'completed' AND ro.completed_at IS NOT NULL)
)
AND (
    sqlc.arg('include_item_filter') = false
    OR EXISTS (
        SELECT 1 FROM receiving_order_line rol2
        JOIN sales_order_line sol ON rol2.sales_order_line_id = sol.id
        WHERE rol2.receiving_order_id = ro.id
        AND sol.item_id IN (sqlc.slice('item_ids'))
    )
)
AND (
    sqlc.arg('include_supplier_filter') = false
    OR so.seller_account_id IN (sqlc.slice('supplier_ids'))
)
AND (
    sqlc.narg('start_date') IS NULL
    OR ro.created_at >= sqlc.narg('start_date')
)
AND (
    sqlc.narg('end_date') IS NULL
    OR ro.created_at <= sqlc.narg('end_date')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR ro.created_at < sqlc.narg('cursor_created_at')
    OR (ro.created_at = sqlc.narg('cursor_created_at') AND ro.id < sqlc.narg('cursor_id'))
)
GROUP BY ro.id, ro.number, ro.completed_at, ro.created_at, ro.updated_at, so.id, so.number, a.id, a.name, ar.external_number
ORDER BY ro.created_at DESC, ro.id DESC
LIMIT ?;

-- name: ListReceivingOrdersBackward :many
SELECT
    ro.id,
    ro.number,
    ro.completed_at,
    ro.created_at,
    ro.updated_at,
    so.id AS purchase_order_id,
    so.number AS purchase_order_number,
    a.id AS supplier_id,
    a.name AS supplier_name,
    ar.external_number AS supplier_number,
    COUNT(rol.id) AS line_count,
    CASE
        WHEN COUNT(rol.id) = 0 THEN 0
        ELSE ROUND(COUNT(CASE WHEN rol.stocked_at IS NOT NULL THEN 1 END) * 100.0 / COUNT(rol.id), 2)
    END AS completion_percentage
FROM receiving_order ro
JOIN sales_order so ON ro.order_id = so.id
LEFT JOIN account_relation ar ON so.seller_account_id = ar.counterparty_account_id AND ar.owner_account_id = ro.account_id
LEFT JOIN account a ON ar.counterparty_account_id = a.id
LEFT JOIN receiving_order_line rol ON rol.receiving_order_id = ro.id
WHERE ro.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR ro.number LIKE sqlc.narg('search_query')
    OR so.number LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('status') IS NULL
    OR (sqlc.narg('status') = 'open' AND ro.completed_at IS NULL)
    OR (sqlc.narg('status') = 'completed' AND ro.completed_at IS NOT NULL)
)
AND (
    sqlc.arg('include_item_filter') = false
    OR EXISTS (
        SELECT 1 FROM receiving_order_line rol2
        JOIN sales_order_line sol ON rol2.sales_order_line_id = sol.id
        WHERE rol2.receiving_order_id = ro.id
        AND sol.item_id IN (sqlc.slice('item_ids'))
    )
)
AND (
    sqlc.arg('include_supplier_filter') = false
    OR so.seller_account_id IN (sqlc.slice('supplier_ids'))
)
AND (
    sqlc.narg('start_date') IS NULL
    OR ro.created_at >= sqlc.narg('start_date')
)
AND (
    sqlc.narg('end_date') IS NULL
    OR ro.created_at <= sqlc.narg('end_date')
)
AND (
    ro.created_at > sqlc.arg('cursor_created_at')
    OR (ro.created_at = sqlc.arg('cursor_created_at') AND ro.id > sqlc.arg('cursor_id'))
)
GROUP BY ro.id, ro.number, ro.completed_at, ro.created_at, ro.updated_at, so.id, so.number, a.id, a.name, ar.external_number
ORDER BY ro.created_at ASC, ro.id ASC
LIMIT ?;

-- name: GetReceivingOrderByID :one
SELECT
    ro.id,
    ro.number,
    ro.completed_at,
    ro.created_at,
    ro.updated_at,
    so.id AS purchase_order_id,
    so.number AS purchase_order_number,
    a.id AS supplier_id,
    a.name AS supplier_name,
    ar.external_number AS supplier_number,
    so.note
FROM receiving_order ro
JOIN sales_order so ON ro.order_id = so.id
LEFT JOIN account_relation ar ON so.seller_account_id = ar.counterparty_account_id AND ar.owner_account_id = ro.account_id
LEFT JOIN account a ON ar.counterparty_account_id = a.id
WHERE ro.id = sqlc.arg('id')
AND ro.account_id = sqlc.arg('account_id');

-- name: ListReceivingOrderLinesByOrderID :many
SELECT
    rol.id,
    rol.stocked_at,
    rol.created_at,
    rol.updated_at,
    q.id AS quantity_id,
    q.value AS quantity_value,
    qu.id AS quantity_unit_id,
    qu.abbreviation AS quantity_unit_abbreviation,
    sol.id AS order_line_id,
    sol.item_id AS order_line_item_id,
    sol.product_id AS order_line_product_id,
    i.sku AS order_line_item_sku,
    i.description AS order_line_item_description,
    oq.value AS order_line_quantity_ordered,
    ou.id AS order_line_unit_id,
    ou.abbreviation AS order_line_unit_abbreviation,
    (SELECT CAST(SUM(rq.value) AS CHAR) FROM delivery_line dl JOIN quantity rq ON dl.quantity_id = rq.id WHERE dl.receiving_order_line_id = rol.id AND dl.rejected_at IS NOT NULL) AS rejected_quantity_value
FROM receiving_order_line rol
JOIN quantity q ON rol.quantity_id = q.id
JOIN unit qu ON q.unit_id = qu.id
JOIN sales_order_line sol ON rol.sales_order_line_id = sol.id
LEFT JOIN item i ON sol.item_id = i.id
JOIN quantity oq ON sol.quantity_id = oq.id
JOIN unit ou ON oq.unit_id = ou.id
WHERE rol.receiving_order_id = sqlc.arg('receiving_order_id')
ORDER BY rol.created_at ASC, rol.id ASC;

-- name: FindUnstockedLineIDs :many
SELECT
    rol.id,
    rol.sales_order_line_id AS order_line_id
FROM receiving_order_line rol
JOIN receiving_order ro ON rol.receiving_order_id = ro.id
LEFT JOIN quantity q ON rol.quantity_id = q.id
WHERE rol.receiving_order_id = sqlc.arg('receiving_order_id')
AND ro.account_id = sqlc.arg('account_id')
AND ro.completed_at IS NULL
AND rol.stocked_at IS NULL
AND (
    sqlc.arg('enforce_non_zero') = false
    OR (q.value IS NOT NULL AND CAST(q.value AS DECIMAL(20,6)) > 0)
);

-- name: StockReceivingOrderLines :exec
UPDATE receiving_order_line
SET stocked_at = NOW(3)
WHERE id IN (sqlc.slice('line_ids'));

-- name: MarkReceivingOrderCompleteByID :exec
UPDATE receiving_order
SET completed_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: MarkReceivingOrderIncompleteByID :exec
UPDATE receiving_order
SET completed_at = NULL
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: CheckAllLinesStocked :one
SELECT COUNT(*) AS unstocked_count
FROM receiving_order_line rol
JOIN receiving_order ro ON rol.receiving_order_id = ro.id
WHERE rol.receiving_order_id = sqlc.arg('receiving_order_id')
AND ro.account_id = sqlc.arg('account_id')
AND rol.stocked_at IS NULL;

-- name: VoidAllReceivingOrderLines :exec
UPDATE receiving_order_line rol
JOIN quantity q ON rol.quantity_id = q.id
JOIN receiving_order ro ON rol.receiving_order_id = ro.id
SET q.value = '0', rol.stocked_at = NULL
WHERE rol.receiving_order_id = sqlc.arg('receiving_order_id')
AND ro.account_id = sqlc.arg('account_id')
AND ro.completed_at IS NULL;

-- name: DeleteDuplicateReceivingOrderLines :exec
DELETE rol FROM receiving_order_line rol
JOIN receiving_order ro ON rol.receiving_order_id = ro.id
JOIN (
    SELECT
        sales_order_line_id,
        MIN(id) AS keep_id
    FROM receiving_order_line
    WHERE receiving_order_line.receiving_order_id = sqlc.arg('receiving_order_id')
    GROUP BY sales_order_line_id
    HAVING COUNT(*) > 1
) dup ON dup.sales_order_line_id = rol.sales_order_line_id
WHERE rol.receiving_order_id = sqlc.arg('receiving_order_id')
AND ro.account_id = sqlc.arg('account_id')
AND rol.id <> dup.keep_id;

-- name: UpdateReceivingOrderLineQuantity :exec
UPDATE quantity q
JOIN receiving_order_line rol ON q.id = rol.quantity_id
SET q.value = sqlc.arg('quantity_value')
WHERE rol.id = sqlc.arg('line_id');

-- name: VoidReceivingOrderLine :exec
UPDATE receiving_order_line rol
JOIN quantity q ON rol.quantity_id = q.id
JOIN receiving_order ro ON rol.receiving_order_id = ro.id
SET q.value = '0', rol.stocked_at = NULL
WHERE rol.id = sqlc.arg('line_id')
AND ro.account_id = sqlc.arg('account_id');

-- name: GetReceivingOrderLine :one
SELECT
    rol.id,
    rol.stocked_at,
    rol.created_at,
    rol.updated_at,
    q.id AS quantity_id,
    q.value AS quantity_value,
    qu.id AS quantity_unit_id,
    qu.abbreviation AS quantity_unit_abbreviation,
    sol.id AS order_line_id,
    sol.item_id AS order_line_item_id,
    sol.product_id AS order_line_product_id,
    i.sku AS order_line_item_sku,
    i.description AS order_line_item_description,
    oq.value AS order_line_quantity_ordered,
    ou.id AS order_line_unit_id,
    ou.abbreviation AS order_line_unit_abbreviation,
    (SELECT CAST(SUM(rq.value) AS CHAR) FROM delivery_line dl JOIN quantity rq ON dl.quantity_id = rq.id WHERE dl.receiving_order_line_id = rol.id AND dl.rejected_at IS NOT NULL) AS rejected_quantity_value
FROM receiving_order_line rol
JOIN quantity q ON rol.quantity_id = q.id
JOIN unit qu ON q.unit_id = qu.id
JOIN sales_order_line sol ON rol.sales_order_line_id = sol.id
LEFT JOIN item i ON sol.item_id = i.id
JOIN quantity oq ON sol.quantity_id = oq.id
JOIN unit ou ON oq.unit_id = ou.id
WHERE rol.id = sqlc.arg('line_id');

-- name: IsReceivingOrderLineInOrder :one
SELECT EXISTS(
    SELECT 1 FROM receiving_order_line
    WHERE id = sqlc.arg('line_id')
    AND receiving_order_id = sqlc.arg('receiving_order_id')
) AS line_exists;

-- name: IsReceivingOrderInAccount :one
SELECT EXISTS(
    SELECT 1 FROM receiving_order
    WHERE id = sqlc.arg('receiving_order_id')
    AND account_id = sqlc.arg('account_id')
) AS order_exists;

-- name: CalculateQuantityYetToBeReceived :one
SELECT
    oq.value AS ordered_value,
    COALESCE(SUM(CAST(rq.value AS DECIMAL(20,6))), 0) AS received_total,
    ou.id AS unit_id
FROM receiving_order_line rol
JOIN sales_order_line sol ON rol.sales_order_line_id = sol.id
JOIN quantity oq ON sol.quantity_id = oq.id
JOIN unit ou ON oq.unit_id = ou.id
LEFT JOIN receiving_order_line all_rol ON all_rol.sales_order_line_id = sol.id
LEFT JOIN quantity rq ON all_rol.quantity_id = rq.id
WHERE rol.id = sqlc.arg('line_id')
GROUP BY oq.value, ou.id;

-- name: GetOrderedQuantityForLine :many
SELECT
    sol.id AS order_line_id,
    oq.value AS ordered_value,
    COALESCE(SUM(CAST(rq.value AS DECIMAL(20,6))), 0) AS received_total,
    ou.id AS unit_id
FROM sales_order_line sol
JOIN quantity oq ON sol.quantity_id = oq.id
JOIN unit ou ON oq.unit_id = ou.id
LEFT JOIN receiving_order_line stocked_rol ON stocked_rol.sales_order_line_id = sol.id AND stocked_rol.stocked_at IS NOT NULL
LEFT JOIN quantity rq ON stocked_rol.quantity_id = rq.id
WHERE sol.id IN (sqlc.slice('order_line_ids'))
GROUP BY sol.id, oq.value, ou.id;

-- name: GetReceivingOrderLineUnitPrice :many
SELECT
    rol.id AS receiving_order_line_id,
    sol.item_id,
    r.value AS unit_price_value,
    r.numerator_unit_id AS unit_price_numerator_unit_id,
    r.denominator_unit_id AS unit_price_denominator_unit_id,
    qu.id AS quantity_unit_id
FROM receiving_order_line rol
JOIN sales_order_line sol ON rol.sales_order_line_id = sol.id
JOIN rate r ON sol.unit_price_id = r.id
JOIN quantity q ON rol.quantity_id = q.id
JOIN unit qu ON q.unit_id = qu.id
WHERE rol.receiving_order_id = sqlc.arg('receiving_order_id');

-- name: GetPurchaseOrderIDForReceivingOrder :one
SELECT ro.order_id AS purchase_order_id
FROM receiving_order ro
WHERE ro.id = sqlc.arg('receiving_order_id')
AND ro.account_id = sqlc.arg('account_id');

-- name: CountDeliveriesByPurchaseOrder :one
SELECT COUNT(*) AS delivery_count
FROM delivery
WHERE sales_order_id = sqlc.arg('purchase_order_id');

-- name: UpsertLot :exec
INSERT IGNORE INTO lot (id, account_id, item_id, lot_number, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('account_id'), sqlc.arg('item_id'), sqlc.arg('lot_number'), NOW(3), NOW(3));

-- name: GetLotByKey :one
SELECT id FROM lot
WHERE account_id = sqlc.arg('account_id')
AND item_id = sqlc.arg('item_id')
AND lot_number = sqlc.arg('lot_number');

-- name: InsertDelivery :exec
INSERT INTO delivery (id, number, sales_order_id, account_id, delivery_status_code, accepted_at, rejected_at, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('number'), sqlc.arg('sales_order_id'), sqlc.arg('account_id'), sqlc.arg('delivery_status_code'), sqlc.narg('accepted_at'), sqlc.narg('rejected_at'), NOW(3), NOW(3));

-- name: InsertDeliveryLine :exec
INSERT INTO delivery_line (id, delivery_id, receiving_order_line_id, quantity_id, unit_cost_id, storage_location_id, lot_id, accepted_at, rejected_at, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('delivery_id'), sqlc.arg('receiving_order_line_id'), sqlc.arg('quantity_id'), sqlc.arg('unit_cost_id'), sqlc.narg('storage_location_id'), sqlc.narg('lot_id'), sqlc.narg('accepted_at'), sqlc.narg('rejected_at'), NOW(3), NOW(3));

-- name: InsertInventoryReceiptForDelivery :exec
INSERT INTO inventory_receipt (
    id, owner_account_id, holder_account_id, item_id,
    quantity_id, unit_cost_id, status_code,
    storage_location_id, lot_id, order_id,
    received_at, created_at, updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('account_id'),
    sqlc.arg('account_id'),
    sqlc.arg('item_id'),
    sqlc.arg('quantity_id'),
    sqlc.arg('unit_cost_id'),
    'available',
    sqlc.narg('storage_location_id'),
    sqlc.narg('lot_id'),
    sqlc.narg('order_id'),
    NOW(3),
    NOW(3),
    NOW(3)
);


-- Each allocation is taken through its own unit's ratio before it is added: an issue in pounds can be
-- covered by allocations recorded in grams, and once they are added together the caller has nothing
-- left to convert with. Divide the total by a unit's ratio to read it in that unit.
-- name: GetAllocationSumForIssue :one
SELECT COALESCE(SUM(CAST(q.value AS DECIMAL(65,30)) * (u.ratio_numerator / u.ratio_denominator)), 0) AS total_allocated
FROM inventory_allocation ia
JOIN quantity q ON q.id = ia.quantity_id
JOIN unit u ON u.id = q.unit_id
WHERE ia.inventory_issue_id = sqlc.arg('issue_id');

-- name: HasUnstockedReceivingOrderLineForOrderLine :one
SELECT EXISTS(
    SELECT 1 FROM receiving_order_line
    WHERE sales_order_line_id = sqlc.arg('sales_order_line_id')
    AND stocked_at IS NULL
) AS has_unstocked;

-- name: MarkPurchaseOrderFulfilled :exec
UPDATE sales_order
SET sales_order_status_code = 'fulfilled',
    completed_at = NOW(3),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND buyer_account_id = sqlc.arg('account_id')
AND sales_order_type_code = 'purchase_order';
