-- name: InsertPart :exec
INSERT INTO part (id, item_id, created_at, updated_at)
VALUES (?, ?, NOW(3), NOW(3));

-- name: InsertItemForPart :exec
INSERT INTO item (
    id, sku, description, notes, unit_value_id, burn_rate_id,
    account_id, item_type_code, unit_cost_id, item_category_id, is_dirty,
    created_at, updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('sku'),
    sqlc.narg('description'),
    sqlc.narg('notes'),
    sqlc.arg('unit_value_id'),
    sqlc.arg('burn_rate_id'),
    sqlc.arg('account_id'),
    'part',
    sqlc.arg('unit_cost_id'),
    sqlc.arg('item_category_id'),
    false,
    NOW(3),
    NOW(3)
);

-- name: InsertRateForPart :exec
INSERT INTO rate (
    id, value, numerator_unit_id, denominator_unit_id,
    created_at, updated_at
) VALUES (?, ?, ?, ?, NOW(3), NOW(3));

-- name: GetPartBase :one
SELECT
    p.id AS part_id,
    p.created_at AS part_created_at,
    p.updated_at AS part_updated_at,
    i.id,
    i.sku,
    i.description,
    i.notes,
    i.item_type_code,
    i.item_category_id,
    i.unit_value_id,
    i.unit_cost_id,
    i.burn_rate_id,
    i.account_id,
    i.is_dirty,
    i.created_at,
    i.updated_at,
    ic.name AS category_name,
    ic.item_category_type_code,
    ic.unit_group_id AS category_unit_group_id,
    ic.created_at AS category_created_at,
    ic.updated_at AS category_updated_at
FROM part p
JOIN item i ON i.id = p.item_id
JOIN item_category ic ON ic.id = i.item_category_id
WHERE p.id = sqlc.arg('part_id')
AND i.account_id = sqlc.arg('account_id')
AND i.deleted_at IS NULL;

-- name: GetPartAttributes :many
SELECT
    a.id,
    a.text,
    a.color_code,
    a.`order`,
    a.property_id,
    a.created_at,
    a.updated_at
FROM _item_attributes ia
JOIN attribute a ON a.id = ia.A
WHERE ia.B = sqlc.arg('item_id');

-- name: ListPartsForwardBase :many
SELECT
    p.id AS part_id,
    p.created_at AS part_created_at,
    p.updated_at AS part_updated_at,
    i.id,
    i.sku,
    i.description,
    i.notes,
    i.item_type_code,
    i.item_category_id,
    i.unit_value_id,
    i.unit_cost_id,
    i.burn_rate_id,
    i.account_id,
    i.is_dirty,
    i.created_at,
    i.updated_at,
    ic.name AS category_name,
    ic.item_category_type_code,
    ic.unit_group_id AS category_unit_group_id,
    ic.created_at AS category_created_at,
    ic.updated_at AS category_updated_at
