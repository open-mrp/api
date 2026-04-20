-- name: AcquireTaskLease :execresult
INSERT INTO task_leases (name, holder, acquired_at, expires_at)
VALUES (?, ?, NOW(6), DATE_ADD(NOW(6), INTERVAL ? SECOND))
ON DUPLICATE KEY UPDATE
    holder      = IF(expires_at < NOW(6) OR holder = VALUES(holder), VALUES(holder), holder),
    acquired_at = IF(expires_at < NOW(6) OR holder = VALUES(holder), VALUES(acquired_at), acquired_at),
    expires_at  = IF(expires_at < NOW(6) OR holder = VALUES(holder), VALUES(expires_at), expires_at);

-- name: RenewTaskLease :execresult
UPDATE task_leases
SET expires_at = DATE_ADD(NOW(6), INTERVAL ? SECOND)
WHERE name = ? AND holder = ?;

-- name: ReleaseTaskLease :exec
DELETE FROM task_leases
WHERE name = ? AND holder = ?;
