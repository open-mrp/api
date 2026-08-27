-- name: ListInventoryChangeLogsForward :many
SELECT
    icl.id,
    icl.action_type_code,
    icl.account_id,
    icl.created_at,
    icl.updated_at,
    i.id AS item_id,
    i.sku AS item_sku,
    i.item_type_code AS item_type_code,
    i.created_at AS item_created_at,
    i.updated_at AS item_updated_at,
    q.id AS quantity_id,
    q.value AS quantity_value,
    u.id AS quantity_unit_id,
    u.name AS quantity_unit_name,
    u.abbreviation AS quantity_unit_abbreviation,
    u.unit_dimension_code AS quantity_unit_type,
    u.ratio_numerator AS quantity_unit_ratio_numerator,
    u.ratio_denominator AS quantity_unit_ratio_denominator,
    u.offset_numerator AS quantity_unit_offset_numerator,
    u.offset_denominator AS quantity_unit_offset_denominator,
    u.created_at AS quantity_unit_created_at,
    u.updated_at AS quantity_unit_updated_at,
    icl.scanning_station_id,
    ss.name AS scanning_station_name,
    ss.scanning_station_type_code AS scanning_station_type,
    ss.created_at AS scanning_station_created_at,
    ss.updated_at AS scanning_station_updated_at,
    icl.responsible_user_id,
    usr.name AS responsible_user_name,
    usr.created_at AS responsible_user_created_at,
    usr.updated_at AS responsible_user_updated_at
FROM inventory_change_log icl
JOIN item i ON i.id = icl.item_id
JOIN quantity q ON q.id = icl.quantity_id
JOIN unit u ON u.id = q.unit_id
LEFT JOIN scanning_station ss ON ss.id = icl.scanning_station_id
LEFT JOIN user usr ON usr.id = icl.responsible_user_id
WHERE icl.account_id = sqlc.arg('account_id')
AND (
    sqlc.arg('include_item_filter') = false
    OR icl.item_id IN (sqlc.slice('item_ids'))
)
AND (
    sqlc.arg('include_action_type_filter') = false
    OR icl.action_type_code IN (sqlc.slice('action_type_codes'))
)
AND (
    sqlc.arg('include_user_filter') = false
    OR icl.responsible_user_id IN (sqlc.slice('changed_by_user_ids'))
)
AND (
    sqlc.narg('start_date') IS NULL
    OR icl.created_at >= sqlc.narg('start_date')
)
AND (
    sqlc.narg('end_date') IS NULL
    OR icl.created_at <= sqlc.narg('end_date')
)
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR icl.created_at < sqlc.narg('cursor_created_at')
    OR (icl.created_at = sqlc.narg('cursor_created_at') AND icl.id < sqlc.narg('cursor_id'))
)
ORDER BY icl.created_at DESC, icl.id DESC
LIMIT ?;

-- name: ListInventoryChangeLogsBackward :many
SELECT
    icl.id,
    icl.action_type_code,
    icl.account_id,
    icl.created_at,
    icl.updated_at,
    i.id AS item_id,
    i.sku AS item_sku,
    i.item_type_code AS item_type_code,
    i.created_at AS item_created_at,
    i.updated_at AS item_updated_at,
    q.id AS quantity_id,
    q.value AS quantity_value,
    u.id AS quantity_unit_id,
    u.name AS quantity_unit_name,
    u.abbreviation AS quantity_unit_abbreviation,
    u.unit_dimension_code AS quantity_unit_type,
    u.ratio_numerator AS quantity_unit_ratio_numerator,
    u.ratio_denominator AS quantity_unit_ratio_denominator,
    u.offset_numerator AS quantity_unit_offset_numerator,
    u.offset_denominator AS quantity_unit_offset_denominator,
    u.created_at AS quantity_unit_created_at,
    u.updated_at AS quantity_unit_updated_at,
    icl.scanning_station_id,
    ss.name AS scanning_station_name,
    ss.scanning_station_type_code AS scanning_station_type,
    ss.created_at AS scanning_station_created_at,
    ss.updated_at AS scanning_station_updated_at,
    icl.responsible_user_id,
    usr.name AS responsible_user_name,
    usr.created_at AS responsible_user_created_at,
    usr.updated_at AS responsible_user_updated_at
