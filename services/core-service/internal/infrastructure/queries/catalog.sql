-- name: ListCatalogProductLines :many
SELECT DISTINCT pl.id, pl.name
FROM product_line pl
JOIN product p ON p.product_line_id = pl.id
JOIN item it ON it.id = p.item_id
WHERE it.account_id = sqlc.arg('account_id')
  AND p.is_portal_ready = 1
  AND it.deleted_at IS NULL
ORDER BY pl.name;

-- name: ListCatalogProductLinesForCustomer :many
SELECT DISTINCT pl.id, pl.name
FROM product_line pl
JOIN product p ON p.product_line_id = pl.id
JOIN item it ON it.id = p.item_id
WHERE it.account_id = sqlc.arg('account_id')
  AND p.is_portal_ready = 1
  AND it.deleted_at IS NULL
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
SELECT
    ia.B AS item_id,
    att.id AS attribute_id,
    att.text AS attribute_name,
    att.property_id,
    pr.name AS property_name
FROM _item_attributes ia
JOIN attribute att ON att.id = ia.A
JOIN property pr ON pr.id = att.property_id
WHERE ia.B IN (sqlc.slice('item_ids'))
ORDER BY pr.name, att.text;