FROM part p
JOIN item i ON i.id = p.item_id
JOIN item_category ic ON ic.id = i.item_category_id
WHERE i.account_id = sqlc.arg('account_id')
AND i.deleted_at IS NULL
AND (
    sqlc.arg('include_category_filter') = false
    OR i.item_category_id IN (sqlc.slice('category_ids'))
)
AND (
    sqlc.arg('include_attribute_filter') = false
    OR EXISTS (
        SELECT 1 FROM _item_attributes ia
        WHERE ia.B = i.id
        AND ia.A IN (sqlc.slice('attribute_ids'))
    )
)
AND (
    sqlc.narg('start_date') IS NULL
    OR i.created_at >= sqlc.narg('start_date')
)
AND (
    sqlc.narg('end_date') IS NULL
    OR i.created_at <= sqlc.narg('end_date')
)
AND (
    sqlc.narg('search_query') IS NULL
    OR i.sku LIKE sqlc.narg('search_query')
    OR i.description LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR (
        (sqlc.narg('cursor_match_tier') IS NULL AND (
            i.created_at < sqlc.narg('cursor_created_at')
            OR (i.created_at = sqlc.narg('cursor_created_at') AND i.id < sqlc.narg('cursor_id'))
        ))
        OR (sqlc.narg('cursor_match_tier') IS NOT NULL AND (
            (CASE
                WHEN CAST(sqlc.narg('search_exact') AS CHAR) IS NULL THEN 0
                WHEN i.sku COLLATE utf8mb4_general_ci = CAST(sqlc.narg('search_exact') AS CHAR) THEN 0
                WHEN i.sku COLLATE utf8mb4_general_ci LIKE CONCAT('% ', CAST(sqlc.narg('search_exact') AS CHAR), ' %')
                    OR i.sku COLLATE utf8mb4_general_ci LIKE CONCAT(CAST(sqlc.narg('search_exact') AS CHAR), ' %')
                    OR i.sku COLLATE utf8mb4_general_ci LIKE CONCAT('% ', CAST(sqlc.narg('search_exact') AS CHAR)) THEN 1
                WHEN sqlc.narg('search_prefix') IS NOT NULL AND i.sku COLLATE utf8mb4_general_ci LIKE sqlc.narg('search_prefix') THEN 2
                ELSE 3
            END) > CAST(sqlc.narg('cursor_match_tier') AS SIGNED)
            OR (
                (CASE
                    WHEN CAST(sqlc.narg('search_exact') AS CHAR) IS NULL THEN 0
                    WHEN i.sku COLLATE utf8mb4_general_ci = CAST(sqlc.narg('search_exact') AS CHAR) THEN 0
                    WHEN i.sku COLLATE utf8mb4_general_ci LIKE CONCAT('% ', CAST(sqlc.narg('search_exact') AS CHAR), ' %')
                        OR i.sku COLLATE utf8mb4_general_ci LIKE CONCAT(CAST(sqlc.narg('search_exact') AS CHAR), ' %')
                        OR i.sku COLLATE utf8mb4_general_ci LIKE CONCAT('% ', CAST(sqlc.narg('search_exact') AS CHAR)) THEN 1
                    WHEN sqlc.narg('search_prefix') IS NOT NULL AND i.sku COLLATE utf8mb4_general_ci LIKE sqlc.narg('search_prefix') THEN 2
                    ELSE 3
                END) = CAST(sqlc.narg('cursor_match_tier') AS SIGNED)
                AND (
                    i.created_at < sqlc.narg('cursor_created_at')
                    OR (i.created_at = sqlc.narg('cursor_created_at') AND i.id < sqlc.narg('cursor_id'))
                )
            )
        ))
    )
)
ORDER BY
    CASE
        WHEN CAST(sqlc.narg('search_exact') AS CHAR) IS NULL THEN 0
        WHEN i.sku COLLATE utf8mb4_general_ci = CAST(sqlc.narg('search_exact') AS CHAR) THEN 0
        WHEN i.sku COLLATE utf8mb4_general_ci LIKE CONCAT('% ', CAST(sqlc.narg('search_exact') AS CHAR), ' %')
            OR i.sku COLLATE utf8mb4_general_ci LIKE CONCAT(CAST(sqlc.narg('search_exact') AS CHAR), ' %')
            OR i.sku COLLATE utf8mb4_general_ci LIKE CONCAT('% ', CAST(sqlc.narg('search_exact') AS CHAR)) THEN 1
        WHEN sqlc.narg('search_prefix') IS NOT NULL AND i.sku COLLATE utf8mb4_general_ci LIKE sqlc.narg('search_prefix') THEN 2
        ELSE 3
    END ASC,
    i.created_at DESC,
    i.id DESC
LIMIT ?;

-- name: ListPartsBackwardBase :many
SELECT
    p.id AS part_id,
    p.created_at AS part_created_at,
    p.updated_at AS part_updated_at,
    i.id,
    i.sku,
    i.description,
    i.notes,
    i.item_type_code,
    i.item_category_id,
    i.unit_value_id,
    i.unit_cost_id,
    i.burn_rate_id,
    i.account_id,
    i.is_dirty,
    i.created_at,
    i.updated_at,
    ic.name AS category_name,
    ic.item_category_type_code,
    ic.unit_group_id AS category_unit_group_id,
    ic.created_at AS category_created_at,
    ic.updated_at AS category_updated_at
