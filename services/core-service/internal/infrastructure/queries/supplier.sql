-- name: ListSuppliersByIDs :many
-- Reads the rows of a page buildSupplierListPageQuery chose. Its columns match GetSupplier's, so one mapper reads both.
SELECT
    ar.id AS relation_id,
    ar.counterparty_account_id AS account_id,
    COALESCE(NULLIF(ar.alias, ''), a.name) AS account_name,
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
    -- The links the supplier's materials list returns: a material whose item was deleted is not one.
    (
        SELECT STRAIGHT_JOIN COUNT(*)
        FROM supplier_material smc FORCE INDEX (supplier_material_supplier_account_id_idx)
        JOIN material mc ON mc.id = smc.material_id
        JOIN item ic ON ic.id = mc.item_id
        WHERE smc.supplier_account_id = ar.counterparty_account_id
          AND smc.owner_account_id = ar.owner_account_id
          AND ic.deleted_at IS NULL
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
  AND ar.counterparty_account_id IN (sqlc.slice('ids'));

-- name: FindSuppliersByNames :many
-- Used by bulk upsert to resolve supplier names to supplier account IDs within the
-- owner account. A name is the owner's alias for the supplier, or the supplier account's
-- own name when it has none. Match is case-insensitive via the column collation.
SELECT
    ar.counterparty_account_id AS account_id,
    COALESCE(NULLIF(ar.alias, ''), a.name) AS account_name
FROM account_relation ar
INNER JOIN account a ON a.id = ar.counterparty_account_id
WHERE ar.owner_account_id = sqlc.arg('owner_account_id')
  AND ar.account_relation_role_code = 'supplier'
  AND (
    (ar.alias <> '' AND ar.alias IN (sqlc.slice('alias_names')))
    OR (COALESCE(ar.alias, '') = '' AND a.name IN (sqlc.slice('account_names')))
  );

-- name: GetSupplier :one
SELECT
    ar.id AS relation_id,
    ar.counterparty_account_id AS account_id,
    COALESCE(NULLIF(ar.alias, ''), a.name) AS account_name,
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
        SELECT STRAIGHT_JOIN COUNT(*)
        FROM supplier_material smc FORCE INDEX (supplier_material_supplier_account_id_idx)
        JOIN material mc ON mc.id = smc.material_id
        JOIN item ic ON ic.id = mc.item_id
        WHERE smc.supplier_account_id = ar.counterparty_account_id
          AND smc.owner_account_id = ar.owner_account_id
          AND ic.deleted_at IS NULL
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
-- A rename is the owner's name for the supplier, so it lands on the relation's alias, never on the supplier's account.
UPDATE account_relation SET
    alias = COALESCE(sqlc.narg('alias'), alias),
    external_number = COALESCE(sqlc.narg('external_number'), external_number),
    notes = CASE WHEN sqlc.arg('update_notes') = true THEN sqlc.narg('notes') ELSE notes END,
    default_billing_address_id = sqlc.narg('default_billing_address_id'),
    default_shipping_address_id = sqlc.narg('default_shipping_address_id'),
    updated_at = NOW(3)
WHERE owner_account_id = sqlc.arg('owner_account_id')
  AND counterparty_account_id = sqlc.arg('counterparty_account_id')
  AND account_relation_role_code = 'supplier';

-- name: LockSupplierNumbers :one
-- No unique index guards supplier numbers, so their writers queue on the owner's account row. Take it before the transaction's first read so the number check sees every earlier holder's commit.
SELECT id FROM account
WHERE id = sqlc.arg('owner_account_id')
FOR UPDATE;

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
