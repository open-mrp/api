-- name: ListProductLinesForward :many
SELECT
    pl.id,
    pl.name,
    pl.description,
    pl.notes,
    pl.is_commission_exempt,
    pl.is_freight_exempt,
    pl.unit_group_id,
    pl.fulfillment_policy_code,
    dlq.id AS default_lot_id,
    dlq.value AS default_lot_value,
    dlq.unit_id AS default_lot_unit_id,
    pl.account_id,
    pl.created_at,
    pl.updated_at
FROM product_line pl
LEFT JOIN quantity dlq ON dlq.id = pl.default_lot_id
WHERE (pl.account_id = sqlc.arg('account_id') OR pl.account_id IS NULL)
-- Residual: free-text search is a substring match on the name, so it cannot use the
-- FULLTEXT index (which does word matching). The set is already bounded to the tenant's
-- (plus global) product lines, which number in the dozens.
AND (
    sqlc.narg('search_query') IS NULL
    OR pl.name LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR pl.created_at < sqlc.narg('cursor_created_at')
    OR (pl.created_at = sqlc.narg('cursor_created_at') AND pl.id < sqlc.narg('cursor_id'))
)
ORDER BY pl.created_at DESC, pl.id DESC
LIMIT ?;

-- name: ListProductLinesBackward :many
SELECT
    pl.id,
    pl.name,
    pl.description,
    pl.notes,
    pl.is_commission_exempt,
    pl.is_freight_exempt,
    pl.unit_group_id,
    pl.fulfillment_policy_code,
    dlq.id AS default_lot_id,
    dlq.value AS default_lot_value,
    dlq.unit_id AS default_lot_unit_id,
    pl.account_id,
    pl.created_at,
    pl.updated_at
FROM product_line pl
LEFT JOIN quantity dlq ON dlq.id = pl.default_lot_id
WHERE (pl.account_id = sqlc.arg('account_id') OR pl.account_id IS NULL)
-- Residual: free-text search is a substring match on the name, so it cannot use the
-- FULLTEXT index (which does word matching). The set is already bounded to the tenant's
-- (plus global) product lines, which number in the dozens.
AND (
    sqlc.narg('search_query') IS NULL
    OR pl.name LIKE sqlc.narg('search_query')
)
AND (
    pl.created_at > sqlc.arg('cursor_created_at')
    OR (pl.created_at = sqlc.arg('cursor_created_at') AND pl.id > sqlc.arg('cursor_id'))
)
ORDER BY pl.created_at ASC, pl.id ASC
LIMIT ?;

-- name: GetProductLinesByIDs :many
SELECT
    pl.id,
    pl.name,
    pl.description,
    pl.notes,
    pl.is_commission_exempt,
    pl.is_freight_exempt,
    pl.unit_group_id,
    pl.fulfillment_policy_code,
    dlq.id AS default_lot_id,
    dlq.value AS default_lot_value,
    dlq.unit_id AS default_lot_unit_id,
    pl.account_id,
    pl.created_at,
    pl.updated_at
FROM product_line pl
LEFT JOIN quantity dlq ON dlq.id = pl.default_lot_id
WHERE pl.id IN (sqlc.slice('ids'));

-- name: GetProductLinesByIDsScoped :many
SELECT
    pl.id,
    pl.name,
    pl.description,
    pl.notes,
    pl.is_commission_exempt,
    pl.is_freight_exempt,
    pl.unit_group_id,
    pl.fulfillment_policy_code,
    dlq.id AS default_lot_id,
    dlq.value AS default_lot_value,
    dlq.unit_id AS default_lot_unit_id,
    pl.account_id,
    pl.created_at,
    pl.updated_at
FROM product_line pl
LEFT JOIN quantity dlq ON dlq.id = pl.default_lot_id
WHERE pl.id IN (sqlc.slice('ids'))
AND (pl.account_id = sqlc.arg('account_id') OR pl.account_id IS NULL);

-- name: GetProductLine :one
SELECT
    pl.id,
    pl.name,
    pl.description,
    pl.notes,
    pl.is_commission_exempt,
    pl.is_freight_exempt,
    pl.unit_group_id,
    pl.fulfillment_policy_code,
    dlq.id AS default_lot_id,
    dlq.value AS default_lot_value,
    dlq.unit_id AS default_lot_unit_id,
    pl.account_id,
    pl.created_at,
    pl.updated_at
FROM product_line pl
LEFT JOIN quantity dlq ON dlq.id = pl.default_lot_id
WHERE pl.id = sqlc.arg('id')
AND (pl.account_id = sqlc.arg('account_id') OR pl.account_id IS NULL);

-- name: InsertProductLine :exec
INSERT INTO product_line (
    id,
    name,
    is_commission_exempt,
    is_freight_exempt,
    unit_group_id,
    default_lot_id,
    fulfillment_policy_code,
    account_id,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('name'),
    sqlc.arg('is_commission_exempt'),
    sqlc.arg('is_freight_exempt'),
    sqlc.arg('unit_group_id'),
    sqlc.narg('default_lot_id'),
    sqlc.narg('fulfillment_policy_code'),
    sqlc.arg('account_id'),
    NOW(3),
    NOW(3)
);