FROM inventory_change_log icl
JOIN item i ON i.id = icl.item_id
JOIN quantity q ON q.id = icl.quantity_id
JOIN unit u ON u.id = q.unit_id
LEFT JOIN scanning_station ss ON ss.id = icl.scanning_station_id
LEFT JOIN user usr ON usr.id = icl.responsible_user_id
WHERE icl.account_id = sqlc.arg('account_id')
AND (
    sqlc.arg('include_item_filter') = false
    OR icl.item_id IN (sqlc.slice('item_ids'))
)
AND (
    sqlc.arg('include_action_type_filter') = false
    OR icl.action_type_code IN (sqlc.slice('action_type_codes'))
)
AND (
    sqlc.arg('include_user_filter') = false
    OR icl.responsible_user_id IN (sqlc.slice('changed_by_user_ids'))
)
AND (
    sqlc.narg('start_date') IS NULL
    OR icl.created_at >= sqlc.narg('start_date')
)
AND (
    sqlc.narg('end_date') IS NULL
    OR icl.created_at <= sqlc.narg('end_date')
)
AND (
    icl.created_at > sqlc.arg('cursor_created_at')
    OR (icl.created_at = sqlc.arg('cursor_created_at') AND icl.id > sqlc.arg('cursor_id'))
)
ORDER BY icl.created_at ASC, icl.id ASC
LIMIT ?;

-- name: GetInventoryChangeLog :one
SELECT
    icl.id,
    icl.action_type_code,
    icl.account_id,
    icl.created_at,
    icl.updated_at,
    i.id AS item_id,
    i.sku AS item_sku,
    i.item_type_code AS item_type_code,
    i.created_at AS item_created_at,
    i.updated_at AS item_updated_at,
    q.id AS quantity_id,
    q.value AS quantity_value,
    u.id AS quantity_unit_id,
    u.name AS quantity_unit_name,
    u.abbreviation AS quantity_unit_abbreviation,
    u.unit_dimension_code AS quantity_unit_type,
    u.ratio_numerator AS quantity_unit_ratio_numerator,
    u.ratio_denominator AS quantity_unit_ratio_denominator,
    u.offset_numerator AS quantity_unit_offset_numerator,
    u.offset_denominator AS quantity_unit_offset_denominator,
    u.created_at AS quantity_unit_created_at,
    u.updated_at AS quantity_unit_updated_at,
    icl.scanning_station_id,
    ss.name AS scanning_station_name,
    ss.scanning_station_type_code AS scanning_station_type,
    ss.created_at AS scanning_station_created_at,
    ss.updated_at AS scanning_station_updated_at,
    icl.responsible_user_id,
    usr.name AS responsible_user_name,
    usr.created_at AS responsible_user_created_at,
    usr.updated_at AS responsible_user_updated_at
FROM inventory_change_log icl
JOIN item i ON i.id = icl.item_id
JOIN quantity q ON q.id = icl.quantity_id
JOIN unit u ON u.id = q.unit_id
LEFT JOIN scanning_station ss ON ss.id = icl.scanning_station_id
LEFT JOIN user usr ON usr.id = icl.responsible_user_id
WHERE icl.id = sqlc.arg('id')
AND icl.account_id = sqlc.arg('account_id');