FROM part p
JOIN item i ON i.id = p.item_id
JOIN item_category ic ON ic.id = i.item_category_id
WHERE i.account_id = sqlc.arg('account_id')
AND i.deleted_at IS NULL
AND (
    sqlc.arg('include_category_filter') = false
    OR i.item_category_id IN (sqlc.slice('category_ids'))
)
AND (
    sqlc.arg('include_attribute_filter') = false
    OR EXISTS (
        SELECT 1 FROM _item_attributes ia
        WHERE ia.B = i.id
        AND ia.A IN (sqlc.slice('attribute_ids'))
    )
)
AND (
    sqlc.narg('start_date') IS NULL
    OR i.created_at >= sqlc.narg('start_date')
)
AND (
    sqlc.narg('end_date') IS NULL
    OR i.created_at <= sqlc.narg('end_date')
)
AND (
    sqlc.narg('search_query') IS NULL
    OR i.sku LIKE sqlc.narg('search_query')
    OR i.description LIKE sqlc.narg('search_query')
)
AND (
    (sqlc.narg('cursor_match_tier') IS NULL AND (
        i.created_at > sqlc.arg('cursor_created_at')
        OR (i.created_at = sqlc.arg('cursor_created_at') AND i.id > sqlc.arg('cursor_id'))
    ))
    OR (sqlc.narg('cursor_match_tier') IS NOT NULL AND (
        (CASE
            WHEN CAST(sqlc.narg('search_exact') AS CHAR) IS NULL THEN 0
            WHEN i.sku COLLATE utf8mb4_general_ci = CAST(sqlc.narg('search_exact') AS CHAR) THEN 0
            WHEN i.sku COLLATE utf8mb4_general_ci LIKE CONCAT('% ', CAST(sqlc.narg('search_exact') AS CHAR), ' %')
                OR i.sku COLLATE utf8mb4_general_ci LIKE CONCAT(CAST(sqlc.narg('search_exact') AS CHAR), ' %')
                OR i.sku COLLATE utf8mb4_general_ci LIKE CONCAT('% ', CAST(sqlc.narg('search_exact') AS CHAR)) THEN 1
            WHEN sqlc.narg('search_prefix') IS NOT NULL AND i.sku COLLATE utf8mb4_general_ci LIKE sqlc.narg('search_prefix') THEN 2
            ELSE 3
        END) < CAST(sqlc.narg('cursor_match_tier') AS SIGNED)
        OR (
            (CASE
                WHEN CAST(sqlc.narg('search_exact') AS CHAR) IS NULL THEN 0
                WHEN i.sku COLLATE utf8mb4_general_ci = CAST(sqlc.narg('search_exact') AS CHAR) THEN 0
                WHEN i.sku COLLATE utf8mb4_general_ci LIKE CONCAT('% ', CAST(sqlc.narg('search_exact') AS CHAR), ' %')
                    OR i.sku COLLATE utf8mb4_general_ci LIKE CONCAT(CAST(sqlc.narg('search_exact') AS CHAR), ' %')
                    OR i.sku COLLATE utf8mb4_general_ci LIKE CONCAT('% ', CAST(sqlc.narg('search_exact') AS CHAR)) THEN 1
                WHEN sqlc.narg('search_prefix') IS NOT NULL AND i.sku COLLATE utf8mb4_general_ci LIKE sqlc.narg('search_prefix') THEN 2
                ELSE 3
            END) = CAST(sqlc.narg('cursor_match_tier') AS SIGNED)
            AND (
                i.created_at > sqlc.arg('cursor_created_at')
                OR (i.created_at = sqlc.arg('cursor_created_at') AND i.id > sqlc.arg('cursor_id'))
            )
        )
    ))
)
ORDER BY
    CASE
        WHEN CAST(sqlc.narg('search_exact') AS CHAR) IS NULL THEN 0
        WHEN i.sku COLLATE utf8mb4_general_ci = CAST(sqlc.narg('search_exact') AS CHAR) THEN 0
        WHEN i.sku COLLATE utf8mb4_general_ci LIKE CONCAT('% ', CAST(sqlc.narg('search_exact') AS CHAR), ' %')
            OR i.sku COLLATE utf8mb4_general_ci LIKE CONCAT(CAST(sqlc.narg('search_exact') AS CHAR), ' %')
            OR i.sku COLLATE utf8mb4_general_ci LIKE CONCAT('% ', CAST(sqlc.narg('search_exact') AS CHAR)) THEN 1
        WHEN sqlc.narg('search_prefix') IS NOT NULL AND i.sku COLLATE utf8mb4_general_ci LIKE sqlc.narg('search_prefix') THEN 2
        ELSE 3
    END DESC,
    i.created_at ASC,
    i.id ASC
