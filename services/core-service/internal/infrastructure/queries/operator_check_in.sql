-- name: InsertOperatorCheckIn :exec
INSERT INTO operator_check_in (
    id, account_id, scanning_station_id, account_user_id, operator_name, checked_in_at, expires_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('account_id'), sqlc.arg('scanning_station_id'), sqlc.arg('account_user_id'),
    sqlc.arg('operator_name'), NOW(3), sqlc.arg('expires_at')
);

-- name: GetOperatorCheckIn :one
SELECT * FROM operator_check_in
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- GetActiveOperatorCheckIn returns who is checked in at a station now, if anyone. Ended check-ins stay
-- behind the active one in checked_in_at order, so the latest row decides.
-- name: GetActiveOperatorCheckIn :one
SELECT * FROM operator_check_in
WHERE account_id = sqlc.arg('account_id')
  AND scanning_station_id = sqlc.arg('scanning_station_id')
ORDER BY checked_in_at DESC, id DESC
LIMIT 1;

-- EndOpenOperatorCheckIns checks out whoever is still checked in at a station.
-- name: EndOpenOperatorCheckIns :exec
UPDATE operator_check_in SET checked_out_at = NOW(3)
WHERE account_id = sqlc.arg('account_id')
  AND scanning_station_id = sqlc.arg('scanning_station_id')
  AND checked_out_at IS NULL;

-- name: CheckOutOperatorCheckIn :execrows
UPDATE operator_check_in SET checked_out_at = NOW(3)
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id') AND checked_out_at IS NULL;