-- name: ListAllInventoryChangeLogs :many
SELECT
    icl.id,
    icl.action_type_code,
    icl.account_id,
    icl.created_at,
    icl.updated_at,
    i.id AS item_id,
    i.sku AS item_sku,
    i.item_type_code AS item_type_code,
    q.id AS quantity_id,
    q.value AS quantity_value,
    u.id AS quantity_unit_id,
    u.name AS quantity_unit_name,
    u.abbreviation AS quantity_unit_abbreviation,
    u.unit_dimension_code AS quantity_unit_type,
    icl.scanning_station_id,
    ss.name AS scanning_station_name,
    ss.scanning_station_type_code AS scanning_station_type,
    icl.responsible_user_id,
    usr.name AS responsible_user_name
FROM inventory_change_log icl
JOIN item i ON i.id = icl.item_id
JOIN quantity q ON q.id = icl.quantity_id
JOIN unit u ON u.id = q.unit_id
LEFT JOIN scanning_station ss ON ss.id = icl.scanning_station_id
LEFT JOIN user usr ON usr.id = icl.responsible_user_id
WHERE icl.account_id = sqlc.arg('account_id')
AND (
    sqlc.arg('include_item_filter') = false
    OR icl.item_id IN (sqlc.slice('item_ids'))
)
AND (
    sqlc.arg('include_action_type_filter') = false
    OR icl.action_type_code IN (sqlc.slice('action_type_codes'))
)
AND (
    sqlc.arg('include_user_filter') = false
    OR icl.responsible_user_id IN (sqlc.slice('changed_by_user_ids'))
)
AND (
    sqlc.narg('start_date') IS NULL
    OR icl.created_at >= sqlc.narg('start_date')
)
AND (
    sqlc.narg('end_date') IS NULL
    OR icl.created_at <= sqlc.narg('end_date')
)
ORDER BY icl.created_at DESC, icl.id DESC;

-- name: ListConsumptionChangeLogsForBurnRate :many
SELECT
    q.value,
    q.unit_id,
    icl.created_at
FROM inventory_change_log icl
JOIN quantity q ON q.id = icl.quantity_id
WHERE icl.item_id = sqlc.arg('item_id')
AND icl.account_id = sqlc.arg('account_id')
AND icl.created_at >= DATE_SUB(NOW(), INTERVAL 30 DAY)
AND icl.action_type_code IN ('scan', 'user_correction')
AND CAST(q.value AS DECIMAL(65,30)) < 0
ORDER BY icl.created_at ASC;

-- SearchInventoryChangeLogsForward is the free-text page: the same list as ListInventoryChangeLogsForward, restricted to rows matching a search term.
--
-- The term is resolved to id sets on item, user and scanning_station before this runs, so the predicate here is an equality on an indexed column rather than a LIKE. That matters more than it looks: matching with `i.sku LIKE ? OR usr.name LIKE ? OR ss.name LIKE ?` across the joins reaches no index at all, and a term matching nothing walks the account's entire log - 47s on the largest tenant.
-- One branch per dimension, unioned, rather than one OR over the three columns: an OR reaches no single index either and cost 24s on that same tenant, where the union costs 0.13s. Each branch drives its own (account_id, <dimension>, created_at DESC, id DESC) composite.
-- Each branch is cut to the page limit and the merge is cut again. Taking the top page from each branch is enough to build the true top page, because every branch carries the same sort - a row in the global page is necessarily in its own branch's page.
-- A dimension the term did not match arrives as an empty set, which the generated SQL renders as `IN (NULL)`; the branch is then an impossible seek rather than a scan. When no dimension matches at all the caller skips this query entirely and returns an empty page.
-- name: SearchInventoryChangeLogsForward :many
SELECT
    icl.id,
    icl.action_type_code,
    icl.account_id,
    icl.created_at,
    icl.updated_at,
    i.id AS item_id,
    i.sku AS item_sku,
    i.item_type_code AS item_type_code,
    i.created_at AS item_created_at,
    i.updated_at AS item_updated_at,
    q.id AS quantity_id,
    q.value AS quantity_value,
    u.id AS quantity_unit_id,
    u.name AS quantity_unit_name,
    u.abbreviation AS quantity_unit_abbreviation,
    u.unit_dimension_code AS quantity_unit_type,
    u.ratio_numerator AS quantity_unit_ratio_numerator,
    u.ratio_denominator AS quantity_unit_ratio_denominator,
    u.offset_numerator AS quantity_unit_offset_numerator,
    u.offset_denominator AS quantity_unit_offset_denominator,
    u.created_at AS quantity_unit_created_at,
    u.updated_at AS quantity_unit_updated_at,
    icl.scanning_station_id,
    ss.name AS scanning_station_name,
    ss.scanning_station_type_code AS scanning_station_type,
    ss.created_at AS scanning_station_created_at,
    ss.updated_at AS scanning_station_updated_at,
    icl.responsible_user_id,
    usr.name AS responsible_user_name,
    usr.created_at AS responsible_user_created_at,
    usr.updated_at AS responsible_user_updated_at