-- name: UpdateProductLine :execresult
UPDATE product_line SET
    name = COALESCE(sqlc.narg('name'), name),
    is_commission_exempt = COALESCE(sqlc.narg('is_commission_exempt'), is_commission_exempt),
    is_freight_exempt = COALESCE(sqlc.narg('is_freight_exempt'), is_freight_exempt),
    unit_group_id = COALESCE(sqlc.narg('unit_group_id'), unit_group_id),
    -- Clearable rather than COALESCE-merged: removing a line's lot convention is a real edit, and a merge would make it unexpressible.
    default_lot_id = IF(sqlc.arg('clear_default_lot'), NULL, COALESCE(sqlc.narg('default_lot_id'), default_lot_id)),
    -- Clearable for the same reason: returning a line to the account default is a real edit.
    fulfillment_policy_code = IF(sqlc.arg('clear_fulfillment_policy'), NULL, COALESCE(sqlc.narg('fulfillment_policy_code'), fulfillment_policy_code)),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: DeleteProductLine :execresult
DELETE FROM product_line
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: CountProductLinesByName :one
SELECT COUNT(*) FROM product_line
WHERE name = ? AND (account_id = ? OR account_id IS NULL)
AND (sqlc.narg('exclude_id') IS NULL OR id != sqlc.narg('exclude_id'));

-- name: FindProductLinesByNames :many
-- names must be pre-lowercased by the caller; the utf8mb4_unicode_ci collation makes the
-- IN comparison case-insensitive, so lowercasing on both sides is not required in SQL.
-- System product lines (account_id IS NULL) are included so the upsert can detect a
-- name collision with a platform-provided line and reject the modification.
SELECT
    pl.id,
    pl.name,
    pl.description,
    pl.notes,
    pl.is_commission_exempt,
    pl.is_freight_exempt,
    pl.unit_group_id,
    pl.account_id,
    pl.created_at,
    pl.updated_at
FROM product_line pl
WHERE pl.name IN (sqlc.slice('names'))
AND (pl.account_id = sqlc.arg('account_id') OR pl.account_id IS NULL);

-- name: GetUnitGroupForProductLine :one
SELECT
    ug.id,
    ug.name,
    ug.base_unit_id,
    ug.unit_type_code,
    ug.created_at,
    ug.updated_at
FROM unit_group ug
WHERE ug.id = sqlc.arg('id')
AND (ug.account_id = sqlc.arg('account_id') OR ug.account_id IS NULL);

-- name: ExportProductLines :many
-- Unpaginated by design; the caller passes a row cap as the limit. System rows
-- (account_id IS NULL) are in scope, matching what the list endpoint returns.
SELECT
    pl.id,
    pl.name,
    pl.is_commission_exempt,
    pl.is_freight_exempt,
    ug.name AS unit_group_name,
    pl.created_at,
    pl.updated_at
FROM product_line pl
LEFT JOIN unit_group ug ON ug.id = pl.unit_group_id
WHERE (pl.account_id = sqlc.arg('account_id') OR pl.account_id IS NULL)
AND (
    sqlc.narg('search_query') IS NULL
    OR pl.name LIKE sqlc.narg('search_query')
)
ORDER BY pl.created_at DESC, pl.id DESC
LIMIT ?;


-- ListProductLineLotDefaults returns every product line in the account that has a lot convention, for resolving what lot an item is made in.
-- name: ListProductLineLotDefaults :many
SELECT
    pl.id,
    pl.name,
    dlq.value AS default_lot_value,
    dlq.unit_id AS default_lot_unit_id
FROM product_line pl
JOIN quantity dlq ON dlq.id = pl.default_lot_id
WHERE (pl.account_id = sqlc.arg('account_id') OR pl.account_id IS NULL);

-- ListProductLineFulfillmentPolicies returns every product line in the account that sets a fulfillment policy.
--
-- Separate from ListProductLineLotDefaults, which inner-joins the lot quantity: a line can set a policy without setting a lot convention, and sharing that query would silently drop its policy.
-- name: ListProductLineFulfillmentPolicies :many
SELECT
    pl.id,
    pl.fulfillment_policy_code
FROM product_line pl
WHERE (pl.account_id = sqlc.arg('account_id') OR pl.account_id IS NULL)
  AND pl.fulfillment_policy_code IS NOT NULL
ORDER BY pl.id;

-- GetProductLineForItem resolves the product line an item sells under, which is where its lot convention comes from. Intermediate items — greige — have no product row and so return nothing; those inherit from what they become.
-- name: GetProductLineForItem :one
SELECT
    pl.id,
    dlq.value AS default_lot_value,
    dlq.unit_id AS default_lot_unit_id
