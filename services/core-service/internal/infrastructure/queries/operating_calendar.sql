-- name: GetOperatingCalendar :one
SELECT * FROM operating_calendar
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id') AND deleted_at IS NULL;

-- name: GetOperatingCalendarByCode :one
SELECT * FROM operating_calendar
WHERE account_id = sqlc.arg('account_id') AND code = sqlc.arg('code') AND deleted_at IS NULL;

-- name: ListOperatingCalendars :many
SELECT * FROM operating_calendar
WHERE account_id = sqlc.arg('account_id')
AND deleted_at IS NULL
AND (sqlc.narg('kind_code') IS NULL OR operating_calendar_kind_code = sqlc.narg('kind_code'))
ORDER BY operating_calendar_kind_code, is_default DESC, code;

-- Resolves the ship calendar for an account: whatever the settings point at, else the account's default ship calendar.
-- Both halves come back in one round trip because this runs inside the issue transaction, where a second seek is a second chance to block.
-- name: ResolveShipCalendar :one
SELECT sqlc.embed(operating_calendar)
FROM operating_calendar
LEFT JOIN account_production_schedule_setting
    ON account_production_schedule_setting.account_id = sqlc.arg('account_id')
WHERE operating_calendar.account_id = sqlc.arg('account_id')
AND operating_calendar.operating_calendar_kind_code = 'ship'
AND operating_calendar.deleted_at IS NULL
AND (
    operating_calendar.id = account_production_schedule_setting.ship_calendar_id
    OR operating_calendar.is_default = TRUE
)
-- An explicitly configured calendar outranks the account default, which is what the ordering buys; LIMIT 1 then takes it.
ORDER BY (operating_calendar.id = account_production_schedule_setting.ship_calendar_id) DESC, operating_calendar.is_default DESC
LIMIT 1;

-- Resolves the receiving calendar for one order's destination, walking address -> customer -> customer's group -> account default -> the account's default receive calendar.
-- One query rather than four reads: the chain is four LEFT JOINs off the address, and the ORDER BY expresses the precedence so the caller never has to know it. Mirrors GetCustomerLeadTimeChain, which resolves the sibling lead-time chain the same way.
-- name: ResolveReceiveCalendar :one
SELECT sqlc.embed(operating_calendar)
FROM operating_calendar
LEFT JOIN address ON address.id = sqlc.narg('address_id')
LEFT JOIN account_relation
    ON account_relation.owner_account_id = sqlc.arg('account_id')
    AND account_relation.counterparty_account_id = sqlc.narg('buyer_account_id')
    AND account_relation.account_relation_role_code = 'customer'
LEFT JOIN account_group ON account_group.id = account_relation.account_group_id
LEFT JOIN account_production_schedule_setting
    ON account_production_schedule_setting.account_id = sqlc.arg('account_id')
WHERE operating_calendar.account_id = sqlc.arg('account_id')
AND operating_calendar.operating_calendar_kind_code = 'receive'
AND operating_calendar.deleted_at IS NULL
AND (
    operating_calendar.id = address.receive_calendar_id
    OR operating_calendar.id = account_relation.receive_calendar_id
    OR operating_calendar.id = account_group.receive_calendar_id
    OR operating_calendar.id = account_production_schedule_setting.receive_calendar_id
    OR operating_calendar.is_default = TRUE
)
ORDER BY
    (operating_calendar.id = address.receive_calendar_id) DESC,
    (operating_calendar.id = account_relation.receive_calendar_id) DESC,
    (operating_calendar.id = account_group.receive_calendar_id) DESC,
    (operating_calendar.id = account_production_schedule_setting.receive_calendar_id) DESC,
    operating_calendar.is_default DESC
LIMIT 1;

-- Closures for a set of calendars inside a date window.
-- Bounded by date on purpose: resolving one commitment needs the months around its ship-by date, and loading an account's whole closure history would grow the cost of every issue forever.
-- name: ListOperatingCalendarClosures :many
SELECT id, account_id, operating_calendar_id, closed_on, name, created_at, updated_at
FROM operating_calendar_closure
WHERE account_id = sqlc.arg('account_id')
AND operating_calendar_id IN (sqlc.slice('calendar_ids'))
AND closed_on BETWEEN sqlc.arg('from_date') AND sqlc.arg('to_date')
ORDER BY closed_on;

