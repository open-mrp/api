-- name: ListAttributesForward :many
SELECT
    attribute.id,
    attribute.text,
    attribute.property_id,
    attribute.account_id,
    attribute.color_code,
    attribute.`order`,
    attribute.is_public,
    attribute.created_at,
    attribute.updated_at
FROM attribute
WHERE attribute.property_id = sqlc.arg('property_id')
AND attribute.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR attribute.text LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('cursor_order') IS NULL
    OR attribute.`order` > sqlc.narg('cursor_order')
    OR (attribute.`order` = sqlc.narg('cursor_order') AND attribute.id > sqlc.narg('cursor_id'))
)
ORDER BY attribute.`order` ASC, attribute.id ASC
LIMIT ?;

-- name: ListAttributesBackward :many
SELECT
    attribute.id,
    attribute.text,
    attribute.property_id,
    attribute.account_id,
    attribute.color_code,
    attribute.`order`,
    attribute.is_public,
    attribute.created_at,
    attribute.updated_at
FROM attribute
WHERE attribute.property_id = sqlc.arg('property_id')
AND attribute.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR attribute.text LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('cursor_order') IS NULL
    OR attribute.`order` < sqlc.narg('cursor_order')
    OR (attribute.`order` = sqlc.narg('cursor_order') AND attribute.id < sqlc.narg('cursor_id'))
)
ORDER BY attribute.`order` DESC, attribute.id DESC
LIMIT ?;

-- name: GetAttribute :one
SELECT
    attribute.id,
    attribute.text,
    attribute.property_id,
    attribute.account_id,
    attribute.color_code,
    attribute.`order`,
    attribute.is_public,
    attribute.created_at,
    attribute.updated_at
FROM attribute
WHERE attribute.id = sqlc.arg('id')
AND attribute.property_id = sqlc.arg('property_id')
AND attribute.account_id = sqlc.arg('account_id');

-- name: InsertAttribute :exec
INSERT INTO attribute (
    id,
    text,
    property_id,
    account_id,
    color_code,
    `order`,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('text'),
    sqlc.arg('property_id'),
    sqlc.arg('account_id'),
    sqlc.arg('color_code'),
    sqlc.arg('order'),
    NOW(3),
    NOW(3)
);

-- name: UpdateAttribute :execresult
UPDATE attribute SET
    text = COALESCE(sqlc.narg('text'), text),
    color_code = COALESCE(sqlc.narg('color_code'), color_code),
    `order` = COALESCE(sqlc.narg('order'), `order`),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND property_id = sqlc.arg('property_id')
AND account_id = sqlc.arg('account_id');

-- name: DeleteAttribute :execresult
DELETE FROM attribute
WHERE id = sqlc.arg('id')
AND property_id = sqlc.arg('property_id')
AND account_id = sqlc.arg('account_id');

-- name: ListAttributesByPropertyIDs :many
SELECT
    attribute.id,
    attribute.text,
    attribute.property_id,
    attribute.account_id,
    attribute.color_code,
    attribute.`order`,
    attribute.is_public,
    attribute.created_at,
    attribute.updated_at
FROM attribute
WHERE attribute.property_id IN (sqlc.slice('property_ids'))
AND attribute.account_id = sqlc.arg('account_id')
ORDER BY attribute.`order` ASC, attribute.created_at DESC;

-- name: GetAttributesByIDs :many
SELECT
    attribute.id,
    attribute.text,
    attribute.property_id,
    attribute.account_id,
    attribute.color_code,
    attribute.`order`,
    attribute.is_public,
    attribute.created_at,
    attribute.updated_at
FROM attribute
WHERE attribute.id IN (sqlc.slice('ids'))
AND attribute.account_id = sqlc.arg('account_id');

-- name: CountAttributesByTextInAccount :one
SELECT COUNT(*) FROM attribute
WHERE text = sqlc.arg('text') AND account_id = sqlc.arg('account_id')
AND (sqlc.narg('exclude_id') IS NULL OR id != sqlc.narg('exclude_id'));

-- name: CountAttributesByProperty :one
SELECT COUNT(*) FROM attribute
WHERE property_id = sqlc.arg('property_id')
AND account_id = sqlc.arg('account_id');

-- name: ShiftAttributeOrdersUp :exec
UPDATE attribute SET
    `order` = `order` + 1,
    updated_at = NOW(3)
WHERE property_id = sqlc.arg('property_id')
AND account_id = sqlc.arg('account_id')
AND `order` >= sqlc.arg('from_order');

-- name: ShiftAttributeOrdersDown :exec
UPDATE attribute SET
    `order` = `order` - 1,
    updated_at = NOW(3)
WHERE property_id = sqlc.arg('property_id')
AND account_id = sqlc.arg('account_id')
AND `order` > sqlc.arg('after_order');

-- name: ShiftAttributeOrdersUpBounded :exec
UPDATE attribute SET
    `order` = `order` + 1,
    updated_at = NOW(3)
WHERE property_id = sqlc.arg('property_id')
AND account_id = sqlc.arg('account_id')
AND `order` >= sqlc.arg('from_order')
AND `order` < sqlc.arg('to_order');

-- name: ShiftAttributeOrdersDownBounded :exec
UPDATE attribute SET
    `order` = `order` - 1,
    updated_at = NOW(3)
WHERE property_id = sqlc.arg('property_id')
AND account_id = sqlc.arg('account_id')
AND `order` > sqlc.arg('after_order')
AND `order` <= sqlc.arg('up_to_order');

-- name: FindAttributesByTextsInAccount :many
-- Used by bulk upsert to enforce the account-wide attribute value uniqueness the
-- manual create path enforces via ExistsByValueInAccount, in one batched lookup.
-- Matching is case-insensitive via the column collation.
SELECT
    a.id,
    a.text,
    a.property_id,
    p.name AS property_name
FROM attribute a
JOIN property p ON p.id = a.property_id
WHERE a.account_id = sqlc.arg('account_id')
AND a.text IN (sqlc.slice('texts'));
