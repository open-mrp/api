-- name: ListSuppliersForward :many
SELECT
    ar.id AS relation_id,
    ar.counterparty_account_id AS account_id,
    a.name AS account_name,
    ar.external_number,
    ar.notes,
    ba.id AS default_billing_address_id,
    ba.name AS default_billing_address_name,
    ba.phone AS default_billing_address_phone,
    ba.email AS default_billing_address_email,
    ba.is_drop_ship AS default_billing_is_drop_ship,
    bg.id AS default_billing_geolocation_id,
    bg.street_line_1 AS default_billing_street_line_1,
    bg.street_line_2 AS default_billing_street_line_2,
    bg.locality AS default_billing_locality,
    bg.state AS default_billing_state,
    bg.postal_code AS default_billing_postal_code,
    bg.country AS default_billing_country,
    ba.created_at AS default_billing_address_created_at,
    ba.updated_at AS default_billing_address_updated_at,
    sa.id AS default_shipping_address_id,
    sa.name AS default_shipping_address_name,
    sa.phone AS default_shipping_address_phone,
    sa.email AS default_shipping_address_email,
    sa.is_drop_ship AS default_shipping_is_drop_ship,
    sg.id AS default_shipping_geolocation_id,
    sg.street_line_1 AS default_shipping_street_line_1,
    sg.street_line_2 AS default_shipping_street_line_2,
    sg.locality AS default_shipping_locality,
    sg.state AS default_shipping_state,
    sg.postal_code AS default_shipping_postal_code,
    sg.country AS default_shipping_country,
    sa.created_at AS default_shipping_address_created_at,
    sa.updated_at AS default_shipping_address_updated_at,
    (
        SELECT COUNT(*) FROM supplier_material sm
        WHERE sm.supplier_account_id = ar.counterparty_account_id
          AND sm.owner_account_id = ar.owner_account_id
    ) AS material_count,
    ar.created_at,
    ar.updated_at
FROM account_relation ar
INNER JOIN account a ON a.id = ar.counterparty_account_id
LEFT JOIN address ba ON ba.id = ar.default_billing_address_id
LEFT JOIN geolocation bg ON bg.id = ba.geolocation_id
LEFT JOIN address sa ON sa.id = ar.default_shipping_address_id
LEFT JOIN geolocation sg ON sg.id = sa.geolocation_id
WHERE ar.owner_account_id = sqlc.arg('owner_account_id')
  AND ar.account_relation_role_code = 'supplier'
  AND (
    sqlc.narg('search_query') IS NULL
    OR a.name LIKE sqlc.narg('search_query')
    OR ar.external_number LIKE sqlc.narg('search_query')
  )
  AND (
    sqlc.narg('start_date') IS NULL
    OR ar.created_at >= sqlc.narg('start_date')
  )
  AND (
    sqlc.narg('end_date') IS NULL
    OR ar.created_at <= sqlc.narg('end_date')
  )
  AND (
    sqlc.arg('has_item_filter') = false
    OR EXISTS (
        SELECT 1 FROM supplier_material sm2
        INNER JOIN material m ON m.id = sm2.material_id
        WHERE sm2.supplier_account_id = ar.counterparty_account_id
          AND sm2.owner_account_id = ar.owner_account_id
          AND m.item_id IN (sqlc.slice('item_ids'))
    )
  )
  AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR ar.created_at < sqlc.narg('cursor_created_at')
    OR (ar.created_at = sqlc.narg('cursor_created_at') AND ar.counterparty_account_id < sqlc.narg('cursor_id'))
  )
ORDER BY ar.created_at DESC, ar.counterparty_account_id DESC
LIMIT ?;

-- name: ListSuppliersBackward :many
SELECT
    ar.id AS relation_id,
    ar.counterparty_account_id AS account_id,
    a.name AS account_name,
    ar.external_number,
    ar.notes,
    ba.id AS default_billing_address_id,
    ba.name AS default_billing_address_name,
    ba.phone AS default_billing_address_phone,
    ba.email AS default_billing_address_email,
    ba.is_drop_ship AS default_billing_is_drop_ship,
    bg.id AS default_billing_geolocation_id,
    bg.street_line_1 AS default_billing_street_line_1,
    bg.street_line_2 AS default_billing_street_line_2,
    bg.locality AS default_billing_locality,
    bg.state AS default_billing_state,
    bg.postal_code AS default_billing_postal_code,
    bg.country AS default_billing_country,
    ba.created_at AS default_billing_address_created_at,
    ba.updated_at AS default_billing_address_updated_at,
    sa.id AS default_shipping_address_id,
    sa.name AS default_shipping_address_name,
    sa.phone AS default_shipping_address_phone,
    sa.email AS default_shipping_address_email,
    sa.is_drop_ship AS default_shipping_is_drop_ship,
    sg.id AS default_shipping_geolocation_id,
    sg.street_line_1 AS default_shipping_street_line_1,
    sg.street_line_2 AS default_shipping_street_line_2,
    sg.locality AS default_shipping_locality,
    sg.state AS default_shipping_state,
    sg.postal_code AS default_shipping_postal_code,
    sg.country AS default_shipping_country,
    sa.created_at AS default_shipping_address_created_at,
    sa.updated_at AS default_shipping_address_updated_at,
    (
        SELECT COUNT(*) FROM supplier_material sm
        WHERE sm.supplier_account_id = ar.counterparty_account_id
          AND sm.owner_account_id = ar.owner_account_id
    ) AS material_count,
    ar.created_at,
    ar.updated_at
