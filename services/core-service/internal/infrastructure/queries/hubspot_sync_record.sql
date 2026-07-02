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
