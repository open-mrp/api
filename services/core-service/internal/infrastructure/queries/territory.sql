-- name: ListTerritoriesForward :many
SELECT
    t.id,
    t.state,
    t.start_zipcode,
    t.end_zipcode,
    t.sales_rep_id,
    t.product_line_id,
    t.created_at,
    t.updated_at,
    u.name AS sales_rep_name,
    u.email AS sales_rep_email,
    pl.name AS product_line_name
FROM territory t
JOIN account_user au ON au.id = t.sales_rep_id
JOIN user u ON u.id = au.user_id
LEFT JOIN product_line pl ON pl.id = t.product_line_id
WHERE t.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR t.state LIKE sqlc.narg('search_query')
    OR u.name LIKE sqlc.narg('search_query')
    OR u.email LIKE sqlc.narg('search_query')
    OR pl.name LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('zipcode_query') IS NULL
    OR (
        (t.start_zipcode IS NULL OR t.start_zipcode <= CAST(sqlc.narg('zipcode_query') AS SIGNED))
        AND (
            (t.end_zipcode IS NULL AND t.start_zipcode = CAST(sqlc.narg('zipcode_query') AS SIGNED))
            OR t.end_zipcode >= CAST(sqlc.narg('zipcode_query') AS SIGNED)
        )
    )
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR t.created_at < sqlc.narg('cursor_created_at')
    OR (t.created_at = sqlc.narg('cursor_created_at') AND t.id < sqlc.narg('cursor_id'))
)
ORDER BY t.created_at DESC, t.id DESC
LIMIT ?;

-- name: ListTerritoriesBackward :many
SELECT
    t.id,
    t.state,
    t.start_zipcode,
    t.end_zipcode,
    t.sales_rep_id,
    t.product_line_id,
    t.created_at,
    t.updated_at,
    u.name AS sales_rep_name,
    u.email AS sales_rep_email,
    pl.name AS product_line_name
FROM territory t
JOIN account_user au ON au.id = t.sales_rep_id
JOIN user u ON u.id = au.user_id
LEFT JOIN product_line pl ON pl.id = t.product_line_id
WHERE t.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR t.state LIKE sqlc.narg('search_query')
    OR u.name LIKE sqlc.narg('search_query')
    OR u.email LIKE sqlc.narg('search_query')
    OR pl.name LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('zipcode_query') IS NULL
    OR (
        (t.start_zipcode IS NULL OR t.start_zipcode <= CAST(sqlc.narg('zipcode_query') AS SIGNED))
        AND (
            (t.end_zipcode IS NULL AND t.start_zipcode = CAST(sqlc.narg('zipcode_query') AS SIGNED))
            OR t.end_zipcode >= CAST(sqlc.narg('zipcode_query') AS SIGNED)
        )
    )
)
AND (
    t.created_at > sqlc.arg('cursor_created_at')
    OR (t.created_at = sqlc.arg('cursor_created_at') AND t.id > sqlc.arg('cursor_id'))
)
ORDER BY t.created_at ASC, t.id ASC
LIMIT ?;

-- name: GetTerritory :one
SELECT
    t.id,
    t.state,
    t.start_zipcode,
    t.end_zipcode,
    t.sales_rep_id,
    t.product_line_id,
    t.created_at,
    t.updated_at,
    u.name AS sales_rep_name,
    u.email AS sales_rep_email,
    pl.name AS product_line_name
FROM territory t
JOIN account_user au ON au.id = t.sales_rep_id
JOIN user u ON u.id = au.user_id
LEFT JOIN product_line pl ON pl.id = t.product_line_id
WHERE t.id = sqlc.arg('id')
AND t.account_id = sqlc.arg('account_id');

-- name: InsertTerritory :exec
INSERT INTO territory (
    id,
    state,
    start_zipcode,
    end_zipcode,
    sales_rep_id,
    account_id,
    product_line_id,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('state'),
    sqlc.narg('start_zipcode'),
    sqlc.narg('end_zipcode'),
    sqlc.arg('sales_rep_id'),
    sqlc.arg('account_id'),
    sqlc.narg('product_line_id'),
    NOW(3),
    NOW(3)
);

-- name: UpdateTerritory :exec
UPDATE territory SET
    state = COALESCE(sqlc.narg('state'), state),
    start_zipcode = CASE
        WHEN sqlc.arg('clear_start_zipcode') = TRUE THEN NULL
        WHEN sqlc.narg('start_zipcode') IS NOT NULL THEN sqlc.narg('start_zipcode')
        ELSE start_zipcode
    END,
    end_zipcode = CASE
        WHEN sqlc.arg('clear_end_zipcode') = TRUE THEN NULL
        WHEN sqlc.narg('end_zipcode') IS NOT NULL THEN sqlc.narg('end_zipcode')
        ELSE end_zipcode
    END,
    sales_rep_id = sqlc.narg('sales_rep_id'),
    product_line_id = CASE
        WHEN sqlc.arg('clear_product_line') = TRUE THEN NULL
        WHEN sqlc.narg('product_line_id') IS NOT NULL THEN sqlc.narg('product_line_id')
        ELSE product_line_id
    END,
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: DeleteTerritory :exec
DELETE FROM territory
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: CheckTerritoryInAccount :one
SELECT EXISTS(
    SELECT 1 FROM territory
    WHERE id = sqlc.arg('id')
    AND account_id = sqlc.arg('account_id')
) AS `exists`;
