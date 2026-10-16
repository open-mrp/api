-- Notes are attributed commentary on one record, or on the whole account when record_id is null.

-- name: InsertNote :exec
INSERT INTO note (
    id, account_id, record_type, record_id, body, type, importance, expires_at, actor_type, actor_id, created_at, updated_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('account_id'), sqlc.narg('record_type'), sqlc.narg('record_id'), sqlc.arg('body'),
    sqlc.arg('type'), sqlc.arg('importance'), sqlc.narg('expires_at'), sqlc.arg('actor_type'), sqlc.narg('actor_id'),
    NOW(3), NOW(3)
);

-- name: GetNote :one
SELECT id, account_id, record_type, record_id, body, type, importance, expires_at, actor_type, actor_id, edited_at, created_at, updated_at
FROM note
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- name: GetNotesByIDs :many
SELECT id, account_id, record_type, record_id, body, type, importance, expires_at, actor_type, actor_id, edited_at, created_at, updated_at
FROM note
WHERE account_id = sqlc.arg('account_id') AND id IN (sqlc.slice('ids'));

-- name: UpdateNote :exec
UPDATE note
SET body = sqlc.arg('body'),
    type = sqlc.arg('type'),
    importance = sqlc.arg('importance'),
    expires_at = sqlc.narg('expires_at'),
    record_type = sqlc.narg('record_type'),
    record_id = sqlc.narg('record_id'),
    edited_at = sqlc.narg('edited_at'),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- name: DeleteNote :exec
DELETE FROM note WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- RecallAccountNotes is the account-wide guidance an agent starts from: every unexpired note on no record except
-- comments, most important first. Reads note_account_record_created_idx at record_id IS NULL.
-- name: RecallAccountNotes :many
SELECT id, account_id, record_type, record_id, body, type, importance, expires_at, actor_type, actor_id, edited_at, created_at, updated_at
FROM note FORCE INDEX (note_account_record_created_idx)
WHERE account_id = sqlc.arg('account_id')
  AND record_id IS NULL
  AND type <> 'comment'
  AND (expires_at IS NULL OR expires_at > NOW(3))
ORDER BY importance DESC, created_at DESC, id DESC
LIMIT ?;

-- RecallRecordNotes is every unexpired note on the records an agent is working on, most important first.
-- name: RecallRecordNotes :many
SELECT id, account_id, record_type, record_id, body, type, importance, expires_at, actor_type, actor_id, edited_at, created_at, updated_at
FROM note FORCE INDEX (note_account_record_created_idx)
WHERE account_id = sqlc.arg('account_id')
  AND record_id IN (sqlc.slice('record_ids'))
  AND (expires_at IS NULL OR expires_at > NOW(3))
ORDER BY importance DESC, created_at DESC, id DESC
LIMIT ?;
