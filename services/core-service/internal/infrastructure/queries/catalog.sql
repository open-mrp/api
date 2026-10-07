-- name: ListCatalogProductLines :many
-- Read from the account's lines and the few with no account (legacy rows), each kept when one of the
-- account's portal-ready products is on it; walking the account's products instead reads all of them.
SELECT pl.id, pl.name
FROM product_line pl FORCE INDEX (product_line_account_id_name_key)
WHERE (pl.account_id = sqlc.narg('line_account_id') OR pl.account_id IS NULL)
  AND EXISTS (
    SELECT 1
    FROM product p FORCE INDEX (product_line_portal_created_idx)
    JOIN item it ON it.id = p.item_id
    WHERE p.product_line_id = pl.id
      AND p.is_portal_ready = 1
      AND it.account_id = sqlc.arg('account_id')
      AND it.item_type_code = 'product'
      AND it.deleted_at IS NULL
  )
ORDER BY pl.name;

-- name: ListCatalogProductLinesForCustomer :many
-- Read as ListCatalogProductLines is, then each line checked against the customer's access.
SELECT pl.id, pl.name
FROM product_line pl FORCE INDEX (product_line_account_id_name_key)
WHERE (pl.account_id = sqlc.narg('line_account_id') OR pl.account_id IS NULL)
  AND EXISTS (
    SELECT 1
    FROM product p FORCE INDEX (product_line_portal_created_idx)
    JOIN item it ON it.id = p.item_id
    WHERE p.product_line_id = pl.id
      AND p.is_portal_ready = 1
      AND it.account_id = sqlc.arg('account_id')
      AND it.item_type_code = 'product'
      AND it.deleted_at IS NULL
  )
  AND (
    -- Pathway 1: product line via account group that the customer's account relation belongs to
    EXISTS (
      SELECT 1 FROM account_group_product_line agpl
      JOIN account_relation ar ON ar.account_group_id = agpl.account_group_id
      WHERE agpl.product_line_id = pl.id
        AND ar.owner_account_id = sqlc.arg('account_id')
        AND ar.counterparty_account_id = sqlc.arg('customer_account_id')
        AND ar.account_relation_role_code = 'customer'
    )
    -- Pathway 2: product line via direct account relation product line assignment
    OR EXISTS (
      SELECT 1 FROM account_relation_product_line arpl
      JOIN account_relation ar ON ar.id = arpl.account_relation_id
      WHERE arpl.product_line_id = pl.id
        AND ar.owner_account_id = sqlc.arg('account_id')
        AND ar.counterparty_account_id = sqlc.arg('customer_account_id')
        AND ar.account_relation_role_code = 'customer'
    )
    -- Pathway 3: product line via account group used as a price group for the customer
    OR EXISTS (
      SELECT 1 FROM account_group_product_line agpl
      JOIN account_relation_price_group arpg ON arpg.account_group_id = agpl.account_group_id
      JOIN account_relation ar ON ar.id = arpg.account_relation_id
      WHERE agpl.product_line_id = pl.id
        AND ar.owner_account_id = sqlc.arg('account_id')
        AND ar.counterparty_account_id = sqlc.arg('customer_account_id')
        AND ar.account_relation_role_code = 'customer'
    )
)
ORDER BY pl.name;

-- name: ListCatalogProducts :many
SELECT
    ic.id AS category_id,
    ic.name AS category_name,
    it.id AS item_id,
    it.sku,
    it.description
FROM product p
JOIN item it ON it.id = p.item_id
JOIN item_category ic ON ic.id = it.item_category_id
WHERE p.product_line_id = sqlc.arg('product_line_id')
  AND it.account_id = sqlc.arg('account_id')
  AND p.is_portal_ready = 1
  AND it.deleted_at IS NULL
  AND (ic.account_id = sqlc.narg('category_account_id') OR ic.account_id IS NULL)
ORDER BY ic.name, it.sku;

-- name: ListCatalogProductsForCustomer :many
SELECT
    ic.id AS category_id,
    ic.name AS category_name,
    it.id AS item_id,
    it.sku,
    it.description
FROM product p
JOIN item it ON it.id = p.item_id
JOIN item_category ic ON ic.id = it.item_category_id
WHERE p.product_line_id = sqlc.arg('product_line_id')
  AND it.account_id = sqlc.arg('account_id')
  AND p.is_portal_ready = 1
  AND it.deleted_at IS NULL
  AND (ic.account_id = sqlc.narg('category_account_id') OR ic.account_id IS NULL)
  AND (
    -- Pathway 1: product line via account group that the customer's account relation belongs to
    EXISTS (
      SELECT 1 FROM account_group_product_line agpl
      JOIN account_relation ar ON ar.account_group_id = agpl.account_group_id
      WHERE agpl.product_line_id = p.product_line_id
        AND ar.owner_account_id = sqlc.arg('account_id')
        AND ar.counterparty_account_id = sqlc.arg('customer_account_id')
        AND ar.account_relation_role_code = 'customer'
    )
    -- Pathway 2: product line via direct account relation product line assignment
    OR EXISTS (
      SELECT 1 FROM account_relation_product_line arpl
      JOIN account_relation ar ON ar.id = arpl.account_relation_id
      WHERE arpl.product_line_id = p.product_line_id
        AND ar.owner_account_id = sqlc.arg('account_id')
        AND ar.counterparty_account_id = sqlc.arg('customer_account_id')
        AND ar.account_relation_role_code = 'customer'
    )
    -- Pathway 3: product line via account group used as a price group for the customer
    OR EXISTS (
      SELECT 1 FROM account_group_product_line agpl
      JOIN account_relation_price_group arpg ON arpg.account_group_id = agpl.account_group_id
      JOIN account_relation ar ON ar.id = arpg.account_relation_id
      WHERE agpl.product_line_id = p.product_line_id
        AND ar.owner_account_id = sqlc.arg('account_id')
        AND ar.counterparty_account_id = sqlc.arg('customer_account_id')
        AND ar.account_relation_role_code = 'customer'
    )
  )
ORDER BY ic.name, it.sku;

-- name: ListCatalogCategoryProperties :many
SELECT
    icp.A AS item_category_id,
    pr.id AS property_id,
    pr.name AS property_name
FROM _item_categories_properties icp
JOIN property pr ON pr.id = icp.B
WHERE icp.A IN (sqlc.slice('category_ids'))
ORDER BY pr.name;

-- name: ListCatalogProductAttributes :many
-- Driven from the items' attribute rows. attribute and property hold every tenant's rows; left to
-- itself the planner scans attribute and probes each one for the items.
SELECT STRAIGHT_JOIN
    ia.B AS item_id,
    att.id AS attribute_id,
    att.text AS attribute_name,
    att.property_id,
    pr.name AS property_name
FROM _item_attributes ia FORCE INDEX (_item_attributes_B_index)
JOIN attribute att ON att.id = ia.A
JOIN property pr ON pr.id = att.property_id
WHERE ia.B IN (sqlc.slice('item_ids'))
ORDER BY pr.name, att.text;