LIMIT ?;

-- name: SoftDeletePart :exec
UPDATE item i
JOIN part p ON p.item_id = i.id
SET i.deleted_at = NOW(3), i.updated_at = NOW(3)
WHERE p.id = sqlc.arg('part_id')
AND i.account_id = sqlc.arg('account_id')
AND i.deleted_at IS NULL;

-- name: PartUpdateItem :execresult
UPDATE item SET
    sku = COALESCE(sqlc.narg('sku'), sku),
    description = sqlc.narg('description'),
    notes = sqlc.narg('notes'),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id')
AND deleted_at IS NULL;

-- name: TouchPartUpdatedAt :exec
UPDATE part SET updated_at = NOW(3)
WHERE id = sqlc.arg('part_id');

-- name: CheckPartSKUExists :one
SELECT EXISTS(
  SELECT 1 FROM item
  WHERE sku = sqlc.arg('sku')
  AND account_id = sqlc.arg('account_id')
  AND id != sqlc.arg('exclude_id')
  AND deleted_at IS NULL
) AS sku_exists;

-- name: ExportPartsWithFilters :many
SELECT
    p.id AS part_id,
    p.created_at AS part_created_at,
    p.updated_at AS part_updated_at,
    i.id,
    i.sku,
    i.description,
    i.notes,
    i.item_type_code,
    i.item_category_id,
    i.unit_value_id,
    i.unit_cost_id,
    i.burn_rate_id,
    i.account_id,
    i.is_dirty,
    i.created_at,
    i.updated_at,
    ic.name AS category_name,
    ic.item_category_type_code,
    ic.unit_group_id AS category_unit_group_id,
    ic.created_at AS category_created_at,
    ic.updated_at AS category_updated_at
FROM part p
JOIN item i ON i.id = p.item_id
JOIN item_category ic ON ic.id = i.item_category_id
WHERE i.account_id = sqlc.arg('account_id')
AND i.deleted_at IS NULL
AND (
    sqlc.arg('include_category_filter') = false
    OR i.item_category_id IN (sqlc.slice('category_ids'))
)
AND (
    sqlc.arg('include_attribute_filter') = false
    OR EXISTS (
        SELECT 1 FROM _item_attributes ia
        WHERE ia.B = i.id
        AND ia.A IN (sqlc.slice('attribute_ids'))
    )
)
AND (
    sqlc.narg('start_date') IS NULL
    OR i.created_at >= sqlc.narg('start_date')
)
AND (
    sqlc.narg('end_date') IS NULL
    OR i.created_at <= sqlc.narg('end_date')
)
AND (
    sqlc.narg('search_query') IS NULL
    OR i.sku LIKE sqlc.narg('search_query')
    OR i.description LIKE sqlc.narg('search_query')
)
ORDER BY
    CASE
        WHEN CAST(sqlc.narg('search_exact') AS CHAR) IS NULL THEN 0
        WHEN i.sku COLLATE utf8mb4_general_ci = CAST(sqlc.narg('search_exact') AS CHAR) THEN 0
        WHEN i.sku COLLATE utf8mb4_general_ci LIKE CONCAT('% ', CAST(sqlc.narg('search_exact') AS CHAR), ' %')
            OR i.sku COLLATE utf8mb4_general_ci LIKE CONCAT(CAST(sqlc.narg('search_exact') AS CHAR), ' %')
            OR i.sku COLLATE utf8mb4_general_ci LIKE CONCAT('% ', CAST(sqlc.narg('search_exact') AS CHAR)) THEN 1
        WHEN sqlc.narg('search_prefix') IS NOT NULL AND i.sku COLLATE utf8mb4_general_ci LIKE sqlc.narg('search_prefix') THEN 2
        ELSE 3
    END ASC,
    i.created_at DESC,
    i.id DESC;