-- name: CreateOperatingCalendar :exec
INSERT INTO operating_calendar (
    id, account_id, code, name, operating_calendar_kind_code,
    days_of_week, cutoff_at, timezone, is_default, created_at, updated_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('account_id'), sqlc.arg('code'), sqlc.arg('name'), sqlc.arg('kind_code'),
    sqlc.arg('days_of_week'), sqlc.narg('cutoff_at'), sqlc.narg('timezone'), sqlc.arg('is_default'), NOW(3), NOW(3)
);

-- A NULL argument leaves a column alone, so clearing a cutoff or a zone needs its own statement below rather than a flag: one parameter cannot mean both "unchanged" and "set to null".
-- name: UpdateOperatingCalendar :exec
UPDATE operating_calendar SET
    name = COALESCE(sqlc.narg('name'), name),
    days_of_week = COALESCE(sqlc.narg('days_of_week'), days_of_week),
    cutoff_at = COALESCE(sqlc.narg('cutoff_at'), cutoff_at),
    timezone = COALESCE(sqlc.narg('timezone'), timezone),
    is_default = COALESCE(sqlc.narg('is_default'), is_default),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id') AND deleted_at IS NULL;

-- name: ClearOperatingCalendarCutoff :exec
UPDATE operating_calendar SET cutoff_at = NULL, updated_at = NOW(3)
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id') AND deleted_at IS NULL;

-- name: ClearOperatingCalendarTimezone :exec
UPDATE operating_calendar SET timezone = NULL, updated_at = NOW(3)
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id') AND deleted_at IS NULL;

-- Demotes every other calendar of the same kind, so exactly one default survives per kind.
-- name: ClearOperatingCalendarDefault :exec
UPDATE operating_calendar SET is_default = FALSE, updated_at = NOW(3)
WHERE account_id = sqlc.arg('account_id')
AND operating_calendar_kind_code = sqlc.arg('kind_code')
AND id != sqlc.arg('keep_id')
AND is_default = TRUE;

-- Soft delete, because an order's commitment was resolved against this calendar and a hard delete would orphan the links that explain it.
-- name: DeleteOperatingCalendar :exec
UPDATE operating_calendar SET deleted_at = NOW(3), updated_at = NOW(3)
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id') AND deleted_at IS NULL;

-- name: CountOperatingCalendarReferences :one
SELECT
    (SELECT COUNT(*) FROM address WHERE address.receive_calendar_id = sqlc.arg('id')) AS address_count,
    (SELECT COUNT(*) FROM account_relation WHERE account_relation.receive_calendar_id = sqlc.arg('id')) AS customer_count,
    (SELECT COUNT(*) FROM account_group WHERE account_group.receive_calendar_id = sqlc.arg('id')) AS group_count,
    (SELECT COUNT(*) FROM account_production_schedule_setting
        WHERE account_production_schedule_setting.ship_calendar_id = sqlc.arg('id')
        OR account_production_schedule_setting.receive_calendar_id = sqlc.arg('id')) AS setting_count;

-- name: UpsertOperatingCalendarClosure :exec
INSERT INTO operating_calendar_closure (
    id, account_id, operating_calendar_id, closed_on, name, created_at, updated_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('account_id'), sqlc.arg('calendar_id'), sqlc.arg('closed_on'), sqlc.arg('name'), NOW(3), NOW(3)
)
-- Re-seeding a year must not rename a closure an operator has already relabelled, so an existing row only has its timestamp touched.
ON DUPLICATE KEY UPDATE updated_at = NOW(3);

-- name: DeleteOperatingCalendarClosure :exec
DELETE FROM operating_calendar_closure
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- Reads a closure back by the key it is actually unique on.
--
-- The write is an upsert, so a caller re-closing a date it already closed gets the row that was already there rather than the ID the write happened to generate and discard. Looking it up by ID would return nothing in exactly that case.
-- name: GetOperatingCalendarClosureByDate :one
SELECT * FROM operating_calendar_closure
WHERE account_id = sqlc.arg('account_id')
AND operating_calendar_id = sqlc.arg('calendar_id')
AND closed_on = sqlc.arg('closed_on');

-- name: GetOperatingCalendarClosure :one
SELECT * FROM operating_calendar_closure
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');
