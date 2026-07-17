-- name: UpsertHubspotSyncRecord :exec
INSERT INTO hubspot_sync_record (
    id,
    account_id,
    augno_type,
    augno_id,
    hubspot_type,
    hubspot_id,
    sync_hash,
    last_synced_at,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('account_id'),
    sqlc.arg('augno_type'),
    sqlc.arg('augno_id'),
    sqlc.arg('hubspot_type'),
    sqlc.arg('hubspot_id'),
    sqlc.narg('sync_hash'),
    NOW(3),
    NOW(3),
    NOW(3)
)
ON DUPLICATE KEY UPDATE
    hubspot_type = VALUES(hubspot_type),
    hubspot_id = VALUES(hubspot_id),
    sync_hash = VALUES(sync_hash),
    last_synced_at = NOW(3),
    last_error = NULL,
    updated_at = NOW(3);

-- name: GetHubspotSyncRecord :one
SELECT
    id,
    account_id,
    augno_type,
    augno_id,
    hubspot_type,
    hubspot_id,
    sync_hash,
    last_synced_at,
    last_error,
    created_at,
    updated_at
FROM hubspot_sync_record
WHERE account_id = sqlc.arg('account_id')
AND augno_type = sqlc.arg('augno_type')
AND augno_id = sqlc.arg('augno_id');

-- Lists what the sync has actually written to HubSpot, newest mappings resolvable by name.
-- Keyset paginates on augno_id so the (account_id, augno_type, augno_id) unique key drives both the filter and the ordering; the joins are PK lookups off that. augno_type is required rather than optional precisely to keep that index prefix intact.
-- The customer joins are LEFT so a mapping whose customer was deleted still lists (with a null name) instead of vanishing.
-- name: ListHubspotSyncRecords :many
SELECT
    r.id,
    r.account_id,
    r.augno_type,
    r.augno_id,
    r.hubspot_type,
    r.hubspot_id,
    r.sync_hash,
    r.last_synced_at,
    r.last_error,
    r.created_at,
    r.updated_at,
    COALESCE(NULLIF(ar.alias, ''), a.name, '') AS augno_name
FROM hubspot_sync_record r
LEFT JOIN account_relation ar ON ar.owner_account_id = r.account_id AND ar.counterparty_account_id = r.augno_id
LEFT JOIN account a ON a.id = r.augno_id
WHERE r.account_id = sqlc.arg('account_id')
AND r.augno_type = sqlc.arg('augno_type')
AND (sqlc.narg('cursor') IS NULL OR r.augno_id > sqlc.narg('cursor'))
ORDER BY r.augno_id ASC
LIMIT ?;
