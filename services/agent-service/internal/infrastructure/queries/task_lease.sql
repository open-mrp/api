-- name: AcquireTaskLease :execrows
INSERT INTO task_leases (name, holder, acquired_at, expires_at)
VALUES ($1, $2, now(), now() + ($3 || ' seconds')::interval)
ON CONFLICT (name) DO UPDATE SET
    holder      = EXCLUDED.holder,
    acquired_at = EXCLUDED.acquired_at,
    expires_at  = EXCLUDED.expires_at
WHERE task_leases.expires_at < now() OR task_leases.holder = EXCLUDED.holder;

-- name: RenewTaskLease :execrows
UPDATE task_leases
SET expires_at = now() + ($1 || ' seconds')::interval
WHERE name = $2 AND holder = $3;

-- name: ReleaseTaskLease :exec
DELETE FROM task_leases
WHERE name = $1 AND holder = $2;