FROM (
    (SELECT icl.id AS id, icl.created_at AS created_at
     FROM inventory_change_log icl
     WHERE icl.account_id = sqlc.arg('account_id')
      AND icl.item_id IN (sqlc.slice('search_item_ids'))
      AND (sqlc.arg('include_item_filter') = false OR icl.item_id IN (sqlc.slice('item_ids')))
      AND (sqlc.arg('include_action_type_filter') = false OR icl.action_type_code IN (sqlc.slice('action_type_codes')))
      AND (sqlc.arg('include_user_filter') = false OR icl.responsible_user_id IN (sqlc.slice('changed_by_user_ids')))
      AND (sqlc.narg('start_date') IS NULL OR icl.created_at >= sqlc.narg('start_date'))
      AND (sqlc.narg('end_date') IS NULL OR icl.created_at <= sqlc.narg('end_date'))
      AND (
          sqlc.narg('cursor_created_at') IS NULL
          OR icl.created_at < sqlc.narg('cursor_created_at')
          OR (icl.created_at = sqlc.narg('cursor_created_at') AND icl.id < sqlc.narg('cursor_id'))
      )
     ORDER BY icl.created_at DESC, icl.id DESC
     LIMIT ?)
    UNION DISTINCT
    (SELECT icl.id AS id, icl.created_at AS created_at
     FROM inventory_change_log icl
     WHERE icl.account_id = sqlc.arg('account_id')
      AND icl.responsible_user_id IN (sqlc.slice('search_user_ids'))
      AND (sqlc.arg('include_item_filter') = false OR icl.item_id IN (sqlc.slice('item_ids')))
      AND (sqlc.arg('include_action_type_filter') = false OR icl.action_type_code IN (sqlc.slice('action_type_codes')))
      AND (sqlc.arg('include_user_filter') = false OR icl.responsible_user_id IN (sqlc.slice('changed_by_user_ids')))
      AND (sqlc.narg('start_date') IS NULL OR icl.created_at >= sqlc.narg('start_date'))
      AND (sqlc.narg('end_date') IS NULL OR icl.created_at <= sqlc.narg('end_date'))
      AND (
          sqlc.narg('cursor_created_at') IS NULL
          OR icl.created_at < sqlc.narg('cursor_created_at')
          OR (icl.created_at = sqlc.narg('cursor_created_at') AND icl.id < sqlc.narg('cursor_id'))
      )
     ORDER BY icl.created_at DESC, icl.id DESC
     LIMIT ?)
    UNION DISTINCT
    (SELECT icl.id AS id, icl.created_at AS created_at
     FROM inventory_change_log icl
     WHERE icl.account_id = sqlc.arg('account_id')
      AND icl.scanning_station_id IN (sqlc.slice('search_station_ids'))
      AND (sqlc.arg('include_item_filter') = false OR icl.item_id IN (sqlc.slice('item_ids')))
      AND (sqlc.arg('include_action_type_filter') = false OR icl.action_type_code IN (sqlc.slice('action_type_codes')))
      AND (sqlc.arg('include_user_filter') = false OR icl.responsible_user_id IN (sqlc.slice('changed_by_user_ids')))
      AND (sqlc.narg('start_date') IS NULL OR icl.created_at >= sqlc.narg('start_date'))
      AND (sqlc.narg('end_date') IS NULL OR icl.created_at <= sqlc.narg('end_date'))
      AND (
          sqlc.narg('cursor_created_at') IS NULL
          OR icl.created_at < sqlc.narg('cursor_created_at')
          OR (icl.created_at = sqlc.narg('cursor_created_at') AND icl.id < sqlc.narg('cursor_id'))
      )
     ORDER BY icl.created_at DESC, icl.id DESC
     LIMIT ?)
) matched
JOIN inventory_change_log icl ON icl.id = matched.id
JOIN item i ON i.id = icl.item_id
JOIN quantity q ON q.id = icl.quantity_id
JOIN unit u ON u.id = q.unit_id
LEFT JOIN scanning_station ss ON ss.id = icl.scanning_station_id
LEFT JOIN user usr ON usr.id = icl.responsible_user_id
ORDER BY icl.created_at DESC, icl.id DESC
LIMIT ?;

