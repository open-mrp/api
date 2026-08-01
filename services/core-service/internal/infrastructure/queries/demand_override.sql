-- Demand overrides let management account for demand the forecast cannot see — a large new customer about to order, a promotion, a discontinued line. They are plain rows with no customer linkage: the point is to move the number, not to model the deal.

-- name: ListDemandOverrideTypes :many
SELECT
    t.id,
    t.code,
    t.name,
    t.created_at,
    t.updated_at
FROM demand_override_type t
ORDER BY t.code ASC;

-- name: CreateDemandOverride :exec
INSERT INTO demand_override (
    id,
    account_id,
    scope_code,
    scope_ref_id,
    period_start_date,
    period_end_date,
    override_type_code,
    value,
    unit_id,
    reason_code,
    note,
    created_by_id,
    effective_from,
    expires_at,
    is_active,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('account_id'),
    sqlc.arg('scope_code'),
    sqlc.arg('scope_ref_id'),
    sqlc.arg('period_start_date'),
    sqlc.arg('period_end_date'),
    sqlc.arg('override_type_code'),
    sqlc.arg('value'),
    sqlc.narg('unit_id'),
    sqlc.narg('reason_code'),
    sqlc.narg('note'),
    sqlc.arg('created_by_id'),
    sqlc.arg('effective_from'),
    sqlc.narg('expires_at'),
    sqlc.arg('is_active'),
    NOW(3),
    NOW(3)
);

-- name: GetDemandOverride :one
SELECT
    o.id,
    o.account_id,
    o.scope_code,
    o.scope_ref_id,
    o.period_start_date,
    o.period_end_date,
    o.override_type_code,
    o.value,
    o.unit_id,
    o.reason_code,
    o.note,
    o.created_by_id,
    o.effective_from,
    o.expires_at,
    o.is_active,
    o.created_at,
    o.updated_at,
    COALESCE(si.description, spl.name, '') AS scope_name,
    si.sku AS scope_handle
FROM demand_override o
LEFT JOIN item si ON si.id = o.scope_ref_id AND o.scope_code = 'item'
LEFT JOIN product_line spl ON spl.id = o.scope_ref_id AND o.scope_code = 'product_line'
WHERE o.account_id = sqlc.arg('account_id')
AND o.id = sqlc.arg('id');

-- name: GetDemandOverridesByIDs :many
SELECT
    o.id,
    o.account_id,
    o.scope_code,
    o.scope_ref_id,
    o.period_start_date,
    o.period_end_date,
    o.override_type_code,
    o.value,
    o.unit_id,
    o.reason_code,
    o.note,
    o.created_by_id,
    o.effective_from,
    o.expires_at,
    o.is_active,
    o.created_at,
    o.updated_at,
    COALESCE(si.description, spl.name, '') AS scope_name,
    si.sku AS scope_handle
FROM demand_override o
LEFT JOIN item si ON si.id = o.scope_ref_id AND o.scope_code = 'item'
LEFT JOIN product_line spl ON spl.id = o.scope_ref_id AND o.scope_code = 'product_line'
WHERE o.account_id = sqlc.arg('account_id')
AND o.id IN (sqlc.slice('ids'));

-- name: UpdateDemandOverride :exec
UPDATE demand_override
SET
    period_start_date = COALESCE(sqlc.narg('period_start_date'), period_start_date),
    period_end_date = COALESCE(sqlc.narg('period_end_date'), period_end_date),
    override_type_code = COALESCE(sqlc.narg('override_type_code'), override_type_code),
    value = COALESCE(sqlc.narg('value'), value),
    unit_id = IF(sqlc.arg('clear_unit_id'), NULL, COALESCE(sqlc.narg('unit_id'), unit_id)),
    reason_code = IF(sqlc.arg('clear_reason_code'), NULL, COALESCE(sqlc.narg('reason_code'), reason_code)),
    note = IF(sqlc.arg('clear_note'), NULL, COALESCE(sqlc.narg('note'), note)),
    expires_at = IF(sqlc.arg('clear_expires_at'), NULL, COALESCE(sqlc.narg('expires_at'), expires_at)),
    is_active = COALESCE(sqlc.narg('is_active'), is_active),
    updated_at = NOW(3)
WHERE account_id = sqlc.arg('account_id')
AND id = sqlc.arg('id');

-- name: DeleteDemandOverride :exec
DELETE FROM demand_override
WHERE account_id = sqlc.arg('account_id')
AND id = sqlc.arg('id');

-- name: ListDemandOverridesForward :many
SELECT
    o.id,
    o.account_id,
    o.scope_code,
    o.scope_ref_id,
    o.period_start_date,
    o.period_end_date,
    o.override_type_code,
    o.value,
    o.unit_id,
    o.reason_code,
    o.note,
    o.created_by_id,
    o.effective_from,
    o.expires_at,
    o.is_active,
    o.created_at,
    o.updated_at,
    COALESCE(si.description, spl.name, '') AS scope_name,
    si.sku AS scope_handle