FROM item i
JOIN product p ON p.item_id = i.id
JOIN product_line pl ON pl.id = p.product_line_id
JOIN quantity dlq ON dlq.id = pl.default_lot_id
WHERE i.id = sqlc.arg('item_id')
AND i.account_id = sqlc.arg('account_id');

-- ListItemProductLines maps items to the product line they sell under, in one query.
-- name: ListItemProductLines :many
SELECT
    i.id AS item_id,
    p.product_line_id
FROM item i
JOIN product p ON p.item_id = i.id
WHERE i.account_id = sqlc.arg('account_id')
AND i.id IN (sqlc.slice('item_ids'))
AND p.product_line_id IS NOT NULL;

-- ResolveItemLotFromDownstream finds the lot convention an intermediate item inherits from what it becomes.
--
-- The greige-to-finished decomposition is already recorded per schedule version, so this reads what the plan decided rather than re-walking batch genealogy. The most recently generated version wins, and among its finished goods the one with the highest weekly demand — the same rule the solver applies when a greige feeds several lines.
-- name: ResolveItemLotFromDownstream :many
SELECT
    pl.id AS product_line_id,
    dlq.value AS default_lot_value,
    dlq.unit_id AS default_lot_unit_id,
    SUM(f.weekly_demand) AS weekly_demand
FROM production_schedule_finished_policy f
JOIN production_schedule s ON s.id = f.production_schedule_id
JOIN product p ON p.item_id = f.item_id
JOIN product_line pl ON pl.id = p.product_line_id
JOIN quantity dlq ON dlq.id = pl.default_lot_id
WHERE f.account_id = sqlc.arg('account_id')
AND f.greige_item_id = sqlc.arg('item_id')
AND s.id = (
    SELECT s2.id FROM production_schedule s2
    WHERE s2.account_id = sqlc.arg('account_id')
    ORDER BY s2.created_at DESC, s2.id DESC
    LIMIT 1
)
GROUP BY pl.id, dlq.value, dlq.unit_id
ORDER BY weekly_demand DESC, pl.id ASC;

-- GetItemLotOverride reads a per-item lot size set by hand, which outranks any line convention.
-- name: GetItemLotOverride :one
SELECT COALESCE(s.lot_multiple_units, 0) AS lot_multiple_units
FROM production_schedule_item_setting s
WHERE s.account_id = sqlc.arg('account_id')
AND s.item_id = sqlc.arg('item_id');

-- The lot is a quantity row like a customer's credit limit, so it has its own lifecycle: inserted when a line first gets a convention, updated in place after that, and deleted when the convention is removed.

-- name: InsertProductLineDefaultLotQuantity :exec
INSERT INTO quantity (id, value, unit_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('value'), sqlc.arg('unit_id'), NOW(3), NOW(3));

-- name: UpdateProductLineDefaultLotQuantity :exec
UPDATE quantity SET value = sqlc.arg('value'), unit_id = sqlc.arg('unit_id'), updated_at = NOW(3)
WHERE id = sqlc.arg('id');

-- name: DeleteProductLineDefaultLotQuantity :exec
DELETE FROM quantity WHERE id = sqlc.arg('id');

-- GetProductLineDefaultLotID reads the quantity a line's lot currently points at, so an edit can update that row rather than orphaning it and inserting another.
-- name: GetProductLineDefaultLotID :one
SELECT pl.default_lot_id FROM product_line pl
WHERE pl.id = sqlc.arg('id')
AND pl.account_id = sqlc.arg('account_id');

-- ListDownstreamItemsForItems is one step of "what do these items become", read from the production flow rather than from batch history.
--
-- The flow is configuration: it answers the question for an item that has never been made, which is exactly the item a planner is adding the first batch of. Walking it a level at a time keeps the whole traversal to one query per depth no matter how wide the frontier is.
-- name: ListDownstreamItemsForItems :many
SELECT DISTINCT p.item_id
FROM consumption c
JOIN production_step ps ON ps.id = c.production_step_id
JOIN production p ON p.production_step_id = ps.id
JOIN item i ON i.id = p.item_id
WHERE ps.account_id = sqlc.arg('account_id')
AND c.item_id IN (sqlc.slice('item_ids'))
AND i.deleted_at IS NULL
ORDER BY p.item_id;

-- ListProductLineLotsForItems reads the lot convention of whatever line each of these items sells under.
--
-- The batch form of GetProductLineForItem, for resolving a whole frontier of downstream items at once. Ordered by line so a tie between two lines resolves the same way on every call — an unstable lot would make the same item batch differently from one page load to the next.
-- name: ListProductLineLotsForItems :many
SELECT
    i.id AS item_id,
    pl.id AS product_line_id,
    dlq.value AS default_lot_value,
    dlq.unit_id AS default_lot_unit_id
FROM item i
JOIN product p ON p.item_id = i.id
JOIN product_line pl ON pl.id = p.product_line_id
JOIN quantity dlq ON dlq.id = pl.default_lot_id
WHERE i.account_id = sqlc.arg('account_id')
AND i.id IN (sqlc.slice('item_ids'))
ORDER BY pl.id, i.id;