-- SearchInventoryChangeLogsBackward is SearchInventoryChangeLogsForward walking the other way, for a backward cursor. See that query for why the search is a union of resolved-id branches.
-- name: SearchInventoryChangeLogsBackward :many
SELECT
    icl.id,
    icl.action_type_code,
    icl.account_id,
    icl.created_at,
    icl.updated_at,
    i.id AS item_id,
    i.sku AS item_sku,
    i.item_type_code AS item_type_code,
    i.created_at AS item_created_at,
    i.updated_at AS item_updated_at,
    q.id AS quantity_id,
    q.value AS quantity_value,
    u.id AS quantity_unit_id,
    u.name AS quantity_unit_name,
    u.abbreviation AS quantity_unit_abbreviation,
    u.unit_dimension_code AS quantity_unit_type,
    u.ratio_numerator AS quantity_unit_ratio_numerator,
    u.ratio_denominator AS quantity_unit_ratio_denominator,
    u.offset_numerator AS quantity_unit_offset_numerator,
    u.offset_denominator AS quantity_unit_offset_denominator,
    u.created_at AS quantity_unit_created_at,
    u.updated_at AS quantity_unit_updated_at,
    icl.scanning_station_id,
    ss.name AS scanning_station_name,
    ss.scanning_station_type_code AS scanning_station_type,
    ss.created_at AS scanning_station_created_at,
    ss.updated_at AS scanning_station_updated_at,
    icl.responsible_user_id,
    usr.name AS responsible_user_name,
    usr.created_at AS responsible_user_created_at,
    usr.updated_at AS responsible_user_updated_at