-- FORCE INDEX pins the one index that satisfies the ORDER BY (created_at, id) without a filesort. The sort-free set deliberately excludes demand_override_account_scope_period_idx and demand_override_account_active_period_idx: they pin scope_code / is_active and cannot satisfy the sort, so with the item/product_line joins the optimizer could pick one, read every matching row and filesort the lot before LIMIT. All filters below run as residuals bounded by LIMIT on the small, account-scoped partition. Do not remove.
FROM demand_override o FORCE INDEX (demand_override_account_created_idx)
LEFT JOIN item si ON si.id = o.scope_ref_id AND o.scope_code = 'item'
LEFT JOIN product_line spl ON spl.id = o.scope_ref_id AND o.scope_code = 'product_line'
WHERE o.account_id = sqlc.arg('account_id')
AND (
    sqlc.arg('include_scope_filter') = false
    OR o.scope_code IN (sqlc.slice('scope_codes'))
)
AND (
    sqlc.arg('include_scope_ref_filter') = false
    OR o.scope_ref_id IN (sqlc.slice('scope_ref_ids'))
)
AND (
    sqlc.arg('include_type_filter') = false
    OR o.override_type_code IN (sqlc.slice('override_type_codes'))
)
AND (sqlc.narg('is_active') IS NULL OR o.is_active = sqlc.narg('is_active'))
-- Period filters use overlap, not containment: an override spanning Q3 is relevant to a question about August even though neither endpoint falls inside August.
AND (sqlc.narg('period_start') IS NULL OR o.period_end_date >= sqlc.narg('period_start'))
AND (sqlc.narg('period_end') IS NULL OR o.period_start_date <= sqlc.narg('period_end'))
-- Free-text search runs against the note, the only prose an override carries. Substring rather than prefix: notes are sentences, so a prefix match would be useless. It is a residual filter on an already small, account-scoped set.
AND (
    sqlc.narg('search_query') IS NULL
    OR o.note LIKE sqlc.narg('search_query')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR o.created_at < sqlc.narg('cursor_created_at')
    OR (o.created_at = sqlc.narg('cursor_created_at') AND o.id < sqlc.narg('cursor_id'))
)
ORDER BY o.created_at DESC, o.id DESC
LIMIT ?;

-- name: ListDemandOverridesBackward :many
SELECT
    o.id,
    o.account_id,
    o.scope_code,
    o.scope_ref_id,
    o.period_start_date,
    o.period_end_date,
    o.override_type_code,
    o.value,
    o.unit_id,
    o.reason_code,
    o.note,
    o.created_by_id,
    o.effective_from,
    o.expires_at,
    o.is_active,
    o.created_at,
    o.updated_at,
    COALESCE(si.description, spl.name, '') AS scope_name,
    si.sku AS scope_handle
-- FORCE INDEX pins the one index that satisfies the ORDER BY (created_at, id) without a filesort; see ListDemandOverridesForward for why. Do not remove.
FROM demand_override o FORCE INDEX (demand_override_account_created_idx)
LEFT JOIN item si ON si.id = o.scope_ref_id AND o.scope_code = 'item'
LEFT JOIN product_line spl ON spl.id = o.scope_ref_id AND o.scope_code = 'product_line'
WHERE o.account_id = sqlc.arg('account_id')
AND (
    sqlc.arg('include_scope_filter') = false
    OR o.scope_code IN (sqlc.slice('scope_codes'))
)
AND (
    sqlc.arg('include_scope_ref_filter') = false
    OR o.scope_ref_id IN (sqlc.slice('scope_ref_ids'))
)
AND (
    sqlc.arg('include_type_filter') = false
    OR o.override_type_code IN (sqlc.slice('override_type_codes'))
)
AND (sqlc.narg('is_active') IS NULL OR o.is_active = sqlc.narg('is_active'))
AND (sqlc.narg('period_start') IS NULL OR o.period_end_date >= sqlc.narg('period_start'))
AND (sqlc.narg('period_end') IS NULL OR o.period_start_date <= sqlc.narg('period_end'))
-- Free-text search runs against the note, the only prose an override carries. Substring rather than prefix: notes are sentences, so a prefix match would be useless. It is a residual filter on an already small, account-scoped set.
AND (
    sqlc.narg('search_query') IS NULL
    OR o.note LIKE sqlc.narg('search_query')
)
AND (
    o.created_at > sqlc.arg('cursor_created_at')
    OR (o.created_at = sqlc.arg('cursor_created_at') AND o.id > sqlc.arg('cursor_id'))
)
ORDER BY o.created_at ASC, o.id ASC
LIMIT ?;