FROM account_relation ar
INNER JOIN account a ON a.id = ar.counterparty_account_id
LEFT JOIN address ba ON ba.id = ar.default_billing_address_id
LEFT JOIN geolocation bg ON bg.id = ba.geolocation_id
LEFT JOIN address sa ON sa.id = ar.default_shipping_address_id
LEFT JOIN geolocation sg ON sg.id = sa.geolocation_id
WHERE ar.owner_account_id = sqlc.arg('owner_account_id')
  AND ar.account_relation_role_code = 'supplier'
  AND (
    sqlc.narg('search_query') IS NULL
    OR a.name LIKE sqlc.narg('search_query')
    OR ar.external_number LIKE sqlc.narg('search_query')
  )
  AND (
    sqlc.narg('start_date') IS NULL
    OR ar.created_at >= sqlc.narg('start_date')
  )
  AND (
    sqlc.narg('end_date') IS NULL
    OR ar.created_at <= sqlc.narg('end_date')
  )
  AND (
    sqlc.arg('has_item_filter') = false
    OR EXISTS (
        SELECT 1 FROM supplier_material sm2
        INNER JOIN material m ON m.id = sm2.material_id
        WHERE sm2.supplier_account_id = ar.counterparty_account_id
          AND sm2.owner_account_id = ar.owner_account_id
          AND m.item_id IN (sqlc.slice('item_ids'))
    )
  )
  AND (
    ar.created_at > sqlc.arg('cursor_created_at')
    OR (ar.created_at = sqlc.arg('cursor_created_at') AND ar.counterparty_account_id > sqlc.arg('cursor_id'))
  )
ORDER BY ar.created_at ASC, ar.counterparty_account_id ASC
LIMIT ?;

-- name: CountSuppliers :one
SELECT COUNT(*) AS total
FROM account_relation ar
INNER JOIN account a ON a.id = ar.counterparty_account_id
WHERE ar.owner_account_id = sqlc.arg('owner_account_id')
  AND ar.account_relation_role_code = 'supplier'
  AND (
    sqlc.narg('search_query') IS NULL
    OR a.name LIKE sqlc.narg('search_query')
    OR ar.external_number LIKE sqlc.narg('search_query')
  )
  AND (
    sqlc.narg('start_date') IS NULL
    OR ar.created_at >= sqlc.narg('start_date')
  )
  AND (
    sqlc.narg('end_date') IS NULL
    OR ar.created_at <= sqlc.narg('end_date')
  )
  AND (
    sqlc.arg('has_item_filter') = false
    OR EXISTS (
        SELECT 1 FROM supplier_material sm2
        INNER JOIN material m ON m.id = sm2.material_id
        WHERE sm2.supplier_account_id = ar.counterparty_account_id
          AND sm2.owner_account_id = ar.owner_account_id
          AND m.item_id IN (sqlc.slice('item_ids'))
    )
  );

-- name: FindSuppliersByNames :many
-- Used by bulk upsert to resolve supplier names to supplier account IDs within the
-- owner account. Match is case-insensitive via the column collation.
SELECT
    ar.counterparty_account_id AS account_id,
    a.name AS account_name
FROM account_relation ar
INNER JOIN account a ON a.id = ar.counterparty_account_id
WHERE ar.owner_account_id = sqlc.arg('owner_account_id')
  AND ar.account_relation_role_code = 'supplier'
  AND a.name IN (sqlc.slice('names'));