FROM (
    (SELECT icl.id AS id, icl.created_at AS created_at
     FROM inventory_change_log icl
     WHERE icl.account_id = sqlc.arg('account_id')
      AND icl.item_id IN (sqlc.slice('search_item_ids'))
      AND (sqlc.arg('include_item_filter') = false OR icl.item_id IN (sqlc.slice('item_ids')))
      AND (sqlc.arg('include_action_type_filter') = false OR icl.action_type_code IN (sqlc.slice('action_type_codes')))
      AND (sqlc.arg('include_user_filter') = false OR icl.responsible_user_id IN (sqlc.slice('changed_by_user_ids')))
      AND (sqlc.narg('start_date') IS NULL OR icl.created_at >= sqlc.narg('start_date'))
      AND (sqlc.narg('end_date') IS NULL OR icl.created_at <= sqlc.narg('end_date'))
      AND (
          icl.created_at > sqlc.arg('cursor_created_at')
          OR (icl.created_at = sqlc.arg('cursor_created_at') AND icl.id > sqlc.arg('cursor_id'))
      )
     ORDER BY icl.created_at ASC, icl.id ASC
     LIMIT ?)
    UNION DISTINCT
    (SELECT icl.id AS id, icl.created_at AS created_at
     FROM inventory_change_log icl
     WHERE icl.account_id = sqlc.arg('account_id')
      AND icl.responsible_user_id IN (sqlc.slice('search_user_ids'))
      AND (sqlc.arg('include_item_filter') = false OR icl.item_id IN (sqlc.slice('item_ids')))
      AND (sqlc.arg('include_action_type_filter') = false OR icl.action_type_code IN (sqlc.slice('action_type_codes')))
      AND (sqlc.arg('include_user_filter') = false OR icl.responsible_user_id IN (sqlc.slice('changed_by_user_ids')))
      AND (sqlc.narg('start_date') IS NULL OR icl.created_at >= sqlc.narg('start_date'))
      AND (sqlc.narg('end_date') IS NULL OR icl.created_at <= sqlc.narg('end_date'))
      AND (
          icl.created_at > sqlc.arg('cursor_created_at')
          OR (icl.created_at = sqlc.arg('cursor_created_at') AND icl.id > sqlc.arg('cursor_id'))
      )
     ORDER BY icl.created_at ASC, icl.id ASC
     LIMIT ?)
    UNION DISTINCT
    (SELECT icl.id AS id, icl.created_at AS created_at
     FROM inventory_change_log icl
     WHERE icl.account_id = sqlc.arg('account_id')
      AND icl.scanning_station_id IN (sqlc.slice('search_station_ids'))
      AND (sqlc.arg('include_item_filter') = false OR icl.item_id IN (sqlc.slice('item_ids')))
      AND (sqlc.arg('include_action_type_filter') = false OR icl.action_type_code IN (sqlc.slice('action_type_codes')))
      AND (sqlc.arg('include_user_filter') = false OR icl.responsible_user_id IN (sqlc.slice('changed_by_user_ids')))
      AND (sqlc.narg('start_date') IS NULL OR icl.created_at >= sqlc.narg('start_date'))
      AND (sqlc.narg('end_date') IS NULL OR icl.created_at <= sqlc.narg('end_date'))
      AND (
          icl.created_at > sqlc.arg('cursor_created_at')
          OR (icl.created_at = sqlc.arg('cursor_created_at') AND icl.id > sqlc.arg('cursor_id'))
      )
     ORDER BY icl.created_at ASC, icl.id ASC
     LIMIT ?)
) matched
JOIN inventory_change_log icl ON icl.id = matched.id
JOIN item i ON i.id = icl.item_id
JOIN quantity q ON q.id = icl.quantity_id
JOIN unit u ON u.id = q.unit_id
LEFT JOIN scanning_station ss ON ss.id = icl.scanning_station_id
LEFT JOIN user usr ON usr.id = icl.responsible_user_id
ORDER BY icl.created_at ASC, icl.id ASC
LIMIT ?;

-- ResolveInventoryChangeLogSearchItemIDs finds the items whose SKU matches a search term, so the list query can filter on item_id instead of joining item and matching a LIKE.
-- name: ResolveInventoryChangeLogSearchItemIDs :many
SELECT id FROM item WHERE account_id = sqlc.arg('account_id') AND sku LIKE sqlc.arg('search_query');

-- ResolveInventoryChangeLogSearchUserIDs finds the users whose name matches a search term.
--
-- Not account-scoped: the log row's own account_id does the scoping, and a name matched against a user outside the account only yields an id that no in-account row references.
-- name: ResolveInventoryChangeLogSearchUserIDs :many
SELECT id FROM user WHERE name LIKE sqlc.arg('search_query');

-- ResolveInventoryChangeLogSearchStationIDs finds the scanning stations whose name matches a search term.
-- name: ResolveInventoryChangeLogSearchStationIDs :many
SELECT id FROM scanning_station WHERE account_id = sqlc.arg('account_id') AND name LIKE sqlc.arg('search_query');
