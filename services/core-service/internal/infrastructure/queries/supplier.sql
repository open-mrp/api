-- name: ListSuppliersByIDs :many
-- Reads the rows of a page supplierListPageQuery chose, and the suppliers documents name. Its columns match
-- GetSupplier's, so one mapper reads both.
SELECT
    ar.id AS relation_id,
    ar.counterparty_account_id AS account_id,
    COALESCE(NULLIF(ar.alias, ''), a.name) AS account_name,
    ar.external_number,
    ar.is_active,
    ar.instructions,
    ab.support_email AS email,
    ab.phone_number,
    ab.website_url,
    ar.carrier_billing_type,
    ar.carrier_billing_account,
    ar.default_carrier_id,
    ar.default_carrier_option_id,
    ar.payment_term_id,
    ar.shipping_term_id,
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
    ar.created_at,
    ar.updated_at
FROM account_relation ar
INNER JOIN account a ON a.id = ar.counterparty_account_id
LEFT JOIN account_branding ab ON ab.owner_account_id = ar.counterparty_account_id
LEFT JOIN address ba ON ba.id = ar.default_billing_address_id
LEFT JOIN geolocation bg ON bg.id = ba.geolocation_id
LEFT JOIN address sa ON sa.id = ar.default_shipping_address_id
LEFT JOIN geolocation sg ON sg.id = sa.geolocation_id
WHERE ar.owner_account_id = sqlc.arg('owner_account_id')
  AND ar.account_relation_role_code = 'supplier'
  AND ar.counterparty_account_id IN (sqlc.slice('ids'));

-- name: CountSupplierMaterials :many
-- The supplier items each supplier has, not counting one whose item was deleted. Read only for callers that show it.
SELECT smc.supplier_account_id, COUNT(*) AS material_count
FROM supplier_material smc
JOIN material mc ON mc.id = smc.material_id
JOIN item ic ON ic.id = mc.item_id
WHERE smc.owner_account_id = sqlc.arg('owner_account_id')
  AND smc.supplier_account_id IN (sqlc.slice('supplier_ids'))
  AND ic.deleted_at IS NULL
GROUP BY smc.supplier_account_id;

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
    ar.is_active,
    ar.instructions,
    ab.support_email AS email,
    ab.phone_number,
    ab.website_url,
    ar.carrier_billing_type,
    ar.carrier_billing_account,
    ar.default_carrier_id,
    ar.default_carrier_option_id,
    ar.payment_term_id,
    ar.shipping_term_id,
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
    ar.created_at,
    ar.updated_at
FROM account_relation ar
INNER JOIN account a ON a.id = ar.counterparty_account_id
LEFT JOIN account_branding ab ON ab.owner_account_id = ar.counterparty_account_id
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
    alias, external_number, is_active, instructions, priority_code,
    carrier_billing_type, carrier_billing_account, default_carrier_id, default_carrier_option_id,
    payment_term_id, shipping_term_id,
    default_billing_address_id, default_shipping_address_id,
    created_at, updated_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('owner_account_id'), sqlc.arg('counterparty_account_id'), 'supplier',
    sqlc.arg('alias'), sqlc.arg('external_number'), sqlc.arg('is_active'), sqlc.narg('instructions'), 'normal',
    sqlc.narg('carrier_billing_type'), sqlc.narg('carrier_billing_account'), sqlc.narg('default_carrier_id'), sqlc.narg('default_carrier_option_id'),
    sqlc.narg('payment_term_id'), sqlc.narg('shipping_term_id'),
    sqlc.narg('default_billing_address_id'), sqlc.narg('default_shipping_address_id'),
    NOW(3), NOW(3)
);

-- name: UpdateSupplierRelation :exec
-- Every column is written with the value the service merged from the request and the row, so the statement has no
-- per-field branches. A rename lands on the relation's alias, never on the supplier's account.
UPDATE account_relation SET
    alias = sqlc.arg('alias'),
    external_number = sqlc.arg('external_number'),
    is_active = sqlc.arg('is_active'),
    instructions = sqlc.narg('instructions'),
    carrier_billing_type = sqlc.narg('carrier_billing_type'),
    carrier_billing_account = sqlc.narg('carrier_billing_account'),
    default_carrier_id = sqlc.narg('default_carrier_id'),
    default_carrier_option_id = sqlc.narg('default_carrier_option_id'),
    payment_term_id = sqlc.narg('payment_term_id'),
    shipping_term_id = sqlc.narg('shipping_term_id'),
    default_billing_address_id = sqlc.narg('default_billing_address_id'),
    default_shipping_address_id = sqlc.narg('default_shipping_address_id'),
    updated_at = NOW(3)
WHERE owner_account_id = sqlc.arg('owner_account_id')
  AND counterparty_account_id = sqlc.arg('counterparty_account_id')
  AND account_relation_role_code = 'supplier';

-- name: LockSupplierNumbers :one
-- Supplier number writers queue on the owner's account row, as customer numbers do. Take it before the transaction's first read so the number check sees every earlier holder's commit.
SELECT id FROM account
WHERE id = sqlc.arg('owner_account_id')
FOR UPDATE;

-- name: SupplierExistsByNumber :one
SELECT COUNT(*) > 0 AS supplier_exists FROM account_relation
WHERE owner_account_id = sqlc.arg('owner_account_id')
AND external_number = sqlc.arg('external_number')
AND account_relation_role_code = 'supplier'
AND (sqlc.narg('exclude_counterparty_id') IS NULL OR counterparty_account_id != sqlc.narg('exclude_counterparty_id'));

-- name: AllocateNextSupplierNumber :execresult
-- Reserves the owner's next supplier number and returns it via LAST_INSERT_ID. A series of its own, so suppliers and
-- customers never share numbers.
INSERT INTO sys_property (id, account_id, sys_property_type_code, value, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('account_id'), 'supplier_number', LAST_INSERT_ID(1), NOW(3), NOW(3))
ON DUPLICATE KEY UPDATE value = LAST_INSERT_ID(value + 1), updated_at = NOW(3);

-- name: SupplierNumberValueInUse :one
SELECT EXISTS (
    SELECT 1 FROM account_relation
    WHERE owner_account_id = sqlc.arg('owner_account_id')
    AND account_relation_role_code = 'supplier'
    AND (
        external_number = sqlc.arg('number')
        OR (external_number LIKE '0%' AND external_number REGEXP '^[0-9]+$' AND CAST(external_number AS UNSIGNED) = sqlc.arg('value'))
    )
) AS in_use;

-- HighestNumericSupplierNumber is the highest all-digit supplier number the owner uses that the counter (an INT) could
-- also hand out. It is read only when the counter is created or turns out to be behind a number someone typed in.
-- name: HighestNumericSupplierNumber :one
SELECT CAST(COALESCE(MAX(CAST(external_number AS UNSIGNED)), 0) AS SIGNED) AS highest
FROM account_relation
WHERE owner_account_id = sqlc.arg('owner_account_id')
AND account_relation_role_code = 'supplier'
AND external_number REGEXP '^0*[0-9]{1,10}$'
AND CAST(external_number AS UNSIGNED) < 2147483647;

-- RaiseSupplierNumberCounter moves the counter up to value, creating it there if the owner has none. It never moves
-- it down.
-- name: RaiseSupplierNumberCounter :exec
INSERT INTO sys_property (id, account_id, sys_property_type_code, value, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('account_id'), 'supplier_number', sqlc.arg('value'), NOW(3), NOW(3))
ON DUPLICATE KEY UPDATE value = GREATEST(value, sqlc.arg('value')), updated_at = NOW(3);

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