-- name: GetSupplier :one
SELECT
    ar.id AS relation_id,
    ar.counterparty_account_id AS account_id,
    a.name AS account_name,
    ar.external_number,
    ar.notes,
    ba.id AS default_billing_address_id,
    ba.name AS default_billing_address_name,
    ba.phone AS default_billing_address_phone,
    ba.email AS default_billing_address_email,
    ba.is_drop_ship AS default_billing_is_drop_ship,
    bg.id AS default_billing_geolocation_id,
    bg.street_line_1 AS default_billing_street_line_1,
    bg.street_line_2 AS default_billing_street_line_2,
    bg.locality AS default_billing_locality,
    bg.state AS default_billing_state,
    bg.postal_code AS default_billing_postal_code,
    bg.country AS default_billing_country,
    ba.created_at AS default_billing_address_created_at,
    ba.updated_at AS default_billing_address_updated_at,
    sa.id AS default_shipping_address_id,
    sa.name AS default_shipping_address_name,
    sa.phone AS default_shipping_address_phone,
    sa.email AS default_shipping_address_email,
    sa.is_drop_ship AS default_shipping_is_drop_ship,
    sg.id AS default_shipping_geolocation_id,
    sg.street_line_1 AS default_shipping_street_line_1,
    sg.street_line_2 AS default_shipping_street_line_2,
    sg.locality AS default_shipping_locality,
    sg.state AS default_shipping_state,
    sg.postal_code AS default_shipping_postal_code,
    sg.country AS default_shipping_country,
    sa.created_at AS default_shipping_address_created_at,
    sa.updated_at AS default_shipping_address_updated_at,
    (
        SELECT COUNT(*) FROM supplier_material sm
        WHERE sm.supplier_account_id = ar.counterparty_account_id
          AND sm.owner_account_id = ar.owner_account_id
    ) AS material_count,
    ar.created_at,
    ar.updated_at
FROM account_relation ar
INNER JOIN account a ON a.id = ar.counterparty_account_id
LEFT JOIN address ba ON ba.id = ar.default_billing_address_id
LEFT JOIN geolocation bg ON bg.id = ba.geolocation_id
LEFT JOIN address sa ON sa.id = ar.default_shipping_address_id
LEFT JOIN geolocation sg ON sg.id = sa.geolocation_id
WHERE ar.owner_account_id = sqlc.arg('owner_account_id')
  AND ar.counterparty_account_id = sqlc.arg('counterparty_account_id')
  AND ar.account_relation_role_code = 'supplier';

-- name: InsertSupplierAccount :exec
INSERT INTO account (id, name, account_type_code, onboarding_status_code, default_billing_address_id, default_shipping_address_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('name'), 'company', 'unclaimed', sqlc.narg('default_billing_address_id'), sqlc.narg('default_shipping_address_id'), NOW(3), NOW(3));

-- name: InsertSupplierRelation :exec
INSERT INTO account_relation (
    id, owner_account_id, counterparty_account_id, account_relation_role_code,
    alias, external_number, notes, priority_code,
    default_billing_address_id, default_shipping_address_id,
    created_at, updated_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('owner_account_id'), sqlc.arg('counterparty_account_id'), 'supplier',
    sqlc.arg('alias'), sqlc.arg('external_number'), sqlc.narg('notes'), 'normal',
    sqlc.narg('default_billing_address_id'), sqlc.narg('default_shipping_address_id'),
    NOW(3), NOW(3)
);

-- name: UpdateSupplierRelation :exec
UPDATE account_relation SET
    external_number = COALESCE(sqlc.narg('external_number'), external_number),
    notes = CASE WHEN sqlc.arg('update_notes') = true THEN sqlc.narg('notes') ELSE notes END,
    default_billing_address_id = sqlc.narg('default_billing_address_id'),
    default_shipping_address_id = sqlc.narg('default_shipping_address_id'),
    updated_at = NOW(3)
WHERE owner_account_id = sqlc.arg('owner_account_id')
  AND counterparty_account_id = sqlc.arg('counterparty_account_id')
  AND account_relation_role_code = 'supplier';

-- name: UpdateSupplierAccountName :exec
UPDATE account SET
    name = sqlc.arg('name'),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id');

-- name: SupplierExistsByNumber :one
SELECT COUNT(*) > 0 AS supplier_exists FROM account_relation
WHERE owner_account_id = sqlc.arg('owner_account_id')
AND external_number = sqlc.arg('external_number')
AND account_relation_role_code = 'supplier'
AND (sqlc.narg('exclude_counterparty_id') IS NULL OR counterparty_account_id != sqlc.narg('exclude_counterparty_id'));

-- name: DeleteSupplierAccountUsers :exec
DELETE FROM account_user
WHERE account_id = sqlc.arg('account_id');

-- name: DeleteSupplierAccountAddresses :exec
DELETE FROM account_address
WHERE account_id = sqlc.arg('account_id');

-- name: DeleteSupplierRelation :exec
DELETE FROM account_relation
WHERE owner_account_id = sqlc.arg('owner_account_id')
  AND counterparty_account_id = sqlc.arg('counterparty_account_id')
  AND account_relation_role_code = 'supplier';

-- name: BulkDeleteSupplierAccountUsers :exec
DELETE FROM account_user
WHERE account_id IN (sqlc.slice('account_ids'));

-- name: BulkDeleteSupplierAccountAddresses :exec
DELETE FROM account_address
WHERE account_id IN (sqlc.slice('account_ids'));

-- name: BulkDeleteSupplierRelations :exec
DELETE FROM account_relation
WHERE owner_account_id = sqlc.arg('owner_account_id')
  AND counterparty_account_id IN (sqlc.slice('counterparty_account_ids'))
  AND account_relation_role_code = 'supplier';
